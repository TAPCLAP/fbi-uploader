package env

import "testing"

func TestLoadCheckTokenConfig_requiresUserToken(t *testing.T) {
	t.Setenv("FB_USER_ACCESS_TOKEN", "")
	t.Setenv("FB_GRAPH_API_VERSION", "")
	t.Setenv("DEBUG", "")
	t.Setenv("BUILD_ENV_PATH", "")

	_, err := LoadCheckTokenConfig()
	if err == nil {
		t.Fatal("expected error when user token is missing")
	}
}

func TestLoadCheckTokenConfig_minimal(t *testing.T) {
	t.Setenv("FB_USER_ACCESS_TOKEN", "EAA-user")
	t.Setenv("FB_GRAPH_API_VERSION", "")
	t.Setenv("DEBUG", "")
	t.Setenv("BUILD_ENV_PATH", "")

	cfg, err := LoadCheckTokenConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.UserAccessToken != "EAA-user" {
		t.Fatalf("UserAccessToken = %q", cfg.UserAccessToken)
	}
	if cfg.GraphAPIVersion != "" {
		t.Fatalf("GraphAPIVersion = %q, want empty", cfg.GraphAPIVersion)
	}
	if cfg.Debug {
		t.Fatal("expected Debug=false")
	}
}

func TestLoadCheckTokenConfig_optional(t *testing.T) {
	t.Setenv("FB_USER_ACCESS_TOKEN", "EAA-user")
	t.Setenv("FB_GRAPH_API_VERSION", "v26.0")
	t.Setenv("DEBUG", "true")
	t.Setenv("BUILD_ENV_PATH", "")

	cfg, err := LoadCheckTokenConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GraphAPIVersion != "v26.0" {
		t.Fatalf("GraphAPIVersion = %q", cfg.GraphAPIVersion)
	}
	if !cfg.Debug {
		t.Fatal("expected Debug=true")
	}
}
