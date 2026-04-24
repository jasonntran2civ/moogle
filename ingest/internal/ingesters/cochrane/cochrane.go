// Package cochrane ingests Cochrane systematic reviews via RSS + DOI
// resolution (spec §5.1.11).
//
// Free for academic only — never serve full content, only metadata + deep
// links. See docs/sources/cochrane.md for the policy.
package cochrane

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"strings"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

const cochraneRSS = "https://www.cochranelibrary.com/cdsr/rss"
const crossrefAPI = "https://api.crossref.org/works/"

type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

type Ingester struct {
	logger   *slog.Logger
	wm       *watermark.Store
	archiver *r2.Archiver
	pub      *pubsubpub.Publisher
	fetcher  *ingestcommon.Fetcher
}

func New(logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(1, 2, "EvidenceLens-Cochrane/0.1 (mailto:contact@example.com)"),
	}
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "cochrane"); err != nil {
		return ingestcommon.RunResult{}, err
	}
	hwm, _ := i.wm.Get(ctx, "cochrane")

	body, err := i.fetcher.Get(ctx, cochraneRSS, nil)
	if err != nil {
		return ingestcommon.RunResult{}, fmt.Errorf("rss: %w", err)
	}
	var feed rssRoot
	if err := xml.Unmarshal(body, &feed); err != nil {
		return ingestcommon.RunResult{}, fmt.Errorf("parse rss: %w", err)
	}

	var counters ingestcommon.Counters
	var maxPubDate string
	for _, it := range feed.Channel.Items {
		counters.Fetched.Add(1)
		if it.PubDate > maxPubDate {
			maxPubDate = it.PubDate
		}
		if hwm != "" && it.PubDate <= hwm {
			continue
		}
		doi := extractDOIFromCochraneURL(it.Link)
		if doi == "" {
			counters.Failed.Add(1)
			continue
		}
		enriched := map[string]any{
			"title":       it.Title,
			"link":        it.Link,
			"doi":         doi,
			"pub_date":    it.PubDate,
			"description": it.Description,
		}
		if cr, err := i.fetcher.Get(ctx, crossrefAPI+doi, nil); err == nil {
			var crossref map[string]any
			if err := json.Unmarshal(cr, &crossref); err == nil {
				enriched["crossref"] = crossref
			}
		}
		raw, _ := json.Marshal(enriched)
		docID := "cochrane:" + doi
		key, err := i.archiver.Put(ctx, "cochrane", docID, raw)
		if err != nil {
			counters.Failed.Add(1)
			continue
		}
		counters.Archived.Add(1)
		if _, err := i.pub.PublishRaw(ctx, "cochrane", docID, key); err == nil {
			counters.Published.Add(1)
		}
	}
	_ = i.wm.Set(ctx, "cochrane", maxPubDate, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched:   counters.Fetched.Load(),
		DocsArchived:  counters.Archived.Load(),
		DocsPublished: counters.Published.Load(),
		HighWatermark: maxPubDate,
	}, nil
}

// extractDOIFromCochraneURL: typical link is
// https://www.cochranelibrary.com/cdsr/doi/10.1002/14651858.CD000234.pub5/full
// → 10.1002/14651858.CD000234.pub5
func extractDOIFromCochraneURL(u string) string {
	const marker = "/doi/"
	idx := strings.Index(u, marker)
	if idx < 0 {
		return ""
	}
	rest := u[idx+len(marker):]
	rest = strings.TrimSuffix(rest, "/full")
	rest = strings.TrimRight(rest, "/")
	if end := strings.IndexAny(rest, "?#"); end > 0 {
		rest = rest[:end]
	}
	return rest
}
