package facebook

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoWithRetry_retriesNetworkError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("server does not support hijack")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatal(err)
			}
			conn.Close()
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"token"}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(RetryConfig{
		MaxAttempts:  5,
		InitialDelay: time.Millisecond,
	}, nil)
	client.HTTP = server.Client()

	_, body, err := client.doWithRetry(context.Background(), "test request", func() (*http.Request, error) {
		return http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
	if len(body) == 0 {
		t.Fatal("expected response body")
	}
}

func TestDoWithRetry_doesNotRetryClientError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad request","code":100}}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(RetryConfig{
		MaxAttempts:  5,
		InitialDelay: time.Millisecond,
	}, nil)
	client.HTTP = server.Client()

	_, _, err := client.doWithRetry(context.Background(), "test request", func() (*http.Request, error) {
		return http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestDoWithRetry_retriesServerError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"message":"temporary","code":1}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(RetryConfig{
		MaxAttempts:  5,
		InitialDelay: time.Millisecond,
	}, nil)
	client.HTTP = server.Client()

	_, _, err := client.doWithRetry(context.Background(), "test request", func() (*http.Request, error) {
		return http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
}

func TestDoWithRetry_exponentialBackoff(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"temporary","code":1}}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 50 * time.Millisecond,
	}, nil)
	client.HTTP = server.Client()

	start := time.Now()
	_, _, err := client.doWithRetry(context.Background(), "test request", func() (*http.Request, error) {
		return http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
	if elapsed < 150*time.Millisecond {
		t.Fatalf("elapsed = %v, expected at least 150ms backoff (50ms + 100ms)", elapsed)
	}
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()

	if isRetryable(nil) {
		t.Fatal("nil error should not be retryable")
	}
	if isRetryable(context.Canceled) {
		t.Fatal("context.Canceled should not be retryable")
	}
	if isRetryable(context.DeadlineExceeded) {
		t.Fatal("context.DeadlineExceeded should not be retryable")
	}
	if !isRetryable(errors.New("connection reset by peer")) {
		t.Fatal("network error should be retryable")
	}
	if !isRetryable(&APIError{StatusCode: 503}) {
		t.Fatal("5xx API error should be retryable")
	}
	if isRetryable(&APIError{StatusCode: 400}) {
		t.Fatal("4xx API error should not be retryable")
	}
}

func TestOperationFromError(t *testing.T) {
	t.Parallel()

	err := opError(opCreateUploadSession, errors.New("timeout"))
	if operationFromError(err) != opCreateUploadSession {
		t.Fatalf("got %q", operationFromError(err))
	}

	err = opError(opUploadBundle, opError(opUploadBundle, errors.New("rupload failed")))
	if operationFromError(err) != opUploadBundle {
		t.Fatalf("got %q", operationFromError(err))
	}
}

func TestRetryConfig_withDefaults(t *testing.T) {
	t.Parallel()

	cfg := RetryConfig{}.withDefaults()
	if cfg.MaxAttempts != defaultRetryAttempts {
		t.Fatalf("MaxAttempts = %d, want %d", cfg.MaxAttempts, defaultRetryAttempts)
	}
	if cfg.InitialDelay != defaultRetryInitialDelay {
		t.Fatalf("InitialDelay = %v, want %v", cfg.InitialDelay, defaultRetryInitialDelay)
	}
}
