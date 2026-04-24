import { Module } from "@nestjs/common";
import { Controller, Get, NotFoundException, Param } from "@nestjs/common";

@Controller("api/document")
class DocumentController {
  @Get(":id")
  async byId(@Param("id") id: string) {
    // TODO: fetch from Meilisearch + enrich with Neo4j neighbors + COI badges.
    throw new NotFoundException(`document ${id} not found (stub)`);
  }
}

@Module({ controllers: [DocumentController] })
export class DocumentModule {}
