package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RateLimiter represents a rate limiter
type RateLimiter struct {
	clients map[string]*ClientRateLimit
	mu      sync.RWMutex
	logger  *logrus.Logger
	limit   int           // Max requests
	window  time.Duration // Time window
}

// ClientRateLimit tracks rate limit info for a client
type ClientRateLimit struct {
	count     int
	lastReset time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// limit: maximum requests per window
// window: time window for rate limiting
func NewRateLimiter(limit int, window time.Duration, logger *logrus.Logger) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*ClientRateLimit),
		logger:  logger,
		limit:   limit,
		window:  window,
	}

	// Cleanup goroutine to remove old entries
	go rl.cleanup()

	return rl
}

// Middleware returns a gin middleware handler
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !rl.allowRequest(clientIP) {
			rl.logger.WithFields(logrus.Fields{
				"client_ip":  clientIP,
				"request_id": GetRequestID(c),
				"path":       c.Request.URL.Path,
			}).Warn("Rate limit exceeded")

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     "Too many requests. Please try again later.",
				"retry_after": rl.window.Seconds(),
			})
			return
		}

		c.Next()
	}
}

// allowRequest checks if a request should be allowed
func (rl *RateLimiter) allowRequest(clientIP string) bool {
	rl.mu.Lock()
	client, exists := rl.clients[clientIP]
	if !exists {
		client = &ClientRateLimit{
			count:     0,
			lastReset: time.Now(),
		}
		rl.clients[clientIP] = client
	}
	rl.mu.Unlock()

	client.mu.Lock()
	defer client.mu.Unlock()

	// Check if we need to reset the window
	if time.Since(client.lastReset) > rl.window {
		client.count = 0
		client.lastReset = time.Now()
	}

	// Check if limit is exceeded
	if client.count >= rl.limit {
		return false
	}

	client.count++
	return true
}

// cleanup periodically removes old entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window * 2)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, client := range rl.clients {
			client.mu.Lock()
			if now.Sub(client.lastReset) > rl.window*2 {
				delete(rl.clients, ip)
			}
			client.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"total_clients": len(rl.clients),
		"limit":         rl.limit,
		"window":        rl.window.String(),
	}
}
