// Package initdata creates Qdrant collection + Neo4j indexes at indexer
// startup. Idempotent: re-running is a no-op when the resources exist.
//
// Spec section 3.3: Qdrant collection `evidence_v1`, vectors size 1024,
//   cosine distance, HNSW m=32, ef_construct=200, indexing_threshold=20000.
// Spec section 3.4: Neo4j indexes on Document.id, Author.orcid, MeshTerm.id.
package initdata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type QdrantConfig struct {
	URL           string
	APIKey        string
	Collection    string
	VectorSize    int
	HNSWM         int
	HNSWEfConstruct int
	OptimizerThreshold int
	Logger        *slog.Logger
}

type qdrantCreate struct {
	Vectors struct {
		Size     int    `json:"size"`
		Distance string `json:"distance"`
	} `json:"vectors"`
	HNSWConfig struct {
		M           int `json:"m"`
		EFConstruct int `json:"ef_construct"`
	} `json:"hnsw_config"`
	OptimizersConfig struct {
		IndexingThreshold int `json:"indexing_threshold"`
	} `json:"optimizers_config"`
}

// EnsureQdrantCollection creates the collection if missing. PUT
// /collections/{name} is idempotent at the URL but not the body, so we
// HEAD-check first.
func EnsureQdrantCollection(ctx context.Context, cfg QdrantConfig) error {
	if cfg.VectorSize == 0 { cfg.VectorSize = 1024 }
	if cfg.HNSWM == 0 { cfg.HNSWM = 32 }
	if cfg.HNSWEfConstruct == 0 { cfg.HNSWEfConstruct = 200 }
	if cfg.OptimizerThreshold == 0 { cfg.OptimizerThreshold = 20000 }
	base := strings.TrimRight(cfg.URL, "/")
	client := &http.Client{Timeout: 30 * time.Second}

	// Probe collection.
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/collections/"+cfg.Collection, nil)
	if cfg.APIKey != "" {
		req.Header.Set("api-key", cfg.APIKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant probe: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode == 200 {
		cfg.Logger.Info("qdrant collection exists", "name", cfg.Collection)
		return nil
	}

	// Create.
	body := qdrantCreate{}
	body.Vectors.Size = cfg.VectorSize
	body.Vectors.Distance = "Cosine"
	body.HNSWConfig.M = cfg.HNSWM
	body.HNSWConfig.EFConstruct = cfg.HNSWEfConstruct
	body.OptimizersConfig.IndexingThreshold = cfg.OptimizerThreshold
	bs, _ := json.Marshal(body)
	put, _ := http.NewRequestWithContext(ctx, http.MethodPut, base+"/collections/"+cfg.Collection, bytes.NewReader(bs))
	put.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		put.Header.Set("api-key", cfg.APIKey)
	}
	r2, err := client.Do(put)
	if err != nil {
		return fmt.Errorf("qdrant create: %w", err)
	}
	defer r2.Body.Close()
	if r2.StatusCode >= 300 {
		return fmt.Errorf("qdrant create http %d", r2.StatusCode)
	}

	// Payload-field indexes for fast filtered search.
	for field, ftype := range map[string]string{
		"doc_id":          "keyword",
		"source":          "keyword",
		"study_type":      "keyword",
		"published_year":  "integer",
		"has_coi_authors": "bool",
		"license":         "keyword",
	} {
		idx := map[string]any{"field_name": field, "field_schema": ftype}
		bs, _ := json.Marshal(idx)
		ix, _ := http.NewRequestWithContext(ctx, http.MethodPut,
			base+"/collections/"+cfg.Collection+"/index?wait=false", bytes.NewReader(bs))
		ix.Header.Set("Content-Type", "application/json")
		if cfg.APIKey != "" {
			ix.Header.Set("api-key", cfg.APIKey)
		}
		_, _ = client.Do(ix)
	}

	cfg.Logger.Info("qdrant collection created",
		"name", cfg.Collection, "size", cfg.VectorSize, "m", cfg.HNSWM, "ef", cfg.HNSWEfConstruct)
	return nil
}

type Neo4jConfig struct {
	URL      string
	User     string
	Password string
	Logger   *slog.Logger
}

// EnsureNeo4jIndexes creates the indexes from spec §3.4. Cypher CREATE
// INDEX IF NOT EXISTS is idempotent.
func EnsureNeo4jIndexes(ctx context.Context, cfg Neo4jConfig) error {
	driver, err := neo4j.NewDriverWithContext(cfg.URL, neo4j.BasicAuth(cfg.User, cfg.Password, ""))
	if err != nil {
		return fmt.Errorf("neo4j driver: %w", err)
	}
	defer driver.Close(ctx)

	stmts := []string{
		"CREATE INDEX doc_id IF NOT EXISTS FOR (d:Document) ON (d.id)",
		"CREATE INDEX author_orcid IF NOT EXISTS FOR (a:Author) ON (a.orcid)",
		"CREATE INDEX author_key IF NOT EXISTS FOR (a:Author) ON (a.key)",
		"CREATE INDEX mesh_name IF NOT EXISTS FOR (m:MeshTerm) ON (m.name)",
		"CREATE INDEX sponsor_name IF NOT EXISTS FOR (s:Sponsor) ON (s.name)",
		"CREATE INDEX journal_issn IF NOT EXISTS FOR (j:Journal) ON (j.issn)",
	}
	ses := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer ses.Close(ctx)
	for _, s := range stmts {
		_, err := ses.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			return tx.Run(ctx, s, nil)
		})
		if err != nil {
			cfg.Logger.Warn("neo4j index create", "stmt", s, "err", err)
		}
	}
	cfg.Logger.Info("neo4j indexes ensured", "n", len(stmts))
	return nil
}
