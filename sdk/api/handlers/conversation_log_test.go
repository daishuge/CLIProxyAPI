package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/logging"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/logging/conversationlog"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	coreexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

type conversationLogTestExecutor struct {
	provider        string
	responsePayload []byte
	responseHeaders http.Header
	streamHeaders   http.Header
	streamChunks    []coreexecutor.StreamChunk
	executeErr      error
	streamErr       error
}

type conversationLogAPIHandler struct{}

func (conversationLogAPIHandler) HandlerType() string { return "openai" }

func (conversationLogAPIHandler) Models() []map[string]any { return nil }

func (e *conversationLogTestExecutor) Identifier() string {
	if strings.TrimSpace(e.provider) == "" {
		return "conversation-log-test"
	}
	return e.provider
}

func (e *conversationLogTestExecutor) Execute(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (coreexecutor.Response, error) {
	if e.executeErr != nil {
		return coreexecutor.Response{}, e.executeErr
	}
	return coreexecutor.Response{Payload: bytes.Clone(e.responsePayload), Headers: e.responseHeaders.Clone()}, nil
}

func (e *conversationLogTestExecutor) ExecuteStream(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (*coreexecutor.StreamResult, error) {
	if e.streamErr != nil {
		return nil, e.streamErr
	}
	ch := make(chan coreexecutor.StreamChunk, len(e.streamChunks))
	for _, chunk := range e.streamChunks {
		ch <- coreexecutor.StreamChunk{Payload: bytes.Clone(chunk.Payload), Err: chunk.Err}
	}
	close(ch)
	return &coreexecutor.StreamResult{Headers: e.streamHeaders.Clone(), Chunks: ch}, nil
}

func (e *conversationLogTestExecutor) Refresh(ctx context.Context, auth *coreauth.Auth) (*coreauth.Auth, error) {
	return auth, nil
}

func (e *conversationLogTestExecutor) CountTokens(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (coreexecutor.Response, error) {
	return coreexecutor.Response{Payload: bytes.Clone(e.responsePayload), Headers: e.responseHeaders.Clone()}, e.executeErr
}

func (e *conversationLogTestExecutor) HttpRequest(ctx context.Context, auth *coreauth.Auth, req *http.Request) (*http.Response, error) {
	return nil, &coreauth.Error{Code: "not_implemented", Message: "HttpRequest not implemented", HTTPStatus: http.StatusNotImplemented}
}

func TestConversationLogNonStreamingEnabledCapturesAndRedacts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := conversationlog.NewStore(conversationlog.Options{
		Enabled:           true,
		Directory:         filepath.Join(t.TempDir(), "conversation"),
		MaxFileSizeBytes:  1024 * 1024,
		MaxTotalSizeBytes: 1024 * 1024,
		MaxEntryBytes:     256 * 1024,
	})
	executor := &conversationLogTestExecutor{
		provider:        "conversation-log-nonstream",
		responsePayload: []byte(`{"id":"resp-1","api_key":"response-secret","usage":{"total_tokens":7}}`),
		responseHeaders: http.Header{
			"Set-Cookie": {"session=response-secret"},
			"X-Trace":    {"trace-1"},
		},
	}
	handler, model := newConversationLogTestHandler(t, store, executor)
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hello"}],"api_key":"request-secret"}`, model))
	ctx := newConversationLogRequestContext(t, handler, http.MethodPost, "/v1/chat/completions", body, "req-nonstream")

	payload, headers, errMsg := handler.ExecuteWithAuthManager(ctx, "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	if headers != nil {
		t.Fatalf("expected no passthrough headers by default, got %#v", headers)
	}
	if !bytes.Contains(payload, []byte(`"resp-1"`)) {
		t.Fatalf("unexpected response payload: %s", payload)
	}

	entry := readSingleConversationEntry(t, store)
	if entry.RequestID != "req-nonstream" {
		t.Fatalf("expected request id req-nonstream, got %q", entry.RequestID)
	}
	if entry.Method != http.MethodPost || entry.Path != "/v1/chat/completions" {
		t.Fatalf("unexpected method/path: %s %s", entry.Method, entry.Path)
	}
	if entry.Provider != "conversation-log-nonstream" || entry.Model != model {
		t.Fatalf("unexpected provider/model: %q %q", entry.Provider, entry.Model)
	}
	if entry.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", entry.StatusCode)
	}
	if string(entry.Usage) != `{"total_tokens":7}` {
		t.Fatalf("unexpected usage: %s", entry.Usage)
	}
	assertHeaderRedacted(t, entry.RequestHeaders, "Authorization")
	assertHeaderRedacted(t, entry.RequestHeaders, "X-Api-Key")
	assertHeaderRedacted(t, entry.ResponseHeaders, "Set-Cookie")

	encoded, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal entry: %v", err)
	}
	if bytes.Contains(encoded, []byte("request-secret")) || bytes.Contains(encoded, []byte("response-secret")) {
		t.Fatalf("conversation log entry leaked a secret: %s", encoded)
	}
	if !bytes.Contains(encoded, []byte("[REDACTED]")) {
		t.Fatalf("expected redacted marker in entry: %s", encoded)
	}
	if entry.Metadata["selected_auth_id"] == "" {
		t.Fatalf("expected selected auth metadata, got %#v", entry.Metadata)
	}
}

func TestConversationLogDisabledDoesNotWriteFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := filepath.Join(t.TempDir(), "conversation")
	store := conversationlog.NewStore(conversationlog.Options{
		Enabled:           false,
		Directory:         dir,
		MaxFileSizeBytes:  1024 * 1024,
		MaxTotalSizeBytes: 1024 * 1024,
		MaxEntryBytes:     256 * 1024,
	})
	executor := &conversationLogTestExecutor{
		provider:        "conversation-log-disabled",
		responsePayload: []byte(`{"id":"resp-disabled"}`),
	}
	handler, model := newConversationLogTestHandler(t, store, executor)
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hello"}]}`, model))
	ctx := newConversationLogRequestContext(t, handler, http.MethodPost, "/v1/chat/completions", body, "req-disabled")

	_, _, errMsg := handler.ExecuteWithAuthManager(ctx, "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected disabled store to leave log directory absent, stat err=%v", err)
	}
}

func TestConversationLogStreamingCapturesChunksUsageAndHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := conversationlog.NewStore(conversationlog.Options{
		Enabled:           true,
		Directory:         filepath.Join(t.TempDir(), "conversation"),
		MaxFileSizeBytes:  1024 * 1024,
		MaxTotalSizeBytes: 1024 * 1024,
		MaxEntryBytes:     256 * 1024,
	})
	executor := &conversationLogTestExecutor{
		provider:      "conversation-log-stream",
		streamHeaders: http.Header{"X-Upstream": {"stream-1"}},
		streamChunks: []coreexecutor.StreamChunk{
			{Payload: []byte(`{"choices":[{"delta":{"content":"hel"}}]}`)},
			{Payload: []byte(`{"choices":[{"delta":{"content":"lo"}}],"usage":{"completion_tokens":2}}`)},
		},
	}
	handler, model := newConversationLogTestHandler(t, store, executor)
	body := []byte(fmt.Sprintf(`{"model":%q,"stream":true,"messages":[{"role":"user","content":"hello"}]}`, model))
	ctx := newConversationLogRequestContext(t, handler, http.MethodPost, "/v1/chat/completions", body, "req-stream")

	data, headers, errs := handler.ExecuteStreamWithAuthManager(ctx, "openai", model, body, "")
	if data == nil || errs == nil {
		t.Fatalf("expected non-nil stream channels")
	}
	if headers != nil {
		t.Fatalf("expected no passthrough headers by default, got %#v", headers)
	}
	var got bytes.Buffer
	for chunk := range data {
		got.Write(chunk)
	}
	for errMsg := range errs {
		if errMsg != nil {
			t.Fatalf("unexpected stream error: %+v", errMsg)
		}
	}
	if !strings.Contains(got.String(), "hel") || !strings.Contains(got.String(), "lo") {
		t.Fatalf("unexpected stream payload: %s", got.String())
	}

	entry := readSingleConversationEntry(t, store)
	if entry.RequestID != "req-stream" {
		t.Fatalf("expected request id req-stream, got %q", entry.RequestID)
	}
	if entry.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", entry.StatusCode)
	}
	if len(entry.Response.Chunks) != 2 {
		t.Fatalf("expected 2 captured chunks, got %#v", entry.Response.Chunks)
	}
	if string(entry.Usage) != `{"completion_tokens":2}` {
		t.Fatalf("unexpected usage: %s", entry.Usage)
	}
	if gotHeader := http.Header(entry.ResponseHeaders).Get("X-Upstream"); gotHeader != "stream-1" {
		t.Fatalf("expected upstream header in log, got %q", gotHeader)
	}
	if entry.Metadata["stream"] != "true" {
		t.Fatalf("expected stream metadata, got %#v", entry.Metadata)
	}
}

func TestConversationLogStreamingTruncatesLargeChunksWithoutTruncatingClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := conversationlog.NewStore(conversationlog.Options{
		Enabled:           true,
		Directory:         filepath.Join(t.TempDir(), "conversation"),
		MaxFileSizeBytes:  64 * 1024,
		MaxTotalSizeBytes: 128 * 1024,
		MaxEntryBytes:     4096,
	})
	largeChunk := strings.Repeat("x", int(conversationBodyBudget(store))*2)
	executor := &conversationLogTestExecutor{
		provider:      "conversation-log-stream-truncate",
		streamHeaders: http.Header{"X-Upstream": {"stream-large"}},
		streamChunks: []coreexecutor.StreamChunk{
			{Payload: []byte(largeChunk)},
		},
	}
	handler, model := newConversationLogTestHandler(t, store, executor)
	body := []byte(fmt.Sprintf(`{"model":%q,"stream":true,"messages":[{"role":"user","content":"hello"}]}`, model))
	ctx := newConversationLogRequestContext(t, handler, http.MethodPost, "/v1/chat/completions", body, "req-stream-large")

	data, _, errs := handler.ExecuteStreamWithAuthManager(ctx, "openai", model, body, "")
	var clientPayload bytes.Buffer
	for chunk := range data {
		clientPayload.Write(chunk)
	}
	for errMsg := range errs {
		if errMsg != nil {
			t.Fatalf("unexpected stream error: %+v", errMsg)
		}
	}
	if clientPayload.String() != largeChunk {
		t.Fatalf("client payload was truncated: got %d bytes, want %d", clientPayload.Len(), len(largeChunk))
	}

	entry := readSingleConversationEntry(t, store)
	if entry.Response.Bytes != int64(len(largeChunk)) {
		t.Fatalf("logged response bytes = %d, want %d", entry.Response.Bytes, len(largeChunk))
	}
	if !entry.Response.Truncated {
		t.Fatalf("expected logged response to be marked truncated")
	}
	captured := strings.Join(entry.Response.Chunks, "")
	if len(captured) != int(conversationBodyBudget(store)) {
		t.Fatalf("captured log chunk bytes = %d, want %d", len(captured), conversationBodyBudget(store))
	}
}

func TestConversationLogStreamingTruncatesUTF8ChunksAtRuneBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := conversationlog.NewStore(conversationlog.Options{
		Enabled:           true,
		Directory:         filepath.Join(t.TempDir(), "conversation"),
		MaxFileSizeBytes:  64 * 1024,
		MaxTotalSizeBytes: 128 * 1024,
		MaxEntryBytes:     4096,
	})
	largeChunk := strings.Repeat("你", int(conversationBodyBudget(store)))
	executor := &conversationLogTestExecutor{
		provider: "conversation-log-stream-utf8-truncate",
		streamChunks: []coreexecutor.StreamChunk{
			{Payload: []byte(largeChunk)},
		},
	}
	handler, model := newConversationLogTestHandler(t, store, executor)
	body := []byte(fmt.Sprintf(`{"model":%q,"stream":true,"messages":[{"role":"user","content":"hello"}]}`, model))
	ctx := newConversationLogRequestContext(t, handler, http.MethodPost, "/v1/chat/completions", body, "req-stream-utf8")

	data, _, errs := handler.ExecuteStreamWithAuthManager(ctx, "openai", model, body, "")
	for range data {
	}
	for errMsg := range errs {
		if errMsg != nil {
			t.Fatalf("unexpected stream error: %+v", errMsg)
		}
	}

	entry := readSingleConversationEntry(t, store)
	captured := strings.Join(entry.Response.Chunks, "")
	if !utf8.ValidString(captured) {
		t.Fatalf("captured log chunk is not valid UTF-8: %q", captured)
	}
	if strings.ContainsRune(captured, utf8.RuneError) {
		t.Fatalf("captured log chunk contains replacement runes from a split UTF-8 sequence: %q", captured)
	}
	if !entry.Response.Truncated {
		t.Fatalf("expected logged response to be marked truncated")
	}
}

func TestConversationLogNonStreamingCapturesErrorStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := conversationlog.NewStore(conversationlog.Options{
		Enabled:           true,
		Directory:         filepath.Join(t.TempDir(), "conversation"),
		MaxFileSizeBytes:  1024 * 1024,
		MaxTotalSizeBytes: 1024 * 1024,
		MaxEntryBytes:     256 * 1024,
	})
	executor := &conversationLogTestExecutor{
		provider: "conversation-log-error",
		executeErr: &coreauth.Error{
			Code:       "bad_gateway",
			Message:    "upstream failed",
			HTTPStatus: http.StatusBadGateway,
		},
	}
	handler, model := newConversationLogTestHandler(t, store, executor)
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hello"}]}`, model))
	ctx := newConversationLogRequestContext(t, handler, http.MethodPost, "/v1/chat/completions", body, "req-error")

	payload, _, errMsg := handler.ExecuteWithAuthManager(ctx, "openai", model, body, "")
	if payload != nil {
		t.Fatalf("expected nil payload on upstream error, got %s", payload)
	}
	if errMsg == nil || errMsg.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502 error message, got %+v", errMsg)
	}

	entry := readSingleConversationEntry(t, store)
	if entry.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected logged status 502, got %d", entry.StatusCode)
	}
	if !strings.Contains(entry.Error, "upstream failed") {
		t.Fatalf("expected logged error to include upstream failure, got %q", entry.Error)
	}
}

func TestConversationLogWriteFailureDoesNotBreakProxyResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dirAsFile := filepath.Join(t.TempDir(), "conversation-file")
	if err := os.WriteFile(dirAsFile, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write directory placeholder: %v", err)
	}
	store := conversationlog.NewStore(conversationlog.Options{
		Enabled:           true,
		Directory:         dirAsFile,
		MaxFileSizeBytes:  1024 * 1024,
		MaxTotalSizeBytes: 1024 * 1024,
		MaxEntryBytes:     256 * 1024,
	})
	executor := &conversationLogTestExecutor{
		provider:        "conversation-log-fail-open",
		responsePayload: []byte(`{"id":"resp-ok"}`),
	}
	handler, model := newConversationLogTestHandler(t, store, executor)
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hello"}]}`, model))
	ctx := newConversationLogRequestContext(t, handler, http.MethodPost, "/v1/chat/completions", body, "req-fail-open")

	payload, _, errMsg := handler.ExecuteWithAuthManager(ctx, "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error despite log write failure: %+v", errMsg)
	}
	if !bytes.Contains(payload, []byte("resp-ok")) {
		t.Fatalf("unexpected response payload: %s", payload)
	}
}

func newConversationLogTestHandler(t *testing.T, store *conversationlog.Store, executor *conversationLogTestExecutor) (*BaseAPIHandler, string) {
	t.Helper()
	provider := executor.Identifier()
	authID := safeConversationLogTestName(t.Name()) + "-auth"
	model := safeConversationLogTestName(t.Name()) + "-model"
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
	handler.SetConversationLogStore(store)
	return handler, model
}

func newConversationLogRequestContext(t *testing.T, handler *BaseAPIHandler, method, path string, body []byte, requestID string) context.Context {
	t.Helper()
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer request-secret")
	req.Header.Set("X-Api-Key", "request-secret")
	ginCtx.Request = req
	logging.SetGinRequestID(ginCtx, requestID)
	ctx, cancel := handler.GetContextWithCancel(conversationLogAPIHandler{}, ginCtx, logging.WithRequestID(context.Background(), requestID))
	t.Cleanup(func() {
		cancel()
	})
	return ctx
}

func readSingleConversationEntry(t *testing.T, store *conversationlog.Store) conversationlog.Entry {
	t.Helper()
	result, err := store.List(conversationlog.ListQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list conversation logs: %v", err)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected exactly one conversation log entry, got %#v", result.Entries)
	}
	entry, err := store.Read(result.Entries[0].ID)
	if err != nil {
		t.Fatalf("read conversation log entry: %v", err)
	}
	return entry
}

func assertHeaderRedacted(t *testing.T, headers map[string][]string, key string) {
	t.Helper()
	values := http.Header(headers).Values(key)
	if len(values) == 0 {
		t.Fatalf("expected header %s to be present", key)
	}
	for _, value := range values {
		if value != "[REDACTED]" {
			t.Fatalf("expected header %s to be redacted, got %q", key, value)
		}
	}
}

func safeConversationLogTestName(name string) string {
	replacer := strings.NewReplacer("/", "-", "_", "-", " ", "-", "(", "-", ")", "-")
	return strings.ToLower(replacer.Replace(name))
}
