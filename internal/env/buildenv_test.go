package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyBuildEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "build.env")
	content := `# comment
COMMENT_AREA=from-file
export COMMENT_COMMIT=abc123
COMMENT_REF="main"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	os.Unsetenv("COMMENT_AREA")
	os.Unsetenv("COMMENT_COMMIT")
	os.Unsetenv("COMMENT_REF")
	os.Unsetenv("COMMENT_BACKEND_URL")
	t.Setenv("COMMENT_BACKEND_URL", "preset")

	t.Setenv("BUILD_ENV_PATH", path)
	if err := ApplyBuildEnv(); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("COMMENT_AREA"); got != "from-file" {
		t.Fatalf("COMMENT_AREA = %q, want from-file", got)
	}
	if got := os.Getenv("COMMENT_COMMIT"); got != "abc123" {
		t.Fatalf("COMMENT_COMMIT = %q, want abc123", got)
	}
	if got := os.Getenv("COMMENT_REF"); got != "main" {
		t.Fatalf("COMMENT_REF = %q, want main", got)
	}
	if got := os.Getenv("COMMENT_BACKEND_URL"); got != "preset" {
		t.Fatalf("COMMENT_BACKEND_URL = %q, want preset (not overwritten)", got)
	}
}

func TestApplyBuildEnv_emptyPath(t *testing.T) {
	os.Unsetenv("BUILD_ENV_PATH")
	if err := ApplyBuildEnv(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyBuildEnv_invalidLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.env")
	if err := os.WriteFile(path, []byte("not-a-valid-line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BUILD_ENV_PATH", path)
	if err := ApplyBuildEnv(); err == nil {
		t.Fatal("expected error for invalid line")
	}
}

func TestLoadUploaderConfig_fromBuildEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "build.env")
	content := `COMMENT_AREA=stand
COMMENT_COMMIT=deadbeef
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("BUILD_ENV_PATH", path)
	t.Setenv("FB_APP_ID", "123")
	t.Setenv("FB_USER_ACCESS_TOKEN", "user")
	t.Setenv("FBINSTANT_ZIP_PATH", "/tmp/x.zip")
	t.Setenv("CONFIG_JSON", `{}`)
	os.Unsetenv("COMMENT_AREA")
	os.Unsetenv("COMMENT_COMMIT")
	os.Unsetenv("COMMENT_REF")
	os.Unsetenv("COMMENT_BACKEND_URL")
	os.Unsetenv("COMMENT_CDN_URL")
	os.Unsetenv("COMMENT_EXTRA_INFO")

	cfg, err := LoadUploaderConfig()
	if err != nil {
		t.Fatal(err)
	}
	want := "area: stand, commit: deadbeef"
	if cfg.Comment != want {
		t.Fatalf("Comment = %q, want %q", cfg.Comment, want)
	}
}
