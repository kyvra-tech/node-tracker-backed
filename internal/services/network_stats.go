package services

import (
	"context"
	"time"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/repositories"
	"github.com/sirupsen/logrus"
)

// NetworkStatsService handles network statistics
type NetworkStatsService struct {
	peerRepo     repositories.PeerRepository
	grpcRepo     repositories.GRPCRepository
	jsonrpcRepo  repositories.JSONRPCServerRepository
	bootstrapRepo repositories.BootstrapRepository
	snapshotRepo repositories.SnapshotRepository
	geoService   *GeoLocationService
	logger       *logrus.Logger
}

// NewNetworkStatsService creates a new network stats service
func NewNetworkStatsService(
	peerRepo repositories.PeerRepository,
	grpcRepo repositories.GRPCRepository,
	jsonrpcRepo repositories.JSONRPCServerRepository,
	bootstrapRepo repositories.BootstrapRepository,
	snapshotRepo repositories.SnapshotRepository,
	geoService *GeoLocationService,
	logger *logrus.Logger,
) *NetworkStatsService {
	return &NetworkStatsService{
		peerRepo:     peerRepo,
		grpcRepo:     grpcRepo,
		jsonrpcRepo:  jsonrpcRepo,
		bootstrapRepo: bootstrapRepo,
		snapshotRepo: snapshotRepo,
		geoService:   geoService,
		logger:       logger,
	}
}

// GetNetworkStats returns current network statistics
func (s *NetworkStatsService) GetNetworkStats(ctx context.Context) (*models.NetworkStats, error) {
	// Get peer counts
	reachablePeers, _ := s.peerRepo.CountReachable(ctx)
	countries, _ := s.peerRepo.CountCountries(ctx)
	topCountries, _ := s.peerRepo.GetTopCountries(ctx, 10)
	avgUptime, _ := s.peerRepo.GetAvgUptime(ctx)

	// Get server counts
	grpcCount, _ := s.grpcRepo.GetServerCount(ctx, true)
	jsonrpcCount, _ := s.jsonrpcRepo.GetServerCount(ctx, true)
	bootstrapCount, _ := s.bootstrapRepo.GetActiveCount(ctx)

	totalNodes := reachablePeers + grpcCount + jsonrpcCount + bootstrapCount

	return &models.NetworkStats{
		TotalNodes:     totalNodes,
		ReachableNodes: reachablePeers,
		CountriesCount: countries,
		AvgUptime:      avgUptime,
		TopCountries:   topCountries,
		GRPCNodes:      grpcCount,
		JSONRPCNodes:   jsonrpcCount,
		BootstrapNodes: bootstrapCount,
	}, nil
}

// GetMapNodes returns all nodes formatted for map display
func (s *NetworkStatsService) GetMapNodes(ctx context.Context) ([]models.MapNode, error) {
	var mapNodes []models.MapNode

	// Get gRPC servers
	grpcServers, err := s.grpcRepo.GetActiveServers(ctx)
	if err == nil {
		for _, server := range grpcServers {
			if server.Latitude != 0 || server.Longitude != 0 {
				status := "online"
				if server.OverallScore < 50 {
					status = "offline"
				}
				mapNodes = append(mapNodes, models.MapNode{
					ID:          server.ID,
					Name:        server.Name,
					Type:        "grpc",
					Coordinates: []float64{server.Latitude, server.Longitude},
					Status:      status,
					Country:     server.Country,
					City:        server.City,
				})
			}
		}
	}

	// Get JSON-RPC servers
	jsonrpcServers, err := s.jsonrpcRepo.GetActiveServers(ctx)
	if err == nil {
		for _, server := range jsonrpcServers {
			if server.Latitude != 0 || server.Longitude != 0 {
				status := "online"
				if server.OverallScore < 50 {
					status = "offline"
				}
				mapNodes = append(mapNodes, models.MapNode{
					ID:          server.ID,
					Name:        server.Name,
					Type:        "jsonrpc",
					Coordinates: []float64{server.Latitude, server.Longitude},
					Status:      status,
					Country:     server.Country,
					City:        server.City,
				})
			}
		}
	}

	// Get bootstrap nodes
	bootstrapNodes, err := s.bootstrapRepo.GetActiveNodes(ctx)
	if err == nil {
		for _, node := range bootstrapNodes {
			if node.Latitude != 0 || node.Longitude != 0 {
				status := "online"
				if node.OverallScore < 50 {
					status = "offline"
				}
				mapNodes = append(mapNodes, models.MapNode{
					ID:          node.ID,
					Name:        node.Name,
					Type:        "bootstrap",
					Coordinates: []float64{node.Latitude, node.Longitude},
					Status:      status,
					Country:     node.Country,
					City:        node.City,
				})
			}
		}
	}

	// Get reachable peers
	peers, err := s.peerRepo.GetReachablePeers(ctx)
	if err == nil {
		for _, peer := range peers {
			if peer.Latitude != 0 || peer.Longitude != 0 {
				status := "online"
				if !peer.IsReachable {
					status = "offline"
				}
				mapNodes = append(mapNodes, models.MapNode{
					ID:          peer.ID,
					Name:        peer.PeerID[:12] + "...", // Truncate peer ID
					Type:        "peer",
					Coordinates: []float64{peer.Latitude, peer.Longitude},
					Status:      status,
					Country:     peer.Country,
					City:        peer.City,
				})
			}
		}
	}

	return mapNodes, nil
}

// CreateSnapshot creates a new network snapshot
func (s *NetworkStatsService) CreateSnapshot(ctx context.Context) error {
	stats, err := s.GetNetworkStats(ctx)
	if err != nil {
		return err
	}

	snapshot := &models.NetworkSnapshot{
		Timestamp:      time.Now(),
		TotalNodes:     stats.TotalNodes,
		ReachableNodes: stats.ReachableNodes,
		CountriesCount: stats.CountriesCount,
		GRPCNodes:      stats.GRPCNodes,
		JSONRPCNodes:   stats.JSONRPCNodes,
		BootstrapNodes: stats.BootstrapNodes,
	}

	return s.snapshotRepo.CreateSnapshot(ctx, snapshot)
}

// GetSnapshots returns recent network snapshots
func (s *NetworkStatsService) GetSnapshots(ctx context.Context, limit int) ([]*models.NetworkSnapshot, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.snapshotRepo.GetSnapshots(ctx, limit)
}

// UpdateAllGeoLocations updates geo data for all nodes without geo data
func (s *NetworkStatsService) UpdateAllGeoLocations(ctx context.Context) error {
	if s.geoService == nil {
		s.logger.Warn("GeoService not available, skipping geo updates")
		return nil
	}

	// Update gRPC servers
	grpcServers, err := s.grpcRepo.GetActiveServers(ctx)
	if err == nil {
		for _, server := range grpcServers {
			if server.Latitude == 0 && server.Longitude == 0 && server.Address != "" {
				geo, err := s.geoService.LookupAddress(ctx, server.Address)
				if err != nil {
					s.logger.WithError(err).WithField("address", server.Address).Debug("Failed to lookup geo for gRPC server")
					continue
				}
				if geo != nil && geo.Status == "success" {
					err := s.grpcRepo.UpdateServerGeo(ctx, server.ID, geo.Country, geo.CountryCode, geo.City, geo.Latitude, geo.Longitude)
					if err != nil {
						s.logger.WithError(err).Error("Failed to update gRPC server geo")
					} else {
						s.logger.WithField("server", server.Name).Info("Updated geo for gRPC server")
					}
				}
			}
		}
	}

	// Update bootstrap nodes
	bootstrapNodes, err := s.bootstrapRepo.GetActiveNodes(ctx)
	if err == nil {
		for _, node := range bootstrapNodes {
			if node.Latitude == 0 && node.Longitude == 0 && node.Address != "" {
				geo, err := s.geoService.LookupAddress(ctx, node.Address)
				if err != nil {
					s.logger.WithError(err).WithField("address", node.Address).Debug("Failed to lookup geo for bootstrap node")
					continue
				}
				if geo != nil && geo.Status == "success" {
					err := s.bootstrapRepo.UpdateNodeGeo(ctx, node.ID, geo.Country, geo.CountryCode, geo.City, geo.Latitude, geo.Longitude)
					if err != nil {
						s.logger.WithError(err).Error("Failed to update bootstrap node geo")
					} else {
						s.logger.WithField("node", node.Name).Info("Updated geo for bootstrap node")
					}
				}
			}
		}
	}

	s.logger.Info("Completed geo location updates for all nodes")
	return nil
}
