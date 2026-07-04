package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// TraceHeader extracts the trace ID from the context and adds it to the response header.
func TraceHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := trace.SpanFromContext(c.Request.Context()).
			SpanContext().
			TraceID().
			String()

		if traceID != "" && traceID != "00000000000000000000000000000000" {
			c.Header("X-Trace-ID", traceID)
		}

		c.Next()
	}
}
