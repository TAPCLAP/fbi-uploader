package env

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// UploaderConfig holds env for fbi-uploader CLI.
type UploaderConfig struct {
	AppID            string
	UserAccessToken  string
	AppAccessToken   string
	ZipPath          string
	ZipPathDir       string
	ConfigJSON       string
	GraphAPIVersion  string
	PushToProduction bool
	Debug            bool
	Comment          string
}

func LoadUploaderConfig() (UploaderConfig, error) {
	if err := ApplyBuildEnv(); err != nil {
		return UploaderConfig{}, err
	}

	appID, err := Required("FB_APP_ID")
	if err != nil {
		return UploaderConfig{}, err
	}
	userToken, err := Required("FB_USER_ACCESS_TOKEN")
	if err != nil {
		return UploaderConfig{}, err
	}
	zipPath, zipPathDir, err := loadBundleSource()
	if err != nil {
		return UploaderConfig{}, err
	}

	configJSON, err := loadConfigJSON()
	if err != nil {
		return UploaderConfig{}, err
	}

	push := BoolEnv("PUSH_TO_PRODUCTION", false)
	var appAccessToken string
	if push {
		appAccessToken, err = Required("FB_APP_ACCESS_TOKEN")
		if err != nil {
			return UploaderConfig{}, fmt.Errorf("PUSH_TO_PRODUCTION is enabled: %w", err)
		}
	}

	return UploaderConfig{
		AppID:            appID,
		UserAccessToken:  userToken,
		AppAccessToken:   appAccessToken,
		ZipPath:          zipPath,
		ZipPathDir:       zipPathDir,
		ConfigJSON:       configJSON,
		GraphAPIVersion:  Getenv("FB_GRAPH_API_VERSION", "v24.0"),
		PushToProduction: push,
		Debug:            BoolEnv("DEBUG", false),
		Comment:          BuildComment(),
	}, nil
}

func loadBundleSource() (zipPath, zipPathDir string, err error) {
	zipPath = strings.TrimSpace(os.Getenv("FBINSTANT_ZIP_PATH"))
	zipPathDir = strings.TrimSpace(os.Getenv("FBINSTANT_ZIP_PATH_DIR"))

	switch {
	case zipPath != "" && zipPathDir != "":
		return "", "", errors.New("set only one of FBINSTANT_ZIP_PATH or FBINSTANT_ZIP_PATH_DIR, not both")
	case zipPath != "":
		return zipPath, "", nil
	case zipPathDir != "":
		return "", zipPathDir, nil
	default:
		return "", "", errors.New("required: set FBINSTANT_ZIP_PATH or FBINSTANT_ZIP_PATH_DIR")
	}
}

func loadConfigJSON() (string, error) {
	inline := strings.TrimSpace(os.Getenv("CONFIG_JSON"))
	filePath := strings.TrimSpace(os.Getenv("CONFIG_JSON_FILE"))

	switch {
	case inline != "" && filePath != "":
		return "", errors.New("set only one of CONFIG_JSON or CONFIG_JSON_FILE, not both")
	case inline != "":
		return inline, nil
	case filePath != "":
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("read CONFIG_JSON_FILE: %w", err)
		}
		return string(data), nil
	default:
		return "", nil
	}
}

// BuildComment assembles the bundle upload comment from optional env vars.
func BuildComment() string {
	type part struct {
		text string
	}
	var parts []part

	if v := strings.TrimSpace(os.Getenv("COMMENT_AREA")); v != "" {
		parts = append(parts, part{"area: " + v})
	}
	if v := strings.TrimSpace(os.Getenv("COMMENT_BACKEND_URL")); v != "" {
		parts = append(parts, part{"backend_url: " + v})
	}
	if v := strings.TrimSpace(os.Getenv("COMMENT_COMMIT")); v != "" {
		parts = append(parts, part{"commit: " + v})
	}
	if v := strings.TrimSpace(os.Getenv("COMMENT_REF")); v != "" {
		parts = append(parts, part{"ref: " + v})
	}
	if v := strings.TrimSpace(os.Getenv("COMMENT_CDN_URL")); v != "" {
		parts = append(parts, part{"cdn: " + v})
	}
	if v := strings.TrimSpace(os.Getenv("COMMENT_EXTRA_INFO")); v != "" {
		parts = append(parts, part{v})
	}

	if len(parts) == 0 {
		return ""
	}

	out := parts[0].text
	for i := 1; i < len(parts); i++ {
		out += ", " + parts[i].text
	}
	return out
}
