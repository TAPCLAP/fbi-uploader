package facebook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// FetchAppAccessToken requests an app access token via client_credentials.
func (c *Client) FetchAppAccessToken(ctx context.Context, appID, appSecret string) (string, error) {
	u, err := url.Parse("https://graph.facebook.com/oauth/access_token")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("client_id", appID)
	q.Set("client_secret", appSecret)
	q.Set("grant_type", "client_credentials")
	u.RawQuery = q.Encode()

	_, body, err := c.doWithRetry(ctx, opFetchAppAccessToken, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	})
	if err != nil {
		return "", err
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("empty access_token in response")
	}
	return tr.AccessToken, nil
}

// ParseTokenResponse parses oauth/access_token JSON (for tests).
func ParseTokenResponse(r io.Reader) (string, error) {
	var tr tokenResponse
	if err := json.NewDecoder(r).Decode(&tr); err != nil {
		return "", err
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("empty access_token")
	}
	return tr.AccessToken, nil
}
