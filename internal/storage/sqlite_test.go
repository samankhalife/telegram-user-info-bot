package storage

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestStorePersistsFullUpdateAndResponse(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	update := tgbotapi.Update{UpdateID: 42, Message: &tgbotapi.Message{
		MessageID: 8,
		Text:      "private text",
		From:      &tgbotapi.User{ID: 100, FirstName: "Sam"},
		Chat:      &tgbotapi.Chat{ID: 200, Type: "private"},
		Contact:   &tgbotapi.Contact{PhoneNumber: "+123"},
	}}
	id, err := store.SaveUpdate(context.Background(), update)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveResponse(context.Background(), Response{
		InteractionID: id,
		ResponseText:  "answer",
		Status:        "sent",
	}); err != nil {
		t.Fatal(err)
	}

	var payload, response string
	if err := store.db.QueryRow("SELECT request_json FROM interactions WHERE id = ?", id).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(payload)) {
		t.Fatalf("stored update is not JSON: %s", payload)
	}
	if err := store.db.QueryRow("SELECT response_text FROM responses WHERE interaction_id = ?", id).Scan(&response); err != nil {
		t.Fatal(err)
	}
	if response != "answer" {
		t.Fatalf("response = %q", response)
	}
}

func TestSaveUpdateIsIdempotent(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	update := tgbotapi.Update{UpdateID: 7, Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 1}}}
	first, err := store.SaveUpdate(context.Background(), update)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.SaveUpdate(context.Background(), update)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("duplicate IDs differ: %d != %d", first, second)
	}
}

func TestDeleteOlderThanCascadesResponses(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	id, err := store.SaveUpdate(context.Background(), tgbotapi.Update{UpdateID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveResponse(context.Background(), Response{InteractionID: id, ResponseText: "x", Status: "sent"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec("UPDATE interactions SET received_at = ? WHERE id = ?", time.Now().AddDate(0, 0, -31).UTC().Format(time.RFC3339Nano), id); err != nil {
		t.Fatal(err)
	}

	deleted, err := store.DeleteOlderThan(context.Background(), time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	var count int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM responses").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("responses remain after cascade: %d", count)
	}
}
