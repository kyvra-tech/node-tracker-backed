package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/services"
)

type BootstrapHandler struct {
	monitor *services.BootstrapMonitor
	logger  *logrus.Logger
}

func NewBootstrapHandler(monitor *services.BootstrapMonitor, logger *logrus.Logger) *BootstrapHandler {
	return &BootstrapHandler{
		monitor: monitor,
		logger:  logger,
	}
}

func (h *BootstrapHandler) GetBootstrapNodes(c *gin.Context) {
	nodes, err := h.monitor.GetBootstrapNodesWithStatus()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get bootstrap nodes")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve bootstrap nodes",
		})
		return
	}

	c.JSON(http.StatusOK, nodes)
}
func (h *BootstrapHandler) SyncBootstrapNodesFromFile(c *gin.Context) {
	err := h.monitor.SyncBootstrapNodesFromFile()
	if err != nil {
		h.logger.WithError(err).Error("Failed to sync bootstrap nodes from file")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to sync bootstrap nodes from file",
			"details": err.Error(),
		})
		return
	}

	// Get updated count
	count, err := h.monitor.GetBootstrapNodeCount()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get bootstrap node count")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get updated count",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Bootstrap nodes synced successfully from file",
		"total_nodes": count,
		"source":      "https://github.com/pactus-project/pactus/blob/main/config/bootstrap.json",
		"timestamp":   time.Now().UTC(),
	})
}

func (h *BootstrapHandler) GetBootstrapNodeCount(c *gin.Context) {
	count, err := h.monitor.GetBootstrapNodeCount()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get bootstrap node count")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve bootstrap node count",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     count,
		"timestamp": time.Now().UTC(),
	})
}

func (h *BootstrapHandler) CheckAllNodes(c *gin.Context) {
	err := h.monitor.CheckAllNodes(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to check all nodes")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check all nodes",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":    "all nodes checked",
		"timestamp": time.Now().UTC(),
	})
}
