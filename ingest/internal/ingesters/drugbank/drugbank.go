// Package drugbank ingests DrugBank Open Data XML (spec section 2 row 24).
// Free for academic, registration required. URL configurable via
// DRUGBANK_XML_URL because the download portal is gated.
package drugbank

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

type Ingester struct {
	logger *slog.Logger; wm *watermark.Store
	archiver *r2.Archiver; pub *pubsubpub.Publisher; fetcher *ingestcommon.Fetcher
}

func New(logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(1, 2, "EvidenceLens-DrugBank/0.1 (mailto:contact@example.com)"),
	}
}

type dbRoot struct {
	XMLName xml.Name `xml:"drugbank"`
	Drugs   []dbDrug `xml:"drug"`
}

type dbDrug struct {
	Type        string `xml:"type,attr"`
	DrugbankID  string `xml:"drugbank-id"`
	Name        string `xml:"name"`
	Description string `xml:"description"`
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "drugbank"); err != nil { return ingestcommon.RunResult{}, err }
	src := os.Getenv("DRUGBANK_XML_URL")
	if src == "" {
		i.logger.Warn("drugbank: DRUGBANK_XML_URL not set; skipping (registration required)")
		return ingestcommon.RunResult{}, nil
	}
	body, err := i.fetcher.Get(ctx, src, nil)
	if err != nil { return ingestcommon.RunResult{}, fmt.Errorf("download: %w", err) }
	var root dbRoot
	if err := xml.Unmarshal(body, &root); err != nil { return ingestcommon.RunResult{}, fmt.Errorf("parse: %w", err) }
	var counters ingestcommon.Counters
	for _, d := range root.Drugs {
		counters.Fetched.Add(1)
		if d.DrugbankID == "" { counters.Failed.Add(1); continue }
		docID := "drugbank:" + d.DrugbankID
		raw, _ := json.Marshal(d)
		key, err := i.archiver.Put(ctx, "drugbank", docID, raw)
		if err != nil { counters.Failed.Add(1); continue }
		counters.Archived.Add(1)
		if _, err := i.pub.PublishRaw(ctx, "drugbank", docID, key); err == nil { counters.Published.Add(1) }
	}
	stamp := time.Now().UTC().Format("2006-01-02")
	_ = i.wm.Set(ctx, "drugbank", stamp, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched: counters.Fetched.Load(), DocsArchived: counters.Archived.Load(),
		DocsPublished: counters.Published.Load(), HighWatermark: stamp,
	}, nil
}
