package openai

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteOpenAIStreamDataSplitsConcatenatedJSONPayloads(t *testing.T) {
	var out bytes.Buffer

	writeOpenAIStreamData(&out, []byte(`{"id":"chunk-1","choices":[{"delta":{"content":"a"}}]}{"id":"chunk-2","choices":[{"delta":{"content":"b"}}]}`))

	events := sseEvents(t, out.String())
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2. Body: %q", len(events), out.String())
	}
	for _, event := range events {
		payload := strings.TrimPrefix(event, "data: ")
		if !json.Valid([]byte(payload)) {
			t.Fatalf("payload is not valid JSON: %q", payload)
		}
	}
	if strings.Contains(events[0], `chunk-2`) {
		t.Fatalf("first SSE event still contains the second JSON object: %q", events[0])
	}
}

func TestWriteOpenAIStreamDataPreservesSingleJSONPayload(t *testing.T) {
	var out bytes.Buffer

	writeOpenAIStreamData(&out, []byte(`{"id":"chunk-1","choices":[{"delta":{"content":"a"}}]}`))

	events := sseEvents(t, out.String())
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1. Body: %q", len(events), out.String())
	}
	if got, want := events[0], `data: {"id":"chunk-1","choices":[{"delta":{"content":"a"}}]}`; got != want {
		t.Fatalf("event = %q, want %q", got, want)
	}
}

func TestWriteOpenAIStreamDataPreservesExistingSSEPayload(t *testing.T) {
	var out bytes.Buffer

	writeOpenAIStreamData(&out, []byte("data: {\"id\":\"chunk-1\"}\n\n"))

	if got, want := out.String(), "data: {\"id\":\"chunk-1\"}\n\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWriteOpenAIStreamDataCompletesExistingSSEPayload(t *testing.T) {
	var out bytes.Buffer

	writeOpenAIStreamData(&out, []byte("data: {\"id\":\"chunk-1\"}"))

	if got, want := out.String(), "data: {\"id\":\"chunk-1\"}\n\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func sseEvents(t *testing.T, body string) []string {
	t.Helper()

	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "\n\n")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
