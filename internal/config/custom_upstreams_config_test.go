package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigReadsCustomUpstreamsAsOpenAICompatibility(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte(`custom-upstreams:
  - name: opencode-go
    base-url: https://api.example.com/v1
    api-key-entries:
      - api-key: sk-test
    models:
      - name: gpt-5-codex
        alias: opencode-go/gpt-5-codex
  - name: backup
    base-url: https://backup.example.com/v1
    api-key-entries:
      - api-key: sk-backup
    models:
      - name: gpt-5.5
        alias: backup/gpt-5.5
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if got := len(cfg.OpenAICompatibility); got != 2 {
		t.Fatalf("OpenAICompatibility len = %d, want 2", got)
	}
	if got := cfg.OpenAICompatibility[0].Name; got != "opencode-go" {
		t.Fatalf("first upstream name = %q, want %q", got, "opencode-go")
	}
	if got := cfg.OpenAICompatibility[0].Models[0].Alias; got != "opencode-go/gpt-5-codex" {
		t.Fatalf("first alias = %q, want %q", got, "opencode-go/gpt-5-codex")
	}
}

func TestLoadConfigMergesCustomUpstreamsWithOpenAICompatibility(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte(`openai-compatibility:
  - name: opencode-go
    base-url: https://old.example.com/v1
    api-key-entries:
      - api-key: sk-old
    models:
      - name: old-model
        alias: old/model
  - name: keep-existing
    base-url: https://keep.example.com/v1
    api-key-entries:
      - api-key: sk-keep
    models:
      - name: keep-model
        alias: keep/model
custom-upstreams:
  - name: opencode-go
    base-url: https://new.example.com/v1
    api-key-entries:
      - api-key: sk-new
    models:
      - name: gpt-5-codex
        alias: opencode-go/gpt-5-codex
  - name: backup
    base-url: https://backup.example.com/v1
    api-key-entries:
      - api-key: sk-backup
    models:
      - name: gpt-5.5
        alias: backup/gpt-5.5
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if got := len(cfg.OpenAICompatibility); got != 3 {
		t.Fatalf("OpenAICompatibility len = %d, want 3: %+v", got, cfg.OpenAICompatibility)
	}
	if got := cfg.OpenAICompatibility[0].BaseURL; got != "https://new.example.com/v1" {
		t.Fatalf("merged opencode-go base-url = %q, want new URL", got)
	}
	if got := cfg.OpenAICompatibility[1].Name; got != "keep-existing" {
		t.Fatalf("second upstream = %q, want keep-existing", got)
	}
	if got := cfg.OpenAICompatibility[2].Name; got != "backup" {
		t.Fatalf("third upstream = %q, want backup", got)
	}
}

func TestSaveConfigPreserveCommentsKeepsCustomUpstreamsKey(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte(`custom-upstreams:
  - name: opencode-go
    base-url: https://old.example.com/v1
    api-key-entries:
      - api-key: sk-old
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	cfg.OpenAICompatibility[0].BaseURL = "https://new.example.com/v1"

	if err := SaveConfigPreserveComments(configPath, cfg); err != nil {
		t.Fatalf("SaveConfigPreserveComments error: %v", err)
	}

	out, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	content := string(out)
	if !strings.Contains(content, "custom-upstreams:") {
		t.Fatalf("saved config missing custom-upstreams key:\n%s", content)
	}
	if strings.Contains(content, "openai-compatibility:") {
		t.Fatalf("saved config unexpectedly wrote legacy key:\n%s", content)
	}
	if !strings.Contains(content, "https://new.example.com/v1") {
		t.Fatalf("saved config missing updated base-url:\n%s", content)
	}
}
