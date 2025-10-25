package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type HealthHandler struct {
	db      *sql.DB
	logger  *logrus.Logger
	version string
}

func NewHealthHandler(db *sql.DB, logger *logrus.Logger, version string) *HealthHandler {
	return &HealthHandler{
		db:      db,
		logger:  logger,
		version: version,
	}
}

// Health performs a basic health check
func (h *HealthHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()

	// Check database connection
	if err := h.db.PingContext(ctx); err != nil {
		h.logger.WithError(err).Error("Database health check failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "unhealthy",
			"timestamp": time.Now().UTC(),
			"version":   h.version,
			"error":     "database unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   h.version,
	})
}
