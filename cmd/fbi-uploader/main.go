package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/TAPCLAP/fbi-uploader/internal/env"
	"github.com/TAPCLAP/fbi-uploader/internal/facebook"
	"github.com/TAPCLAP/fbi-uploader/internal/ziputil"
)

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

	logger.Info("repacking zip with config.json", slog.String("source", cfg.ZipPath))
	repackedZip, cleanup, err := ziputil.RepackWithConfig(cfg.ZipPath, cfg.ConfigJSON)
	if err != nil {
		logger.Error("repack failed", slog.String("error", err.Error()))
		return 1
	}
	defer cleanup()
	logger.Debug("repacked zip created", slog.String("path", repackedZip))

	retryCfg := env.LoadAPIRetryConfig()
	client := facebook.NewClient(facebook.RetryConfig{
		MaxAttempts:  retryCfg.MaxAttempts,
		InitialDelay: retryCfg.InitialDelay,
	}, logger)
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

func newLogger(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
