package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
)

// SnapshotRepository defines the interface for network snapshot data access
type SnapshotRepository interface {
	CreateSnapshot(ctx context.Context, snapshot *models.NetworkSnapshot) error
	GetLatestSnapshot(ctx context.Context) (*models.NetworkSnapshot, error)
	GetSnapshots(ctx context.Context, limit int) ([]*models.NetworkSnapshot, error)
	GetSnapshotsByDateRange(ctx context.Context, start, end time.Time) ([]*models.NetworkSnapshot, error)
}

type snapshotRepository struct {
	db *sql.DB
}

// NewSnapshotRepository creates a new snapshot repository
func NewSnapshotRepository(db *sql.DB) SnapshotRepository {
	return &snapshotRepository{db: db}
}

func (r *snapshotRepository) CreateSnapshot(ctx context.Context, snapshot *models.NetworkSnapshot) error {
	query := `
		INSERT INTO network_snapshots (timestamp, total_nodes, reachable_nodes, countries_count, grpc_nodes, jsonrpc_nodes, bootstrap_nodes, snapshot_data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	var snapshotData []byte
	if snapshot.SnapshotData != nil {
		snapshotData = snapshot.SnapshotData
	} else {
		snapshotData, _ = json.Marshal(map[string]interface{}{})
	}

	err := r.db.QueryRowContext(ctx, query,
		snapshot.Timestamp, snapshot.TotalNodes, snapshot.ReachableNodes,
		snapshot.CountriesCount, snapshot.GRPCNodes, snapshot.JSONRPCNodes,
		snapshot.BootstrapNodes, snapshotData,
	).Scan(&snapshot.ID, &snapshot.CreatedAt)

	if err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}

	return nil
}

func (r *snapshotRepository) GetLatestSnapshot(ctx context.Context) (*models.NetworkSnapshot, error) {
	query := `
		SELECT id, timestamp, total_nodes, reachable_nodes, countries_count, grpc_nodes, jsonrpc_nodes, bootstrap_nodes, snapshot_data, created_at
		FROM network_snapshots
		ORDER BY timestamp DESC
		LIMIT 1
	`

	snapshot := &models.NetworkSnapshot{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&snapshot.ID, &snapshot.Timestamp, &snapshot.TotalNodes, &snapshot.ReachableNodes,
		&snapshot.CountriesCount, &snapshot.GRPCNodes, &snapshot.JSONRPCNodes,
		&snapshot.BootstrapNodes, &snapshot.SnapshotData, &snapshot.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest snapshot: %w", err)
	}

	return snapshot, nil
}

func (r *snapshotRepository) GetSnapshots(ctx context.Context, limit int) ([]*models.NetworkSnapshot, error) {
	query := `
		SELECT id, timestamp, total_nodes, reachable_nodes, countries_count, grpc_nodes, jsonrpc_nodes, bootstrap_nodes, snapshot_data, created_at
		FROM network_snapshots
		ORDER BY timestamp DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query snapshots: %w", err)
	}
	defer rows.Close()

	return r.scanSnapshots(rows)
}

func (r *snapshotRepository) GetSnapshotsByDateRange(ctx context.Context, start, end time.Time) ([]*models.NetworkSnapshot, error) {
	query := `
		SELECT id, timestamp, total_nodes, reachable_nodes, countries_count, grpc_nodes, jsonrpc_nodes, bootstrap_nodes, snapshot_data, created_at
		FROM network_snapshots
		WHERE timestamp >= $1 AND timestamp <= $2
		ORDER BY timestamp DESC
	`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("query snapshots by date range: %w", err)
	}
	defer rows.Close()

	return r.scanSnapshots(rows)
}

// Helper function to scan multiple snapshots
func (r *snapshotRepository) scanSnapshots(rows *sql.Rows) ([]*models.NetworkSnapshot, error) {
	var snapshots []*models.NetworkSnapshot

	for rows.Next() {
		snapshot := &models.NetworkSnapshot{}
		err := rows.Scan(
			&snapshot.ID, &snapshot.Timestamp, &snapshot.TotalNodes, &snapshot.ReachableNodes,
			&snapshot.CountriesCount, &snapshot.GRPCNodes, &snapshot.JSONRPCNodes,
			&snapshot.BootstrapNodes, &snapshot.SnapshotData, &snapshot.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return snapshots, nil
}
