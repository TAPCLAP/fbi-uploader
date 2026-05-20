package facebook

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	connectTimeout = 30 * time.Second
	requestTimeout = 600 * time.Second
)

type Client struct {
	HTTP   *http.Client
	Retry  RetryConfig
	Logger *slog.Logger
}

func NewClient(retry RetryConfig, logger *slog.Logger) *Client {
	return &Client{
		HTTP: &http.Client{
			Timeout: requestTimeout,
			Transport: &http.Transport{
				ResponseHeaderTimeout: connectTimeout,
			},
		},
		Retry:  retry.withDefaults(),
		Logger: logger,
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
