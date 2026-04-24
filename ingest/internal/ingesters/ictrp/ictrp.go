// Package ictrp ingests WHO ICTRP weekly bulk XML (spec §5.1.4).
//
// Bulk weekly XML zip from trialsearch.who.int. Diff against previous
// snapshot SHA256 stored in the ingestion_state watermark; on hash
// change, stream the zip and emit one event per trial.
package ictrp

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

// bulkURL is the public WHO ICTRP TrialResults zip endpoint.
// Override via env ICTRP_BULK_URL if WHO moves it.
var bulkURL = "https://www.who.int/trialsearch/Files/TrialResults.zip"

type Ingester struct {
	logger   *slog.Logger
	wm       *watermark.Store
	archiver *r2.Archiver
	pub      *pubsubpub.Publisher
	fetcher  *ingestcommon.Fetcher
}

func New(logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	if v := ingestcommon.GetEnv("ICTRP_BULK_URL", ""); v != "" {
		bulkURL = v
	}
	return &Ingester{
		logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(1, 2, "EvidenceLens-ICTRP/0.1 (mailto:contact@example.com)"),
	}
}

type ictrpTrial struct {
	XMLName       xml.Name `xml:"Trial"`
	TrialID       string   `xml:"TrialID"`
	Source        string   `xml:"Source_Register"`
	Status        string   `xml:"Recruitment_Status"`
	Phase         string   `xml:"Phase"`
	Title         string   `xml:"Public_title"`
	Conditions    string   `xml:"Condition"`
	Interventions string   `xml:"Intervention"`
	Countries     string   `xml:"Countries"`
	StartDate     string   `xml:"Date_enrollement"`
}

type ictrpRoot struct {
	XMLName xml.Name     `xml:"Trials_downloaded_from_ICTRP"`
	Trials  []ictrpTrial `xml:"Trial"`
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "ictrp"); err != nil {
		return ingestcommon.RunResult{}, err
	}
	prevHash, _ := i.wm.Get(ctx, "ictrp")
	i.logger.Info("ictrp run starting", "previous_hash", prevHash)

	body, err := i.fetcher.Get(ctx, bulkURL, nil)
	if err != nil {
		_ = i.wm.Set(ctx, "ictrp", prevHash, "failed", err.Error())
		return ingestcommon.RunResult{}, fmt.Errorf("download zip: %w", err)
	}

	hash := sha256.Sum256(body)
	hashHex := hex.EncodeToString(hash[:])
	if hashHex == prevHash {
		i.logger.Info("ictrp: no change since last snapshot", "hash", hashHex)
		_ = i.wm.Set(ctx, "ictrp", hashHex, "idle", "")
		return ingestcommon.RunResult{}, nil
	}

	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return ingestcommon.RunResult{}, fmt.Errorf("unzip: %w", err)
	}

	var counters ingestcommon.Counters
	for _, f := range zr.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".xml") {
			continue
		}
		if err := i.processFile(ctx, f, &counters); err != nil {
			i.logger.Warn("ictrp file failed", "name", f.Name, "err", err)
		}
		if ctx.Err() != nil {
			break
		}
	}

	_ = i.wm.Set(ctx, "ictrp", hashHex, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched:   counters.Fetched.Load(),
		DocsArchived:  counters.Archived.Load(),
		DocsPublished: counters.Published.Load(),
		HighWatermark: hashHex,
	}, nil
}

func (i *Ingester) processFile(ctx context.Context, f *zip.File, c *ingestcommon.Counters) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	var root ictrpRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("xml: %w", err)
	}
	for _, t := range root.Trials {
		c.Fetched.Add(1)
		id := strings.TrimSpace(t.TrialID)
		if id == "" {
			c.Failed.Add(1)
			continue
		}
		docID := "ictrp:" + id
		raw, _ := json.Marshal(t)
		key, err := i.archiver.Put(ctx, "ictrp", docID, raw)
		if err != nil {
			c.Failed.Add(1)
			continue
		}
		c.Archived.Add(1)
		if _, err := i.pub.PublishRaw(ctx, "ictrp", docID, key); err == nil {
			c.Published.Add(1)
		}
	}
	return nil
}
