package auth

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	internalconfig "github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
)

type concurrencyLimitExecutor struct {
	id    string
	delay time.Duration

	mu        sync.Mutex
	active    int
	maxActive int
	calls     int
}

func (e *concurrencyLimitExecutor) Identifier() string { return e.id }

func (e *concurrencyLimitExecutor) Execute(ctx context.Context, _ *Auth, _ cliproxyexecutor.Request, _ cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	e.begin()
	defer e.end()
	select {
	case <-ctx.Done():
		return cliproxyexecutor.Response{}, ctx.Err()
	case <-time.After(e.delay):
		return cliproxyexecutor.Response{Payload: []byte("ok")}, nil
	}
}

func (e *concurrencyLimitExecutor) ExecuteStream(context.Context, *Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (*cliproxyexecutor.StreamResult, error) {
	return nil, &Error{HTTPStatus: http.StatusNotImplemented, Message: "ExecuteStream not implemented"}
}

func (e *concurrencyLimitExecutor) Refresh(_ context.Context, auth *Auth) (*Auth, error) {
	return auth, nil
}

func (e *concurrencyLimitExecutor) CountTokens(ctx context.Context, _ *Auth, _ cliproxyexecutor.Request, _ cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	e.begin()
	defer e.end()
	select {
	case <-ctx.Done():
		return cliproxyexecutor.Response{}, ctx.Err()
	case <-time.After(e.delay):
		return cliproxyexecutor.Response{Payload: []byte("tokens")}, nil
	}
}

func (e *concurrencyLimitExecutor) HttpRequest(context.Context, *Auth, *http.Request) (*http.Response, error) {
	return nil, &Error{HTTPStatus: http.StatusNotImplemented, Message: "HttpRequest not implemented"}
}

func (e *concurrencyLimitExecutor) begin() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	e.active++
	if e.active > e.maxActive {
		e.maxActive = e.active
	}
}

func (e *concurrencyLimitExecutor) end() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.active--
}

func (e *concurrencyLimitExecutor) MaxActive() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.maxActive
}

func (e *concurrencyLimitExecutor) Calls() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

func newConcurrencyLimitManager(t *testing.T, limit int) (*Manager, *concurrencyLimitExecutor, string) {
	t.Helper()

	const provider = "codex"
	model := "gpt-5.3-codex-spark-" + t.Name()
	auth := &Auth{ID: "auth-" + t.Name(), Provider: provider}
	executor := &concurrencyLimitExecutor{id: provider, delay: 20 * time.Millisecond}

	m := NewManager(nil, nil, nil)
	m.SetConfig(&internalconfig.Config{
		UpstreamConcurrency: internalconfig.UpstreamConcurrencyConfig{
			Providers:           map[string]int{provider: limit},
			QueueTimeoutSeconds: 1,
		},
	})
	m.RegisterExecutor(executor)

	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, provider, []*registry.ModelInfo{{ID: model}})
	t.Cleanup(func() { reg.UnregisterClient(auth.ID) })

	if _, errRegister := m.Register(context.Background(), auth); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}
	return m, executor, model
}

func TestManagerUpstreamConcurrencyProviderLimit(t *testing.T) {
	m, executor, model := newConcurrencyLimitManager(t, 2)

	const requests = 8
	var wg sync.WaitGroup
	errCh := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errExecute := m.Execute(context.Background(), []string{"codex"}, cliproxyexecutor.Request{Model: model}, cliproxyexecutor.Options{})
			errCh <- errExecute
		}()
	}
	wg.Wait()
	close(errCh)

	for errExecute := range errCh {
		if errExecute != nil {
			t.Fatalf("execute returned error: %v", errExecute)
		}
	}
	if maxActive := executor.MaxActive(); maxActive > 2 {
		t.Fatalf("max active executions = %d, want <= 2", maxActive)
	}
	if calls := executor.Calls(); calls != requests {
		t.Fatalf("calls = %d, want %d", calls, requests)
	}
}

func TestManagerUpstreamConcurrencyDisabledByDefault(t *testing.T) {
	m, executor, model := newConcurrencyLimitManager(t, 0)

	const requests = 6
	var wg sync.WaitGroup
	errCh := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errExecute := m.Execute(context.Background(), []string{"codex"}, cliproxyexecutor.Request{Model: model}, cliproxyexecutor.Options{})
			errCh <- errExecute
		}()
	}
	wg.Wait()
	close(errCh)

	for errExecute := range errCh {
		if errExecute != nil {
			t.Fatalf("execute returned error: %v", errExecute)
		}
	}
	if maxActive := executor.MaxActive(); maxActive <= 2 {
		t.Fatalf("max active executions = %d, want concurrent execution when disabled", maxActive)
	}
}

func TestManagerUpstreamConcurrencyWaitRespectsContextCancellation(t *testing.T) {
	m, executor, model := newConcurrencyLimitManager(t, 1)
	executor.delay = 200 * time.Millisecond

	firstDone := make(chan error, 1)
	go func() {
		_, errExecute := m.Execute(context.Background(), []string{"codex"}, cliproxyexecutor.Request{Model: model}, cliproxyexecutor.Options{})
		firstDone <- errExecute
	}()

	for deadline := time.Now().Add(time.Second); executor.Calls() == 0 && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	if calls := executor.Calls(); calls != 1 {
		t.Fatalf("first request did not acquire permit, calls=%d", calls)
	}

	ctx, cancel := context.WithCancel(context.Background())
	secondDone := make(chan error, 1)
	go func() {
		_, errExecute := m.Execute(ctx, []string{"codex"}, cliproxyexecutor.Request{Model: model}, cliproxyexecutor.Options{})
		secondDone <- errExecute
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case errExecute := <-secondDone:
		if !errors.Is(errExecute, context.Canceled) {
			t.Fatalf("second error = %v, want context.Canceled", errExecute)
		}
	case <-time.After(time.Second):
		t.Fatal("second request did not exit after context cancellation")
	}

	if errFirst := <-firstDone; errFirst != nil {
		t.Fatalf("first request returned error: %v", errFirst)
	}
	if calls := executor.Calls(); calls != 1 {
		t.Fatalf("executor calls = %d, want 1 after canceled wait", calls)
	}
}

type streamConcurrencyLimitExecutor struct {
	id string

	mu    sync.Mutex
	calls int

	started chan struct{}
	release chan struct{}
}

func (e *streamConcurrencyLimitExecutor) Identifier() string { return e.id }

func (e *streamConcurrencyLimitExecutor) Execute(context.Context, *Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, &Error{HTTPStatus: http.StatusNotImplemented, Message: "Execute not implemented"}
}

func (e *streamConcurrencyLimitExecutor) ExecuteStream(context.Context, *Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (*cliproxyexecutor.StreamResult, error) {
	e.mu.Lock()
	e.calls++
	e.mu.Unlock()
	e.started <- struct{}{}

	ch := make(chan cliproxyexecutor.StreamChunk)
	go func() {
		defer close(ch)
		ch <- cliproxyexecutor.StreamChunk{Payload: []byte("chunk")}
		<-e.release
	}()
	return &cliproxyexecutor.StreamResult{Headers: http.Header{}, Chunks: ch}, nil
}

func (e *streamConcurrencyLimitExecutor) Refresh(_ context.Context, auth *Auth) (*Auth, error) {
	return auth, nil
}

func (e *streamConcurrencyLimitExecutor) CountTokens(context.Context, *Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, &Error{HTTPStatus: http.StatusNotImplemented, Message: "CountTokens not implemented"}
}

func (e *streamConcurrencyLimitExecutor) HttpRequest(context.Context, *Auth, *http.Request) (*http.Response, error) {
	return nil, &Error{HTTPStatus: http.StatusNotImplemented, Message: "HttpRequest not implemented"}
}

func TestManagerUpstreamConcurrencyStreamPermitHeldUntilStreamCloses(t *testing.T) {
	const provider = "codex"
	model := "gpt-5.3-codex-spark-stream-" + t.Name()
	auth := &Auth{ID: "stream-auth-" + t.Name(), Provider: provider}
	executor := &streamConcurrencyLimitExecutor{
		id:      provider,
		started: make(chan struct{}, 2),
		release: make(chan struct{}),
	}

	m := NewManager(nil, nil, nil)
	m.SetConfig(&internalconfig.Config{
		UpstreamConcurrency: internalconfig.UpstreamConcurrencyConfig{
			Providers:           map[string]int{provider: 1},
			QueueTimeoutSeconds: 1,
		},
	})
	m.RegisterExecutor(executor)

	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, provider, []*registry.ModelInfo{{ID: model}})
	t.Cleanup(func() { reg.UnregisterClient(auth.ID) })
	if _, errRegister := m.Register(context.Background(), auth); errRegister != nil {
		t.Fatalf("register auth: %v", errRegister)
	}

	first, errFirst := m.ExecuteStream(context.Background(), []string{provider}, cliproxyexecutor.Request{Model: model}, cliproxyexecutor.Options{})
	if errFirst != nil {
		t.Fatalf("first stream error: %v", errFirst)
	}
	<-executor.started

	secondStarted := make(chan error, 1)
	go func() {
		second, errSecond := m.ExecuteStream(context.Background(), []string{provider}, cliproxyexecutor.Request{Model: model}, cliproxyexecutor.Options{})
		if errSecond != nil {
			secondStarted <- errSecond
			return
		}
		for range second.Chunks {
		}
		secondStarted <- nil
	}()

	select {
	case <-executor.started:
		t.Fatal("second stream started before first stream permit was released")
	case <-time.After(30 * time.Millisecond):
	}

	close(executor.release)
	for range first.Chunks {
	}

	select {
	case <-executor.started:
	case <-time.After(time.Second):
		t.Fatal("second stream did not start after first stream closed")
	}
	select {
	case errSecond := <-secondStarted:
		if errSecond != nil {
			t.Fatalf("second stream error: %v", errSecond)
		}
	case <-time.After(time.Second):
		t.Fatal("second stream did not finish")
	}
}
