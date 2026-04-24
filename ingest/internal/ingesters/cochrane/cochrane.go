// Package cochrane ingests Cochrane systematic reviews via RSS + DOI
// resolution (spec §5.1.11).
//
// Free for academic only — never serve full content, only metadata +
// deep links. Documented in docs/sources/cochrane.md.
//
// TODO: implement RSS poll + per-review DOI -> Crossref enrich.
package cochrane

import (
	"context"
	"log/slog"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

type Ingester struct {
	logger *slog.Logger; wm *watermark.Store
	archiver *r2.Archiver; pub *pubsubpub.Publisher
}

func New(logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{logger: logger, wm: wm, archiver: arch, pub: pub}
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	_ = i.wm.MarkRunning(ctx, "cochrane")
	i.logger.Warn("cochrane ingester stub: RSS poll TODO")
	return ingestcommon.RunResult{}, nil
}
