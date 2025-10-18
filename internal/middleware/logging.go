package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// StructuredLogger creates a structured logger middleware
func StructuredLogger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Get status code
		statusCode := c.Writer.Status()

		// Get client IP
		clientIP := c.ClientIP()

		// Get request ID
		requestID := GetRequestID(c)

		// Create log entry
		entry := logger.WithFields(logrus.Fields{
			"request_id":  requestID,
			"method":      c.Request.Method,
			"path":        path,
			"query":       query,
			"status":      statusCode,
			"latency_ms":  latency.Milliseconds(),
			"client_ip":   clientIP,
			"user_agent":  c.Request.UserAgent(),
			"error_count": len(c.Errors),
		})

		// Log with appropriate level
		if len(c.Errors) > 0 {
			// Log errors
			entry.WithField("errors", c.Errors.String()).Error("Request completed with errors")
		} else if statusCode >= 500 {
			entry.Error("Request failed with server error")
		} else if statusCode >= 400 {
			entry.Warn("Request failed with client error")
		} else {
			entry.Info("Request completed successfully")
		}
	}
}

// LoggerWithFormatter creates a custom logger with formatter
func LoggerWithFormatter(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		// Only log if we're not already logging via StructuredLogger
		if c.Writer.Status() >= 500 {
			logger.WithFields(logrus.Fields{
				"request_id": GetRequestID(c),
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"status":     c.Writer.Status(),
				"latency":    time.Since(startTime),
			}).Error("Server error occurred")
		}
	}
}
