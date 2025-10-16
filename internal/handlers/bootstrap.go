package handlers

import (
	"net/http"

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
