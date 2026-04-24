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

export interface SearchResult {
  query: string;
  results: any[];
  totalEstimated: number;
  variant?: string;
  elapsedMs: number;
}

@Injectable()
export class SearchService {
  private readonly logger = new Logger(SearchService.name);
  // TODO: gRPC client to scorer-pool. Stub returns deterministic results
  // so the gateway compiles + returns shape until proto stubs land.
  async search(
    query: string,
    filters?: SearchFilters,
    topK = 50,
    variant?: string,
    sessionId?: string,
  ): Promise<SearchResult> {
    const start = Date.now();
    this.logger.debug(`search q=${query} topK=${topK} variant=${variant}`);
    return {
      query,
      results: [],
      totalEstimated: 0,
      variant,
      elapsedMs: Date.now() - start,
    };
  }
}
