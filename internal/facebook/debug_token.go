package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const opInspectUserAccessToken = "inspect user access token"

const graphAPIHost = "https://graph.facebook.com"

// DebugTokenParams configures a Graph API debug_token request.
type DebugTokenParams struct {
	InputToken      string
	AccessToken     string
	GraphAPIVersion string
}

// UserTokenInfo describes expiration of a user access token from debug_token.
type UserTokenInfo struct {
	Valid        bool
	NeverExpires bool
	ExpiresAt    time.Time
	// Incomplete is true when the response has no usable is_valid field
	// (for example {"data":[]}). The token was not proven invalid or expired.
	Incomplete bool
}

// IsExpired reports whether the token is invalid or past its expiration time.
// Incomplete responses are not treated as expired.
func (info UserTokenInfo) IsExpired(now time.Time) bool {
	if info.Incomplete {
		return false
	}
	if !info.Valid {
		return true
	}
	if info.NeverExpires {
		return false
	}
	return !info.ExpiresAt.IsZero() && !info.ExpiresAt.After(now)
}

// FormatRemaining formats a duration as days, hours, and minutes.
func FormatRemaining(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	days := d / (24 * time.Hour)
	d %= 24 * time.Hour
	hours := d / time.Hour
	d %= time.Hour
	minutes := d / time.Minute
	return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
}

type debugTokenResponse struct {
	Data json.RawMessage `json:"data"`
}

type debugTokenData struct {
	IsValid   *bool  `json:"is_valid"`
	ExpiresAt *int64 `json:"expires_at"`
}

// DebugTokenEndpoint is the debug_token URL without query parameters.
func DebugTokenEndpoint(graphAPIVersion string) string {
	if v := strings.TrimSpace(graphAPIVersion); v != "" {
		return graphAPIHost + "/" + v + "/debug_token"
	}
	return graphAPIHost + "/debug_token"
}

// DebugTokenURL builds the Graph API debug_token URL.
// If AccessToken is empty, InputToken is used for both query parameters.
func DebugTokenURL(p DebugTokenParams) (string, error) {
	accessToken := p.AccessToken
	if accessToken == "" {
		accessToken = p.InputToken
	}

	u, err := url.Parse(DebugTokenEndpoint(p.GraphAPIVersion))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("input_token", p.InputToken)
	q.Set("access_token", accessToken)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// FetchDebugToken calls Graph API debug_token and returns the raw response body.
// The body is returned even when the request fails with an API error, so callers can inspect it.
func (c *Client) FetchDebugToken(ctx context.Context, p DebugTokenParams) ([]byte, error) {
	rawURL, err := DebugTokenURL(p)
	if err != nil {
		return nil, err
	}

	_, body, err := c.doWithRetry(ctx, opInspectUserAccessToken, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	})
	return body, err
}

// InspectUserAccessToken calls Graph API debug_token for the given user token.
func (c *Client) InspectUserAccessToken(ctx context.Context, userToken string) (UserTokenInfo, error) {
	body, err := c.FetchDebugToken(ctx, DebugTokenParams{InputToken: userToken})
	if err != nil {
		return UserTokenInfo{}, err
	}

	return ParseDebugTokenResponse(body)
}

// ParseDebugTokenResponse parses debug_token JSON (for tests).
// Missing or unexpected data (empty array, null, object without is_valid)
// returns Incomplete info and no error.
func ParseDebugTokenResponse(body []byte) (UserTokenInfo, error) {
	var resp debugTokenResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return UserTokenInfo{}, fmt.Errorf("parse debug_token response: %w", err)
	}

	data, ok := parseDebugTokenData(resp.Data)
	if !ok || data.IsValid == nil {
		return UserTokenInfo{Incomplete: true}, nil
	}

	info := UserTokenInfo{Valid: *data.IsValid}
	if data.ExpiresAt == nil || *data.ExpiresAt == 0 {
		info.NeverExpires = true
		return info, nil
	}

	info.ExpiresAt = time.Unix(*data.ExpiresAt, 0).UTC()
	return info, nil
}

func parseDebugTokenData(raw json.RawMessage) (debugTokenData, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return debugTokenData{}, false
	}
	var data debugTokenData
	if err := json.Unmarshal(raw, &data); err != nil {
		return debugTokenData{}, false
	}
	return data, true
}

// ParseDebugTokenResponseReader is like ParseDebugTokenResponse but reads from r.
func ParseDebugTokenResponseReader(r io.Reader) (UserTokenInfo, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return UserTokenInfo{}, err
	}
	return ParseDebugTokenResponse(body)
}
