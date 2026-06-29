package router

import (
	"net/http"

	"github.com/f0bima/go-core/middleware"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Config holds router configuration.
type Config struct {
	ServiceName string
	Middlewares []gin.HandlerFunc
}

// New creates a new Gin router with standard middleware.
func New(cfg *Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Add standard middlewares
	r.Use(middleware.Recovery())
	r.Use(middleware.Logging())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS())

	// Add OpenTelemetry middleware
	r.Use(otelgin.Middleware(cfg.ServiceName))

	// Add custom middlewares
	for _, mw := range cfg.Middlewares {
		r.Use(mw)
	}

	// Add standard endpoints
	r.GET("/health", healthHandler)
	r.GET("/ready", readyHandler)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return r
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": c.GetString("serviceName"),
	})
}

func readyHandler(c *gin.Context) {
	// Add readiness checks here (database, cache, etc.)
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}
