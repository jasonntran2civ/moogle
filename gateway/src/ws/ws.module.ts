import { Module } from "@nestjs/common";
import {
  ConnectedSocket, MessageBody, OnGatewayConnection, OnGatewayDisconnect,
  SubscribeMessage, WebSocketGateway, WebSocketServer,
} from "@nestjs/websockets";
import { Logger } from "@nestjs/common";
import type { WebSocket, Server } from "ws";
import { SearchModule } from "../search/search.module.js";

interface Frame { type: string; id: string; [k: string]: any }

@WebSocketGateway({ path: "/ws" })
class GatewayWebSocket implements OnGatewayConnection, OnGatewayDisconnect {
  private readonly logger = new Logger(GatewayWebSocket.name);
  @WebSocketServer() server!: Server;

  handleConnection(client: WebSocket) {
    this.logger.log("ws.connect");
    // TODO: enforce per-IP simultaneous connection cap (10 per spec section 5.6).
    client.on("error", err => this.logger.error("ws.error", err.message));
  }

  handleDisconnect() {
    this.logger.log("ws.disconnect");
  }

  @SubscribeMessage("ping")
  ping(@MessageBody() data: Frame) {
    return { type: "pong", id: data.id };
  }

  @SubscribeMessage("search")
  async search(@MessageBody() data: Frame, @ConnectedSocket() client: WebSocket) {
    // TODO: stream PartialResults waves via gRPC scorer-pool.
    // Stub: emit a single "search.final" with empty results.
    client.send(JSON.stringify({
      type: "search.final", id: data.id, wave: 1, isFinal: true,
      elapsedMs: 0, totalEstimated: 0, results: [],
    }));
  }

  @SubscribeMessage("getDoc")
  async getDoc(@MessageBody() data: Frame, @ConnectedSocket() client: WebSocket) {
    // TODO: fetch document by id from Meilisearch.
    client.send(JSON.stringify({
      type: "error", id: data.id, code: "not_found",
      message: `document ${data["documentId"]} not found (stub)`,
    }));
  }

  @SubscribeMessage("subscribe")
  async subscribe(@MessageBody() data: Frame, @ConnectedSocket() client: WebSocket) {
    // TODO: NATS subscription bridge for "recalls" topic.
    client.send(JSON.stringify({ type: "subscribed", id: data.id, topic: data["topic"] }));
  }
}

@Module({ imports: [SearchModule], providers: [GatewayWebSocket] })
export class GatewayWebSocketModule {}
