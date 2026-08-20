package cliproxy

import "testing"

func TestWithProviderBuiltinsPreservesGemini25ProForPlugins(t *testing.T) {
	for _, provider := range []string{"gemini", "gemini-cli"} {
		models := withProviderBuiltins(provider, []*ModelInfo{{ID: "gemini-3.7-flash"}})
		found := false
		for _, model := range models {
			if model != nil && model.ID == "gemini-2.5-pro" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("provider %q did not receive gemini-2.5-pro compatibility model", provider)
		}
	}
}

func TestWithProviderBuiltinsLeavesOtherProvidersUnchanged(t *testing.T) {
	input := []*ModelInfo{{ID: "claude-sonnet-5"}}
	output := withProviderBuiltins("claude", input)
	if len(output) != 1 || output[0].ID != "claude-sonnet-5" {
		t.Fatalf("non-Gemini provider models changed: %#v", output)
	}
}
