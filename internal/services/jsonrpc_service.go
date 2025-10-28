package services

import (
	"context"
	"fmt"
	"time"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
	"github.com/sirupsen/logrus"
)

type JsonRPCService struct {
	grpcMonitor      *GRPCMonitor
	bootstrapMonitor *BootstrapMonitor
	logger           *logrus.Logger
}

func NewJsonRPCService(grpcMonitor *GRPCMonitor, bootstrapMonitor *BootstrapMonitor, logger *logrus.Logger) *JsonRPCService {
	return &JsonRPCService{
		grpcMonitor:      grpcMonitor,
		bootstrapMonitor: bootstrapMonitor,
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
