package services

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type GRPCServerService struct {
	logger   *logrus.Logger
	filePath string
}

type ServersConfig struct {
	Mainnet []string `json:"mainnet"`
	Testnet []string `json:"testnet"`
}

func NewGRPCServerService(logger *logrus.Logger, filePath string) *GRPCServerService {
	return &GRPCServerService{
		logger:   logger,
		filePath: filePath,
	}
}

// LoadGRPCServers reads gRPC servers from servers.json
func (gs *GRPCServerService) LoadGRPCServers() (*ServersConfig, error) {
	data, err := os.ReadFile(gs.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config ServersConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	gs.logger.WithFields(logrus.Fields{
		"mainnet_count": len(config.Mainnet),
		"testnet_count": len(config.Testnet),
	}).Info("Successfully loaded gRPC servers")

	return &config, nil
}
