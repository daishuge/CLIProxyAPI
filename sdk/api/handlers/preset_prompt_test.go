package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	coreexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v6/sdk/config"
	"github.com/tidwall/gjson"
)

const presetPromptTestMarker = "T14_INTERNAL_PRESET_PROMPT_MARKER"

type presetPromptCaptureExecutor struct {
	provider        string
	responsePayload []byte
	streamChunks    []coreexecutor.StreamChunk
	countPayload    []byte

	mu              sync.Mutex
	executePayloads [][]byte
	executeOriginal [][]byte
	streamPayloads  [][]byte
	streamOriginal  [][]byte
	countPayloads   [][]byte
	countOriginal   [][]byte
}

func (e *presetPromptCaptureExecutor) Identifier() string {
	if strings.TrimSpace(e.provider) == "" {
		return "preset-prompt-test"
	}
	return e.provider
}

func (e *presetPromptCaptureExecutor) Execute(_ context.Context, _ *coreauth.Auth, req coreexecutor.Request, opts coreexecutor.Options) (coreexecutor.Response, error) {
	e.mu.Lock()
	e.executePayloads = append(e.executePayloads, bytes.Clone(req.Payload))
	e.executeOriginal = append(e.executeOriginal, bytes.Clone(opts.OriginalRequest))
	e.mu.Unlock()

	payload := e.responsePayload
	if len(payload) == 0 {
		payload = []byte(`{"id":"preset-response","choices":[{"message":{"content":"client visible"}}]}`)
	}
	return coreexecutor.Response{Payload: bytes.Clone(payload)}, nil
}

func (e *presetPromptCaptureExecutor) ExecuteStream(_ context.Context, _ *coreauth.Auth, req coreexecutor.Request, opts coreexecutor.Options) (*coreexecutor.StreamResult, error) {
	e.mu.Lock()
	e.streamPayloads = append(e.streamPayloads, bytes.Clone(req.Payload))
	e.streamOriginal = append(e.streamOriginal, bytes.Clone(opts.OriginalRequest))
	e.mu.Unlock()

	chunks := e.streamChunks
	if len(chunks) == 0 {
		chunks = []coreexecutor.StreamChunk{{Payload: []byte("data: {\"choices\":[{\"delta\":{\"content\":\"client visible\"}}]}\n\n")}}
	}
	ch := make(chan coreexecutor.StreamChunk, len(chunks))
	for _, chunk := range chunks {
		ch <- coreexecutor.StreamChunk{Payload: bytes.Clone(chunk.Payload), Err: chunk.Err}
	}
	close(ch)
	return &coreexecutor.StreamResult{Chunks: ch}, nil
}

func (e *presetPromptCaptureExecutor) Refresh(_ context.Context, auth *coreauth.Auth) (*coreauth.Auth, error) {
	return auth, nil
}

func (e *presetPromptCaptureExecutor) CountTokens(_ context.Context, _ *coreauth.Auth, req coreexecutor.Request, opts coreexecutor.Options) (coreexecutor.Response, error) {
	e.mu.Lock()
	e.countPayloads = append(e.countPayloads, bytes.Clone(req.Payload))
	e.countOriginal = append(e.countOriginal, bytes.Clone(opts.OriginalRequest))
	e.mu.Unlock()

	payload := e.countPayload
	if len(payload) == 0 {
		payload = []byte(`{"total_tokens":7}`)
	}
	return coreexecutor.Response{Payload: bytes.Clone(payload)}, nil
}

func (e *presetPromptCaptureExecutor) HttpRequest(_ context.Context, _ *coreauth.Auth, _ *http.Request) (*http.Response, error) {
	return nil, &coreauth.Error{Code: "not_implemented", Message: "HttpRequest not implemented", HTTPStatus: http.StatusNotImplemented}
}

func (e *presetPromptCaptureExecutor) ExecutePayloads() [][]byte {
	e.mu.Lock()
	defer e.mu.Unlock()
	return cloneByteSlices(e.executePayloads)
}

func (e *presetPromptCaptureExecutor) ExecuteOriginalRequests() [][]byte {
	e.mu.Lock()
	defer e.mu.Unlock()
	return cloneByteSlices(e.executeOriginal)
}

func (e *presetPromptCaptureExecutor) StreamPayloads() [][]byte {
	e.mu.Lock()
	defer e.mu.Unlock()
	return cloneByteSlices(e.streamPayloads)
}

func (e *presetPromptCaptureExecutor) StreamOriginalRequests() [][]byte {
	e.mu.Lock()
	defer e.mu.Unlock()
	return cloneByteSlices(e.streamOriginal)
}

func (e *presetPromptCaptureExecutor) CountPayloads() [][]byte {
	e.mu.Lock()
	defer e.mu.Unlock()
	return cloneByteSlices(e.countPayloads)
}

func cloneByteSlices(src [][]byte) [][]byte {
	out := make([][]byte, len(src))
	for i := range src {
		out[i] = bytes.Clone(src[i])
	}
	return out
}

func TestPresetPromptInjectsOpenAIRequestOnly(t *testing.T) {
	executor := &presetPromptCaptureExecutor{
		provider:        "preset-prompt-openai",
		responsePayload: []byte(`{"id":"resp-1","choices":[{"message":{"role":"assistant","content":"client visible"}}]}`),
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{
		Enabled: true,
		Prompt:  presetPromptTestMarker,
	})
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"system","content":"existing system"},{"role":"user","content":"hello"}]}`, model))

	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	if bytes.Contains(payload, []byte(presetPromptTestMarker)) {
		t.Fatalf("downstream response leaked preset prompt: %s", payload)
	}

	payloads := executor.ExecutePayloads()
	if len(payloads) != 1 {
		t.Fatalf("captured upstream payloads = %d, want 1", len(payloads))
	}
	forwarded := payloads[0]
	if got := gjson.GetBytes(forwarded, "messages.0.role").String(); got != "system" {
		t.Fatalf("messages.0.role = %q, want system; payload=%s", got, forwarded)
	}
	if got := gjson.GetBytes(forwarded, "messages.0.content").String(); got != presetPromptTestMarker {
		t.Fatalf("messages.0.content = %q, want preset marker; payload=%s", got, forwarded)
	}
	if got := gjson.GetBytes(forwarded, "messages.1.content").String(); got != "existing system" {
		t.Fatalf("existing system message moved incorrectly, got %q; payload=%s", got, forwarded)
	}
	if got := gjson.GetBytes(forwarded, "messages.2.content").String(); got != "hello" {
		t.Fatalf("user message moved incorrectly, got %q; payload=%s", got, forwarded)
	}

	originals := executor.ExecuteOriginalRequests()
	if len(originals) != 1 || !bytes.Equal(originals[0], body) {
		t.Fatalf("OriginalRequest was mutated: got %q want %q", originals, body)
	}
	if bytes.Contains(originals[0], []byte(presetPromptTestMarker)) {
		t.Fatalf("OriginalRequest leaked preset prompt: %s", originals[0])
	}
}

func TestPresetPromptRedactsNonStreamingResponseLeak(t *testing.T) {
	executor := &presetPromptCaptureExecutor{
		provider:        "preset-prompt-redact-response",
		responsePayload: []byte(fmt.Sprintf(`{"choices":[{"message":{"content":%q}}]}`, "prefix "+presetPromptTestMarker+" suffix")),
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{
		Enabled: true,
		Prompt:  presetPromptTestMarker,
	})
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hello"}]}`, model))

	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	if bytes.Contains(payload, []byte(presetPromptTestMarker)) {
		t.Fatalf("downstream response leaked preset prompt: %s", payload)
	}
	if !bytes.Contains(payload, []byte(presetPromptRedactionText)) {
		t.Fatalf("downstream response missing redaction marker: %s", payload)
	}
}

func TestPresetPromptRedactsJSONEscapedResponseLeak(t *testing.T) {
	prompt := "line 1\nline 2"
	executor := &presetPromptCaptureExecutor{
		provider:        "preset-prompt-redact-escaped-response",
		responsePayload: []byte(fmt.Sprintf(`{"choices":[{"message":{"content":%q}}]}`, prompt)),
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{
		Enabled: true,
		Prompt:  prompt,
	})
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hello"}]}`, model))

	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	if bytes.Contains(payload, []byte(`line 1\nline 2`)) || bytes.Contains(payload, []byte(prompt)) {
		t.Fatalf("downstream response leaked escaped preset prompt: %s", payload)
	}
	if !bytes.Contains(payload, []byte(presetPromptRedactionText)) {
		t.Fatalf("downstream response missing redaction marker: %s", payload)
	}
}

func TestPresetPromptUsesAPIKeySpecificPrompt(t *testing.T) {
	executor := &presetPromptCaptureExecutor{provider: "preset-prompt-api-key"}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{
		Enabled: true,
		Prompt:  "global prompt",
	})
	handler.SetAPIKeyControls([]sdkconfig.APIKeyControl{
		{
			APIKey: "client-key",
			PresetPrompt: &sdkconfig.PresetPromptConfig{
				Enabled: true,
				Prompt:  "key prompt",
			},
		},
	})
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hello"}]}`, model))

	_, _, errMsg := handler.ExecuteWithAuthManager(contextWithPresetPromptAPIKey("client-key"), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	payloads := executor.ExecutePayloads()
	if len(payloads) != 1 {
		t.Fatalf("captured upstream payloads = %d, want 1", len(payloads))
	}
	if got := gjson.GetBytes(payloads[0], "messages.0.content").String(); got != "key prompt" {
		t.Fatalf("messages.0.content = %q, want key prompt; payload=%s", got, payloads[0])
	}
}

func TestPresetPromptAPIKeySpecificDisableOverridesGlobal(t *testing.T) {
	executor := &presetPromptCaptureExecutor{provider: "preset-prompt-api-key-disabled"}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{
		Enabled: true,
		Prompt:  presetPromptTestMarker,
	})
	handler.SetAPIKeyControls([]sdkconfig.APIKeyControl{
		{
			APIKey:       "client-key",
			PresetPrompt: &sdkconfig.PresetPromptConfig{Enabled: false},
		},
	})
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hello"}]}`, model))

	_, _, errMsg := handler.ExecuteWithAuthManager(contextWithPresetPromptAPIKey("client-key"), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	payloads := executor.ExecutePayloads()
	if len(payloads) != 1 || !bytes.Equal(payloads[0], body) {
		t.Fatalf("disabled per-key preset prompt changed payload: got %q want %q", payloads, body)
	}
}

func TestPresetPromptDisabledLeavesPayloadUnchanged(t *testing.T) {
	executor := &presetPromptCaptureExecutor{provider: "preset-prompt-disabled"}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{
		Enabled: false,
		Prompt:  presetPromptTestMarker,
	})
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hello"}]}`, model))

	_, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	payloads := executor.ExecutePayloads()
	if len(payloads) != 1 || !bytes.Equal(payloads[0], body) {
		t.Fatalf("disabled preset prompt changed payload: got %q want %q", payloads, body)
	}
}

func TestPresetPromptInjectsStreamingRequestWithoutLeakingChunks(t *testing.T) {
	executor := &presetPromptCaptureExecutor{
		provider:     "preset-prompt-stream",
		streamChunks: []coreexecutor.StreamChunk{{Payload: []byte("data: {\"choices\":[{\"delta\":{\"content\":\"client visible\"}}]}\n\n")}},
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{
		Enabled: true,
		Prompt:  presetPromptTestMarker,
	})
	body := []byte(fmt.Sprintf(`{"model":%q,"stream":true,"messages":[{"role":"user","content":"hello"}]}`, model))

	data, _, errs := handler.ExecuteStreamWithAuthManager(context.Background(), "openai", model, body, "")
	var got bytes.Buffer
	for chunk := range data {
		got.Write(chunk)
	}
	for errMsg := range errs {
		if errMsg != nil {
			t.Fatalf("unexpected stream error: %+v", errMsg)
		}
	}
	if bytes.Contains(got.Bytes(), []byte(presetPromptTestMarker)) {
		t.Fatalf("stream response leaked preset prompt: %s", got.String())
	}

	payloads := executor.StreamPayloads()
	if len(payloads) != 1 {
		t.Fatalf("captured stream payloads = %d, want 1", len(payloads))
	}
	if gotPrompt := gjson.GetBytes(payloads[0], "messages.0.content").String(); gotPrompt != presetPromptTestMarker {
		t.Fatalf("stream upstream prompt = %q, want preset marker; payload=%s", gotPrompt, payloads[0])
	}
	originals := executor.StreamOriginalRequests()
	if len(originals) != 1 || !bytes.Equal(originals[0], body) {
		t.Fatalf("stream OriginalRequest was mutated: got %q want %q", originals, body)
	}
}

func TestPresetPromptRedactsStreamingResponseLeakAcrossChunks(t *testing.T) {
	executor := &presetPromptCaptureExecutor{
		provider: "preset-prompt-stream-redact",
		streamChunks: []coreexecutor.StreamChunk{
			{Payload: []byte(`data: {"delta":"prefix T14_INTERNAL_`)},
			{Payload: []byte("PRESET_PROMPT_MARKER suffix\"}\n\n")},
		},
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{
		Enabled: true,
		Prompt:  presetPromptTestMarker,
	})
	body := []byte(fmt.Sprintf(`{"model":%q,"stream":true,"messages":[{"role":"user","content":"hello"}]}`, model))

	data, _, errs := handler.ExecuteStreamWithAuthManager(context.Background(), "openai", model, body, "")
	var got bytes.Buffer
	for chunk := range data {
		got.Write(chunk)
	}
	for errMsg := range errs {
		if errMsg != nil {
			t.Fatalf("unexpected stream error: %+v", errMsg)
		}
	}
	if strings.Contains(got.String(), presetPromptTestMarker) {
		t.Fatalf("stream response leaked preset prompt: %s", got.String())
	}
	if !strings.Contains(got.String(), presetPromptRedactionText) {
		t.Fatalf("stream response missing redaction marker: %s", got.String())
	}
}

func TestPresetPromptStreamRedactorKeepsJSONFrameWhole(t *testing.T) {
	executor := &presetPromptCaptureExecutor{
		provider: "preset-prompt-stream-frame",
		streamChunks: []coreexecutor.StreamChunk{
			{Payload: []byte(`data: {"choices":[{"delta":{"content":"H`)}},
			{Payload: []byte(`ello"}}]}` + "\n\n")},
		},
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{
		Enabled: true,
		Prompt:  presetPromptTestMarker,
	})
	body := []byte(fmt.Sprintf(`{"model":%q,"stream":true,"messages":[{"role":"user","content":"hello"}]}`, model))

	data, _, errs := handler.ExecuteStreamWithAuthManager(context.Background(), "openai", model, body, "")
	var chunks [][]byte
	for chunk := range data {
		chunks = append(chunks, bytes.Clone(chunk))
	}
	for errMsg := range errs {
		if errMsg != nil {
			t.Fatalf("unexpected stream error: %+v", errMsg)
		}
	}
	if len(chunks) != 1 {
		t.Fatalf("stream chunks = %d, want 1 complete frame: %q", len(chunks), chunks)
	}
	payload := strings.TrimSpace(string(bytes.TrimPrefix(bytes.TrimSpace(chunks[0]), []byte("data:"))))
	if !gjson.Valid(payload) {
		t.Fatalf("redactor emitted invalid JSON frame: %q", chunks[0])
	}
	if got := gjson.Get(payload, "choices.0.delta.content").String(); got != "Hello" {
		t.Fatalf("content = %q, want Hello; frame=%s", got, chunks[0])
	}
}

func TestPresetPromptDoesNotMutateCountRequests(t *testing.T) {
	executor := &presetPromptCaptureExecutor{provider: "preset-prompt-count"}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{
		Enabled: true,
		Prompt:  presetPromptTestMarker,
	})
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"count me"}]}`, model))

	payload, _, errMsg := handler.ExecuteCountWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteCountWithAuthManager returned error: %+v", errMsg)
	}
	if bytes.Contains(payload, []byte(presetPromptTestMarker)) {
		t.Fatalf("count response leaked preset prompt: %s", payload)
	}
	payloads := executor.CountPayloads()
	if len(payloads) != 1 || !bytes.Equal(payloads[0], body) {
		t.Fatalf("count payload was mutated: got %q want %q", payloads, body)
	}
}

func TestPresetPromptPayloadFormatMutations(t *testing.T) {
	handler := NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, nil)
	handler.SetPresetPromptConfig(sdkconfig.PresetPromptConfig{
		Enabled: true,
		Prompt:  presetPromptTestMarker,
	})

	tests := []struct {
		name        string
		handlerType string
		body        []byte
		assert      func(t *testing.T, got []byte)
	}{
		{
			name:        "openai prepends empty messages",
			handlerType: "openai",
			body:        []byte(`{"model":"m","messages":[ ]}`),
			assert: func(t *testing.T, got []byte) {
				if value := gjson.GetBytes(got, "messages.0.content").String(); value != presetPromptTestMarker {
					t.Fatalf("messages.0.content = %q, want preset marker; payload=%s", value, got)
				}
			},
		},
		{
			name:        "openai responses prepends instructions",
			handlerType: "openai-response",
			body:        []byte(`{"model":"m","instructions":"existing instructions","input":"hello"}`),
			assert: func(t *testing.T, got []byte) {
				want := presetPromptTestMarker + presetPromptSeparator + "existing instructions"
				if value := gjson.GetBytes(got, "instructions").String(); value != want {
					t.Fatalf("instructions = %q, want %q; payload=%s", value, want, got)
				}
			},
		},
		{
			name:        "openai responses creates instructions",
			handlerType: "openai-response",
			body:        []byte(`{"model":"m","input":"hello"}`),
			assert: func(t *testing.T, got []byte) {
				if value := gjson.GetBytes(got, "instructions").String(); value != presetPromptTestMarker {
					t.Fatalf("instructions = %q, want preset marker; payload=%s", value, got)
				}
			},
		},
		{
			name:        "openai responses image generation tool unchanged",
			handlerType: "openai-response",
			body:        []byte(`{"model":"m","input":"draw this","tools":[{"type":"image_generation"}]}`),
			assert: func(t *testing.T, got []byte) {
				if !bytes.Equal(got, []byte(`{"model":"m","input":"draw this","tools":[{"type":"image_generation"}]}`)) {
					t.Fatalf("image generation responses payload changed: %s", got)
				}
			},
		},
		{
			name:        "claude prepends system array",
			handlerType: "claude",
			body:        []byte(`{"model":"m","system":[{"type":"text","text":"existing"}],"messages":[{"role":"user","content":"hello"}]}`),
			assert: func(t *testing.T, got []byte) {
				if value := gjson.GetBytes(got, "system.0.text").String(); value != presetPromptTestMarker {
					t.Fatalf("system.0.text = %q, want preset marker; payload=%s", value, got)
				}
				if value := gjson.GetBytes(got, "system.1.text").String(); value != "existing" {
					t.Fatalf("system.1.text = %q, want existing; payload=%s", value, got)
				}
			},
		},
		{
			name:        "claude prepends system string",
			handlerType: "claude",
			body:        []byte(`{"model":"m","system":"existing","messages":[{"role":"user","content":"hello"}]}`),
			assert: func(t *testing.T, got []byte) {
				want := presetPromptTestMarker + presetPromptSeparator + "existing"
				if value := gjson.GetBytes(got, "system").String(); value != want {
					t.Fatalf("system = %q, want %q; payload=%s", value, want, got)
				}
			},
		},
		{
			name:        "gemini prepends existing system instruction parts",
			handlerType: "gemini",
			body:        []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}],"systemInstruction":{"parts":[{"text":"existing"}]}}`),
			assert: func(t *testing.T, got []byte) {
				if value := gjson.GetBytes(got, "systemInstruction.parts.0.text").String(); value != presetPromptTestMarker {
					t.Fatalf("systemInstruction.parts.0.text = %q, want preset marker; payload=%s", value, got)
				}
				if value := gjson.GetBytes(got, "systemInstruction.parts.1.text").String(); value != "existing" {
					t.Fatalf("systemInstruction.parts.1.text = %q, want existing; payload=%s", value, got)
				}
			},
		},
		{
			name:        "gemini creates system instruction",
			handlerType: "gemini-cli",
			body:        []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`),
			assert: func(t *testing.T, got []byte) {
				if value := gjson.GetBytes(got, "systemInstruction.parts.0.text").String(); value != presetPromptTestMarker {
					t.Fatalf("systemInstruction.parts.0.text = %q, want preset marker; payload=%s", value, got)
				}
			},
		},
		{
			name:        "unsupported format unchanged",
			handlerType: "unknown",
			body:        []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			assert: func(t *testing.T, got []byte) {
				if !bytes.Equal(got, []byte(`{"messages":[{"role":"user","content":"hello"}]}`)) {
					t.Fatalf("unsupported format changed payload: %s", got)
				}
			},
		},
		{
			name:        "invalid json unchanged",
			handlerType: "openai",
			body:        []byte(`{"messages":[`),
			assert: func(t *testing.T, got []byte) {
				if !bytes.Equal(got, []byte(`{"messages":[`)) {
					t.Fatalf("invalid JSON changed payload: %s", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.applyPresetPromptToPayload(tt.handlerType, tt.body)
			tt.assert(t, got)
		})
	}
}

func newPresetPromptTestHandler(t *testing.T, executor *presetPromptCaptureExecutor, cfg sdkconfig.PresetPromptConfig) (*BaseAPIHandler, string) {
	t.Helper()
	provider := executor.Identifier()
	authID := safePresetPromptTestName(t.Name()) + "-auth"
	model := safePresetPromptTestName(t.Name()) + "-model"
	manager := coreauth.NewManager(nil, nil, nil)
	manager.RegisterExecutor(executor)
	if _, err := manager.Register(context.Background(), &coreauth.Auth{ID: authID, Provider: provider, Status: coreauth.StatusActive}); err != nil {
		t.Fatalf("manager.Register: %v", err)
	}
	registry.GetGlobalRegistry().RegisterClient(authID, provider, []*registry.ModelInfo{{ID: model}})
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(authID)
	})

	handler := NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, manager)
	handler.SetPresetPromptConfig(cfg)
	return handler, model
}

func safePresetPromptTestName(name string) string {
	replacer := strings.NewReplacer("/", "-", "_", "-", " ", "-", "(", "-", ")", "-")
	return strings.ToLower(replacer.Replace(name))
}

func contextWithPresetPromptAPIKey(apiKey string) context.Context {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Set("apiKey", apiKey)
	return context.WithValue(context.Background(), "gin", ginCtx)
}
