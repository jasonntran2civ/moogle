// Package meili batches Document writes to Meilisearch.
//
// Spec section 5.4 batcher: 1000 docs OR 5s, whichever first. Lifts
// Moogle's size-threshold pattern and adds the time trigger.
package meili

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

type Config struct {
	URL               string
	APIKey            string
	IndexName         string
	BatchSize         int
	FlushAfterSeconds int
	Logger            *slog.Logger
}

type Batcher struct {
	cfg      Config
	client   meilisearch.ServiceManager
	in       chan json.RawMessage
	flushReq chan struct{}
	wg       sync.WaitGroup
}

func New(cfg Config) (*Batcher, error) {
	if cfg.BatchSize == 0 { cfg.BatchSize = 1000 }
	if cfg.FlushAfterSeconds == 0 { cfg.FlushAfterSeconds = 5 }
	c := meilisearch.New(cfg.URL, meilisearch.WithAPIKey(cfg.APIKey))
	return &Batcher{
		cfg: cfg, client: c,
		in: make(chan json.RawMessage, cfg.BatchSize*2),
		flushReq: make(chan struct{}, 1),
	}, nil
}

// Flush requests a manual flush (wired to SIGUSR1 in indexer/cmd).
func (b *Batcher) Flush() {
	select { case b.flushReq <- struct{}{}: default: }
}

func (b *Batcher) Submit(doc json.RawMessage) {
	select {
	case b.in <- doc:
	default:
		// Channel full: drop with log. Operator should scale up indexer.
		b.cfg.Logger.Warn("meili batcher dropping; channel full")
	}
}

func (b *Batcher) Run(ctx context.Context) {
	b.wg.Add(1)
	defer b.wg.Done()

	tick := time.NewTicker(time.Duration(b.cfg.FlushAfterSeconds) * time.Second)
	defer tick.Stop()

	batch := make([]json.RawMessage, 0, b.cfg.BatchSize)
	flush := func() {
		if len(batch) == 0 { return }
		b.cfg.Logger.Info("flush", "n", len(batch))
		idx := b.client.Index(b.cfg.IndexName)
		// Index expects [].any; convert. addOrReplace: same as upsert.
		toSend := make([]any, len(batch))
		for i, m := range batch {
			var obj any
			_ = json.Unmarshal(m, &obj)
			toSend[i] = obj
		}
		if _, err := idx.AddDocuments(toSend, "id"); err != nil {
			b.cfg.Logger.Error("meili add docs", "n", len(batch), "err", err)
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
		case doc := <-b.in:
			batch = append(batch, doc)
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
