// Package logging provides a thin wrapper around log/slog with JSON output.
package logging

import (
	"log/slog"
	"os"
)

// New builds a structured logger.
func New(env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "development" || env == "" {
		level = slog.LevelDebug
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(h)
}
