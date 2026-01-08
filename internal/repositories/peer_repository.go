package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
)

// PeerRepository defines the interface for peer data access
type PeerRepository interface {
	// Peer operations
	GetAllPeers(ctx context.Context) ([]*models.ReachablePeer, error)
	GetReachablePeers(ctx context.Context) ([]*models.ReachablePeer, error)
	GetPeerByID(ctx context.Context, id int) (*models.ReachablePeer, error)
	GetPeerByPeerID(ctx context.Context, peerID string) (*models.ReachablePeer, error)
	
	// CRUD operations
	CreatePeer(ctx context.Context, peer *models.ReachablePeer) error
	UpsertPeer(ctx context.Context, peer *models.ReachablePeer) error
	UpdatePeer(ctx context.Context, peer *models.ReachablePeer) error
	UpdatePeerGeo(ctx context.Context, id int, geo *models.GeoLocation) error
	
	// Aggregations
	CountReachable(ctx context.Context) (int, error)
	CountCountries(ctx context.Context) (int, error)
	GetTopCountries(ctx context.Context, limit int) ([]models.CountryStats, error)
	GetAvgUptime(ctx context.Context) (float64, error)
}

type peerRepository struct {
	db *sql.DB
}

// NewPeerRepository creates a new peer repository
func NewPeerRepository(db *sql.DB) PeerRepository {
	return &peerRepository{db: db}
}

func (r *peerRepository) GetAllPeers(ctx context.Context) ([]*models.ReachablePeer, error) {
	query := `
		SELECT id, peer_id, address, protocol, user_agent, last_seen, first_seen,
			   ip_address, country, country_code, city, latitude, longitude, timezone, asn, organization,
			   is_reachable, connection_attempts, successful_connections, overall_score,
			   created_at, updated_at
		FROM reachable_peers
		ORDER BY last_seen DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all peers: %w", err)
	}
	defer rows.Close()

	return r.scanPeers(rows)
}

func (r *peerRepository) GetReachablePeers(ctx context.Context) ([]*models.ReachablePeer, error) {
	query := `
		SELECT id, peer_id, address, protocol, user_agent, last_seen, first_seen,
			   ip_address, country, country_code, city, latitude, longitude, timezone, asn, organization,
			   is_reachable, connection_attempts, successful_connections, overall_score,
			   created_at, updated_at
		FROM reachable_peers
		WHERE is_reachable = true
		ORDER BY last_seen DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query reachable peers: %w", err)
	}
	defer rows.Close()

	return r.scanPeers(rows)
}

func (r *peerRepository) GetPeerByID(ctx context.Context, id int) (*models.ReachablePeer, error) {
	query := `
		SELECT id, peer_id, address, protocol, user_agent, last_seen, first_seen,
			   ip_address, country, country_code, city, latitude, longitude, timezone, asn, organization,
			   is_reachable, connection_attempts, successful_connections, overall_score,
			   created_at, updated_at
		FROM reachable_peers
		WHERE id = $1
	`

	peer := &models.ReachablePeer{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&peer.ID, &peer.PeerID, &peer.Address, &peer.Protocol, &peer.UserAgent,
		&peer.LastSeen, &peer.FirstSeen, &peer.IPAddress, &peer.Country, &peer.CountryCode,
		&peer.City, &peer.Latitude, &peer.Longitude, &peer.Timezone, &peer.ASN, &peer.Organization,
		&peer.IsReachable, &peer.ConnectionAttempts, &peer.SuccessfulConnections, &peer.OverallScore,
		&peer.CreatedAt, &peer.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get peer by id: %w", err)
	}

	return peer, nil
}

func (r *peerRepository) GetPeerByPeerID(ctx context.Context, peerID string) (*models.ReachablePeer, error) {
	query := `
		SELECT id, peer_id, address, protocol, user_agent, last_seen, first_seen,
			   ip_address, country, country_code, city, latitude, longitude, timezone, asn, organization,
			   is_reachable, connection_attempts, successful_connections, overall_score,
			   created_at, updated_at
		FROM reachable_peers
		WHERE peer_id = $1
	`

	peer := &models.ReachablePeer{}
	err := r.db.QueryRowContext(ctx, query, peerID).Scan(
		&peer.ID, &peer.PeerID, &peer.Address, &peer.Protocol, &peer.UserAgent,
		&peer.LastSeen, &peer.FirstSeen, &peer.IPAddress, &peer.Country, &peer.CountryCode,
		&peer.City, &peer.Latitude, &peer.Longitude, &peer.Timezone, &peer.ASN, &peer.Organization,
		&peer.IsReachable, &peer.ConnectionAttempts, &peer.SuccessfulConnections, &peer.OverallScore,
		&peer.CreatedAt, &peer.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get peer by peer_id: %w", err)
	}

	return peer, nil
}

func (r *peerRepository) CreatePeer(ctx context.Context, peer *models.ReachablePeer) error {
	query := `
		INSERT INTO reachable_peers (
			peer_id, address, protocol, user_agent, last_seen, first_seen,
			ip_address, country, country_code, city, latitude, longitude, timezone, asn, organization,
			is_reachable, connection_attempts, successful_connections, overall_score
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		peer.PeerID, peer.Address, peer.Protocol, peer.UserAgent, peer.LastSeen, peer.FirstSeen,
		peer.IPAddress, peer.Country, peer.CountryCode, peer.City, peer.Latitude, peer.Longitude,
		peer.Timezone, peer.ASN, peer.Organization, peer.IsReachable, peer.ConnectionAttempts,
		peer.SuccessfulConnections, peer.OverallScore,
	).Scan(&peer.ID, &peer.CreatedAt, &peer.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create peer: %w", err)
	}

	return nil
}

func (r *peerRepository) UpsertPeer(ctx context.Context, peer *models.ReachablePeer) error {
	query := `
		INSERT INTO reachable_peers (
			peer_id, address, protocol, user_agent, last_seen, first_seen,
			ip_address, country, country_code, city, latitude, longitude, timezone, asn, organization,
			is_reachable, connection_attempts, successful_connections, overall_score
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (peer_id) DO UPDATE SET
			address = EXCLUDED.address,
			protocol = EXCLUDED.protocol,
			user_agent = EXCLUDED.user_agent,
			last_seen = EXCLUDED.last_seen,
			ip_address = EXCLUDED.ip_address,
			country = EXCLUDED.country,
			country_code = EXCLUDED.country_code,
			city = EXCLUDED.city,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			timezone = EXCLUDED.timezone,
			asn = EXCLUDED.asn,
			organization = EXCLUDED.organization,
			is_reachable = EXCLUDED.is_reachable,
			connection_attempts = reachable_peers.connection_attempts + 1,
			successful_connections = CASE WHEN EXCLUDED.is_reachable THEN reachable_peers.successful_connections + 1 ELSE reachable_peers.successful_connections END,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		peer.PeerID, peer.Address, peer.Protocol, peer.UserAgent, peer.LastSeen, peer.FirstSeen,
		peer.IPAddress, peer.Country, peer.CountryCode, peer.City, peer.Latitude, peer.Longitude,
		peer.Timezone, peer.ASN, peer.Organization, peer.IsReachable, peer.ConnectionAttempts,
		peer.SuccessfulConnections, peer.OverallScore,
	).Scan(&peer.ID, &peer.CreatedAt, &peer.UpdatedAt)

	if err != nil {
		return fmt.Errorf("upsert peer: %w", err)
	}

	return nil
}

func (r *peerRepository) UpdatePeer(ctx context.Context, peer *models.ReachablePeer) error {
	query := `
		UPDATE reachable_peers SET
			address = $1, protocol = $2, user_agent = $3, last_seen = $4,
			is_reachable = $5, connection_attempts = $6, successful_connections = $7,
			overall_score = $8, updated_at = NOW()
		WHERE id = $9
	`

	_, err := r.db.ExecContext(ctx, query,
		peer.Address, peer.Protocol, peer.UserAgent, peer.LastSeen,
		peer.IsReachable, peer.ConnectionAttempts, peer.SuccessfulConnections,
		peer.OverallScore, peer.ID,
	)

	if err != nil {
		return fmt.Errorf("update peer: %w", err)
	}

	return nil
}

func (r *peerRepository) UpdatePeerGeo(ctx context.Context, id int, geo *models.GeoLocation) error {
	query := `
		UPDATE reachable_peers SET
			ip_address = $1, country = $2, country_code = $3, city = $4,
			latitude = $5, longitude = $6, timezone = $7, asn = $8, organization = $9,
			updated_at = NOW()
		WHERE id = $10
	`

	_, err := r.db.ExecContext(ctx, query,
		geo.Query, geo.Country, geo.CountryCode, geo.City,
		geo.Latitude, geo.Longitude, geo.Timezone, geo.AS, geo.Org, id,
	)

	if err != nil {
		return fmt.Errorf("update peer geo: %w", err)
	}

	return nil
}

func (r *peerRepository) CountReachable(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM reachable_peers WHERE is_reachable = true`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count reachable: %w", err)
	}

	return count, nil
}

func (r *peerRepository) CountCountries(ctx context.Context) (int, error) {
	query := `SELECT COUNT(DISTINCT country_code) FROM reachable_peers WHERE country_code IS NOT NULL AND country_code != ''`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count countries: %w", err)
	}

	return count, nil
}

func (r *peerRepository) GetTopCountries(ctx context.Context, limit int) ([]models.CountryStats, error) {
	query := `
		SELECT country, country_code, COUNT(*) as count
		FROM reachable_peers
		WHERE country IS NOT NULL AND country != '' AND is_reachable = true
		GROUP BY country, country_code
		ORDER BY count DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get top countries: %w", err)
	}
	defer rows.Close()

	var stats []models.CountryStats
	for rows.Next() {
		var s models.CountryStats
		if err := rows.Scan(&s.Country, &s.CountryCode, &s.Count); err != nil {
			return nil, fmt.Errorf("scan country stats: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, nil
}

func (r *peerRepository) GetAvgUptime(ctx context.Context) (float64, error) {
	query := `
		SELECT COALESCE(AVG(
			CASE WHEN connection_attempts > 0 
				 THEN (successful_connections::float / connection_attempts::float) * 100 
				 ELSE 0 
			END
		), 0)
		FROM reachable_peers
		WHERE is_reachable = true
	`

	var avg float64
	err := r.db.QueryRowContext(ctx, query).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("get avg uptime: %w", err)
	}

	return avg, nil
}

// Helper function to scan multiple peers
func (r *peerRepository) scanPeers(rows *sql.Rows) ([]*models.ReachablePeer, error) {
	var peers []*models.ReachablePeer

	for rows.Next() {
		peer := &models.ReachablePeer{}
		err := rows.Scan(
			&peer.ID, &peer.PeerID, &peer.Address, &peer.Protocol, &peer.UserAgent,
			&peer.LastSeen, &peer.FirstSeen, &peer.IPAddress, &peer.Country, &peer.CountryCode,
			&peer.City, &peer.Latitude, &peer.Longitude, &peer.Timezone, &peer.ASN, &peer.Organization,
			&peer.IsReachable, &peer.ConnectionAttempts, &peer.SuccessfulConnections, &peer.OverallScore,
			&peer.CreatedAt, &peer.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan peer: %w", err)
		}
		peers = append(peers, peer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return peers, nil
}
