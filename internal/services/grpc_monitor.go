package services

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/repositories"
	"github.com/pactus-project/pactus/wallet"
)

type GRPCMonitor struct {
	grpcRepo          repositories.GRPCRepository
	grpcStatusRepo    repositories.GRPCStatusRepository
	grpcChecker       *GRPCChecker
	grpcServerService *GRPCServerService
	logger            *logrus.Logger
}

func NewGRPCMonitor(
	grpcRepo repositories.GRPCRepository,
	grpcStatusRepo repositories.GRPCStatusRepository,
	grpcChecker *GRPCChecker,
	logger *logrus.Logger,
	grpcServerService *GRPCServerService,
) *GRPCMonitor {
	return &GRPCMonitor{
		grpcRepo:          grpcRepo,
		grpcStatusRepo:    grpcStatusRepo,
		grpcChecker:       grpcChecker,
		grpcServerService: grpcServerService,
		logger:            logger,
	}
}

// CheckAllServers checks all active gRPC servers
func (gm *GRPCMonitor) CheckAllServers(ctx context.Context) error {
	servers, err := gm.grpcRepo.GetActiveServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active servers: %w", err)
	}

	today := time.Now().Truncate(24 * time.Hour)

	for _, server := range servers {
		if err := gm.checkSingleServer(ctx, server, today); err != nil {
			gm.logger.WithError(err).WithField("server_id", server.ID).Error("Failed to check server")
			continue
		}
	}

	// Update overall scores
	if err := gm.grpcRepo.UpdateAllScores(ctx); err != nil {
		gm.logger.WithError(err).Error("Failed to update overall scores")
	}

	return nil
}

// checkSingleServer checks a single server's health
func (gm *GRPCMonitor) checkSingleServer(ctx context.Context, server *models.GRPCServer, date time.Time) error {
	// Check if already recorded for today
	exists, err := gm.grpcStatusRepo.HasStatusForDate(ctx, server.ID, date)
	if err != nil {
		return err
	}

	if exists {
		gm.logger.WithFields(logrus.Fields{
			"server_id": server.ID,
			"date":      date.Format("2006-01-02"),
		}).Info("Status already recorded for today")
		return nil
	}

	// Check the server
	result := gm.grpcChecker.CheckGRPCServer(ctx, server.Address)

	// Color: 1 = green (success), 0 = grey (failure)
	color := 0
	if result.Success {
		color = 1
	}

	// Save the result
	status := &models.GRPCDailyStatus{
		ServerID:       server.ID,
		Date:           date,
		Color:          color,
		Attempts:       result.Attempts,
		Success:        result.Success,
		ErrorMsg:       result.ErrorMsg,
		ResponseTimeMs: result.ResponseTimeMs,
	}

	return gm.grpcStatusRepo.CreateStatus(ctx, status)
}

// GetGRPCServersWithStatus returns all servers with their 30-day status
func (gm *GRPCMonitor) GetGRPCServersWithStatus(ctx context.Context) ([]*models.GRPCServerResponse, error) {
	servers, err := gm.grpcRepo.GetActiveServers(ctx)
	if err != nil {
		return nil, err
	}

	var response []*models.GRPCServerResponse

	for _, server := range servers {
		statuses, err := gm.grpcStatusRepo.GetRecentStatusesByServer(ctx, server.ID, 30)
		if err != nil {
			gm.logger.WithError(err).WithField("server_id", server.ID).Error("Failed to get statuses")
			continue
		}

		serverResponse := &models.GRPCServerResponse{
			Name:         server.Name,
			Address:      server.Address,
			Network:      server.Network,
			Email:        server.Email,
			Website:      server.Website,
			Status:       statuses,
			OverallScore: server.OverallScore,
		}

		response = append(response, serverResponse)
	}

	return response, nil
}

func (gm *GRPCMonitor) SyncGRPCServers(ctx context.Context) error {
	gm.logger.Info("Starting gRPC server sync from Pactus")

	mainnetServers, err := wallet.GetServerList("mainnet")
	if err != nil {
		return fmt.Errorf("failed to load gRPC servers from GitHub: %w", err)
	}

	testnetServers, err := wallet.GetServerList("testnet")
	if err != nil {
		return fmt.Errorf("failed to load gRPC servers from GitHub: %w", err)
	}

	// Sync mainnet servers
	if err := gm.syncNetworkServers(ctx, "mainnet", mainnetServers); err != nil {
		return fmt.Errorf("failed to sync mainnet: %w", err)
	}

	// Sync testnet servers
	if err := gm.syncNetworkServers(ctx, "testnet", testnetServers); err != nil {
		return fmt.Errorf("failed to sync testnet: %w", err)
	}

	gm.logger.Info("Completed gRPC server sync")
	return nil
}

// syncNetworkServers syncs servers for a specific network
func (gm *GRPCMonitor) syncNetworkServers(ctx context.Context, network string, servers []wallet.ServerInfo) error {
	for _, server := range servers {
		// Check if server exists
		exists, err := gm.grpcRepo.ServerExists(ctx, server.Address)
		if err != nil {
			return err
		}

		if !exists {
			// Add new server
			server := &models.GRPCServer{
				Name:     server.Name,
				Address:  server.Address,
				Network:  network,
				Email:    server.Email,
				Website:  server.Website,
				IsActive: true,
			}

			if err := gm.grpcRepo.CreateServer(ctx, server); err != nil {
				gm.logger.WithError(err).WithField("address", server.Address).Error("Failed to add server")
				continue
			}
			gm.logger.WithField("address", server.Address).Info("Added new server")
		} else {
			// Update existing server
			server := &models.GRPCServer{
				Name:    server.Name,
				Address: server.Address,
				Network: network,
				Email:   server.Email,
				Website: server.Website,
			}

			if err := gm.grpcRepo.UpdateServer(ctx, server); err != nil {
				gm.logger.WithError(err).WithField("address", server.Address).Error("Failed to update server")
				continue
			}
			gm.logger.WithField("address", server.Address).Info("Updated existing server")
		}
	}

	return nil
}

// extractServerName extracts a display name from the address
func (gm *GRPCMonitor) extractServerName(address string) string {
	// You can implement more sophisticated name extraction if needed
	// For now, just use the address as the name
	return address
}

// GetGRPCServerCount returns the count of active servers
func (gm *GRPCMonitor) GetGRPCServerCount(ctx context.Context) (int, error) {
	return gm.grpcRepo.GetServerCount(ctx, true)
}
