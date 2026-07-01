package registry

import "testing"

func TestApplyLocalCatalogOverridesAddsMissingModel(t *testing.T) {
	parsed := &staticModelsJSON{
		Claude: []*ModelInfo{{ID: "claude-sonnet-4-6"}},
	}

	applyLocalCatalogOverrides(parsed)

	found := false
	for _, m := range parsed.Claude {
		if m != nil && m.ID == "claude-sonnet-5" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected claude-sonnet-5 to be added, got %d claude models", len(parsed.Claude))
	}
	if len(parsed.Claude) != 2 {
		t.Fatalf("expected exactly 2 claude models (original + override), got %d", len(parsed.Claude))
	}
}

func TestApplyLocalCatalogOverridesIsIdempotentWhenAlreadyPresent(t *testing.T) {
	// Simulates upstream eventually shipping the same model ID: the override
	// must not duplicate it.
	parsed := &staticModelsJSON{
		Claude: []*ModelInfo{
			{ID: "claude-sonnet-4-6"},
			{ID: "claude-sonnet-5", DisplayName: "Upstream Sonnet 5"},
		},
	}

	applyLocalCatalogOverrides(parsed)

	count := 0
	var displayName string
	for _, m := range parsed.Claude {
		if m != nil && m.ID == "claude-sonnet-5" {
			count++
			displayName = m.DisplayName
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 claude-sonnet-5 entry, got %d", count)
	}
	if displayName != "Upstream Sonnet 5" {
		t.Fatalf("expected upstream's entry to be left untouched, got display_name=%q", displayName)
	}
}

func TestApplyLocalCatalogOverridesHandlesNilCatalog(t *testing.T) {
	applyLocalCatalogOverrides(nil) // must not panic
}
