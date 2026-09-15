package facebook

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
)

const maxDebugBody = 16 * 1024

func (c *Client) debugLogEnabled() bool {
	return c != nil && c.Logger != nil && c.Logger.Enabled(context.Background(), slog.LevelDebug)
}

func (c *Client) logHTTPRequest(operation string, req *http.Request) {
	if !c.debugLogEnabled() || req == nil {
		return
	}

	attrs := []any{
		slog.String("operation", operation),
		slog.String("method", req.Method),
		slog.String("url", sanitizeSecrets(req.URL.String())),
	}
	if h := formatHeaderLog(req.Header); h != "" {
		attrs = append(attrs, slog.String("headers", h))
	}
	if preview, ok := peekRequestBody(req); ok {
		attrs = append(attrs, slog.String("body", sanitizeSecrets(preview)))
	} else if req.ContentLength > 0 {
		attrs = append(attrs, slog.Int64("content_length", req.ContentLength))
	}
	c.Logger.Debug("http request", attrs...)
}

func (c *Client) logHTTPResponse(operation string, resp *http.Response, body []byte) {
	if !c.debugLogEnabled() || resp == nil {
		return
	}

	attrs := []any{
		slog.String("operation", operation),
		slog.Int("status", resp.StatusCode),
	}
	if h := formatHeaderLog(resp.Header); h != "" {
		attrs = append(attrs, slog.String("headers", h))
	}
	if len(body) > 0 {
		attrs = append(attrs, slog.String("body", sanitizeSecrets(truncate(string(body), maxDebugBody))))
	}
	c.Logger.Debug("http response", attrs...)
}

func (c *Client) logHTTPTransportError(operation string, req *http.Request, err error) {
	if !c.debugLogEnabled() {
		return
	}
	attrs := []any{
		slog.String("operation", operation),
		slog.String("error", safeErrorString(err)),
	}
	if req != nil {
		attrs = append(attrs,
			slog.String("method", req.Method),
			slog.String("url", sanitizeSecrets(req.URL.String())),
		)
	}
	c.Logger.Debug("http request failed", attrs...)
}

func formatHeaderLog(h http.Header) string {
	if len(h) == 0 {
		return ""
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(sanitizeSecrets(strings.Join(h.Values(k), ", ")))
	}
	return b.String()
}

func peekRequestBody(req *http.Request) (string, bool) {
	if req.Body == nil || req.Body == http.NoBody {
		return "", false
	}
	if req.ContentLength <= 0 || req.ContentLength > maxDebugBody {
		return "", false
	}
	ct := req.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") && !strings.HasPrefix(ct, "text/") {
		return "", false
	}

	buf, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(buf))
	if err != nil || len(buf) == 0 {
		return "", false
	}
	return string(buf), true
}
