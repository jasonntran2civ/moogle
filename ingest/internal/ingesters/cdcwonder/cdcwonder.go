// Package cdcwonder ingests CDC WONDER mortality / VAERS data
// (spec section 2 row 29). REST + XML POST query.
package cdcwonder

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

const apiURL = "https://wonder.cdc.gov/controller/datarequest/D76"  // current mortality, all races, all ages

type Ingester struct {
	logger *slog.Logger; wm *watermark.Store
	archiver *r2.Archiver; pub *pubsubpub.Publisher; fetcher *ingestcommon.Fetcher
}

func New(logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(1, 2, "EvidenceLens-CDC-WONDER/0.1 (mailto:contact@example.com)"),
	}
}

const queryXML = `<request-parameters>
  <parameter><name>accept_datause_restrictions</name><value>true</value></parameter>
  <parameter><name>B_1</name><value>D76.V1-level1</value></parameter>
  <parameter><name>O_age</name><value>D76.V5</value></parameter>
</request-parameters>`

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "cdc-wonder"); err != nil { return ingestcommon.RunResult{}, err }
	body, err := i.fetcher.Post(ctx, apiURL, strings.NewReader(queryXML),
		map[string]string{"Content-Type": "application/xml"})
	if err != nil { return ingestcommon.RunResult{}, fmt.Errorf("wonder query: %w", err) }
	stamp := time.Now().UTC().Format("2006-01-02")
	docID := "cdc-wonder:mortality:" + stamp
	raw, _ := json.Marshal(map[string]any{"snapshot": stamp, "raw_xml": string(body)})
	key, err := i.archiver.Put(ctx, "cdc-wonder", docID, raw)
	if err != nil { return ingestcommon.RunResult{}, err }
	if _, err := i.pub.PublishRaw(ctx, "cdc-wonder", docID, key); err != nil {
		return ingestcommon.RunResult{}, err
	}
	_ = i.wm.Set(ctx, "cdc-wonder", stamp, "idle", "")
	return ingestcommon.RunResult{DocsFetched: 1, DocsArchived: 1, DocsPublished: 1, HighWatermark: stamp}, nil
}
