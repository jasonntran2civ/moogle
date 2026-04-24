// Package qdrant batches embedding upserts into Qdrant.
//
// Spec §5.4: batch 100 vectors OR 5s. Collection evidence_v1, HNSW,
// 1024-d. Idempotent by `id`.
package qdrant

import (
	"context"
	"encoding/json"
	"log/slog"
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
// fields for Qdrant payload.
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
	cfg Config
	in  chan docPayload
	wg  sync.WaitGroup
}

func New(cfg Config) (*Batcher, error) {
	if cfg.BatchSize == 0 { cfg.BatchSize = 100 }
	if cfg.FlushAfterSeconds == 0 { cfg.FlushAfterSeconds = 5 }
	return &Batcher{cfg: cfg, in: make(chan docPayload, cfg.BatchSize*2)}, nil
}

func (b *Batcher) Submit(raw json.RawMessage) {
	var d docPayload
	if err := json.Unmarshal(raw, &d); err != nil {
		b.cfg.Logger.Warn("qdrant submit unmarshal", "err", err)
		return
	}
	if len(d.Embedding) == 0 {
		// No embedding -> nothing to upsert in Qdrant.
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
		if len(batch) == 0 { return }
		b.cfg.Logger.Info("flush", "n", len(batch))
		// TODO real Qdrant client upsert: github.com/qdrant/go-client.
		// For now, log only. Replace this block with PointsClient.Upsert.
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-tick.C:
			flush()
		case d := <-b.in:
			batch = append(batch, d)
			if len(batch) >= b.cfg.BatchSize {
				flush()
			}
		}
	}
}

func (b *Batcher) Close() {
	close(b.in)
	b.wg.Wait()
}
