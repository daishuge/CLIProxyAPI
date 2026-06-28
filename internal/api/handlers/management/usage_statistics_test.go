package management

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	coreusage "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/usage"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/usage"
)

// newUsageStatsHandler returns a handler wired to a fresh statistics store with
// one recorded request for the supplied API key/model.
func newUsageStatsHandler(t *testing.T, apiKey, model string) (*Handler, *usage.RequestStatistics) {
	t.Helper()
	usage.SetStatisticsEnabled(true)
	stats := usage.NewRequestStatistics()
	stats.Record(context.Background(), coreusage.Record{
		APIKey:      apiKey,
		Model:       model,
		RequestedAt: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
		Latency:     1200 * time.Millisecond,
		TTFT:        200 * time.Millisecond,
		Detail: coreusage.Detail{
			InputTokens:  10,
			OutputTokens: 20,
			CachedTokens: 5,
			TotalTokens:  30,
		},
	})
	h := &Handler{}
	h.SetUsageStatistics(stats)
	return h, stats
}

func TestGetUsageStatisticsReturnsSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newUsageStatsHandler(t, "stats-key", "gpt-5.4")

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/usage-statistics", nil)
	h.GetUsageStatistics(ginCtx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body struct {
		Usage          usage.StatisticsSnapshot `json:"usage"`
		FailedRequests int64                    `json:"failed_requests"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Usage.TotalRequests != 1 {
		t.Fatalf("total_requests = %d, want 1", body.Usage.TotalRequests)
	}
	if _, ok := body.Usage.APIs["stats-key"]; !ok {
		t.Fatalf("snapshot missing api key entry: %s", rec.Body.String())
	}
}

func TestExportImportUsageStatisticsRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	src, _ := newUsageStatsHandler(t, "export-key", "gpt-5.5")

	exportRec := httptest.NewRecorder()
	exportCtx, _ := gin.CreateTestContext(exportRec)
	exportCtx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/usage-statistics/export", nil)
	src.ExportUsageStatistics(exportCtx)
	if exportRec.Code != http.StatusOK {
		t.Fatalf("export status = %d, want %d", exportRec.Code, http.StatusOK)
	}

	// Import the exported envelope into a fresh handler/store and confirm the
	// merge applied exactly one request detail.
	usage.SetStatisticsEnabled(true)
	dst := &Handler{}
	dst.SetUsageStatistics(usage.NewRequestStatistics())

	importRec := httptest.NewRecorder()
	importCtx, _ := gin.CreateTestContext(importRec)
	importCtx.Request = httptest.NewRequest(http.MethodPost, "/v0/management/usage-statistics/import", bytes.NewReader(exportRec.Body.Bytes()))
	importCtx.Request.Header.Set("Content-Type", "application/json")
	dst.ImportUsageStatistics(importCtx)

	if importRec.Code != http.StatusOK {
		t.Fatalf("import status = %d, want %d body=%s", importRec.Code, http.StatusOK, importRec.Body.String())
	}
	var importBody struct {
		Added         int64 `json:"added"`
		Skipped       int64 `json:"skipped"`
		TotalRequests int64 `json:"total_requests"`
	}
	if err := json.Unmarshal(importRec.Body.Bytes(), &importBody); err != nil {
		t.Fatalf("decode import response: %v", err)
	}
	if importBody.Added != 1 || importBody.Skipped != 0 || importBody.TotalRequests != 1 {
		t.Fatalf("import result = %+v, want added=1 skipped=0 total=1", importBody)
	}
}

func TestImportUsageStatisticsRejectsInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	usage.SetStatisticsEnabled(true)
	h := &Handler{}
	h.SetUsageStatistics(usage.NewRequestStatistics())

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/v0/management/usage-statistics/import", bytes.NewReader([]byte("not-json")))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	h.ImportUsageStatistics(ginCtx)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestImportUsageStatisticsUnavailableWithoutStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/v0/management/usage-statistics/import", bytes.NewReader([]byte(`{"usage":{}}`)))
	ginCtx.Request.Header.Set("Content-Type", "application/json")
	h.ImportUsageStatistics(ginCtx)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
