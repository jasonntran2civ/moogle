// Cloud Run entrypoint for ingester-pubmed.
//
// Skeleton lifted from Moogle's spider/cmd/spider/main.go pattern (env
// helpers, signal handling, graceful shutdown), extended for Cloud Run's
// HTTP /run invocation model and OTel tracing.
package main

import (
	"context"
	"log/slog"

	"github.com/evidencelens/evidencelens/ingest/internal/ingesters/pubmed"
	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/otel"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

const serviceName = "ingester-pubmed"

func main() {
	ctx, cancel := ingestcommon.Setup(context.Background())
	defer cancel()
	logger := ingestcommon.MustNewLogger(serviceName)

	shutdown, err := otel.Init(ctx, serviceName)
	if err != nil {
		logger.Warn("otel init failed; continuing without tracing", "err", err)
	} else {
		defer func() { _ = shutdown(context.Background()) }()
	}

	wm, err := watermark.New(ctx, ingestcommon.MustEnv("DATABASE_URL"))
	if err != nil {
		logger.Error("watermark store", "err", err)
		return
	}
	defer wm.Close()

	archiver, err := r2.New(
		ingestcommon.MustEnv("R2_ACCOUNT_ID"),
		ingestcommon.MustEnv("R2_ACCESS_KEY_ID"),
		ingestcommon.MustEnv("R2_SECRET_ACCESS_KEY"),
		ingestcommon.MustEnv("R2_BUCKET"),
		ingestcommon.MustEnv("R2_ENDPOINT"),
	)
	if err != nil {
		logger.Error("r2 init", "err", err)
		return
	}

	pub, err := pubsubpub.New(ctx, ingestcommon.MustEnv("GCP_PROJECT"), ingestcommon.GetEnv("PUBSUB_TOPIC_RAW_DOCS", "raw-docs"))
	if err != nil {
		logger.Error("pubsub init", "err", err)
		return
	}
	defer pub.Close()

	ing := pubmed.New(pubmed.Config{
		APIKey:    ingestcommon.GetEnv("NCBI_API_KEY", ""),
		Tool:      ingestcommon.GetEnv("NCBI_TOOL", "evidencelens"),
		Email:     ingestcommon.GetEnv("NCBI_EMAIL", "contact@example.com"),
		MaxPerRun: ingestcommon.GetEnvInt("PUBMED_MAX_PER_RUN", 5000),
	}, logger, wm, archiver, pub)

	runner := &ingestcommon.Runner{
		Source: "pubmed",
		Logger: logger,
		Run:    ing.Run,
	}

	if err := ingestcommon.ServeRun(ctx, runner); err != nil {
		logger.Error("http server", "err", err)
	}
	slog.Info("shutdown complete")
}
