package logger

import (
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// InitLogger initializes the global structured logger with JSON formatting and OTLP export.
func InitLogger(serviceName string) {
	jsonHandler := contextHandler{
		Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
		}),
	}

	otelHandler := otelslog.NewHandler(serviceName, otelslog.WithSource(true))

	tee := teeHandler{
		handlers: []slog.Handler{jsonHandler, otelHandler},
	}

	logger := slog.New(tee)
	slog.SetDefault(logger)
}
