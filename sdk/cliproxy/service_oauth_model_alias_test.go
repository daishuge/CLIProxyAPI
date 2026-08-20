package cliproxy

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

func TestApplyOAuthModelAlias_Rename(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {
				{Name: "gpt-5", Alias: "g5", DisplayName: "Configured GPT Five"},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "gpt-5", Name: "models/gpt-5", DisplayName: "Upstream GPT Five"},
	}

	out := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	if len(out) != 1 {
		t.Fatalf("expected 1 model, got %d", len(out))
	}
	if out[0].ID != "g5" {
		t.Fatalf("expected model id %q, got %q", "g5", out[0].ID)
	}
	if out[0].Name != "models/g5" {
		t.Fatalf("expected model name %q, got %q", "models/g5", out[0].Name)
	}
	if out[0].DisplayName != "Configured GPT Five" {
		t.Fatalf("expected display name %q, got %q", "Configured GPT Five", out[0].DisplayName)
	}
}

func TestApplyAutomaticThinkingAliases(t *testing.T) {
	models := []*ModelInfo{
		{
			ID:       "gpt-5.6-sol",
			Name:     "models/gpt-5.6-sol",
			Thinking: &registry.ThinkingSupport{Levels: []string{"low", "medium", "high", "xhigh"}},
		},
	}

	out := applyAutomaticThinkingAliases(models, nil)
	ids := make(map[string]*ModelInfo, len(out))
	for _, model := range out {
		ids[model.ID] = model
	}
	for _, id := range []string{
		"gpt-5.6-sol",
		"gpt-5.6-sol-low",
		"gpt-5.6-sol-medium",
		"gpt-5.6-sol-high",
		"gpt-5.6-sol-xhigh",
	} {
		if ids[id] == nil {
			t.Fatalf("missing model alias %q in %#v", id, ids)
		}
	}
	if got := ids["gpt-5.6-sol-high"].ThinkingAliasBase; got != "gpt-5.6-sol" {
		t.Fatalf("ThinkingAliasBase = %q", got)
	}
	if got := ids["gpt-5.6-sol-high"].Name; got != "models/gpt-5.6-sol-high" {
		t.Fatalf("generated model name = %q", got)
	}
}

func TestApplyAutomaticThinkingAliases_ExplicitAliasWins(t *testing.T) {
	models := []*ModelInfo{
		{ID: "base", Thinking: &registry.ThinkingSupport{Levels: []string{"high"}}},
		{ID: "base-high"},
	}

	out := applyAutomaticThinkingAliases(models, nil)
	count := 0
	for _, model := range out {
		if model.ID == "base-high" {
			count++
			if model.ThinkingAliasBase != "" {
				t.Fatalf("explicit alias should keep priority, got generated marker %q", model.ThinkingAliasBase)
			}
		}
	}
	if count != 1 {
		t.Fatalf("base-high count = %d, want 1", count)
	}
}

func TestApplyAutomaticThinkingAliases_ExcludedAliasStaysHidden(t *testing.T) {
	models := []*ModelInfo{
		{ID: "base", Thinking: &registry.ThinkingSupport{Levels: []string{"low", "high"}}},
	}

	out := applyAutomaticThinkingAliases(models, []string{"*-high"})
	ids := make(map[string]bool, len(out))
	for _, model := range out {
		ids[model.ID] = true
	}
	if !ids["base"] || !ids["base-low"] {
		t.Fatalf("expected base and low alias, got %#v", ids)
	}
	if ids["base-high"] {
		t.Fatalf("excluded generated alias was registered")
	}
}

func TestApplyOAuthModelAlias_ForkAddsAlias(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {
				{Name: "gpt-5", Alias: "g5", Fork: true, DisplayName: "Configured GPT Five"},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "gpt-5", Name: "models/gpt-5", DisplayName: "Upstream GPT Five"},
	}

	out := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	if len(out) != 2 {
		t.Fatalf("expected 2 models, got %d", len(out))
	}
	if out[0].ID != "gpt-5" {
		t.Fatalf("expected first model id %q, got %q", "gpt-5", out[0].ID)
	}
	if out[1].ID != "g5" {
		t.Fatalf("expected second model id %q, got %q", "g5", out[1].ID)
	}
	if out[1].Name != "models/g5" {
		t.Fatalf("expected forked model name %q, got %q", "models/g5", out[1].Name)
	}
	if out[0].DisplayName != "Upstream GPT Five" {
		t.Fatalf("expected original display name %q, got %q", "Upstream GPT Five", out[0].DisplayName)
	}
	if out[1].DisplayName != "Configured GPT Five" {
		t.Fatalf("expected alias display name %q, got %q", "Configured GPT Five", out[1].DisplayName)
	}
}

func TestApplyOAuthModelAlias_PreservesUpstreamDisplayNameByDefault(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {
				{Name: "gpt-5", Alias: "g5"},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "gpt-5", DisplayName: "Upstream GPT Five"},
	}

	out := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	if len(out) != 1 {
		t.Fatalf("expected 1 model, got %d", len(out))
	}
	if out[0].DisplayName != "Upstream GPT Five" {
		t.Fatalf("expected upstream display name %q, got %q", "Upstream GPT Five", out[0].DisplayName)
	}
}

func TestApplyOAuthModelAlias_ForkAddsMultipleAliases(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {
				{Name: "gpt-5", Alias: "g5", Fork: true},
				{Name: "gpt-5", Alias: "g5-2", Fork: true},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "gpt-5", Name: "models/gpt-5"},
	}

	out := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	if len(out) != 3 {
		t.Fatalf("expected 3 models, got %d", len(out))
	}
	if out[0].ID != "gpt-5" {
		t.Fatalf("expected first model id %q, got %q", "gpt-5", out[0].ID)
	}
	if out[1].ID != "g5" {
		t.Fatalf("expected second model id %q, got %q", "g5", out[1].ID)
	}
	if out[1].Name != "models/g5" {
		t.Fatalf("expected forked model name %q, got %q", "models/g5", out[1].Name)
	}
	if out[2].ID != "g5-2" {
		t.Fatalf("expected third model id %q, got %q", "g5-2", out[2].ID)
	}
	if out[2].Name != "models/g5-2" {
		t.Fatalf("expected forked model name %q, got %q", "models/g5-2", out[2].Name)
	}
}

func TestApplyOAuthModelAlias_PluginProvider(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"sample-provider": {
				{Name: "sample-model-latest", Alias: "sample-latest"},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "sample-model-latest", Name: "models/sample-model-latest"},
	}

	out := applyOAuthModelAlias(cfg, "sample-provider", "oauth", models)
	if len(out) != 1 {
		t.Fatalf("expected 1 model, got %d", len(out))
	}
	if out[0].ID != "sample-latest" {
		t.Fatalf("expected plugin alias id %q, got %q", "sample-latest", out[0].ID)
	}
	if out[0].Name != "models/sample-latest" {
		t.Fatalf("expected plugin alias name %q, got %q", "models/sample-latest", out[0].Name)
	}
}

func TestApplyOAuthModelAlias_PluginProviderSkipsAPIKey(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"sample-provider": {
				{Name: "sample-model-latest", Alias: "sample-latest"},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "sample-model-latest", Name: "models/sample-model-latest"},
	}

	out := applyOAuthModelAlias(cfg, "sample-provider", "api_key", models)
	if len(out) != 1 || out[0].ID != "sample-model-latest" {
		t.Fatalf("expected API key plugin model to remain unchanged, got %#v", out)
	}
}

func TestApplyOAuthModelAlias_PerAuthAlias(t *testing.T) {
	models := []*ModelInfo{
		{ID: "gpt-5.3-codex-spark", Name: "models/gpt-5.3-codex-spark"},
	}
	attributes := map[string]string{
		"model_aliases": `[{"name":"gpt-5.3-codex-spark","alias":"gpt-5.5","display-name":"Configured GPT Five"}]`,
	}

	out := applyOAuthModelAliasForAuth(nil, "codex", "oauth", attributes, models)
	if len(out) != 1 {
		t.Fatalf("expected 1 model, got %d", len(out))
	}
	if out[0].ID != "gpt-5.5" {
		t.Fatalf("expected per-auth alias id %q, got %q", "gpt-5.5", out[0].ID)
	}
	if out[0].Name != "models/gpt-5.5" {
		t.Fatalf("expected per-auth alias name %q, got %q", "models/gpt-5.5", out[0].Name)
	}
	if out[0].DisplayName != "Configured GPT Five" {
		t.Fatalf("expected per-auth display name %q, got %q", "Configured GPT Five", out[0].DisplayName)
	}
}

func TestApplyOAuthModelAlias_ForkAddsFixedThinkingAlias(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {
				{Name: "gpt-5.3-codex-spark-high", Alias: "spark-fast", Fork: true},
			},
		},
	}
	models := []*ModelInfo{
		{
			ID:       "gpt-5.3-codex-spark",
			Name:     "models/gpt-5.3-codex-spark",
			Thinking: &registry.ThinkingSupport{Levels: []string{"low", "medium", "high", "xhigh"}},
		},
	}

	aliased := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	ids := make(map[string]*ModelInfo, len(aliased))
	for _, model := range aliased {
		ids[model.ID] = model
	}
	// The configured "gpt-5.3-codex-spark-high" name is mapped against the base
	// "gpt-5.3-codex-spark" model, so the alias is registered as a fixed-thinking fork.
	if ids["spark-fast"] == nil {
		t.Fatalf("missing fixed thinking alias in %#v", ids)
	}
	// A fixed-thinking alias is pinned to a single level, so it must not carry the
	// base model's thinking metadata (which would generate further level aliases).
	if ids["spark-fast"].Thinking != nil {
		t.Fatalf("fixed thinking alias should not retain thinking metadata")
	}
	// fork: true keeps the original base model alongside the alias.
	if ids["gpt-5.3-codex-spark"] == nil {
		t.Fatalf("forked base model should be preserved in %#v", ids)
	}

	out := applyAutomaticThinkingAliases(aliased, nil)
	ids = make(map[string]*ModelInfo, len(out))
	for _, model := range out {
		ids[model.ID] = model
	}
	if ids["spark-fast-low"] != nil {
		t.Fatalf("fixed thinking alias generated a misleading level alias")
	}
	if ids["gpt-5.3-codex-spark-high"] == nil {
		t.Fatalf("missing automatic base thinking alias in %#v", ids)
	}
}

func TestApplyOAuthModelAlias_FixedThinkingAliasRequiresSupportedBase(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {
				{Name: "plain-model-high", Alias: "plain-fast", Fork: true},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "plain-model", Name: "models/plain-model"},
	}

	out := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	for _, model := range out {
		if model.ID == "plain-fast" {
			t.Fatalf("non-thinking base model should not get fixed thinking alias")
		}
	}
}
