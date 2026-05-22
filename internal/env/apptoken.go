package env

// AppTokenConfig holds env for fbi-app-token CLI.
type AppTokenConfig struct {
	AppID     string
	AppSecret string
	Debug     bool
}

func LoadAppTokenConfig() (AppTokenConfig, error) {
	if err := ApplyBuildEnv(); err != nil {
		return AppTokenConfig{}, err
	}

	appID, err := Required("FB_APP_ID")
	if err != nil {
		return AppTokenConfig{}, err
	}
	appSecret, err := Required("FB_APP_SECRET")
	if err != nil {
		return AppTokenConfig{}, err
	}
	return AppTokenConfig{
		AppID:     appID,
		AppSecret: appSecret,
		Debug:     BoolEnv("DEBUG", false),
	}, nil
}
