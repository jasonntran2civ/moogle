// Package fda ingests openFDA drug + device endpoints (spec §5.1.5).
//
// Sub-endpoints: drug/drugsfda, drug/enforcement, device/event, device/510k.
// Pagination via skip parameter, max 25,000 per query — partition by date range.
//
// Recalls also publish RecallEvent to NATS recall-fanout (priority lane,
// SLO ≤ 1min E2E).
package fda

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

type Config struct {
	APIKey    string
	MaxPerRun int
	Endpoints []string
	NATSURL   string // for recall-fanout priority publish
}

type Ingester struct {
	cfg      Config
	logger   *slog.Logger
	wm       *watermark.Store
	archiver *r2.Archiver
	pub      *pubsubpub.Publisher
	fetcher  *ingestcommon.Fetcher
}

func New(cfg Config, logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		cfg: cfg, logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(4, 8, "EvidenceLens-FDA/0.1 (mailto:contact@example.com)"),
	}
}

type fdaResponse struct {
	Meta struct {
		Results struct{ Total int `json:"total"` } `json:"results"`
	} `json:"meta"`
	Results []map[string]any `json:"results"`
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "fda"); err != nil {
		return ingestcommon.RunResult{}, err
	}
	hwm, _ := i.wm.Get(ctx, "fda")
	if hwm == "" {
		hwm = time.Now().AddDate(0, 0, -2).Format("20060102")
	}
	to := time.Now().Format("20060102")
	var counters ingestcommon.Counters

	for _, ep := range i.cfg.Endpoints {
		isRecall := ep == "drug/enforcement"
		dateField := pickDateField(ep)
		skip := 0
		for int(counters.Fetched.Load()) < i.cfg.MaxPerRun {
			q := url.Values{}
			q.Set("search", fmt.Sprintf("%s:[%s+TO+%s]", dateField, hwm, to))
			q.Set("limit", "100")
			q.Set("skip", fmt.Sprintf("%d", skip))
			if i.cfg.APIKey != "" {
				q.Set("api_key", i.cfg.APIKey)
			}
			body, err := i.fetcher.Get(ctx, fmt.Sprintf("https://api.fda.gov/%s.json?%s", ep, q.Encode()), nil)
			if err != nil {
				break
			}
			var resp fdaResponse
			if err := json.Unmarshal(body, &resp); err != nil || len(resp.Results) == 0 {
				break
			}
			for _, r := range resp.Results {
				counters.Fetched.Add(1)
				id := pickID(ep, r)
				rawJSON, _ := json.Marshal(r)
				source := "openfda-" + sanitize(ep)
				key, err := i.archiver.Put(ctx, source, id, rawJSON)
				if err != nil {
					counters.Failed.Add(1)
					continue
				}
				counters.Archived.Add(1)
				if _, err := i.pub.PublishRaw(ctx, source, id, key); err == nil {
					counters.Published.Add(1)
				}
				// TODO recall priority lane: publish RecallEvent to NATS recall-fanout.
				_ = isRecall
			}
			skip += len(resp.Results)
		}
	}

	_ = i.wm.Set(ctx, "fda", to, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched: counters.Fetched.Load(), DocsArchived: counters.Archived.Load(),
		DocsPublished: counters.Published.Load(), HighWatermark: to,
	}, nil
}

func pickDateField(ep string) string {
	switch ep {
	case "drug/drugsfda":
		return "submissions.submission_status_date"
	case "drug/enforcement":
		return "report_date"
	case "device/event":
		return "date_received"
	case "device/510k":
		return "decision_date"
	}
	return "report_date"
}

func pickID(ep string, r map[string]any) string {
	keys := map[string]string{
		"drug/drugsfda":    "application_number",
		"drug/enforcement": "recall_number",
		"device/event":     "report_number",
		"device/510k":      "k_number",
	}
	if k, ok := keys[ep]; ok {
		if v, ok := r[k].(string); ok {
			return v
		}
	}
	return fmt.Sprintf("%v", r["@id"])
}

func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '/' {
			out = append(out, '-')
		} else {
			out = append(out, c)
		}
	}
	return string(out)
}
