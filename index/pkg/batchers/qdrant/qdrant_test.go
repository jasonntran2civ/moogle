package qdrant

import "testing"

func TestIDToUint64Stable(t *testing.T) {
	a := idToUint64("pubmed:12345678")
	b := idToUint64("pubmed:12345678")
	if a != b {
		t.Error("idToUint64 not stable")
	}
	if idToUint64("pubmed:1") == idToUint64("pubmed:2") {
		t.Error("idToUint64 collision on small ids")
	}
	if a == 0 {
		t.Error("idToUint64 should not yield zero")
	}
}
