// Package omim ingests OMIM gene–disease catalog via api.omim.org
// (spec section 2 row 26). Requires OMIM_API_KEY (free for research).
package omim

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
		fetcher: ingestcommon.NewFetcher(2, 4, "EvidenceLens-OMIM/0.1 (mailto:contact@example.com)"),
		apiKey:  os.Getenv("OMIM_API_KEY"),
	}
}

type omimResp struct {
	OmimResponse struct {
		EntryList []struct {
			Entry map[string]any `json:"entry"`
		} `json:"entryList"`
	} `json:"omim"`
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "omim"); err != nil { return ingestcommon.RunResult{}, err }
	if i.apiKey == "" {
		i.logger.Warn("omim: OMIM_API_KEY not set; skipping")
		return ingestcommon.RunResult{}, nil
	}
	hwm, _ := i.wm.Get(ctx, "omim")
	startMim, _ := strconvAtoiSafe(hwm, 100000)
	var counters ingestcommon.Counters
	mim := startMim
	for int(counters.Fetched.Load()) < i.cfg.MaxPerRun {
		batch := make([]string, 0, 20)
		for j := 0; j < 20; j++ {
			batch = append(batch, fmt.Sprintf("%d", mim+j))
		}
		mim += 20
		q := url.Values{}
		q.Set("mimNumber", joinComma(batch))
		q.Set("apiKey", i.apiKey)
		q.Set("format", "json")
		req := "https://api.omim.org/api/entry?" + q.Encode()
		body, err := i.fetcher.Get(ctx, req, nil)
		if err != nil { break }
		var r omimResp
		if err := json.Unmarshal(body, &r); err != nil { break }
		if len(r.OmimResponse.EntryList) == 0 { continue }
		for _, e := range r.OmimResponse.EntryList {
			counters.Fetched.Add(1)
			mimNum := fmt.Sprintf("%v", e.Entry["mimNumber"])
			if mimNum == "<nil>" || mimNum == "" { counters.Failed.Add(1); continue }
			docID := "omim:" + mimNum
			raw, _ := json.Marshal(e.Entry)
			key, err := i.archiver.Put(ctx, "omim", docID, raw)
			if err != nil { counters.Failed.Add(1); continue }
			counters.Archived.Add(1)
			if _, err := i.pub.PublishRaw(ctx, "omim", docID, key); err == nil { counters.Published.Add(1) }
		}
	}
	stamp := fmt.Sprintf("%d", mim)
	_ = i.wm.Set(ctx, "omim", stamp, "idle", "")
	_ = time.Now
	return ingestcommon.RunResult{
		DocsFetched: counters.Fetched.Load(), DocsArchived: counters.Archived.Load(),
		DocsPublished: counters.Published.Load(), HighWatermark: stamp,
	}, nil
}

func strconvAtoiSafe(s string, fallback int) (int, error) {
	if s == "" { return fallback, nil }
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil { return fallback, err }
	return n, nil
}

func joinComma(xs []string) string {
	out := ""
	for i, x := range xs {
		if i > 0 { out += "," }
		out += x
	}
	return out
}
