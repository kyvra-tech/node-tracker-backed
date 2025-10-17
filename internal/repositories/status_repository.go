package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
)

// StatusRepository defines the interface for daily status data access
type StatusRepository interface {
	// Status operations
	CreateStatus(ctx context.Context, status *models.DailyStatus) error
	GetStatusByNodeAndDate(ctx context.Context, nodeID int, date time.Time) (*models.DailyStatus, error)
	GetRecentStatusesByNode(ctx context.Context, nodeID int, days int) ([]models.StatusItem, error)
	HasStatusForDate(ctx context.Context, nodeID int, date time.Time) (bool, error)

	// Batch operations
	GetStatusesByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*models.DailyStatus, error)
	DeleteOldStatuses(ctx context.Context, beforeDate time.Time) error
}

type statusRepository struct {
	db *sql.DB
}

// NewStatusRepository creates a new status repository
func NewStatusRepository(db *sql.DB) StatusRepository {
	return &statusRepository{db: db}
}

func (r *statusRepository) CreateStatus(ctx context.Context, status *models.DailyStatus) error {
	query := `
		INSERT INTO daily_status (node_id, date, color, attempts, success, error_msg, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (node_id, date) 
		DO UPDATE SET 
			color = EXCLUDED.color,
			attempts = EXCLUDED.attempts,
			success = EXCLUDED.success,
			error_msg = EXCLUDED.error_msg,
			created_at = NOW()
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		status.NodeID, status.Date, status.Color,
		status.Attempts, status.Success, status.ErrorMsg,
	).Scan(&status.ID, &status.CreatedAt)

	if err != nil {
		return fmt.Errorf("create status: %w", err)
	}

	return nil
}

func (r *statusRepository) GetStatusByNodeAndDate(ctx context.Context, nodeID int, date time.Time) (*models.DailyStatus, error) {
	query := `
		SELECT id, node_id, date, color, attempts, success, error_msg, created_at
		FROM daily_status
		WHERE node_id = $1 AND date = $2
	`

	status := &models.DailyStatus{}
	err := r.db.QueryRowContext(ctx, query, nodeID, date).Scan(
		&status.ID, &status.NodeID, &status.Date, &status.Color,
		&status.Attempts, &status.Success, &status.ErrorMsg, &status.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("get status: %w", err)
	}

	return status, nil
}

func (r *statusRepository) GetRecentStatusesByNode(ctx context.Context, nodeID int, days int) ([]models.StatusItem, error) {
	query := `
		SELECT color, date
		FROM daily_status
		WHERE node_id = $1 AND date >= CURRENT_DATE - INTERVAL '1 day' * $2
		ORDER BY date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, nodeID, days)
	if err != nil {
		return nil, fmt.Errorf("query recent statuses: %w", err)
	}
	defer rows.Close()

	var statuses []models.StatusItem
	for rows.Next() {
		var color int
		var date time.Time

		if err := rows.Scan(&color, &date); err != nil {
			return nil, fmt.Errorf("scan status: %w", err)
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

func (r *statusRepository) HasStatusForDate(ctx context.Context, nodeID int, date time.Time) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM daily_status WHERE node_id = $1 AND date = $2)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, nodeID, date).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check status exists: %w", err)
	}

	return exists, nil
}

func (r *statusRepository) GetStatusesByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*models.DailyStatus, error) {
	query := `
		SELECT id, node_id, date, color, attempts, success, error_msg, created_at
		FROM daily_status
		WHERE date >= $1 AND date <= $2
		ORDER BY date DESC, node_id
	`

	rows, err := r.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query statuses by date range: %w", err)
	}
	defer rows.Close()

	var statuses []*models.DailyStatus
	for rows.Next() {
		status := &models.DailyStatus{}
		err := rows.Scan(
			&status.ID, &status.NodeID, &status.Date, &status.Color,
			&status.Attempts, &status.Success, &status.ErrorMsg, &status.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan status: %w", err)
		}
		statuses = append(statuses, status)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return statuses, nil
}

func (r *statusRepository) DeleteOldStatuses(ctx context.Context, beforeDate time.Time) error {
	query := `DELETE FROM daily_status WHERE date < $1`

	result, err := r.db.ExecContext(ctx, query, beforeDate)
	if err != nil {
		return fmt.Errorf("delete old statuses: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		// Log this if needed
	}

	return nil
}
