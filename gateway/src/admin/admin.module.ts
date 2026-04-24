import { Controller, Get, Module } from "@nestjs/common";

@Controller("admin")
class AdminController {
  // Internal-only. Behind Cloudflare Access in production.
  @Get("status")
  async status() {
    return {
      uptime_seconds: process.uptime(),
      pid: process.pid,
      // TODO: add scorer / NATS / postgres connectivity probes.
    };
  }
}

@Module({ controllers: [AdminController] })
export class AdminModule {}
