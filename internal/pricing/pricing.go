// Package pricing exposes per-model USD pricing to any package that needs to
// estimate the cost of an API key's usage: the enforcement middleware in
// internal/api, and the management panel in internal/api/handlers/management.
//
// Pricing sources, in order of precedence:
//  1. Exact match in the external file's `models` map (case-insensitive).
//  2. First matching glob in the external file's `patterns` array (case-insensitive).
//  3. Exact match in the built-in GPT/Codex price table.
//  4. `gpt-*` prefix -> the built-in unknown-GPT fallback.
//  5. Nothing (the model contributes zero to the budget).
//
// External pricing file location, in order:
//  1. `PPAP_MODEL_PRICING_FILE` env var.
//  2. `model-pricing.json` next to the running binary.
//  3. `model-pricing.json` in the process working directory.
//
// The file is read lazily on the first lookup (sync.Once). Restart the service
// to pick up edits.
package pricing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Price is the per-million-token USD price triple.
type Price struct {
	Input       float64
	CachedInput float64
	Output      float64
}

// Zero is the sentinel returned when no price applies (untracked model).
var Zero = Price{}

// gptModelPrices is the built-in fallback for GPT/Codex models.
var gptModelPrices = map[string]Price{
	"gpt-5.5":                    {Input: 5, CachedInput: 0.5, Output: 30},
	"gpt-5.5-low-fast":           {Input: 5, CachedInput: 0.5, Output: 30},
	"gpt-5.5-medium-fast":        {Input: 5, CachedInput: 0.5, Output: 30},
	"gpt-5.5-high-fast":          {Input: 5, CachedInput: 0.5, Output: 30},
	"gpt-5.5-xhigh-fast":         {Input: 5, CachedInput: 0.5, Output: 30},
	"gpt-5.4":                    {Input: 2.5, CachedInput: 0.25, Output: 15},
	"gpt-5.4-mini":               {Input: 0.75, CachedInput: 0.075, Output: 4.5},
	"gpt-5.4-nano":               {Input: 0.2, CachedInput: 0.02, Output: 1.25},
	"gpt-5.3-codex":              {Input: 1.75, CachedInput: 0.175, Output: 14},
	"gpt-5.3-codex-spark":        {Input: 1.75, CachedInput: 0.175, Output: 14},
	"gpt-5.3-codex-spark-low":    {Input: 1.75, CachedInput: 0.175, Output: 14},
	"gpt-5.3-codex-spark-medium": {Input: 1.75, CachedInput: 0.175, Output: 14},
	"gpt-5.3-codex-spark-high":   {Input: 1.75, CachedInput: 0.175, Output: 14},
	"gpt-5.3-codex-spark-xhigh":  {Input: 1.75, CachedInput: 0.175, Output: 14},
	"gpt-5":                      {Input: 1.25, CachedInput: 0.125, Output: 10},
	"gpt-5-mini":                 {Input: 0.25, CachedInput: 0.025, Output: 2},
	"gpt-5-nano":                 {Input: 0.05, CachedInput: 0.005, Output: 0.4},
	"gpt-5-pro":                  {Input: 15, CachedInput: 0, Output: 120},
}

// unknownGPTPrice is the conservative fallback for unrecognised gpt-* models.
var unknownGPTPrice = Price{Input: 15, CachedInput: 0, Output: 120}

type externalPriceEntry struct {
	Input       float64 `json:"input"`
	CachedInput float64 `json:"cached_input"`
	Output      float64 `json:"output"`
}

type externalPricingFile struct {
	Models   map[string]externalPriceEntry `json:"models"`
	Patterns []struct {
		Pattern string             `json:"pattern"`
		Price   externalPriceEntry `json:"price"`
	} `json:"patterns"`
}

type patternPrice struct {
	pattern string
	price   Price
}

var (
	externalOnce     sync.Once
	externalModels   map[string]Price
	externalPatterns []patternPrice
	externalLoadedOK bool
	externalLoadPath string
)

// NormalizeModelName lower-cases and strips routing prefixes from a model id
// so it can be matched against the pricing tables. Exported so callers can
// share the same normalisation the enforcement middleware uses.
func NormalizeModelName(model string) string {
	model = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(model, "models/")))
	if idx := strings.LastIndex(model, "/"); idx >= 0 {
		model = strings.TrimSpace(model[idx+1:])
	}
	return strings.TrimPrefix(model, "models/")
}

// ForModel resolves the price table entry for a (raw or normalised) model id.
//
// Returns ok=false when the model is unknown to every source, in which case
// requests for that model contribute zero to the budget.
func ForModel(model string) (Price, bool) {
	normalized := NormalizeModelName(model)
	if normalized == "" {
		return Zero, false
	}
	if price, ok := externalForModel(normalized); ok {
		return price, true
	}
	if price, ok := gptModelPrices[normalized]; ok {
		return price, true
	}
	if strings.HasPrefix(normalized, "gpt-") {
		return unknownGPTPrice, true
	}
	return Zero, false
}

// Source returns metadata about how ForModel would resolve `model`, useful for
// the management panel to explain to operators why a specific price applies.
type Source string

const (
	SourceExternalExact   Source = "external:exact"
	SourceExternalPattern Source = "external:pattern"
	SourceBuiltinGPT      Source = "builtin:gpt"
	SourceBuiltinGPTFall  Source = "builtin:gpt-fallback"
	SourceUnknown         Source = "unknown"
)

// LookupWithSource is a diagnostic variant that also reports where the price
// came from. The management panel uses it to display "priced by …" tooltips.
func LookupWithSource(model string) (Price, Source, string) {
	normalized := NormalizeModelName(model)
	if normalized == "" {
		return Zero, SourceUnknown, ""
	}
	externalOnce.Do(loadExternal)
	if externalLoadedOK {
		if price, ok := externalModels[normalized]; ok {
			return price, SourceExternalExact, normalized
		}
		for _, entry := range externalPatterns {
			if matched, _ := filepath.Match(entry.pattern, normalized); matched {
				return entry.price, SourceExternalPattern, entry.pattern
			}
		}
	}
	if price, ok := gptModelPrices[normalized]; ok {
		return price, SourceBuiltinGPT, normalized
	}
	if strings.HasPrefix(normalized, "gpt-") {
		return unknownGPTPrice, SourceBuiltinGPTFall, "gpt-*"
	}
	return Zero, SourceUnknown, ""
}

// ExternalLoadPath returns the path the external pricing file was loaded from,
// or "" if no external file was used. Read after ForModel or LookupWithSource
// has run at least once.
func ExternalLoadPath() string {
	externalOnce.Do(loadExternal)
	return externalLoadPath
}

func externalForModel(normalized string) (Price, bool) {
	externalOnce.Do(loadExternal)
	if !externalLoadedOK {
		return Zero, false
	}
	if price, ok := externalModels[normalized]; ok {
		return price, true
	}
	for _, entry := range externalPatterns {
		if matched, _ := filepath.Match(entry.pattern, normalized); matched {
			return entry.price, true
		}
	}
	return Zero, false
}

func loadExternal() {
	for _, path := range candidatePaths() {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var parsed externalPricingFile
		if err := json.Unmarshal(data, &parsed); err != nil {
			log.Warnf("external pricing file %s failed to parse: %v (falling back to built-in table)", path, err)
			continue
		}
		externalModels = normalizeExternalModelMap(parsed.Models)
		externalPatterns = normalizeExternalPatternList(parsed.Patterns)
		externalLoadedOK = true
		externalLoadPath = path
		return
	}
}

func candidatePaths() []string {
	var out []string
	if env := strings.TrimSpace(os.Getenv("PPAP_MODEL_PRICING_FILE")); env != "" {
		out = append(out, env)
	}
	if exePath, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
			exePath = resolved
		}
		out = append(out, filepath.Join(filepath.Dir(exePath), "model-pricing.json"))
	}
	out = append(out, "model-pricing.json")
	return out
}

func normalizeExternalModelMap(in map[string]externalPriceEntry) map[string]Price {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]Price, len(in))
	for k, v := range in {
		out[strings.ToLower(strings.TrimSpace(k))] = Price{
			Input:       v.Input,
			CachedInput: v.CachedInput,
			Output:      v.Output,
		}
	}
	return out
}

func normalizeExternalPatternList(in []struct {
	Pattern string             `json:"pattern"`
	Price   externalPriceEntry `json:"price"`
}) []patternPrice {
	if len(in) == 0 {
		return nil
	}
	out := make([]patternPrice, 0, len(in))
	for _, entry := range in {
		pattern := strings.ToLower(strings.TrimSpace(entry.Pattern))
		if pattern == "" {
			continue
		}
		out = append(out, patternPrice{
			pattern: pattern,
			price: Price{
				Input:       entry.Price.Input,
				CachedInput: entry.Price.CachedInput,
				Output:      entry.Price.Output,
			},
		})
	}
	return out
}

// EstimateRequestCostUSD computes the USD cost of a single request given its
// token stats. Callers who only need aggregate cost across many models should
// call this per model and sum, using the tokens payloads PPAP already tracks.
func EstimateRequestCostUSD(price Price, inputTokens, cachedTokens, outputTokens, totalTokens int64) float64 {
	if inputTokens < 0 {
		inputTokens = 0
	}
	if cachedTokens < 0 {
		cachedTokens = 0
	}
	if cachedTokens > inputTokens {
		cachedTokens = inputTokens
	}
	uncachedInputTokens := inputTokens - cachedTokens
	if outputTokens < 0 {
		outputTokens = 0
	}
	if outputTokens == 0 && totalTokens > inputTokens {
		outputTokens = totalTokens - inputTokens
	}
	return (float64(uncachedInputTokens)*price.Input +
		float64(cachedTokens)*price.CachedInput +
		float64(outputTokens)*price.Output) / 1_000_000
}
