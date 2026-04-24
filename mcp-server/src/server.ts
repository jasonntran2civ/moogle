/**
 * EvidenceLens MCP server (spec §5.8).
 *
 * Anthropic Model Context Protocol v2025-06 over both stdio (Claude
 * Desktop) and HTTP+SSE (remote clients). Public endpoint:
 *   https://mcp-evidencelens.<account>.workers.dev/sse
 *
 * Tool dispatch proxies to the gateway's /api/tool/{name} endpoint so
 * the MCP server stays thin (no business logic) and the gateway remains
 * the single source of truth for tool semantics.
 */
import express from "express";
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { SSEServerTransport } from "@modelcontextprotocol/sdk/server/sse.js";
import {
  CallToolRequestSchema,
  ListResourcesRequestSchema,
  ListToolsRequestSchema,
  ReadResourceRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
import { request as undiciRequest } from "undici";

import { TOOLS, RESOURCE_URI_TEMPLATE } from "./tools.js";

const GATEWAY_URL = process.env.GATEWAY_URL ?? "http://localhost:8080";

function makeServer(): Server {
  const server = new Server(
    { name: "evidencelens", version: "0.1.0" },
    { capabilities: { tools: {}, resources: {} } },
  );

  server.setRequestHandler(ListToolsRequestSchema, async () => ({ tools: TOOLS }));

  server.setRequestHandler(CallToolRequestSchema, async (req) => {
    const { name, arguments: args } = req.params;
    const url = `${GATEWAY_URL}/api/tool/${encodeURIComponent(name)}`;
    const res = await undiciRequest(url, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(args ?? {}),
    });
    if (res.statusCode >= 400) {
      const text = await res.body.text();
      return { content: [{ type: "text", text: `tool ${name} failed: ${res.statusCode} ${text}` }], isError: true };
    }
    const data = await res.body.json();
    return { content: [{ type: "text", text: JSON.stringify(data) }] };
  });

  server.setRequestHandler(ListResourcesRequestSchema, async () => ({
    resources: [{
      uri: RESOURCE_URI_TEMPLATE,
      name: "EvidenceLens documents",
      description: "Documents addressable as evidencelens://document/{id}",
      mimeType: "application/json",
    }],
  }));

  server.setRequestHandler(ReadResourceRequestSchema, async (req) => {
    const { uri } = req.params;
    const m = uri.match(/^evidencelens:\/\/document\/(.+)$/);
    if (!m) throw new Error(`unsupported resource uri: ${uri}`);
    const docId = decodeURIComponent(m[1]);
    const res = await undiciRequest(`${GATEWAY_URL}/api/document/${encodeURIComponent(docId)}`);
    const text = await res.body.text();
    return {
      contents: [{ uri, mimeType: "application/json", text }],
    };
  });

  return server;
}

async function runStdio(): Promise<void> {
  const transport = new StdioServerTransport();
  const server = makeServer();
  await server.connect(transport);
  console.error("[mcp] stdio transport ready");
}

async function runHttp(port: number): Promise<void> {
  const app = express();
  const transports = new Map<string, SSEServerTransport>();

  app.get("/.well-known/mcp.json", (_req, res) => {
    res.json({
      schema_version: "2025-06",
      name: "evidencelens",
      description: "Free, public, agentic biomedical evidence search",
      transport: {
        type: "http+sse",
        endpoint: "/sse",
      },
    });
  });

  app.get("/sse", async (req, res) => {
    const transport = new SSEServerTransport("/messages", res);
    transports.set(transport.sessionId, transport);
    res.on("close", () => transports.delete(transport.sessionId));
    const server = makeServer();
    await server.connect(transport);
  });

  app.post("/messages", express.json(), async (req, res) => {
    const sessionId = String(req.query.sessionId ?? "");
    const transport = transports.get(sessionId);
    if (!transport) { res.status(404).end(); return; }
    await transport.handlePostMessage(req, res);
  });

  app.get("/healthz", (_req, res) => res.json({ status: "ok" }));

  app.listen(port, () => console.log(`[mcp] http+sse listening on :${port}`));
}

const transport = process.env.MCP_TRANSPORT ?? "stdio";
if (transport === "http") {
  runHttp(parseInt(process.env.MCP_PORT ?? "8082", 10)).catch(err => {
    console.error("[mcp] fatal", err); process.exit(1);
  });
} else {
  runStdio().catch(err => { console.error("[mcp] fatal", err); process.exit(1); });
}
