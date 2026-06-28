package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// DefaultPersistenceInterval is the fallback flush cadence used when the
// configured interval is non-positive.
const DefaultPersistenceInterval = 30 * time.Second

type persistedStatistics struct {
	Version int                `json:"version"`
	SavedAt time.Time          `json:"saved_at"`
	Usage   StatisticsSnapshot `json:"usage"`
}

var (
	persistenceMu   sync.RWMutex
	persistencePath string
)

// StartPersistence loads any existing snapshot from path into stats, then runs a
// background goroutine that flushes the in-memory snapshot to path on the given
// interval and once more on shutdown. It returns a stop function that cancels
// the goroutine after a final flush. When statistics are disabled, the path is
// empty, or stats is nil, it is a no-op returning an empty stop function.
func StartPersistence(ctx context.Context, stats *RequestStatistics, path string, interval time.Duration) func() {
	path = strings.TrimSpace(path)
	if path == "" || stats == nil || !StatisticsEnabled() {
		return func() {}
	}
	if interval <= 0 {
		interval = DefaultPersistenceInterval
	}

	persistenceMu.Lock()
	persistencePath = path
	persistenceMu.Unlock()

	if result, err := LoadStatisticsFile(stats, path); err != nil {
		log.Warnf("usage statistics persistence load failed from %s: %v", path, err)
	} else if result.Added > 0 || result.Skipped > 0 {
		log.Infof("usage statistics persistence loaded from %s: added=%d skipped=%d", path, result.Added, result.Skipped)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	stopCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var once sync.Once

	save := func() {
		if err := SaveStatisticsFile(path, stats.Snapshot()); err != nil {
			log.Warnf("usage statistics persistence save failed to %s: %v", path, err)
		}
	}

	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				save()
			case <-ctx.Done():
				save()
				return
			case <-stopCtx.Done():
				save()
				return
			}
		}
	}()

	return func() {
		once.Do(func() {
			cancel()
			<-done
		})
	}
}

// SaveConfiguredStatistics flushes the shared statistics store to the path
// registered by the most recent StartPersistence call. It is a no-op when no
// path has been configured.
func SaveConfiguredStatistics() error {
	persistenceMu.RLock()
	path := persistencePath
	persistenceMu.RUnlock()
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return SaveStatisticsFile(path, GetRequestStatistics().Snapshot())
}

// LoadStatisticsFile reads a persisted snapshot from path and merges it into
// stats. A missing or empty file is treated as success with an empty result.
func LoadStatisticsFile(stats *RequestStatistics, path string) (MergeResult, error) {
	if stats == nil {
		return MergeResult{}, nil
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return MergeResult{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return MergeResult{}, nil
		}
		return MergeResult{}, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return MergeResult{}, nil
	}

	var payload persistedStatistics
	if err := json.Unmarshal(data, &payload); err == nil && hasSnapshotData(payload.Usage) {
		return stats.MergeSnapshot(payload.Usage), nil
	}

	var snapshot StatisticsSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return MergeResult{}, fmt.Errorf("invalid usage statistics json: %w", err)
	}
	return stats.MergeSnapshot(snapshot), nil
}

// SaveStatisticsFile atomically writes snapshot to path, creating the parent
// directory if needed and replacing the destination via a temp-file rename.
func SaveStatisticsFile(path string, snapshot StatisticsSnapshot) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	payload := persistedStatistics{
		Version: 1,
		SavedAt: time.Now().UTC(),
		Usage:   snapshot,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := filepath.Join(dir, fmt.Sprintf(".%s.tmp.%d", filepath.Base(path), os.Getpid()))
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(path)
		if errRename := os.Rename(tmp, path); errRename != nil {
			_ = os.Remove(tmp)
			return errRename
		}
	}
	return nil
}

func hasSnapshotData(snapshot StatisticsSnapshot) bool {
	if snapshot.TotalRequests > 0 || snapshot.SuccessCount > 0 || snapshot.FailureCount > 0 {
		return true
	}
	return len(snapshot.APIs) > 0
}
