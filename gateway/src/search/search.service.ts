import { Injectable, Logger } from "@nestjs/common";

export interface SearchFilters {
  studyTypes?: string[];
  publishedYearMin?: number;
  publishedYearMax?: number;
  meshTerms?: string[];
  sources?: string[];
  licenses?: string[];
  onlyWithCoi?: boolean;
  onlyWithFullText?: boolean;
  excludePredatoryJournals?: boolean;
}

export interface ScoredResult {
  document: any;
  finalScore: number;
  breakdown: any;
}

export interface SearchResult {
  query: string;
  results: ScoredResult[];
  totalEstimated: number;
  variant?: string;
  elapsedMs: number;
}

export type WaveCallback = (
  wave: number,
  isFinal: boolean,
  results: ScoredResult[],
  elapsedMs: number,
) => void | Promise<void>;

const SCORER_URL = process.env.SCORER_HTTP_URL ?? "http://scorer:8090";

@Injectable()
export class SearchService {
  private readonly logger = new Logger(SearchService.name);

  /**
   * Synchronous search: drains the SSE stream from scorer-pool, returns
   * the final wave's results.
   */
  async search(
    query: string,
    filters?: SearchFilters,
    topK = 50,
    variant?: string,
    sessionId?: string,
  ): Promise<SearchResult> {
    const start = Date.now();
    let final: ScoredResult[] = [];
    let totalElapsed = 0;
    try {
      await this.streamFromScorer(query, filters, topK, (wave, isFinal, results, elapsedMs) => {
        if (isFinal || wave === 3) final = results;
        totalElapsed = elapsedMs;
      });
    } catch (e) {
      this.logger.warn(`search upstream error: ${(e as Error).message}`);
    }
    return {
      query,
      results: final,
      totalEstimated: final.length,
      variant,
      elapsedMs: totalElapsed || Date.now() - start,
    };
  }

  /**
   * Streamed search: invoke `cb` for each wave as it arrives.
   * Used by the WebSocket gateway to forward wave frames downstream.
   */
  async streamFromScorer(
    query: string,
    filters: SearchFilters | undefined,
    topK: number,
    cb: WaveCallback,
  ): Promise<void> {
    const start = Date.now();
    const res = await fetch(`${SCORER_URL}/search`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ query, filters, top_k: topK }),
    });
    if (!res.ok || !res.body) {
      throw new Error(`scorer upstream ${res.status}`);
    }
    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buf = "";
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buf += decoder.decode(value, { stream: true });
      // SSE frames are separated by blank line.
      let i: number;
      while ((i = buf.indexOf("\n\n")) >= 0) {
        const frame = buf.slice(0, i);
        buf = buf.slice(i + 2);
        const dataLine = frame.split("\n").find(l => l.startsWith("data:"));
        if (!dataLine) continue;
        const json = dataLine.slice(5).trim();
        if (json === "{}") continue; // sentinel
        try {
          const obj = JSON.parse(json);
          if (obj.type === "search.partial" || obj.type === "search.final") {
            await cb(obj.wave, obj.isFinal, obj.results ?? [], Date.now() - start);
          }
        } catch { /* malformed frame, skip */ }
      }
    }
  }
}
