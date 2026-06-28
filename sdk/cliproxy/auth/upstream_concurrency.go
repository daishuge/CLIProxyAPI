package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	internalconfig "github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	log "github.com/sirupsen/logrus"
)

// upstreamConcurrencyLimiter bounds the number of in-flight upstream requests for a single key.
type upstreamConcurrencyLimiter struct {
	key   string
	limit int
	sem   chan struct{}
}

// upstreamConcurrencyPermit represents a single acquired slot on a limiter.
type upstreamConcurrencyPermit struct {
	limiter *upstreamConcurrencyLimiter
}

// Release returns the acquired slot to the limiter. It is safe to call on a nil permit.
func (p *upstreamConcurrencyPermit) Release() {
	if p == nil || p.limiter == nil {
		return
	}
	select {
	case <-p.limiter.sem:
	default:
		log.WithField("key", p.limiter.key).Warn("upstream concurrency permit release without acquire")
	}
}

// upstreamConcurrencyConfig returns the current upstream concurrency configuration snapshot.
func (m *Manager) upstreamConcurrencyConfig() internalconfig.UpstreamConcurrencyConfig {
	if m == nil {
		return internalconfig.UpstreamConcurrencyConfig{}
	}
	cfg, _ := m.runtimeConfig.Load().(*internalconfig.Config)
	if cfg == nil {
		return internalconfig.UpstreamConcurrencyConfig{}
	}
	return cfg.UpstreamConcurrency
}

// upstreamLimiterFor returns (creating if necessary) the limiter for the given provider and limit.
// When the configured limit changes, the limiter is rebuilt to honour the new bound.
func (m *Manager) upstreamLimiterFor(provider string, limit int) *upstreamConcurrencyLimiter {
	if m == nil || limit <= 0 {
		return nil
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return nil
	}
	key := "provider:" + provider

	m.upstreamLimitersMu.Lock()
	defer m.upstreamLimitersMu.Unlock()
	if m.upstreamLimiters == nil {
		m.upstreamLimiters = make(map[string]*upstreamConcurrencyLimiter)
	}
	limiter := m.upstreamLimiters[key]
	if limiter != nil && limiter.limit == limit {
		return limiter
	}
	limiter = &upstreamConcurrencyLimiter{
		key:   key,
		limit: limit,
		sem:   make(chan struct{}, limit),
	}
	m.upstreamLimiters[key] = limiter
	return limiter
}

// acquireUpstreamConcurrency blocks until a permit is available for the provider or the queue times out.
// It returns a nil permit (and nil error) when no limit is configured for the provider.
func (m *Manager) acquireUpstreamConcurrency(ctx context.Context, provider string, auth *Auth, req cliproxyexecutor.Request) (*upstreamConcurrencyPermit, error) {
	cfg := m.upstreamConcurrencyConfig()
	limit := cfg.LimitForProvider(provider)
	if limit <= 0 {
		return nil, nil
	}
	limiter := m.upstreamLimiterFor(provider, limit)
	if limiter == nil {
		return nil, nil
	}

	timeout := cfg.QueueTimeout()
	waitCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	start := time.Now()
	select {
	case limiter.sem <- struct{}{}:
		wait := time.Since(start)
		entry := log.WithFields(log.Fields{
			"provider": strings.ToLower(strings.TrimSpace(provider)),
			"auth_id":  authIDForConcurrencyLog(auth),
			"model":    strings.TrimSpace(req.Model),
			"limit":    limit,
			"wait_ms":  wait.Milliseconds(),
		})
		if wait > 0 {
			entry.Debug("upstream concurrency permit acquired")
		}
		return &upstreamConcurrencyPermit{limiter: limiter}, nil
	case <-waitCtx.Done():
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		log.WithFields(log.Fields{
			"provider": strings.ToLower(strings.TrimSpace(provider)),
			"auth_id":  authIDForConcurrencyLog(auth),
			"model":    strings.TrimSpace(req.Model),
			"limit":    limit,
			"wait_ms":  time.Since(start).Milliseconds(),
		}).Warn("upstream concurrency queue timeout")
		return nil, &Error{
			Code:       "upstream_concurrency_queue_timeout",
			Message:    "upstream concurrency queue timeout",
			Retryable:  true,
			HTTPStatus: http.StatusServiceUnavailable,
		}
	}
}

// authIDForConcurrencyLog extracts a stable identifier for concurrency log fields.
func authIDForConcurrencyLog(auth *Auth) string {
	if auth == nil {
		return ""
	}
	return strings.TrimSpace(auth.ID)
}

func (m *Manager) executeWithUpstreamConcurrency(ctx context.Context, provider string, auth *Auth, req cliproxyexecutor.Request, execute func() (cliproxyexecutor.Response, error)) (cliproxyexecutor.Response, error) {
	permit, errPermit := m.acquireUpstreamConcurrency(ctx, provider, auth, req)
	if errPermit != nil {
		return cliproxyexecutor.Response{}, errPermit
	}
	if permit != nil {
		defer permit.Release()
	}
	return execute()
}

// streamResultWithPermitRelease wraps a stream result so the concurrency permit is released
// once the underlying chunk channel is fully drained or the context is cancelled.
func streamResultWithPermitRelease(ctx context.Context, result *cliproxyexecutor.StreamResult, permit *upstreamConcurrencyPermit) *cliproxyexecutor.StreamResult {
	if result == nil || permit == nil {
		return result
	}
	out := make(chan cliproxyexecutor.StreamChunk)
	go func() {
		defer close(out)
		defer permit.Release()
		for chunk := range result.Chunks {
			if ctx == nil {
				out <- chunk
				continue
			}
			select {
			case <-ctx.Done():
				discardStreamChunks(result.Chunks)
				return
			case out <- chunk:
			}
		}
	}()
	return &cliproxyexecutor.StreamResult{Headers: result.Headers, Chunks: out}
}
