-- EvidenceLens operational Postgres schema.
--
-- Per spec section 3.5. Operational state only -- the canonical document
-- store lives in MongoDB-equivalent (Meilisearch + Qdrant + Neo4j).
-- Postgres holds ingestion watermarks, recall events, share links, BYOK
-- telemetry, A/B audit, and the author-payment cache.
--
-- Run automatically by docker-entrypoint-initdb.d on first container start.

CREATE EXTENSION IF NOT EXISTS pg_trgm;        -- for author name fuzzy lookup
CREATE EXTENSION IF NOT EXISTS unaccent;       -- name normalization

-- ---- Ingestion watermarks ----
CREATE TABLE IF NOT EXISTS ingestion_state (
    source              TEXT PRIMARY KEY,
    last_run_at         TIMESTAMPTZ,
    last_high_watermark TEXT,
    status              TEXT CHECK (status IN ('idle','running','failed','degraded')) DEFAULT 'idle',
    last_error          TEXT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---- Recall events (mirror of NATS recall-fanout for query/audit) ----
CREATE TABLE IF NOT EXISTS recall_events (
    id                   TEXT PRIMARY KEY,
    agency               TEXT NOT NULL,
    product_name         TEXT NOT NULL,
    drug_class           TEXT,
    recall_class         TEXT CHECK (recall_class IN ('I','II','III')),
    emitted_at           TIMESTAMPTZ NOT NULL,
    fanout_completed_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS recall_events_emitted_idx ON recall_events (emitted_at DESC);
CREATE INDEX IF NOT EXISTS recall_events_drug_class_idx ON recall_events (drug_class);

-- ---- Share links (deep links to result sets) ----
CREATE TABLE IF NOT EXISTS share_links (
    slug       TEXT PRIMARY KEY,
    query      JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    hit_count  BIGINT NOT NULL DEFAULT 0
);

-- ---- BYOK proxy telemetry (NEVER stores keys) ----
CREATE TABLE IF NOT EXISTS byok_proxy_telemetry (
    id                BIGSERIAL PRIMARY KEY,
    session_id        TEXT,
    provider          TEXT CHECK (provider IN ('anthropic','openai','groq','openrouter','together','deepinfra','ollama')),
    model             TEXT,
    prompt_tokens     INT,
    completion_tokens INT,
    cached_tokens     INT,
    duration_ms       INT,
    error             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS byok_proxy_created_idx ON byok_proxy_telemetry (created_at DESC);

-- ---- A/B assignment audit ----
CREATE TABLE IF NOT EXISTS ab_assignment_audit (
    session_id     TEXT NOT NULL,
    experiment_key TEXT NOT NULL,
    variant        TEXT NOT NULL,
    assigned_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (session_id, experiment_key)
);

-- ---- Author × Open Payments cache ----
CREATE TABLE IF NOT EXISTS author_payment_cache (
    author_key      TEXT NOT NULL,           -- normalized "lastname:firstname:state"
    year            INT  NOT NULL,
    payments_jsonb  JSONB NOT NULL,
    cached_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
    PRIMARY KEY (author_key, year)
);
CREATE INDEX IF NOT EXISTS author_payment_cache_expires_idx ON author_payment_cache (expires_at);
CREATE INDEX IF NOT EXISTS author_payment_cache_author_trgm_idx ON author_payment_cache USING gin (author_key gin_trgm_ops);

-- ---- Open Payments raw (ingester writes here; joiner reads via /lookup) ----
CREATE TABLE IF NOT EXISTS open_payments (
    record_id       TEXT PRIMARY KEY,
    physician_npi   TEXT,
    physician_name  TEXT NOT NULL,
    physician_state TEXT,
    sponsor_name    TEXT NOT NULL,
    payment_year    INT NOT NULL,
    amount_usd      NUMERIC(14,2) NOT NULL,
    payment_type    TEXT,
    raw_jsonb       JSONB
);
CREATE INDEX IF NOT EXISTS open_payments_npi_idx ON open_payments (physician_npi);
CREATE INDEX IF NOT EXISTS open_payments_name_state_idx ON open_payments (physician_name, physician_state);
CREATE INDEX IF NOT EXISTS open_payments_year_idx ON open_payments (payment_year);
CREATE INDEX IF NOT EXISTS open_payments_name_trgm_idx ON open_payments USING gin (physician_name gin_trgm_ops);
