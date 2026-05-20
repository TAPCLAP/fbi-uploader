package facebook

import (
	"net/http"
	"testing"
)

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
