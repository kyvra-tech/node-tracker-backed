package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/repositories"
)

type BootstrapMonitor struct {
	bootstrapRepo    repositories.BootstrapRepository
	statusRepo       repositories.StatusRepository
	nodeChecker      *NodeChecker
	bootstrapService *BootstrapService
	logger           *logrus.Logger
}

func NewBootstrapMonitor(
	bootstrapRepo repositories.BootstrapRepository,
	statusRepo repositories.StatusRepository,
	nodeChecker *NodeChecker,
	logger *logrus.Logger,
	bootstrapService *BootstrapService,
) *BootstrapMonitor {
	return &BootstrapMonitor{
		bootstrapRepo:    bootstrapRepo,
		statusRepo:       statusRepo,
		nodeChecker:      nodeChecker,
		bootstrapService: bootstrapService,
		logger:           logger,
	}
}

// CheckAllNodes performs health checks on all active nodes
func (bm *BootstrapMonitor) CheckAllNodes(ctx context.Context) error {
	nodes, err := bm.bootstrapRepo.GetActiveNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active nodes: %w", err)
	}

	today := time.Now().Truncate(24 * time.Hour)

	// Use concurrent processing with worker pool
	const maxConcurrent = 10 // Process 10 nodes at a time
	semaphore := make(chan struct{}, maxConcurrent)
	errChan := make(chan error, len(nodes))
	var wg sync.WaitGroup

	for _, node := range nodes {
		wg.Add(1)
		go func(n *models.BootstrapNode) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := bm.checkSingleNode(ctx, n, today); err != nil {
				bm.logger.WithError(err).WithField("node_id", n.ID).Error("Failed to check node")
				errChan <- err
			}
		}(node)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Collect errors (non-blocking)
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// Update overall scores after checking all nodes
	if err := bm.bootstrapRepo.UpdateAllScores(ctx); err != nil {
		bm.logger.WithError(err).Error("Failed to update overall scores")
	}

	if len(errors) > 0 {
		bm.logger.WithField("error_count", len(errors)).Warn("Some nodes failed during check")
	}

	return nil
}

// checkSingleNode checks a single node's health
func (bm *BootstrapMonitor) checkSingleNode(ctx context.Context, node *models.BootstrapNode, date time.Time) error {
	// Check if we already have a record for today
	exists, err := bm.statusRepo.HasStatusForDate(ctx, node.ID, date)
	if err != nil {
		return err
	}

	if exists {
		bm.logger.WithFields(logrus.Fields{
			"node_id": node.ID,
			"date":    date.Format("2006-01-02"),
		}).Info("Status already recorded for today")
		return nil
	}

	// Check the node
	result := bm.nodeChecker.CheckNode(ctx, node.Address)

	// Determine color based on success
	color := 0 // red/gray for failure
	if result.Success {
		color = 1 // green for success
	}

	// Save the result
	status := &models.DailyStatus{
		NodeID:   node.ID,
		Date:     date,
		Color:    color,
		Attempts: result.Attempts,
		Success:  result.Success,
		ErrorMsg: result.ErrorMsg,
	}

	return bm.statusRepo.CreateStatus(ctx, status)
}

// GetBootstrapNodesWithStatus retrieves all active nodes with their recent status history
func (bm *BootstrapMonitor) GetBootstrapNodesWithStatus(ctx context.Context) ([]*models.BootstrapNodeResponse, error) {
	nodes, err := bm.bootstrapRepo.GetActiveNodes(ctx)
	if err != nil {
		return nil, err
	}

	var response []*models.BootstrapNodeResponse

	for _, node := range nodes {
		statuses, err := bm.statusRepo.GetRecentStatusesByNode(ctx, node.ID, 30) // Last 30 days
		if err != nil {
			bm.logger.WithError(err).WithField("node_id", node.ID).Error("Failed to get statuses")
			continue
		}

		nodeResponse := &models.BootstrapNodeResponse{
			Name:         node.Name,
			Email:        node.Email,
			Website:      node.Website,
			Address:      node.Address,
			Status:       statuses,
			OverallScore: node.OverallScore,
		}

		response = append(response, nodeResponse)
	}

	return response, nil
}

// SyncBootstrapNodesFromFile synchronizes nodes from the local JSON file
func (bm *BootstrapMonitor) SyncBootstrapNodesFromFile(ctx context.Context) error {
	bm.logger.Info("Starting bootstrap node sync from local file")

	// Load bootstrap nodes from local file
	githubNodes, err := bm.bootstrapService.LoadBootstrapNodes()
	if err != nil {
		return fmt.Errorf("failed to load bootstrap nodes: %w", err)
	}

	// Get current nodes from database
	currentNodes, err := bm.bootstrapRepo.GetAllNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current nodes: %w", err)
	}

	// Create maps for efficient lookup
	currentNodesMap := make(map[string]*models.BootstrapNode)
	for _, node := range currentNodes {
		currentNodesMap[node.Address] = node
	}

	githubNodesMap := make(map[string]*BootstrapNode)
	for _, node := range githubNodes {
		githubNodesMap[node.Address] = node
	}

	// Process changes
	stats := &SyncStats{}

	// Add new nodes and update existing ones
	for _, githubNode := range githubNodes {
		if existingNode, exists := currentNodesMap[githubNode.Address]; exists {
			// Update existing node if needed
			if bm.shouldUpdateNode(existingNode, githubNode) {
				updatedNode := &models.BootstrapNode{
					Name:    githubNode.Name,
					Email:   githubNode.Email,
					Website: githubNode.Website,
					Address: githubNode.Address,
				}
				if err := bm.bootstrapRepo.UpdateNode(ctx, updatedNode); err != nil {
					bm.logger.WithError(err).WithField("address", githubNode.Address).Error("Failed to update node")
					stats.Errors++
					continue
				}
				stats.Updated++
			}
		} else {
			// Add new node
			newNode := &models.BootstrapNode{
				Name:     githubNode.Name,
				Email:    githubNode.Email,
				Website:  githubNode.Website,
				Address:  githubNode.Address,
				IsActive: true,
			}
			if err := bm.bootstrapRepo.CreateNode(ctx, newNode); err != nil {
				bm.logger.WithError(err).WithField("address", githubNode.Address).Error("Failed to add node")
				stats.Errors++
				continue
			}
			stats.Added++
		}
	}

	// Deactivate nodes that are no longer in the file
	var nodesToDeactivate []string
	for _, node := range currentNodes {
		if _, exists := githubNodesMap[node.Address]; !exists && node.IsActive {
			nodesToDeactivate = append(nodesToDeactivate, node.Address)
		}
	}

	if len(nodesToDeactivate) > 0 {
		if err := bm.bootstrapRepo.DeactivateNodes(ctx, nodesToDeactivate); err != nil {
			bm.logger.WithError(err).Error("Failed to deactivate removed nodes")
		} else {
			stats.Deactivated = len(nodesToDeactivate)
		}
	}

	bm.logger.WithFields(logrus.Fields{
		"added":       stats.Added,
		"updated":     stats.Updated,
		"deactivated": stats.Deactivated,
		"errors":      stats.Errors,
	}).Info("Completed bootstrap node sync")

	return nil
}

// GetBootstrapNodeCount returns the count of active bootstrap nodes
func (bm *BootstrapMonitor) GetBootstrapNodeCount(ctx context.Context) (int, error) {
	return bm.bootstrapRepo.GetNodeCount(ctx, true)
}

// Helper types and functions

type SyncStats struct {
	Added       int
	Updated     int
	Deactivated int
	Errors      int
}

func (bm *BootstrapMonitor) shouldUpdateNode(existing *models.BootstrapNode, github *BootstrapNode) bool {
	return existing.Name != github.Name ||
		existing.Email != github.Email ||
		existing.Website != github.Website
}
