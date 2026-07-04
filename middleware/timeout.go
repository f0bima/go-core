package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout adds a timeout to requests.
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Timeout-Seconds", timeout.String())
		c.Next()
	}
}
