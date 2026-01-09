package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/services"
)

// JSONRPC request structure
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

// JSONRPC response structure
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id"`
}

// JSONRPC error structure
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type JsonRPCHandler struct {
	service *services.JsonRPCService
	logger  *logrus.Logger
}

func NewJsonRPCHandler(service *services.JsonRPCService, logger *logrus.Logger) *JsonRPCHandler {
	return &JsonRPCHandler{
		service: service,
		logger:  logger,
	}
}

func (h *JsonRPCHandler) HandleRequest(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Check if it's a batch request (starts with '[')
	if len(body) > 0 && body[0] == '[' {
		h.handleBatchRequest(c, body)
		return
	}

	// Single request handling (existing code)
	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.WithError(err).Error("Failed to parse JSON-RPC request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse JSON-RPC request"})
		return
	}

	response := h.processRequest(c.Request.Context(), req)
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, response)
}

// Process a single request
func (h *JsonRPCHandler) processRequest(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	var result interface{}
	var methodErr error

	switch req.Method {
	case "getNodes":
		result, methodErr = h.service.GetNodes(ctx, struct{}{})
	case "getBootstrapNodes":
		result, methodErr = h.service.GetBootstrapNodes(ctx, struct{}{})
	case "checkAllNodes":
		result, methodErr = h.service.CheckAllNodes(ctx, struct{}{})
	case "checkAllBootstrapNodes":
		result, methodErr = h.service.CheckAllBootstrapNodes(ctx, struct{}{})
	case "getNodeCount":
		result, methodErr = h.service.GetNodeCount(ctx, struct{}{})
	case "getBootstrapNodeCount":
		result, methodErr = h.service.GetBootstrapNodeCount(ctx, struct{}{})
	case "syncNodes":
		result, methodErr = h.service.SyncNodes(ctx, struct{}{})
	case "syncBootstrapNodes":
		result, methodErr = h.service.SyncBootstrapNodes(ctx, struct{}{})
	case "getHealth":
		result, methodErr = h.service.GetHealth(ctx, struct{}{})
	// Phase 2 methods
	case "getNetworkStats":
		result, methodErr = h.service.GetNetworkStats(ctx, struct{}{})
	case "getMapNodes":
		result, methodErr = h.service.GetMapNodes(ctx, struct{}{})
	case "updateGeoLocations":
		result, methodErr = h.service.UpdateGeoLocations(ctx, struct{}{})
	case "registerNode":
		var params services.RegisterNodeParams
		json.Unmarshal(req.Params, &params)
		result, methodErr = h.service.RegisterNode(ctx, params)
	default:
		h.logger.WithField("method", req.Method).Error("Method not found")
		response.Error = &JSONRPCError{
			Code:    -32601,
			Message: "Method not found",
		}
	}

	if methodErr != nil {
		h.logger.WithError(methodErr).Error("Failed to process JSON-RPC request")
		response.Error = &JSONRPCError{
			Code:    -32000,
			Message: methodErr.Error(),
		}
	} else if response.Error == nil {
		response.Result = result
	}

	return response
}

// Handle batch requests
func (h *JsonRPCHandler) handleBatchRequest(c *gin.Context, body []byte) {
	var requests []JSONRPCRequest
	if err := json.Unmarshal(body, &requests); err != nil {
		h.logger.WithError(err).Error("Failed to parse batch JSON-RPC request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse batch request"})
		return
	}

	responses := make([]JSONRPCResponse, len(requests))
	for i, req := range requests {
		responses[i] = h.processRequest(c.Request.Context(), req)
	}

	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, responses)
}
