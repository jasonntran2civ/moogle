// Package pmcoa ingests the PubMed Central Open Access Subset
// (spec section 2 row 2) via FTP-served `oa_file_list.csv` then per-record
// XML. Per-article licenses are mixed (CC-BY etc); processor must respect
// per-record license field before storing full text.
package pmcoa

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

const fileListURL = "https://ftp.ncbi.nlm.nih.gov/pub/pmc/oa_file_list.csv"
const articleBase = "https://ftp.ncbi.nlm.nih.gov/pub/pmc/"

type Config struct{ MaxPerRun int }
type Ingester struct {
	cfg Config; logger *slog.Logger; wm *watermark.Store
	archiver *r2.Archiver; pub *pubsubpub.Publisher; fetcher *ingestcommon.Fetcher
}

func New(cfg Config, logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		cfg: cfg, logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(3, 6, "EvidenceLens-PMC-OA/0.1 (mailto:contact@example.com)"),
	}
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "pmc-oa"); err != nil { return ingestcommon.RunResult{}, err }
	hwm, _ := i.wm.Get(ctx, "pmc-oa") // PMCID we last processed
	body, err := i.fetcher.Get(ctx, fileListURL, nil)
	if err != nil { return ingestcommon.RunResult{}, fmt.Errorf("file list: %w", err) }
	r := csv.NewReader(bytes.NewReader(body))
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil { return ingestcommon.RunResult{}, fmt.Errorf("header: %w", err) }
	col := map[string]int{}
	for i, h := range header { col[strings.TrimSpace(h)] = i }
	var counters ingestcommon.Counters
	maxPMCID := hwm
	for int(counters.Fetched.Load()) < i.cfg.MaxPerRun {
		rec, err := r.Read()
		if err == io.EOF { break }
		if err != nil { continue }
		pmcid := safe(rec, col, "Accession ID")
		if pmcid == "" { continue }
		if hwm != "" && pmcid <= hwm { continue }
		fileRel := safe(rec, col, "File")
		license := safe(rec, col, "License")
		if fileRel == "" { continue }
		counters.Fetched.Add(1)
		// Archive metadata only here; the actual XML/PDF fetch happens in
		// the processor when license permits, to avoid pulling closed-access
		// content twice.
		meta := map[string]string{
			"pmcid": pmcid, "file": articleBase + fileRel, "license": license,
		}
		raw, _ := json.Marshal(meta)
		docID := "pmc:" + pmcid
		key, err := i.archiver.Put(ctx, "pmc-oa", docID, raw)
		if err != nil { counters.Failed.Add(1); continue }
		counters.Archived.Add(1)
		if _, err := i.pub.PublishRaw(ctx, "pmc-oa", docID, key); err == nil { counters.Published.Add(1) }
		if pmcid > maxPMCID { maxPMCID = pmcid }
		if ctx.Err() != nil { break }
	}
	if maxPMCID == "" { maxPMCID = time.Now().UTC().Format("2006-01-02") }
	_ = i.wm.Set(ctx, "pmc-oa", maxPMCID, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched: counters.Fetched.Load(), DocsArchived: counters.Archived.Load(),
		DocsPublished: counters.Published.Load(), HighWatermark: maxPMCID,
	}, nil
}

func safe(rec []string, col map[string]int, name string) string {
	if i, ok := col[name]; ok && i < len(rec) { return strings.TrimSpace(rec[i]) }
	return ""
}
