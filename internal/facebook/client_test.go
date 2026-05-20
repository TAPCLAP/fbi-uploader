package facebook

import "testing"

func TestExtractSessionID(t *testing.T) {
	t.Parallel()

	id, err := ExtractSessionID("upload:abc123")
	if err != nil {
		t.Fatal(err)
	}
	if id != "abc123" {
		t.Fatalf("got %q", id)
	}

	_, err = ExtractSessionID("bad")
	if err == nil {
		t.Fatal("expected error for bad format")
	}

	_, err = ExtractSessionID("upload:")
	if err == nil {
		t.Fatal("expected error for empty session id")
	}
}
