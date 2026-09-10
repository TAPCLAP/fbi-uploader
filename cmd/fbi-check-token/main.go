package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/TAPCLAP/fbi-uploader/internal/env"
	"github.com/TAPCLAP/fbi-uploader/internal/facebook"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := env.LoadCheckTokenConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}

	logger := newLogger(cfg.Debug)
	ctx := context.Background()

	retryCfg := env.LoadAPIRetryConfig()
	timeoutCfg := env.LoadAPITimeoutConfig()
	client := facebook.NewClient(facebook.RetryConfig{
		MaxAttempts:  retryCfg.MaxAttempts,
		InitialDelay: retryCfg.InitialDelay,
	}, facebook.TimeoutConfig{
		ConnectTimeout:  timeoutCfg.ConnectTimeout,
		ResponseTimeout: timeoutCfg.ResponseTimeout,
		RequestTimeout:  timeoutCfg.RequestTimeout,
	}, logger)

	params := facebook.DebugTokenParams{
		InputToken:      cfg.UserAccessToken,
		GraphAPIVersion: cfg.GraphAPIVersion,
	}

	logger.Info("inspecting user access token",
		slog.String("endpoint", facebook.DebugTokenEndpoint(cfg.GraphAPIVersion)),
	)

	body, fetchErr := client.FetchDebugToken(ctx, params)
	if err := facebook.WritePrettyJSON(os.Stdout, body); err != nil {
		logger.Error("write stdout failed", slog.String("error", err.Error()))
		return 1
	}
	if fetchErr != nil {
		logger.Error("check user access token failed", slog.String("error", fetchErr.Error()))
		return 1
	}

	info, err := facebook.ParseDebugTokenResponse(body)
	if err != nil {
		logger.Error("parse debug_token response failed", slog.String("error", err.Error()))
		return 1
	}

	if info.NeverExpires {
		logger.Info("user access token does not expire")
	} else {
		logger.Info("user access token expires at",
			slog.Time("expires_at", info.ExpiresAt),
			slog.String("expires_in", facebook.FormatRemaining(time.Until(info.ExpiresAt))),
		)
	}

	if !info.IsExpired(time.Now()) {
		return 0
	}

	if info.NeverExpires {
		logger.Error("user access token is invalid")
	} else {
		logger.Error("user access token expired", slog.Time("expires_at", info.ExpiresAt))
	}
	return 1
}

func newLogger(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
