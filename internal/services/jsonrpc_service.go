package services

import (
	"context"
	"fmt"
	"time"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/repositories"
	"github.com/sirupsen/logrus"
)

type JsonRPCService struct {
	grpcMonitor       *GRPCMonitor
	bootstrapMonitor  *BootstrapMonitor
	registrationRepo  repositories.RegistrationRepository
	networkStats      *NetworkStatsService
	logger            *logrus.Logger
}

func NewJsonRPCService(
	grpcMonitor *GRPCMonitor,
	bootstrapMonitor *BootstrapMonitor,
	registrationRepo repositories.RegistrationRepository,
	networkStats *NetworkStatsService,
	logger *logrus.Logger,
) *JsonRPCService {
	return &JsonRPCService{
		grpcMonitor:      grpcMonitor,
		bootstrapMonitor: bootstrapMonitor,
		registrationRepo: registrationRepo,
		networkStats:     networkStats,
		logger:           logger,
	}
}

// ========== NODE METHODS ==========

// GetNodes returns all gRPC nodes with their status
func (s *JsonRPCService) GetNodes(ctx context.Context, params struct{}) ([]*models.JsonRPCNodeResponse, error) {
	servers, err := s.grpcMonitor.GetGRPCServersWithStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	var response []*models.JsonRPCNodeResponse
	for _, server := range servers {
		jsonRPCNode := &models.JsonRPCNodeResponse{
			Name:         server.Name,
			Address:      server.Address,
			Network:      server.Network,
			Email:        server.Email,
			Website:      server.Website,
			Status:       server.Status,
			OverallScore: server.OverallScore,
			Country:      server.Country,
			City:         server.City,
			Latitude:     server.Latitude,
			Longitude:    server.Longitude,
		}
		response = append(response, jsonRPCNode)
	}

	return response, nil
}

// GetBootstrapNodes returns all bootstrap nodes with their status
func (s *JsonRPCService) GetBootstrapNodes(ctx context.Context, params struct{}) ([]*models.BootstrapNodeResponse, error) {
	nodes, err := s.bootstrapMonitor.GetBootstrapNodesWithStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bootstrap nodes: %w", err)
	}
	return nodes, nil
}

// CheckAllNodes triggers a health check for all gRPC nodes
func (s *JsonRPCService) CheckAllNodes(ctx context.Context, params struct{}) (*models.StatusResponse, error) {
	err := s.grpcMonitor.CheckAllServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check all nodes: %w", err)
	}

	return &models.StatusResponse{
		Status:    "all nodes checked",
		Timestamp: time.Now().UTC(),
	}, nil
}

// CheckAllBootstrapNodes triggers a health check for all bootstrap nodes
func (s *JsonRPCService) CheckAllBootstrapNodes(ctx context.Context, params struct{}) (*models.StatusResponse, error) {
	err := s.bootstrapMonitor.CheckAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check all bootstrap nodes: %w", err)
	}

	return &models.StatusResponse{
		Status:    "all bootstrap nodes checked",
		Timestamp: time.Now().UTC(),
	}, nil
}

// GetNodeCount returns the count of active gRPC nodes
func (s *JsonRPCService) GetNodeCount(ctx context.Context, params struct{}) (*models.CountResponse, error) {
	count, err := s.grpcMonitor.GetGRPCServerCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get node count: %w", err)
	}

	return &models.CountResponse{
		Total:     count,
		Timestamp: time.Now().UTC(),
	}, nil
}

// GetBootstrapNodeCount returns the count of active bootstrap nodes
func (s *JsonRPCService) GetBootstrapNodeCount(ctx context.Context, params struct{}) (*models.CountResponse, error) {
	count, err := s.bootstrapMonitor.GetBootstrapNodeCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bootstrap node count: %w", err)
	}

	return &models.CountResponse{
		Total:     count,
		Timestamp: time.Now().UTC(),
	}, nil
}

// SyncNodes triggers a sync of all gRPC nodes from source
func (s *JsonRPCService) SyncNodes(ctx context.Context, params struct{}) (*models.SyncResponse, error) {
	err := s.grpcMonitor.SyncGRPCServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to sync nodes: %w", err)
	}

	count, err := s.grpcMonitor.GetGRPCServerCount(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get server count")
	}

	return &models.SyncResponse{
		Message:      "nodes synced successfully",
		TotalServers: count,
		Timestamp:    time.Now().UTC(),
	}, nil
}

// SyncBootstrapNodes triggers a sync of all bootstrap nodes from source
func (s *JsonRPCService) SyncBootstrapNodes(ctx context.Context, params struct{}) (*models.SyncResponse, error) {
	err := s.bootstrapMonitor.SyncBootstrapNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to sync bootstrap nodes: %w", err)
	}

	count, err := s.bootstrapMonitor.GetBootstrapNodeCount(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get bootstrap node count")
	}

	return &models.SyncResponse{
		Message:      "bootstrap nodes synced successfully",
		TotalServers: count,
		Timestamp:    time.Now().UTC(),
	}, nil
}

// GetHealth returns the health status of the service
func (s *JsonRPCService) GetHealth(ctx context.Context, params struct{}) (*models.HealthResponse, error) {
	return &models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Version:   "1.0.0",
	}, nil
}

// ========== PHASE 2 METHODS ==========

// GetNetworkStats returns network statistics
func (s *JsonRPCService) GetNetworkStats(ctx context.Context, params struct{}) (*models.NetworkStats, error) {
	if s.networkStats == nil {
		return nil, fmt.Errorf("network stats service not available")
	}
	return s.networkStats.GetNetworkStats(ctx)
}

// GetMapNodes returns all nodes formatted for map display
func (s *JsonRPCService) GetMapNodes(ctx context.Context, params struct{}) ([]models.MapNode, error) {
	if s.networkStats == nil {
		return nil, fmt.Errorf("network stats service not available")
	}
	return s.networkStats.GetMapNodes(ctx)
}

// UpdateGeoLocations updates geographic data for all servers
func (s *JsonRPCService) UpdateGeoLocations(ctx context.Context, params struct{}) (*models.StatusResponse, error) {
	if s.networkStats == nil {
		return nil, fmt.Errorf("network stats service not available")
	}

	if err := s.networkStats.UpdateAllGeoLocations(ctx); err != nil {
		return nil, fmt.Errorf("failed to update geo locations: %w", err)
	}

	return &models.StatusResponse{
		Status:    "geo locations updated",
		Timestamp: time.Now().UTC(),
	}, nil
}

func getNodeStatus(score float64) string {
	if score >= 50 {
		return "online"
	}
	return "offline"
}

// RegisterNode handles public node registration
func (s *JsonRPCService) RegisterNode(ctx context.Context, params RegisterNodeParams) (*models.RegistrationResponse, error) {
	if s.registrationRepo == nil {
		return nil, fmt.Errorf("registration not available")
	}

	// Create registration record
	registration := &models.NodeRegistration{
		NodeType:  params.NodeType,
		Name:      params.Name,
		Address:   params.Address,
		Network:   params.Network,
		Email:     params.Email,
		Website:   params.Website,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	// Check for duplicates
	exists, _ := s.registrationRepo.ExistsByAddress(ctx, params.Address)
	if exists {
		return nil, fmt.Errorf("a registration for this address already exists")
	}

	// Save registration
	if err := s.registrationRepo.Create(ctx, registration); err != nil {
		return nil, fmt.Errorf("failed to create registration: %w", err)
	}

	s.logger.WithField("address", params.Address).Info("New node registration submitted")

	return &models.RegistrationResponse{
		ID:      registration.ID,
		Status:  "pending",
		Message: "Registration submitted successfully. We will review your submission shortly.",
	}, nil
}
