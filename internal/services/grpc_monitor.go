package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
)

type GRPCMonitor struct {
	db                *sql.DB
	logger            *logrus.Logger
	grpcChecker       *GRPCChecker
	grpcServerService *GRPCServerService
}

func NewGRPCMonitor(
	db *sql.DB,
	grpcChecker *GRPCChecker,
	logger *logrus.Logger,
	grpcServerService *GRPCServerService,
) *GRPCMonitor {
	return &GRPCMonitor{
		db:                db,
		logger:            logger,
		grpcChecker:       grpcChecker,
		grpcServerService: grpcServerService,
	}
}

// CheckAllServers checks all active gRPC servers
func (gm *GRPCMonitor) CheckAllServers(ctx context.Context) error {
	servers, err := gm.getActiveServers()
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
	if err := gm.updateOverallScores(); err != nil {
		gm.logger.WithError(err).Error("Failed to update overall scores")
	}

	return nil
}

func (gm *GRPCMonitor) checkSingleServer(ctx context.Context, server *models.GRPCServer, date time.Time) error {
	// Check if already recorded for today
	exists, err := gm.hasStatusForDate(server.ID, date)
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

	return gm.saveDailyStatus(status)
}

func (gm *GRPCMonitor) getActiveServers() ([]*models.GRPCServer, error) {
	query := `
        SELECT id, name, address, network, overall_score, is_active, created_at, updated_at
        FROM grpc_servers 
        WHERE is_active = true
        ORDER BY network, id
    `

	rows, err := gm.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*models.GRPCServer
	for rows.Next() {
		server := &models.GRPCServer{}
		err := rows.Scan(
			&server.ID, &server.Name, &server.Address, &server.Network,
			&server.OverallScore, &server.IsActive, &server.CreatedAt, &server.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	return servers, rows.Err()
}

func (gm *GRPCMonitor) hasStatusForDate(serverID int, date time.Time) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM grpc_daily_status WHERE server_id = $1 AND date = $2)`

	var exists bool
	err := gm.db.QueryRow(query, serverID, date).Scan(&exists)
	return exists, err
}

func (gm *GRPCMonitor) saveDailyStatus(status *models.GRPCDailyStatus) error {
	query := `
        INSERT INTO grpc_daily_status (server_id, date, color, attempts, success, error_msg, response_time_ms)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (server_id, date) 
        DO UPDATE SET 
            color = EXCLUDED.color,
            attempts = EXCLUDED.attempts,
            success = EXCLUDED.success,
            error_msg = EXCLUDED.error_msg,
            response_time_ms = EXCLUDED.response_time_ms,
            created_at = NOW()
    `

	_, err := gm.db.Exec(query,
		status.ServerID, status.Date, status.Color,
		status.Attempts, status.Success, status.ErrorMsg, status.ResponseTimeMs,
	)

	return err
}

func (gm *GRPCMonitor) updateOverallScores() error {
	query := `
        UPDATE grpc_servers 
        SET overall_score = (
            SELECT COALESCE(
                ROUND(
                    (COUNT(CASE WHEN success = true THEN 1 END) * 100.0 / COUNT(*))::numeric, 2
                ), 0
            )
            FROM grpc_daily_status 
            WHERE server_id = grpc_servers.id 
            AND date >= CURRENT_DATE - INTERVAL '30 days'
        ),
        updated_at = NOW()
        WHERE is_active = true
    `

	_, err := gm.db.Exec(query)
	return err
}

// GetGRPCServersWithStatus returns all servers with their 30-day status
func (gm *GRPCMonitor) GetGRPCServersWithStatus() ([]*models.GRPCServerResponse, error) {
	servers, err := gm.getActiveServers()
	if err != nil {
		return nil, err
	}

	var response []*models.GRPCServerResponse

	for _, server := range servers {
		statuses, err := gm.getRecentStatuses(server.ID, 30)
		if err != nil {
			gm.logger.WithError(err).WithField("server_id", server.ID).Error("Failed to get statuses")
			continue
		}

		serverResponse := &models.GRPCServerResponse{
			Name:         server.Name,
			Address:      server.Address,
			Network:      server.Network,
			Status:       statuses,
			OverallScore: server.OverallScore,
		}

		response = append(response, serverResponse)
	}

	return response, nil
}

func (gm *GRPCMonitor) getRecentStatuses(serverID int, days int) ([]models.StatusItem, error) {
	query := `
        SELECT color, date
        FROM grpc_daily_status
        WHERE server_id = $1 AND date >= CURRENT_DATE - INTERVAL '%d days'
        ORDER BY date DESC
    `

	rows, err := gm.db.Query(fmt.Sprintf(query, days), serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []models.StatusItem
	for rows.Next() {
		var color int
		var date time.Time

		if err := rows.Scan(&color, &date); err != nil {
			return nil, err
		}

		status := models.StatusItem{
			Color: color,
			Date:  date.Format("2006-01-02"),
		}
		statuses = append(statuses, status)
	}

	return statuses, rows.Err()
}

// SyncGRPCServersFromFile syncs servers from servers.json
func (gm *GRPCMonitor) SyncGRPCServersFromFile() error {
	gm.logger.Info("Starting gRPC server sync from file")

	config, err := gm.grpcServerService.LoadGRPCServers()
	if err != nil {
		return fmt.Errorf("failed to load servers: %w", err)
	}

	// Sync mainnet servers
	if err := gm.syncNetworkServers("mainnet", config.Mainnet); err != nil {
		return err
	}

	// Sync testnet servers
	if err := gm.syncNetworkServers("testnet", config.Testnet); err != nil {
		return err
	}

	gm.logger.Info("Completed gRPC server sync")
	return nil
}

func (gm *GRPCMonitor) syncNetworkServers(network string, addresses []string) error {
	for _, address := range addresses {
		// Check if server exists
		exists, err := gm.serverExists(address)
		if err != nil {
			return err
		}

		if !exists {
			// Add new server
			if err := gm.addServer(network, address); err != nil {
				gm.logger.WithError(err).WithField("address", address).Error("Failed to add server")
				continue
			}
		}
	}

	return nil
}

func (gm *GRPCMonitor) serverExists(address string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM grpc_servers WHERE address = $1)`

	var exists bool
	err := gm.db.QueryRow(query, address).Scan(&exists)
	return exists, err
}

func (gm *GRPCMonitor) addServer(network, address string) error {
	query := `
        INSERT INTO grpc_servers (name, address, network, is_active, created_at, updated_at)
        VALUES ($1, $2, $3, true, NOW(), NOW())
        ON CONFLICT (address) DO NOTHING
    `

	// Extract name from address (e.g., "bootstrap1.pactus.org:50051" -> "bootstrap1.pactus.org")
	name := address
	if len(address) > 0 {
		// You can add more sophisticated name extraction if needed
		name = address
	}

	_, err := gm.db.Exec(query, name, address, network)
	return err
}

// GetGRPCServerCount returns the count of active servers
func (gm *GRPCMonitor) GetGRPCServerCount() (int, error) {
	query := `SELECT COUNT(*) FROM grpc_servers WHERE is_active = true`

	var count int
	err := gm.db.QueryRow(query).Scan(&count)
	return count, err
}
