package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Timeout creates a timeout middleware
func Timeout(timeout time.Duration, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace request context
		c.Request = c.Request.WithContext(ctx)

		// Channel to signal completion
		finished := make(chan struct{})

		// Run the request in a goroutine
		go func() {
			c.Next()
			finished <- struct{}{}
		}()

		// Wait for completion or timeout
		select {
		case <-finished:
			// Request completed successfully
			return
		case <-ctx.Done():
			// Timeout occurred
			logger.WithFields(logrus.Fields{
				"request_id": GetRequestID(c),
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"timeout":    timeout.String(),
			}).Warn("Request timeout")

			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error":      "Request timeout",
				"request_id": GetRequestID(c),
				"message":    "Request took too long to process",
			})
		}
	}
}
