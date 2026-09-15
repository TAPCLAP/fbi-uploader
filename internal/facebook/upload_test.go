package facebook

import (
	"net/http"
	"testing"
)

func TestUploadNamespace(t *testing.T) {
	t.Parallel()

	cases := []struct {
		token string
		want  string
		ok    bool
	}{
		{token: "GGabc123", want: "gg_graph_api", ok: true},
		{token: "GG|123|secret", want: "gg_graph_api", ok: true},
		{token: "EAABWzLixnjYBO", want: "fb_game_bundle", ok: true},
		{token: "EAA...", want: "fb_game_bundle", ok: true},
		{token: "eaabwz", ok: false},
		{token: "ggabc", ok: false},
		{token: "", ok: false},
		{token: "Bearer EAA", ok: false},
	}

	for _, tc := range cases {
		got, err := uploadNamespace(tc.token)
		if tc.ok {
			if err != nil {
				t.Fatalf("token %q: %v", tc.token, err)
			}
			if got != tc.want {
				t.Fatalf("token %q: got %q want %q", tc.token, got, tc.want)
			}
			continue
		}
		if err == nil {
			t.Fatalf("token %q: expected error", tc.token)
		}
	}
}

func TestRuploadURL(t *testing.T) {
	t.Parallel()

	got, err := ruploadURL("GGtoken", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://rupload.facebook.com/gg_graph_api/upload:abc123" {
		t.Fatalf("got %q", got)
	}

	got, err = ruploadURL("EAAtoken", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://rupload.facebook.com/fb_game_bundle/upload:abc123" {
		t.Fatalf("got %q", got)
	}

	if _, err := ruploadURL("bad-token", "abc123"); err == nil {
		t.Fatal("expected error for unsupported token prefix")
	}
}

func TestParseBundleInstanceID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		body string
		want int64
	}{
		{`{"bundle_instance_id":123}`, 123},
		{`{"success":true,"bundle_instance_id":456}`, 456},
	}

	for _, tc := range cases {
		got, err := parseBundleInstanceID([]byte(tc.body))
		if err != nil {
			t.Fatalf("body %s: %v", tc.body, err)
		}
		if got != tc.want {
			t.Fatalf("body %s: got %d want %d", tc.body, got, tc.want)
		}
	}
}

func TestCheckRuploadResponse_partialRequestError(t *testing.T) {
	t.Parallel()

	body := []byte(`{"debug_info":{"retriable":true,"type":"PartialRequestError","message":"Partial request (did not match length of file)"}}`)
	err := checkRuploadResponse(&http.Response{StatusCode: http.StatusOK}, body)
	if err == nil {
		t.Fatal("expected error")
	}
	if !isRetryable(err) {
		t.Fatalf("expected retryable error, got %v", err)
	}
}
