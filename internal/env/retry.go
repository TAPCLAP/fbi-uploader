package env

import "time"

// APIRetryConfig holds retry settings for Facebook API HTTP requests.
type APIRetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
}

func LoadAPIRetryConfig() APIRetryConfig {
	delayMS := IntEnv("FB_API_RETRY_DELAY_MS", 1000)
	if delayMS < 0 {
		delayMS = 1000
	}
	return APIRetryConfig{
		MaxAttempts:  IntEnv("FB_API_RETRIES", 10),
		InitialDelay: time.Duration(delayMS) * time.Millisecond,
	}
}
