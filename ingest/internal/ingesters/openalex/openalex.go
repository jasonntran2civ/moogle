// Package openalex ingests OpenAlex works + emits citation edges
// (spec §5.1.6).
//
// Two paths:
//   - Bulk snapshot via S3 (s3://openalex/data/), ~300GB compressed —
//     stream-process directly without disk staging (escalation trigger
//     if exceeds Cloud Run free tier on first run).
//   - Per-doc REST updates via api.openalex.org/works/{id}.
//
// Citation edges (citing_doc_id, cited_doc_id) emit to a separate
// Pub/Sub topic citation-edges for the indexer's Neo4j batcher.
//
// TODO: implement bulk + REST paths. Currently a stub.
package openalex

import (
	"context"
	"log/slog"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

type Config struct {
	MaxPerRun int
}

type Ingester struct {
	cfg          Config
	logger       *slog.Logger
	wm           *watermark.Store
	archiver     *r2.Archiver
	pub          *pubsubpub.Publisher
	citationsPub *pubsubpub.Publisher
}

func New(cfg Config, logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub, citationsPub *pubsubpub.Publisher) *Ingester {
	return &Ingester{cfg: cfg, logger: logger, wm: wm, archiver: arch, pub: pub, citationsPub: citationsPub}
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	_ = i.wm.MarkRunning(ctx, "openalex")
	i.logger.Warn("openalex ingester is a stub; bulk+REST paths TODO")
	_ = i.wm.Set(ctx, "openalex", "stub", "idle", "")
	return ingestcommon.RunResult{}, nil
}
