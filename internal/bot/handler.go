package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/samankhalife/telegram-user-info-bot/internal/storage"
)

type Sender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
}

type Recorder interface {
	SaveUpdate(ctx context.Context, update tgbotapi.Update) (int64, error)
	SaveResponse(ctx context.Context, response storage.Response) error
	MarkInteraction(ctx context.Context, interactionID int64, status, errorMessage string) error
}

type Handler struct {
	sender   Sender
	recorder Recorder
	logger   *slog.Logger
}

func NewHandler(sender Sender, recorder Recorder, logger *slog.Logger) *Handler {
	return &Handler{sender: sender, recorder: recorder, logger: logger}
}

func (h *Handler) Handle(update tgbotapi.Update) {
	message, _ := messageFromUpdate(update)
	if message == nil || message.Chat == nil {
		return
	}

	ctx := context.Background()
	interactionID, err := h.recorder.SaveUpdate(ctx, update)
	if err != nil {
		h.logger.Error("failed to store incoming update", "error", err, "update_id", update.UpdateID)
		return
	}

	switch command(message.Text) {
	case "start", "help":
		h.sendWelcome(ctx, interactionID, message.Chat.ID)
	case "info":
		h.sendReport(ctx, interactionID, message.Chat.ID, update)
	default:
		h.sendReport(ctx, interactionID, message.Chat.ID, update)
	}
}

func (h *Handler) sendWelcome(ctx context.Context, interactionID, chatID int64) {
	text := "This bot displays information that the Telegram Bot API provides along with your message.\n\n" +
		"To view the current information, send /info or simply send any other message. Your phone number and location are only shared when you press the buttons below and give your consent. Incoming requests and bot responses are stored for 30 days."
	message := tgbotapi.NewMessage(chatID, text)
	message.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonContact("Share phone number"),
			tgbotapi.NewKeyboardButtonLocation("Share location"),
		),
	)
	h.sendAndRecord(ctx, interactionID, message)
}

func (h *Handler) sendReport(ctx context.Context, interactionID, chatID int64, update tgbotapi.Update) {
	for _, text := range FormatUpdate(update) {
		message := tgbotapi.NewMessage(chatID, text)
		message.ParseMode = tgbotapi.ModeHTML
		message.DisableWebPagePreview = true
		if !h.sendAndRecord(ctx, interactionID, message) {
			return
		}
	}
}

func (h *Handler) sendAndRecord(ctx context.Context, interactionID int64, message tgbotapi.MessageConfig) bool {
	sent, sendErr := h.sender.Send(message)
	response := storage.Response{
		InteractionID:  interactionID,
		ResponseText:   message.Text,
		ResponseMarkup: message.ReplyMarkup,
		ParseMode:      message.ParseMode,
		Status:         "sent",
	}
	if sendErr != nil {
		response.Status = "failed"
		response.ErrorMessage = sendErr.Error()
	} else {
		response.TelegramMessageID = sent.MessageID
	}
	if err := h.recorder.SaveResponse(ctx, response); err != nil {
		h.logger.Error("failed to store bot response", "error", err, "interaction_id", interactionID)
	}
	if sendErr != nil {
		_ = h.recorder.MarkInteraction(ctx, interactionID, "failed", sendErr.Error())
		h.logger.Error("failed to send bot response", "error", sendErr, "interaction_id", interactionID)
		return false
	}
	if err := h.recorder.MarkInteraction(ctx, interactionID, "completed", ""); err != nil {
		h.logger.Error("failed to mark interaction completed", "error", err, "interaction_id", interactionID)
	}
	return true
}

func command(text string) string {
	field := strings.Fields(text)
	if len(field) == 0 || !strings.HasPrefix(field[0], "/") {
		return ""
	}
	name := strings.TrimPrefix(field[0], "/")
	if at := strings.IndexByte(name, '@'); at >= 0 {
		name = name[:at]
	}
	return strings.ToLower(name)
}

func (h *Handler) String() string {
	return fmt.Sprintf("Handler{%T}", h.sender)
}
