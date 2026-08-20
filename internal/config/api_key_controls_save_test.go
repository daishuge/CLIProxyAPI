package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveConfigPreserveCommentsRemovesEmptyAPIKeyControls(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	initial := "# retained\nport: 8317\napi-key-controls:\n  - name: old\n    api-key: old-key\n"
	if err := os.WriteFile(configPath, []byte(initial), 0o600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.APIKeyControls = nil
	if err := SaveConfigPreserveComments(configPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	text := string(written)
	if strings.Contains(text, "api-key-controls:") {
		t.Fatalf("empty api-key-controls remained on disk:\n%s", text)
	}
	if !strings.Contains(text, "# retained") {
		t.Fatalf("unrelated comment was not preserved:\n%s", text)
	}
}
