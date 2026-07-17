// Package api — external pricing table loader.
//
// The hard-coded per-model prices in api_key_controls.go only cover GPT/Codex
// models, which means keys scoped to Claude/Anthropic families accrue zero
// estimated cost and their `max-cost-usd` budget can never trigger. This file
// adds an optional external JSON pricing file that supplements (and overrides)
// the built-in table so operators can add Claude, Gemini, or any future
// model family without a new PPAP release.
//
// Resolution order for a given model id:
//  1. Exact match in the external file's `models` map (case-insensitive).
//  2. First matching glob in the external file's `patterns` array (case-insensitive).
//  3. Exact match in the built-in GPT price table.
//  4. `gpt-*` prefix -> the built-in `apiKeyUnknownGPTPrice` fallback.
//  5. Nothing -> the request contributes zero to the budget.
//
// Path resolution for the external file:
//  1. `PPAP_MODEL_PRICING_FILE` env var (highest priority; empty disables).
//  2. `model-pricing.json` in the same directory as the running binary.
//  3. `model-pricing.json` in the process working directory.
//
// The file is read lazily on the first pricing lookup (sync.Once). Restart the
// service to pick up edits.
package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type externalPriceEntry struct {
	Input       float64 `json:"input"`
	CachedInput float64 `json:"cached_input"`
	Output      float64 `json:"output"`
}

type externalPricingFile struct {
	// Models is a map of exact model id -> price. Keys are matched
	// case-insensitively against the normalised request model id.
	Models map[string]externalPriceEntry `json:"models"`
	// Patterns is an ordered array of glob-based fallbacks applied when no
	// exact-match entry exists. Standard path.Match syntax is used, so
	// `claude-sonnet-*` matches `claude-sonnet-5`, `claude-sonnet-5-thinking`
	// etc. First match wins, so order patterns from most-specific to
	// most-general.
	Patterns []struct {
		Pattern string             `json:"pattern"`
		Price   externalPriceEntry `json:"price"`
	} `json:"patterns"`
}

type patternPrice struct {
	pattern string
	price   apiKeyModelPrice
}

var (
	externalPricingOnce      sync.Once
	externalPricingModels    map[string]apiKeyModelPrice
	externalPricingPatterns  []patternPrice
	externalPricingLoadedOK  bool
	externalPricingLoadPath  string
)

// loadExternalPricing populates the package-level tables from the first
// discovered pricing file. Safe to call from priceForAPIKeyModel via sync.Once.
func loadExternalPricing() {
	for _, path := range candidateExternalPricingPaths() {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var parsed externalPricingFile
		if err := json.Unmarshal(data, &parsed); err != nil {
			// Malformed JSON is loud in the log but not fatal — the built-in
			// GPT table remains authoritative.
			logExternalPricingError(path, err)
			continue
		}
		externalPricingModels = normaliseExternalModelMap(parsed.Models)
		externalPricingPatterns = normaliseExternalPatternList(parsed.Patterns)
		externalPricingLoadedOK = true
		externalPricingLoadPath = path
		return
	}
}

// candidateExternalPricingPaths returns the ordered list of paths to probe for
// the pricing file. Empty strings and duplicates are filtered by the caller.
func candidateExternalPricingPaths() []string {
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

// normaliseExternalModelMap lower-cases keys and converts entries to the
// internal apiKeyModelPrice type used by the rest of the package.
func normaliseExternalModelMap(in map[string]externalPriceEntry) map[string]apiKeyModelPrice {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]apiKeyModelPrice, len(in))
	for k, v := range in {
		out[strings.ToLower(strings.TrimSpace(k))] = apiKeyModelPrice{
			input:       v.Input,
			cachedInput: v.CachedInput,
			output:      v.Output,
		}
	}
	return out
}

// normaliseExternalPatternList converts the parsed patterns into the internal
// representation, preserving order.
func normaliseExternalPatternList(in []struct {
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
			price: apiKeyModelPrice{
				input:       entry.Price.Input,
				cachedInput: entry.Price.CachedInput,
				output:      entry.Price.Output,
			},
		})
	}
	return out
}

// externalPriceForModel resolves the pricing file's entry for the supplied
// (already normalised, lower-cased) model id. Returns ok=false when the model
// has no match in the external table.
func externalPriceForModel(model string) (apiKeyModelPrice, bool) {
	externalPricingOnce.Do(loadExternalPricing)
	if !externalPricingLoadedOK {
		return apiKeyModelPrice{}, false
	}
	if price, ok := externalPricingModels[model]; ok {
		return price, true
	}
	for _, entry := range externalPricingPatterns {
		if matched, _ := filepath.Match(entry.pattern, model); matched {
			return entry.price, true
		}
	}
	return apiKeyModelPrice{}, false
}
