package services

import (
	"context"
	"fmt"
	"time"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/repositories"
	"github.com/sirupsen/logrus"
)

// RegistrationService handles public node registration
type RegistrationService struct {
	registrationRepo repositories.RegistrationRepository
	grpcRepo         repositories.GRPCRepository
	jsonrpcRepo      repositories.JSONRPCServerRepository
	grpcChecker      *GRPCChecker
	jsonrpcMonitor   *JSONRPCMonitorService
	geoService       *GeoLocationService
	logger           *logrus.Logger
}

// NewRegistrationService creates a new registration service
func NewRegistrationService(
	registrationRepo repositories.RegistrationRepository,
	grpcRepo repositories.GRPCRepository,
	jsonrpcRepo repositories.JSONRPCServerRepository,
	grpcChecker *GRPCChecker,
	jsonrpcMonitor *JSONRPCMonitorService,
	geoService *GeoLocationService,
	logger *logrus.Logger,
) *RegistrationService {
	return &RegistrationService{
		registrationRepo: registrationRepo,
		grpcRepo:         grpcRepo,
		jsonrpcRepo:      jsonrpcRepo,
		grpcChecker:      grpcChecker,
		jsonrpcMonitor:   jsonrpcMonitor,
		geoService:       geoService,
		logger:           logger,
	}
}

// SubmitRegistration handles new node registration submission
func (s *RegistrationService) SubmitRegistration(ctx context.Context, req *models.RegistrationRequest) (*models.RegistrationResponse, error) {
	// Validate the node is reachable
	isReachable, err := s.validateNode(ctx, req.NodeType, req.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to validate node: %w", err)
	}
	if !isReachable {
		return nil, fmt.Errorf("node at %s is not reachable", req.Address)
	}

	// Check for duplicates in existing servers
	exists, err := s.checkDuplicate(ctx, req.NodeType, req.Address)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("a node with address %s is already registered", req.Address)
	}

	// Check for pending registration
	pendingExists, err := s.registrationRepo.ExistsByAddress(ctx, req.Address)
	if err != nil {
		return nil, err
	}
	if pendingExists {
		return nil, fmt.Errorf("a registration for address %s is already pending", req.Address)
	}

	// Create registration
	registration := &models.NodeRegistration{
		NodeType: req.NodeType,
		Name:     req.Name,
		Address:  req.Address,
		Network:  req.Network,
		Email:    req.Email,
		Website:  req.Website,
		Status:   "pending",
	}

	if err := s.registrationRepo.Create(ctx, registration); err != nil {
		return nil, fmt.Errorf("failed to create registration: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"type":    req.NodeType,
		"address": req.Address,
		"email":   req.Email,
	}).Info("New node registration submitted")

	return &models.RegistrationResponse{
		ID:      registration.ID,
		Status:  "pending",
		Message: "Your node registration has been submitted and is pending review.",
	}, nil
}

// validateNode checks if the node is reachable
func (s *RegistrationService) validateNode(ctx context.Context, nodeType, address string) (bool, error) {
	switch nodeType {
	case "grpc":
		result := s.grpcChecker.CheckGRPCServer(ctx, address)
		return result.Success, nil
	case "jsonrpc":
		result := s.jsonrpcMonitor.ValidateJSONRPCEndpoint(ctx, address)
		return result.Success, nil
	default:
		return false, fmt.Errorf("unknown node type: %s", nodeType)
	}
}

// checkDuplicate checks if the address already exists
func (s *RegistrationService) checkDuplicate(ctx context.Context, nodeType, address string) (bool, error) {
	switch nodeType {
	case "grpc":
		return s.grpcRepo.ServerExists(ctx, address)
	case "jsonrpc":
		return s.jsonrpcRepo.ExistsByAddress(ctx, address)
	}
	return false, nil
}

// ApproveRegistration approves a pending registration
func (s *RegistrationService) ApproveRegistration(ctx context.Context, id int, reviewedBy string) error {
	registration, err := s.registrationRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if registration == nil {
		return fmt.Errorf("registration not found: %d", id)
	}

	if registration.Status != "pending" {
		return fmt.Errorf("registration is not pending")
	}

	// Resolve geo location
	ip := s.geoService.ExtractIPFromAddress(registration.Address)
	var geo *models.GeoLocation
	if ip != "" {
		geo, _ = s.geoService.GetLocation(ctx, ip)
	}

	// Add to appropriate server table
	switch registration.NodeType {
	case "grpc":
		server := &models.GRPCServer{
			Name:     registration.Name,
			Address:  registration.Address,
			Network:  registration.Network,
			Email:    registration.Email,
			Website:  registration.Website,
			IsActive: true,
		}
		if geo != nil && geo.IsValid() {
			server.Country = geo.Country
			server.CountryCode = geo.CountryCode
			server.City = geo.City
			server.Latitude = geo.Latitude
			server.Longitude = geo.Longitude
		}
		if err := s.grpcRepo.CreateServer(ctx, server); err != nil {
			return err
		}
	case "jsonrpc":
		server := &models.JSONRPCServer{
			Name:     registration.Name,
			Address:  registration.Address,
			Network:  registration.Network,
			Email:    registration.Email,
			Website:  registration.Website,
			IsActive: true,
		}
		if geo != nil && geo.IsValid() {
			server.Country = geo.Country
			server.CountryCode = geo.CountryCode
			server.City = geo.City
			server.Latitude = geo.Latitude
			server.Longitude = geo.Longitude
		}
		if err := s.jsonrpcRepo.CreateServer(ctx, server); err != nil {
			return err
		}
	}

	// Update registration status
	now := time.Now()
	return s.registrationRepo.UpdateStatus(ctx, id, "approved", "", reviewedBy, &now)
}

// RejectRegistration rejects a pending registration
func (s *RegistrationService) RejectRegistration(ctx context.Context, id int, reason, reviewedBy string) error {
	registration, err := s.registrationRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if registration == nil {
		return fmt.Errorf("registration not found: %d", id)
	}

	if registration.Status != "pending" {
		return fmt.Errorf("registration is not pending")
	}

	now := time.Now()
	return s.registrationRepo.UpdateStatus(ctx, id, "rejected", reason, reviewedBy, &now)
}

// GetPendingRegistrations returns all pending registrations
func (s *RegistrationService) GetPendingRegistrations(ctx context.Context) ([]*models.NodeRegistration, error) {
	return s.registrationRepo.GetByStatus(ctx, "pending")
}

// GetRegistrationByID returns a registration by ID
func (s *RegistrationService) GetRegistrationByID(ctx context.Context, id int) (*models.NodeRegistration, error) {
	return s.registrationRepo.GetByID(ctx, id)
}
