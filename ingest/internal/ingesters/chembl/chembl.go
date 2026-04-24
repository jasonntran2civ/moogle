// Package chembl ingests ChEMBL bioactivity records via the public REST
// API at chembl.ebi.ac.uk/api/data/molecule.json (spec section 2 row 25).
// Public, no key required.
package chembl

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

type Config struct{ MaxPerRun int }

type Ingester struct {
	cfg Config; logger *slog.Logger; wm *watermark.Store
	archiver *r2.Archiver; pub *pubsubpub.Publisher; fetcher *ingestcommon.Fetcher
}

func New(cfg Config, logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		cfg: cfg, logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(3, 6, "EvidenceLens-ChEMBL/0.1 (mailto:contact@example.com)"),
	}
}

type chemblPage struct {
	Molecules []map[string]any `json:"molecules"`
	PageMeta  struct{ NextURL string `json:"next"` } `json:"page_meta"`
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "chembl"); err != nil { return ingestcommon.RunResult{}, err }
	hwm, _ := i.wm.Get(ctx, "chembl")
	const base = "https://www.ebi.ac.uk/chembl/api/data/molecule.json?limit=200"
	url := base
	if hwm != "" { url = base + "&offset=" + hwm }
	var counters ingestcommon.Counters
	for int(counters.Fetched.Load()) < i.cfg.MaxPerRun {
		body, err := i.fetcher.Get(ctx, url, nil)
		if err != nil { break }
		var p chemblPage
		if err := json.Unmarshal(body, &p); err != nil || len(p.Molecules) == 0 { break }
		for _, m := range p.Molecules {
			counters.Fetched.Add(1)
			id := fmt.Sprintf("%v", m["molecule_chembl_id"])
			docID := "chembl:" + id
			raw, _ := json.Marshal(m)
			key, err := i.archiver.Put(ctx, "chembl", docID, raw)
			if err != nil { counters.Failed.Add(1); continue }
			counters.Archived.Add(1)
			if _, err := i.pub.PublishRaw(ctx, "chembl", docID, key); err == nil { counters.Published.Add(1) }
		}
		if p.PageMeta.NextURL == "" { break }
		url = "https://www.ebi.ac.uk" + p.PageMeta.NextURL + ".json"
	}
	stamp := time.Now().UTC().Format("2006-01-02")
	_ = i.wm.Set(ctx, "chembl", stamp, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched: counters.Fetched.Load(), DocsArchived: counters.Archived.Load(),
		DocsPublished: counters.Published.Load(), HighWatermark: stamp,
	}, nil
}
