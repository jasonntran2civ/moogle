// Package disgenet ingests gene–disease associations from DisGeNET
// (spec section 2 row 28). REST: api.disgenet.com (CC-BY-NC).
// Requires DISGENET_API_KEY (free academic).
package disgenet

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
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
	apiKey string
}

func New(cfg Config, logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		cfg: cfg, logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(1, 2, "EvidenceLens-DisGeNET/0.1 (mailto:contact@example.com)"),
		apiKey:  os.Getenv("DISGENET_API_KEY"),
	}
}

type page struct {
	Status string `json:"status"`
	Payload []map[string]any `json:"payload"`
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "disgenet"); err != nil { return ingestcommon.RunResult{}, err }
	if i.apiKey == "" {
		i.logger.Warn("disgenet: DISGENET_API_KEY not set; skipping")
		return ingestcommon.RunResult{}, nil
	}
	hwm, _ := i.wm.Get(ctx, "disgenet")
	startPage, _ := strconvAtoiSafe(hwm, 0)
	var counters ingestcommon.Counters
	for p := startPage; int(counters.Fetched.Load()) < i.cfg.MaxPerRun; p++ {
		q := url.Values{}
		q.Set("page_number", fmt.Sprintf("%d", p))
		q.Set("per_page", "100")
		req := "https://api.disgenet.com/api/v1/gda/summary?" + q.Encode()
		body, err := i.fetcher.Get(ctx, req, map[string]string{"Authorization": i.apiKey})
		if err != nil { break }
		var r page
		if err := json.Unmarshal(body, &r); err != nil || len(r.Payload) == 0 { break }
		for _, item := range r.Payload {
			counters.Fetched.Add(1)
			id := fmt.Sprintf("%v_%v", item["gene_symbol_of_intersection"], item["disease_id_of_intersection"])
			docID := "disgenet:" + id
			raw, _ := json.Marshal(item)
			key, err := i.archiver.Put(ctx, "disgenet", docID, raw)
			if err != nil { counters.Failed.Add(1); continue }
			counters.Archived.Add(1)
			if _, err := i.pub.PublishRaw(ctx, "disgenet", docID, key); err == nil { counters.Published.Add(1) }
		}
		startPage = p + 1
	}
	stamp := fmt.Sprintf("%d", startPage)
	_ = i.wm.Set(ctx, "disgenet", stamp, "idle", "")
	_ = time.Now
	return ingestcommon.RunResult{
		DocsFetched: counters.Fetched.Load(), DocsArchived: counters.Archived.Load(),
		DocsPublished: counters.Published.Load(), HighWatermark: stamp,
	}, nil
}

func strconvAtoiSafe(s string, fb int) (int, error) {
	if s == "" { return fb, nil }
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil { return fb, err }
	return n, nil
}
