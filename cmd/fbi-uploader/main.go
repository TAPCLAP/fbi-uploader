package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/TAPCLAP/fbi-uploader/internal/env"
	"github.com/TAPCLAP/fbi-uploader/internal/facebook"
	"github.com/TAPCLAP/fbi-uploader/internal/ziputil"
)

const expiredTokenExitDelay = 60 * time.Second

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := env.LoadUploaderConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}

	logger := newLogger(cfg.Debug)
	ctx := context.Background()

	var repackedZip string
	var cleanup func()
	switch {
	case cfg.ConfigJSON == "" && cfg.ZipPath != "":
		logger.Info("using bundle zip as-is", slog.String("source", cfg.ZipPath))
		repackedZip = cfg.ZipPath
		cleanup = func() {}
	case cfg.ZipPathDir != "":
		if cfg.ConfigJSON != "" {
			logger.Info("packing directory with config.json", slog.String("source", cfg.ZipPathDir))
		} else {
			logger.Info("packing directory", slog.String("source", cfg.ZipPathDir))
		}
		repackedZip, cleanup, err = ziputil.PackDirWithConfig(cfg.ZipPathDir, cfg.ConfigJSON)
	default:
		logger.Info("repacking zip with config.json", slog.String("source", cfg.ZipPath))
		repackedZip, cleanup, err = ziputil.RepackWithConfig(cfg.ZipPath, cfg.ConfigJSON)
	}
	if err != nil {
		logger.Error("bundle pack failed", slog.String("error", err.Error()))
		return 1
	}
	defer cleanup()
	logger.Debug("repacked zip created", slog.String("path", repackedZip))

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

	if code := checkUserAccessToken(ctx, client, logger, cfg.UserAccessToken); code != 0 {
		return code
	}

	uploadParams := facebook.UploadParams{
		AppID:           cfg.AppID,
		GraphAPIVersion: cfg.GraphAPIVersion,
		UserAccessToken: cfg.UserAccessToken,
		ZipPath:         repackedZip,
		Comment:         cfg.Comment,
	}

	logger.Info("uploading bundle to facebook")
	result, err := client.UploadBundleWithRetry(ctx, uploadParams)
	if err != nil {
		logger.Error("upload failed", slog.String("error", err.Error()))
		return 1
	}

	logger.Info("upload complete",
		slog.Int64("bundle_instance_id", result.BundleInstanceID),
		slog.String("session_id", result.SessionID),
	)

	if !cfg.PushToProduction {
		logger.Info("skipping push to production (PUSH_TO_PRODUCTION is false)")
		return 0
	}

	logger.Info("pushing bundle to production")
	err = client.PushToProduction(ctx, facebook.PushParams{
		AppID:            cfg.AppID,
		AppAccessToken:   cfg.AppAccessToken,
		BundleInstanceID: result.BundleInstanceID,
	})
	if err != nil {
		logger.Error("push to production failed", slog.String("error", err.Error()))
		return 1
	}

	logger.Info("push to production complete", slog.Int64("bundle_instance_id", result.BundleInstanceID))
	return 0
}

func checkUserAccessToken(ctx context.Context, client *facebook.Client, logger *slog.Logger, userToken string) int {
	info, err := client.InspectUserAccessToken(ctx, userToken)
	if err != nil {
		logger.Error("check user access token failed", slog.String("error", err.Error()))
		return 1
	}

	if info.NeverExpires {
		logger.Info("user access token does not expire")
	} else {
		logger.Info("user access token expires at", slog.Time("expires_at", info.ExpiresAt))
	}

	if !info.IsExpired(time.Now()) {
		return 0
	}

	if info.NeverExpires {
		logger.Error("user access token is invalid")
	} else {
		logger.Error("user access token expired", slog.Time("expires_at", info.ExpiresAt))
	}
	logger.Info("waiting before exit", slog.Duration("delay", expiredTokenExitDelay))
	time.Sleep(expiredTokenExitDelay)
	return 1
}

func newLogger(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
