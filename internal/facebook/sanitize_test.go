package facebook

import "testing"

func TestSanitizeSecrets(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "url access_token",
			in:   `Post "https://graph.facebook.com/v24.0/123/uploads?access_token=EAAsecret123&file_name=x.zip": dial tcp: timeout`,
			want: `Post "https://graph.facebook.com/v24.0/123/uploads?access_token=[REDACTED]&file_name=x.zip": dial tcp: timeout`,
		},
		{
			name: "url client_secret",
			in:   `Get "https://graph.facebook.com/oauth/access_token?client_id=1&client_secret=topsecret&grant_type=client_credentials": timeout`,
			want: `Get "https://graph.facebook.com/oauth/access_token?client_id=1&client_secret=[REDACTED]&grant_type=client_credentials": timeout`,
		},
		{
			name: "json access_token",
			in:   `facebook api error (status 400): {"error":{"message":"bad"},"access_token":"EAAsecret123"}`,
			want: `facebook api error (status 400): {"error":{"message":"bad"},"access_token":"[REDACTED]"}`,
		},
		{
			name: "oauth user token",
			in:   `create upload session: authorization failed OAuth EAAuserTokenValue here`,
			want: `create upload session: authorization failed OAuth [REDACTED] here`,
		},
		{
			name: "oauth app token",
			in:   `push to production: OAuth 123456789|app-secret-value failed`,
			want: `push to production: OAuth [REDACTED] failed`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeSecrets(tc.in); got != tc.want {
				t.Fatalf("sanitizeSecrets() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAPIError_redactsSecretsInBody(t *testing.T) {
	t.Parallel()

	err := &APIError{
		StatusCode: 400,
		Body:       `{"access_token":"EAAsecret123","error":{"message":"invalid"}}`,
	}
	if got := err.Error(); got != `facebook api error (status 400): {"access_token":"[REDACTED]","error":{"message":"invalid"}}` {
		t.Fatalf("Error() = %q", got)
	}
}

func TestOperationError_redactsURLToken(t *testing.T) {
	t.Parallel()

	err := opError(opCreateUploadSession, errorStringForTest(`Post "https://graph.facebook.com/uploads?access_token=EAAsecret": timeout`))

	got := err.Error()
	want := `create upload session: Post "https://graph.facebook.com/uploads?access_token=[REDACTED]": timeout`
	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

type errorStringForTest string

func (e errorStringForTest) Error() string { return string(e) }
