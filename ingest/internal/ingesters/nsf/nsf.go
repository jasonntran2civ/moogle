// Package nsf ingests NSF Award Search records (spec section 2 row 23).
// Public REST: api.nsf.gov/services/v1/awards.json
package nsf

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

type Config struct{ MaxPerRun int }
type Ingester struct {
	cfg Config; logger *slog.Logger; wm *watermark.Store
	archiver *r2.Archiver; pub *pubsubpub.Publisher; fetcher *ingestcommon.Fetcher
}

func New(cfg Config, logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		cfg: cfg, logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(2, 4, "EvidenceLens-NSF/0.1 (mailto:contact@example.com)"),
	}
}

type awardsResp struct {
	Response struct {
		Award []map[string]any `json:"award"`
	} `json:"response"`
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "nsf"); err != nil { return ingestcommon.RunResult{}, err }
	hwm, _ := i.wm.Get(ctx, "nsf")
	if hwm == "" { hwm = time.Now().AddDate(0, 0, -7).Format("01/02/2006") } // NSF uses MM/DD/YYYY
	to := time.Now().Format("01/02/2006")
	var counters ingestcommon.Counters
	offset := 1
	for int(counters.Fetched.Load()) < i.cfg.MaxPerRun {
		q := url.Values{}
		q.Set("dateStart", hwm)
		q.Set("dateEnd", to)
		q.Set("offset", fmt.Sprintf("%d", offset))
		q.Set("printFields",
			"id,title,abstractText,piFirstName,piLastName,awardeeName,awardeeStateCode,fundsObligatedAmt,startDate,expDate")
		req := "https://api.nsf.gov/services/v1/awards.json?" + q.Encode()
		body, err := i.fetcher.Get(ctx, req, nil)
		if err != nil { break }
		var r awardsResp
		if err := json.Unmarshal(body, &r); err != nil || len(r.Response.Award) == 0 { break }
		for _, a := range r.Response.Award {
			counters.Fetched.Add(1)
			id := fmt.Sprintf("%v", a["id"])
			if id == "" || id == "<nil>" { counters.Failed.Add(1); continue }
			docID := "nsf:" + id
			raw, _ := json.Marshal(a)
			key, err := i.archiver.Put(ctx, "nsf", docID, raw)
			if err != nil { counters.Failed.Add(1); continue }
			counters.Archived.Add(1)
			if _, err := i.pub.PublishRaw(ctx, "nsf", docID, key); err == nil { counters.Published.Add(1) }
		}
		offset += len(r.Response.Award)
	}
	stamp := time.Now().Format("2006-01-02")
	_ = i.wm.Set(ctx, "nsf", to, "idle", "")
	_ = stamp
	return ingestcommon.RunResult{
		DocsFetched: counters.Fetched.Load(), DocsArchived: counters.Archived.Load(),
		DocsPublished: counters.Published.Load(), HighWatermark: to,
	}, nil
}
