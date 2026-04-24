package main

import (
	"context"

	"github.com/evidencelens/evidencelens/ingest/internal/ingesters/nsf"
	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/otel"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

func main() {
	ctx, cancel := ingestcommon.Setup(context.Background())
	defer cancel()
	logger := ingestcommon.MustNewLogger("ingester-nsf")
	shutdown, _ := otel.Init(ctx, "ingester-nsf")
	defer func() { if shutdown != nil { _ = shutdown(context.Background()) } }()
	wm, err := watermark.New(ctx, ingestcommon.MustEnv("DATABASE_URL"))
	if err != nil { logger.Error("watermark", "err", err); return }
	defer wm.Close()
	arch, err := r2.New(ingestcommon.MustEnv("R2_ACCOUNT_ID"), ingestcommon.MustEnv("R2_ACCESS_KEY_ID"), ingestcommon.MustEnv("R2_SECRET_ACCESS_KEY"), ingestcommon.MustEnv("R2_BUCKET"), ingestcommon.MustEnv("R2_ENDPOINT"))
	if err != nil { logger.Error("r2", "err", err); return }
	pub, err := pubsubpub.New(ctx, ingestcommon.MustEnv("GCP_PROJECT"), ingestcommon.GetEnv("PUBSUB_TOPIC_RAW_DOCS", "raw-docs"))
	if err != nil { logger.Error("pubsub", "err", err); return }
	defer pub.Close()
	ing := nsf.New(nsf.Config{MaxPerRun: ingestcommon.GetEnvInt("NSF_MAX_PER_RUN", 5000)}, logger, wm, arch, pub)
	runner := &ingestcommon.Runner{Source: "nsf", Logger: logger, Run: ing.Run}
	if err := ingestcommon.ServeRun(ctx, runner); err != nil { logger.Error("http", "err", err) }
}
