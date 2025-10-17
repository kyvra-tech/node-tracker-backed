package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
)

// GRPCStatusRepository defines the interface for gRPC daily status data access
type GRPCStatusRepository interface {
	// Status operations
	CreateStatus(ctx context.Context, status *models.GRPCDailyStatus) error
	GetStatusByServerAndDate(ctx context.Context, serverID int, date time.Time) (*models.GRPCDailyStatus, error)
	GetRecentStatusesByServer(ctx context.Context, serverID int, days int) ([]models.StatusItem, error)
	HasStatusForDate(ctx context.Context, serverID int, date time.Time) (bool, error)

	// Batch operations
	GetStatusesByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*models.GRPCDailyStatus, error)
	DeleteOldStatuses(ctx context.Context, beforeDate time.Time) error
}

type grpcStatusRepository struct {
	db *sql.DB
}

// NewGRPCStatusRepository creates a new gRPC status repository
func NewGRPCStatusRepository(db *sql.DB) GRPCStatusRepository {
	return &grpcStatusRepository{db: db}
}

func (r *grpcStatusRepository) CreateStatus(ctx context.Context, status *models.GRPCDailyStatus) error {
	query := `
		INSERT INTO grpc_daily_status (server_id, date, color, attempts, success, error_msg, response_time_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (server_id, date) 
		DO UPDATE SET 
			color = EXCLUDED.color,
			attempts = EXCLUDED.attempts,
			success = EXCLUDED.success,
			error_msg = EXCLUDED.error_msg,
			response_time_ms = EXCLUDED.response_time_ms,
			created_at = NOW()
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		status.ServerID, status.Date, status.Color,
		status.Attempts, status.Success, status.ErrorMsg, status.ResponseTimeMs,
	).Scan(&status.ID, &status.CreatedAt)

	if err != nil {
		return fmt.Errorf("create grpc status: %w", err)
	}

	return nil
}

func (r *grpcStatusRepository) GetStatusByServerAndDate(ctx context.Context, serverID int, date time.Time) (*models.GRPCDailyStatus, error) {
	query := `
		SELECT id, server_id, date, color, attempts, success, error_msg, response_time_ms, created_at
		FROM grpc_daily_status
		WHERE server_id = $1 AND date = $2
	`

	status := &models.GRPCDailyStatus{}
	err := r.db.QueryRowContext(ctx, query, serverID, date).Scan(
		&status.ID, &status.ServerID, &status.Date, &status.Color,
		&status.Attempts, &status.Success, &status.ErrorMsg, &status.ResponseTimeMs, &status.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("get grpc status: %w", err)
	}

	return status, nil
}

func (r *grpcStatusRepository) GetRecentStatusesByServer(ctx context.Context, serverID int, days int) ([]models.StatusItem, error) {
	query := `
		SELECT color, date
		FROM grpc_daily_status
		WHERE server_id = $1 AND date >= CURRENT_DATE - INTERVAL '1 day' * $2
		ORDER BY date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID, days)
	if err != nil {
		return nil, fmt.Errorf("query recent grpc statuses: %w", err)
	}
	defer rows.Close()

	var statuses []models.StatusItem
	for rows.Next() {
		var color int
		var date time.Time

		if err := rows.Scan(&color, &date); err != nil {
			return nil, fmt.Errorf("scan grpc status: %w", err)
		}

		statuses = append(statuses, models.StatusItem{
			Color: color,
			Date:  date.Format("2006-01-02"),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return statuses, nil
}

func (r *grpcStatusRepository) HasStatusForDate(ctx context.Context, serverID int, date time.Time) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM grpc_daily_status WHERE server_id = $1 AND date = $2)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, serverID, date).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check grpc status exists: %w", err)
	}

	return exists, nil
}

func (r *grpcStatusRepository) GetStatusesByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*models.GRPCDailyStatus, error) {
	query := `
		SELECT id, server_id, date, color, attempts, success, error_msg, response_time_ms, created_at
		FROM grpc_daily_status
		WHERE date >= $1 AND date <= $2
		ORDER BY date DESC, server_id
	`

	rows, err := r.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query grpc statuses by date range: %w", err)
	}
	defer rows.Close()

	var statuses []*models.GRPCDailyStatus
	for rows.Next() {
		status := &models.GRPCDailyStatus{}
		err := rows.Scan(
			&status.ID, &status.ServerID, &status.Date, &status.Color,
			&status.Attempts, &status.Success, &status.ErrorMsg, &status.ResponseTimeMs, &status.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan grpc status: %w", err)
		}
		statuses = append(statuses, status)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return statuses, nil
}

func (r *grpcStatusRepository) DeleteOldStatuses(ctx context.Context, beforeDate time.Time) error {
	query := `DELETE FROM grpc_daily_status WHERE date < $1`

	result, err := r.db.ExecContext(ctx, query, beforeDate)
	if err != nil {
		return fmt.Errorf("delete old grpc statuses: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		// Log this if needed
	}

	return nil
}
