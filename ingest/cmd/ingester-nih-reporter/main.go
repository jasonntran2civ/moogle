// ingester-nih-reporter: NIH RePORTER funding records (spec section 5.1.9).
package main

import (
	"context"

	"github.com/evidencelens/evidencelens/ingest/internal/ingesters/nihreporter"
	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/otel"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

func main() {
	ctx, cancel := ingestcommon.Setup(context.Background())
	defer cancel()
	logger := ingestcommon.MustNewLogger("ingester-nih-reporter")
	shutdown, _ := otel.Init(ctx, "ingester-nih-reporter")
	defer func() { if shutdown != nil { _ = shutdown(context.Background()) } }()

	wm, _ := watermark.New(ctx, ingestcommon.MustEnv("DATABASE_URL"))
	defer wm.Close()
	arch, _ := r2.New(ingestcommon.MustEnv("R2_ACCOUNT_ID"), ingestcommon.MustEnv("R2_ACCESS_KEY_ID"), ingestcommon.MustEnv("R2_SECRET_ACCESS_KEY"), ingestcommon.MustEnv("R2_BUCKET"), ingestcommon.MustEnv("R2_ENDPOINT"))
	pub, _ := pubsubpub.New(ctx, ingestcommon.MustEnv("GCP_PROJECT"), ingestcommon.GetEnv("PUBSUB_TOPIC_RAW_DOCS", "raw-docs"))
	defer pub.Close()

	ing := nihreporter.New(nihreporter.Config{MaxPerRun: ingestcommon.GetEnvInt("NIH_MAX_PER_RUN", 5000)}, logger, wm, arch, pub)
	runner := &ingestcommon.Runner{Source: "nih-reporter", Logger: logger, Run: ing.Run}
	if err := ingestcommon.ServeRun(ctx, runner); err != nil { logger.Error("http", "err", err) }
}
