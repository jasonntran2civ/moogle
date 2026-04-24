// ingester-fda: openFDA drug + device endpoints (spec section 5.1.5).
//
// Recall events get a priority lane: in addition to the normal raw-docs
// publish, recalls also publish a RecallEvent to NATS recall-fanout for
// the gateway WS subscribers (SLO ≤ 1min E2E per spec section 14.1).
package main

import (
	"context"

	"github.com/evidencelens/evidencelens/ingest/internal/ingesters/fda"
	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/otel"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

func main() {
	ctx, cancel := ingestcommon.Setup(context.Background())
	defer cancel()
	logger := ingestcommon.MustNewLogger("ingester-fda")
	shutdown, _ := otel.Init(ctx, "ingester-fda")
	defer func() { if shutdown != nil { _ = shutdown(context.Background()) } }()

	wm, err := watermark.New(ctx, ingestcommon.MustEnv("DATABASE_URL"))
	if err != nil { logger.Error("watermark", "err", err); return }
	defer wm.Close()
	arch, err := r2.New(ingestcommon.MustEnv("R2_ACCOUNT_ID"), ingestcommon.MustEnv("R2_ACCESS_KEY_ID"), ingestcommon.MustEnv("R2_SECRET_ACCESS_KEY"), ingestcommon.MustEnv("R2_BUCKET"), ingestcommon.MustEnv("R2_ENDPOINT"))
	if err != nil { logger.Error("r2", "err", err); return }
	pub, err := pubsubpub.New(ctx, ingestcommon.MustEnv("GCP_PROJECT"), ingestcommon.GetEnv("PUBSUB_TOPIC_RAW_DOCS", "raw-docs"))
	if err != nil { logger.Error("pubsub", "err", err); return }
	defer pub.Close()

	ing := fda.New(fda.Config{
		APIKey:    ingestcommon.GetEnv("OPENFDA_API_KEY", ""),
		MaxPerRun: ingestcommon.GetEnvInt("FDA_MAX_PER_RUN", 5000),
		Endpoints: []string{"drug/drugsfda", "drug/enforcement", "device/event", "device/510k"},
		NATSURL:   ingestcommon.GetEnv("NATS_URL", "nats://localhost:4222"),
	}, logger, wm, arch, pub)
	runner := &ingestcommon.Runner{Source: "fda", Logger: logger, Run: ing.Run}
	if err := ingestcommon.ServeRun(ctx, runner); err != nil { logger.Error("http", "err", err) }
}
