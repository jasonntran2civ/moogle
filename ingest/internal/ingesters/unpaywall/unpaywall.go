// Package unpaywall resolves a DOI to a free OA PDF URL via
// api.unpaywall.org/v2/{doi}?email=... (spec §5.1.8). Stub.
package unpaywall

import (
	"context"
	"log/slog"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

type Config struct{ Email string }

type Ingester struct {
	cfg Config; logger *slog.Logger; wm *watermark.Store
	archiver *r2.Archiver; pub *pubsubpub.Publisher
}

func New(cfg Config, logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{cfg: cfg, logger: logger, wm: wm, archiver: arch, pub: pub}
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	_ = i.wm.MarkRunning(ctx, "unpaywall")
	i.logger.Info("unpaywall: trigger-on-demand by processor; no scheduled work")
	return ingestcommon.RunResult{}, nil
}
