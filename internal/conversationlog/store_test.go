package conversationlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	appconfig "github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func TestStoreDisabledDoesNotCreateFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "conversation-logs")
	store := NewStore(Options{
		Enabled:           false,
		Directory:         dir,
		MaxFileSizeBytes:  1024,
		MaxTotalSizeBytes: 4096,
		MaxEntryBytes:     1024,
	})

	location, err := store.Write(Entry{
		RequestID: "req-disabled",
		Request: Payload{
			Body: json.RawMessage(`{"messages":[{"role":"user","content":"private prompt"}]}`),
		},
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if location.ID != "" || location.File != "" || location.Size != 0 {
		t.Fatalf("disabled Write() location = %+v, want zero value", location)
	}
	if _, errStat := os.Stat(dir); !os.IsNotExist(errStat) {
		t.Fatalf("disabled store created directory or returned unexpected error: %v", errStat)
	}

	list, err := store.List(ListQuery{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list.Entries) != 0 {
		t.Fatalf("disabled List() returned %d entries, want 0", len(list.Entries))
	}
}

func TestOptionsFromConfigResolvesRelativeDirectory(t *testing.T) {
	configDir := t.TempDir()
	cfg := &appconfig.Config{
		ConversationLog: appconfig.ConversationLogConfig{
			Enabled:        true,
			Directory:      " audit ",
			MaxFileSizeMB:  2,
			MaxTotalSizeMB: 8,
			MaxEntryBytes:  4096,
		},
	}

	opts := OptionsFromConfig(cfg, filepath.Join(configDir, "config.yaml"))

	if !opts.Enabled {
		t.Fatal("Enabled = false, want true")
	}
	wantDir := filepath.Join(configDir, "audit")
	if opts.Directory != wantDir {
		t.Fatalf("Directory = %q, want %q", opts.Directory, wantDir)
	}
	if opts.MaxFileSizeBytes != 2*1024*1024 {
		t.Fatalf("MaxFileSizeBytes = %d, want %d", opts.MaxFileSizeBytes, 2*1024*1024)
	}
	if opts.MaxTotalSizeBytes != 8*1024*1024 {
		t.Fatalf("MaxTotalSizeBytes = %d, want %d", opts.MaxTotalSizeBytes, 8*1024*1024)
	}
	if opts.MaxEntryBytes != 4096 {
		t.Fatalf("MaxEntryBytes = %d, want 4096", opts.MaxEntryBytes)
	}
}

func TestStoreWriteReadListAndRedact(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "conversation-logs")
	now := time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)
	store := NewStore(Options{
		Enabled:           true,
		Directory:         dir,
		MaxFileSizeBytes:  1024 * 1024,
		MaxTotalSizeBytes: 4 * 1024 * 1024,
		MaxEntryBytes:     1024 * 1024,
	})
	store.SetNowForTest(func() time.Time { return now })

	location, err := store.Write(Entry{
		RequestID:   "req-1",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "codex",
		Model:       "gpt-5.5",
		UpstreamURL: "https://upstream.example/v1/chat/completions?api_key=secret&trace=keep",
		StatusCode:  200,
		RequestHeaders: map[string][]string{
			"Authorization": []string{"Bearer secret"},
			"X-Trace":       []string{"visible"},
		},
		ResponseHeaders: map[string][]string{
			"Set-Cookie":   []string{"sid=secret"},
			"Content-Type": []string{"application/json"},
		},
		Request: Payload{
			Body: json.RawMessage(`{"messages":[{"role":"user","content":"hello"}]}`),
		},
		Response: Payload{
			Text: "world",
		},
		Usage:    json.RawMessage(`{"total_tokens":12}`),
		Metadata: map[string]string{"api_key": "secret", "route": "visible"},
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if location.ID == "" || location.File == "" || location.Size == 0 {
		t.Fatalf("Write() location = %+v, want populated fields", location)
	}

	if runtime.GOOS != "windows" {
		assertPerm(t, dir, 0o700)
		assertPerm(t, location.File, 0o600)
	}

	got, err := store.Read(location.ID)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got.RequestID != "req-1" || got.Method != "POST" || got.Model != "gpt-5.5" {
		t.Fatalf("Read() = %+v, want request metadata", got)
	}
	if got.CreatedAt != now {
		t.Fatalf("CreatedAt = %s, want %s", got.CreatedAt, now)
	}
	if got.RequestHeaders["Authorization"][0] != "[REDACTED]" {
		t.Fatalf("Authorization header was not redacted: %+v", got.RequestHeaders)
	}
	if got.RequestHeaders["X-Trace"][0] != "visible" {
		t.Fatalf("non-sensitive header was not preserved: %+v", got.RequestHeaders)
	}
	if got.ResponseHeaders["Set-Cookie"][0] != "[REDACTED]" {
		t.Fatalf("Set-Cookie header was not redacted: %+v", got.ResponseHeaders)
	}
	if strings.Contains(got.UpstreamURL, "secret") || !strings.Contains(got.UpstreamURL, "trace=keep") {
		t.Fatalf("UpstreamURL redaction mismatch: %q", got.UpstreamURL)
	}
	if got.Metadata["api_key"] != "[REDACTED]" || got.Metadata["route"] != "visible" {
		t.Fatalf("metadata redaction mismatch: %+v", got.Metadata)
	}

	list, err := store.List(ListQuery{Limit: 1})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list.Entries) != 1 {
		t.Fatalf("List() returned %d entries, want 1", len(list.Entries))
	}
	if list.Entries[0].ID != location.ID || list.Entries[0].RequestID != "req-1" {
		t.Fatalf("List() summary = %+v, want written entry", list.Entries[0])
	}
}

func TestStoreListSkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conversation-20260520T103000.000000000Z-aaaaaaaaaaaaaaaa.jsonl")
	good := Entry{ID: "good", RequestID: "req-good", CreatedAt: time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)}
	goodLine, err := json.Marshal(good)
	if err != nil {
		t.Fatalf("marshal good entry: %v", err)
	}
	content := append([]byte("not-json\n"), append(goodLine, '\n')...)
	if errWrite := os.WriteFile(path, content, 0o600); errWrite != nil {
		t.Fatalf("write malformed fixture: %v", errWrite)
	}

	store := NewStore(Options{
		Enabled:           true,
		Directory:         dir,
		MaxFileSizeBytes:  1024,
		MaxTotalSizeBytes: 4096,
		MaxEntryBytes:     1024,
	})

	list, err := store.List(ListQuery{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if list.Malformed != 1 {
		t.Fatalf("Malformed = %d, want 1", list.Malformed)
	}
	if len(list.Entries) != 1 || list.Entries[0].ID != "good" {
		t.Fatalf("Entries = %+v, want only good entry", list.Entries)
	}

	entry, err := store.Read("good")
	if err != nil {
		t.Fatalf("Read(good) error = %v", err)
	}
	if entry.RequestID != "req-good" {
		t.Fatalf("Read(good) = %+v, want good entry", entry)
	}
}

func TestStoreListSkipsOversizedMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conversation-20260520T103000.000000000Z-bbbbbbbbbbbbbbbb.jsonl")
	good := Entry{ID: "good", RequestID: "req-good", CreatedAt: time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)}
	goodLine, err := json.Marshal(good)
	if err != nil {
		t.Fatalf("marshal good entry: %v", err)
	}
	content := []byte(strings.Repeat("x", 128*1024) + "\n" + string(goodLine) + "\n")
	if errWrite := os.WriteFile(path, content, 0o600); errWrite != nil {
		t.Fatalf("write oversized malformed fixture: %v", errWrite)
	}

	store := NewStore(Options{
		Enabled:           true,
		Directory:         dir,
		MaxFileSizeBytes:  256 * 1024,
		MaxTotalSizeBytes: 512 * 1024,
		MaxEntryBytes:     128,
	})

	list, err := store.List(ListQuery{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if list.Malformed != 1 {
		t.Fatalf("Malformed = %d, want 1", list.Malformed)
	}
	if len(list.Entries) != 1 || list.Entries[0].ID != "good" {
		t.Fatalf("Entries = %+v, want only good entry", list.Entries)
	}
	entry, err := store.Read("good")
	if err != nil {
		t.Fatalf("Read(good) error = %v", err)
	}
	if entry.RequestID != "req-good" {
		t.Fatalf("Read(good) = %+v, want good entry", entry)
	}
}

func TestStoreListPaginationAcrossShards(t *testing.T) {
	dir := t.TempDir()
	current := time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)
	store := NewStore(Options{
		Enabled:           true,
		Directory:         dir,
		MaxFileSizeBytes:  700,
		MaxTotalSizeBytes: 16 * 1024,
		MaxEntryBytes:     4096,
	})
	store.SetNowForTest(func() time.Time {
		current = current.Add(time.Second)
		return current
	})

	for i := 0; i < 5; i++ {
		_, err := store.Write(Entry{
			ID:        fmt.Sprintf("entry-%d", i),
			RequestID: fmt.Sprintf("req-%d", i),
			Method:    "POST",
			Path:      "/v1/chat/completions",
			Request:   Payload{Text: strings.Repeat(fmt.Sprintf("request-%d", i), 20)},
			Response:  Payload{Text: strings.Repeat(fmt.Sprintf("response-%d", i), 20)},
		})
		if err != nil {
			t.Fatalf("Write(%d) error = %v", i, err)
		}
	}

	files, err := listConversationFiles(dir, false)
	if err != nil {
		t.Fatalf("listConversationFiles() error = %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("expected writes to span multiple shards, got %d file(s)", len(files))
	}

	first, err := store.List(ListQuery{Limit: 2})
	if err != nil {
		t.Fatalf("List(first) error = %v", err)
	}
	assertSummaryIDs(t, first.Entries, []string{"entry-4", "entry-3"})
	if first.NextCursor == "" {
		t.Fatalf("first page NextCursor is empty")
	}

	second, err := store.List(ListQuery{Limit: 2, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("List(second) error = %v", err)
	}
	assertSummaryIDs(t, second.Entries, []string{"entry-2", "entry-1"})
	if second.NextCursor == "" {
		t.Fatalf("second page NextCursor is empty")
	}

	third, err := store.List(ListQuery{Limit: 2, Cursor: second.NextCursor})
	if err != nil {
		t.Fatalf("List(third) error = %v", err)
	}
	assertSummaryIDs(t, third.Entries, []string{"entry-0"})
	if third.NextCursor != "" {
		t.Fatalf("third page NextCursor = %q, want empty", third.NextCursor)
	}

	if _, err := store.List(ListQuery{Cursor: "not-a-cursor"}); err == nil {
		t.Fatalf("List() with invalid cursor returned nil error")
	}
}

func TestStoreListFiltersAndCursor(t *testing.T) {
	dir := t.TempDir()
	current := time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)
	store := NewStore(Options{
		Enabled:           true,
		Directory:         dir,
		MaxFileSizeBytes:  1024 * 1024,
		MaxTotalSizeBytes: 2 * 1024 * 1024,
		MaxEntryBytes:     4096,
	})
	store.SetNowForTest(func() time.Time {
		current = current.Add(time.Minute)
		return current
	})

	entries := []Entry{
		{ID: "codex-old", RequestID: "req-1", Method: "POST", Path: "/v1/chat/completions", Provider: "codex", Model: "gpt-5.5", StatusCode: 200},
		{ID: "openai-error", RequestID: "req-2", Method: "POST", Path: "/v1/responses", Provider: "openai", Model: "gpt-4.1", StatusCode: 502, Error: "upstream failed"},
		{ID: "codex-new", RequestID: "req-3", Method: "POST", Path: "/v1/chat/completions", Provider: "codex", Model: "gpt-5.5", StatusCode: 200},
	}
	for _, entry := range entries {
		if _, err := store.Write(entry); err != nil {
			t.Fatalf("Write(%s) error = %v", entry.ID, err)
		}
	}

	statusOK := 200
	first, err := store.List(ListQuery{Limit: 1, Provider: "CODE", StatusCode: &statusOK})
	if err != nil {
		t.Fatalf("List(first) error = %v", err)
	}
	assertSummaryIDs(t, first.Entries, []string{"codex-new"})
	if first.NextCursor == "" {
		t.Fatalf("first filtered page NextCursor is empty")
	}

	second, err := store.List(ListQuery{Limit: 1, Provider: "code", StatusCode: &statusOK, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("List(second) error = %v", err)
	}
	assertSummaryIDs(t, second.Entries, []string{"codex-old"})
	if second.NextCursor != "" {
		t.Fatalf("second filtered page NextCursor = %q, want empty", second.NextCursor)
	}

	hasError := true
	errorsOnly, err := store.List(ListQuery{HasError: &hasError})
	if err != nil {
		t.Fatalf("List(errorsOnly) error = %v", err)
	}
	assertSummaryIDs(t, errorsOnly.Entries, []string{"openai-error"})
}

func TestStoreConcurrentWritesProduceReadableUniqueEntries(t *testing.T) {
	store := NewStore(Options{
		Enabled:           true,
		Directory:         t.TempDir(),
		MaxFileSizeBytes:  1024 * 1024,
		MaxTotalSizeBytes: 2 * 1024 * 1024,
		MaxEntryBytes:     4096,
	})

	const count = 32
	var wg sync.WaitGroup
	errCh := make(chan error, count)
	for i := 0; i < count; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.Write(Entry{
				ID:         fmt.Sprintf("concurrent-%02d", i),
				RequestID:  fmt.Sprintf("req-concurrent-%02d", i),
				Method:     "POST",
				Path:       "/v1/chat/completions",
				StatusCode: 200,
				Request:    Payload{Text: fmt.Sprintf("request-%02d", i)},
				Response:   Payload{Text: fmt.Sprintf("response-%02d", i)},
			})
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent Write() error = %v", err)
	}

	list, err := store.List(ListQuery{Limit: count})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list.Entries) != count {
		t.Fatalf("List() returned %d entries, want %d", len(list.Entries), count)
	}
	seen := map[string]bool{}
	for _, summary := range list.Entries {
		if seen[summary.ID] {
			t.Fatalf("duplicate summary id %q in %+v", summary.ID, list.Entries)
		}
		seen[summary.ID] = true
		entry, err := store.Read(summary.ID)
		if err != nil {
			t.Fatalf("Read(%s) error = %v", summary.ID, err)
		}
		if entry.ID != summary.ID {
			t.Fatalf("Read(%s) returned %q", summary.ID, entry.ID)
		}
	}
}

func TestStoreReadRejectsTraversalAndIgnoresNonConversationFiles(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(Options{
		Enabled:           true,
		Directory:         dir,
		MaxFileSizeBytes:  1024 * 1024,
		MaxTotalSizeBytes: 1024 * 1024,
		MaxEntryBytes:     4096,
	})
	if _, err := store.Write(Entry{ID: "safe-entry", RequestID: "req-safe"}); err != nil {
		t.Fatalf("Write(safe-entry) error = %v", err)
	}
	ignoredLine, err := json.Marshal(Entry{ID: "ignored-entry", CreatedAt: time.Date(2026, 5, 20, 10, 31, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("marshal ignored entry: %v", err)
	}
	ignoredFiles := []string{
		"not-conversation.jsonl",
		"conversation-20260520T103000.000000000Z-ignored.jsonl.bak",
	}
	for _, name := range ignoredFiles {
		if errWrite := os.WriteFile(filepath.Join(dir, name), append(ignoredLine, '\n'), 0o600); errWrite != nil {
			t.Fatalf("write ignored fixture %s: %v", name, errWrite)
		}
	}
	if errMkdir := os.Mkdir(filepath.Join(dir, "conversation-20260520T103000.000000000Z-directory.jsonl"), 0o700); errMkdir != nil {
		t.Fatalf("create ignored directory fixture: %v", errMkdir)
	}

	for _, id := range []string{"", "../safe-entry", `..\safe-entry`, "nested/safe-entry", `nested\safe-entry`} {
		if _, errRead := store.Read(id); !errors.Is(errRead, ErrNotFound) {
			t.Fatalf("Read(%q) error = %v, want ErrNotFound", id, errRead)
		}
	}
	list, err := store.List(ListQuery{Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	assertSummaryIDs(t, list.Entries, []string{"safe-entry"})
	if _, errRead := store.Read("ignored-entry"); !errors.Is(errRead, ErrNotFound) {
		t.Fatalf("Read(ignored-entry) error = %v, want ErrNotFound", errRead)
	}
}

func TestStoreRotationAndRetention(t *testing.T) {
	dir := t.TempDir()
	current := time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)
	store := NewStore(Options{
		Enabled:           true,
		Directory:         dir,
		MaxFileSizeBytes:  520,
		MaxTotalSizeBytes: 900,
		MaxEntryBytes:     4096,
	})
	store.SetNowForTest(func() time.Time {
		current = current.Add(time.Second)
		return current
	})

	var latestID string
	for i := 0; i < 5; i++ {
		location, err := store.Write(Entry{
			ID:        "entry-" + string(rune('a'+i)),
			RequestID: "req-retention",
			CreatedAt: current,
			Method:    "POST",
			Path:      "/v1/chat/completions",
			Request:   Payload{Text: strings.Repeat("request", 20)},
			Response:  Payload{Text: strings.Repeat("response", 20)},
		})
		if err != nil {
			t.Fatalf("Write(%d) error = %v", i, err)
		}
		latestID = location.ID
	}

	files, err := listConversationFiles(dir, false)
	if err != nil {
		t.Fatalf("listConversationFiles() error = %v", err)
	}
	if len(files) >= 5 {
		t.Fatalf("retention kept %d files, want fewer than all written files", len(files))
	}
	total := int64(0)
	for _, path := range files {
		info, errInfo := os.Stat(path)
		if errInfo != nil {
			t.Fatalf("stat retained file: %v", errInfo)
		}
		total += info.Size()
	}
	if total > 900 {
		t.Fatalf("retained size = %d, want <= 900", total)
	}
	if _, errRead := store.Read(latestID); errRead != nil {
		t.Fatalf("latest retained entry was not readable: %v", errRead)
	}
	if _, errRead := store.Read("entry-a"); !errors.Is(errRead, ErrNotFound) {
		t.Fatalf("oldest entry error = %v, want ErrNotFound after retention", errRead)
	}
}

func TestStoreRejectsOversizedEntries(t *testing.T) {
	store := NewStore(Options{
		Enabled:           true,
		Directory:         t.TempDir(),
		MaxFileSizeBytes:  1024,
		MaxTotalSizeBytes: 4096,
		MaxEntryBytes:     128,
	})

	_, err := store.Write(Entry{
		RequestID: "req-large",
		Request:   Payload{Text: strings.Repeat("x", 512)},
	})
	if !errors.Is(err, ErrEntryTooLarge) {
		t.Fatalf("Write() error = %v, want ErrEntryTooLarge", err)
	}
}

func TestRedactJSONRedactsNestedSensitiveKeys(t *testing.T) {
	raw := json.RawMessage(`{"messages":[{"role":"user","content":"visible","metadata":{"api_key":"request-secret"}}],"tools":[{"credential":"tool-secret"}],"count":2}`)

	redacted := RedactJSON(raw)

	if !json.Valid(redacted) {
		t.Fatalf("RedactJSON() returned invalid JSON: %s", redacted)
	}
	text := string(redacted)
	if strings.Contains(text, "request-secret") || strings.Contains(text, "tool-secret") {
		t.Fatalf("RedactJSON() leaked secret values: %s", text)
	}
	if !strings.Contains(text, "visible") || !strings.Contains(text, "[REDACTED]") {
		t.Fatalf("RedactJSON() lost visible content or redaction marker: %s", text)
	}
}

func assertSummaryIDs(t *testing.T, entries []EntrySummary, want []string) {
	t.Helper()
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(entries), len(want), entries)
	}
	for i, entry := range entries {
		if entry.ID != want[i] {
			t.Fatalf("entry[%d].ID = %q, want %q; entries=%+v", i, entry.ID, want[i], entries)
		}
	}
}

func assertPerm(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %v, want %v", path, got, want)
	}
}
