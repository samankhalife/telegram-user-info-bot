package config

import (
	"log/slog"
	"testing"
)

func TestLoadRequiresToken(t *testing.T) {
	t.Setenv("BOT_TOKEN", "")
	t.Setenv("LOG_LEVEL", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want missing token error")
	}
}

func TestLoad(t *testing.T) {
	t.Setenv("BOT_TOKEN", " test-token ")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("RETENTION_DAYS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Token != "test-token" {
		t.Fatalf("Token = %q, want test-token", cfg.Token)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("LogLevel = %v, want debug", cfg.LogLevel)
	}
	if cfg.DatabasePath != "./data/bot.db" || cfg.RetentionDays != 30 {
		t.Fatalf("unexpected storage defaults: %#v", cfg)
	}
}

func TestLoadRejectsInvalidRetention(t *testing.T) {
	t.Setenv("BOT_TOKEN", "test-token")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("RETENTION_DAYS", "0")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid retention error")
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("BOT_TOKEN", "test-token")
	t.Setenv("LOG_LEVEL", "verbose")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid log level error")
	}
}
