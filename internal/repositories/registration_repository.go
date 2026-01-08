package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
)

// RegistrationRepository defines the interface for registration data access
type RegistrationRepository interface {
	Create(ctx context.Context, registration *models.NodeRegistration) error
	GetByID(ctx context.Context, id int) (*models.NodeRegistration, error)
	GetByStatus(ctx context.Context, status string) ([]*models.NodeRegistration, error)
	GetAll(ctx context.Context) ([]*models.NodeRegistration, error)
	UpdateStatus(ctx context.Context, id int, status, reason, reviewedBy string, reviewedAt *time.Time) error
	ExistsByAddress(ctx context.Context, address string) (bool, error)
}

type registrationRepository struct {
	db *sql.DB
}

// NewRegistrationRepository creates a new registration repository
func NewRegistrationRepository(db *sql.DB) RegistrationRepository {
	return &registrationRepository{db: db}
}

func (r *registrationRepository) Create(ctx context.Context, registration *models.NodeRegistration) error {
	query := `
		INSERT INTO node_registrations (node_type, name, address, network, email, website, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		registration.NodeType, registration.Name, registration.Address,
		registration.Network, registration.Email, registration.Website, registration.Status,
	).Scan(&registration.ID, &registration.CreatedAt)

	if err != nil {
		return fmt.Errorf("create registration: %w", err)
	}

	return nil
}

func (r *registrationRepository) GetByID(ctx context.Context, id int) (*models.NodeRegistration, error) {
	query := `
		SELECT id, node_type, name, address, network, email, website, status, rejection_reason, created_at, reviewed_at, reviewed_by
		FROM node_registrations
		WHERE id = $1
	`

	registration := &models.NodeRegistration{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&registration.ID, &registration.NodeType, &registration.Name, &registration.Address,
		&registration.Network, &registration.Email, &registration.Website, &registration.Status,
		&registration.RejectionReason, &registration.CreatedAt, &registration.ReviewedAt, &registration.ReviewedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get registration by id: %w", err)
	}

	return registration, nil
}

func (r *registrationRepository) GetByStatus(ctx context.Context, status string) ([]*models.NodeRegistration, error) {
	query := `
		SELECT id, node_type, name, address, network, email, website, status, rejection_reason, created_at, reviewed_at, reviewed_by
		FROM node_registrations
		WHERE status = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("query registrations by status: %w", err)
	}
	defer rows.Close()

	return r.scanRegistrations(rows)
}

func (r *registrationRepository) GetAll(ctx context.Context) ([]*models.NodeRegistration, error) {
	query := `
		SELECT id, node_type, name, address, network, email, website, status, rejection_reason, created_at, reviewed_at, reviewed_by
		FROM node_registrations
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all registrations: %w", err)
	}
	defer rows.Close()

	return r.scanRegistrations(rows)
}

func (r *registrationRepository) UpdateStatus(ctx context.Context, id int, status, reason, reviewedBy string, reviewedAt *time.Time) error {
	query := `
		UPDATE node_registrations SET
			status = $1, rejection_reason = $2, reviewed_by = $3, reviewed_at = $4
		WHERE id = $5
	`

	_, err := r.db.ExecContext(ctx, query, status, reason, reviewedBy, reviewedAt, id)
	if err != nil {
		return fmt.Errorf("update registration status: %w", err)
	}

	return nil
}

func (r *registrationRepository) ExistsByAddress(ctx context.Context, address string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM node_registrations WHERE address = $1 AND status != 'rejected')`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, address).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check registration exists: %w", err)
	}

	return exists, nil
}

// Helper function to scan multiple registrations
func (r *registrationRepository) scanRegistrations(rows *sql.Rows) ([]*models.NodeRegistration, error) {
	var registrations []*models.NodeRegistration

	for rows.Next() {
		registration := &models.NodeRegistration{}
		err := rows.Scan(
			&registration.ID, &registration.NodeType, &registration.Name, &registration.Address,
			&registration.Network, &registration.Email, &registration.Website, &registration.Status,
			&registration.RejectionReason, &registration.CreatedAt, &registration.ReviewedAt, &registration.ReviewedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("scan registration: %w", err)
		}
		registrations = append(registrations, registration)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return registrations, nil
}
