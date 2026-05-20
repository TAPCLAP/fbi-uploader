package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type PushParams struct {
	AppID           string
	AppAccessToken  string
	BundleInstanceID int64
}

// PushToProduction pushes an uploaded bundle to production.
func (c *Client) PushToProduction(ctx context.Context, p PushParams) error {
	url := fmt.Sprintf("https://api.facebook.com/instant-games/assets/%s/push-to-production", p.AppID)

	payload, err := json.Marshal(map[string]string{
		"bundle_instance_id": fmt.Sprintf("%d", p.BundleInstanceID),
		// "version_id": fmt.Sprintf("%d", p.BundleInstanceID),
	})
	if err != nil {
		return err
	}

	_, _, err = c.doWithRetry(ctx, opPushToProduction, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("OAuth %s", p.AppAccessToken))
		req.Header.Set("X-API-Version", "1.0.0")
		return req, nil
	})
	return err
}
