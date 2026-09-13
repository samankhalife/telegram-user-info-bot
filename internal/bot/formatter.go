package bot

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const maxTelegramMessageBytes = 3900

type report struct {
	lines []string
}

func (r *report) section(title string) {
	if len(r.lines) > 0 {
		r.lines = append(r.lines, "")
	}
	r.lines = append(r.lines, "<b>"+html.EscapeString(title)+"</b>")
}

func (r *report) value(label string, value any) {
	text := fmt.Sprint(value)
	if text == "" {
		text = "—"
	}
	r.lines = append(r.lines, "<b>"+html.EscapeString(label)+":</b> "+html.EscapeString(text))
}

func FormatUpdate(update tgbotapi.Update) []string {
	message, updateType := messageFromUpdate(update)
	if message == nil {
		return []string{fmt.Sprintf("<b>Update ID:</b> %d\n<b>Type:</b> unsupported update", update.UpdateID)}
	}

	var r report
	r.section("Update")
	r.value("Update ID", update.UpdateID)
	r.value("Update type", updateType)

	if message.From != nil {
		formatUser(&r, "User", message.From)
	}
	if message.SenderChat != nil {
		formatChat(&r, "Sender chat", message.SenderChat)
	}
	if message.Chat != nil {
		formatChat(&r, "Chat", message.Chat)
	}

	r.section("Message")
	r.value("Message ID", message.MessageID)
	if message.Date != 0 {
		r.value("Date", time.Unix(int64(message.Date), 0).UTC().Format(time.RFC3339))
	}
	if message.EditDate != 0 {
		r.value("Edited at", time.Unix(int64(message.EditDate), 0).UTC().Format(time.RFC3339))
	}
	r.value("Text", message.Text)
	r.value("Caption", message.Caption)
	r.value("Protected content", message.HasProtectedContent)
	r.value("Automatic forward", message.IsAutomaticForward)
	optional(&r, "Media group ID", message.MediaGroupID)
	optional(&r, "Author signature", message.AuthorSignature)
	if message.ReplyToMessage != nil {
		r.value("Reply to message ID", message.ReplyToMessage.MessageID)
	}
	if message.ViaBot != nil {
		r.value("Via bot ID", message.ViaBot.ID)
	}

	formatForward(&r, message)
	formatMedia(&r, message)
	formatSharedData(&r, message)
	formatServiceData(&r, message)

	return chunkLines(r.lines, maxTelegramMessageBytes)
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

func formatUser(r *report, title string, user *tgbotapi.User) {
	r.section(title)
	r.value("ID", user.ID)
	r.value("Is bot", user.IsBot)
	r.value("First name", user.FirstName)
	r.value("Last name", user.LastName)
	r.value("Username", username(user.UserName))
	r.value("Language code", user.LanguageCode)
}

func formatChat(r *report, title string, chat *tgbotapi.Chat) {
	r.section(title)
	r.value("ID", chat.ID)
	r.value("Type", chat.Type)
	r.value("Title", chat.Title)
	r.value("Username", username(chat.UserName))
	r.value("First name", chat.FirstName)
	r.value("Last name", chat.LastName)
	optional(r, "Bio", chat.Bio)
	optional(r, "Description", chat.Description)
	optional(r, "Invite link", chat.InviteLink)
	if chat.LinkedChatID != 0 {
		r.value("Linked chat ID", chat.LinkedChatID)
	}
	r.value("Private forwards", chat.HasPrivateForwards)
	r.value("Protected content", chat.HasProtectedContent)
}

func formatForward(r *report, message *tgbotapi.Message) {
	if message.ForwardFrom == nil && message.ForwardFromChat == nil && message.ForwardSenderName == "" && message.ForwardDate == 0 {
		return
	}
	r.section("Forward")
	if message.ForwardFrom != nil {
		r.value("Original user ID", message.ForwardFrom.ID)
		r.value("Original user", message.ForwardFrom.String())
	}
	if message.ForwardFromChat != nil {
		r.value("Original chat ID", message.ForwardFromChat.ID)
		r.value("Original chat title", message.ForwardFromChat.Title)
	}
	optional(r, "Hidden sender name", message.ForwardSenderName)
	optional(r, "Signature", message.ForwardSignature)
	if message.ForwardFromMessageID != 0 {
		r.value("Original message ID", message.ForwardFromMessageID)
	}
	if message.ForwardDate != 0 {
		r.value("Original date", time.Unix(int64(message.ForwardDate), 0).UTC().Format(time.RFC3339))
	}
}

func formatMedia(r *report, message *tgbotapi.Message) {
	switch {
	case len(message.Photo) > 0:
		photo := message.Photo[len(message.Photo)-1]
		r.section("Photo")
		r.value("File ID", photo.FileID)
		r.value("Unique file ID", photo.FileUniqueID)
		r.value("Size", fmt.Sprintf("%dx%d", photo.Width, photo.Height))
		r.value("File size", photo.FileSize)
	case message.Document != nil:
		r.section("Document")
		r.value("File ID", message.Document.FileID)
		r.value("Unique file ID", message.Document.FileUniqueID)
		r.value("File name", message.Document.FileName)
		r.value("MIME type", message.Document.MimeType)
		r.value("File size", message.Document.FileSize)
	case message.Video != nil:
		r.section("Video")
		r.value("File ID", message.Video.FileID)
		r.value("Unique file ID", message.Video.FileUniqueID)
		r.value("Size", fmt.Sprintf("%dx%d", message.Video.Width, message.Video.Height))
		r.value("Duration", message.Video.Duration)
		r.value("MIME type", message.Video.MimeType)
		r.value("File size", message.Video.FileSize)
	case message.Audio != nil:
		r.section("Audio")
		r.value("File ID", message.Audio.FileID)
		r.value("Unique file ID", message.Audio.FileUniqueID)
		r.value("Duration", message.Audio.Duration)
		r.value("Performer", message.Audio.Performer)
		r.value("Title", message.Audio.Title)
		r.value("File name", message.Audio.FileName)
	case message.Voice != nil:
		r.section("Voice")
		r.value("File ID", message.Voice.FileID)
		r.value("Unique file ID", message.Voice.FileUniqueID)
		r.value("Duration", message.Voice.Duration)
		r.value("MIME type", message.Voice.MimeType)
	case message.Sticker != nil:
		r.section("Sticker")
		r.value("File ID", message.Sticker.FileID)
		r.value("Unique file ID", message.Sticker.FileUniqueID)
		r.value("Emoji", message.Sticker.Emoji)
		r.value("Set name", message.Sticker.SetName)
	case message.Animation != nil:
		r.section("Animation")
		r.value("File ID", message.Animation.FileID)
		r.value("Unique file ID", message.Animation.FileUniqueID)
		r.value("File name", message.Animation.FileName)
		r.value("MIME type", message.Animation.MimeType)
	}
}

func formatSharedData(r *report, message *tgbotapi.Message) {
	if message.Contact != nil {
		r.section("Shared contact")
		r.value("Phone number", message.Contact.PhoneNumber)
		r.value("First name", message.Contact.FirstName)
		r.value("Last name", message.Contact.LastName)
		if message.Contact.UserID != 0 {
			r.value("Telegram user ID", message.Contact.UserID)
		}
		optional(r, "vCard", message.Contact.VCard)
	}
	if message.Location != nil {
		r.section("Shared location")
		r.value("Latitude", strconv.FormatFloat(message.Location.Latitude, 'f', 6, 64))
		r.value("Longitude", strconv.FormatFloat(message.Location.Longitude, 'f', 6, 64))
		if message.Location.HorizontalAccuracy != 0 {
			r.value("Accuracy (m)", message.Location.HorizontalAccuracy)
		}
		if message.Location.LivePeriod != 0 {
			r.value("Live period (s)", message.Location.LivePeriod)
		}
	}
	if message.Venue != nil {
		r.section("Venue")
		r.value("Title", message.Venue.Title)
		r.value("Address", message.Venue.Address)
	}
}

func formatServiceData(r *report, message *tgbotapi.Message) {
	if len(message.NewChatMembers) > 0 {
		r.section("New chat members")
		for _, member := range message.NewChatMembers {
			r.value("Member", fmt.Sprintf("%s (ID: %d)", member.String(), member.ID))
		}
	}
	if message.LeftChatMember != nil {
		r.section("Left chat member")
		r.value("Member", fmt.Sprintf("%s (ID: %d)", message.LeftChatMember.String(), message.LeftChatMember.ID))
	}
	optional(r, "New chat title", message.NewChatTitle)
	if message.MigrateToChatID != 0 {
		r.value("Migrated to chat ID", message.MigrateToChatID)
	}
	if message.MigrateFromChatID != 0 {
		r.value("Migrated from chat ID", message.MigrateFromChatID)
	}
	if message.Dice != nil {
		r.section("Dice")
		r.value("Emoji", message.Dice.Emoji)
		r.value("Value", message.Dice.Value)
	}
	if message.Poll != nil {
		r.section("Poll")
		r.value("ID", message.Poll.ID)
		r.value("Question", message.Poll.Question)
		r.value("Type", message.Poll.Type)
		r.value("Anonymous", message.Poll.IsAnonymous)
		r.value("Total voters", message.Poll.TotalVoterCount)
	}
}

func optional(r *report, label, value string) {
	if value != "" {
		r.value(label, value)
	}
}

func username(value string) string {
	if value == "" {
		return ""
	}
	return "@" + value
}

func chunkLines(lines []string, limit int) []string {
	var chunks []string
	var current strings.Builder

	flush := func() {
		if current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
		}
	}

	for _, line := range lines {
		for len(line) > limit {
			flush()
			cut := validHTMLCut(line, limit)
			chunks = append(chunks, line[:cut])
			line = line[cut:]
		}
		needed := len(line)
		if current.Len() > 0 {
			needed++
		}
		if current.Len()+needed > limit {
			flush()
		}
		if current.Len() > 0 {
			current.WriteByte('\n')
		}
		current.WriteString(line)
	}
	flush()
	return chunks
}

func validHTMLCut(value string, max int) int {
	if len(value) <= max {
		return len(value)
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	if amp := strings.LastIndexByte(value[:cut], '&'); amp >= 0 && !strings.Contains(value[amp:cut], ";") {
		cut = amp
	}
	if cut == 0 {
		cut = max
		for cut > 0 && !utf8.RuneStart(value[cut]) {
			cut--
		}
	}
	return cut
}
