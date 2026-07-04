package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/f0bima/go-core/cache"
	"github.com/f0bima/go-core/config"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// App holds all shared infrastructure components used by every service.
type App struct {
	Config            *config.Config
	DB                *gorm.DB
	Cache             cache.Provider
	Router            *gin.Engine
	Tracer            trace.Tracer
	Meter             metric.Meter
	server            *http.Server
	shutdownTelemetry func(ctx context.Context) error
}



// Shutdown performs telemetry flush and cache cleanup.
func (a *App) Shutdown(ctx context.Context) error {
	// Close cache connection
	if a.Cache != nil {
		if err := a.Cache.Close(); err != nil {
			slog.Error("Failed to close cache", "error", err)
		}
	}

	if a.shutdownTelemetry != nil {
		return a.shutdownTelemetry(ctx)
	}
	return nil
}

// Run starts the HTTP server and blocks until a shutdown signal is received.
func (a *App) Run() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("Starting server", "port", a.Config.Port, "service", a.Config.ServiceName)

	a.server = &http.Server{
		Addr:    ":" + a.Config.Port,
		Handler: a.Router,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("Shutting down gracefully...")
	case err := <-errCh:
		slog.Error("Server failed to start", "error", err)
	}

	a.shutdown()
}

// shutdown performs cleanup: HTTP server graceful stop, database close, and telemetry flush.
func (a *App) shutdown() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if a.server != nil {
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Failed to shutdown HTTP server", "error", err)
		}
	}

	if sqlDB, err := a.DB.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			slog.Error("Failed to close database", "error", err)
		}
	}

	if err := a.Shutdown(shutdownCtx); err != nil {
		slog.Error("Failed to shutdown telemetry", "error", err)
	}
}
