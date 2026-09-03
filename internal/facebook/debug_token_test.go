package facebook

import (
	"strings"
	"testing"
	"time"
)

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
	body := `{"data":{"is_valid":true,"expires_at":0}}`
	info, err := ParseDebugTokenResponse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if !info.Valid || !info.NeverExpires {
		t.Fatalf("got valid=%v neverExpires=%v", info.Valid, info.NeverExpires)
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
