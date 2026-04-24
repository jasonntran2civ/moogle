import { Controller, Get, Module, Query } from "@nestjs/common";

@Controller("api/trials")
class TrialsController {
  @Get()
  async list(
    @Query("condition") condition?: string,
    @Query("location") location?: string,
    @Query("status") status?: string,
    @Query("phase") phase?: string,
    @Query("top_k") topK = "50",
  ) {
    // TODO: filtered Meilisearch query restricted to source=ctgov | ictrp.
    return { condition, location, status, phase, topK: parseInt(topK, 10), results: [] };
  }
}

@Module({ controllers: [TrialsController] })
export class TrialsModule {}
