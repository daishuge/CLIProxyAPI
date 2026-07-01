package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigOptional_PerAuthDisableCooling(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	configYAML := []byte(`
gemini-api-key:
  - api-key: "gemini-key"
    disable-cooling: true
codex-api-key:
  - api-key: "codex-key"
    base-url: "https://codex.example.com"
    disable-cooling: true
claude-api-key:
  - api-key: "claude-key"
    disable-cooling: true
openai-compatibility:
  - name: "compat"
    base-url: "https://compat.example.com/v1"
    disable-cooling: true
    api-key-entries:
      - api-key: "compat-key"
`)
	if err := os.WriteFile(configPath, configYAML, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfigOptional(configPath, false)
	if err != nil {
		t.Fatalf("LoadConfigOptional() error = %v", err)
	}

	if len(cfg.GeminiKey) != 1 || !cfg.GeminiKey[0].DisableCooling {
		t.Fatalf("gemini disable-cooling was not parsed: %+v", cfg.GeminiKey)
	}
	if len(cfg.CodexKey) != 1 || !cfg.CodexKey[0].DisableCooling {
		t.Fatalf("codex disable-cooling was not parsed: %+v", cfg.CodexKey)
	}
	if len(cfg.ClaudeKey) != 1 || !cfg.ClaudeKey[0].DisableCooling {
		t.Fatalf("claude disable-cooling was not parsed: %+v", cfg.ClaudeKey)
	}
	if len(cfg.OpenAICompatibility) != 1 || !cfg.OpenAICompatibility[0].DisableCooling {
		t.Fatalf("openai compatibility disable-cooling was not parsed: %+v", cfg.OpenAICompatibility)
	}
}
