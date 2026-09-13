package bot

import (
	"strings"
	"testing"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestFormatUpdateIncludesVisibleDataAndEscapesHTML(t *testing.T) {
	update := tgbotapi.Update{
		UpdateID: 42,
		Message: &tgbotapi.Message{
			MessageID: 7,
			Date:      1_700_000_000,
			Text:      "hello <admin> & friends",
			From: &tgbotapi.User{
				ID:           123,
				FirstName:    "Sam <S>",
				LastName:     "K",
				UserName:     "sam",
				LanguageCode: "fa",
			},
			Chat: &tgbotapi.Chat{ID: 123, Type: "private", FirstName: "Sam"},
			Contact: &tgbotapi.Contact{
				PhoneNumber: "+1234",
				FirstName:   "Sam",
				UserID:      123,
			},
			Location: &tgbotapi.Location{Latitude: 35.6892, Longitude: 51.3890},
		},
	}

	got := strings.Join(FormatUpdate(update), "\n")
	for _, want := range []string{"<b>Update ID:</b> 42", "<b>ID:</b> 123", "@sam", "+1234", "35.689200", "51.389000"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected report to contain %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "<admin>") || strings.Contains(got, "Sam <S>") {
		t.Fatalf("user-controlled HTML was not escaped: %s", got)
	}
	if !strings.Contains(got, "hello &lt;admin&gt; &amp; friends") {
		t.Fatalf("escaped text missing: %s", got)
	}
}

func TestFormatUpdateHandlesMissingOptionalFields(t *testing.T) {
	update := tgbotapi.Update{UpdateID: 1, Message: &tgbotapi.Message{MessageID: 2, Chat: &tgbotapi.Chat{ID: -10, Type: "group"}}}
	got := strings.Join(FormatUpdate(update), "\n")
	if !strings.Contains(got, "<b>Chat</b>") || !strings.Contains(got, "<b>Text:</b> —") {
		t.Fatalf("unexpected report: %s", got)
	}
}

func TestChunkLinesRespectsLimit(t *testing.T) {
	chunks := chunkLines([]string{strings.Repeat("a", 30), strings.Repeat("ب", 30)}, 20)
	if len(chunks) < 3 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for _, chunk := range chunks {
		if len(chunk) > 20 {
			t.Fatalf("chunk exceeds limit: %d", len(chunk))
		}
		if !utf8.ValidString(chunk) {
			t.Fatalf("chunk is not valid UTF-8: %q", chunk)
		}
	}
}

func TestCommand(t *testing.T) {
	for input, want := range map[string]string{
		"/info":        "info",
		"/start@mybot": "start",
		"hello":        "",
		"":             "",
	} {
		if got := command(input); got != want {
			t.Errorf("command(%q) = %q, want %q", input, got, want)
		}
	}
}
