package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Token         string
	LogLevel      slog.Level
	DatabasePath  string
	RetentionDays int
}

func Load() (Config, error) {
	token := strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	if token == "" {
		return Config{}, fmt.Errorf("BOT_TOKEN is required")
	}

	level, err := parseLogLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		return Config{}, err
	}

	databasePath := strings.TrimSpace(os.Getenv("DATABASE_PATH"))
	if databasePath == "" {
		databasePath = "./data/bot.db"
	}

	retentionDays := 30
	if value := strings.TrimSpace(os.Getenv("RETENTION_DAYS")); value != "" {
		retentionDays, err = strconv.Atoi(value)
		if err != nil || retentionDays <= 0 {
			return Config{}, fmt.Errorf("RETENTION_DAYS must be a positive integer")
		}
	}

	return Config{
		Token:         token,
		LogLevel:      level,
		DatabasePath:  databasePath,
		RetentionDays: retentionDays,
	}, nil
}

func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid LOG_LEVEL %q: use debug, info, warn, or error", value)
	}
}
