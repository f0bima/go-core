package middleware

import (
	"log/slog"
	"runtime/debug"

	"github.com/f0bima/go-core/response"
	"github.com/gin-gonic/gin"
)

// Recovery recovers from panics, logs the error using slog, and returns a 500 response.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Capture stack trace
				stackTrace := string(debug.Stack())

				// Log the panic with slog
				slog.ErrorContext(c.Request.Context(), "Panic recovered",
					slog.Any("error", err),
					slog.String("stack_trace", stackTrace),
				)

				// Format the response
				if e, ok := err.(string); ok {
					response.InternalServerError(c, "INTERNAL_SERVER_ERROR", e)
				} else if e, ok := err.(error); ok {
					response.InternalServerError(c, "INTERNAL_SERVER_ERROR", e.Error())
				} else {
					response.InternalServerError(c, "INTERNAL_SERVER_ERROR", "Unknown panic occurred")
				}
				
				c.Abort()
			}
		}()
		c.Next()
	}
}
