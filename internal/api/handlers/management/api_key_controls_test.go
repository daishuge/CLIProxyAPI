package management

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

func apiKeyControlsTestRouter(h *Handler) *gin.Engine {
	engine := gin.New()
	engine.GET("/api-key-controls", h.GetAPIKeyControls)
	engine.POST("/api-key-controls", h.PostAPIKeyControls)
	engine.PATCH("/api-key-controls", h.PatchAPIKeyControls)
	engine.DELETE("/api-key-controls", h.DeleteAPIKeyControls)
	return engine
}

func apiKeyControlsRequest(t *testing.T, engine http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestAPIKeyControlsCRUDPersistsEndpointSpecificResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("port: 8317\n"), 0o600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	cfg := &config.Config{}
	h := NewHandler(cfg, configPath, nil)
	engine := apiKeyControlsTestRouter(h)

	created := apiKeyControlsRequest(t, engine, http.MethodPost, "/api-key-controls", `{"name":"primary","models":["gpt-5.6"]}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body = %s", created.Code, created.Body.String())
	}
	var createBody struct {
		Created apiKeyControlResponse `json:"created"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("decode POST response: %v", err)
	}
	if !strings.HasPrefix(createBody.Created.APIKey, "sk-ppap-") {
		t.Fatalf("generated API key has unexpected prefix: %q", createBody.Created.APIKey)
	}
	generatedKey := createBody.Created.APIKey
	if len(cfg.APIKeyControls) != 1 || len(cfg.APIKeys) != 1 || cfg.APIKeys[0] != generatedKey {
		t.Fatalf("created key was not added to both config sections: %#v", cfg)
	}

	listed := apiKeyControlsRequest(t, engine, http.MethodGet, "/api-key-controls?mask_key=1", "")
	if listed.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body = %s", listed.Code, listed.Body.String())
	}
	var listBody struct {
		Controls []apiKeyControlResponse `json:"api-key-controls"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if len(listBody.Controls) != 1 || listBody.Controls[0].APIKey != "" || listBody.Controls[0].APIKeyMask == "" || listBody.Controls[0].APIKeyHash == "" {
		t.Fatalf("masked GET returned unexpected item: %#v", listBody.Controls)
	}

	patched := apiKeyControlsRequest(t, engine, http.MethodPatch, "/api-key-controls", `{"target_name":"primary","value":{"name":"production","max-requests":42,"excluded-models":["gpt-4*"]}}`)
	if patched.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, body = %s", patched.Code, patched.Body.String())
	}
	var patchBody struct {
		Updated apiKeyControlResponse `json:"updated"`
	}
	if err := json.Unmarshal(patched.Body.Bytes(), &patchBody); err != nil {
		t.Fatalf("decode PATCH response: %v", err)
	}
	if patchBody.Updated.Name != "production" || patchBody.Updated.MaxRequests != 42 {
		t.Fatalf("PATCH returned unexpected control: %#v", patchBody.Updated)
	}

	deleted := apiKeyControlsRequest(t, engine, http.MethodDelete, "/api-key-controls", `{"name":"production"}`)
	if deleted.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, body = %s", deleted.Code, deleted.Body.String())
	}
	if len(cfg.APIKeyControls) != 0 || len(cfg.APIKeys) != 0 {
		t.Fatalf("DELETE did not remove both config entries: %#v", cfg)
	}

	persisted, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("reload persisted config: %v", err)
	}
	if len(persisted.APIKeyControls) != 0 || len(persisted.APIKeys) != 0 {
		t.Fatalf("persisted config still contains deleted key: %#v", persisted)
	}
}

func TestAPIKeyControlsMutationRollbackOnPersistenceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	cfg.APIKeys = []string{"existing"}
	h := NewHandler(cfg, filepath.Join(t.TempDir(), "missing", "config.yaml"), nil)
	recorder := apiKeyControlsRequest(t, apiKeyControlsTestRouter(h), http.MethodPost, "/api-key-controls", `{"name":"broken","api-key":"existing"}`)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("POST status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if len(cfg.APIKeyControls) != 0 || len(cfg.APIKeys) != 1 || cfg.APIKeys[0] != "existing" {
		t.Fatalf("failed persistence left an in-memory mutation: %#v", cfg)
	}
}

func TestAPIKeyControlsPatchRejectsDuplicateNameWithoutMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("port: 8317\n"), 0o600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}
	cfg := &config.Config{}
	cfg.APIKeyControls = []config.APIKeyControl{
		{Name: "first", APIKey: "key-1"},
		{Name: "second", APIKey: "key-2"},
	}
	h := NewHandler(cfg, configPath, nil)
	recorder := apiKeyControlsRequest(t, apiKeyControlsTestRouter(h), http.MethodPatch, "/api-key-controls", `{"target_name":"first","value":{"name":"second"}}`)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("PATCH status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if cfg.APIKeyControls[0].Name != "first" || cfg.APIKeyControls[1].Name != "second" {
		t.Fatalf("duplicate-name rejection mutated controls: %#v", cfg.APIKeyControls)
	}
}
