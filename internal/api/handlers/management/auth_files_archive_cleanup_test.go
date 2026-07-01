package management

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

func TestDownloadAuthFilesArchive_ReturnsJSONFilesOnly(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	authDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(authDir, "a.json"), []byte(`{"type":"codex"}`), 0o600); err != nil {
		t.Fatalf("write a.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "b.JSON"), []byte(`{"type":"gemini"}`), 0o600); err != nil {
		t.Fatalf("write b.JSON: %v", err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "notes.txt"), []byte("ignore"), 0o600); err != nil {
		t.Fatalf("write notes.txt: %v", err)
	}

	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/auth-files/export", nil)

	h.DownloadAuthFilesArchive(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	reader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	names := make(map[string]bool)
	for _, file := range reader.File {
		names[file.Name] = true
	}
	if !names["a.json"] || !names["b.JSON"] {
		t.Fatalf("expected json auth files in archive, got %#v", names)
	}
	if names["notes.txt"] {
		t.Fatalf("unexpected non-json file in archive")
	}
}

func TestCleanupDisabledAuthFiles_RemovesDisabledFilesOnly(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")
	gin.SetMode(gin.TestMode)

	authDir := t.TempDir()
	disabledName := "disabled.json"
	activeName := "active.json"
	disabledPath := filepath.Join(authDir, disabledName)
	activePath := filepath.Join(authDir, activeName)
	if err := os.WriteFile(disabledPath, []byte(`{"type":"codex"}`), 0o600); err != nil {
		t.Fatalf("write disabled file: %v", err)
	}
	if err := os.WriteFile(activePath, []byte(`{"type":"codex"}`), 0o600); err != nil {
		t.Fatalf("write active file: %v", err)
	}

	manager := coreauth.NewManager(nil, nil, nil)
	if _, err := manager.Register(context.Background(), &coreauth.Auth{
		ID:       disabledName,
		FileName: disabledName,
		Provider: "codex",
		Disabled: true,
		Status:   coreauth.StatusDisabled,
		Attributes: map[string]string{
			"path": disabledPath,
		},
	}); err != nil {
		t.Fatalf("register disabled auth: %v", err)
	}
	if _, err := manager.Register(context.Background(), &coreauth.Auth{
		ID:       activeName,
		FileName: activeName,
		Provider: "codex",
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"path": activePath,
		},
	}); err != nil {
		t.Fatalf("register active auth: %v", err)
	}

	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, manager)
	h.tokenStore = &memoryAuthStore{}
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/v0/management/auth-files/disabled", nil)

	h.CleanupDisabledAuthFiles(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(disabledPath); !os.IsNotExist(err) {
		t.Fatalf("expected disabled file removed, stat err: %v", err)
	}
	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("expected active file to remain: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := int(payload["deleted"].(float64)); got != 1 {
		t.Fatalf("deleted = %d, want 1", got)
	}
}
