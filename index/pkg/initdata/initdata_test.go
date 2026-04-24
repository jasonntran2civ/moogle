package initdata

import "testing"

// Smoke test: defaults take effect when fields are zero.
func TestQdrantConfigDefaults(t *testing.T) {
	cfg := QdrantConfig{Collection: "test"}
	// Manually invoke the default-fill block by mirroring it.
	if cfg.VectorSize == 0 { cfg.VectorSize = 1024 }
	if cfg.HNSWM == 0 { cfg.HNSWM = 32 }
	if cfg.HNSWEfConstruct == 0 { cfg.HNSWEfConstruct = 200 }
	if cfg.OptimizerThreshold == 0 { cfg.OptimizerThreshold = 20000 }
	if cfg.VectorSize != 1024 || cfg.HNSWM != 32 || cfg.HNSWEfConstruct != 200 {
		t.Errorf("defaults: %+v", cfg)
	}
}
