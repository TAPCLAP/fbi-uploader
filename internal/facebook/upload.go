package facebook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type UploadParams struct {
	AppID           string
	GraphAPIVersion string
	UserAccessToken string
	ZipPath         string
	Comment         string
}

type UploadResult struct {
	SessionID        string
	BundleInstanceID int64
	RawResponse      []byte
}

type createUploadResponse struct {
	ID string `json:"id"`
}

type ruploadResponse struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	BundleInstanceID int64 `json:"bundle_instance_id"`
	PlayableLink     string `json:"playable_link"`
}

// UploadBundle creates a session and uploads the zip via rupload.
func (c *Client) UploadBundle(ctx context.Context, p UploadParams) (UploadResult, error) {
	info, err := os.Stat(p.ZipPath)
	if err != nil {
		return UploadResult{}, fmt.Errorf("stat zip: %w", err)
	}
	if info.IsDir() {
		return UploadResult{}, fmt.Errorf("zip path is a directory: %s", p.ZipPath)
	}

	fileName := filepath.Base(p.ZipPath)
	fileLength := info.Size()

	sessionID, err := c.createUploadSession(ctx, p, fileName, fileLength)
	if err != nil {
		return UploadResult{}, err
	}

	bundleID, raw, err := c.ruploadBundle(ctx, p, sessionID, fileName, fileLength)
	if err != nil {
		return UploadResult{}, opError(opUploadBundle, err)
	}

	return UploadResult{
		SessionID:        sessionID,
		BundleInstanceID: bundleID,
		RawResponse:      raw,
	}, nil
}

func (c *Client) createUploadSession(ctx context.Context, p UploadParams, fileName string, fileLength int64) (string, error) {
	base := fmt.Sprintf("https://graph.facebook.com/%s/%s/uploads", p.GraphAPIVersion, p.AppID)
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("file_name", fileName)
	q.Set("file_length", fmt.Sprintf("%d", fileLength))
	q.Set("file_type", "application/zip")
	q.Set("access_token", p.UserAccessToken)
	u.RawQuery = q.Encode()

	_, body, err := c.doWithRetry(ctx, opCreateUploadSession, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	})
	if err != nil {
		return "", err
	}

	var cur createUploadResponse
	if err := json.Unmarshal(body, &cur); err != nil {
		return "", fmt.Errorf("parse upload session response: %w", err)
	}
	return ExtractSessionID(cur.ID)
}

func (c *Client) ruploadBundle(ctx context.Context, p UploadParams, sessionID, fileName string, fileLength int64) (int64, []byte, error) {
	uploadURL := fmt.Sprintf("https://rupload.facebook.com/gg_graph_api/upload:%s", sessionID)

	f, err := os.Open(p.ZipPath)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, f)
	if err != nil {
		return 0, nil, err
	}

	lengthStr := fmt.Sprintf("%d", fileLength)
	req.Header.Set("Authorization", "OAuth "+p.UserAccessToken)
	req.Header.Set("Offset", "0")
	req.Header.Set("X-Entity-Length", lengthStr)
	req.Header.Set("Content-Length", lengthStr)
	req.Header.Set("type", "BUNDLE")
	req.Header.Set("name", fileName)
	if p.Comment != "" {
		req.Header.Set("comment", p.Comment)
	}
	req.ContentLength = fileLength

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	body, err := readBody(resp)
	if err != nil {
		return 0, nil, err
	}
	if err := checkRuploadResponse(resp, body); err != nil {
		return 0, body, err
	}

	bundleID, err := parseBundleInstanceID(body)
	if err != nil {
		return 0, body, err
	}
	return bundleID, body, nil
}

func checkRuploadResponse(resp *http.Response, body []byte) error {
	if err := checkResponse(resp, body); err != nil {
		return err
	}

	var payload struct {
		DebugInfo struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"debug_info"`
	}
	if json.Unmarshal(body, &payload) == nil && payload.DebugInfo.Type != "" {
		return fmt.Errorf("rupload error: %s: %s", payload.DebugInfo.Type, payload.DebugInfo.Message)
	}
	return nil
}

// UploadBundleWithRetry retries the full upload flow (new session each attempt).
// rupload cannot be retried on the same session after partial data was sent.
func (c *Client) UploadBundleWithRetry(ctx context.Context, p UploadParams) (UploadResult, error) {
	cfg := c.Retry.withDefaults()

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		result, err := c.UploadBundle(ctx, p)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !isRetryable(err) || attempt == cfg.MaxAttempts {
			break
		}
		c.logUploadRetry(operationFromError(lastErr), attempt, cfg.MaxAttempts, delay, lastErr)
		if err := sleepWithContext(ctx, delay); err != nil {
			return UploadResult{}, err
		}
		delay *= 2
	}

	return UploadResult{}, lastErr
}

func parseBundleInstanceID(body []byte) (int64, error) {
	var rr ruploadResponse
	if err := json.Unmarshal(body, &rr); err != nil {
		return 0, fmt.Errorf("parse rupload response: %w", err)
	}
	if rr.BundleInstanceID != 0 {
		return rr.BundleInstanceID, nil
	}
	return 0, fmt.Errorf("bundle instance id not found in response: %s", truncate(string(body), 256))
}
