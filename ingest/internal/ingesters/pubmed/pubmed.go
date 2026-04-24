// Package pubmed implements the PubMed ingester per spec section 5.1.1.
//
// API: NCBI E-utilities (esearch + efetch).
// Watermark: PubMed EDAT (entry date), ISO-8601 string.
// Concurrency: 10 in-flight requests, NCBI rate limit 10/sec with API key.
// First run: TODO bulk-baseline FTP fetch from ftp.ncbi.nlm.nih.gov/pubmed/baseline/.
//   Current implementation pages through esearch+efetch from Unix epoch.
//
// This is the reference ingester — every other ingester follows the same
// shape (Config + Run(ctx) returning RunResult).
package pubmed

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

// Config tunes the ingester. All optional except (implicitly) MaxPerRun.
type Config struct {
	APIKey    string
	Tool      string
	Email     string
	MaxPerRun int
}

// Ingester wires Config + dependencies. Use New to construct.
type Ingester struct {
	cfg      Config
	logger   *slog.Logger
	wm       *watermark.Store
	archiver *r2.Archiver
	pub      *pubsubpub.Publisher
	fetcher  *ingestcommon.Fetcher
}

// New constructs an Ingester. Rate-limit: 10 req/s with key, 3 req/s without.
func New(cfg Config, logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	rate := 3
	if cfg.APIKey != "" {
		rate = 10
	}
	ua := fmt.Sprintf("EvidenceLens-PubMed/%s (mailto:%s)", cfg.Tool, cfg.Email)
	return &Ingester{
		cfg:      cfg,
		logger:   logger,
		wm:       wm,
		archiver: arch,
		pub:      pub,
		fetcher:  ingestcommon.NewFetcher(rate, rate*2, ua),
	}
}

// Run executes one ingestion cycle. Idempotent: rerunning with the same
// watermark is a no-op until new EDAT records appear.
func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "pubmed"); err != nil {
		return ingestcommon.RunResult{}, err
	}

	hwm, err := i.wm.Get(ctx, "pubmed")
	if err != nil {
		return ingestcommon.RunResult{}, err
	}
	if hwm == "" {
		// First run: cap to last 7 days for sanity; bulk baseline is a TODO.
		hwm = time.Now().AddDate(0, 0, -7).Format("2006/01/02")
		i.logger.Info("first run; using 7-day lookback", "from", hwm)
	}

	// 1. esearch — get the PMIDs added since hwm.
	pmids, err := i.esearch(ctx, hwm)
	if err != nil {
		_ = i.wm.Set(ctx, "pubmed", hwm, "failed", err.Error())
		return ingestcommon.RunResult{}, err
	}
	if len(pmids) > i.cfg.MaxPerRun {
		pmids = pmids[:i.cfg.MaxPerRun]
	}
	i.logger.Info("esearch returned", "count", len(pmids), "since", hwm)

	// 2. efetch in batches of 200, 10 in-flight.
	var counters ingestcommon.Counters
	counters.Fetched.Add(int64(len(pmids)))

	batches := chunk(pmids, 200)
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	var hwmMu sync.Mutex
	var maxEDAT string

	for _, batch := range batches {
		batch := batch
		select {
		case <-ctx.Done():
			break
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			records, edat, err := i.efetch(ctx, batch)
			if err != nil {
				counters.Failed.Add(int64(len(batch)))
				i.logger.Warn("efetch batch failed", "size", len(batch), "err", err)
				return
			}
			hwmMu.Lock()
			if edat > maxEDAT {
				maxEDAT = edat
			}
			hwmMu.Unlock()
			for _, raw := range records {
				key, err := i.archiver.Put(ctx, "pubmed", raw.PMID, raw.Bytes)
				if err != nil {
					counters.Failed.Add(1)
					continue
				}
				counters.Archived.Add(1)
				if _, err := i.pub.PublishRaw(ctx, "pubmed", raw.PMID, key); err != nil {
					counters.Failed.Add(1)
					continue
				}
				counters.Published.Add(1)
			}
		}()
	}
	wg.Wait()

	newHWM := maxEDAT
	if newHWM == "" {
		newHWM = hwm
	}
	_ = i.wm.Set(ctx, "pubmed", newHWM, "idle", "")

	return ingestcommon.RunResult{
		DocsFetched:   counters.Fetched.Load(),
		DocsArchived:  counters.Archived.Load(),
		DocsPublished: counters.Published.Load(),
		HighWatermark: newHWM,
	}, nil
}

// ---- E-utilities calls ----

const eutilsBase = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils"

type esearchResult struct {
	XMLName xml.Name `xml:"eSearchResult"`
	IDList  struct {
		ID []string `xml:"Id"`
	} `xml:"IdList"`
	WebEnv     string `xml:"WebEnv"`
	QueryKey   string `xml:"QueryKey"`
	Count      int    `xml:"Count"`
	RetMax     int    `xml:"RetMax"`
}

func (i *Ingester) esearch(ctx context.Context, sinceDate string) ([]string, error) {
	q := url.Values{}
	q.Set("db", "pubmed")
	q.Set("term", fmt.Sprintf("%s[EDAT]:3000[EDAT]", sinceDate))
	q.Set("retmax", "10000")
	q.Set("usehistory", "y")
	q.Set("tool", i.cfg.Tool)
	q.Set("email", i.cfg.Email)
	if i.cfg.APIKey != "" {
		q.Set("api_key", i.cfg.APIKey)
	}
	url := fmt.Sprintf("%s/esearch.fcgi?%s", eutilsBase, q.Encode())

	body, err := i.fetcher.Get(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	var r esearchResult
	if err := xml.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("parse esearch: %w", err)
	}
	return r.IDList.ID, nil
}

type efetchRaw struct {
	PMID  string
	Bytes []byte
	EDAT  string
}

func (i *Ingester) efetch(ctx context.Context, pmids []string) ([]efetchRaw, string, error) {
	q := url.Values{}
	q.Set("db", "pubmed")
	q.Set("id", strings.Join(pmids, ","))
	q.Set("retmode", "xml")
	q.Set("tool", i.cfg.Tool)
	q.Set("email", i.cfg.Email)
	if i.cfg.APIKey != "" {
		q.Set("api_key", i.cfg.APIKey)
	}
	url := fmt.Sprintf("%s/efetch.fcgi?%s", eutilsBase, q.Encode())

	body, err := i.fetcher.Get(ctx, url, nil)
	if err != nil {
		return nil, "", err
	}

	// Parse just enough to extract PMIDs and EDATs; processor does full
	// parse. We split the multi-record XML by PubmedArticle and store
	// each article's bytes separately so re-processing one record
	// doesn't require re-fetching the whole batch.
	records, maxEDAT := splitArticles(body)
	return records, maxEDAT, nil
}

// splitArticles splits a PubmedArticleSet document into per-article
// byte slices and returns the max EDAT seen.
func splitArticles(body []byte) ([]efetchRaw, string) {
	type article struct {
		PMID    string `xml:"MedlineCitation>PMID"`
		History struct {
			PubMedPubDate []struct {
				Status string `xml:"PubStatus,attr"`
				Year   string `xml:"Year"`
				Month  string `xml:"Month"`
				Day    string `xml:"Day"`
			} `xml:"PubMedPubDate"`
		} `xml:"PubmedData>History"`
	}

	var maxEDAT string
	var out []efetchRaw

	dec := xml.NewDecoder(strings.NewReader(string(body)))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "PubmedArticle" {
			continue
		}
		var a article
		// Capture raw bytes by re-encoding the element.
		var raw strings.Builder
		raw.WriteString("<PubmedArticle>")
		if err := dec.DecodeElement(&a, &se); err != nil {
			continue
		}
		raw.WriteString("</PubmedArticle>")

		edat := ""
		for _, p := range a.History.PubMedPubDate {
			if p.Status == "entrez" || p.Status == "pubmed" {
				edat = fmt.Sprintf("%s/%s/%s", p.Year, padDate(p.Month), padDate(p.Day))
				break
			}
		}
		if edat > maxEDAT {
			maxEDAT = edat
		}
		out = append(out, efetchRaw{
			PMID:  a.PMID,
			Bytes: []byte(raw.String()),
			EDAT:  edat,
		})
	}
	return out, maxEDAT
}

func padDate(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

func chunk(s []string, n int) [][]string {
	var out [][]string
	for i := 0; i < len(s); i += n {
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}
