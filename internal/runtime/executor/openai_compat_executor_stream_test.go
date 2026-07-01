package executor

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v6/sdk/translator"
)

func TestOpenAICompatExecutorStreamReportsRawJSONErrorLines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`{"error":{"message":"upstream failed"}}` + "\n"))
	}))
	defer server.Close()

	executor := NewOpenAICompatExecutor("openai-compatibility", &config.Config{})
	result, err := executor.ExecuteStream(context.Background(), openAICompatTestAuth(server.URL), openAICompatTestRequest(), openAICompatTestOptions())
	if err != nil {
		t.Fatalf("ExecuteStream error: %v", err)
	}

	chunk, ok := <-result.Chunks
	if !ok {
		t.Fatal("stream closed without an error chunk")
	}
	if chunk.Err == nil {
		t.Fatalf("first chunk error = nil, payload = %q", chunk.Payload)
	}
	status, ok := chunk.Err.(interface{ StatusCode() int })
	if !ok {
		t.Fatalf("error type %T does not expose StatusCode", chunk.Err)
	}
	if got := status.StatusCode(); got != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", got, http.StatusBadGateway)
	}
}

func TestOpenAICompatExecutorStreamSkipsSSEMetadataLines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(": keep-alive\n"))
		_, _ = w.Write([]byte("event: message\n"))
		_, _ = w.Write([]byte("id: chunk-1\n"))
		_, _ = w.Write([]byte("retry: 1000\n"))
		_, _ = w.Write([]byte(`  data: {"id":"chatcmpl-1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":null}],"usage":null}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	executor := NewOpenAICompatExecutor("openai-compatibility", &config.Config{})
	result, err := executor.ExecuteStream(context.Background(), openAICompatTestAuth(server.URL), openAICompatTestRequest(), openAICompatTestOptions())
	if err != nil {
		t.Fatalf("ExecuteStream error: %v", err)
	}

	var combined bytes.Buffer
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			t.Fatalf("unexpected stream error: %v", chunk.Err)
		}
		combined.Write(chunk.Payload)
		combined.WriteByte('\n')
	}
	if !bytes.Contains(combined.Bytes(), []byte(`"content":"hi"`)) {
		t.Fatalf("expected translated stream to include content chunk, got %q", combined.String())
	}
}

func openAICompatTestAuth(baseURL string) *cliproxyauth.Auth {
	return &cliproxyauth.Auth{Attributes: map[string]string{
		"base_url": baseURL,
		"api_key":  "test",
	}}
}

func openAICompatTestRequest() cliproxyexecutor.Request {
	payload := []byte(`{"model":"gpt-test","messages":[{"role":"user","content":"hi"}],"stream":true}`)
	return cliproxyexecutor.Request{
		Model:   "gpt-test",
		Payload: payload,
		Format:  sdktranslator.FromString("openai"),
	}
}

func openAICompatTestOptions() cliproxyexecutor.Options {
	return cliproxyexecutor.Options{
		Stream:       true,
		SourceFormat: sdktranslator.FromString("openai"),
	}
}
