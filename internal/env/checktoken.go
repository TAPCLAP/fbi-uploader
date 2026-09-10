package env

import (
	"os"
	"strings"
)

// CheckTokenConfig holds env for fbi-check-token CLI.
type CheckTokenConfig struct {
	UserAccessToken string
	GraphAPIVersion string
	Debug           bool
}

func LoadCheckTokenConfig() (CheckTokenConfig, error) {
	if err := ApplyBuildEnv(); err != nil {
		return CheckTokenConfig{}, err
	}

	userToken, err := Required("FB_USER_ACCESS_TOKEN")
	if err != nil {
		return CheckTokenConfig{}, err
	}

	return CheckTokenConfig{
		UserAccessToken: userToken,
		GraphAPIVersion: strings.TrimSpace(os.Getenv("FB_GRAPH_API_VERSION")),
		Debug:           BoolEnv("DEBUG", false),
	}, nil
}
