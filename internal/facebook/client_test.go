package facebook

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClient_appliesTimeouts(t *testing.T) {
	t.Parallel()

	client := NewClient(RetryConfig{}, TimeoutConfig{
		ConnectTimeout:  5 * time.Second,
		ResponseTimeout: 12 * time.Second,
		RequestTimeout:  20 * time.Second,
	}, nil)

	if client.HTTP.Timeout != 20*time.Second {
		t.Fatalf("RequestTimeout = %v, want 20s", client.HTTP.Timeout)
	}
	tr, ok := client.HTTP.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if tr.ResponseHeaderTimeout != 12*time.Second {
		t.Fatalf("ResponseTimeout = %v, want 12s", tr.ResponseHeaderTimeout)
	}
	if client.Timeouts.ConnectTimeout != 5*time.Second {
		t.Fatalf("ConnectTimeout = %v, want 5s", client.Timeouts.ConnectTimeout)
	}
}

func TestTimeoutConfig_withDefaults(t *testing.T) {
	t.Parallel()

	cfg := TimeoutConfig{}.withDefaults()
	if cfg.ConnectTimeout != defaultConnectTimeout {
		t.Fatalf("ConnectTimeout = %v, want %v", cfg.ConnectTimeout, defaultConnectTimeout)
	}
	if cfg.ResponseTimeout != defaultResponseTimeout {
		t.Fatalf("ResponseTimeout = %v, want %v", cfg.ResponseTimeout, defaultResponseTimeout)
	}
	if cfg.RequestTimeout != defaultRequestTimeout {
		t.Fatalf("RequestTimeout = %v, want %v", cfg.RequestTimeout, defaultRequestTimeout)
	}
}

func TestExtractSessionID(t *testing.T) {
	t.Parallel()

	id, err := ExtractSessionID("upload:abc123")
	if err != nil {
		t.Fatal(err)
	}
	if id != "abc123" {
		t.Fatalf("got %q", id)
	}

	_, err = ExtractSessionID("bad")
	if err == nil {
		t.Fatal("expected error for bad format")
	}

	_, err = ExtractSessionID("upload:")
	if err == nil {
		t.Fatal("expected error for empty session id")
	}
}
