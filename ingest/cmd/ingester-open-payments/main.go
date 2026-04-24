// ingester-open-payments: CMS Open Payments annual bulk CSV (spec section
// 5.1.10). FLAGSHIP — drives every COIBadge in the frontend.
//
// Two roles:
//   1. Annual bulk fetch: download CSV from
//      download.cms.gov/openpayments/PGYY_P0NNNNNN.zip and bulk-load into
//      Postgres open_payments table.
//   2. /lookup endpoint: synchronous HTTP API consumed by the processor's
//      author-payment-joiner to fuzzy-match author -> NPI/payments.
package main

import (
	"context"

	"github.com/evidencelens/evidencelens/ingest/internal/ingesters/openpayments"
	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/otel"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
)

func main() {
	ctx, cancel := ingestcommon.Setup(context.Background())
	defer cancel()
	logger := ingestcommon.MustNewLogger("ingester-open-payments")
	shutdown, _ := otel.Init(ctx, "ingester-open-payments")
	defer func() { if shutdown != nil { _ = shutdown(context.Background()) } }()

	dsn := ingestcommon.MustEnv("DATABASE_URL")
	arch, err := r2.New(ingestcommon.MustEnv("R2_ACCOUNT_ID"), ingestcommon.MustEnv("R2_ACCESS_KEY_ID"), ingestcommon.MustEnv("R2_SECRET_ACCESS_KEY"), ingestcommon.MustEnv("R2_BUCKET"), ingestcommon.MustEnv("R2_ENDPOINT"))
	if err != nil { logger.Error("r2", "err", err); return }

	srv := openpayments.NewServer(openpayments.Config{
		DatabaseURL:        dsn,
		MinFuzzyConfidence: 0.90,
	}, logger, arch)
	if err := srv.ListenAndServe(ctx); err != nil {
		logger.Error("server", "err", err)
	}
}
