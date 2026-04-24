// Package qdrant batches embedding upserts into Qdrant.
//
// Spec §5.4: batch 100 vectors OR 5s. Collection evidence_v1, HNSW,
// 1024-d. Idempotent by `id`.
package qdrant

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Config struct {
	URL               string
	APIKey            string
	Collection        string
	BatchSize         int
	FlushAfterSeconds int
	Logger            *slog.Logger
}

// docPayload models the bits of Document we need: id + embedding + facet
// fields used for Qdrant payload filtering.
type docPayload struct {
	ID            string    `json:"id"`
	Embedding     []float32 `json:"embedding"`
	Source        string    `json:"source"`
	StudyType     string    `json:"study_type"`
	PublishedYear int       `json:"published_year"`
	HasCOI        bool      `json:"has_coi_authors"`
	License       string    `json:"license"`
}

type Batcher struct {
	cfg      Config
	in       chan docPayload
	flushReq chan struct{}
	wg       sync.WaitGroup
	client   *http.Client
}

func New(cfg Config) (*Batcher, error) {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushAfterSeconds == 0 {
		cfg.FlushAfterSeconds = 5
	}
	return &Batcher{
		cfg:      cfg,
		in:       make(chan docPayload, cfg.BatchSize*2),
		flushReq: make(chan struct{}, 1),
		client:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Flush requests a manual flush (wired to SIGUSR1).
func (b *Batcher) Flush() {
	select { case b.flushReq <- struct{}{}: default: }
}

func (b *Batcher) Submit(raw json.RawMessage) {
	var d docPayload
	if err := json.Unmarshal(raw, &d); err != nil {
		b.cfg.Logger.Warn("qdrant submit unmarshal", "err", err)
		return
	}
	if len(d.Embedding) == 0 {
		return
	}
	select {
	case b.in <- d:
	default:
		b.cfg.Logger.Warn("qdrant batcher dropping; channel full")
	}
}

func (b *Batcher) Run(ctx context.Context) {
	b.wg.Add(1)
	defer b.wg.Done()

	tick := time.NewTicker(time.Duration(b.cfg.FlushAfterSeconds) * time.Second)
	defer tick.Stop()

	batch := make([]docPayload, 0, b.cfg.BatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := b.upsert(ctx, batch); err != nil {
			b.cfg.Logger.Error("qdrant upsert", "n", len(batch), "err", err)
		} else {
			b.cfg.Logger.Info("flush", "n", len(batch))
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-tick.C:
			flush()
		case <-b.flushReq:
			flush()
		case d := <-b.in:
			batch = append(batch, d)
			if len(batch) >= b.cfg.BatchSize {
				flush()
			}
		}
	}
}

// qdrantPoint is the wire format for PUT /collections/{name}/points
type qdrantPoint struct {
	ID      uint64         `json:"id"`
	Vector  []float32      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

type qdrantPointsBody struct {
	Points []qdrantPoint `json:"points"`
}

// upsert calls Qdrant's HTTP API to upsert a batch of points. Using HTTP
// rather than the official gRPC client keeps the dependency surface
// small and lets us avoid a generated proto vendor tree.
func (b *Batcher) upsert(ctx context.Context, batch []docPayload) error {
	points := make([]qdrantPoint, 0, len(batch))
	for _, d := range batch {
		points = append(points, qdrantPoint{
			ID:     idToUint64(d.ID),
			Vector: d.Embedding,
			Payload: map[string]any{
				"doc_id":          d.ID,
				"source":          d.Source,
				"study_type":      d.StudyType,
				"published_year":  d.PublishedYear,
				"has_coi_authors": d.HasCOI,
				"license":         d.License,
			},
		})
	}
	body, _ := json.Marshal(qdrantPointsBody{Points: points})

	url := strings.TrimRight(b.cfg.URL, "/") + "/collections/" + b.cfg.Collection + "/points?wait=false"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if b.cfg.APIKey != "" {
		req.Header.Set("api-key", b.cfg.APIKey)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("qdrant http %d", resp.StatusCode)
	}
	return nil
}

func (b *Batcher) Close() {
	close(b.in)
	b.wg.Wait()
}

// idToUint64 maps a string document id to a 64-bit Qdrant point id by
// taking the first 8 bytes of SHA1(id). Qdrant accepts uint64 or UUID
// ids; uint64 is more space-efficient.
func idToUint64(s string) uint64 {
	h := sha1.Sum([]byte(s))
	return binary.BigEndian.Uint64(h[:8])
}
