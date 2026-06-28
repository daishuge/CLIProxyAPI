package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/logging/conversationlog"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

func TestPresetPromptConversationLogMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name               string
		loggingEnabled     bool
		promptEnabled      bool
		wantLogEntry       bool
		wantUpstreamPrompt bool
	}{
		{
			name:           "logging off prompt off",
			loggingEnabled: false,
			promptEnabled:  false,
		},
		{
			name:           "logging on prompt off",
			loggingEnabled: true,
			promptEnabled:  false,
			wantLogEntry:   true,
		},
		{
			name:               "logging off prompt on",
			loggingEnabled:     false,
			promptEnabled:      true,
			wantUpstreamPrompt: true,
		},
		{
			name:               "logging on prompt on",
			loggingEnabled:     true,
			promptEnabled:      true,
			wantLogEntry:       true,
			wantUpstreamPrompt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, dir := newPresetPromptConversationLogStore(t, tt.loggingEnabled)
			cfg := sdkconfig.PresetPromptConfig{Enabled: tt.promptEnabled, Prompt: presetPromptTestMarker}
			executor := &presetPromptCaptureExecutor{provider: safePresetPromptTestName(t.Name())}
			handler, model := newPresetPromptTestHandler(t, executor, cfg)
			handler.SetConversationLogStore(store)

			body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"matrix user"}]}`, model))
			ctx := newConversationLogRequestContext(t, handler, http.MethodPost, "/v1/chat/completions", body, "req-"+safePresetPromptTestName(t.Name()))

			payload, _, errMsg := handler.ExecuteWithAuthManager(ctx, "openai", model, body, "")
			if errMsg != nil {
				t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
			}
			assertPayloadDoesNotContain(t, payload, presetPromptTestMarker, "client response")

			payloads := executor.ExecutePayloads()
			if len(payloads) != 1 {
				t.Fatalf("captured upstream payloads = %d, want 1", len(payloads))
			}
			assertPayloadContainsState(t, payloads[0], presetPromptTestMarker, tt.wantUpstreamPrompt, "upstream request")

			originals := executor.ExecuteOriginalRequests()
			if len(originals) != 1 || !bytes.Equal(originals[0], body) {
				t.Fatalf("OriginalRequest was mutated: got %q want %q", originals, body)
			}
			assertPayloadDoesNotContain(t, originals[0], presetPromptTestMarker, "original request")

			result, err := store.List(conversationlog.ListQuery{Limit: 10})
			if err != nil {
				t.Fatalf("list conversation logs: %v", err)
			}
			if !tt.wantLogEntry {
				if len(result.Entries) != 0 {
					t.Fatalf("expected no conversation log entries, got %#v", result.Entries)
				}
				if _, err := os.Stat(dir); !os.IsNotExist(err) {
					t.Fatalf("expected disabled log directory to remain absent, stat err=%v", err)
				}
				return
			}
			if len(result.Entries) != 1 {
				t.Fatalf("expected one conversation log entry, got %#v", result.Entries)
			}
			entry, err := store.Read(result.Entries[0].ID)
			if err != nil {
				t.Fatalf("read conversation log entry: %v", err)
			}
			assertConversationEntryHasOriginalRequestOnly(t, entry, presetPromptTestMarker)
		})
	}
}

func TestPresetPromptExactPromptFileSmoke(t *testing.T) {
	gin.SetMode(gin.TestMode)

	promptPath := strings.TrimSpace(os.Getenv("PPAP_T15_EXACT_PROMPT_FILE"))
	if promptPath == "" {
		t.Skip("set PPAP_T15_EXACT_PROMPT_FILE to run the exact production prompt smoke")
	}
	data, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("read exact prompt file: %v", err)
	}
	prompt := strings.TrimRight(string(data), "\r\n")
	if strings.TrimSpace(prompt) == "" {
		t.Fatal("exact prompt file is empty")
	}

	store, _ := newPresetPromptConversationLogStore(t, true)
	executor := &presetPromptCaptureExecutor{provider: safePresetPromptTestName(t.Name())}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{
		Enabled: true,
		Prompt:  prompt,
	})
	handler.SetConversationLogStore(store)

	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"exact prompt smoke"}]}`, model))
	ctx := newConversationLogRequestContext(t, handler, http.MethodPost, "/v1/chat/completions", body, "req-exact-prompt")

	payload, _, errMsg := handler.ExecuteWithAuthManager(ctx, "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	assertPayloadDoesNotContain(t, payload, prompt, "client response")

	payloads := executor.ExecutePayloads()
	if len(payloads) != 1 {
		t.Fatalf("captured upstream payloads = %d, want 1", len(payloads))
	}
	assertPayloadContainsState(t, payloads[0], prompt, true, "upstream request")

	originals := executor.ExecuteOriginalRequests()
	if len(originals) != 1 || !bytes.Equal(originals[0], body) {
		t.Fatalf("OriginalRequest was mutated: got %q want %q", originals, body)
	}
	assertPayloadDoesNotContain(t, originals[0], prompt, "original request")

	entry := readSingleConversationEntry(t, store)
	assertConversationEntryHasOriginalRequestOnly(t, entry, prompt)
}

func newPresetPromptConversationLogStore(t *testing.T, enabled bool) (*conversationlog.Store, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "conversation")
	store := conversationlog.NewStore(conversationlog.Options{
		Enabled:           enabled,
		Directory:         dir,
		MaxFileSizeBytes:  1024 * 1024,
		MaxTotalSizeBytes: 1024 * 1024,
		MaxEntryBytes:     256 * 1024,
	})
	return store, dir
}

func assertConversationEntryHasOriginalRequestOnly(t *testing.T, entry conversationlog.Entry, prompt string) {
	t.Helper()
	if entry.StatusCode != http.StatusOK {
		t.Fatalf("expected logged status 200, got %d", entry.StatusCode)
	}
	if !bytes.Contains(entry.Request.Body, []byte("user")) {
		t.Fatalf("expected original request body in conversation log, got %s", entry.Request.Body)
	}
	assertPayloadDoesNotContain(t, entry.Request.Body, prompt, "conversation log request")
	assertPayloadDoesNotContain(t, entry.Response.Body, prompt, "conversation log response")
	encoded, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal conversation log entry: %v", err)
	}
	assertPayloadDoesNotContain(t, encoded, prompt, "conversation log entry")
	if entry.Metadata["operation"] != "execute" {
		t.Fatalf("expected execute operation metadata, got %#v", entry.Metadata)
	}
}

func assertPayloadContainsState(t *testing.T, payload []byte, needle string, want bool, label string) {
	t.Helper()
	got := bytes.Contains(payload, []byte(needle))
	if got == want {
		return
	}
	if want {
		t.Fatalf("%s did not contain expected preset prompt marker", label)
	}
	t.Fatalf("%s leaked preset prompt marker", label)
}

func assertPayloadDoesNotContain(t *testing.T, payload []byte, needle string, label string) {
	t.Helper()
	if bytes.Contains(payload, []byte(needle)) {
		t.Fatalf("%s leaked preset prompt marker", label)
	}
}
