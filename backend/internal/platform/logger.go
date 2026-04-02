package platform

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger builds a slog logger suited to the configured environment.
func NewLogger(env string, level string) *slog.Logger {
	options := &slog.HandlerOptions{Level: parseLevel(level)}
	if strings.EqualFold(env, "development") {
		return slog.New(slog.NewTextHandler(os.Stdout, options))
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, options))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
