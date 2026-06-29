package logger

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/trace"
)

type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		r.AddAttrs(slog.String("traceId", span.SpanContext().TraceID().String()))
	}
	if span.SpanContext().HasSpanID() {
		r.AddAttrs(slog.String("spanId", span.SpanContext().SpanID().String()))
	}
	return h.Handler.Handle(ctx, r)
}

type teeHandler struct {
	handlers []slog.Handler
}

func (t teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range t.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (t teeHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, h := range t.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (t teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var handlers []slog.Handler
	for _, h := range t.handlers {
		handlers = append(handlers, h.WithAttrs(attrs))
	}
	return teeHandler{handlers: handlers}
}

func (t teeHandler) WithGroup(name string) slog.Handler {
	var handlers []slog.Handler
	for _, h := range t.handlers {
		handlers = append(handlers, h.WithGroup(name))
	}
	return teeHandler{handlers: handlers}
}

// InitLogger initializes the global structured logger with JSON formatting and OTLP export.
func InitLogger(serviceName string) {
	jsonHandler := contextHandler{
		Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	}

	otelHandler := otelslog.NewHandler(serviceName)

	tee := teeHandler{
		handlers: []slog.Handler{jsonHandler, otelHandler},
	}

	logger := slog.New(tee)
	slog.SetDefault(logger)
}
