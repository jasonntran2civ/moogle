package ingestcommon

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

// Runner orchestrates one ingester process: signal handling, graceful
// shutdown, structured logging, optional Cloud Run /run HTTP entrypoint.
//
// Pattern lifted from Moogle's spider main.go (signal handling) and
// extended with structured slog + run-loop counters.
type Runner struct {
	Source    string         // "pubmed" | "trials" | ...
	Logger    *slog.Logger
	Run       func(context.Context) (RunResult, error)
}

// RunResult records what happened in one /run invocation. Surfaces to
// logs and OTel metrics.
type RunResult struct {
	DocsFetched     int64
	DocsArchived    int64
	DocsPublished   int64
	HighWatermark   string
	DurationSeconds float64
	Error           error
}

// Counters provides atomic increment helpers usable from worker
// goroutines without a mutex.
type Counters struct {
	Fetched   atomic.Int64
	Archived  atomic.Int64
	Published atomic.Int64
	Failed    atomic.Int64
}

// Setup wires SIGINT / SIGTERM handling. Returns a context that is
// cancelled on signal.
func Setup(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		slog.Info("shutdown signal received")
		cancel()
	}()
	return ctx, cancel
}

// MustNewLogger returns a JSON slog Logger writing to stdout, with the
// service name baked in.
func MustNewLogger(service string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(h).With("service", service)
}

// RunOnce invokes Runner.Run with a deadline derived from
// CLOUD_RUN_TIMEOUT env (default 15min). Used by the Cloud Run /run
// HTTP handler.
func (r *Runner) RunOnce(ctx context.Context, deadline time.Duration) RunResult {
	if deadline > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, deadline)
		defer cancel()
	}
	start := time.Now()
	res, err := r.Run(ctx)
	res.DurationSeconds = time.Since(start).Seconds()
	res.Error = err
	if err != nil {
		r.Logger.Error("run failed",
			"err", err,
			"duration_s", res.DurationSeconds,
			"docs_fetched", res.DocsFetched,
		)
	} else {
		r.Logger.Info("run complete",
			"docs_fetched", res.DocsFetched,
			"docs_archived", res.DocsArchived,
			"docs_published", res.DocsPublished,
			"watermark", res.HighWatermark,
			"duration_s", res.DurationSeconds,
		)
	}
	return res
}
