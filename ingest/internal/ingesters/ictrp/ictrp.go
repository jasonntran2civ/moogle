// Package ictrp ingests WHO ICTRP weekly bulk XML (spec §5.1.4).
//
// Bulk weekly XML at https://trialsearch.who.int/ -> TrialResults.zip.
// Diff against previous snapshot stored in R2 to emit only new/changed
// trials.
//
// TODO: implement zip download + diff. This skeleton stubs the run
// loop so the service builds and deploys; first real run is a manual
// trigger after the diff strategy is finalized.
package ictrp

import (
	"context"
	"log/slog"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

type Ingester struct {
	logger   *slog.Logger
	wm       *watermark.Store
	archiver *r2.Archiver
	pub      *pubsubpub.Publisher
}

func New(logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{logger: logger, wm: wm, archiver: arch, pub: pub}
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	_ = i.wm.MarkRunning(ctx, "ictrp")
	i.logger.Warn("ictrp ingester is a stub; no-op until zip+diff implemented")
	_ = i.wm.Set(ctx, "ictrp", "stub", "idle", "")
	return ingestcommon.RunResult{}, nil
}
