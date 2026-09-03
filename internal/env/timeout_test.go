package env

import (
	"testing"
	"time"
)

func TestLoadAPITimeoutConfig_defaults(t *testing.T) {
	t.Setenv("FB_API_CONNECT_TIMEOUT_MS", "")
	t.Setenv("FB_API_RESPONSE_TIMEOUT_MS", "")
	t.Setenv("FB_API_REQUEST_TIMEOUT_MS", "")

	cfg := LoadAPITimeoutConfig()
	if cfg.ConnectTimeout != 30*time.Second {
		t.Fatalf("ConnectTimeout = %v, want 30s", cfg.ConnectTimeout)
	}
	if cfg.ResponseTimeout != 600*time.Second {
		t.Fatalf("ResponseTimeout = %v, want 600s", cfg.ResponseTimeout)
	}
	if cfg.RequestTimeout != 600*time.Second {
		t.Fatalf("RequestTimeout = %v, want 600s", cfg.RequestTimeout)
	}
}

func TestLoadAPITimeoutConfig_custom(t *testing.T) {
	t.Setenv("FB_API_CONNECT_TIMEOUT_MS", "5000")
	t.Setenv("FB_API_RESPONSE_TIMEOUT_MS", "120000")
	t.Setenv("FB_API_REQUEST_TIMEOUT_MS", "180000")

	cfg := LoadAPITimeoutConfig()
	if cfg.ConnectTimeout != 5*time.Second {
		t.Fatalf("ConnectTimeout = %v, want 5s", cfg.ConnectTimeout)
	}
	if cfg.ResponseTimeout != 120*time.Second {
		t.Fatalf("ResponseTimeout = %v, want 120s", cfg.ResponseTimeout)
	}
	if cfg.RequestTimeout != 180*time.Second {
		t.Fatalf("RequestTimeout = %v, want 180s", cfg.RequestTimeout)
	}
}

func TestLoadAPITimeoutConfig_negativeFallsBackToDefault(t *testing.T) {
	t.Setenv("FB_API_CONNECT_TIMEOUT_MS", "-1")
	t.Setenv("FB_API_RESPONSE_TIMEOUT_MS", "-5")
	t.Setenv("FB_API_REQUEST_TIMEOUT_MS", "-10")

	cfg := LoadAPITimeoutConfig()
	if cfg.ConnectTimeout != 30*time.Second {
		t.Fatalf("ConnectTimeout = %v, want 30s", cfg.ConnectTimeout)
	}
	if cfg.ResponseTimeout != 600*time.Second {
		t.Fatalf("ResponseTimeout = %v, want 600s", cfg.ResponseTimeout)
	}
	if cfg.RequestTimeout != 600*time.Second {
		t.Fatalf("RequestTimeout = %v, want 600s", cfg.RequestTimeout)
	}
}
