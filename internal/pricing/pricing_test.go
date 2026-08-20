package pricing

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func resetExternalPricingForTest() {
	externalOnce = sync.Once{}
	externalModels = nil
	externalPatterns = nil
	externalLoadedOK = false
	externalLoadPath = ""
}

func TestNormalizeAndEstimateRequestCost(t *testing.T) {
	if got := NormalizeModelName("provider/models/GPT-5.6"); got != "gpt-5.6" {
		t.Fatalf("NormalizeModelName() = %q", got)
	}
	price := Price{Input: 2, CachedInput: 0.2, Output: 10}
	got := EstimateRequestCostUSD(price, 1000, 400, 500, 1500)
	want := (600.0*2 + 400.0*0.2 + 500.0*10) / 1_000_000
	if got != want {
		t.Fatalf("EstimateRequestCostUSD() = %f, want %f", got, want)
	}
}

func TestForModelUsesBuiltInAndGPTFallback(t *testing.T) {
	resetExternalPricingForTest()
	t.Setenv("PPAP_MODEL_PRICING_FILE", filepath.Join(t.TempDir(), "missing.json"))
	known, ok := ForModel("gpt-5.4")
	if !ok || known.Input <= 0 || known.Output <= 0 {
		t.Fatalf("known GPT price missing: %#v, %v", known, ok)
	}
	fallback, ok := ForModel("gpt-5.6")
	if !ok || fallback != unknownGPTPrice {
		t.Fatalf("GPT fallback mismatch: %#v, %v", fallback, ok)
	}
	if _, ok := ForModel("claude-sonnet-5"); ok {
		t.Fatal("unknown non-GPT model unexpectedly received built-in pricing")
	}
}

func TestForModelPrefersExternalExactAndPatternPrices(t *testing.T) {
	resetExternalPricingForTest()
	pricingPath := filepath.Join(t.TempDir(), "pricing.json")
	contents := `{"models":{"gpt-5.6":{"input":3,"cached_input":0.3,"output":18}},"patterns":[{"pattern":"claude-*","price":{"input":4,"cached_input":0.4,"output":20}}]}`
	if err := os.WriteFile(pricingPath, []byte(contents), 0o600); err != nil {
		t.Fatalf("write pricing file: %v", err)
	}
	t.Setenv("PPAP_MODEL_PRICING_FILE", pricingPath)

	exact, source, matched := LookupWithSource("GPT-5.6")
	if exact.Input != 3 || exact.Output != 18 || source != SourceExternalExact || matched != "gpt-5.6" {
		t.Fatalf("external exact lookup mismatch: %#v, %q, %q", exact, source, matched)
	}
	pattern, source, matched := LookupWithSource("claude-sonnet-5")
	if pattern.Input != 4 || pattern.Output != 20 || source != SourceExternalPattern || matched != "claude-*" {
		t.Fatalf("external pattern lookup mismatch: %#v, %q, %q", pattern, source, matched)
	}
	if ExternalLoadPath() != pricingPath {
		t.Fatalf("ExternalLoadPath() = %q, want %q", ExternalLoadPath(), pricingPath)
	}
}
