package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
	"github.com/lib/pq"
)

// BootstrapRepository defines the interface for bootstrap node data access
type BootstrapRepository interface {
	// Node operations
	GetActiveNodes(ctx context.Context) ([]*models.BootstrapNode, error)
	GetAllNodes(ctx context.Context) ([]*models.BootstrapNode, error)
	GetNodeByID(ctx context.Context, id int) (*models.BootstrapNode, error)
	GetNodeByAddress(ctx context.Context, address string) (*models.BootstrapNode, error)

	// CRUD operations
	CreateNode(ctx context.Context, node *models.BootstrapNode) error
	UpdateNode(ctx context.Context, node *models.BootstrapNode) error
	UpdateNodeScore(ctx context.Context, nodeID int, score float64) error
	UpdateNodeGeo(ctx context.Context, nodeID int, country, countryCode, city string, lat, lon float64) error
	DeactivateNodes(ctx context.Context, addresses []string) error

	// Aggregations
	GetNodeCount(ctx context.Context, activeOnly bool) (int, error)
	GetActiveCount(ctx context.Context) (int, error)
	UpdateAllScores(ctx context.Context) error
}

type bootstrapRepository struct {
	db *sql.DB
}

// NewBootstrapRepository creates a new bootstrap repository
func NewBootstrapRepository(db *sql.DB) BootstrapRepository {
	return &bootstrapRepository{db: db}
}

func (r *bootstrapRepository) GetActiveNodes(ctx context.Context) ([]*models.BootstrapNode, error) {
	query := `
		SELECT id, name, email, website, address, overall_score, is_active, COALESCE(country, ''), COALESCE(country_code, ''), COALESCE(city, ''), COALESCE(latitude, 0), COALESCE(longitude, 0), created_at, updated_at
		FROM bootstrap_nodes 
		WHERE is_active = true
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query active nodes: %w", err)
	}
	defer rows.Close()

	return r.scanNodes(rows)
}

func (r *bootstrapRepository) GetAllNodes(ctx context.Context) ([]*models.BootstrapNode, error) {
	query := `
		SELECT id, name, email, website, address, overall_score, is_active, COALESCE(country, ''), COALESCE(country_code, ''), COALESCE(city, ''), COALESCE(latitude, 0), COALESCE(longitude, 0), created_at, updated_at
		FROM bootstrap_nodes 
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all nodes: %w", err)
	}
	defer rows.Close()

	return r.scanNodes(rows)
}

func (r *bootstrapRepository) GetNodeByID(ctx context.Context, id int) (*models.BootstrapNode, error) {
	query := `
		SELECT id, name, email, website, address, overall_score, is_active, COALESCE(country, ''), COALESCE(country_code, ''), COALESCE(city, ''), COALESCE(latitude, 0), COALESCE(longitude, 0), created_at, updated_at
		FROM bootstrap_nodes 
		WHERE id = $1
	`

	node := &models.BootstrapNode{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&node.ID, &node.Name, &node.Email, &node.Website, &node.Address,
		&node.OverallScore, &node.IsActive,
		&node.Country, &node.CountryCode, &node.City, &node.Latitude, &node.Longitude,
		&node.CreatedAt, &node.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("node not found: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get node by id: %w", err)
	}

	return node, nil
}

func (r *bootstrapRepository) GetNodeByAddress(ctx context.Context, address string) (*models.BootstrapNode, error) {
	query := `
		SELECT id, name, email, website, address, overall_score, is_active, COALESCE(country, ''), COALESCE(country_code, ''), COALESCE(city, ''), COALESCE(latitude, 0), COALESCE(longitude, 0), created_at, updated_at
		FROM bootstrap_nodes 
		WHERE address = $1
	`

	node := &models.BootstrapNode{}
	err := r.db.QueryRowContext(ctx, query, address).Scan(
		&node.ID, &node.Name, &node.Email, &node.Website, &node.Address,
		&node.OverallScore, &node.IsActive,
		&node.Country, &node.CountryCode, &node.City, &node.Latitude, &node.Longitude,
		&node.CreatedAt, &node.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("get node by address: %w", err)
	}

	return node, nil
}

func (r *bootstrapRepository) CreateNode(ctx context.Context, node *models.BootstrapNode) error {
	query := `
		INSERT INTO bootstrap_nodes (name, email, website, address, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (address) DO NOTHING
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		node.Name, node.Email, node.Website, node.Address, node.IsActive,
	).Scan(&node.ID, &node.CreatedAt, &node.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil // Conflict, node already exists
	}
	if err != nil {
		return fmt.Errorf("create node: %w", err)
	}

	return nil
}

func (r *bootstrapRepository) UpdateNode(ctx context.Context, node *models.BootstrapNode) error {
	query := `
		UPDATE bootstrap_nodes 
		SET name = $1, email = $2, website = $3, updated_at = NOW()
		WHERE address = $4
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		node.Name, node.Email, node.Website, node.Address,
	).Scan(&node.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("node not found: %s", node.Address)
	}
	if err != nil {
		return fmt.Errorf("update node: %w", err)
	}

	return nil
}

func (r *bootstrapRepository) UpdateNodeScore(ctx context.Context, nodeID int, score float64) error {
	query := `
		UPDATE bootstrap_nodes 
		SET overall_score = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, score, nodeID)
	if err != nil {
		return fmt.Errorf("update node score: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("node not found: %d", nodeID)
	}

	return nil
}

func (r *bootstrapRepository) DeactivateNodes(ctx context.Context, addresses []string) error {
	if len(addresses) == 0 {
		return nil
	}

	query := `
		UPDATE bootstrap_nodes 
		SET is_active = false, updated_at = NOW() 
		WHERE address = ANY($1)
	`

	_, err := r.db.ExecContext(ctx, query, pq.Array(addresses))
	if err != nil {
		return fmt.Errorf("deactivate nodes: %w", err)
	}

	return nil
}

func (r *bootstrapRepository) GetNodeCount(ctx context.Context, activeOnly bool) (int, error) {
	query := `SELECT COUNT(*) FROM bootstrap_nodes`
	if activeOnly {
		query += ` WHERE is_active = true`
	}

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get node count: %w", err)
	}

	return count, nil
}

func (r *bootstrapRepository) UpdateAllScores(ctx context.Context) error {
	query := `
		UPDATE bootstrap_nodes 
		SET overall_score = (
			SELECT COALESCE(
				ROUND(
					(COUNT(CASE WHEN success = true THEN 1 END) * 100.0 / COUNT(*))::numeric, 2
				), 0
			)
			FROM daily_status 
			WHERE node_id = bootstrap_nodes.id 
			AND date >= CURRENT_DATE - INTERVAL '30 days'
		),
		updated_at = NOW()
		WHERE is_active = true
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("update all scores: %w", err)
	}

	return nil
}

func (r *bootstrapRepository) GetActiveCount(ctx context.Context) (int, error) {
	return r.GetNodeCount(ctx, true)
}

func (r *bootstrapRepository) UpdateNodeGeo(ctx context.Context, nodeID int, country, countryCode, city string, lat, lon float64) error {
	query := `
		UPDATE bootstrap_nodes 
		SET country = $1, country_code = $2, city = $3, latitude = $4, longitude = $5, updated_at = NOW()
		WHERE id = $6
	`

	result, err := r.db.ExecContext(ctx, query, country, countryCode, city, lat, lon, nodeID)
	if err != nil {
		return fmt.Errorf("update node geo: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("node not found: %d", nodeID)
	}

	return nil
}

// Helper function to scan multiple nodes
func (r *bootstrapRepository) scanNodes(rows *sql.Rows) ([]*models.BootstrapNode, error) {
	var nodes []*models.BootstrapNode

	for rows.Next() {
		node := &models.BootstrapNode{}
		err := rows.Scan(
			&node.ID, &node.Name, &node.Email, &node.Website, &node.Address,
			&node.OverallScore, &node.IsActive,
			&node.Country, &node.CountryCode, &node.City, &node.Latitude, &node.Longitude,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		nodes = append(nodes, node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return nodes, nil
}
