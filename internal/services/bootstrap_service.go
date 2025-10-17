package services

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type BootstrapService struct {
	logger   *logrus.Logger
	filePath string
}

type BootstrapNode struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Website string `json:"website"`
	Address string `json:"address"`
}

// NewBootstrapService creates a new bootstrap service that reads from a file
func NewBootstrapService(logger *logrus.Logger, filePath string) *BootstrapService {
	return &BootstrapService{
		logger:   logger,
		filePath: filePath,
	}
}

// LoadBootstrapNodes reads bootstrap nodes from a local JSON file
func (bs *BootstrapService) LoadBootstrapNodes() ([]*BootstrapNode, error) {
	// Read the file
	data, err := os.ReadFile(bs.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON into slice of BootstrapNode
	var nodes []*BootstrapNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate the nodes
	if err := bs.validateNodes(nodes); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	bs.logger.WithField("count", len(nodes)).Info("Successfully loaded bootstrap nodes")
	return nodes, nil
}

// validateNodes checks if nodes are valid
func (bs *BootstrapService) validateNodes(nodes []*BootstrapNode) error {
	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found")
	}

	for i, node := range nodes {
		if node.Address == "" {
			return fmt.Errorf("node %d has empty address", i)
		}
		if node.Name == "" {
			return fmt.Errorf("node %d has empty name", i)
		}
	}

	return nil
}
