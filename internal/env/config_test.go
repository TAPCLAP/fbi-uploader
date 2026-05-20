package env

import (
	"os"
	"testing"
)

func TestBuildComment(t *testing.T) {
	t.Setenv("COMMENT_AREA", "")
	t.Setenv("COMMENT_BACKEND_URL", "")
	t.Setenv("COMMENT_COMMIT", "")
	t.Setenv("COMMENT_REF", "")
	t.Setenv("COMMENT_CDN_URL", "")
	t.Setenv("COMMENT_EXTRA_INFO", "")

	if got := BuildComment(); got != "" {
		t.Fatalf("expected empty comment, got %q", got)
	}

	t.Setenv("COMMENT_AREA", "stand")
	t.Setenv("COMMENT_BACKEND_URL", "https://api.example.dev")
	t.Setenv("COMMENT_COMMIT", "abc123")
	t.Setenv("COMMENT_REF", "main")
	t.Setenv("COMMENT_CDN_URL", "https://cdn.example.dev/")
	t.Setenv("COMMENT_EXTRA_INFO", "extra note")

	want := "area: stand, backend_url: https://api.example.dev, commit: abc123, ref: main, cdn: https://cdn.example.dev/, extra note"
	if got := BuildComment(); got != want {
		t.Fatalf("BuildComment() = %q, want %q", got, want)
	}
}

func TestBuildCommentPartial(t *testing.T) {
	t.Setenv("COMMENT_AREA", "prod")
	t.Setenv("COMMENT_BACKEND_URL", "")
	t.Setenv("COMMENT_COMMIT", "deadbeef")
	t.Setenv("COMMENT_REF", "")
	t.Setenv("COMMENT_CDN_URL", "")
	t.Setenv("COMMENT_EXTRA_INFO", "")

	want := "area: prod, commit: deadbeef"
	if got := BuildComment(); got != want {
		t.Fatalf("BuildComment() = %q, want %q", got, want)
	}
}

func TestLoadUploaderConfig_pushRequiresAppToken(t *testing.T) {
	t.Setenv("FB_APP_ID", "123")
	t.Setenv("FB_USER_ACCESS_TOKEN", "user")
	t.Setenv("FBINSTANT_ZIP_PATH", "/tmp/x.zip")
	t.Setenv("CONFIG_JSON", `{}`)
	t.Setenv("PUSH_TO_PRODUCTION", "true")
	t.Setenv("FB_APP_ACCESS_TOKEN", "")

	_, err := LoadUploaderConfig()
	if err == nil {
		t.Fatal("expected error when push enabled without app token")
	}
}

func TestLoadUploaderConfig_noPushWithoutAppToken(t *testing.T) {
	t.Setenv("FB_APP_ID", "123")
	t.Setenv("FB_USER_ACCESS_TOKEN", "user")
	t.Setenv("FBINSTANT_ZIP_PATH", "/tmp/x.zip")
	t.Setenv("CONFIG_JSON", `{}`)
	t.Setenv("PUSH_TO_PRODUCTION", "false")
	os.Unsetenv("FB_APP_ACCESS_TOKEN")

	cfg, err := LoadUploaderConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppAccessToken != "" {
		t.Fatalf("expected empty app token, got %q", cfg.AppAccessToken)
	}
}
