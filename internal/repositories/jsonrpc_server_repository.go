package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
)

// JSONRPCServerRepository defines the interface for JSON-RPC server data access
type JSONRPCServerRepository interface {
	// Server operations
	GetActiveServers(ctx context.Context) ([]*models.JSONRPCServer, error)
	GetAllServers(ctx context.Context) ([]*models.JSONRPCServer, error)
	GetServerByID(ctx context.Context, id int) (*models.JSONRPCServer, error)
	GetServerByAddress(ctx context.Context, address string) (*models.JSONRPCServer, error)
	GetServersByNetwork(ctx context.Context, network string) ([]*models.JSONRPCServer, error)

	// CRUD operations
	CreateServer(ctx context.Context, server *models.JSONRPCServer) error
	UpdateServer(ctx context.Context, server *models.JSONRPCServer) error
	UpdateServerGeo(ctx context.Context, id int, geo *models.GeoLocation) error
	UpdateServerScore(ctx context.Context, serverID int, score float64) error
	DeactivateServer(ctx context.Context, address string) error
	ExistsByAddress(ctx context.Context, address string) (bool, error)

	// Aggregations
	GetServerCount(ctx context.Context, activeOnly bool) (int, error)
	UpdateAllScores(ctx context.Context) error
}

type jsonrpcServerRepository struct {
	db *sql.DB
}

// NewJSONRPCServerRepository creates a new JSON-RPC server repository
func NewJSONRPCServerRepository(db *sql.DB) JSONRPCServerRepository {
	return &jsonrpcServerRepository{db: db}
}

func (r *jsonrpcServerRepository) GetActiveServers(ctx context.Context) ([]*models.JSONRPCServer, error) {
	query := `
		SELECT id, name, address, network, email, website, country, country_code, city, latitude, longitude,
			   overall_score, is_active, is_verified, created_at, updated_at
		FROM jsonrpc_servers
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

func (r *jsonrpcServerRepository) GetAllServers(ctx context.Context) ([]*models.JSONRPCServer, error) {
	query := `
		SELECT id, name, address, network, email, website, country, country_code, city, latitude, longitude,
			   overall_score, is_active, is_verified, created_at, updated_at
		FROM jsonrpc_servers
		ORDER BY network, id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all servers: %w", err)
	}
	defer rows.Close()

	return r.scanServers(rows)
}

func (r *jsonrpcServerRepository) GetServerByID(ctx context.Context, id int) (*models.JSONRPCServer, error) {
	query := `
		SELECT id, name, address, network, email, website, country, country_code, city, latitude, longitude,
			   overall_score, is_active, is_verified, created_at, updated_at
		FROM jsonrpc_servers
		WHERE id = $1
	`

	server := &models.JSONRPCServer{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&server.ID, &server.Name, &server.Address, &server.Network, &server.Email, &server.Website,
		&server.Country, &server.CountryCode, &server.City, &server.Latitude, &server.Longitude,
		&server.OverallScore, &server.IsActive, &server.IsVerified, &server.CreatedAt, &server.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get server by id: %w", err)
	}

	return server, nil
}

func (r *jsonrpcServerRepository) GetServerByAddress(ctx context.Context, address string) (*models.JSONRPCServer, error) {
	query := `
		SELECT id, name, address, network, email, website, country, country_code, city, latitude, longitude,
			   overall_score, is_active, is_verified, created_at, updated_at
		FROM jsonrpc_servers
		WHERE address = $1
	`

	server := &models.JSONRPCServer{}
	err := r.db.QueryRowContext(ctx, query, address).Scan(
		&server.ID, &server.Name, &server.Address, &server.Network, &server.Email, &server.Website,
		&server.Country, &server.CountryCode, &server.City, &server.Latitude, &server.Longitude,
		&server.OverallScore, &server.IsActive, &server.IsVerified, &server.CreatedAt, &server.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get server by address: %w", err)
	}

	return server, nil
}

func (r *jsonrpcServerRepository) GetServersByNetwork(ctx context.Context, network string) ([]*models.JSONRPCServer, error) {
	query := `
		SELECT id, name, address, network, email, website, country, country_code, city, latitude, longitude,
			   overall_score, is_active, is_verified, created_at, updated_at
		FROM jsonrpc_servers
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

func (r *jsonrpcServerRepository) CreateServer(ctx context.Context, server *models.JSONRPCServer) error {
	query := `
		INSERT INTO jsonrpc_servers (name, address, network, email, website, country, country_code, city, latitude, longitude, is_active, is_verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (address) DO NOTHING
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		server.Name, server.Address, server.Network, server.Email, server.Website,
		server.Country, server.CountryCode, server.City, server.Latitude, server.Longitude,
		server.IsActive, server.IsVerified,
	).Scan(&server.ID, &server.CreatedAt, &server.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil // Conflict, server already exists
	}
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	return nil
}

func (r *jsonrpcServerRepository) UpdateServer(ctx context.Context, server *models.JSONRPCServer) error {
	query := `
		UPDATE jsonrpc_servers SET
			name = $1, network = $2, email = $3, website = $4,
			country = $5, country_code = $6, city = $7, latitude = $8, longitude = $9,
			updated_at = NOW()
		WHERE address = $10
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		server.Name, server.Network, server.Email, server.Website,
		server.Country, server.CountryCode, server.City, server.Latitude, server.Longitude,
		server.Address,
	).Scan(&server.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("server not found: %s", server.Address)
	}
	if err != nil {
		return fmt.Errorf("update server: %w", err)
	}

	return nil
}

func (r *jsonrpcServerRepository) UpdateServerGeo(ctx context.Context, id int, geo *models.GeoLocation) error {
	query := `
		UPDATE jsonrpc_servers SET
			country = $1, country_code = $2, city = $3, latitude = $4, longitude = $5,
			updated_at = NOW()
		WHERE id = $6
	`

	_, err := r.db.ExecContext(ctx, query,
		geo.Country, geo.CountryCode, geo.City, geo.Latitude, geo.Longitude, id,
	)

	if err != nil {
		return fmt.Errorf("update server geo: %w", err)
	}

	return nil
}

func (r *jsonrpcServerRepository) UpdateServerScore(ctx context.Context, serverID int, score float64) error {
	query := `
		UPDATE jsonrpc_servers 
		SET overall_score = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, score, serverID)
	if err != nil {
		return fmt.Errorf("update server score: %w", err)
	}

	return nil
}

func (r *jsonrpcServerRepository) DeactivateServer(ctx context.Context, address string) error {
	query := `
		UPDATE jsonrpc_servers 
		SET is_active = false, updated_at = NOW() 
		WHERE address = $1
	`

	_, err := r.db.ExecContext(ctx, query, address)
	if err != nil {
		return fmt.Errorf("deactivate server: %w", err)
	}

	return nil
}

func (r *jsonrpcServerRepository) ExistsByAddress(ctx context.Context, address string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM jsonrpc_servers WHERE address = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, address).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check server exists: %w", err)
	}

	return exists, nil
}

func (r *jsonrpcServerRepository) GetServerCount(ctx context.Context, activeOnly bool) (int, error) {
	query := `SELECT COUNT(*) FROM jsonrpc_servers`
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

func (r *jsonrpcServerRepository) UpdateAllScores(ctx context.Context) error {
	query := `
		UPDATE jsonrpc_servers 
		SET overall_score = (
			SELECT COALESCE(
				ROUND(
					(COUNT(CASE WHEN success = true THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0))::numeric, 2
				), 0
			)
			FROM jsonrpc_daily_status 
			WHERE server_id = jsonrpc_servers.id 
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
func (r *jsonrpcServerRepository) scanServers(rows *sql.Rows) ([]*models.JSONRPCServer, error) {
	var servers []*models.JSONRPCServer

	for rows.Next() {
		server := &models.JSONRPCServer{}
		err := rows.Scan(
			&server.ID, &server.Name, &server.Address, &server.Network, &server.Email, &server.Website,
			&server.Country, &server.CountryCode, &server.City, &server.Latitude, &server.Longitude,
			&server.OverallScore, &server.IsActive, &server.IsVerified, &server.CreatedAt, &server.UpdatedAt,
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
