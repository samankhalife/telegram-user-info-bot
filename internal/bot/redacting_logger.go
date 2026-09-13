package bot

import (
	"fmt"
	"log/slog"
	"strings"
)

type RedactingLogger struct {
	logger *slog.Logger
	token  string
}

func NewRedactingLogger(logger *slog.Logger, token string) *RedactingLogger {
	return &RedactingLogger{logger: logger, token: token}
}

func (l *RedactingLogger) Println(values ...interface{}) {
	l.logger.Warn("telegram client", "message", l.redact(fmt.Sprintln(values...)))
}

func (l *RedactingLogger) Printf(format string, values ...interface{}) {
	l.logger.Warn("telegram client", "message", l.redact(fmt.Sprintf(format, values...)))
}

func (l *RedactingLogger) redact(value string) string {
	if l.token == "" {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(strings.ReplaceAll(value, l.token, "[REDACTED]"))
}
