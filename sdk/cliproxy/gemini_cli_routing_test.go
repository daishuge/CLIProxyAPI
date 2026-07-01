package cliproxy

import (
	"context"
	"testing"

	internalregistry "github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

// TestGeminiCLIRouting_OAuthAccountResolvesToGeminiCLI verifies that a
// type=gemini OAuth account (normalized to provider=gemini-cli by the synthesizer)
// routes gemini-2.5-pro to the gemini-cli provider, both when it is the only
// Google provider AND when a native gemini API key is also present.
//
// The native `gemini` executor is API-key-only and does NO OpenAI->Gemini
// translation, so an OAuth request handed to it fails upstream (Google 400).
// gemini-2.5-pro is catalogued in BOTH the `gemini` and `gemini-cli` sections;
// GetModelProviders must place gemini-cli first so the OAuth-capable executor is
// selected. A native-only deployment (no gemini-cli auth) still resolves to
// gemini, so the shared source stays correct for API-key setups (e.g. the RPi).
func TestGeminiCLIRouting_OAuthAccountResolvesToGeminiCLI(t *testing.T) {
	service := &Service{cfg: &sdkconfig.Config{}}
	reg := internalregistry.GetGlobalRegistry()

	geminiCLIAuth := &coreauth.Auth{
		ID:         "gemini-oauth-file.json",
		Provider:   "gemini-cli",
		Status:     coreauth.StatusActive,
		Attributes: map[string]string{"auth_kind": "oauth"},
	}
	reg.UnregisterClient(geminiCLIAuth.ID)
	t.Cleanup(func() { reg.UnregisterClient(geminiCLIAuth.ID) })
	service.registerModelsForAuth(context.Background(), geminiCLIAuth)

	// Case 1: gemini-cli OAuth account only (matches the live Singapore deployment,
	// which has 0 gemini API keys).
	providers := reg.GetModelProviders("gemini-2.5-pro")
	if len(providers) == 0 || providers[0] != "gemini-cli" {
		t.Fatalf("gemini-cli-only: expected [gemini-cli], got %v", providers)
	}
	if !reg.ClientSupportsModel(geminiCLIAuth.ID, "gemini-2.5-pro") {
		t.Fatal("expected the gemini-cli OAuth client to support gemini-2.5-pro")
	}

	// Case 2: a native gemini API key is ALSO registered. The counts tie, and the
	// provider precedence must keep gemini-cli first so OAuth requests reach the
	// translating executor rather than the API-key-only native gemini executor.
	geminiAPIKeyAuth := &coreauth.Auth{
		ID:         "gemini-apikey-auth",
		Provider:   "gemini",
		Status:     coreauth.StatusActive,
		Attributes: map[string]string{"auth_kind": "apikey"},
	}
	reg.UnregisterClient(geminiAPIKeyAuth.ID)
	t.Cleanup(func() { reg.UnregisterClient(geminiAPIKeyAuth.ID) })
	service.registerModelsForAuth(context.Background(), geminiAPIKeyAuth)

	providersBoth := reg.GetModelProviders("gemini-2.5-pro")
	if len(providersBoth) == 0 || providersBoth[0] != "gemini-cli" {
		t.Fatalf("both-present: expected gemini-cli to win the tie-break, got %v", providersBoth)
	}
}

// TestGeminiCLIRouting_NativeOnlyStillResolvesToGemini guards the RPi/API-key
// path: with only a native gemini API key (no gemini-cli OAuth), gemini-2.5-pro
// must still resolve to the native gemini provider. The gemini-cli precedence
// must not steal traffic when no gemini-cli auth exists.
func TestGeminiCLIRouting_NativeOnlyStillResolvesToGemini(t *testing.T) {
	service := &Service{cfg: &sdkconfig.Config{}}
	reg := internalregistry.GetGlobalRegistry()

	geminiAPIKeyAuth := &coreauth.Auth{
		ID:         "gemini-only-apikey-auth",
		Provider:   "gemini",
		Status:     coreauth.StatusActive,
		Attributes: map[string]string{"auth_kind": "apikey"},
	}
	reg.UnregisterClient(geminiAPIKeyAuth.ID)
	t.Cleanup(func() { reg.UnregisterClient(geminiAPIKeyAuth.ID) })
	service.registerModelsForAuth(context.Background(), geminiAPIKeyAuth)

	providers := reg.GetModelProviders("gemini-2.5-pro")
	if len(providers) == 0 || providers[0] != "gemini" {
		t.Fatalf("native-only: expected [gemini], got %v", providers)
	}
}
