// Package openpayments runs the dual-mode CMS Open Payments service
// (spec §5.1.10): bulk CSV ingest + /lookup HTTP endpoint for the
// processor's author-payment-joiner.
//
// Conservative bias: false positives are worse than false negatives.
// Default fuzzy threshold 0.90 (configurable). See
// docs/sources/open-payments.md for matching policy.
package openpayments

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DatabaseURL        string
	MinFuzzyConfidence float64 // 0..1
	BulkURL            string  // optional; can also pass via /run?url=...
}

type Server struct {
	cfg      Config
	logger   *slog.Logger
	pool     *pgxpool.Pool
	archiver *r2.Archiver
	fetcher  *ingestcommon.Fetcher
}

func NewServer(cfg Config, logger *slog.Logger, arch *r2.Archiver) *Server {
	return &Server{
		cfg: cfg, logger: logger, archiver: arch,
		fetcher: ingestcommon.NewFetcher(0, 0, "EvidenceLens-OpenPayments/0.1"),
	}
}

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

// handleRun streams the upstream zip, extracts the General-Payment CSV,
// and COPY-loads into open_payments.
func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	bulkURL := s.cfg.BulkURL
	if v := r.URL.Query().Get("url"); v != "" {
		bulkURL = v
	}
	if bulkURL == "" {
		http.Error(w, `{"error":"bulk url required (env BULK_URL or ?url=...)"}`, http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	s.logger.Info("open-payments bulk ingest", "url", bulkURL)

	body, err := s.fetcher.Get(ctx, bulkURL, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadGateway)
		return
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"unzip: %s"}`, err.Error()), http.StatusUnprocessableEntity)
		return
	}

	var totalRows int64
	for _, f := range zr.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			continue
		}
		s.logger.Info("loading csv", "file", f.Name, "uncompressed_size", f.UncompressedSize64)
		n, err := s.loadCSV(ctx, f)
		if err != nil {
			s.logger.Warn("csv load failed", "file", f.Name, "err", err)
			continue
		}
		totalRows += n
	}

	stamp := time.Now().UTC().Format("2006-01-02")
	if _, err := s.archiver.Put(ctx, "open-payments", "bulk-"+stamp, body); err != nil {
		s.logger.Warn("r2 archive bulk", "err", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":        "ok",
		"rows_loaded":   totalRows,
		"snapshot_date": stamp,
	})
}

// loadCSV streams one CSV file from the zip and COPY-loads matching
// columns into the open_payments table.
//
// Schema mapping (CMS General-Payment column names):
//   Record_ID, Physician_NPI, Physician_First_Name, Physician_Last_Name,
//   Recipient_State,
//   Submitting_Applicable_Manufacturer_or_Applicable_GPO_Name,
//   Total_Amount_of_Payment_USDollars,
//   Nature_of_Payment_or_Transfer_of_Value, Program_Year
func (s *Server) loadCSV(ctx context.Context, f *zip.File) (int64, error) {
	rc, err := f.Open()
	if err != nil {
		return 0, err
	}
	defer rc.Close()

	cr := csv.NewReader(rc)
	cr.FieldsPerRecord = -1
	header, err := cr.Read()
	if err != nil {
		return 0, fmt.Errorf("header: %w", err)
	}
	col := indexCols(header)
	if _, ok := col["Record_ID"]; !ok {
		return 0, fmt.Errorf("missing Record_ID column")
	}

	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Release()

	rows := make(chan []any, 1024)
	errCh := make(chan error, 1)
	go func() {
		defer close(rows)
		for {
			rec, err := cr.Read()
			if err == io.EOF {
				return
			}
			if err != nil {
				errCh <- err
				return
			}
			if ctx.Err() != nil {
				return
			}
			recID := safe(rec, col, "Record_ID")
			if recID == "" {
				continue
			}
			amt, _ := strconv.ParseFloat(safe(rec, col, "Total_Amount_of_Payment_USDollars"), 64)
			year, _ := strconv.Atoi(safe(rec, col, "Program_Year"))
			rawJSON, _ := json.Marshal(rowToMap(header, rec))
			physName := strings.TrimSpace(
				safe(rec, col, "Physician_First_Name") + " " + safe(rec, col, "Physician_Last_Name"),
			)
			rows <- []any{
				recID,
				nullable(safe(rec, col, "Physician_NPI")),
				physName,
				nullable(safe(rec, col, "Recipient_State")),
				safe(rec, col, "Submitting_Applicable_Manufacturer_or_Applicable_GPO_Name"),
				year,
				amt,
				nullable(safe(rec, col, "Nature_of_Payment_or_Transfer_of_Value")),
				rawJSON,
			}
		}
	}()

	n, err := conn.Conn().CopyFrom(ctx,
		pgx.Identifier{"open_payments"},
		[]string{
			"record_id", "physician_npi", "physician_name", "physician_state",
			"sponsor_name", "payment_year", "amount_usd", "payment_type", "raw_jsonb",
		},
		pgx.CopyFromChannel(rows),
	)
	if err != nil {
		return n, err
	}
	select {
	case e := <-errCh:
		return n, e
	default:
		return n, nil
	}
}

func indexCols(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		m[h] = i
	}
	return m
}

func safe(rec []string, col map[string]int, name string) string {
	if i, ok := col[name]; ok && i < len(rec) {
		return strings.TrimSpace(rec[i])
	}
	return ""
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func rowToMap(header, rec []string) map[string]string {
	m := make(map[string]string, len(header))
	for i, h := range header {
		if i < len(rec) {
			m[h] = rec[i]
		}
	}
	return m
}

// ---- /lookup endpoint ----

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

	sql := `
		SELECT physician_name, sponsor_name, payment_year, amount_usd,
		       coalesce(payment_type,'other'), record_id,
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
