package openai

import (
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

// TestResponsesWebsocketProviderSet_HyphenThinkingSuffix verifies the /v1/responses
// WebSocket provider/auth resolution strips a hyphen thinking-level alias to its
// registered base model. Before the fix this path used parenthesis-only ParseSuffix,
// so "claude-opus-4-8-high" produced an empty provider set and a hyphen modelKey,
// and responsesWebsocketAuthMatchesModel rejected every auth (ClientSupportsModel
// missed) — the same class of failure fixed on the REST path.
func TestResponsesWebsocketProviderSet_HyphenThinkingSuffix(t *testing.T) {
	reg := registry.GetGlobalRegistry()
	const authID = "ws-hyphen-claude-auth"
	reg.UnregisterClient(authID)
	t.Cleanup(func() { reg.UnregisterClient(authID) })
	reg.RegisterClient(authID, "claude", []*registry.ModelInfo{
		{
			ID:       "claude-opus-4-8",
			Thinking: &registry.ThinkingSupport{Levels: []string{"low", "medium", "high", "xhigh", "max"}},
		},
	})

	providerSet, modelKey := responsesWebsocketProviderSetForModel("claude-opus-4-8-high")
	if modelKey != "claude-opus-4-8" {
		t.Fatalf("expected base modelKey claude-opus-4-8, got %q", modelKey)
	}
	if _, ok := providerSet["claude"]; !ok {
		t.Fatalf("expected claude in provider set, got %v", providerSet)
	}

	auth := &coreauth.Auth{
		ID:       authID,
		Provider: "claude",
		Status:   coreauth.StatusActive,
	}
	if !responsesWebsocketAuthMatchesModel(auth, providerSet, modelKey, reg, time.Now()) {
		t.Fatal("expected the claude auth to match claude-opus-4-8-high on the /v1/responses WebSocket path")
	}
}
