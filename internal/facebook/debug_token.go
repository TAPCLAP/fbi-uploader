package facebook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const opInspectUserAccessToken = "inspect user access token"

// UserTokenInfo describes expiration of a user access token from debug_token.
type UserTokenInfo struct {
	Valid        bool
	NeverExpires bool
	ExpiresAt    time.Time
}

// IsExpired reports whether the token is invalid or past its expiration time.
func (info UserTokenInfo) IsExpired(now time.Time) bool {
	if !info.Valid {
		return true
	}
	if info.NeverExpires {
		return false
	}
	return !info.ExpiresAt.IsZero() && !info.ExpiresAt.After(now)
}

type debugTokenResponse struct {
	Data struct {
		IsValid   bool  `json:"is_valid"`
		ExpiresAt int64 `json:"expires_at"`
	} `json:"data"`
}

// InspectUserAccessToken calls Graph API debug_token for the given user token.
func (c *Client) InspectUserAccessToken(ctx context.Context, userToken string) (UserTokenInfo, error) {
	u, err := url.Parse("https://graph.facebook.com/debug_token")
	if err != nil {
		return UserTokenInfo{}, err
	}
	q := u.Query()
	q.Set("input_token", userToken)
	q.Set("access_token", userToken)
	u.RawQuery = q.Encode()

	_, body, err := c.doWithRetry(ctx, opInspectUserAccessToken, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	})
	if err != nil {
		return UserTokenInfo{}, err
	}

	return ParseDebugTokenResponse(body)
}

// ParseDebugTokenResponse parses debug_token JSON (for tests).
func ParseDebugTokenResponse(body []byte) (UserTokenInfo, error) {
	var resp debugTokenResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return UserTokenInfo{}, fmt.Errorf("parse debug_token response: %w", err)
	}

	info := UserTokenInfo{Valid: resp.Data.IsValid}
	if resp.Data.ExpiresAt == 0 {
		info.NeverExpires = true
		return info, nil
	}

	info.ExpiresAt = time.Unix(resp.Data.ExpiresAt, 0).UTC()
	return info, nil
}

// ParseDebugTokenResponseReader is like ParseDebugTokenResponse but reads from r.
func ParseDebugTokenResponseReader(r io.Reader) (UserTokenInfo, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return UserTokenInfo{}, err
	}
	return ParseDebugTokenResponse(body)
}
