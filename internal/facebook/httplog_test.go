package facebook

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func debugLogBuffer(t *testing.T) (*Client, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewClient(RetryConfig{MaxAttempts: 1, InitialDelay: time.Millisecond}, TimeoutConfig{}, logger)
	return client, &buf
}

func TestLogHTTPRequest_redactsTokens(t *testing.T) {
	t.Parallel()

	client, buf := debugLogBuffer(t)
	req, err := http.NewRequest(http.MethodPost, "https://graph.facebook.com/v26.0/1/uploads?access_token=EAAsecret123&file_name=x.zip", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "OAuth EAAsecret123")
	client.logHTTPRequest(opCreateUploadSession, req)

	out := buf.String()
	if strings.Contains(out, "EAAsecret123") {
		t.Fatalf("token leaked: %s", out)
	}
	if !strings.Contains(out, "access_token=[REDACTED]") {
		t.Fatalf("expected redacted access_token in %s", out)
	}
	if !strings.Contains(out, "OAuth [REDACTED]") {
		t.Fatalf("expected redacted Authorization in %s", out)
	}
	if !strings.Contains(out, "msg=\"http request\"") {
		t.Fatalf("expected http request log in %s", out)
	}
}

func TestLogHTTPRequest_logsJSONBodyAndRestoresIt(t *testing.T) {
	t.Parallel()

	client, buf := debugLogBuffer(t)
	payload := `{"bundle_instance_id":"123"}`
	req, err := http.NewRequest(http.MethodPost, "https://api.facebook.com/instant-games/assets/1/push-to-production", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "OAuth 123|secret")
	req.ContentLength = int64(len(payload))
	client.logHTTPRequest(opPushToProduction, req)

	out := buf.String()
	if strings.Contains(out, "123|secret") {
		t.Fatalf("token leaked: %s", out)
	}
	if !strings.Contains(out, "bundle_instance_id") {
		t.Fatalf("expected JSON body in %s", out)
	}

	got, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != payload {
		t.Fatalf("body consumed: got %q", got)
	}
}

func TestLogHTTPRequest_skipsBinaryBody(t *testing.T) {
	t.Parallel()

	client, buf := debugLogBuffer(t)
	payload := bytes.Repeat([]byte("ZIPDATA"), 32)
	req, err := http.NewRequest(http.MethodPost, "https://rupload.facebook.com/fb_game_bundle/upload:abc", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "OAuth EAAsecret123")
	req.ContentLength = int64(len(payload))
	client.logHTTPRequest(opUploadBundle, req)

	out := buf.String()
	if strings.Contains(out, "ZIPDATA") {
		t.Fatalf("binary body logged: %s", out)
	}
	if strings.Contains(out, "EAAsecret123") {
		t.Fatalf("token leaked: %s", out)
	}
	if !strings.Contains(out, "content_length=") {
		t.Fatalf("expected content_length in %s", out)
	}
}

func TestLogHTTPRequest_noopAtInfoLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	client := &Client{Logger: logger}
	req, err := http.NewRequest(http.MethodGet, "https://graph.facebook.com/debug_token?access_token=EAAsecret", nil)
	if err != nil {
		t.Fatal(err)
	}
	client.logHTTPRequest(opInspectUserAccessToken, req)
	if buf.Len() != 0 {
		t.Fatalf("unexpected log: %s", buf.String())
	}
}

func TestDoWithRetry_debugLogsRequestAndResponse(t *testing.T) {
	t.Parallel()

	client, buf := debugLogBuffer(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"upload:abc123"}`))
	}))
	t.Cleanup(server.Close)
	client.HTTP = server.Client()

	_, body, err := client.doWithRetry(context.Background(), opCreateUploadSession, func() (*http.Request, error) {
		return http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL+"/uploads?access_token=EAAsecret123", nil)
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte(`"id":"upload:abc123"`)) {
		t.Fatalf("body = %s", body)
	}

	out := buf.String()
	if strings.Contains(out, "EAAsecret123") {
		t.Fatalf("token leaked: %s", out)
	}
	if !strings.Contains(out, "msg=\"http request\"") {
		t.Fatalf("missing request log: %s", out)
	}
	if !strings.Contains(out, "msg=\"http response\"") {
		t.Fatalf("missing response log: %s", out)
	}
	if !strings.Contains(out, "upload:abc123") {
		t.Fatalf("expected response body in %s", out)
	}
	if !strings.Contains(out, "access_token=[REDACTED]") {
		t.Fatalf("expected redacted token in %s", out)
	}
}
