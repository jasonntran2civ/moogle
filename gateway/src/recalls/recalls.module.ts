import { Controller, Get, Module, OnModuleDestroy, OnModuleInit, Query } from "@nestjs/common";
import { Pool } from "pg";

@Controller("api/recalls")
class RecallsController implements OnModuleInit, OnModuleDestroy {
  private pool: Pool | null = null;

  async onModuleInit(): Promise<void> {
    if (!process.env.DATABASE_URL) return;
    this.pool = new Pool({ connectionString: process.env.DATABASE_URL });
  }

  async onModuleDestroy(): Promise<void> {
    if (this.pool) await this.pool.end();
  }

  @Get("recent")
  async recent(
    @Query("since_days") sinceDays = "30",
    @Query("drug_class") drugClass?: string,
    @Query("product_name") productName?: string,
    @Query("top_k") topK = "20",
  ) {
    if (!this.pool) {
      return { sinceDays: parseInt(sinceDays, 10), events: [], note: "DATABASE_URL not configured" };
    }
    const days = parseInt(sinceDays, 10) || 30;
    const limit = Math.min(parseInt(topK, 10) || 20, 200);
    const where: string[] = [`emitted_at >= NOW() - ($1::int * INTERVAL '1 day')`];
    const params: any[] = [days];
    if (drugClass)  { params.push(drugClass);   where.push(`drug_class = $${params.length}`); }
    if (productName){ params.push(productName); where.push(`product_name = $${params.length}`); }
    params.push(limit);
    const sql =
      `SELECT id AS recall_id, agency, product_name, drug_class, recall_class, emitted_at
       FROM recall_events
       WHERE ${where.join(" AND ")}
       ORDER BY emitted_at DESC
       LIMIT $${params.length}`;
    const r = await this.pool.query(sql, params);
    return { sinceDays: days, drugClass, productName, events: r.rows };
  }
}

@Module({ controllers: [RecallsController] })
export class RecallsModule {}
