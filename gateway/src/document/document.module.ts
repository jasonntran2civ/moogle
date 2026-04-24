import { Controller, Get, HttpException, HttpStatus, Module, NotFoundException, Param } from "@nestjs/common";

const MEILI_URL = process.env.MEILI_URL ?? "http://meilisearch:7700";
const MEILI_KEY = process.env.MEILI_KEY ?? "";
const NEO4J_HTTP = process.env.NEO4J_HTTP_URL ?? "http://neo4j:7474";

@Controller("api/document")
class DocumentController {
  @Get(":id")
  async byId(@Param("id") id: string) {
    const headers: Record<string, string> = {};
    if (MEILI_KEY) headers.authorization = `Bearer ${MEILI_KEY}`;
    const r = await fetch(
      `${MEILI_URL}/indexes/documents/documents/${encodeURIComponent(id)}`,
      { headers },
    );
    if (r.status === 404) throw new NotFoundException(`document ${id} not found`);
    if (!r.ok) throw new HttpException(`meili ${r.status}`, HttpStatus.BAD_GATEWAY);
    const doc = await r.json() as any;

    // Best-effort enrich: pull a small citation neighborhood (<=20 nodes)
    // for the result drawer's CitationGraph component. Failures don't
    // block; the drawer falls back gracefully.
    try {
      doc.citationNeighborhood = await fetchNeighborhood(id);
    } catch {
      doc.citationNeighborhood = null;
    }
    return doc;
  }
}

async function fetchNeighborhood(id: string) {
  // Neo4j HTTP transactional Cypher endpoint. Minimal projection.
  const cypher = {
    statements: [{
      statement:
        "MATCH (d:Document {id: $id})-[:CITES|AUTHORED_BY*1..1]-(n) " +
        "RETURN n.id AS id, n.title AS title, n.pagerank AS pagerank LIMIT 30",
      parameters: { id },
    }],
  };
  const auth = "Basic " + Buffer.from(
    `${process.env.NEO4J_USER ?? "neo4j"}:${process.env.NEO4J_PASSWORD ?? ""}`,
  ).toString("base64");
  const r = await fetch(`${NEO4J_HTTP}/db/neo4j/tx/commit`, {
    method: "POST",
    headers: { "content-type": "application/json", authorization: auth },
    body: JSON.stringify(cypher),
    signal: AbortSignal.timeout(1500),
  });
  if (!r.ok) return null;
  const data = await r.json() as any;
  const rows = data?.results?.[0]?.data ?? [];
  return rows.map((row: any) => ({
    id: row.row[0], title: row.row[1], pagerank: row.row[2] ?? 0,
  }));
}

@Module({ controllers: [DocumentController] })
export class DocumentModule {}
