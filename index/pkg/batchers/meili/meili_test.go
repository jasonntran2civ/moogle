package meili

import "testing"

func TestFlattenPromotesFacets(t *testing.T) {
	d := map[string]any{
		"id":              "pubmed:1",
		"title":           "x",
		"published_at":    "2024-03-15T00:00:00Z",
		"study_type":      "RCT",
		"journal":         map[string]any{"name": "NEJM", "is_predatory": false},
		"authors":         []any{map[string]any{"display_name": "Smith J"}},
		"has_coi_authors": true,
		"full_text":       "long text",
	}
	out := flatten(d)
	if out["id"] != "pubmed:1" || out["study_type"] != "RCT" {
		t.Fatalf("flatten basic: %+v", out)
	}
	if out["published_year"] != 2024 {
		t.Errorf("published_year: %v", out["published_year"])
	}
	if out["journal_name"] != "NEJM" {
		t.Errorf("journal_name: %v", out["journal_name"])
	}
	if out["has_full_text"] != true {
		t.Errorf("has_full_text should be true when full_text non-empty")
	}
	names, _ := out["authors_display"].([]string)
	if len(names) != 1 || names[0] != "Smith J" {
		t.Errorf("authors_display: %v", names)
	}
}

func TestFlattenHandlesEmptyFullText(t *testing.T) {
	out := flatten(map[string]any{"id": "x", "full_text": ""})
	if out["has_full_text"] != false {
		t.Error("has_full_text should be false on empty string")
	}
}
