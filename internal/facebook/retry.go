package facebook

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

const (
	defaultRetryAttempts     = 10
	defaultRetryInitialDelay   = time.Second
	opCreateUploadSession      = "create upload session"
	opUploadBundle             = "upload bundle"
	opPushToProduction         = "push to production"
	opFetchAppAccessToken      = "fetch app access token"
)

type operationError struct {
	operation string
	err       error
}

func (e *operationError) Error() string {
	return sanitizeSecrets(e.operation + ": " + e.err.Error())
}

func (e *operationError) Unwrap() error {
	return e.err
}

func opError(operation string, err error) error {
	if err == nil {
		return nil
	}
	var existing *operationError
	if errors.As(err, &existing) {
		return err
	}
	return &operationError{operation: operation, err: err}
}

func operationFromError(err error) string {
	var opErr *operationError
	if errors.As(err, &opErr) {
		return opErr.operation
	}
	return opUploadBundle
}

// RetryConfig controls HTTP request retries on transient failures.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
}

func (cfg RetryConfig) withDefaults() RetryConfig {
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = defaultRetryAttempts
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = defaultRetryInitialDelay
	}
	return cfg
}

type requestFactory func() (*http.Request, error)

func (c *Client) doWithRetry(ctx context.Context, operation string, newReq requestFactory) (*http.Response, []byte, error) {
	cfg := c.Retry.withDefaults()

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		req, err := newReq()
		if err != nil {
			return nil, nil, err
		}

		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			if !isRetryable(err) || attempt == cfg.MaxAttempts {
				return nil, nil, opError(operation, err)
			}
			c.logRequestRetry(operation, attempt, cfg.MaxAttempts, delay, err)
			if err := sleepWithContext(ctx, delay); err != nil {
				return nil, nil, err
			}
			delay *= 2
			continue
		}

		body, err := readBody(resp)
		if err != nil {
			lastErr = err
			if !isRetryable(err) || attempt == cfg.MaxAttempts {
				return nil, nil, opError(operation, err)
			}
			c.logRequestRetry(operation, attempt, cfg.MaxAttempts, delay, err)
			if err := sleepWithContext(ctx, delay); err != nil {
				return nil, nil, err
			}
			delay *= 2
			continue
		}

		if err := checkResponse(resp, body); err != nil {
			lastErr = err
			if !isRetryable(err) || attempt == cfg.MaxAttempts {
				return resp, body, opError(operation, err)
			}
			c.logRequestRetry(operation, attempt, cfg.MaxAttempts, delay, err)
			if err := sleepWithContext(ctx, delay); err != nil {
				return nil, nil, err
			}
			delay *= 2
			continue
		}

		return resp, body, nil
	}

	return nil, nil, opError(operation, lastErr)
}

func (c *Client) logRequestRetry(operation string, attempt, maxAttempts int, delay time.Duration, err error) {
	if c.Logger == nil {
		return
	}
	c.Logger.Warn("request failed, retrying",
		slog.String("operation", operation),
		slog.Int("attempt", attempt),
		slog.Int("max_attempts", maxAttempts),
		slog.Duration("retry_after", delay),
		slog.String("error", safeErrorString(err)),
	)
}

func (c *Client) logUploadRetry(operation string, attempt, maxAttempts int, delay time.Duration, err error) {
	if c.Logger == nil {
		return
	}
	c.Logger.Warn("upload failed, retrying",
		slog.String("operation", operation),
		slog.Int("upload_attempt", attempt),
		slog.Int("max_upload_attempts", maxAttempts),
		slog.Duration("retry_after", delay),
		slog.String("error", safeErrorString(err)),
	)
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= 500
	}
	return true
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
