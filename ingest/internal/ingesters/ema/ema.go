// Package ema ingests EMA OpenData CSVs (spec section 2 row 16).
// Public, weekly snapshots.
package ema

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

const csvURL = "https://www.ema.europa.eu/sites/default/files/Medicines_output_european_public_assessment_reports.csv"

type Ingester struct {
	logger *slog.Logger; wm *watermark.Store
	archiver *r2.Archiver; pub *pubsubpub.Publisher; fetcher *ingestcommon.Fetcher
}

func New(logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(1, 2, "EvidenceLens-EMA/0.1 (mailto:contact@example.com)"),
	}
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "ema"); err != nil { return ingestcommon.RunResult{}, err }
	body, err := i.fetcher.Get(ctx, csvURL, nil)
	if err != nil { return ingestcommon.RunResult{}, fmt.Errorf("download csv: %w", err) }
	r := csv.NewReader(bytes.NewReader(body))
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil { return ingestcommon.RunResult{}, fmt.Errorf("header: %w", err) }
	var counters ingestcommon.Counters
	for {
		rec, err := r.Read()
		if err == io.EOF { break }
		if err != nil { continue }
		counters.Fetched.Add(1)
		row := map[string]string{}
		for i, h := range header { if i < len(rec) { row[h] = rec[i] } }
		id := row["Product number"]
		if id == "" { id = row["EMA product number"] }
		if id == "" { counters.Failed.Add(1); continue }
		docID := "ema:" + id
		raw, _ := json.Marshal(row)
		key, err := i.archiver.Put(ctx, "ema", docID, raw)
		if err != nil { counters.Failed.Add(1); continue }
		counters.Archived.Add(1)
		if _, err := i.pub.PublishRaw(ctx, "ema", docID, key); err == nil { counters.Published.Add(1) }
		if ctx.Err() != nil { break }
	}
	stamp := time.Now().UTC().Format("2006-01-02")
	_ = i.wm.Set(ctx, "ema", stamp, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched: counters.Fetched.Load(), DocsArchived: counters.Archived.Load(),
		DocsPublished: counters.Published.Load(), HighWatermark: stamp,
	}, nil
}
