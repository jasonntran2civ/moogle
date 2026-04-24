// Package openalex ingests OpenAlex works + emits citation edges
// (spec §5.1.6).
//
// Two paths:
//   - Bulk snapshot via S3 stream-process (no disk staging) for first run.
//   - Per-doc REST updates via api.openalex.org/works for daily delta.
//
// Citation edges (citing_doc_id, cited_doc_id) emit to a separate
// Pub/Sub topic citation-edges for the indexer's Neo4j batcher.
package openalex

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

const (
	openalexBucket = "openalex"
	openalexPrefix = "data/works/"
)

type Config struct {
	MaxPerRun int
	UseBulk   bool // true = S3 bulk; false = REST delta
}

type Ingester struct {
	cfg          Config
	logger       *slog.Logger
	wm           *watermark.Store
	archiver     *r2.Archiver
	pub          *pubsubpub.Publisher
	citationsPub *pubsubpub.Publisher
	fetcher      *ingestcommon.Fetcher
	s3           *s3.Client
}

func New(cfg Config, logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub, citationsPub *pubsubpub.Publisher) *Ingester {
	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("us-east-1"))
	if err != nil {
		logger.Warn("openalex: aws config", "err", err)
	}
	s3c := s3.NewFromConfig(awsCfg)
	return &Ingester{
		cfg: cfg, logger: logger, wm: wm, archiver: arch, pub: pub, citationsPub: citationsPub,
		fetcher: ingestcommon.NewFetcher(10, 20, "EvidenceLens-OpenAlex/0.1 (mailto:contact@example.com)"),
		s3:      s3c,
	}
}

// openalexWork — subset of the OpenAlex schema we map into the canonical
// Document. Full schema is huge; we keep only what the processor needs.
type openalexWork struct {
	ID              string   `json:"id"`
	DOI             string   `json:"doi"`
	Title           string   `json:"display_name"`
	PublicationYear int      `json:"publication_year"`
	CitedByCount    int64    `json:"cited_by_count"`
	ReferencedWorks []string `json:"referenced_works"`
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "openalex"); err != nil {
		return ingestcommon.RunResult{}, err
	}
	if i.cfg.UseBulk {
		return i.runBulk(ctx)
	}
	return i.runREST(ctx)
}

// runBulk streams .gz JSONL files directly from the public openalex S3
// bucket WITHOUT disk staging (per spec §19.4 risk mitigation).
func (i *Ingester) runBulk(ctx context.Context) (ingestcommon.RunResult, error) {
	hwm, _ := i.wm.Get(ctx, "openalex")
	var counters ingestcommon.Counters

	pager := s3.NewListObjectsV2Paginator(i.s3, &s3.ListObjectsV2Input{
		Bucket: aws.String(openalexBucket),
		Prefix: aws.String(openalexPrefix),
	})
	for pager.HasMorePages() && int(counters.Fetched.Load()) < i.cfg.MaxPerRun {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return ingestcommon.RunResult{}, fmt.Errorf("list bulk: %w", err)
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			if hwm != "" && key <= hwm {
				continue
			}
			if err := i.streamPart(ctx, key, &counters); err != nil {
				i.logger.Warn("part failed", "key", key, "err", err)
			}
			hwm = key
			if int(counters.Fetched.Load()) >= i.cfg.MaxPerRun {
				break
			}
		}
	}
	_ = i.wm.Set(ctx, "openalex", hwm, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched:   counters.Fetched.Load(),
		DocsArchived:  counters.Archived.Load(),
		DocsPublished: counters.Published.Load(),
		HighWatermark: hwm,
	}, nil
}

func (i *Ingester) streamPart(ctx context.Context, key string, c *ingestcommon.Counters) error {
	resp, err := i.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(openalexBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gz.Close()

	scanner := bufio.NewScanner(gz)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		c.Fetched.Add(1)
		var w openalexWork
		line := scanner.Bytes()
		if err := json.Unmarshal(line, &w); err != nil {
			c.Failed.Add(1)
			continue
		}
		i.publishWork(ctx, &w, line, c)
		if int(c.Fetched.Load()) >= i.cfg.MaxPerRun {
			break
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return scanner.Err()
}

// runREST fetches recently-updated works via the REST API.
func (i *Ingester) runREST(ctx context.Context) (ingestcommon.RunResult, error) {
	hwm, _ := i.wm.Get(ctx, "openalex")
	if hwm == "" {
		hwm = time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	}
	var counters ingestcommon.Counters

	cursor := "*"
	for int(counters.Fetched.Load()) < i.cfg.MaxPerRun {
		q := url.Values{}
		q.Set("filter", "from_updated_date:"+hwm)
		q.Set("per-page", "200")
		q.Set("cursor", cursor)
		body, err := i.fetcher.Get(ctx, "https://api.openalex.org/works?"+q.Encode(), nil)
		if err != nil {
			break
		}
		var resp struct {
			Results []openalexWork `json:"results"`
			Meta    struct {
				NextCursor string `json:"next_cursor"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(body, &resp); err != nil || len(resp.Results) == 0 {
			break
		}
		for _, w := range resp.Results {
			counters.Fetched.Add(1)
			raw, _ := json.Marshal(w)
			i.publishWork(ctx, &w, raw, &counters)
		}
		if resp.Meta.NextCursor == "" {
			break
		}
		cursor = resp.Meta.NextCursor
	}

	newHWM := time.Now().Format("2006-01-02")
	_ = i.wm.Set(ctx, "openalex", newHWM, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched:   counters.Fetched.Load(),
		DocsArchived:  counters.Archived.Load(),
		DocsPublished: counters.Published.Load(),
		HighWatermark: newHWM,
	}, nil
}

func (i *Ingester) publishWork(ctx context.Context, w *openalexWork, raw []byte, c *ingestcommon.Counters) {
	id := openalexShortID(w.ID)
	if id == "" {
		c.Failed.Add(1)
		return
	}
	docID := "openalex:" + id

	key, err := i.archiver.Put(ctx, "openalex", docID, raw)
	if err != nil {
		c.Failed.Add(1)
		return
	}
	c.Archived.Add(1)

	if _, err := i.pub.PublishRaw(ctx, "openalex", docID, key); err == nil {
		c.Published.Add(1)
	}

	if i.citationsPub != nil {
		for _, cited := range w.ReferencedWorks {
			cidShort := openalexShortID(cited)
			if cidShort == "" {
				continue
			}
			edgeKey := fmt.Sprintf("edge:%s:%s", id, cidShort)
			_, _ = i.citationsPub.PublishRaw(ctx, "openalex", edgeKey, "")
		}
	}
}

// openalexShortID strips the URL prefix to yield "W12345678".
func openalexShortID(s string) string {
	if s == "" {
		return ""
	}
	idx := strings.LastIndex(s, "/")
	if idx < 0 {
		return s
	}
	return s[idx+1:]
}
