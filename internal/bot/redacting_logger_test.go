package bot

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestRedactingLoggerRemovesBotToken(t *testing.T) {
	const token = "123456789:very-secret-token"
	var output bytes.Buffer
	logger := NewRedactingLogger(slog.New(slog.NewTextHandler(&output, nil)), token)

	logger.Printf("Post https://api.telegram.org/bot%s/getUpdates: unexpected EOF", token)

	if strings.Contains(output.String(), token) {
		t.Fatalf("token leaked into log: %s", output.String())
	}
	if !strings.Contains(output.String(), "[REDACTED]") {
		t.Fatalf("redaction marker missing: %s", output.String())
	}
}
