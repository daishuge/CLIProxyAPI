package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	proxyconfig "github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

// setUserAPIKey installs the resolved principal the way the access middleware
// does, so the per-key controls middleware can read it.
func setUserAPIKey(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userApiKey", key)
		c.Next()
	}
}

func TestAPIKeyControl_AllowsModelAndPreservesJSONBody(t *testing.T) {
	server := newTestServer(t)
	server.cfg.APIKeyControls = []proxyconfig.APIKeyControl{
		{APIKey: "test-key", Models: []string{"allowed-*"}},
	}
	server.engine.POST("/test/api-key-control", setUserAPIKey("test-key"), server.apiKeyControlsMiddleware(), func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil {
			t.Fatalf("handler failed to read body: %v", err)
		}
		if !strings.Contains(string(body), `"model":"allowed-model"`) {
			t.Fatalf("request body was not preserved: %s", string(body))
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test/api-key-control", strings.NewReader(`{"model":"allowed-model","messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.engine.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestAPIKeyControl_BlocksDisallowedModel(t *testing.T) {
	server := newTestServer(t)
	server.cfg.APIKeyControls = []proxyconfig.APIKeyControl{
		{APIKey: "test-key", Models: []string{"allowed-*"}},
	}
	server.engine.POST("/test/api-key-control-denied", setUserAPIKey("test-key"), server.apiKeyControlsMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test/api-key-control-denied", strings.NewReader(`{"model":"blocked-model","messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.engine.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "model_not_allowed") {
		t.Fatalf("response missing model_not_allowed code: %s", rr.Body.String())
	}
}

func TestAPIKeyControl_BlocksDisabledKey(t *testing.T) {
	server := newTestServer(t)
	disabled := false
	server.cfg.APIKeyControls = []proxyconfig.APIKeyControl{
		{APIKey: "test-key", Enabled: &disabled},
	}
	server.engine.POST("/test/api-key-control-disabled", setUserAPIKey("test-key"), server.apiKeyControlsMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test/api-key-control-disabled", strings.NewReader(`{"model":"any-model"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.engine.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "api_key_disabled") {
		t.Fatalf("response missing api_key_disabled code: %s", rr.Body.String())
	}
}

func TestAPIKeyModelAllowed_ExcludedTakesPrecedence(t *testing.T) {
	control := &proxyconfig.APIKeyControl{
		Models:         []string{"*"},
		ExcludedModels: []string{"gpt-5-pro"},
	}
	if apiKeyModelAllowed(control, "gpt-5-pro") {
		t.Fatal("apiKeyModelAllowed() = true for excluded model, want false")
	}
	if !apiKeyModelAllowed(control, "gpt-5-mini") {
		t.Fatal("apiKeyModelAllowed() = false for allowed model, want true")
	}
}

func TestModelPatternMatches(t *testing.T) {
	cases := []struct {
		model   string
		pattern string
		want    bool
	}{
		{"gpt-5.3-codex-spark-high", "gpt-5.3-codex-spark*", true},
		{"gpt-5.5-codex", "gpt-5.3-codex-spark*", false},
		{"anything", "*", true},
		{"gpt-5-mini", "gpt-5-mini", true},
		{"gpt-5-mini", "*mini", true},
		{"gpt-5-mini", "gpt-*-mini", true},
		{"gpt-5-nano", "gpt-*-mini", false},
	}
	for _, tc := range cases {
		if got := modelPatternMatches(tc.model, tc.pattern); got != tc.want {
			t.Fatalf("modelPatternMatches(%q, %q) = %v, want %v", tc.model, tc.pattern, got, tc.want)
		}
	}
}

func TestWithinAPIKeyBudget_NoLookupAllows(t *testing.T) {
	// Without a usage stats provider installed (pre-Step6), usage-derived
	// budgets must degrade to a permissive no-op.
	SetAPIKeyUsageStatsLookup(nil)
	control := &proxyconfig.APIKeyControl{APIKey: "k", MaxRequests: 1}
	if !withinAPIKeyBudget(control) {
		t.Fatal("withinAPIKeyBudget() = false without usage provider, want true")
	}
}

func TestWithinAPIKeyBudget_RequestCountEnforced(t *testing.T) {
	SetAPIKeyUsageStatsLookup(func(key string) (apiKeyUsageStats, bool) {
		if key != "k" {
			return apiKeyUsageStats{}, false
		}
		return apiKeyUsageStats{TotalRequests: 5}, true
	})
	t.Cleanup(func() { SetAPIKeyUsageStatsLookup(nil) })

	if withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "k", MaxRequests: 5}) {
		t.Fatal("withinAPIKeyBudget() = true at request limit, want false")
	}
	if !withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "k", MaxRequests: 6}) {
		t.Fatal("withinAPIKeyBudget() = false below request limit, want true")
	}
	if !withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "k", MaxRequests: 5, Unlimited: true}) {
		t.Fatal("withinAPIKeyBudget() = false for unlimited control, want true")
	}
}

func TestWithinAPIKeyBudget_EstimatedCostEnforced(t *testing.T) {
	usageFor := func(input, cached, output, total int64) (apiKeyUsageStats, bool) {
		return apiKeyUsageStats{
			Models: map[string]apiKeyModelUsage{
				"gpt-5.5-low-fast": {
					Details: []apiKeyTokenStats{
						{InputTokens: input, CachedTokens: cached, OutputTokens: output, TotalTokens: total},
					},
				},
			},
		}, true
	}

	SetAPIKeyUsageStatsLookup(func(key string) (apiKeyUsageStats, bool) {
		return usageFor(100_000, 20_000, 100_000, 200_000)
	})
	t.Cleanup(func() { SetAPIKeyUsageStatsLookup(nil) })
	if !withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "cost-key", MaxCostUSD: 30}) {
		t.Fatal("withinAPIKeyBudget() = false below estimated cost limit, want true")
	}

	SetAPIKeyUsageStatsLookup(func(key string) (apiKeyUsageStats, bool) {
		return usageFor(1_000_000, 200_000, 1_000_000, 2_000_000)
	})
	if withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "cost-key", MaxCostUSD: 30}) {
		t.Fatal("withinAPIKeyBudget() = true at estimated cost limit, want false")
	}
}
