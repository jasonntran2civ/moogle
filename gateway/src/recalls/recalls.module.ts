import { Controller, Get, Module, Query } from "@nestjs/common";

@Controller("api/recalls")
class RecallsController {
  @Get("recent")
  async recent(
    @Query("since_days") sinceDays = "30",
    @Query("drug_class") drugClass?: string,
    @Query("product_name") productName?: string,
    @Query("top_k") topK = "20",
  ) {
    // TODO: query Postgres recall_events.
    return {
      sinceDays: parseInt(sinceDays, 10),
      drugClass, productName,
      topK: parseInt(topK, 10),
      events: [],
    };
  }
}

@Module({ controllers: [RecallsController] })
export class RecallsModule {}
