package observability

import (
	"log/slog"
	"os"
)

func NewLogger(environment string) *slog.Logger {
	var level slog.Level

	if environment == "production" {
		level = slog.LevelInfo
	} else {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}
