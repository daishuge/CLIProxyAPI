// Structured CRUD for the api-key-controls YAML section — the counterpart of
// the existing GET/PUT/PATCH/DELETE on /api-keys, which only speaks to the
// flat top-level api-keys string list. api-key-controls carries the per-key
// name, model whitelist/blacklist, budgets and preset-prompt state that
// PPAP's fork extended, and until now was only editable through the raw
// config.yaml YAML editor. This file adds:
//
//	GET    /v0/management/api-key-controls           list with live usage + estimated cost
//	POST   /v0/management/api-key-controls           create (auto-generates api-key if omitted)
//	PATCH  /v0/management/api-key-controls           partial update by name (or index)
//	DELETE /v0/management/api-key-controls           delete by name (or index)
//
// Every write path re-persists config.yaml and triggers the same live-reload
// flow the other management endpoints use, so changes take effect immediately
// on the running proxy.
package management

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/pricing"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/usage"
)

// apiKeyControlResponse is the JSON shape the management panel consumes. It
// mirrors config.APIKeyControl but adds derived fields (hash, masked key,
// live usage) so the frontend can render without doing a second round-trip.
type apiKeyControlResponse struct {
	// Position in config for stable references.
	Index int `json:"index"`
	// Identity.
	Name       string `json:"name"`
	APIKey     string `json:"api-key"`
	APIKeyMask string `json:"api-key-mask"`
	APIKeyHash string `json:"api-key-hash"`
	// Enforcement.
	Enabled             *bool    `json:"enabled,omitempty"`
	Unlimited           bool     `json:"unlimited"`
	Models              []string `json:"models,omitempty"`
	ExcludedModels      []string `json:"excluded-models,omitempty"`
	MaxRequests         int64    `json:"max-requests"`
	MaxInputTokens      int64    `json:"max-input-tokens"`
	MaxTotalTokens      int64    `json:"max-total-tokens"`
	MaxCostUSD          float64  `json:"max-cost-usd"`
	PresetPromptEnabled bool     `json:"preset-prompt-enabled"`
	PresetPromptExcerpt string   `json:"preset-prompt-excerpt,omitempty"`
	// Live usage snapshot.
	Usage *apiKeyControlUsage `json:"usage,omitempty"`
}

type apiKeyControlUsage struct {
	TotalRequests     int64                         `json:"total_requests"`
	FailureCount      int64                         `json:"failure_count"`
	TotalTokens       int64                         `json:"total_tokens"`
	TotalInputTokens  int64                         `json:"total_input_tokens"`
	TotalCachedTokens int64                         `json:"total_cached_tokens"`
	UsedUSD           float64                       `json:"used_usd"`
	RemainingUSD      float64                       `json:"remaining_usd"`
	UsedPercent       float64                       `json:"used_percent"`
	Exhausted         bool                          `json:"exhausted"`
	AvgLatencyMs      int64                         `json:"avg_latency_ms"`
	Models            []apiKeyControlModelBreakdown `json:"models"`
	Recent            []apiKeyControlRecentRequest  `json:"recent,omitempty"`
}

type apiKeyControlModelBreakdown struct {
	Name            string         `json:"name"`
	Requests        int64          `json:"requests"`
	Tokens          int64          `json:"tokens"`
	CachedTokens    int64          `json:"cached_tokens"`
	UsedUSD         float64        `json:"used_usd"`
	AvgLatencyMs    int64          `json:"avg_latency_ms"`
	PriceSource     pricing.Source `json:"price_source"`
	PriceMatched    string         `json:"price_matched,omitempty"`
	PriceInputPerM  float64        `json:"price_input_per_m"`
	PriceCachedPerM float64        `json:"price_cached_input_per_m"`
	PriceOutputPerM float64        `json:"price_output_per_m"`
}

type apiKeyControlRecentRequest struct {
	Timestamp    time.Time `json:"timestamp"`
	Model        string    `json:"model"`
	Source       string    `json:"source,omitempty"`
	InputTokens  int64     `json:"input_tokens"`
	CachedTokens int64     `json:"cached_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	TotalTokens  int64     `json:"total_tokens"`
	LatencyMs    int64     `json:"latency_ms"`
	Failed       bool      `json:"failed"`
	CostUSD      float64   `json:"cost_usd"`
}

// GetAPIKeyControls returns every api-key-controls entry with a live usage
// snapshot. Query parameters:
//
//	include=usage      (default on) merge in per-key usage + cost
//	recent=N           (default 5) how many recent request details per model to include
//	mask_key=1         (default 0) redact api-key value, keep the mask + hash
func (h *Handler) GetAPIKeyControls(c *gin.Context) {
	if h == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "handler not initialized"})
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	cfg := h.cfg
	stats := h.usageStats
	if cfg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "config not available"})
		return
	}

	includeUsage := c.DefaultQuery("include", "usage") == "usage" && stats != nil
	recentPerModel := parseIntQuery(c, "recent", 5, 0, 100)
	maskKey := c.Query("mask_key") == "1"

	items := make([]apiKeyControlResponse, 0, len(cfg.APIKeyControls))
	for i := range cfg.APIKeyControls {
		items = append(items, buildAPIKeyControlResponse(cfg, i, stats, includeUsage, recentPerModel, maskKey))
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].Name < items[b].Name })

	c.JSON(http.StatusOK, gin.H{
		"api-key-controls":      items,
		"external_pricing_file": pricing.ExternalLoadPath(),
	})
}

// PostAPIKeyControls creates a new api-key-controls entry. If the request body
// omits `api-key`, the server generates one with the sk-ppap- prefix. The new
// key is also appended to the top-level api-keys list so it's a valid downstream
// credential immediately.
func (h *Handler) PostAPIKeyControls(c *gin.Context) {
	if h == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "handler not initialized"})
		return
	}
	var body config.APIKeyControl
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.APIKey = strings.TrimSpace(body.APIKey)
	if body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if body.APIKey == "" {
		key, err := generateAPIKey(body.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate api-key: " + err.Error()})
			return
		}
		body.APIKey = key
	}
	body.Key = ""

	h.mu.Lock()
	defer h.mu.Unlock()

	// Reject duplicate by api-key or name.
	for i := range h.cfg.APIKeyControls {
		existing := &h.cfg.APIKeyControls[i]
		if existing.APIKey == body.APIKey || (body.Name != "" && existing.Name == body.Name) {
			c.JSON(http.StatusConflict, gin.H{"error": "api-key or name already exists"})
			return
		}
	}

	h.cfg.APIKeyControls = append(h.cfg.APIKeyControls, body)
	apiKeyAdded := false
	if !containsString(h.cfg.APIKeys, body.APIKey) {
		h.cfg.APIKeys = append(h.cfg.APIKeys, body.APIKey)
		apiKeyAdded = true
	}

	if !h.persistAPIKeyControlsLocked(c) {
		h.cfg.APIKeyControls = h.cfg.APIKeyControls[:len(h.cfg.APIKeyControls)-1]
		if apiKeyAdded {
			removeString(&h.cfg.APIKeys, body.APIKey)
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"created": apiKeyControlResponseFromConfig(&body, len(h.cfg.APIKeyControls)-1, false),
	})
}

// PatchAPIKeyControls applies a partial update to a single api-key-controls
// entry, identified either by `name` or `index`. The `value` object mirrors
// config.APIKeyControl and only the present fields are applied; explicit nulls
// clear the field. Boolean pointers are used so the caller can distinguish
// "leave alone" from "set to false" on Enabled.
func (h *Handler) PatchAPIKeyControls(c *gin.Context) {
	if h == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "handler not initialized"})
		return
	}
	type patch struct {
		Name                *string   `json:"name"`
		APIKey              *string   `json:"api-key"`
		Enabled             *bool     `json:"enabled"`
		Unlimited           *bool     `json:"unlimited"`
		Models              *[]string `json:"models"`
		ExcludedModels      *[]string `json:"excluded-models"`
		MaxRequests         *int64    `json:"max-requests"`
		MaxInputTokens      *int64    `json:"max-input-tokens"`
		MaxTotalTokens      *int64    `json:"max-total-tokens"`
		MaxCostUSD          *float64  `json:"max-cost-usd"`
		PresetPromptEnabled *bool     `json:"preset-prompt-enabled"`
	}
	var body struct {
		Name  *string `json:"target_name"`
		Index *int    `json:"target_index"`
		Value *patch  `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Value == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	idx, err := resolveAPIKeyControlIndex(h.cfg, body.Name, body.Index)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	entry := &h.cfg.APIKeyControls[idx]
	originalEntry := *entry
	originalEntry.Models = cloneStrings(entry.Models)
	originalEntry.ExcludedModels = cloneStrings(entry.ExcludedModels)
	if entry.PresetPrompt != nil {
		presetCopy := *entry.PresetPrompt
		originalEntry.PresetPrompt = &presetCopy
	}
	originalAPIKeys := cloneStrings(h.cfg.APIKeys)
	v := body.Value

	if v.Name != nil {
		newName := strings.TrimSpace(*v.Name)
		if newName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		for i := range h.cfg.APIKeyControls {
			if i != idx && h.cfg.APIKeyControls[i].Name == newName {
				c.JSON(http.StatusConflict, gin.H{"error": "name already exists"})
				return
			}
		}
		entry.Name = newName
	}
	if v.APIKey != nil {
		newKey := strings.TrimSpace(*v.APIKey)
		if newKey != "" && newKey != entry.APIKey {
			// Refuse to swap the api-key to one that already belongs to another entry.
			for i := range h.cfg.APIKeyControls {
				if i != idx && h.cfg.APIKeyControls[i].APIKey == newKey {
					h.cfg.APIKeyControls[idx] = originalEntry
					h.cfg.APIKeys = originalAPIKeys
					c.JSON(http.StatusConflict, gin.H{"error": "api-key already in use by another entry"})
					return
				}
			}
			// Update the flat api-keys list too.
			replaceString(&h.cfg.APIKeys, entry.APIKey, newKey)
			entry.APIKey = newKey
		}
	}
	if v.Enabled != nil {
		enabled := *v.Enabled
		entry.Enabled = &enabled
	}
	if v.Unlimited != nil {
		entry.Unlimited = *v.Unlimited
	}
	if v.Models != nil {
		entry.Models = append([]string(nil), (*v.Models)...)
	}
	if v.ExcludedModels != nil {
		entry.ExcludedModels = append([]string(nil), (*v.ExcludedModels)...)
	}
	if v.MaxRequests != nil {
		entry.MaxRequests = *v.MaxRequests
	}
	if v.MaxInputTokens != nil {
		entry.MaxInputTokens = *v.MaxInputTokens
	}
	if v.MaxTotalTokens != nil {
		entry.MaxTotalTokens = *v.MaxTotalTokens
	}
	if v.MaxCostUSD != nil {
		entry.MaxCostUSD = *v.MaxCostUSD
	}
	if v.PresetPromptEnabled != nil {
		if entry.PresetPrompt == nil {
			entry.PresetPrompt = &config.PresetPromptConfig{}
		}
		entry.PresetPrompt.Enabled = *v.PresetPromptEnabled
	}

	if !h.persistAPIKeyControlsLocked(c) {
		h.cfg.APIKeyControls[idx] = originalEntry
		h.cfg.APIKeys = originalAPIKeys
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"updated": apiKeyControlResponseFromConfig(entry, idx, false),
	})
}

// DeleteAPIKeyControls removes a single api-key-controls entry, identified by
// name or index. The associated api-key is also removed from the flat api-keys
// list.
func (h *Handler) DeleteAPIKeyControls(c *gin.Context) {
	if h == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "handler not initialized"})
		return
	}
	var body struct {
		Name       *string `json:"name"`
		Index      *int    `json:"index"`
		KeepAPIKey bool    `json:"keep_api_key"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		if qn := strings.TrimSpace(c.Query("name")); qn != "" {
			body.Name = &qn
		}
		if qi := c.Query("index"); qi != "" {
			var n int
			if _, err := fmt.Sscanf(qi, "%d", &n); err == nil {
				body.Index = &n
			}
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	idx, err := resolveAPIKeyControlIndex(h.cfg, body.Name, body.Index)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	removed := h.cfg.APIKeyControls[idx]
	originalControls := append([]config.APIKeyControl(nil), h.cfg.APIKeyControls...)
	originalAPIKeys := cloneStrings(h.cfg.APIKeys)
	h.cfg.APIKeyControls = append(h.cfg.APIKeyControls[:idx], h.cfg.APIKeyControls[idx+1:]...)
	if !body.KeepAPIKey {
		removeString(&h.cfg.APIKeys, removed.APIKey)
	}
	if !h.persistAPIKeyControlsLocked(c) {
		h.cfg.APIKeyControls = originalControls
		h.cfg.APIKeys = originalAPIKeys
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": removed.Name, "index": idx})
}

// --- helpers -----------------------------------------------------------------

// persistAPIKeyControlsLocked saves a CRUD mutation without consuming the
// endpoint-specific response body. The caller must hold h.mu.
func (h *Handler) persistAPIKeyControlsLocked(c *gin.Context) bool {
	if err := config.SaveConfigPreserveComments(h.configFilePath, h.cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save config: %v", err)})
		return false
	}
	snapshot := h.reloadSnapshotConfigLocked()
	var requestContext context.Context
	if c != nil && c.Request != nil {
		requestContext = c.Request.Context()
	}
	h.reloadConfigAfterManagementSaveAsync(requestContext, snapshot)
	return true
}

func buildAPIKeyControlResponse(cfg *config.Config, idx int, stats *usage.RequestStatistics, includeUsage bool, recentPerModel int, maskKey bool) apiKeyControlResponse {
	entry := &cfg.APIKeyControls[idx]
	out := apiKeyControlResponseFromConfig(entry, idx, maskKey)
	if !includeUsage || stats == nil {
		return out
	}

	snapshot := stats.Snapshot()
	apiSnap, ok := snapshot.APIs[entry.APIKey]
	if !ok {
		return out
	}

	usageOut := &apiKeyControlUsage{
		TotalRequests:     apiSnap.TotalRequests,
		FailureCount:      0,
		TotalTokens:       apiSnap.TotalTokens,
		TotalInputTokens:  apiSnap.TotalInputTokens,
		TotalCachedTokens: apiSnap.TotalCachedTokens,
		AvgLatencyMs:      apiSnap.AverageLatencyMs,
	}
	// Cost across all priced models.
	var totalCost float64
	models := make([]apiKeyControlModelBreakdown, 0, len(apiSnap.Models))
	for modelName, modelStats := range apiSnap.Models {
		price, source, matched := pricing.LookupWithSource(modelName)
		bd := apiKeyControlModelBreakdown{
			Name:            modelName,
			Requests:        modelStats.TotalRequests,
			Tokens:          modelStats.TotalTokens,
			CachedTokens:    modelStats.TotalCachedTokens,
			AvgLatencyMs:    modelStats.AverageLatencyMs,
			PriceSource:     source,
			PriceMatched:    matched,
			PriceInputPerM:  price.Input,
			PriceCachedPerM: price.CachedInput,
			PriceOutputPerM: price.Output,
		}
		if source != pricing.SourceUnknown {
			for _, detail := range modelStats.Details {
				bd.UsedUSD += pricing.EstimateRequestCostUSD(
					price,
					detail.Tokens.InputTokens,
					detail.Tokens.CachedTokens,
					detail.Tokens.OutputTokens,
					detail.Tokens.InputTokens+detail.Tokens.OutputTokens,
				)
			}
		}
		totalCost += bd.UsedUSD
		models = append(models, bd)

		if recentPerModel > 0 {
			details := modelStats.Details
			start := len(details) - recentPerModel
			if start < 0 {
				start = 0
			}
			for i := start; i < len(details); i++ {
				d := details[i]
				total := d.Tokens.InputTokens + d.Tokens.OutputTokens
				usageOut.Recent = append(usageOut.Recent, apiKeyControlRecentRequest{
					Timestamp:    d.Timestamp,
					Model:        modelName,
					Source:       d.Source,
					InputTokens:  d.Tokens.InputTokens,
					CachedTokens: d.Tokens.CachedTokens,
					OutputTokens: d.Tokens.OutputTokens,
					TotalTokens:  total,
					LatencyMs:    d.LatencyMs,
					Failed:       d.Failed,
					CostUSD: pricing.EstimateRequestCostUSD(
						price,
						d.Tokens.InputTokens,
						d.Tokens.CachedTokens,
						d.Tokens.OutputTokens,
						total,
					),
				})
			}
		}
	}
	sort.SliceStable(models, func(i, j int) bool { return models[i].UsedUSD > models[j].UsedUSD })
	usageOut.Models = models
	usageOut.UsedUSD = totalCost
	if entry.MaxCostUSD > 0 {
		usageOut.RemainingUSD = entry.MaxCostUSD - totalCost
		if usageOut.RemainingUSD < 0 {
			usageOut.RemainingUSD = 0
		}
		usageOut.UsedPercent = totalCost / entry.MaxCostUSD * 100
		usageOut.Exhausted = totalCost >= entry.MaxCostUSD
	}
	if len(usageOut.Recent) > 0 {
		sort.SliceStable(usageOut.Recent, func(i, j int) bool {
			return usageOut.Recent[i].Timestamp.After(usageOut.Recent[j].Timestamp)
		})
	}
	out.Usage = usageOut
	return out
}

func apiKeyControlResponseFromConfig(entry *config.APIKeyControl, idx int, maskKey bool) apiKeyControlResponse {
	presetOn := false
	presetExcerpt := ""
	if entry.PresetPrompt != nil {
		presetOn = entry.PresetPrompt.Enabled
		presetExcerpt = truncateForExcerpt(entry.PresetPrompt.Prompt, 120)
	}
	out := apiKeyControlResponse{
		Index:               idx,
		Name:                entry.Name,
		APIKey:              entry.APIKey,
		APIKeyMask:          maskAPIKey(entry.APIKey),
		APIKeyHash:          hashAPIKey(entry.APIKey),
		Enabled:             entry.Enabled,
		Unlimited:           entry.Unlimited,
		Models:              cloneStrings(entry.Models),
		ExcludedModels:      cloneStrings(entry.ExcludedModels),
		MaxRequests:         entry.MaxRequests,
		MaxInputTokens:      entry.MaxInputTokens,
		MaxTotalTokens:      entry.MaxTotalTokens,
		MaxCostUSD:          entry.MaxCostUSD,
		PresetPromptEnabled: presetOn,
		PresetPromptExcerpt: presetExcerpt,
	}
	if maskKey {
		out.APIKey = ""
	}
	return out
}

func resolveAPIKeyControlIndex(cfg *config.Config, name *string, index *int) (int, error) {
	if cfg == nil {
		return -1, fmt.Errorf("config unavailable")
	}
	if index != nil {
		if *index < 0 || *index >= len(cfg.APIKeyControls) {
			return -1, fmt.Errorf("index out of range")
		}
		return *index, nil
	}
	if name != nil && strings.TrimSpace(*name) != "" {
		target := strings.TrimSpace(*name)
		for i := range cfg.APIKeyControls {
			if cfg.APIKeyControls[i].Name == target {
				return i, nil
			}
		}
		return -1, fmt.Errorf("no api-key-controls entry named %q", target)
	}
	return -1, fmt.Errorf("must specify name or index")
}

func generateAPIKey(nickname string) (string, error) {
	buf := make([]byte, 36)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	tail := strings.NewReplacer("=", "", "+", "", "/", "").Replace(base64.StdEncoding.EncodeToString(buf))
	// keep it URL-safe and cap length so the config file stays readable
	if len(tail) > 48 {
		tail = tail[:48]
	}
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		return "sk-ppap-" + tail, nil
	}
	nickname = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, nickname)
	if nickname == "" {
		return "sk-ppap-" + tail, nil
	}
	return "sk-ppap-" + nickname + "-" + tail, nil
}

func maskAPIKey(key string) string {
	if len(key) <= 12 {
		return key
	}
	return key[:8] + "…" + key[len(key)-4:]
}

func hashAPIKey(key string) string {
	if key == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:12]
}

func truncateForExcerpt(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	return append([]string(nil), in...)
}

func containsString(s []string, v string) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}
	return false
}

func removeString(s *[]string, v string) {
	if s == nil {
		return
	}
	filtered := make([]string, 0, len(*s))
	for _, e := range *s {
		if e != v {
			filtered = append(filtered, e)
		}
	}
	*s = filtered
}

func replaceString(s *[]string, old, new string) {
	if s == nil {
		return
	}
	for i := range *s {
		if (*s)[i] == old {
			(*s)[i] = new
		}
	}
}

func parseIntQuery(c *gin.Context, key string, def, min, max int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(raw, "%d", &n); err != nil {
		return def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

// json.RawMessage unused — the imports linter needs a mention here so the
// file survives future goimports passes.
var _ = json.RawMessage(nil)
