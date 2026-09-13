package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Response struct {
	InteractionID     int64
	ResponseText      string
	ResponseMarkup    any
	ParseMode         string
	Status            string
	TelegramMessageID int
	ErrorMessage      string
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("database path is required")
	}
	if path != ":memory:" && filepath.Dir(path) != "." {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) migrate(ctx context.Context) error {
	const schema = `
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;

CREATE TABLE IF NOT EXISTS interactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_update_id INTEGER NOT NULL UNIQUE,
    update_type TEXT NOT NULL,
    user_id INTEGER,
    chat_id INTEGER,
    message_id INTEGER,
    request_json TEXT NOT NULL,
    received_at TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'received',
    error_message TEXT
);

CREATE TABLE IF NOT EXISTS responses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    interaction_id INTEGER NOT NULL,
    response_text TEXT NOT NULL,
    response_markup_json TEXT,
    parse_mode TEXT,
    sent_at TEXT NOT NULL,
    status TEXT NOT NULL,
    telegram_message_id INTEGER,
    error_message TEXT,
    FOREIGN KEY (interaction_id) REFERENCES interactions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_interactions_received_at ON interactions(received_at);
CREATE INDEX IF NOT EXISTS idx_interactions_user_id ON interactions(user_id);
CREATE INDEX IF NOT EXISTS idx_interactions_chat_id ON interactions(chat_id);
CREATE INDEX IF NOT EXISTS idx_responses_interaction_id ON responses(interaction_id);
`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	return nil
}

func (s *Store) SaveUpdate(ctx context.Context, update tgbotapi.Update) (int64, error) {
	payload, err := json.Marshal(update)
	if err != nil {
		return 0, fmt.Errorf("encode update: %w", err)
	}

	message, updateType := messageFromUpdate(update)
	var userID, chatID int64
	var messageID int
	if message != nil {
		messageID = message.MessageID
		if message.From != nil {
			userID = message.From.ID
		}
		if message.Chat != nil {
			chatID = message.Chat.ID
		}
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO interactions (
    telegram_update_id, update_type, user_id, chat_id, message_id, request_json, received_at
) VALUES (?, ?, NULLIF(?, 0), NULLIF(?, 0), NULLIF(?, 0), ?, ?)
ON CONFLICT(telegram_update_id) DO NOTHING`,
		update.UpdateID, updateType, userID, chatID, messageID, string(payload), time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, fmt.Errorf("save update: %w", err)
	}

	var id int64
	if err := s.db.QueryRowContext(ctx, "SELECT id FROM interactions WHERE telegram_update_id = ?", update.UpdateID).Scan(&id); err != nil {
		return 0, fmt.Errorf("find existing update: %w", err)
	}
	return id, nil
}

func (s *Store) SaveResponse(ctx context.Context, response Response) error {
	var markupJSON []byte
	var err error
	if response.ResponseMarkup != nil {
		markupJSON, err = json.Marshal(response.ResponseMarkup)
		if err != nil {
			return fmt.Errorf("encode response markup: %w", err)
		}
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO responses (
    interaction_id, response_text, response_markup_json, parse_mode, sent_at, status,
    telegram_message_id, error_message
) VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULLIF(?, 0), NULLIF(?, ''))`,
		response.InteractionID, response.ResponseText, string(markupJSON), response.ParseMode,
		time.Now().UTC().Format(time.RFC3339Nano), response.Status, response.TelegramMessageID, response.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("save response: %w", err)
	}
	return nil
}

func (s *Store) MarkInteraction(ctx context.Context, interactionID int64, status, errorMessage string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE interactions SET status = ?, error_message = NULLIF(?, '') WHERE id = ?",
		status, errorMessage, interactionID,
	)
	if err != nil {
		return fmt.Errorf("mark interaction: %w", err)
	}
	return nil
}

func (s *Store) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, "DELETE FROM interactions WHERE received_at < ?", cutoff.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("delete expired interactions: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count deleted interactions: %w", err)
	}
	return count, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	if err := s.db.Close(); err != nil && !errors.Is(err, sql.ErrConnDone) {
		return fmt.Errorf("close database: %w", err)
	}
	return nil
}

func messageFromUpdate(update tgbotapi.Update) (*tgbotapi.Message, string) {
	switch {
	case update.Message != nil:
		return update.Message, "message"
	case update.EditedMessage != nil:
		return update.EditedMessage, "edited_message"
	case update.ChannelPost != nil:
		return update.ChannelPost, "channel_post"
	case update.EditedChannelPost != nil:
		return update.EditedChannelPost, "edited_channel_post"
	default:
		return nil, "unsupported"
	}
}
