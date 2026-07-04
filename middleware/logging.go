package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// Logging logs each request with standard fields, including the request and response body.
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		var reqBodyBytes []byte
		if c.Request.Body != nil {
			reqBodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
		}

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		var requestBody interface{}
		if len(reqBodyBytes) > 0 {
			if json.Valid(reqBodyBytes) {
				if err := json.Unmarshal(reqBodyBytes, &requestBody); err != nil {
					requestBody = string(reqBodyBytes)
				}
			} else {
				requestBody = string(reqBodyBytes)
			}
		}

		var responseBody interface{}
		if json.Valid(blw.body.Bytes()) {
			if err := json.Unmarshal(blw.body.Bytes(), &responseBody); err != nil {
				responseBody = blw.body.String()
			}
		} else {
			responseBody = blw.body.String()
		}

		slog.InfoContext(c.Request.Context(), "HTTP Request",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("remote_addr", c.Request.RemoteAddr),
			slog.Int("status", status),
			slog.String("latency", latency.String()),
			slog.Any("request_body", requestBody),
			slog.Any("response_body", responseBody),
		)
	}
}
