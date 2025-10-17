package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/services"
)

type GRPCHandler struct {
	monitor *services.GRPCMonitor
	logger  *logrus.Logger
}

func NewGRPCHandler(monitor *services.GRPCMonitor, logger *logrus.Logger) *GRPCHandler {
	return &GRPCHandler{
		monitor: monitor,
		logger:  logger,
	}
}

func (h *GRPCHandler) GetGRPCServers(c *gin.Context) {
	ctx := c.Request.Context()

	servers, err := h.monitor.GetGRPCServersWithStatus(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get gRPC servers")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve gRPC servers",
		})
		return
	}

	c.JSON(http.StatusOK, servers)
}

func (h *GRPCHandler) SyncGRPCServersFromFile(c *gin.Context) {
	ctx := c.Request.Context()

	err := h.monitor.SyncGRPCServersFromFile(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sync gRPC servers")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to sync gRPC servers",
			"details": err.Error(),
		})
		return
	}

	count, err := h.monitor.GetGRPCServerCount(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get server count")
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "gRPC servers synced successfully",
		"total_servers": count,
		"timestamp":     time.Now().UTC(),
	})
}

func (h *GRPCHandler) CheckAllServers(c *gin.Context) {
	ctx := c.Request.Context()

	err := h.monitor.CheckAllServers(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to check all servers")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check all servers",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "all servers checked",
		"timestamp": time.Now().UTC(),
	})
}

func (h *GRPCHandler) GetGRPCServerCount(c *gin.Context) {
	ctx := c.Request.Context()

	count, err := h.monitor.GetGRPCServerCount(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get server count")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve server count",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     count,
		"timestamp": time.Now().UTC(),
	})
}
