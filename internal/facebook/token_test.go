package facebook

import (
	"strings"
	"testing"
)

func TestParseTokenResponse(t *testing.T) {
	body := `{"access_token":"EAABwzLixnjYBO","token_type":"bearer"}`
	token, err := ParseTokenResponse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if token != "EAABwzLixnjYBO" {
		t.Fatalf("got %q", token)
	}
}

func TestParseTokenResponse_empty(t *testing.T) {
	_, err := ParseTokenResponse(strings.NewReader(`{"access_token":""}`))
	if err == nil {
		t.Fatal("expected error")
	}
}
