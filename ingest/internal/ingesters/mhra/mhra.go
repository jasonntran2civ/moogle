// Package mhra ingests UK MHRA recall + safety alert pages by scraping
// products.mhra.gov.uk (spec section 2 row 17). UK Open Government licence.
package mhra

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/url"
	"regexp"
	"time"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

const seedURL = "https://www.gov.uk/drug-device-alerts.atom"

type Ingester struct {
	logger *slog.Logger; wm *watermark.Store
	archiver *r2.Archiver; pub *pubsubpub.Publisher; fetcher *ingestcommon.Fetcher
}

func New(logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(1, 2, "EvidenceLens-MHRA/0.1 (mailto:contact@example.com)"),
	}
}

var entryRE = regexp.MustCompile(`(?s)<entry>(.*?)</entry>`)
var titleRE = regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
var linkRE  = regexp.MustCompile(`<link[^>]*href="([^"]+)"`)
var idRE    = regexp.MustCompile(`<id[^>]*>([^<]+)</id>`)

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "mhra"); err != nil { return ingestcommon.RunResult{}, err }
	body, err := i.fetcher.Get(ctx, seedURL, nil)
	if err != nil { return ingestcommon.RunResult{}, err }
	var counters ingestcommon.Counters
	for _, m := range entryRE.FindAllSubmatch(body, -1) {
		counters.Fetched.Add(1)
		entry := m[1]
		title := first(titleRE.FindSubmatch(entry))
		link  := first(linkRE.FindSubmatch(entry))
		id    := first(idRE.FindSubmatch(entry))
		if id == "" { counters.Failed.Add(1); continue }
		docID := "mhra:" + sha1Hex(id)
		_ = url.Parse  // imports placeholder
		raw, _ := json.Marshal(map[string]string{
			"id": id, "title": title, "link": link,
		})
		key, err := i.archiver.Put(ctx, "mhra", docID, raw)
		if err != nil { counters.Failed.Add(1); continue }
		counters.Archived.Add(1)
		if _, err := i.pub.PublishRaw(ctx, "mhra", docID, key); err == nil { counters.Published.Add(1) }
	}
	stamp := time.Now().UTC().Format("2006-01-02")
	_ = i.wm.Set(ctx, "mhra", stamp, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched: counters.Fetched.Load(), DocsArchived: counters.Archived.Load(),
		DocsPublished: counters.Published.Load(), HighWatermark: stamp,
	}, nil
}

func first(m [][]byte) string {
	if len(m) >= 2 { return string(m[1]) }
	return ""
}
func sha1Hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
