package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	DefaultConversationLogDir        = "conversation-logs"
	DefaultConversationLogFileMB     = 16
	DefaultConversationLogTotalMB    = 256
	DefaultConversationLogEntryBytes = 2 * 1024 * 1024
	DefaultPresetPromptMaxBytes      = 32 * 1024
	PresetPromptHardMaxBytes         = 256 * 1024
	customUpstreamsConfigKey         = "custom-upstreams"
	openAICompatibilityConfigKey     = "openai-compatibility"
)

// APIKeyControl defines optional model, prompt, and usage limits for one downstream API key.
// Budget values <= 0 are treated as unlimited. The Unlimited flag bypasses usage budgets,
// but model allow/deny rules and per-key prompt settings still apply.
type APIKeyControl struct {
	APIKey         string              `yaml:"api-key,omitempty" json:"api-key,omitempty"`
	Key            string              `yaml:"key,omitempty" json:"key,omitempty"`
	Name           string              `yaml:"name,omitempty" json:"name,omitempty"`
	Enabled        *bool               `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Unlimited      bool                `yaml:"unlimited,omitempty" json:"unlimited,omitempty"`
	Models         []string            `yaml:"models,omitempty" json:"models,omitempty"`
	ExcludedModels []string            `yaml:"excluded-models,omitempty" json:"excluded-models,omitempty"`
	MaxRequests    int64               `yaml:"max-requests,omitempty" json:"max-requests,omitempty"`
	MaxInputTokens int64               `yaml:"max-input-tokens,omitempty" json:"max-input-tokens,omitempty"`
	MaxTotalTokens int64               `yaml:"max-total-tokens,omitempty" json:"max-total-tokens,omitempty"`
	MaxCostUSD     float64             `yaml:"max-cost-usd,omitempty" json:"max-cost-usd,omitempty"`
	PresetPrompt   *PresetPromptConfig `yaml:"preset-prompt,omitempty" json:"preset-prompt,omitempty"`
}

// ConversationLogConfig controls full request/response conversation log storage.
// It is disabled by default because entries can contain sensitive user and model content.
type ConversationLogConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Directory      string `yaml:"directory,omitempty" json:"directory,omitempty"`
	MaxFileSizeMB  int    `yaml:"max-file-size-mb,omitempty" json:"max-file-size-mb,omitempty"`
	MaxTotalSizeMB int    `yaml:"max-total-size-mb,omitempty" json:"max-total-size-mb,omitempty"`
	MaxEntryBytes  int    `yaml:"max-entry-bytes,omitempty" json:"max-entry-bytes,omitempty"`
}

func DefaultConversationLogConfig() ConversationLogConfig {
	return ConversationLogConfig{
		Directory:      DefaultConversationLogDir,
		MaxFileSizeMB:  DefaultConversationLogFileMB,
		MaxTotalSizeMB: DefaultConversationLogTotalMB,
		MaxEntryBytes:  DefaultConversationLogEntryBytes,
	}
}

func (c *ConversationLogConfig) Normalize() {
	if c == nil {
		return
	}
	defaults := DefaultConversationLogConfig()
	c.Directory = strings.TrimSpace(c.Directory)
	if c.Directory == "" {
		c.Directory = defaults.Directory
	}
	if c.MaxFileSizeMB <= 0 {
		c.MaxFileSizeMB = defaults.MaxFileSizeMB
	}
	if c.MaxTotalSizeMB <= 0 {
		c.MaxTotalSizeMB = defaults.MaxTotalSizeMB
	}
	if c.MaxEntryBytes <= 0 {
		c.MaxEntryBytes = defaults.MaxEntryBytes
	}
}

// PresetPromptConfig controls optional prompt text inserted only into upstream requests.
type PresetPromptConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Prompt   string `yaml:"prompt,omitempty" json:"prompt,omitempty"`
	MaxBytes int    `yaml:"max-bytes,omitempty" json:"max-bytes,omitempty"`
}

func DefaultPresetPromptConfig() PresetPromptConfig {
	return PresetPromptConfig{MaxBytes: DefaultPresetPromptMaxBytes}
}

func (c *PresetPromptConfig) Normalize() {
	if c == nil {
		return
	}
	if c.MaxBytes <= 0 {
		c.MaxBytes = DefaultPresetPromptMaxBytes
	}
	if c.MaxBytes > PresetPromptHardMaxBytes {
		log.WithFields(log.Fields{"value": c.MaxBytes, "max": PresetPromptHardMaxBytes}).
			Warn("preset-prompt.max-bytes too large; clamping")
		c.MaxBytes = PresetPromptHardMaxBytes
	}
}

func (c PresetPromptConfig) Validate() error {
	if c.Enabled && strings.TrimSpace(c.Prompt) == "" {
		return errors.New("preset-prompt.prompt must be set when preset-prompt.enabled is true")
	}
	if promptBytes := len([]byte(c.Prompt)); promptBytes > c.MaxBytes {
		return fmt.Errorf("preset-prompt.prompt is too large: %d bytes exceeds preset-prompt.max-bytes %d", promptBytes, c.MaxBytes)
	}
	return nil
}

// UpstreamConcurrencyConfig configures provider-level upstream concurrency gates.
type UpstreamConcurrencyConfig struct {
	Default             int            `yaml:"default" json:"default"`
	Providers           map[string]int `yaml:"providers,omitempty" json:"providers,omitempty"`
	QueueTimeoutSeconds int            `yaml:"queue-timeout-seconds" json:"queue-timeout-seconds"`
}

func (c *UpstreamConcurrencyConfig) Normalize() {
	if c == nil {
		return
	}
	if c.Default < 0 {
		c.Default = 0
	}
	if c.QueueTimeoutSeconds < 0 {
		c.QueueTimeoutSeconds = 0
	}
	if len(c.Providers) == 0 {
		return
	}
	normalized := make(map[string]int, len(c.Providers))
	for key, limit := range c.Providers {
		provider := strings.ToLower(strings.TrimSpace(key))
		if provider == "" {
			continue
		}
		if limit < 0 {
			limit = 0
		}
		normalized[provider] = limit
	}
	c.Providers = normalized
}

func (c UpstreamConcurrencyConfig) LimitForProvider(provider string) int {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if limit, ok := c.Providers[provider]; ok {
		if limit < 0 {
			return 0
		}
		return limit
	}
	if c.Default < 0 {
		return 0
	}
	return c.Default
}

func (c UpstreamConcurrencyConfig) QueueTimeout() time.Duration {
	if c.QueueTimeoutSeconds > 0 {
		return time.Duration(c.QueueTimeoutSeconds) * time.Second
	}
	return 30 * time.Second
}
