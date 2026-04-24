// Package openpayments runs the dual-mode CMS Open Payments service
// (spec §5.1.10): bulk CSV ingest + /lookup HTTP endpoint for the
// processor's author-payment-joiner.
//
// Conservative bias: false positives are worse than false negatives.
// Default fuzzy threshold 0.90 (configurable). See
// docs/sources/open-payments.md for matching policy.
package openpayments

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DatabaseURL        string
	MinFuzzyConfidence float64 // 0..1
}

type Server struct {
	cfg      Config
	logger   *slog.Logger
	pool     *pgxpool.Pool
	archiver *r2.Archiver
}

func NewServer(cfg Config, logger *slog.Logger, arch *r2.Archiver) *Server {
	return &Server{cfg: cfg, logger: logger, archiver: arch}
}

// ListenAndServe wires the /run (bulk ingest), /lookup (joiner query),
// /healthz routes and blocks until ctx cancels.
func (s *Server) ListenAndServe(ctx context.Context) error {
	pool, err := pgxpool.New(ctx, s.cfg.DatabaseURL)
	if err != nil {
		return err
	}
	s.pool = pool
	defer pool.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/run", s.handleRun)
	mux.HandleFunc("/lookup", s.handleLookup)

	srv := &http.Server{
		Addr:              ":" + ingestcommon.GetEnv("PORT", "8080"),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() { <-ctx.Done(); _ = srv.Shutdown(context.Background()) }()
	s.logger.Info("open-payments listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// handleRun triggers the annual bulk CSV refresh. TODO: implement
// download + COPY ... FROM stdin into open_payments.
func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("open-payments /run: bulk CSV refresh TODO")
	_, _ = w.Write([]byte(`{"status":"stub"}`))
}

// handleLookup serves the processor's joiner. Query params:
//   name=Smith,John
//   state=CA          (optional)
//   year=2023         (optional)
type lookupResponse struct {
	Author     string         `json:"author"`
	Payments   []paymentMatch `json:"payments"`
	Confidence float64        `json:"confidence"`
}

type paymentMatch struct {
	SponsorName    string  `json:"sponsor_name"`
	Year           int     `json:"year"`
	AmountUSD      float64 `json:"amount_usd"`
	PaymentType    string  `json:"payment_type"`
	SourceRecordID string  `json:"source_record_id"`
}

func (s *Server) handleLookup(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}
	state := r.URL.Query().Get("state")
	year := r.URL.Query().Get("year")

	// Fuzzy lookup via pg_trgm similarity over physician_name.
	// Conservative threshold: only return matches with similarity >= MinFuzzyConfidence.
	sql := `
		SELECT physician_name, sponsor_name, payment_year, amount_usd, payment_type, record_id,
		       similarity(physician_name, $1) AS sim
		FROM open_payments
		WHERE physician_name % $1
		  AND ($2 = '' OR physician_state = $2)
		  AND ($3 = '' OR payment_year = $3::int)
		  AND similarity(physician_name, $1) >= $4
		ORDER BY sim DESC, amount_usd DESC
		LIMIT 100`
	rows, err := s.pool.Query(r.Context(), sql, name, state, year, s.cfg.MinFuzzyConfidence)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var resp lookupResponse
	resp.Author = name
	maxSim := 0.0
	for rows.Next() {
		var pm paymentMatch
		var matchedName string
		var sim float64
		if err := rows.Scan(&matchedName, &pm.SponsorName, &pm.Year, &pm.AmountUSD, &pm.PaymentType, &pm.SourceRecordID, &sim); err != nil {
			continue
		}
		if sim > maxSim {
			maxSim = sim
		}
		resp.Payments = append(resp.Payments, pm)
	}
	resp.Confidence = maxSim

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
