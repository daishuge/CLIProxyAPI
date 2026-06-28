// Package usage provides usage tracking and logging functionality for the CLI Proxy API server.
// It includes plugins for monitoring API usage, token consumption, and other metrics
// to help with observability and billing purposes.
//
// This is the PPAP private "rich usage analytics" sink. It coexists with the
// upstream v7.2 native usage queue (internal/redisqueue): both register on the
// shared coreusage.DefaultManager and receive the same records. Recording is
// gated by the `usage-statistics-enabled` config flag (default false) so the
// plugin imposes zero aggregation overhead unless explicitly enabled.
package usage

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	internallogging "github.com/router-for-me/CLIProxyAPI/v7/internal/logging"
	coreusage "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/usage"
)

var statisticsEnabled atomic.Bool

func init() {
	// Recording starts enabled at the package level so the in-process tests and
	// direct Record() callers observe aggregation without extra setup. The
	// server overrides this from the `usage-statistics-enabled` config flag
	// (default false) during startup via SetStatisticsEnabled, keeping the hot
	// path zero-overhead in production unless the operator opts in.
	statisticsEnabled.Store(true)
	coreusage.RegisterPlugin(NewLoggerPlugin())
}

// LoggerPlugin collects in-memory request statistics for usage analysis.
// It implements coreusage.Plugin to receive usage records emitted by the runtime.
type LoggerPlugin struct {
	stats *RequestStatistics
}

// NewLoggerPlugin constructs a new logger plugin instance.
//
// Returns:
//   - *LoggerPlugin: A new logger plugin instance wired to the shared statistics store.
func NewLoggerPlugin() *LoggerPlugin { return &LoggerPlugin{stats: defaultRequestStatistics} }

// HandleUsage implements coreusage.Plugin.
// It updates the in-memory statistics store whenever a usage record is received.
//
// Parameters:
//   - ctx: The context for the usage record
//   - record: The usage record to aggregate
func (p *LoggerPlugin) HandleUsage(ctx context.Context, record coreusage.Record) {
	if !statisticsEnabled.Load() {
		return
	}
	if p == nil || p.stats == nil {
		return
	}
	p.stats.Record(ctx, record)
}

// SetStatisticsEnabled toggles whether in-memory statistics are recorded.
func SetStatisticsEnabled(enabled bool) { statisticsEnabled.Store(enabled) }

// StatisticsEnabled reports the current recording state.
func StatisticsEnabled() bool { return statisticsEnabled.Load() }

// RequestStatistics maintains aggregated request metrics in memory.
type RequestStatistics struct {
	mu sync.RWMutex

	totalRequests     int64
	successCount      int64
	failureCount      int64
	totalTokens       int64
	totalInputTokens  int64
	totalCachedTokens int64

	totalLatencyMs          int64
	latencySamples          int64
	totalFirstByteLatencyMs int64
	firstByteSamples        int64
	firstRequestAt          time.Time
	lastRequestAt           time.Time

	apis map[string]*apiStats

	requestsByDay  map[string]int64
	requestsByHour map[int]int64
	tokensByDay    map[string]int64
	tokensByHour   map[int]int64
}

// apiStats holds aggregated metrics for a single API key.
type apiStats struct {
	TotalRequests           int64
	TotalTokens             int64
	TotalInputTokens        int64
	TotalCachedTokens       int64
	TotalLatencyMs          int64
	LatencySamples          int64
	TotalFirstByteLatencyMs int64
	FirstByteLatencySamples int64
	FirstRequestAt          time.Time
	LastRequestAt           time.Time
	Models                  map[string]*modelStats
}

// modelStats holds aggregated metrics for a specific model within an API.
type modelStats struct {
	TotalRequests           int64
	TotalTokens             int64
	TotalInputTokens        int64
	TotalCachedTokens       int64
	TotalLatencyMs          int64
	LatencySamples          int64
	TotalFirstByteLatencyMs int64
	FirstByteLatencySamples int64
	FirstRequestAt          time.Time
	LastRequestAt           time.Time
	Details                 []RequestDetail
}

// RequestDetail stores the timestamp, latency, and token usage for a single request.
type RequestDetail struct {
	Timestamp          time.Time  `json:"timestamp"`
	LatencyMs          int64      `json:"latency_ms"`
	FirstByteLatencyMs int64      `json:"first_byte_latency_ms"`
	Source             string     `json:"source"`
	AuthIndex          string     `json:"auth_index"`
	Tokens             TokenStats `json:"tokens"`
	Failed             bool       `json:"failed"`
}

// TokenStats captures the token usage breakdown for a request.
type TokenStats struct {
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	ReasoningTokens int64 `json:"reasoning_tokens"`
	CachedTokens    int64 `json:"cached_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
}

// StatisticsSnapshot represents an immutable view of the aggregated metrics.
type StatisticsSnapshot struct {
	TotalRequests             int64   `json:"total_requests"`
	SuccessCount              int64   `json:"success_count"`
	FailureCount              int64   `json:"failure_count"`
	TotalTokens               int64   `json:"total_tokens"`
	TotalInputTokens          int64   `json:"total_input_tokens"`
	TotalCachedTokens         int64   `json:"total_cached_tokens"`
	CacheHitRate              float64 `json:"cache_hit_rate"`
	AverageLatencyMs          int64   `json:"average_latency_ms"`
	AverageFirstByteLatencyMs int64   `json:"average_first_byte_latency_ms"`
	TPS                       float64 `json:"tps"`

	APIs map[string]APISnapshot `json:"apis"`

	RequestsByDay  map[string]int64 `json:"requests_by_day"`
	RequestsByHour map[string]int64 `json:"requests_by_hour"`
	TokensByDay    map[string]int64 `json:"tokens_by_day"`
	TokensByHour   map[string]int64 `json:"tokens_by_hour"`
}

// APISnapshot summarises metrics for a single API key.
type APISnapshot struct {
	TotalRequests             int64                    `json:"total_requests"`
	TotalTokens               int64                    `json:"total_tokens"`
	TotalInputTokens          int64                    `json:"total_input_tokens"`
	TotalCachedTokens         int64                    `json:"total_cached_tokens"`
	CacheHitRate              float64                  `json:"cache_hit_rate"`
	AverageLatencyMs          int64                    `json:"average_latency_ms"`
	AverageFirstByteLatencyMs int64                    `json:"average_first_byte_latency_ms"`
	TPS                       float64                  `json:"tps"`
	Models                    map[string]ModelSnapshot `json:"models"`
}

// ModelSnapshot summarises metrics for a specific model.
type ModelSnapshot struct {
	TotalRequests             int64           `json:"total_requests"`
	TotalTokens               int64           `json:"total_tokens"`
	TotalInputTokens          int64           `json:"total_input_tokens"`
	TotalCachedTokens         int64           `json:"total_cached_tokens"`
	CacheHitRate              float64         `json:"cache_hit_rate"`
	AverageLatencyMs          int64           `json:"average_latency_ms"`
	AverageFirstByteLatencyMs int64           `json:"average_first_byte_latency_ms"`
	TPS                       float64         `json:"tps"`
	Details                   []RequestDetail `json:"details"`
}

var defaultRequestStatistics = NewRequestStatistics()

// GetRequestStatistics returns the shared statistics store.
func GetRequestStatistics() *RequestStatistics { return defaultRequestStatistics }

// NewRequestStatistics constructs an empty statistics store.
func NewRequestStatistics() *RequestStatistics {
	return &RequestStatistics{
		apis:           make(map[string]*apiStats),
		requestsByDay:  make(map[string]int64),
		requestsByHour: make(map[int]int64),
		tokensByDay:    make(map[string]int64),
		tokensByHour:   make(map[int]int64),
	}
}

// Record ingests a new usage record and updates the aggregates.
func (s *RequestStatistics) Record(ctx context.Context, record coreusage.Record) {
	if s == nil {
		return
	}
	if !statisticsEnabled.Load() {
		return
	}
	timestamp := record.RequestedAt
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	detail := normaliseDetail(record.Detail)
	totalTokens := detail.TotalTokens
	statsKey := record.APIKey
	if statsKey == "" {
		statsKey = resolveAPIIdentifier(ctx, record)
	}
	failed := record.Failed
	if !failed {
		failed = !resolveSuccess(ctx)
	}
	success := !failed
	modelName := record.Model
	if modelName == "" {
		modelName = "unknown"
	}
	dayKey := timestamp.Format("2006-01-02")
	hourKey := timestamp.Hour()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.totalRequests++
	if success {
		s.successCount++
	} else {
		s.failureCount++
	}
	s.totalTokens += totalTokens
	s.totalInputTokens += detail.InputTokens
	s.totalCachedTokens += detail.CachedTokens
	latencyMs := normaliseLatency(record.Latency)
	// v7.2 renamed the record's time-to-first-byte field to TTFT (was
	// FirstByteLatency in the PPAP source). Aggregate from the upstream field.
	firstByteLatencyMs := normaliseLatency(record.TTFT)
	if latencyMs > 0 {
		s.totalLatencyMs += latencyMs
		s.latencySamples++
	}
	if firstByteLatencyMs > 0 {
		s.totalFirstByteLatencyMs += firstByteLatencyMs
		s.firstByteSamples++
	}
	updateTimeWindow(&s.firstRequestAt, &s.lastRequestAt, timestamp)

	stats, ok := s.apis[statsKey]
	if !ok {
		stats = &apiStats{Models: make(map[string]*modelStats)}
		s.apis[statsKey] = stats
	}
	s.updateAPIStats(stats, modelName, RequestDetail{
		Timestamp:          timestamp,
		LatencyMs:          latencyMs,
		FirstByteLatencyMs: firstByteLatencyMs,
		Source:             record.Source,
		AuthIndex:          record.AuthIndex,
		Tokens:             detail,
		Failed:             failed,
	})

	s.requestsByDay[dayKey]++
	s.requestsByHour[hourKey]++
	s.tokensByDay[dayKey] += totalTokens
	s.tokensByHour[hourKey] += totalTokens
}

func (s *RequestStatistics) updateAPIStats(stats *apiStats, model string, detail RequestDetail) {
	stats.TotalRequests++
	stats.TotalTokens += detail.Tokens.TotalTokens
	stats.TotalInputTokens += detail.Tokens.InputTokens
	stats.TotalCachedTokens += detail.Tokens.CachedTokens
	if detail.LatencyMs > 0 {
		stats.TotalLatencyMs += detail.LatencyMs
		stats.LatencySamples++
	}
	if detail.FirstByteLatencyMs > 0 {
		stats.TotalFirstByteLatencyMs += detail.FirstByteLatencyMs
		stats.FirstByteLatencySamples++
	}
	updateTimeWindow(&stats.FirstRequestAt, &stats.LastRequestAt, detail.Timestamp)

	modelStatsValue, ok := stats.Models[model]
	if !ok {
		modelStatsValue = &modelStats{}
		stats.Models[model] = modelStatsValue
	}
	modelStatsValue.TotalRequests++
	modelStatsValue.TotalTokens += detail.Tokens.TotalTokens
	modelStatsValue.TotalInputTokens += detail.Tokens.InputTokens
	modelStatsValue.TotalCachedTokens += detail.Tokens.CachedTokens
	if detail.LatencyMs > 0 {
		modelStatsValue.TotalLatencyMs += detail.LatencyMs
		modelStatsValue.LatencySamples++
	}
	if detail.FirstByteLatencyMs > 0 {
		modelStatsValue.TotalFirstByteLatencyMs += detail.FirstByteLatencyMs
		modelStatsValue.FirstByteLatencySamples++
	}
	updateTimeWindow(&modelStatsValue.FirstRequestAt, &modelStatsValue.LastRequestAt, detail.Timestamp)
	modelStatsValue.Details = append(modelStatsValue.Details, detail)
}

// Snapshot returns a copy of the aggregated metrics for external consumption.
func (s *RequestStatistics) Snapshot() StatisticsSnapshot {
	result := StatisticsSnapshot{}
	if s == nil {
		return result
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result.TotalRequests = s.totalRequests
	result.SuccessCount = s.successCount
	result.FailureCount = s.failureCount
	result.TotalTokens = s.totalTokens
	result.TotalInputTokens = s.totalInputTokens
	result.TotalCachedTokens = s.totalCachedTokens
	result.CacheHitRate = cacheHitRate(s.totalCachedTokens, s.totalInputTokens)
	result.AverageLatencyMs = averageMs(s.totalLatencyMs, s.latencySamples)
	result.AverageFirstByteLatencyMs = averageMs(s.totalFirstByteLatencyMs, s.firstByteSamples)
	result.TPS = throughputPerSecond(s.totalRequests, s.firstRequestAt, s.lastRequestAt)

	result.APIs = make(map[string]APISnapshot, len(s.apis))
	for apiName, stats := range s.apis {
		apiSnapshot := APISnapshot{
			TotalRequests:             stats.TotalRequests,
			TotalTokens:               stats.TotalTokens,
			TotalInputTokens:          stats.TotalInputTokens,
			TotalCachedTokens:         stats.TotalCachedTokens,
			CacheHitRate:              cacheHitRate(stats.TotalCachedTokens, stats.TotalInputTokens),
			AverageLatencyMs:          averageMs(stats.TotalLatencyMs, stats.LatencySamples),
			AverageFirstByteLatencyMs: averageMs(stats.TotalFirstByteLatencyMs, stats.FirstByteLatencySamples),
			TPS:                       throughputPerSecond(stats.TotalRequests, stats.FirstRequestAt, stats.LastRequestAt),
			Models:                    make(map[string]ModelSnapshot, len(stats.Models)),
		}
		for modelName, modelStatsValue := range stats.Models {
			requestDetails := make([]RequestDetail, len(modelStatsValue.Details))
			copy(requestDetails, modelStatsValue.Details)
			apiSnapshot.Models[modelName] = ModelSnapshot{
				TotalRequests:             modelStatsValue.TotalRequests,
				TotalTokens:               modelStatsValue.TotalTokens,
				TotalInputTokens:          modelStatsValue.TotalInputTokens,
				TotalCachedTokens:         modelStatsValue.TotalCachedTokens,
				CacheHitRate:              cacheHitRate(modelStatsValue.TotalCachedTokens, modelStatsValue.TotalInputTokens),
				AverageLatencyMs:          averageMs(modelStatsValue.TotalLatencyMs, modelStatsValue.LatencySamples),
				AverageFirstByteLatencyMs: averageMs(modelStatsValue.TotalFirstByteLatencyMs, modelStatsValue.FirstByteLatencySamples),
				TPS:                       throughputPerSecond(modelStatsValue.TotalRequests, modelStatsValue.FirstRequestAt, modelStatsValue.LastRequestAt),
				Details:                   requestDetails,
			}
		}
		result.APIs[apiName] = apiSnapshot
	}

	result.RequestsByDay = make(map[string]int64, len(s.requestsByDay))
	for k, v := range s.requestsByDay {
		result.RequestsByDay[k] = v
	}

	result.RequestsByHour = make(map[string]int64, len(s.requestsByHour))
	for hour, v := range s.requestsByHour {
		key := formatHour(hour)
		result.RequestsByHour[key] = v
	}

	result.TokensByDay = make(map[string]int64, len(s.tokensByDay))
	for k, v := range s.tokensByDay {
		result.TokensByDay[k] = v
	}

	result.TokensByHour = make(map[string]int64, len(s.tokensByHour))
	for hour, v := range s.tokensByHour {
		key := formatHour(hour)
		result.TokensByHour[key] = v
	}

	return result
}

// MergeResult reports how many imported request details were applied or skipped
// as duplicates during a snapshot merge.
type MergeResult struct {
	Added   int64 `json:"added"`
	Skipped int64 `json:"skipped"`
}

// MergeSnapshot merges an exported statistics snapshot into the current store.
// Existing data is preserved and duplicate request details are skipped.
func (s *RequestStatistics) MergeSnapshot(snapshot StatisticsSnapshot) MergeResult {
	result := MergeResult{}
	if s == nil {
		return result
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]struct{})
	for apiName, stats := range s.apis {
		if stats == nil {
			continue
		}
		for modelName, modelStatsValue := range stats.Models {
			if modelStatsValue == nil {
				continue
			}
			for _, detail := range modelStatsValue.Details {
				seen[dedupKey(apiName, modelName, detail)] = struct{}{}
			}
		}
	}

	for apiName, apiSnapshot := range snapshot.APIs {
		apiName = strings.TrimSpace(apiName)
		if apiName == "" {
			continue
		}
		stats, ok := s.apis[apiName]
		if !ok || stats == nil {
			stats = &apiStats{Models: make(map[string]*modelStats)}
			s.apis[apiName] = stats
		} else if stats.Models == nil {
			stats.Models = make(map[string]*modelStats)
		}
		for modelName, modelSnapshot := range apiSnapshot.Models {
			modelName = strings.TrimSpace(modelName)
			if modelName == "" {
				modelName = "unknown"
			}
			for _, detail := range modelSnapshot.Details {
				detail.Tokens = normaliseTokenStats(detail.Tokens)
				if detail.LatencyMs < 0 {
					detail.LatencyMs = 0
				}
				if detail.FirstByteLatencyMs < 0 {
					detail.FirstByteLatencyMs = 0
				}
				if detail.Timestamp.IsZero() {
					detail.Timestamp = time.Now()
				}
				key := dedupKey(apiName, modelName, detail)
				if _, exists := seen[key]; exists {
					result.Skipped++
					continue
				}
				seen[key] = struct{}{}
				s.recordImported(apiName, modelName, stats, detail)
				result.Added++
			}
		}
	}

	return result
}

func (s *RequestStatistics) recordImported(apiName, modelName string, stats *apiStats, detail RequestDetail) {
	totalTokens := detail.Tokens.TotalTokens
	if totalTokens < 0 {
		totalTokens = 0
	}

	s.totalRequests++
	if detail.Failed {
		s.failureCount++
	} else {
		s.successCount++
	}
	s.totalTokens += totalTokens
	s.totalInputTokens += detail.Tokens.InputTokens
	s.totalCachedTokens += detail.Tokens.CachedTokens
	if detail.LatencyMs > 0 {
		s.totalLatencyMs += detail.LatencyMs
		s.latencySamples++
	}
	if detail.FirstByteLatencyMs > 0 {
		s.totalFirstByteLatencyMs += detail.FirstByteLatencyMs
		s.firstByteSamples++
	}
	updateTimeWindow(&s.firstRequestAt, &s.lastRequestAt, detail.Timestamp)

	s.updateAPIStats(stats, modelName, detail)

	dayKey := detail.Timestamp.Format("2006-01-02")
	hourKey := detail.Timestamp.Hour()

	s.requestsByDay[dayKey]++
	s.requestsByHour[hourKey]++
	s.tokensByDay[dayKey] += totalTokens
	s.tokensByHour[hourKey] += totalTokens
}

func dedupKey(apiName, modelName string, detail RequestDetail) string {
	timestamp := detail.Timestamp.UTC().Format(time.RFC3339Nano)
	tokens := normaliseTokenStats(detail.Tokens)
	return fmt.Sprintf(
		"%s|%s|%s|%s|%s|%t|%d|%d|%d|%d|%d",
		apiName,
		modelName,
		timestamp,
		detail.Source,
		detail.AuthIndex,
		detail.Failed,
		tokens.InputTokens,
		tokens.OutputTokens,
		tokens.ReasoningTokens,
		tokens.CachedTokens,
		tokens.TotalTokens,
	)
}

func resolveAPIIdentifier(ctx context.Context, record coreusage.Record) string {
	if ctx != nil {
		if endpoint := strings.TrimSpace(internallogging.GetEndpoint(ctx)); endpoint != "" {
			return endpoint
		}
	}
	if record.Provider != "" {
		return record.Provider
	}
	return "unknown"
}

func resolveSuccess(ctx context.Context) bool {
	status := internallogging.GetResponseStatus(ctx)
	if status == 0 {
		return true
	}
	return status < httpStatusBadRequest
}

const httpStatusBadRequest = 400

func normaliseDetail(detail coreusage.Detail) TokenStats {
	tokens := TokenStats{
		InputTokens:     detail.InputTokens,
		OutputTokens:    detail.OutputTokens,
		ReasoningTokens: detail.ReasoningTokens,
		CachedTokens:    detail.CachedTokens,
		TotalTokens:     detail.TotalTokens,
	}
	if tokens.TotalTokens == 0 {
		tokens.TotalTokens = detail.InputTokens + detail.OutputTokens + detail.ReasoningTokens
	}
	if tokens.TotalTokens == 0 {
		tokens.TotalTokens = detail.InputTokens + detail.OutputTokens + detail.ReasoningTokens + detail.CachedTokens
	}
	return tokens
}

func normaliseTokenStats(tokens TokenStats) TokenStats {
	if tokens.TotalTokens == 0 {
		tokens.TotalTokens = tokens.InputTokens + tokens.OutputTokens + tokens.ReasoningTokens
	}
	if tokens.TotalTokens == 0 {
		tokens.TotalTokens = tokens.InputTokens + tokens.OutputTokens + tokens.ReasoningTokens + tokens.CachedTokens
	}
	return tokens
}

func normaliseLatency(latency time.Duration) int64 {
	if latency <= 0 {
		return 0
	}
	return latency.Milliseconds()
}

func updateTimeWindow(first, last *time.Time, timestamp time.Time) {
	if first == nil || last == nil || timestamp.IsZero() {
		return
	}
	if first.IsZero() || timestamp.Before(*first) {
		*first = timestamp
	}
	if last.IsZero() || timestamp.After(*last) {
		*last = timestamp
	}
}

func cacheHitRate(cachedTokens, inputTokens int64) float64 {
	if inputTokens <= 0 || cachedTokens <= 0 {
		return 0
	}
	return float64(cachedTokens) / float64(inputTokens) * 100
}

func averageMs(totalMs, samples int64) int64 {
	if totalMs <= 0 || samples <= 0 {
		return 0
	}
	return totalMs / samples
}

func throughputPerSecond(totalRequests int64, first, last time.Time) float64 {
	if totalRequests <= 0 || first.IsZero() || last.IsZero() {
		return 0
	}
	elapsed := last.Sub(first).Seconds()
	if elapsed <= 0 {
		return float64(totalRequests)
	}
	if elapsed < 1 {
		elapsed = 1
	}
	return float64(totalRequests) / elapsed
}

func formatHour(hour int) string {
	if hour < 0 {
		hour = 0
	}
	hour = hour % 24
	return fmt.Sprintf("%02d", hour)
}
