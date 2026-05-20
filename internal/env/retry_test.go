package env

import (
	"testing"
	"time"
)

func TestLoadAPIRetryConfig_defaults(t *testing.T) {
	t.Setenv("FB_API_RETRIES", "")
	t.Setenv("FB_API_RETRY_DELAY_MS", "")

	cfg := LoadAPIRetryConfig()
	if cfg.MaxAttempts != 10 {
		t.Fatalf("MaxAttempts = %d, want 10", cfg.MaxAttempts)
	}
	if cfg.InitialDelay != time.Second {
		t.Fatalf("InitialDelay = %v, want 1s", cfg.InitialDelay)
	}
}

func TestLoadAPIRetryConfig_custom(t *testing.T) {
	t.Setenv("FB_API_RETRIES", "5")
	t.Setenv("FB_API_RETRY_DELAY_MS", "250")

	cfg := LoadAPIRetryConfig()
	if cfg.MaxAttempts != 5 {
		t.Fatalf("MaxAttempts = %d, want 5", cfg.MaxAttempts)
	}
	if cfg.InitialDelay != 250*time.Millisecond {
		t.Fatalf("InitialDelay = %v, want 250ms", cfg.InitialDelay)
	}
}
