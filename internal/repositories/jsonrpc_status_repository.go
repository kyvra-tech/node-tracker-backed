package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
)

// JSONRPCStatusRepository defines the interface for JSON-RPC status data access
type JSONRPCStatusRepository interface {
	GetRecentStatusesByServer(ctx context.Context, serverID int, days int) ([]models.StatusItem, error)
	GetStatusByServerAndDate(ctx context.Context, serverID int, date time.Time) (*models.JSONRPCDailyStatus, error)
	HasStatusForDate(ctx context.Context, serverID int, date time.Time) (bool, error)
	CreateStatus(ctx context.Context, status *models.JSONRPCDailyStatus) error
	UpdateStatus(ctx context.Context, status *models.JSONRPCDailyStatus) error
}

type jsonrpcStatusRepository struct {
	db *sql.DB
}

// NewJSONRPCStatusRepository creates a new JSON-RPC status repository
func NewJSONRPCStatusRepository(db *sql.DB) JSONRPCStatusRepository {
	return &jsonrpcStatusRepository{db: db}
}

func (r *jsonrpcStatusRepository) GetRecentStatusesByServer(ctx context.Context, serverID int, days int) ([]models.StatusItem, error) {
	// Generate all dates for the last N days
	statuses := make([]models.StatusItem, days)
	today := time.Now().Truncate(24 * time.Hour)

	// Initialize with grey (no data)
	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, -(days - 1 - i))
		statuses[i] = models.StatusItem{
			Date:  date.Format("2006-01-02"),
			Color: 0, // grey
		}
	}

	// Query actual status data
	query := `
		SELECT date, color
		FROM jsonrpc_daily_status
		WHERE server_id = $1 AND date >= $2
		ORDER BY date ASC
	`

	startDate := today.AddDate(0, 0, -(days - 1))
	rows, err := r.db.QueryContext(ctx, query, serverID, startDate)
	if err != nil {
		return nil, fmt.Errorf("query statuses: %w", err)
	}
	defer rows.Close()

	// Map query results to statuses
	statusMap := make(map[string]int)
	for rows.Next() {
		var date time.Time
		var color int
		if err := rows.Scan(&date, &color); err != nil {
			return nil, fmt.Errorf("scan status: %w", err)
		}
		statusMap[date.Format("2006-01-02")] = color
	}

	// Update statuses with actual data
	for i := range statuses {
		if color, ok := statusMap[statuses[i].Date]; ok {
			statuses[i].Color = color
		}
	}

	return statuses, nil
}

func (r *jsonrpcStatusRepository) GetStatusByServerAndDate(ctx context.Context, serverID int, date time.Time) (*models.JSONRPCDailyStatus, error) {
	query := `
		SELECT id, server_id, date, color, attempts, success, response_time_ms, error_msg, blockchain_height, created_at
		FROM jsonrpc_daily_status
		WHERE server_id = $1 AND date = $2
	`

	status := &models.JSONRPCDailyStatus{}
	err := r.db.QueryRowContext(ctx, query, serverID, date).Scan(
		&status.ID, &status.ServerID, &status.Date, &status.Color, &status.Attempts,
		&status.Success, &status.ResponseTimeMs, &status.ErrorMsg, &status.BlockchainHeight, &status.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get status by server and date: %w", err)
	}

	return status, nil
}

func (r *jsonrpcStatusRepository) HasStatusForDate(ctx context.Context, serverID int, date time.Time) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM jsonrpc_daily_status WHERE server_id = $1 AND date = $2)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, serverID, date).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check status exists: %w", err)
	}

	return exists, nil
}

func (r *jsonrpcStatusRepository) CreateStatus(ctx context.Context, status *models.JSONRPCDailyStatus) error {
	query := `
		INSERT INTO jsonrpc_daily_status (server_id, date, color, attempts, success, response_time_ms, error_msg, blockchain_height)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (server_id, date) DO UPDATE SET
			color = EXCLUDED.color,
			attempts = EXCLUDED.attempts,
			success = EXCLUDED.success,
			response_time_ms = EXCLUDED.response_time_ms,
			error_msg = EXCLUDED.error_msg,
			blockchain_height = EXCLUDED.blockchain_height
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		status.ServerID, status.Date, status.Color, status.Attempts,
		status.Success, status.ResponseTimeMs, status.ErrorMsg, status.BlockchainHeight,
	).Scan(&status.ID, &status.CreatedAt)

	if err != nil {
		return fmt.Errorf("create status: %w", err)
	}

	return nil
}

func (r *jsonrpcStatusRepository) UpdateStatus(ctx context.Context, status *models.JSONRPCDailyStatus) error {
	query := `
		UPDATE jsonrpc_daily_status SET
			color = $1, attempts = $2, success = $3, response_time_ms = $4, error_msg = $5, blockchain_height = $6
		WHERE id = $7
	`

	_, err := r.db.ExecContext(ctx, query,
		status.Color, status.Attempts, status.Success, status.ResponseTimeMs, status.ErrorMsg, status.BlockchainHeight,
		status.ID,
	)

	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	return nil
}
