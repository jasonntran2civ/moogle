import { Controller, Get } from "@nestjs/common";

@Controller()
export class HealthController {
  @Get("healthz")
  healthz() {
    return { status: "ok" };
  }

  @Get("readyz")
  readyz() {
    // TODO: gRPC ping to scorer-pool, NATS ping, Postgres ping.
    return { status: "ready" };
  }
}
