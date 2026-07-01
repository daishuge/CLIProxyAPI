package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRepositoryExampleConfigsParseAndPreservePPAPDefaults(t *testing.T) {
	repoRoot := repositoryRoot(t)

	tests := []struct {
		name                string
		file                string
		wantWebsocketAuth   bool
		wantPanelRepo       string
		wantUsageStatsPath  string
		wantConversationDir string
	}{
		{
			name:                "default example",
			file:                "config.example.yaml",
			wantWebsocketAuth:   true,
			wantPanelRepo:       DefaultPanelGitHubRepository,
			wantConversationDir: DefaultConversationLogDir,
		},
		{
			name:                "docker example",
			file:                "config.docker.example.yaml",
			wantWebsocketAuth:   true,
			wantPanelRepo:       DefaultPanelGitHubRepository,
			wantUsageStatsPath:  "/CLIProxyAPI/data/usage-statistics.json",
			wantConversationDir: "/CLIProxyAPI/data/conversation-logs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := copyExampleConfigToTemp(t, filepath.Join(repoRoot, tt.file))
			cfg, err := LoadConfigOptional(configPath, false)
			if err != nil {
				t.Fatalf("LoadConfigOptional(%s) error: %v", tt.file, err)
			}
			if cfg.WebsocketAuth != tt.wantWebsocketAuth {
				t.Fatalf("ws-auth = %v, want %v", cfg.WebsocketAuth, tt.wantWebsocketAuth)
			}
			if cfg.RemoteManagement.PanelGitHubRepository != tt.wantPanelRepo {
				t.Fatalf("panel repo = %q, want %q", cfg.RemoteManagement.PanelGitHubRepository, tt.wantPanelRepo)
			}
			if tt.wantUsageStatsPath != "" && cfg.UsageStatisticsPath != tt.wantUsageStatsPath {
				t.Fatalf("usage statistics path = %q, want %q", cfg.UsageStatisticsPath, tt.wantUsageStatsPath)
			}
			if cfg.ConversationLog.Enabled {
				t.Fatal("conversation-log.enabled = true, want false in example configs")
			}
			if cfg.ConversationLog.Directory != tt.wantConversationDir {
				t.Fatalf("conversation-log.directory = %q, want %q", cfg.ConversationLog.Directory, tt.wantConversationDir)
			}
			if cfg.ConversationLog.MaxFileSizeMB != DefaultConversationLogFileMB {
				t.Fatalf("conversation-log.max-file-size-mb = %d, want %d", cfg.ConversationLog.MaxFileSizeMB, DefaultConversationLogFileMB)
			}
			if cfg.ConversationLog.MaxTotalSizeMB != DefaultConversationLogTotalMB {
				t.Fatalf("conversation-log.max-total-size-mb = %d, want %d", cfg.ConversationLog.MaxTotalSizeMB, DefaultConversationLogTotalMB)
			}
			if cfg.ConversationLog.MaxEntryBytes != DefaultConversationLogEntryBytes {
				t.Fatalf("conversation-log.max-entry-bytes = %d, want %d", cfg.ConversationLog.MaxEntryBytes, DefaultConversationLogEntryBytes)
			}
			if cfg.PresetPrompt.Enabled {
				t.Fatal("preset-prompt.enabled = true, want false in example configs")
			}
			if cfg.PresetPrompt.Prompt != "" {
				t.Fatalf("preset-prompt.prompt = %q, want empty in example configs", cfg.PresetPrompt.Prompt)
			}
			if cfg.PresetPrompt.MaxBytes != DefaultPresetPromptMaxBytes {
				t.Fatalf("preset-prompt.max-bytes = %d, want %d", cfg.PresetPrompt.MaxBytes, DefaultPresetPromptMaxBytes)
			}
		})
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func copyExampleConfigToTemp(t *testing.T, source string) string {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read example config: %v", err)
	}
	target := filepath.Join(t.TempDir(), filepath.Base(source))
	if err := os.WriteFile(target, data, 0o600); err != nil {
		t.Fatalf("write temp example config: %v", err)
	}
	return target
}
