package bootstrap

import (
	"context"
	"log/slog"

	"github.com/f0bima/go-core/cache"
	"github.com/f0bima/go-core/config"
	"github.com/f0bima/go-core/database"
	"github.com/f0bima/go-core/logger"
	"github.com/f0bima/go-core/telemetry"
	"go.opentelemetry.io/otel"
)

// Bootstrap initializes the full application foundation and returns a ready-to-use App.
func Bootstrap(extraEnvPaths ...string) *App {
	cfg := config.LoadConfig(extraEnvPaths...)

	logger.InitLogger(cfg.ServiceName)

	ctx := context.Background()

	shutdownTelemetry, err := telemetry.InitTelemetry(ctx, cfg.ServiceName)
	if err != nil {
		slog.Error("Failed to initialize telemetry", "error", err)
	}

	db := database.NewPostgresDB(cfg)

	redisCache := cache.InitCache()

	tracer := otel.Tracer(cfg.ServiceName)
	meter := otel.Meter(cfg.ServiceName)

	return &App{
		Config:            cfg,
		DB:                db,
		Cache:             redisCache,
		Tracer:            tracer,
		Meter:             meter,
		shutdownTelemetry: shutdownTelemetry,
	}
}
