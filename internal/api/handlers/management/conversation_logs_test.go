package management

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/logging/conversationlog"
)

func TestConversationLogManagementListDetailTailAndFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, store := newConversationLogManagementFixture(t, true)
	current := time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)
	store.SetNowForTest(func() time.Time {
		current = current.Add(time.Minute)
		return current
	})

	writeConversationLogEntry(t, store, conversationlog.Entry{
		ID:         "codex-old",
		RequestID:  "req-1",
		Method:     "POST",
		Path:       "/v1/chat/completions",
		Provider:   "codex",
		Model:      "gpt-5.5",
		StatusCode: 200,
		Response:   conversationlog.Payload{Text: "old response"},
	})
	writeConversationLogEntry(t, store, conversationlog.Entry{
		ID:         "openai-error",
		RequestID:  "req-2",
		Method:     "POST",
		Path:       "/v1/responses",
		Provider:   "openai",
		Model:      "gpt-4.1",
		StatusCode: 502,
		Error:      "upstream failed",
		Response:   conversationlog.Payload{Text: "error response"},
	})
	writeConversationLogEntry(t, store, conversationlog.Entry{
		ID:         "codex-new",
		RequestID:  "req-3",
		Method:     "POST",
		Path:       "/v1/chat/completions",
		Provider:   "codex",
		Model:      "gpt-5.5",
		StatusCode: 200,
		Response:   conversationlog.Payload{Text: "new response"},
	})

	first := performConversationLogManagementRequest(h.ListConversationLogs, "/v0/management/conversation-logs?limit=2", nil)
	if first.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d: %s", first.Code, http.StatusOK, first.Body.String())
	}
	firstPage := decodeConversationLogListResponse(t, first)
	assertConversationLogSummaryIDs(t, firstPage.Entries, []string{"codex-new", "openai-error"})
	if !firstPage.Enabled || firstPage.NextCursor == "" {
		t.Fatalf("first page enabled/cursor mismatch: %+v", firstPage)
	}

	second := performConversationLogManagementRequest(h.ListConversationLogs, "/v0/management/conversation-logs?limit=2&cursor="+firstPage.NextCursor, nil)
	if second.Code != http.StatusOK {
		t.Fatalf("second list status = %d, want %d: %s", second.Code, http.StatusOK, second.Body.String())
	}
	secondPage := decodeConversationLogListResponse(t, second)
	assertConversationLogSummaryIDs(t, secondPage.Entries, []string{"codex-old"})
	if secondPage.NextCursor != "" {
		t.Fatalf("second page NextCursor = %q, want empty", secondPage.NextCursor)
	}

	filtered := performConversationLogManagementRequest(h.ListConversationLogs, "/v0/management/conversation-logs?provider=codex&status_code=200", nil)
	if filtered.Code != http.StatusOK {
		t.Fatalf("filtered status = %d, want %d: %s", filtered.Code, http.StatusOK, filtered.Body.String())
	}
	filteredPage := decodeConversationLogListResponse(t, filtered)
	assertConversationLogSummaryIDs(t, filteredPage.Entries, []string{"codex-new", "codex-old"})

	errorsOnly := performConversationLogManagementRequest(h.ListConversationLogs, "/v0/management/conversation-logs?has_error=true", nil)
	if errorsOnly.Code != http.StatusOK {
		t.Fatalf("errors-only status = %d, want %d: %s", errorsOnly.Code, http.StatusOK, errorsOnly.Body.String())
	}
	errorsPage := decodeConversationLogListResponse(t, errorsOnly)
	assertConversationLogSummaryIDs(t, errorsPage.Entries, []string{"openai-error"})

	tail := performConversationLogManagementRequest(h.TailConversationLogs, "/v0/management/conversation-logs/tail?limit=1", nil)
	if tail.Code != http.StatusOK {
		t.Fatalf("tail status = %d, want %d: %s", tail.Code, http.StatusOK, tail.Body.String())
	}
	tailPage := decodeConversationLogListResponse(t, tail)
	if !tailPage.Tail {
		t.Fatalf("tail flag = false, want true: %+v", tailPage)
	}
	assertConversationLogSummaryIDs(t, tailPage.Entries, []string{"codex-new"})

	detail := performConversationLogManagementRequest(h.GetConversationLog, "/v0/management/conversation-logs/codex-new", gin.Params{{Key: "id", Value: "codex-new"}})
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d: %s", detail.Code, http.StatusOK, detail.Body.String())
	}
	detailBody := decodeConversationLogDetailResponse(t, detail)
	if !detailBody.Enabled || detailBody.Entry.ID != "codex-new" || detailBody.Entry.Response.Text != "new response" {
		t.Fatalf("detail body mismatch: %+v", detailBody)
	}
}

func TestConversationLogManagementRejectsBadFiltersAndMissingEntries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, store := newConversationLogManagementFixture(t, true)
	writeConversationLogEntry(t, store, conversationlog.Entry{ID: "safe-entry", RequestID: "req-safe"})

	badLimit := performConversationLogManagementRequest(h.ListConversationLogs, "/v0/management/conversation-logs?limit=0", nil)
	if badLimit.Code != http.StatusBadRequest {
		t.Fatalf("bad limit status = %d, want %d", badLimit.Code, http.StatusBadRequest)
	}
	badStatus := performConversationLogManagementRequest(h.ListConversationLogs, "/v0/management/conversation-logs?status_code=nope", nil)
	if badStatus.Code != http.StatusBadRequest {
		t.Fatalf("bad status filter status = %d, want %d", badStatus.Code, http.StatusBadRequest)
	}
	badCursor := performConversationLogManagementRequest(h.ListConversationLogs, "/v0/management/conversation-logs?cursor=-1", nil)
	if badCursor.Code != http.StatusBadRequest {
		t.Fatalf("bad cursor status = %d, want %d", badCursor.Code, http.StatusBadRequest)
	}
	badRange := performConversationLogManagementRequest(h.ListConversationLogs, "/v0/management/conversation-logs?from=2026-05-20T11:00:00Z&to=2026-05-20T10:00:00Z", nil)
	if badRange.Code != http.StatusBadRequest {
		t.Fatalf("bad range status = %d, want %d", badRange.Code, http.StatusBadRequest)
	}

	missing := performConversationLogManagementRequest(h.GetConversationLog, "/v0/management/conversation-logs/missing", gin.Params{{Key: "id", Value: "missing"}})
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missing.Code, http.StatusNotFound)
	}
	traversal := performConversationLogManagementRequest(h.GetConversationLog, "/v0/management/conversation-logs/../safe-entry", gin.Params{{Key: "id", Value: "../safe-entry"}})
	if traversal.Code != http.StatusNotFound {
		t.Fatalf("traversal status = %d, want %d", traversal.Code, http.StatusNotFound)
	}
}

func TestConversationLogManagementHandlesDisabledAndLargeEntries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	disabled, _ := newConversationLogManagementFixture(t, false)
	disabledList := performConversationLogManagementRequest(disabled.ListConversationLogs, "/v0/management/conversation-logs", nil)
	if disabledList.Code != http.StatusOK {
		t.Fatalf("disabled list status = %d, want %d: %s", disabledList.Code, http.StatusOK, disabledList.Body.String())
	}
	disabledBody := decodeConversationLogListResponse(t, disabledList)
	if disabledBody.Enabled || len(disabledBody.Entries) != 0 {
		t.Fatalf("disabled body mismatch: %+v", disabledBody)
	}

	h, store := newConversationLogManagementFixture(t, true)
	largeText := strings.Repeat("large-entry-", 8192)
	writeConversationLogEntry(t, store, conversationlog.Entry{
		ID:         "large-entry",
		RequestID:  "req-large",
		Method:     "POST",
		Path:       "/v1/chat/completions",
		StatusCode: 200,
		Response:   conversationlog.Payload{Text: largeText},
	})
	detail := performConversationLogManagementRequest(h.GetConversationLog, "/v0/management/conversation-logs/large-entry", gin.Params{{Key: "id", Value: "large-entry"}})
	if detail.Code != http.StatusOK {
		t.Fatalf("large detail status = %d, want %d: %s", detail.Code, http.StatusOK, detail.Body.String())
	}
	detailBody := decodeConversationLogDetailResponse(t, detail)
	if detailBody.Entry.Response.Text != largeText {
		t.Fatalf("large response text length = %d, want %d", len(detailBody.Entry.Response.Text), len(largeText))
	}
}

type conversationLogListResponse struct {
	Enabled    bool                           `json:"enabled"`
	Entries    []conversationlog.EntrySummary `json:"entries"`
	NextCursor string                         `json:"next_cursor"`
	Malformed  int                            `json:"malformed"`
	Tail       bool                           `json:"tail"`
}

type conversationLogDetailResponse struct {
	Enabled bool                  `json:"enabled"`
	Entry   conversationlog.Entry `json:"entry"`
}

func newConversationLogManagementFixture(t *testing.T, enabled bool) (*Handler, *conversationlog.Store) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "conversation-logs")
	store := conversationlog.NewStore(conversationlog.Options{
		Enabled:           enabled,
		Directory:         dir,
		MaxFileSizeBytes:  1024 * 1024,
		MaxTotalSizeBytes: 4 * 1024 * 1024,
		MaxEntryBytes:     512 * 1024,
	})
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, nil)
	h.SetConversationLogStore(store)
	return h, store
}

func writeConversationLogEntry(t *testing.T, store *conversationlog.Store, entry conversationlog.Entry) {
	t.Helper()
	if _, err := store.Write(entry); err != nil {
		t.Fatalf("Write(%s) error = %v", entry.ID, err)
	}
}

func performConversationLogManagementRequest(handler gin.HandlerFunc, target string, params gin.Params) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, target, nil)
	ginCtx.Params = params
	handler(ginCtx)
	return rec
}

func decodeConversationLogListResponse(t *testing.T, rec *httptest.ResponseRecorder) conversationLogListResponse {
	t.Helper()
	var body conversationLogListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode list response: %v; body=%s", err, rec.Body.String())
	}
	return body
}

func decodeConversationLogDetailResponse(t *testing.T, rec *httptest.ResponseRecorder) conversationLogDetailResponse {
	t.Helper()
	var body conversationLogDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode detail response: %v; body=%s", err, rec.Body.String())
	}
	return body
}

func assertConversationLogSummaryIDs(t *testing.T, entries []conversationlog.EntrySummary, want []string) {
	t.Helper()
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(entries), len(want), entries)
	}
	for i, entry := range entries {
		if entry.ID != want[i] {
			t.Fatalf("entries[%d].ID = %q, want %q; entries=%+v", i, entry.ID, want[i], entries)
		}
	}
}
