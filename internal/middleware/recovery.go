package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Recovery creates a panic recovery middleware with custom logging
func Recovery(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stack := string(debug.Stack())

				// Log the panic
				logger.WithFields(logrus.Fields{
					"request_id": GetRequestID(c),
					"method":     c.Request.Method,
					"path":       c.Request.URL.Path,
					"client_ip":  c.ClientIP(),
					"panic":      err,
					"stack":      stack,
				}).Error("Panic recovered")

				// Return error response
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":      "Internal server error",
					"request_id": GetRequestID(c),
					"message":    "An unexpected error occurred. Please try again later.",
				})
			}
		}()

		c.Next()
	}
}

// RecoveryWithWriter creates a panic recovery middleware with custom writer
func RecoveryWithWriter(logger *logrus.Logger, notifyFunc func(c *gin.Context, err interface{})) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stack := string(debug.Stack())

				// Log the panic
				logger.WithFields(logrus.Fields{
					"request_id": GetRequestID(c),
					"method":     c.Request.Method,
					"path":       c.Request.URL.Path,
					"client_ip":  c.ClientIP(),
					"panic":      err,
					"stack":      stack,
				}).Error("Panic recovered")

				// Notify external system if provided (e.g., Sentry, Slack)
				if notifyFunc != nil {
					notifyFunc(c, err)
				}

				// Return error response
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":      "Internal server error",
					"request_id": GetRequestID(c),
					"message":    fmt.Sprintf("Panic: %v", err),
				})
			}
		}()

		c.Next()
	}
}
