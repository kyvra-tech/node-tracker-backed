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

// JsonRPCHandlerPhase2 extends JsonRPCHandler with Phase 2 methods
type JsonRPCHandlerPhase2 struct {
	*JsonRPCHandler
	phase2Service *services.JsonRPCServicePhase2
	logger        *logrus.Logger
}

// NewJsonRPCHandlerPhase2 creates a new Phase 2 JSON-RPC handler
func NewJsonRPCHandlerPhase2(
	base *JsonRPCHandler,
	phase2Service *services.JsonRPCServicePhase2,
	logger *logrus.Logger,
) *JsonRPCHandlerPhase2 {
	return &JsonRPCHandlerPhase2{
		JsonRPCHandler: base,
		phase2Service:  phase2Service,
		logger:         logger,
	}
}

// HandleRequest processes JSON-RPC requests (overrides base handler)
func (h *JsonRPCHandlerPhase2) HandleRequest(c *gin.Context) {
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

	// Single request handling
	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.WithError(err).Error("Failed to parse JSON-RPC request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse JSON-RPC request"})
		return
	}

	response := h.processRequestPhase2(c.Request.Context(), req)
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, response)
}

// processRequestPhase2 handles both Phase 1 and Phase 2 methods
func (h *JsonRPCHandlerPhase2) processRequestPhase2(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	var result interface{}
	var methodErr error

	// Try Phase 2 methods first
	switch req.Method {
	// Phase 2: JSON-RPC Nodes
	case "getJSONRPCNodes":
		var params struct{ Network string }
		json.Unmarshal(req.Params, &params)
		result, methodErr = h.phase2Service.GetJSONRPCNodes(ctx, params)
	case "checkAllJSONRPCNodes":
		result, methodErr = h.phase2Service.CheckAllJSONRPCNodes(ctx, struct{}{})
	case "getJSONRPCNodeCount":
		result, methodErr = h.phase2Service.GetJSONRPCNodeCount(ctx, struct{}{})
	case "updateGeoLocations":
		result, methodErr = h.phase2Service.UpdateGeoLocations(ctx, struct{}{})

	// Phase 2: Network Stats
	case "getNetworkStats":
		result, methodErr = h.phase2Service.GetNetworkStats(ctx, struct{}{})
	case "getMapNodes":
		result, methodErr = h.phase2Service.GetMapNodes(ctx, struct{}{})
	case "getSnapshots":
		var params struct{ Limit int }
		json.Unmarshal(req.Params, &params)
		result, methodErr = h.phase2Service.GetSnapshots(ctx, params)

	// Phase 2: Registration
	case "registerNode":
		var params services.RegisterNodeParams
		json.Unmarshal(req.Params, &params)
		result, methodErr = h.phase2Service.RegisterNode(ctx, params)
	case "getRegistrationStatus":
		var params struct{ ID int }
		json.Unmarshal(req.Params, &params)
		result, methodErr = h.phase2Service.GetRegistrationStatus(ctx, params)
	case "getPendingRegistrations":
		result, methodErr = h.phase2Service.GetPendingRegistrations(ctx, struct{}{})
	case "approveRegistration":
		var params services.ApproveRegistrationParams
		json.Unmarshal(req.Params, &params)
		result, methodErr = h.phase2Service.ApproveRegistration(ctx, params)
	case "rejectRegistration":
		var params services.RejectRegistrationParams
		json.Unmarshal(req.Params, &params)
		result, methodErr = h.phase2Service.RejectRegistration(ctx, params)

	// Phase 1 methods - delegate to base handler
	case "getNodes", "getBootstrapNodes", "checkAllNodes", "checkAllBootstrapNodes",
		"getNodeCount", "getBootstrapNodeCount", "syncNodes", "syncBootstrapNodes", "getHealth":
		return h.JsonRPCHandler.processRequest(ctx, req)

	default:
		h.logger.WithField("method", req.Method).Error("Method not found")
		response.Error = &JSONRPCError{
			Code:    -32601,
			Message: "Method not found",
		}
		return response
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

// handleBatchRequest handles batch JSON-RPC requests
func (h *JsonRPCHandlerPhase2) handleBatchRequest(c *gin.Context, body []byte) {
	var requests []JSONRPCRequest
	if err := json.Unmarshal(body, &requests); err != nil {
		h.logger.WithError(err).Error("Failed to parse batch JSON-RPC request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse batch request"})
		return
	}

	responses := make([]JSONRPCResponse, len(requests))
	for i, req := range requests {
		responses[i] = h.processRequestPhase2(c.Request.Context(), req)
	}

	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, responses)
}
