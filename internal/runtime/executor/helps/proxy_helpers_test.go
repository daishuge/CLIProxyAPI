package helps

import (
	"context"
	"net/http"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

func TestNewProxyAwareHTTPClientDirectBypassesGlobalProxy(t *testing.T) {
	t.Parallel()

	client := NewProxyAwareHTTPClient(
		context.Background(),
		&config.Config{SDKConfig: sdkconfig.SDKConfig{ProxyURL: "http://global-proxy.example.com:8080"}},
		&cliproxyauth.Auth{ProxyURL: "direct"},
		0,
	)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("expected direct transport to disable proxy function")
	}
}

func TestNewProxyAwareHTTPClientReusesProxyTransport(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{SDKConfig: sdkconfig.SDKConfig{ProxyURL: "http://proxy-reuse.example.com:8080"}}
	clientA := NewProxyAwareHTTPClient(context.Background(), cfg, nil, 0)
	clientB := NewProxyAwareHTTPClient(context.Background(), cfg, nil, 0)

	if clientA.Transport == nil || clientB.Transport == nil {
		t.Fatal("expected proxy transports")
	}
	if clientA.Transport != clientB.Transport {
		t.Fatal("expected same proxy URL to reuse cached transport")
	}
}

func TestNewProxyAwareHTTPClientDoesNotShareDifferentProxyTransports(t *testing.T) {
	t.Parallel()

	clientA := NewProxyAwareHTTPClient(
		context.Background(),
		&config.Config{SDKConfig: sdkconfig.SDKConfig{ProxyURL: "http://proxy-a.example.com:8080"}},
		nil,
		0,
	)
	clientB := NewProxyAwareHTTPClient(
		context.Background(),
		&config.Config{SDKConfig: sdkconfig.SDKConfig{ProxyURL: "http://proxy-b.example.com:8080"}},
		nil,
		0,
	)

	if clientA.Transport == nil || clientB.Transport == nil {
		t.Fatal("expected proxy transports")
	}
	if clientA.Transport == clientB.Transport {
		t.Fatal("expected different proxy URLs to use different transports")
	}
}

func TestNewProxyAwareHTTPClientReusesDirectTransport(t *testing.T) {
	t.Parallel()

	auth := &cliproxyauth.Auth{ProxyURL: "direct"}
	clientA := NewProxyAwareHTTPClient(context.Background(), nil, auth, 0)
	clientB := NewProxyAwareHTTPClient(context.Background(), nil, auth, 0)

	if clientA.Transport == nil || clientB.Transport == nil {
		t.Fatal("expected direct transports")
	}
	if clientA.Transport != clientB.Transport {
		t.Fatal("expected direct proxy mode to reuse cached transport")
	}
	transport, ok := clientA.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", clientA.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("expected direct transport to disable proxy function")
	}
}

type testRoundTripper struct{}

func (testRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, nil
}

func TestNewProxyAwareHTTPClientUsesContextRoundTripperWithoutProxy(t *testing.T) {
	t.Parallel()

	rt := testRoundTripper{}
	ctx := context.WithValue(context.Background(), "cliproxy.roundtripper", rt)
	client := NewProxyAwareHTTPClient(ctx, &config.Config{}, nil, 0)

	if client.Transport != rt {
		t.Fatalf("transport = %T, want context round tripper", client.Transport)
	}
}
