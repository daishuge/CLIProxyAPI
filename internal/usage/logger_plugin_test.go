package usage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	coreusage "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/usage"
)

func TestRequestStatisticsRecordIncludesLatency(t *testing.T) {
	SetStatisticsEnabled(true)
	stats := NewRequestStatistics()
	stats.Record(context.Background(), coreusage.Record{
		APIKey:      "test-key",
		Model:       "gpt-5.4",
		RequestedAt: time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
		Latency:     1500 * time.Millisecond,
		TTFT:        250 * time.Millisecond,
		Detail: coreusage.Detail{
			InputTokens:  10,
			OutputTokens: 20,
			CachedTokens: 5,
			TotalTokens:  30,
		},
	})

	snapshot := stats.Snapshot()
	details := snapshot.APIs["test-key"].Models["gpt-5.4"].Details
	if len(details) != 1 {
		t.Fatalf("details len = %d, want 1", len(details))
	}
	if details[0].LatencyMs != 1500 {
		t.Fatalf("latency_ms = %d, want 1500", details[0].LatencyMs)
	}
	if details[0].FirstByteLatencyMs != 250 {
		t.Fatalf("first_byte_latency_ms = %d, want 250", details[0].FirstByteLatencyMs)
	}
	if snapshot.TotalCachedTokens != 5 {
		t.Fatalf("total_cached_tokens = %d, want 5", snapshot.TotalCachedTokens)
	}
	if snapshot.CacheHitRate != 50 {
		t.Fatalf("cache_hit_rate = %f, want 50", snapshot.CacheHitRate)
	}
	if snapshot.AverageLatencyMs != 1500 {
		t.Fatalf("average_latency_ms = %d, want 1500", snapshot.AverageLatencyMs)
	}
	if snapshot.AverageFirstByteLatencyMs != 250 {
		t.Fatalf("average_first_byte_latency_ms = %d, want 250", snapshot.AverageFirstByteLatencyMs)
	}
	if snapshot.TPS != 1 {
		t.Fatalf("tps = %f, want 1", snapshot.TPS)
	}
}

func TestRequestStatisticsRecordSkippedWhenDisabled(t *testing.T) {
	SetStatisticsEnabled(false)
	t.Cleanup(func() { SetStatisticsEnabled(true) })

	stats := NewRequestStatistics()
	stats.Record(context.Background(), coreusage.Record{
		APIKey:      "disabled-key",
		Model:       "gpt-5.4",
		RequestedAt: time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
		Detail:      coreusage.Detail{InputTokens: 1, TotalTokens: 1},
	})

	snapshot := stats.Snapshot()
	if snapshot.TotalRequests != 0 {
		t.Fatalf("total_requests = %d, want 0 while disabled", snapshot.TotalRequests)
	}
}

func TestRequestStatisticsMergeSnapshotDedupIgnoresLatency(t *testing.T) {
	stats := NewRequestStatistics()
	timestamp := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	first := StatisticsSnapshot{
		APIs: map[string]APISnapshot{
			"test-key": {
				Models: map[string]ModelSnapshot{
					"gpt-5.4": {
						Details: []RequestDetail{{
							Timestamp: timestamp,
							LatencyMs: 0,
							Source:    "user@example.com",
							AuthIndex: "0",
							Tokens: TokenStats{
								InputTokens:  10,
								OutputTokens: 20,
								TotalTokens:  30,
							},
						}},
					},
				},
			},
		},
	}
	second := StatisticsSnapshot{
		APIs: map[string]APISnapshot{
			"test-key": {
				Models: map[string]ModelSnapshot{
					"gpt-5.4": {
						Details: []RequestDetail{{
							Timestamp: timestamp,
							LatencyMs: 2500,
							Source:    "user@example.com",
							AuthIndex: "0",
							Tokens: TokenStats{
								InputTokens:  10,
								OutputTokens: 20,
								TotalTokens:  30,
							},
						}},
					},
				},
			},
		},
	}

	result := stats.MergeSnapshot(first)
	if result.Added != 1 || result.Skipped != 0 {
		t.Fatalf("first merge = %+v, want added=1 skipped=0", result)
	}

	result = stats.MergeSnapshot(second)
	if result.Added != 0 || result.Skipped != 1 {
		t.Fatalf("second merge = %+v, want added=0 skipped=1", result)
	}

	snapshot := stats.Snapshot()
	details := snapshot.APIs["test-key"].Models["gpt-5.4"].Details
	if len(details) != 1 {
		t.Fatalf("details len = %d, want 1", len(details))
	}
}

func TestUsageStatisticsPersistenceRoundTrip(t *testing.T) {
	SetStatisticsEnabled(true)
	stats := NewRequestStatistics()
	stats.Record(context.Background(), coreusage.Record{
		APIKey:      "persist-key",
		Model:       "gpt-5.5",
		RequestedAt: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
		Latency:     1800 * time.Millisecond,
		TTFT:        300 * time.Millisecond,
		Detail: coreusage.Detail{
			InputTokens:  12,
			OutputTokens: 18,
			CachedTokens: 6,
			TotalTokens:  30,
		},
	})

	path := filepath.Join(t.TempDir(), "usage-statistics.json")
	if err := SaveStatisticsFile(path, stats.Snapshot()); err != nil {
		t.Fatalf("SaveStatisticsFile() error = %v", err)
	}

	restored := NewRequestStatistics()
	result, err := LoadStatisticsFile(restored, path)
	if err != nil {
		t.Fatalf("LoadStatisticsFile() error = %v", err)
	}
	if result.Added != 1 || result.Skipped != 0 {
		t.Fatalf("LoadStatisticsFile() result = %+v, want added=1 skipped=0", result)
	}

	snapshot := restored.Snapshot()
	if snapshot.TotalRequests != 1 || snapshot.TotalTokens != 30 || snapshot.TotalCachedTokens != 6 {
		t.Fatalf("snapshot totals = requests %d tokens %d cached %d", snapshot.TotalRequests, snapshot.TotalTokens, snapshot.TotalCachedTokens)
	}
	detail := snapshot.APIs["persist-key"].Models["gpt-5.5"].Details[0]
	if detail.FirstByteLatencyMs != 300 || detail.LatencyMs != 1800 {
		t.Fatalf("detail latency = %d first byte = %d", detail.LatencyMs, detail.FirstByteLatencyMs)
	}
}

func TestUsageStatisticsPersistenceFlushesOnStop(t *testing.T) {
	SetStatisticsEnabled(true)
	stats := NewRequestStatistics()
	path := filepath.Join(t.TempDir(), "usage-statistics.json")

	ctx, cancel := context.WithCancel(context.Background())
	stop := StartPersistence(ctx, stats, path, time.Hour)
	stats.Record(context.Background(), coreusage.Record{
		APIKey:      "stop-key",
		Model:       "gpt-5.5",
		RequestedAt: time.Date(2026, 5, 2, 13, 0, 0, 0, time.UTC),
		Detail: coreusage.Detail{
			InputTokens: 1,
			TotalTokens: 1,
		},
	})
	cancel()
	stop()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("persisted file stat error = %v", err)
	}

	restored := NewRequestStatistics()
	result, err := LoadStatisticsFile(restored, path)
	if err != nil {
		t.Fatalf("LoadStatisticsFile() error = %v", err)
	}
	if result.Added != 1 {
		t.Fatalf("LoadStatisticsFile() added = %d, want 1", result.Added)
	}
}
