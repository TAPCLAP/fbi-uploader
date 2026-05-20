package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/TAPCLAP/fbi-uploader/internal/env"
	"github.com/TAPCLAP/fbi-uploader/internal/facebook"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := env.LoadAppTokenConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}

	logger := newLogger(cfg.Debug)

	retryCfg := env.LoadAPIRetryConfig()
	client := facebook.NewClient(facebook.RetryConfig{
		MaxAttempts:  retryCfg.MaxAttempts,
		InitialDelay: retryCfg.InitialDelay,
	}, logger)
	token, err := client.FetchAppAccessToken(context.Background(), cfg.AppID, cfg.AppSecret)
	if err != nil {
		logger.Error("fetch app access token failed", slog.String("error", err.Error()))
		return 1
	}

	_, err = fmt.Fprint(os.Stdout, token)
	if err != nil {
		logger.Error("write stdout failed", slog.String("error", err.Error()))
		return 1
	}
	return 0
}

func newLogger(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
