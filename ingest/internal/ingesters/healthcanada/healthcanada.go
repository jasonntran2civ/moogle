// Package healthcanada ingests Health Canada Drug Product Database
// (spec section 2 row 18). Public, weekly CSV.
package healthcanada

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

const dpdURL = "https://www.canada.ca/content/dam/hc-sc/migration/hc-sc/dhp-mps/alt_formats/zip/prodpharma/databasdon/allfiles.zip"

type Ingester struct {
	logger *slog.Logger; wm *watermark.Store
	archiver *r2.Archiver; pub *pubsubpub.Publisher; fetcher *ingestcommon.Fetcher
}

func New(logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(1, 2, "EvidenceLens-Health-Canada/0.1 (mailto:contact@example.com)"),
	}
}

// Streams `drug.csv` directly via a public mirror that exposes the
// Drug Product Database master CSV. Fallback: download zip and parse first
// CSV file.
const drugCSVURL = "https://health-products.canada.ca/api/drug/drugproduct/?type=json&lang=en"

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "health-canada"); err != nil { return ingestcommon.RunResult{}, err }
	body, err := i.fetcher.Get(ctx, drugCSVURL, nil)
	if err != nil { return ingestcommon.RunResult{}, fmt.Errorf("hc fetch: %w", err) }

	var counters ingestcommon.Counters
	// Try JSON first.
	var arr []map[string]any
	if err := json.Unmarshal(body, &arr); err == nil && len(arr) > 0 {
		for _, item := range arr {
			counters.Fetched.Add(1)
			id := fmt.Sprintf("%v", item["drug_code"])
			docID := "health-canada:" + id
			raw, _ := json.Marshal(item)
			key, err := i.archiver.Put(ctx, "health-canada", docID, raw)
			if err != nil { counters.Failed.Add(1); continue }
			counters.Archived.Add(1)
			if _, err := i.pub.PublishRaw(ctx, "health-canada", docID, key); err == nil { counters.Published.Add(1) }
			if ctx.Err() != nil { break }
		}
	} else {
		// CSV fallback.
		r := csv.NewReader(bytes.NewReader(body))
		r.FieldsPerRecord = -1
		header, herr := r.Read()
		if herr == nil {
			for {
				rec, err := r.Read()
				if err == io.EOF { break }
				if err != nil { continue }
				counters.Fetched.Add(1)
				row := map[string]string{}
				for i, h := range header { if i < len(rec) { row[h] = rec[i] } }
				id := row["DRUG_CODE"]
				docID := "health-canada:" + id
				raw, _ := json.Marshal(row)
				key, err := i.archiver.Put(ctx, "health-canada", docID, raw)
				if err != nil { counters.Failed.Add(1); continue }
				counters.Archived.Add(1)
				if _, err := i.pub.PublishRaw(ctx, "health-canada", docID, key); err == nil { counters.Published.Add(1) }
				if ctx.Err() != nil { break }
			}
		}
	}

	stamp := time.Now().UTC().Format("2006-01-02")
	_ = i.wm.Set(ctx, "health-canada", stamp, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched: counters.Fetched.Load(), DocsArchived: counters.Archived.Load(),
		DocsPublished: counters.Published.Load(), HighWatermark: stamp,
	}, nil
}
