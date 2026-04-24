// Package preprint ingests bioRxiv + medRxiv via the bioRxiv API
// (spec §5.1.2). Both servers share the same schema; one ingester serves
// both via the Servers config.
//
// Endpoint: api.biorxiv.org/details/{server}/{from-date}/{to-date}/{cursor}
// Watermark: ISO date of last successful fetch.
package preprint

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

type Config struct {
	Servers   []string // ["biorxiv", "medrxiv"]
	MaxPerRun int
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
		cfg:      cfg,
		logger:   logger,
		wm:       wm,
		archiver: arch,
		pub:      pub,
		fetcher:  ingestcommon.NewFetcher(5, 10, "EvidenceLens-preprint/0.1 (mailto:contact@example.com)"),
	}
}

type detailsResponse struct {
	Messages []json.RawMessage `json:"messages"`
	Collection []struct {
		DOI            string `json:"doi"`
		Title          string `json:"title"`
		Date           string `json:"date"`
		Server         string `json:"server"`
	} `json:"collection"`
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "preprint"); err != nil {
		return ingestcommon.RunResult{}, err
	}
	hwm, _ := i.wm.Get(ctx, "preprint")
	if hwm == "" {
		hwm = time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	}
	to := time.Now().Format("2006-01-02")
	var counters ingestcommon.Counters

	for _, server := range i.cfg.Servers {
		cursor := 0
		for {
			if int(counters.Fetched.Load()) >= i.cfg.MaxPerRun {
				break
			}
			url := fmt.Sprintf("https://api.biorxiv.org/details/%s/%s/%s/%d", server, hwm, to, cursor)
			body, err := i.fetcher.Get(ctx, url, nil)
			if err != nil {
				i.logger.Warn("preprint fetch", "server", server, "err", err)
				break
			}
			var resp detailsResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				break
			}
			if len(resp.Collection) == 0 {
				break
			}
			for _, item := range resp.Collection {
				counters.Fetched.Add(1)
				docID := fmt.Sprintf("%s:%s", server, item.DOI)
				rawJSON, _ := json.Marshal(item)
				key, err := i.archiver.Put(ctx, server, docID, rawJSON)
				if err != nil {
					counters.Failed.Add(1)
					continue
				}
				counters.Archived.Add(1)
				if _, err := i.pub.PublishRaw(ctx, server, docID, key); err == nil {
					counters.Published.Add(1)
				}
			}
			cursor += len(resp.Collection)
		}
	}

	_ = i.wm.Set(ctx, "preprint", to, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched:   counters.Fetched.Load(),
		DocsArchived:  counters.Archived.Load(),
		DocsPublished: counters.Published.Load(),
		HighWatermark: to,
	}, nil
}
