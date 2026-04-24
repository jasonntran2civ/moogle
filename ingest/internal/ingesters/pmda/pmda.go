// Package pmda ingests Japan PMDA approvals + alerts (spec section 2
// row 20). Public RSS / XML.
package pmda

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"log/slog"
	"time"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

const feedURL = "https://www.pmda.go.jp/english/safety/info-services/drugs/0001.xml"

type Ingester struct {
	logger *slog.Logger; wm *watermark.Store
	archiver *r2.Archiver; pub *pubsubpub.Publisher; fetcher *ingestcommon.Fetcher
}

func New(logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(1, 2, "EvidenceLens-PMDA/0.1 (mailto:contact@example.com)"),
	}
}

type pmdaItem struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
	GUID  string `xml:"guid"`
	Date  string `xml:"pubDate"`
}

type pmdaFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct{ Items []pmdaItem `xml:"item"` } `xml:"channel"`
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "pmda"); err != nil { return ingestcommon.RunResult{}, err }
	body, err := i.fetcher.Get(ctx, feedURL, nil)
	if err != nil { return ingestcommon.RunResult{}, err }
	var feed pmdaFeed
	if err := xml.Unmarshal(body, &feed); err != nil { return ingestcommon.RunResult{}, err }
	var counters ingestcommon.Counters
	for _, it := range feed.Channel.Items {
		counters.Fetched.Add(1)
		id := it.GUID
		if id == "" { id = it.Link }
		docID := "pmda:" + sha1Hex(id)
		raw, _ := json.Marshal(it)
		key, err := i.archiver.Put(ctx, "pmda", docID, raw)
		if err != nil { counters.Failed.Add(1); continue }
		counters.Archived.Add(1)
		if _, err := i.pub.PublishRaw(ctx, "pmda", docID, key); err == nil { counters.Published.Add(1) }
	}
	stamp := time.Now().UTC().Format("2006-01-02")
	_ = i.wm.Set(ctx, "pmda", stamp, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched: counters.Fetched.Load(), DocsArchived: counters.Archived.Load(),
		DocsPublished: counters.Published.Load(), HighWatermark: stamp,
	}, nil
}

func sha1Hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
