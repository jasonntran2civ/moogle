// Package guidelines scrapes USPSTF + NICE + AHRQ HTML pages with Colly
// (spec §5.1.12). Per-source crawl rules in YAML config.
//
// Render-to-markdown via gomarkdownify or external mdream; result body
// stored in R2 raw archive.
//
// TODO: implement Colly crawl + markdown render + diff.
package guidelines

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
	_ = i.wm.MarkRunning(ctx, "guidelines")
	i.logger.Warn("guidelines ingester stub: Colly crawl TODO")
	return ingestcommon.RunResult{}, nil
}
