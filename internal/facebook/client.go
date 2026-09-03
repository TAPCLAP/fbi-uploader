package facebook

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	defaultConnectTimeout  = 30 * time.Second
	defaultResponseTimeout = 600 * time.Second
	defaultRequestTimeout  = 600 * time.Second
)

type TimeoutConfig struct {
	ConnectTimeout  time.Duration
	ResponseTimeout time.Duration
	RequestTimeout  time.Duration
}

func (cfg TimeoutConfig) withDefaults() TimeoutConfig {
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = defaultConnectTimeout
	}
	if cfg.ResponseTimeout <= 0 {
		cfg.ResponseTimeout = defaultResponseTimeout
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaultRequestTimeout
	}
	return cfg
}

type Client struct {
	HTTP     *http.Client
	Retry    RetryConfig
	Timeouts TimeoutConfig
	Logger   *slog.Logger
}

func NewClient(retry RetryConfig, timeouts TimeoutConfig, logger *slog.Logger) *Client {
	timeouts = timeouts.withDefaults()
	return &Client{
		HTTP: &http.Client{
			Timeout: timeouts.RequestTimeout,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: timeouts.ConnectTimeout,
				}).DialContext,
				ResponseHeaderTimeout: timeouts.ResponseTimeout,
			},
		},
		Retry:    retry.withDefaults(),
		Timeouts: timeouts,
		Logger:   logger,
	}
}

type APIError struct {
	StatusCode int
	Body       string
	Message    string
	Code       int
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return sanitizeSecrets(fmt.Sprintf("facebook api error (status %d, code %d): %s", e.StatusCode, e.Code, e.Message))
	}
	return sanitizeSecrets(fmt.Sprintf("facebook api error (status %d): %s", e.StatusCode, truncate(e.Body, 512)))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

type graphErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

func checkResponse(resp *http.Response, body []byte) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var errPayload graphErrorResponse
		if json.Unmarshal(body, &errPayload) == nil && errPayload.Error.Message != "" {
			return &APIError{
				StatusCode: resp.StatusCode,
				Body:       string(body),
				Message:    errPayload.Error.Message,
				Code:       errPayload.Error.Code,
			}
		}
		return nil
	}

	var errPayload graphErrorResponse
	msg := ""
	code := 0
	if json.Unmarshal(body, &errPayload) == nil {
		msg = errPayload.Error.Message
		code = errPayload.Error.Code
	}
	return &APIError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Message:    msg,
		Code:       code,
	}
}

func readBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ExtractSessionID parses upload session id from "upload:SESSION_ID".
func ExtractSessionID(id string) (string, error) {
	const prefix = "upload:"
	if !strings.HasPrefix(id, prefix) {
		return "", fmt.Errorf("unexpected upload session id format: %q", id)
	}
	sessionID := strings.TrimPrefix(id, prefix)
	if sessionID == "" {
		return "", fmt.Errorf("empty session id in %q", id)
	}
	return sessionID, nil
}
