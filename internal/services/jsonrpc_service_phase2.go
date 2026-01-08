package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
	"github.com/sirupsen/logrus"
)

// JsonRPCServicePhase2 extends JsonRPCService with Phase 2 functionality
type JsonRPCServicePhase2 struct {
	*JsonRPCService
	jsonrpcMonitor     *JSONRPCMonitorService
	networkStats       *NetworkStatsService
	registrationService *RegistrationService
	logger             *logrus.Logger
}

// NewJsonRPCServicePhase2 creates a new Phase 2 JSON-RPC service
func NewJsonRPCServicePhase2(
	base *JsonRPCService,
	jsonrpcMonitor *JSONRPCMonitorService,
	networkStats *NetworkStatsService,
	registrationService *RegistrationService,
	logger *logrus.Logger,
) *JsonRPCServicePhase2 {
	return &JsonRPCServicePhase2{
		JsonRPCService:      base,
		jsonrpcMonitor:     jsonrpcMonitor,
		networkStats:       networkStats,
		registrationService: registrationService,
		logger:             logger,
	}
}

// ========== JSON-RPC NODES (Phase 2) ==========

// GetJSONRPCNodes returns all JSON-RPC nodes with their status
func (s *JsonRPCServicePhase2) GetJSONRPCNodes(ctx context.Context, params struct{ Network string }) ([]*models.JSONRPCServerResponse, error) {
	servers, err := s.jsonrpcMonitor.GetServersWithStatus(ctx, params.Network)
	if err != nil {
		return nil, fmt.Errorf("failed to get JSON-RPC nodes: %w", err)
	}
	return servers, nil
}

// CheckAllJSONRPCNodes triggers a health check for all JSON-RPC nodes
func (s *JsonRPCServicePhase2) CheckAllJSONRPCNodes(ctx context.Context, params struct{}) (*models.StatusResponse, error) {
	err := s.jsonrpcMonitor.CheckAllServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check all JSON-RPC nodes: %w", err)
	}

	return &models.StatusResponse{
		Status:    "all JSON-RPC nodes checked",
		Timestamp: time.Now().UTC(),
	}, nil
}

// GetJSONRPCNodeCount returns the count of active JSON-RPC nodes
func (s *JsonRPCServicePhase2) GetJSONRPCNodeCount(ctx context.Context, params struct{}) (*models.CountResponse, error) {
	count, err := s.jsonrpcMonitor.GetServerCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get JSON-RPC node count: %w", err)
	}

	return &models.CountResponse{
		Total:     count,
		Timestamp: time.Now().UTC(),
	}, nil
}

// UpdateGeoLocations updates geographic data for all servers
func (s *JsonRPCServicePhase2) UpdateGeoLocations(ctx context.Context, params struct{}) (*models.StatusResponse, error) {
	err := s.jsonrpcMonitor.UpdateServerGeoLocations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update geo locations: %w", err)
	}

	return &models.StatusResponse{
		Status:    "geo locations updated",
		Timestamp: time.Now().UTC(),
	}, nil
}

// ========== NETWORK STATS (Phase 2) ==========

// GetNetworkStats returns network statistics
func (s *JsonRPCServicePhase2) GetNetworkStats(ctx context.Context, params struct{}) (*models.NetworkStats, error) {
	stats, err := s.networkStats.GetNetworkStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get network stats: %w", err)
	}
	return stats, nil
}

// GetMapNodes returns all nodes formatted for map display
func (s *JsonRPCServicePhase2) GetMapNodes(ctx context.Context, params struct{}) ([]models.MapNode, error) {
	nodes, err := s.networkStats.GetMapNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get map nodes: %w", err)
	}
	return nodes, nil
}

// GetSnapshots returns recent network snapshots
func (s *JsonRPCServicePhase2) GetSnapshots(ctx context.Context, params struct{ Limit int }) ([]*models.NetworkSnapshot, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 10
	}
	
	snapshots, err := s.networkStats.GetSnapshots(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}
	return snapshots, nil
}

// ========== REGISTRATION (Phase 2) ==========

// RegisterNodeParams contains registration request parameters
type RegisterNodeParams struct {
	NodeType string `json:"nodeType"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	Network  string `json:"network"`
	Email    string `json:"email"`
	Website  string `json:"website"`
}

// RegisterNode handles public node registration
func (s *JsonRPCServicePhase2) RegisterNode(ctx context.Context, params RegisterNodeParams) (*models.RegistrationResponse, error) {
	req := &models.RegistrationRequest{
		NodeType: params.NodeType,
		Name:     params.Name,
		Address:  params.Address,
		Network:  params.Network,
		Email:    params.Email,
		Website:  params.Website,
	}

	response, err := s.registrationService.SubmitRegistration(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to register node: %w", err)
	}

	return response, nil
}

// GetRegistrationStatus returns the status of a registration
func (s *JsonRPCServicePhase2) GetRegistrationStatus(ctx context.Context, params struct{ ID int }) (*models.NodeRegistration, error) {
	registration, err := s.registrationService.GetRegistrationByID(ctx, params.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get registration: %w", err)
	}
	if registration == nil {
		return nil, fmt.Errorf("registration not found: %d", params.ID)
	}
	return registration, nil
}

// GetPendingRegistrations returns all pending registrations (admin only)
func (s *JsonRPCServicePhase2) GetPendingRegistrations(ctx context.Context, params struct{}) ([]*models.NodeRegistration, error) {
	registrations, err := s.registrationService.GetPendingRegistrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending registrations: %w", err)
	}
	return registrations, nil
}

// ApproveRegistrationParams contains approval parameters
type ApproveRegistrationParams struct {
	ID         int    `json:"id"`
	ReviewedBy string `json:"reviewedBy"`
}

// ApproveRegistration approves a pending registration (admin only)
func (s *JsonRPCServicePhase2) ApproveRegistration(ctx context.Context, params ApproveRegistrationParams) (*models.StatusResponse, error) {
	err := s.registrationService.ApproveRegistration(ctx, params.ID, params.ReviewedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to approve registration: %w", err)
	}

	return &models.StatusResponse{
		Status:    "registration approved",
		Timestamp: time.Now().UTC(),
	}, nil
}

// RejectRegistrationParams contains rejection parameters
type RejectRegistrationParams struct {
	ID         int    `json:"id"`
	Reason     string `json:"reason"`
	ReviewedBy string `json:"reviewedBy"`
}

// RejectRegistration rejects a pending registration (admin only)
func (s *JsonRPCServicePhase2) RejectRegistration(ctx context.Context, params RejectRegistrationParams) (*models.StatusResponse, error) {
	err := s.registrationService.RejectRegistration(ctx, params.ID, params.Reason, params.ReviewedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to reject registration: %w", err)
	}

	return &models.StatusResponse{
		Status:    "registration rejected",
		Timestamp: time.Now().UTC(),
	}, nil
}

// ParseParams parses JSON-RPC params into the target struct
func ParseParams[T any](rawParams json.RawMessage) (T, error) {
	var params T
	if len(rawParams) > 0 && string(rawParams) != "{}" {
		if err := json.Unmarshal(rawParams, &params); err != nil {
			return params, fmt.Errorf("failed to parse params: %w", err)
		}
	}
	return params, nil
}
