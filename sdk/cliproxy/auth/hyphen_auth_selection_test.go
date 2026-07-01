package auth

import (
	"context"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
)

// TestCanonicalModelKey_StripsHyphenThinkingLevel verifies that the auth-layer
// model-key normalizer reduces a hyphen thinking-level alias to its registered
// base model. Before the fix it used the parenthesis-only ParseSuffix and left
// "claude-opus-4-8-high" unchanged, which made the registered base model lookups
// (ClientSupportsModel, ModelStates, scheduler shards) miss.
func TestCanonicalModelKey_StripsHyphenThinkingLevel(t *testing.T) {
	// canonicalModelKey consults the model registry to decide whether a trailing
	// hyphen segment is a real thinking level for the base model, so register a
	// client that advertises claude-opus-4-8 with the "high" level.
	reg := registry.GetGlobalRegistry()
	const clientID = "hyphen-canonical-claude"
	reg.UnregisterClient(clientID)
	t.Cleanup(func() { reg.UnregisterClient(clientID) })
	reg.RegisterClient(clientID, "claude", []*registry.ModelInfo{
		{
			ID:       "claude-opus-4-8",
			Thinking: &registry.ThinkingSupport{Levels: []string{"low", "medium", "high", "xhigh", "max"}},
		},
	})

	cases := []struct {
		in   string
		want string
	}{
		{"claude-opus-4-8-high", "claude-opus-4-8"},        // hyphen level -> base
		{"claude-opus-4-8(high)", "claude-opus-4-8"},       // paren level -> base (still works)
		{"claude-opus-4-8", "claude-opus-4-8"},             // base unchanged
		{"claude-opus-4-8-turbo", "claude-opus-4-8-turbo"}, // NOT a level -> preserved
	}
	for _, tc := range cases {
		if got := canonicalModelKey(tc.in); got != tc.want {
			t.Errorf("canonicalModelKey(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestAuthSelection_SelectsAuthForHyphenSuffixModel is the auth-SELECTION test the
// canary block requires: it proves an auth IS considered supporting a hyphen
// thinking-suffix model. authSupportsRouteModel is the exact site (conductor.go)
// that returned false for "claude-opus-4-8-high" (the registry only knows the
// base) and made the candidate list empty -> auth_not_found (503).
func TestAuthSelection_SelectsAuthForHyphenSuffixModel(t *testing.T) {
	reg := registry.GetGlobalRegistry()
	const authID = "hyphen-select-claude-auth"
	reg.UnregisterClient(authID)
	t.Cleanup(func() { reg.UnregisterClient(authID) })

	manager := NewManager(nil, nil, nil)

	auth := &Auth{
		ID:       authID,
		Provider: "claude",
		Status:   StatusActive,
	}
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("register auth: %v", err)
	}
	reg.RegisterClient(authID, "claude", []*registry.ModelInfo{
		{
			ID:       "claude-opus-4-8",
			Thinking: &registry.ThinkingSupport{Levels: []string{"low", "medium", "high", "xhigh", "max"}},
		},
	})
	manager.RefreshSchedulerEntry(authID)

	picked, ok := manager.GetByID(authID)
	if !ok || picked == nil {
		t.Fatalf("auth %q not present in manager", authID)
	}

	// The base model must be supported (sanity), and the hyphen alias must now be
	// treated as supported via the canonicalModelKey reduction inside
	// authSupportsRouteModel.
	if !manager.authSupportsRouteModel(reg, picked, "claude-opus-4-8") {
		t.Fatal("expected auth to support base model claude-opus-4-8")
	}
	if !manager.authSupportsRouteModel(reg, picked, "claude-opus-4-8-high") {
		t.Fatal("expected auth to support hyphen thinking-suffix model claude-opus-4-8-high (auth-selection path)")
	}
	if !manager.authSupportsRouteModel(reg, picked, "claude-opus-4-8(high)") {
		t.Fatal("expected auth to support parenthesized thinking-suffix model claude-opus-4-8(high)")
	}
	// A genuinely unrelated/unregistered model must still NOT be supported.
	if manager.authSupportsRouteModel(reg, picked, "gpt-5.4") {
		t.Fatal("did not expect claude auth to support an unrelated model gpt-5.4")
	}
}
