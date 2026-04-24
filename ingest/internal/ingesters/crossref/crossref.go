// Package crossref enriches a single DOI's metadata via api.crossref.org/works/{doi}
// (spec §5.1.7). Triggered by the processor.
package crossref

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

type Config struct{ Email string }

type Ingester struct {
	cfg      Config
	logger   *slog.Logger
	wm       *watermark.Store
	archiver *r2.Archiver
	pub      *pubsubpub.Publisher
	fetcher  *ingestcommon.Fetcher
}

func New(cfg Config, logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		cfg: cfg, logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(50, 100, fmt.Sprintf("EvidenceLens/0.1 (mailto:%s)", cfg.Email)),
	}
}

// Run reads the DOI from env (POST body would be plumbed via http_server).
// For now the cron-triggered run pulls a batch of DOIs needing enrichment
// from a TODO Postgres queue. Stub returns no-op.
func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	_ = i.wm.MarkRunning(ctx, "crossref")
	doi := os.Getenv("CROSSREF_DOI")
	if doi == "" {
		i.logger.Info("crossref: no DOI in env; cron-batched enrichment TODO")
		return ingestcommon.RunResult{}, nil
	}
	body, err := i.fetcher.Get(ctx, fmt.Sprintf("https://api.crossref.org/works/%s", doi), nil)
	if err != nil {
		return ingestcommon.RunResult{}, err
	}
	id := "doi:" + doi
	rawJSON, _ := json.Marshal(json.RawMessage(body))
	key, err := i.archiver.Put(ctx, "crossref", id, rawJSON)
	if err != nil {
		return ingestcommon.RunResult{}, err
	}
	if _, err := i.pub.PublishRaw(ctx, "crossref", id, key); err != nil {
		return ingestcommon.RunResult{}, err
	}
	return ingestcommon.RunResult{DocsFetched: 1, DocsArchived: 1, DocsPublished: 1}, nil
}
