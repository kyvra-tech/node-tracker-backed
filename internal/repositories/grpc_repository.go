package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
)

// GRPCRepository defines the interface for gRPC server data access
type GRPCRepository interface {
	// Server operations
	GetActiveServers(ctx context.Context) ([]*models.GRPCServer, error)
	GetAllServers(ctx context.Context) ([]*models.GRPCServer, error)
	GetServerByID(ctx context.Context, id int) (*models.GRPCServer, error)
	GetServerByAddress(ctx context.Context, address string) (*models.GRPCServer, error)
	GetServersByNetwork(ctx context.Context, network string) ([]*models.GRPCServer, error)

	// CRUD operations
	CreateServer(ctx context.Context, server *models.GRPCServer) error
	UpdateServer(ctx context.Context, server *models.GRPCServer) error
	UpdateServerScore(ctx context.Context, serverID int, score float64) error
	UpdateServerGeo(ctx context.Context, serverID int, country, countryCode, city string, lat, lon float64) error
	DeactivateServer(ctx context.Context, address string) error
	ServerExists(ctx context.Context, address string) (bool, error)

	// Aggregations
	GetServerCount(ctx context.Context, activeOnly bool) (int, error)
	UpdateAllScores(ctx context.Context) error
}

type grpcRepository struct {
	db *sql.DB
}

// NewGRPCRepository creates a new gRPC repository
func NewGRPCRepository(db *sql.DB) GRPCRepository {
	return &grpcRepository{db: db}
}

func (r *grpcRepository) GetActiveServers(ctx context.Context) ([]*models.GRPCServer, error) {
	query := `
SELECT id, name, address, network, overall_score, is_active, email, website, COALESCE(country, ''), COALESCE(country_code, ''), COALESCE(city, ''), COALESCE(latitude, 0), COALESCE(longitude, 0), created_at, updated_at
FROM grpc_servers 
WHERE is_active = true
ORDER BY network, id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query active servers: %w", err)
	}
	defer rows.Close()

	return r.scanServers(rows)
}

func (r *grpcRepository) GetAllServers(ctx context.Context) ([]*models.GRPCServer, error) {
	query := `
SELECT id, name, address, network, overall_score, is_active, email, website, COALESCE(country, ''), COALESCE(country_code, ''), COALESCE(city, ''), COALESCE(latitude, 0), COALESCE(longitude, 0), created_at, updated_at
FROM grpc_servers 
ORDER BY network, id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all servers: %w", err)
	}
	defer rows.Close()

	return r.scanServers(rows)
}

func (r *grpcRepository) GetServerByID(ctx context.Context, id int) (*models.GRPCServer, error) {
	query := `
SELECT id, name, address, network, overall_score, is_active, email, website, COALESCE(country, ''), COALESCE(country_code, ''), COALESCE(city, ''), COALESCE(latitude, 0), COALESCE(longitude, 0), created_at, updated_at
FROM grpc_servers 
WHERE id = $1
	`

	server := &models.GRPCServer{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&server.ID, &server.Name, &server.Address, &server.Network,
		&server.OverallScore, &server.IsActive, &server.Email, &server.Website,
		&server.Country, &server.CountryCode, &server.City, &server.Latitude, &server.Longitude,
		&server.CreatedAt, &server.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("server not found: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get server by id: %w", err)
	}

	return server, nil
}

func (r *grpcRepository) GetServerByAddress(ctx context.Context, address string) (*models.GRPCServer, error) {
	query := `
SELECT id, name, address, network, overall_score, is_active, email, website, COALESCE(country, ''), COALESCE(country_code, ''), COALESCE(city, ''), COALESCE(latitude, 0), COALESCE(longitude, 0), created_at, updated_at
FROM grpc_servers 
WHERE address = $1
	`

	server := &models.GRPCServer{}
	err := r.db.QueryRowContext(ctx, query, address).Scan(
		&server.ID, &server.Name, &server.Address, &server.Network,
		&server.OverallScore, &server.IsActive, &server.Email, &server.Website,
		&server.Country, &server.CountryCode, &server.City, &server.Latitude, &server.Longitude,
		&server.CreatedAt, &server.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("get server by address: %w", err)
	}

	return server, nil
}

func (r *grpcRepository) GetServersByNetwork(ctx context.Context, network string) ([]*models.GRPCServer, error) {
	query := `
SELECT id, name, address, network, overall_score, is_active, email, website, COALESCE(country, ''), COALESCE(country_code, ''), COALESCE(city, ''), COALESCE(latitude, 0), COALESCE(longitude, 0), created_at, updated_at
FROM grpc_servers 
WHERE network = $1 AND is_active = true
ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, network)
	if err != nil {
		return nil, fmt.Errorf("query servers by network: %w", err)
	}
	defer rows.Close()

	return r.scanServers(rows)
}

func (r *grpcRepository) CreateServer(ctx context.Context, server *models.GRPCServer) error {
	query := `
        INSERT INTO grpc_servers (name, address, network, email, website, is_active, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
        ON CONFLICT (address) DO NOTHING
        RETURNING id, created_at, updated_at
    `

	err := r.db.QueryRowContext(ctx, query,
		server.Name, server.Address, server.Network, server.Email, server.Website, server.IsActive,
	).Scan(&server.ID, &server.CreatedAt, &server.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil // Conflict, server already exists
	}
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	return nil
}

func (r *grpcRepository) UpdateServer(ctx context.Context, server *models.GRPCServer) error {
	query := `
	UPDATE grpc_servers 
	SET name = $1, network = $2, email = $3, website = $4, updated_at = NOW()
	WHERE address = $5
	RETURNING updated_at
`

	err := r.db.QueryRowContext(ctx, query,
		server.Name, server.Network, server.Email, server.Website, server.Address,
	).Scan(&server.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("server not found: %s", server.Address)
	}
	if err != nil {
		return fmt.Errorf("update server: %w", err)
	}

	return nil
}

func (r *grpcRepository) UpdateServerScore(ctx context.Context, serverID int, score float64) error {
	query := `
		UPDATE grpc_servers 
		SET overall_score = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, score, serverID)
	if err != nil {
		return fmt.Errorf("update server score: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("server not found: %d", serverID)
	}

	return nil
}

func (r *grpcRepository) UpdateServerGeo(ctx context.Context, serverID int, country, countryCode, city string, lat, lon float64) error {
	query := `
		UPDATE grpc_servers 
		SET country = $1, country_code = $2, city = $3, latitude = $4, longitude = $5, updated_at = NOW()
		WHERE id = $6
	`

	result, err := r.db.ExecContext(ctx, query, country, countryCode, city, lat, lon, serverID)
	if err != nil {
		return fmt.Errorf("update server geo: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("server not found: %d", serverID)
	}

	return nil
}

func (r *grpcRepository) DeactivateServer(ctx context.Context, address string) error {
	query := `
		UPDATE grpc_servers 
		SET is_active = false, updated_at = NOW() 
		WHERE address = $1
	`

	_, err := r.db.ExecContext(ctx, query, address)
	if err != nil {
		return fmt.Errorf("deactivate server: %w", err)
	}

	return nil
}

func (r *grpcRepository) ServerExists(ctx context.Context, address string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM grpc_servers WHERE address = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, address).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check server exists: %w", err)
	}

	return exists, nil
}

func (r *grpcRepository) GetServerCount(ctx context.Context, activeOnly bool) (int, error) {
	query := `SELECT COUNT(*) FROM grpc_servers`
	if activeOnly {
		query += ` WHERE is_active = true`
	}

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get server count: %w", err)
	}

	return count, nil
}

func (r *grpcRepository) UpdateAllScores(ctx context.Context) error {
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

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("update all scores: %w", err)
	}

	return nil
}

// Helper function to scan multiple servers
func (r *grpcRepository) scanServers(rows *sql.Rows) ([]*models.GRPCServer, error) {
	var servers []*models.GRPCServer

	for rows.Next() {
		server := &models.GRPCServer{}
		err := rows.Scan(
			&server.ID, &server.Name, &server.Address, &server.Network,
			&server.OverallScore, &server.IsActive, &server.Email, &server.Website,
			&server.Country, &server.CountryCode, &server.City, &server.Latitude, &server.Longitude,
			&server.CreatedAt, &server.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan server: %w", err)
		}
		servers = append(servers, server)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return servers, nil
}
