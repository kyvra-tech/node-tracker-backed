package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/repositories"
	"github.com/sirupsen/logrus"
)

// JSONRPCMonitorService handles JSON-RPC server monitoring
type JSONRPCMonitorService struct {
	serverRepo repositories.JSONRPCServerRepository
	statusRepo repositories.JSONRPCStatusRepository
	geoService *GeoLocationService
	logger     *logrus.Logger
	httpClient *http.Client
}

// NewJSONRPCMonitorService creates a new JSON-RPC monitor service
func NewJSONRPCMonitorService(
	serverRepo repositories.JSONRPCServerRepository,
	statusRepo repositories.JSONRPCStatusRepository,
	geoService *GeoLocationService,
	logger *logrus.Logger,
) *JSONRPCMonitorService {
	return &JSONRPCMonitorService{
		serverRepo: serverRepo,
		statusRepo: statusRepo,
		geoService: geoService,
		logger:     logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CheckAllServers performs health check on all active JSON-RPC servers
func (s *JSONRPCMonitorService) CheckAllServers(ctx context.Context) error {
	servers, err := s.serverRepo.GetActiveServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active servers: %w", err)
	}

	today := time.Now().Truncate(24 * time.Hour)

	const maxConcurrent = 10
	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, server := range servers {
		wg.Add(1)
		go func(srv *models.JSONRPCServer) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := s.checkSingleServer(ctx, srv, today); err != nil {
				s.logger.WithError(err).WithField("server_id", srv.ID).Error("Failed to check server")
			}
		}(server)
	}

	wg.Wait()

	// Update overall scores
	if err := s.serverRepo.UpdateAllScores(ctx); err != nil {
		s.logger.WithError(err).Error("Failed to update scores")
	}

	return nil
}

// checkSingleServer checks a single server's health
func (s *JSONRPCMonitorService) checkSingleServer(ctx context.Context, server *models.JSONRPCServer, date time.Time) error {
	exists, err := s.statusRepo.HasStatusForDate(ctx, server.ID, date)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// Perform JSON-RPC health check
	result := s.ValidateJSONRPCEndpoint(ctx, server.Address)

	status := &models.JSONRPCDailyStatus{
		ServerID:         server.ID,
		Date:             date,
		Color:            0,
		Attempts:         result.Attempts,
		Success:          result.Success,
		ResponseTimeMs:   result.ResponseTimeMs,
		ErrorMsg:         result.ErrorMsg,
		BlockchainHeight: result.BlockHeight,
	}

	if result.Success {
		status.Color = 1
	}

	return s.statusRepo.CreateStatus(ctx, status)
}

// JSONRPCCheckResult holds the result of a JSON-RPC endpoint check
type JSONRPCCheckResult struct {
	Success        bool
	Attempts       int
	ResponseTimeMs int
	BlockHeight    int64
	ErrorMsg       string
}

// ValidateJSONRPCEndpoint checks if a JSON-RPC endpoint is responding correctly
func (s *JSONRPCMonitorService) ValidateJSONRPCEndpoint(ctx context.Context, address string) *JSONRPCCheckResult {
	result := &JSONRPCCheckResult{Attempts: 5}

	for i := 0; i < 5; i++ {
		start := time.Now()

		// Call getBlockchainInfo method (Pactus JSON-RPC)
		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "pactus.blockchain.get_blockchain_info",
			"params":  map[string]interface{}{},
			"id":      1,
		}

		body, _ := json.Marshal(request)
		req, err := http.NewRequestWithContext(ctx, "POST", address, bytes.NewReader(body))
		if err != nil {
			result.ErrorMsg = err.Error()
			time.Sleep(time.Second)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			result.ErrorMsg = err.Error()
			time.Sleep(time.Second)
			continue
		}

		responseBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 200 {
			result.Success = true
			result.ResponseTimeMs = int(time.Since(start).Milliseconds())

			// Parse response for block height
			var response map[string]interface{}
			if json.Unmarshal(responseBody, &response) == nil {
				if r, ok := response["result"].(map[string]interface{}); ok {
					if height, ok := r["last_block_height"].(float64); ok {
						result.BlockHeight = int64(height)
					}
				}
			}
			break
		}

		result.ErrorMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(responseBody))
		time.Sleep(time.Second)
	}

	return result
}

// GetServersWithStatus returns all servers with their 30-day status
func (s *JSONRPCMonitorService) GetServersWithStatus(ctx context.Context, network string) ([]*models.JSONRPCServerResponse, error) {
	var servers []*models.JSONRPCServer
	var err error

	if network != "" {
		servers, err = s.serverRepo.GetServersByNetwork(ctx, network)
	} else {
		servers, err = s.serverRepo.GetActiveServers(ctx)
	}

	if err != nil {
		return nil, err
	}

	var response []*models.JSONRPCServerResponse
	for _, server := range servers {
		statuses, err := s.statusRepo.GetRecentStatusesByServer(ctx, server.ID, 30)
		if err != nil {
			s.logger.WithError(err).WithField("server_id", server.ID).Error("Failed to get statuses")
			continue
		}

		response = append(response, &models.JSONRPCServerResponse{
			ID:           server.ID,
			Name:         server.Name,
			Address:      server.Address,
			Network:      server.Network,
			Email:        server.Email,
			Website:      server.Website,
			Country:      server.Country,
			City:         server.City,
			Latitude:     server.Latitude,
			Longitude:    server.Longitude,
			Status:       statuses,
			OverallScore: server.OverallScore,
		})
	}

	return response, nil
}

// GetServerCount returns the count of active JSON-RPC servers
func (s *JSONRPCMonitorService) GetServerCount(ctx context.Context) (int, error) {
	return s.serverRepo.GetServerCount(ctx, true)
}

// UpdateServerGeoLocations updates geographic data for all servers
func (s *JSONRPCMonitorService) UpdateServerGeoLocations(ctx context.Context) error {
	servers, err := s.serverRepo.GetActiveServers(ctx)
	if err != nil {
		return err
	}

	// Use concurrency to speed up updates
	// Note: basic ip-api.com free tier has 45 req/min rate limit.
	// We use a small concurrency limit to avoid overwhelming it immediately,
	// but if many updates are needed, we might still hit limits.
	const maxConcurrent = 5
	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, server := range servers {
		// Skip if already has geo data
		if server.Country != "" {
			continue
		}

		wg.Add(1)
		go func(srv *models.JSONRPCServer) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ip := s.geoService.ExtractIPFromAddress(srv.Address)
			if ip == "" {
				s.logger.WithField("address", srv.Address).Debug("Could not extract IP from address")
				return
			}

			// Check context before making request
			select {
			case <-ctx.Done():
				return
			default:
			}

			geo, err := s.geoService.GetLocation(ctx, ip)
			if err != nil {
				s.logger.WithError(err).WithField("server_id", srv.ID).Warn("Failed to get geo location")
				return
			}

			if err := s.serverRepo.UpdateServerGeo(ctx, srv.ID, geo); err != nil {
				s.logger.WithError(err).WithField("server_id", srv.ID).Error("Failed to update geo data")
			}
		}(server)
	}

	wg.Wait()
	return nil
}
