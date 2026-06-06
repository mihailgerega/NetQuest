package logger

import (
	"log/slog"
	"os"
)

func New(serviceName, env string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return slog.New(handler).With(
		slog.String("service", serviceName),
		slog.String("env", env),
	)
}
