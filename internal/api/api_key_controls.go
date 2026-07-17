package api

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/tidwall/gjson"
)

// apiKeyControlsMiddleware enforces the optional per-downstream-key controls
// (enable/disable, model allow/deny patterns, and usage budgets) configured
// under `api-key-controls`. It is designed to run *after* the access
// authentication middleware, which stores the resolved principal under the
// Gin context key "userApiKey". When no controls are configured, or no control
// matches the request key, the middleware is a transparent pass-through.
func (s *Server) apiKeyControlsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.enforceAPIKeyControls(c) {
			return
		}
		c.Next()
	}
}

// requestAPIKey returns the downstream API key resolved for the current
// request. v7.2 access middleware stores it under "userApiKey"; the legacy
// PPAP key "apiKey" is also honoured for forward-compatibility.
func requestAPIKey(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if key := strings.TrimSpace(c.GetString("userApiKey")); key != "" {
		return key
	}
	return strings.TrimSpace(c.GetString("apiKey"))
}

// enforceAPIKeyControls applies the matching control for the current request
// key. It returns true when the request may proceed, and false when the
// middleware has already written an error response and aborted the context.
func (s *Server) enforceAPIKeyControls(c *gin.Context) bool {
	if s == nil || c == nil {
		return true
	}
	cfg := s.cfg
	if cfg == nil || len(cfg.APIKeyControls) == 0 {
		return true
	}
	apiKey := requestAPIKey(c)
	if apiKey == "" {
		return true
	}
	control := findAPIKeyControl(cfg, apiKey)
	if control == nil {
		return true
	}
	if control.Enabled != nil && !*control.Enabled {
		abortAPIKeyControl(c, http.StatusForbidden, "api_key_disabled", "API key is disabled")
		return false
	}
	if !withinAPIKeyBudget(control) {
		abortAPIKeyControl(c, http.StatusTooManyRequests, "api_key_budget_exceeded", "API key usage budget exceeded")
		return false
	}
	modelName, ok := extractRequestModel(c)
	if ok && modelName != "" && !apiKeyModelAllowed(control, modelName) {
		abortAPIKeyControl(c, http.StatusForbidden, "model_not_allowed", "Model is not allowed for this API key")
		return false
	}
	return true
}

// findAPIKeyControl locates the control entry whose api-key (or legacy key)
// field equals the supplied downstream API key.
func findAPIKeyControl(cfg *config.Config, apiKey string) *config.APIKeyControl {
	if cfg == nil || apiKey == "" {
		return nil
	}
	for i := range cfg.APIKeyControls {
		key := strings.TrimSpace(cfg.APIKeyControls[i].APIKey)
		if key == "" {
			key = strings.TrimSpace(cfg.APIKeyControls[i].Key)
		}
		if key == apiKey {
			return &cfg.APIKeyControls[i]
		}
	}
	return nil
}

// apiKeyUsageStats is the minimal usage view the budget enforcement needs.
// It mirrors the fields produced by the in-memory request-statistics store
// (the PPAP usage LoggerPlugin, ported in Step6). Keeping a local shape here
// lets the controls package enforce model/enable rules and stay build-green
// before the usage aggregation symbols land.
type apiKeyUsageStats struct {
	// TotalRequests is the number of requests served for the key.
	TotalRequests int64
	// TotalInputTokens is the cumulative prompt token count for the key.
	TotalInputTokens int64
	// TotalTokens is the cumulative total token count for the key.
	TotalTokens int64
	// Models maps a model id to its per-model usage detail, used for the
	// estimated-cost budget.
	Models map[string]apiKeyModelUsage
}

// apiKeyModelUsage holds the per-model token details for one API key.
type apiKeyModelUsage struct {
	// Details is the list of per-request token breakdowns for the model.
	Details []apiKeyTokenStats
}

// apiKeyTokenStats captures the token usage breakdown for a single request.
type apiKeyTokenStats struct {
	InputTokens  int64
	CachedTokens int64
	OutputTokens int64
	TotalTokens  int64
}

// apiKeyUsageStatsLookup resolves the aggregated usage for a downstream API
// key. server.go wires this at startup via installAPIKeyUsageStatsLookup, so
// under normal boot it is non-nil and the request-count, input/total-token, and
// estimated-cost budgets are enforceable. If ever left nil (e.g. tests that
// reset it), every usage-derived budget degrades to a permissive no-op so that
// only the model allow/deny and enabled controls are enforced.
var apiKeyUsageStatsLookup func(apiKey string) (apiKeyUsageStats, bool)

// SetAPIKeyUsageStatsLookup installs the usage statistics provider used by the
// per-key budget enforcement. Passing nil disables usage-derived budgets.
func SetAPIKeyUsageStatsLookup(lookup func(apiKey string) (apiKeyUsageStats, bool)) {
	apiKeyUsageStatsLookup = lookup
}

// withinAPIKeyBudget reports whether the supplied control is still within its
// configured request/token/cost budgets. When no usage provider is installed
// it returns true, since the underlying counters are unavailable — this is only
// expected in tests that intentionally reset the lookup.
func withinAPIKeyBudget(control *config.APIKeyControl) bool {
	if control == nil || control.Unlimited {
		return true
	}
	if control.MaxRequests <= 0 && control.MaxInputTokens <= 0 && control.MaxTotalTokens <= 0 && control.MaxCostUSD <= 0 {
		return true
	}
	if apiKeyUsageStatsLookup == nil {
		// Usage aggregation not installed (only expected in tests that reset
		// the lookup). Budgets cannot be enforced without per-key counters,
		// so the request is allowed.
		return true
	}
	key := strings.TrimSpace(control.APIKey)
	if key == "" {
		key = strings.TrimSpace(control.Key)
	}
	if key == "" {
		return true
	}
	stats, ok := apiKeyUsageStatsLookup(key)
	if !ok {
		return true
	}
	if control.MaxRequests > 0 && stats.TotalRequests >= control.MaxRequests {
		return false
	}
	if control.MaxInputTokens > 0 && stats.TotalInputTokens >= control.MaxInputTokens {
		return false
	}
	if control.MaxTotalTokens > 0 && stats.TotalTokens >= control.MaxTotalTokens {
		return false
	}
	if control.MaxCostUSD > 0 && estimateAPIKeyCostUSD(stats) >= control.MaxCostUSD {
		return false
	}
	return true
}

// apiKeyModelPrice captures the per-million-token USD price for a model.
type apiKeyModelPrice struct {
	input       float64
	cachedInput float64
	output      float64
}

// apiKeyGPTModelPrices holds the GPT/Codex per-million-token USD prices used to
// estimate per-key spend for the MaxCostUSD budget.
var apiKeyGPTModelPrices = map[string]apiKeyModelPrice{
	"gpt-5.5":                    {input: 5, cachedInput: 0.5, output: 30},
	"gpt-5.5-low-fast":           {input: 5, cachedInput: 0.5, output: 30},
	"gpt-5.5-medium-fast":        {input: 5, cachedInput: 0.5, output: 30},
	"gpt-5.5-high-fast":          {input: 5, cachedInput: 0.5, output: 30},
	"gpt-5.5-xhigh-fast":         {input: 5, cachedInput: 0.5, output: 30},
	"gpt-5.4":                    {input: 2.5, cachedInput: 0.25, output: 15},
	"gpt-5.4-mini":               {input: 0.75, cachedInput: 0.075, output: 4.5},
	"gpt-5.4-nano":               {input: 0.2, cachedInput: 0.02, output: 1.25},
	"gpt-5.3-codex":              {input: 1.75, cachedInput: 0.175, output: 14},
	"gpt-5.3-codex-spark":        {input: 1.75, cachedInput: 0.175, output: 14},
	"gpt-5.3-codex-spark-low":    {input: 1.75, cachedInput: 0.175, output: 14},
	"gpt-5.3-codex-spark-medium": {input: 1.75, cachedInput: 0.175, output: 14},
	"gpt-5.3-codex-spark-high":   {input: 1.75, cachedInput: 0.175, output: 14},
	"gpt-5.3-codex-spark-xhigh":  {input: 1.75, cachedInput: 0.175, output: 14},
	"gpt-5":                      {input: 1.25, cachedInput: 0.125, output: 10},
	"gpt-5-mini":                 {input: 0.25, cachedInput: 0.025, output: 2},
	"gpt-5-nano":                 {input: 0.05, cachedInput: 0.005, output: 0.4},
	"gpt-5-pro":                  {input: 15, cachedInput: 0, output: 120},
}

// apiKeyUnknownGPTPrice is the conservative fallback price for unrecognised
// gpt-* models when estimating per-key spend.
var apiKeyUnknownGPTPrice = apiKeyModelPrice{input: 15, cachedInput: 0, output: 120}

// estimateAPIKeyCostUSD sums the estimated USD spend across all priced models
// recorded for the API key.
func estimateAPIKeyCostUSD(stats apiKeyUsageStats) float64 {
	var total float64
	for model, modelStats := range stats.Models {
		price, ok := priceForAPIKeyModel(model)
		if !ok {
			continue
		}
		for _, detail := range modelStats.Details {
			total += estimateTokenStatsCostUSD(detail, price)
		}
	}
	return total
}

// priceForAPIKeyModel resolves the price table entry for a model id.
//
// Resolution order:
//  1. External pricing file (JSON) — exact match, then glob patterns.
//  2. Built-in GPT/Codex price table.
//  3. `gpt-*` prefix -> the conservative `apiKeyUnknownGPTPrice` fallback.
//  4. Nothing (the model contributes zero to the budget).
//
// See api_key_pricing_external.go for how the JSON file is loaded and where
// it is looked up. Restart the service to pick up file edits.
func priceForAPIKeyModel(model string) (apiKeyModelPrice, bool) {
	model = normalizeAPIKeyCostModel(model)
	if model == "" {
		return apiKeyModelPrice{}, false
	}
	if price, ok := externalPriceForModel(model); ok {
		return price, true
	}
	if price, ok := apiKeyGPTModelPrices[model]; ok {
		return price, true
	}
	if strings.HasPrefix(model, "gpt-") {
		return apiKeyUnknownGPTPrice, true
	}
	return apiKeyModelPrice{}, false
}

// normalizeAPIKeyCostModel lower-cases and strips routing prefixes from a model
// id so it can be matched against the price table.
func normalizeAPIKeyCostModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(model, "models/")))
	if idx := strings.LastIndex(model, "/"); idx >= 0 {
		model = strings.TrimSpace(model[idx+1:])
	}
	return strings.TrimPrefix(model, "models/")
}

// estimateTokenStatsCostUSD computes the USD cost of a single request's token
// breakdown using the supplied per-million-token price.
func estimateTokenStatsCostUSD(tokens apiKeyTokenStats, price apiKeyModelPrice) float64 {
	inputTokens := clampNonNegative(tokens.InputTokens)
	cachedTokens := clampNonNegative(tokens.CachedTokens)
	if cachedTokens > inputTokens {
		cachedTokens = inputTokens
	}
	uncachedInputTokens := inputTokens - cachedTokens
	outputTokens := clampNonNegative(tokens.OutputTokens)
	if outputTokens == 0 && tokens.TotalTokens > inputTokens {
		outputTokens = tokens.TotalTokens - inputTokens
	}
	return (float64(uncachedInputTokens)*price.input +
		float64(cachedTokens)*price.cachedInput +
		float64(outputTokens)*price.output) / 1_000_000
}

// clampNonNegative returns value or 0 when value is negative.
func clampNonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

// extractRequestModel resolves the requested model from the URL path, the
// `model` query parameter, or a JSON request body. The request body is read
// fully and restored so downstream handlers observe the original payload.
func extractRequestModel(c *gin.Context) (string, bool) {
	if c == nil || c.Request == nil {
		return "", false
	}
	if model := extractGeminiModelFromPath(c.Request.URL.Path); model != "" {
		return model, true
	}
	if queryModel := strings.TrimSpace(c.Query("model")); queryModel != "" {
		return queryModel, true
	}
	if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodPatch {
		return "", false
	}
	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	if contentType != "" && !strings.Contains(contentType, "json") {
		return "", false
	}
	if c.Request.Body == nil {
		return "", false
	}
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Request.Body = io.NopCloser(bytes.NewReader(nil))
		return "", false
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))
	if len(bytes.TrimSpace(rawBody)) == 0 {
		return "", false
	}
	modelResult := gjson.GetBytes(rawBody, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String {
		return "", false
	}
	return strings.TrimSpace(modelResult.String()), true
}

// extractGeminiModelFromPath extracts the model id segment from a Gemini-style
// `/models/<model>:method` path.
func extractGeminiModelFromPath(path string) string {
	const marker = "/models/"
	idx := strings.Index(path, marker)
	if idx < 0 {
		return ""
	}
	model := path[idx+len(marker):]
	if model == "" {
		return ""
	}
	if cut := strings.IndexAny(model, ":/?#"); cut >= 0 {
		model = model[:cut]
	}
	return strings.TrimPrefix(strings.TrimSpace(model), "models/")
}

// apiKeyModelAllowed reports whether the supplied model is permitted for the
// control. Excluded patterns take precedence; an empty allow-list permits all
// non-excluded models.
func apiKeyModelAllowed(control *config.APIKeyControl, model string) bool {
	if control == nil {
		return true
	}
	model = strings.TrimPrefix(strings.TrimSpace(model), "models/")
	if model == "" {
		return true
	}
	for _, pattern := range control.ExcludedModels {
		if modelPatternMatches(model, pattern) {
			return false
		}
	}
	if len(control.Models) == 0 {
		return true
	}
	for _, pattern := range control.Models {
		if modelPatternMatches(model, pattern) {
			return true
		}
	}
	return false
}

// modelPatternMatches reports whether model matches pattern. A pattern of "*"
// matches anything; "*" wildcards inside a pattern match arbitrary substrings.
func modelPatternMatches(model, pattern string) bool {
	model = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(model, "models/")))
	pattern = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(pattern, "models/")))
	if model == "" || pattern == "" {
		return false
	}
	if pattern == "*" || pattern == model {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return false
	}
	parts := strings.Split(pattern, "*")
	position := 0
	if parts[0] != "" {
		if !strings.HasPrefix(model, parts[0]) {
			return false
		}
		position = len(parts[0])
	}
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if part == "" {
			continue
		}
		next := strings.Index(model[position:], part)
		if next < 0 {
			return false
		}
		position += next + len(part)
	}
	last := parts[len(parts)-1]
	return last == "" || strings.HasSuffix(model, last)
}

// abortAPIKeyControl writes a structured access error and aborts the request.
func abortAPIKeyControl(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    "api_key_access_error",
			"code":    code,
		},
	})
}
