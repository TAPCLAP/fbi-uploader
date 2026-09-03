package env

import "time"

const (
	defaultConnectTimeoutMS  = 30_000
	defaultResponseTimeoutMS = 600_000
	defaultRequestTimeoutMS  = 600_000
)

// APITimeoutConfig holds HTTP timeouts for Facebook API requests.
type APITimeoutConfig struct {
	ConnectTimeout  time.Duration
	ResponseTimeout time.Duration
	RequestTimeout  time.Duration
}

func LoadAPITimeoutConfig() APITimeoutConfig {
	return APITimeoutConfig{
		ConnectTimeout:  durationMSEnv("FB_API_CONNECT_TIMEOUT_MS", defaultConnectTimeoutMS),
		ResponseTimeout: durationMSEnv("FB_API_RESPONSE_TIMEOUT_MS", defaultResponseTimeoutMS),
		RequestTimeout:  durationMSEnv("FB_API_REQUEST_TIMEOUT_MS", defaultRequestTimeoutMS),
	}
}

func durationMSEnv(key string, defaultMS int) time.Duration {
	ms := IntEnv(key, defaultMS)
	if ms < 0 {
		ms = defaultMS
	}
	return time.Duration(ms) * time.Millisecond
}
