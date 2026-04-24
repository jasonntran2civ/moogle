// Package nihreporter ingests NIH RePORTER funding records (spec §5.1.9).
// REST: api.reporter.nih.gov/v2/projects/search with date filters.
package nihreporter

import (
	"bytes"
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
		fetcher: ingestcommon.NewFetcher(1, 2, "EvidenceLens-NIH-RePORTER/0.1 (mailto:contact@example.com)"),
	}
}

type searchReq struct {
	Criteria struct {
		ProjectStartDate struct {
			FromDate string `json:"from_date"`
			ToDate   string `json:"to_date"`
		} `json:"project_start_date"`
	} `json:"criteria"`
	Offset        int      `json:"offset"`
	Limit         int      `json:"limit"`
	SortField     string   `json:"sort_field"`
	SortOrder     string   `json:"sort_order"`
	IncludeFields []string `json:"include_fields"`
}

type nihProject struct {
	ApplID       int64   `json:"appl_id"`
	ProjectNum   string  `json:"project_num"`
	ProjectTitle string  `json:"project_title"`
	AbstractText string  `json:"abstract_text"`
	AwardAmount  float64 `json:"award_amount"`
	FiscalYear   int     `json:"fiscal_year"`
	OrgName      string  `json:"org_name"`
	OrgState     string  `json:"org_state"`
	ProjectStart string  `json:"project_start_date"`
	ProjectEnd   string  `json:"project_end_date"`
	PIs          []struct {
		FullName string `json:"full_name"`
	} `json:"principal_investigators"`
	AgencyIcAdmin struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"agency_ic_admin"`
}

type searchResp struct {
	Meta struct {
		Total int `json:"total"`
	} `json:"meta"`
	Results []nihProject `json:"results"`
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "nih-reporter"); err != nil {
		return ingestcommon.RunResult{}, err
	}
	hwm, _ := i.wm.Get(ctx, "nih-reporter")
	if hwm == "" {
		hwm = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	}
	to := time.Now().Format("2006-01-02")
	var counters ingestcommon.Counters

	offset := 0
	for int(counters.Fetched.Load()) < i.cfg.MaxPerRun {
		req := searchReq{
			Offset:    offset,
			Limit:     500,
			SortField: "appl_id",
			SortOrder: "asc",
			IncludeFields: []string{
				"ApplId", "ProjectNum", "ProjectTitle", "AbstractText",
				"AwardAmount", "FiscalYear", "Organization",
				"PrincipalInvestigators", "AgencyIcAdmin",
				"ProjectStartDate", "ProjectEndDate",
			},
		}
		req.Criteria.ProjectStartDate.FromDate = hwm
		req.Criteria.ProjectStartDate.ToDate = to
		body, err := json.Marshal(req)
		if err != nil {
			break
		}
		respBytes, err := i.fetcher.Post(ctx,
			"https://api.reporter.nih.gov/v2/projects/search",
			bytes.NewReader(body),
			map[string]string{"Content-Type": "application/json"},
		)
		if err != nil {
			i.logger.Warn("nih-reporter fetch", "err", err)
			break
		}
		var resp searchResp
		if err := json.Unmarshal(respBytes, &resp); err != nil || len(resp.Results) == 0 {
			break
		}
		for _, p := range resp.Results {
			counters.Fetched.Add(1)
			raw, _ := json.Marshal(p)
			docID := fmt.Sprintf("nih-reporter:%d", p.ApplID)
			key, err := i.archiver.Put(ctx, "nih-reporter", docID, raw)
			if err != nil {
				counters.Failed.Add(1)
				continue
			}
			counters.Archived.Add(1)
			if _, err := i.pub.PublishRaw(ctx, "nih-reporter", docID, key); err == nil {
				counters.Published.Add(1)
			}
		}
		offset += len(resp.Results)
		if offset >= resp.Meta.Total {
			break
		}
	}

	_ = i.wm.Set(ctx, "nih-reporter", to, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched:   counters.Fetched.Load(),
		DocsArchived:  counters.Archived.Load(),
		DocsPublished: counters.Published.Load(),
		HighWatermark: to,
	}, nil
}
