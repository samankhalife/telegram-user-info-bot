package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	botapp "github.com/samankhalife/telegram-user-info-bot/internal/bot"
	"github.com/samankhalife/telegram-user-info-bot/internal/config"
	"github.com/samankhalife/telegram-user-info-bot/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	if err := tgbotapi.SetLogger(botapp.NewRedactingLogger(logger, cfg.Token)); err != nil {
		logger.Error("failed to configure Telegram logger", "error", err)
		os.Exit(1)
	}

	store, err := storage.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deleteExpired(ctx, store, cfg.RetentionDays, logger)
	go runRetention(ctx, store, cfg.RetentionDays, logger)

	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		logger.Error("failed to initialize Telegram bot", "error", err)
		os.Exit(1)
	}

	updatesConfig := tgbotapi.NewUpdate(0)
	updatesConfig.Timeout = 60
	updates := api.GetUpdatesChan(updatesConfig)
	handler := botapp.NewHandler(api, store, logger)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	logger.Info("bot started", "username", api.Self.UserName)
	for {
		select {
		case update, ok := <-updates:
			if !ok {
				logger.Info("updates channel closed")
				return
			}
			handler.Handle(update)
		case <-stop:
			cancel()
			api.StopReceivingUpdates()
			logger.Info("bot stopped")
			return
		}
	}
}

func runRetention(ctx context.Context, store *storage.Store, retentionDays int, logger *slog.Logger) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			deleteExpired(ctx, store, retentionDays, logger)
		case <-ctx.Done():
			return
		}
	}
}

func deleteExpired(ctx context.Context, store *storage.Store, retentionDays int, logger *slog.Logger) {
	deleted, err := store.DeleteOlderThan(ctx, time.Now().AddDate(0, 0, -retentionDays))
	if err != nil {
		logger.Error("failed to delete expired interactions", "error", err)
		return
	}
	if deleted > 0 {
		logger.Info("deleted expired interactions", "count", deleted, "retention_days", retentionDays)
	}
}
