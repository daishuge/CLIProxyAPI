package management

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

func TestCustomUpstreamsAPIUsesOpenAICompatibilityPool(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	h := &Handler{
		cfg:            &config.Config{},
		configFilePath: writeTestConfigFile(t),
	}

	body := []byte(`[
		{
			"name":"opencode-go",
			"base-url":"https://api.example.com/v1",
			"api-key-entries":[{"api-key":"sk-test-1"}],
			"models":[{"name":"gpt-5-codex","alias":"opencode-go/gpt-5-codex"}]
		},
		{
			"name":"backup-upstream",
			"base-url":"https://backup.example.com/v1",
			"api-key-entries":[{"api-key":"sk-test-2"}],
			"models":[{"name":"gpt-5.5","alias":"backup/gpt-5.5"}]
		}
	]`)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/v0/management/custom-upstreams", bytes.NewReader(body))

	h.PutCustomUpstreams(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := len(h.cfg.OpenAICompatibility); got != 2 {
		t.Fatalf("OpenAICompatibility len = %d, want 2", got)
	}
	if got := h.cfg.OpenAICompatibility[0].Name; got != "opencode-go" {
		t.Fatalf("first upstream name = %q, want %q", got, "opencode-go")
	}

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/custom-upstreams", nil)

	h.GetCustomUpstreams(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload struct {
		CustomUpstreams []config.OpenAICompatibility `json:"custom-upstreams"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.CustomUpstreams) != 2 {
		t.Fatalf("custom-upstreams len = %d, want 2", len(payload.CustomUpstreams))
	}
	if strings.Contains(rec.Body.String(), "openai-compatibility") {
		t.Fatalf("custom-upstreams response leaked legacy key name: %s", rec.Body.String())
	}
}

func TestCustomUpstreamsPutAcceptsWrappedPayload(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	h := &Handler{
		cfg:            &config.Config{},
		configFilePath: writeTestConfigFile(t),
	}

	body := []byte(`{"custom-upstreams":[{"name":"opencode-go","base-url":"https://api.example.com/v1","api-key-entries":[{"api-key":"sk-test"}]}]}`)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/v0/management/custom-upstreams", bytes.NewReader(body))

	h.PutCustomUpstreams(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := len(h.cfg.OpenAICompatibility); got != 1 {
		t.Fatalf("OpenAICompatibility len = %d, want 1", got)
	}
	if got := h.cfg.OpenAICompatibility[0].Name; got != "opencode-go" {
		t.Fatalf("upstream name = %q, want opencode-go", got)
	}
}
