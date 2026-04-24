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
	"github.com/nats-io/nats.go"
)

type Config struct {
	APIKey    string
	MaxPerRun int
	Endpoints []string
	NATSURL   string // for recall-fanout priority publish
}

type Ingester struct {
	cfg       Config
	logger    *slog.Logger
	wm        *watermark.Store
	archiver  *r2.Archiver
	pub       *pubsubpub.Publisher
	fetcher   *ingestcommon.Fetcher
	recallPub *nats.Conn
}

func New(cfg Config, logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	i := &Ingester{
		cfg: cfg, logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(4, 8, "EvidenceLens-FDA/0.1 (mailto:contact@example.com)"),
	}
	if cfg.NATSURL != "" {
		if nc, err := nats.Connect(cfg.NATSURL,
			nats.MaxReconnects(-1),
			nats.ReconnectWait(time.Second),
		); err == nil {
			i.recallPub = nc
			logger.Info("fda recall priority lane connected", "nats", cfg.NATSURL)
		} else {
			logger.Warn("fda recall priority lane disabled (nats unreachable)", "err", err)
		}
	}
	return i
}

// publishRecall tees a RecallEvent JSON to NATS subject `recall-fanout`.
// Best-effort: failures are logged and never block the main publish path.
func (i *Ingester) publishRecall(ctx context.Context, recallID string, r map[string]any) {
	openfda, _ := r["openfda"].(map[string]any)
	drugClass := ""
	if cls, ok := openfda["pharm_class_epc"].([]any); ok && len(cls) > 0 {
		if s, ok := cls[0].(string); ok {
			drugClass = s
		}
	}
	productName := ""
	if v, ok := r["product_description"].(string); ok {
		productName = v
	}
	recallClass := ""
	if v, ok := r["classification"].(string); ok {
		recallClass = v
	}
	payload, _ := json.Marshal(map[string]any{
		"recall_id":    recallID,
		"agency":       "fda",
		"product_name": productName,
		"drug_class":   drugClass,
		"recall_class": recallClass,
		"emitted_at":   time.Now().UTC().Format(time.RFC3339),
	})
	if err := i.recallPub.Publish("recall-fanout", payload); err != nil {
		i.logger.Warn("recall-fanout publish failed", "recall_id", recallID, "err", err)
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
				// Recall priority lane: tee a RecallEvent into the
				// recall-fanout NATS bridge so the gateway WS subscribers
				// see it within the spec section 14.1 1-min E2E SLO.
				if isRecall && i.recallPub != nil {
					i.publishRecall(ctx, id, r)
				}
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
