package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigOptionalConversationLogDefaults(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("port: 8317\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfigOptional(configPath, false)
	if err != nil {
		t.Fatalf("LoadConfigOptional() error = %v", err)
	}

	assertDefaultConversationLogConfig(t, cfg.ConversationLog)
}

func TestLoadConfigOptionalConversationLogEnabledAndNormalized(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte(`conversation-log:
  enabled: true
  directory: " audit "
  max-file-size-mb: -1
  max-total-size-mb: 0
  max-entry-bytes: -5
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfigOptional(configPath, false)
	if err != nil {
		t.Fatalf("LoadConfigOptional() error = %v", err)
	}

	if !cfg.ConversationLog.Enabled {
		t.Fatal("conversation-log.enabled = false, want true")
	}
	if cfg.ConversationLog.Directory != "audit" {
		t.Fatalf("conversation-log.directory = %q, want %q", cfg.ConversationLog.Directory, "audit")
	}
	if cfg.ConversationLog.MaxFileSizeMB != DefaultConversationLogFileMB {
		t.Fatalf("conversation-log.max-file-size-mb = %d, want default %d", cfg.ConversationLog.MaxFileSizeMB, DefaultConversationLogFileMB)
	}
	if cfg.ConversationLog.MaxTotalSizeMB != DefaultConversationLogTotalMB {
		t.Fatalf("conversation-log.max-total-size-mb = %d, want default %d", cfg.ConversationLog.MaxTotalSizeMB, DefaultConversationLogTotalMB)
	}
	if cfg.ConversationLog.MaxEntryBytes != DefaultConversationLogEntryBytes {
		t.Fatalf("conversation-log.max-entry-bytes = %d, want default %d", cfg.ConversationLog.MaxEntryBytes, DefaultConversationLogEntryBytes)
	}
}

func assertDefaultConversationLogConfig(t *testing.T, cfg ConversationLogConfig) {
	t.Helper()
	if cfg.Enabled {
		t.Fatal("conversation-log.enabled = true, want false")
	}
	if cfg.Directory != DefaultConversationLogDir {
		t.Fatalf("conversation-log.directory = %q, want %q", cfg.Directory, DefaultConversationLogDir)
	}
	if cfg.MaxFileSizeMB != DefaultConversationLogFileMB {
		t.Fatalf("conversation-log.max-file-size-mb = %d, want %d", cfg.MaxFileSizeMB, DefaultConversationLogFileMB)
	}
	if cfg.MaxTotalSizeMB != DefaultConversationLogTotalMB {
		t.Fatalf("conversation-log.max-total-size-mb = %d, want %d", cfg.MaxTotalSizeMB, DefaultConversationLogTotalMB)
	}
	if cfg.MaxEntryBytes != DefaultConversationLogEntryBytes {
		t.Fatalf("conversation-log.max-entry-bytes = %d, want %d", cfg.MaxEntryBytes, DefaultConversationLogEntryBytes)
	}
}
