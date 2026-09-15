package facebook

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDebugTokenURL_defaults(t *testing.T) {
	raw, err := DebugTokenURL(DebugTokenParams{InputToken: "user-token"})
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme+"://"+u.Host+u.Path != "https://graph.facebook.com/debug_token" {
		t.Fatalf("endpoint = %s", u.Scheme+"://"+u.Host+u.Path)
	}
	q := u.Query()
	if q.Get("input_token") != "user-token" || q.Get("access_token") != "user-token" {
		t.Fatalf("query = %v", q)
	}
}

func TestDebugTokenURL_appTokenAndVersion(t *testing.T) {
	raw, err := DebugTokenURL(DebugTokenParams{
		InputToken:      "user-token",
		AccessToken:     "app-token",
		GraphAPIVersion: "v26.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/v26.0/debug_token" {
		t.Fatalf("path = %q", u.Path)
	}
	q := u.Query()
	if q.Get("input_token") != "user-token" || q.Get("access_token") != "app-token" {
		t.Fatalf("query = %v", q)
	}
}

func TestDebugTokenEndpoint(t *testing.T) {
	if got := DebugTokenEndpoint(""); got != "https://graph.facebook.com/debug_token" {
		t.Fatalf("got %q", got)
	}
	if got := DebugTokenEndpoint(" v26.0 "); got != "https://graph.facebook.com/v26.0/debug_token" {
		t.Fatalf("got %q", got)
	}
}

func TestParseDebugTokenResponse_withExpiry(t *testing.T) {
	body := `{"data":{"app_id":"123","type":"USER","is_valid":true,"expires_at":1700000000}}`
	info, err := ParseDebugTokenResponse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if !info.Valid {
		t.Fatal("expected valid token")
	}
	if info.NeverExpires {
		t.Fatal("expected token with expiry")
	}
	want := time.Unix(1700000000, 0).UTC()
	if !info.ExpiresAt.Equal(want) {
		t.Fatalf("expires_at: got %v want %v", info.ExpiresAt, want)
	}
}

func TestParseDebugTokenResponse_neverExpires(t *testing.T) {
	cases := []string{
		`{"data":{"is_valid":true,"expires_at":0}}`,
		`{"data":{"is_valid":true}}`,
	}
	for _, body := range cases {
		info, err := ParseDebugTokenResponse([]byte(body))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", body, err)
		}
		if info.Incomplete || !info.Valid || !info.NeverExpires {
			t.Fatalf("%s: got incomplete=%v valid=%v neverExpires=%v", body, info.Incomplete, info.Valid, info.NeverExpires)
		}
	}
}

func TestParseDebugTokenResponse_invalid(t *testing.T) {
	info, err := ParseDebugTokenResponse([]byte(`{"data":{"is_valid":false}}`))
	if err != nil {
		t.Fatal(err)
	}
	if info.Incomplete || info.Valid {
		t.Fatalf("got incomplete=%v valid=%v", info.Incomplete, info.Valid)
	}
	if !info.IsExpired(time.Now()) {
		t.Fatal("expected invalid token to count as expired")
	}
}

func TestParseDebugTokenResponse_incomplete(t *testing.T) {
	cases := []string{
		`{"data":[]}`,
		`{"data":{}}`,
		`{"data":null}`,
		`{}`,
		`{"data":{"expires_at":1700000000}}`,
	}
	for _, body := range cases {
		info, err := ParseDebugTokenResponse([]byte(body))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", body, err)
		}
		if !info.Incomplete {
			t.Fatalf("%s: expected Incomplete", body)
		}
		if info.IsExpired(time.Unix(1700000100, 0).UTC()) {
			t.Fatalf("%s: incomplete response should not count as expired", body)
		}
	}
}

func TestUserTokenInfo_IsExpired(t *testing.T) {
	now := time.Unix(1700000100, 0).UTC()

	tests := []struct {
		name string
		info UserTokenInfo
		want bool
	}{
		{
			name: "incomplete",
			info: UserTokenInfo{Incomplete: true},
			want: false,
		},
		{
			name: "invalid",
			info: UserTokenInfo{Valid: false},
			want: true,
		},
		{
			name: "never expires",
			info: UserTokenInfo{Valid: true, NeverExpires: true},
			want: false,
		},
		{
			name: "not yet expired",
			info: UserTokenInfo{Valid: true, ExpiresAt: now.Add(time.Hour)},
			want: false,
		},
		{
			name: "expired",
			info: UserTokenInfo{Valid: true, ExpiresAt: now.Add(-time.Hour)},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.IsExpired(now); got != tt.want {
				t.Fatalf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatRemaining(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0d 0h 0m"},
		{-time.Hour, "0d 0h 0m"},
		{45 * time.Second, "0d 0h 0m"},
		{90 * time.Minute, "0d 1h 30m"},
		{2*24*time.Hour + 5*time.Hour + 7*time.Minute, "2d 5h 7m"},
	}
	for _, tc := range cases {
		if got := FormatRemaining(tc.d); got != tc.want {
			t.Fatalf("FormatRemaining(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestParseDebugTokenResponse_invalidJSON(t *testing.T) {
	_, err := ParseDebugTokenResponseReader(strings.NewReader(`{`))
	if err == nil {
		t.Fatal("expected error")
	}
}
