// Package nihreporter ingests NIH RePORTER funding records (spec §5.1.9).
// REST: api.reporter.nih.gov/v2/projects/search with date filters.
// Joins to documents via funding.grant_id lookups in the processor.
//
// Stub implementation. TODO: implement project search + pagination.
package nihreporter

import (
	"context"
	"log/slog"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

type Config struct{ MaxPerRun int }

type Ingester struct {
	cfg Config; logger *slog.Logger; wm *watermark.Store
	archiver *r2.Archiver; pub *pubsubpub.Publisher
}

func New(cfg Config, logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{cfg: cfg, logger: logger, wm: wm, archiver: arch, pub: pub}
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	_ = i.wm.MarkRunning(ctx, "nih-reporter")
	i.logger.Warn("nih-reporter ingester stub: implement projects/search pagination")
	return ingestcommon.RunResult{}, nil
}
