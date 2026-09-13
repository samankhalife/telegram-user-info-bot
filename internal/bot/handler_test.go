package bot

import (
	"context"
	"io"
	"log/slog"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/samankhalife/telegram-user-info-bot/internal/storage"
)

type fakeSender struct {
	messages []tgbotapi.MessageConfig
}

func (f *fakeSender) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	if message, ok := c.(tgbotapi.MessageConfig); ok {
		f.messages = append(f.messages, message)
	}
	return tgbotapi.Message{MessageID: 99}, nil
}

type fakeRecorder struct {
	updates   []tgbotapi.Update
	responses []storage.Response
	statuses  []string
}

func (f *fakeRecorder) SaveUpdate(_ context.Context, update tgbotapi.Update) (int64, error) {
	f.updates = append(f.updates, update)
	return 1, nil
}

func (f *fakeRecorder) SaveResponse(_ context.Context, response storage.Response) error {
	f.responses = append(f.responses, response)
	return nil
}

func (f *fakeRecorder) MarkInteraction(_ context.Context, _ int64, status, _ string) error {
	f.statuses = append(f.statuses, status)
	return nil
}

func TestHandlerSendsWelcomeWithConsentKeyboard(t *testing.T) {
	sender := &fakeSender{}
	recorder := &fakeRecorder{}
	handler := NewHandler(sender, recorder, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler.Handle(tgbotapi.Update{Message: &tgbotapi.Message{
		Text: "/start",
		Chat: &tgbotapi.Chat{ID: 123, Type: "private"},
	}})

	if len(sender.messages) != 1 {
		t.Fatalf("expected one message, got %d", len(sender.messages))
	}
	keyboard, ok := sender.messages[0].ReplyMarkup.(tgbotapi.ReplyKeyboardMarkup)
	if !ok || len(keyboard.Keyboard) != 1 || len(keyboard.Keyboard[0]) != 2 {
		t.Fatalf("expected contact/location keyboard, got %#v", sender.messages[0].ReplyMarkup)
	}
	if !keyboard.Keyboard[0][0].RequestContact || !keyboard.Keyboard[0][1].RequestLocation {
		t.Fatalf("keyboard does not request explicit contact/location sharing: %#v", keyboard)
	}
	if len(recorder.updates) != 1 || len(recorder.responses) != 1 || recorder.responses[0].Status != "sent" {
		t.Fatalf("expected stored update and response, got %#v %#v", recorder.updates, recorder.responses)
	}
}

func TestHandlerSendsInfoReport(t *testing.T) {
	sender := &fakeSender{}
	recorder := &fakeRecorder{}
	handler := NewHandler(sender, recorder, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler.Handle(tgbotapi.Update{UpdateID: 9, Message: &tgbotapi.Message{
		Text:      "/info",
		MessageID: 10,
		From:      &tgbotapi.User{ID: 123, FirstName: "Sam"},
		Chat:      &tgbotapi.Chat{ID: 123, Type: "private"},
	}})

	if len(sender.messages) == 0 {
		t.Fatal("expected info response")
	}
	if sender.messages[0].ParseMode != tgbotapi.ModeHTML {
		t.Fatalf("expected HTML parse mode, got %q", sender.messages[0].ParseMode)
	}
}
