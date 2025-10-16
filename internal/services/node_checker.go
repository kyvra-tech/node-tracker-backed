package services

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type NodeChecker struct {
	timeout    time.Duration
	maxRetries int
	logger     *logrus.Logger
}

func NewNodeChecker(timeout time.Duration, maxRetries int, logger *logrus.Logger) *NodeChecker {
	return &NodeChecker{
		timeout:    timeout,
		maxRetries: maxRetries,
		logger:     logger,
	}
}

type CheckResult struct {
	Success  bool
	Attempts int
	ErrorMsg string
	Duration time.Duration
}

func (nc *NodeChecker) CheckNode(ctx context.Context, address string) *CheckResult {
	result := &CheckResult{}

	host, port, err := nc.parseAddress(address)
	if err != nil {
		result.ErrorMsg = fmt.Sprintf("failed to parse address: %v", err)
		return result
	}

	start := time.Now()

	for attempt := 1; attempt <= nc.maxRetries; attempt++ {
		result.Attempts = attempt

		if nc.attemptConnection(ctx, host, port) {
			result.Success = true
			result.Duration = time.Since(start)
			nc.logger.WithFields(logrus.Fields{
				"address":  address,
				"attempts": attempt,
				"duration": result.Duration,
			}).Info("Node connection successful")
			return result
		}

		if attempt < nc.maxRetries {
			time.Sleep(time.Second * 2) // Wait between retries
		}
	}

	result.Duration = time.Since(start)
	result.ErrorMsg = fmt.Sprintf("failed to connect after %d attempts", nc.maxRetries)

	nc.logger.WithFields(logrus.Fields{
		"address":  address,
		"attempts": result.Attempts,
		"duration": result.Duration,
	}).Warn("Node connection failed")

	return result
}

func (nc *NodeChecker) parseAddress(address string) (string, string, error) {
	// Parse different address formats:
	// /dns/bootstrap1.pactus.org/tcp/21888/p2p/...
	// /ip4/65.108.211.187/tcp/21888/p2p/...

	parts := strings.Split(address, "/")
	if len(parts) < 5 {
		return "", "", fmt.Errorf("invalid address format")
	}

	var host, port string

	for i := 0; i < len(parts)-1; i++ {
		switch parts[i] {
		case "dns", "ip4", "ip6":
			if i+1 < len(parts) {
				host = parts[i+1]
			}
		case "tcp":
			if i+1 < len(parts) {
				port = parts[i+1]
			}
		}
	}

	if host == "" || port == "" {
		return "", "", fmt.Errorf("could not extract host and port from address")
	}

	return host, port, nil
}

func (nc *NodeChecker) attemptConnection(ctx context.Context, host, port string) bool {
	ctx, cancel := context.WithTimeout(ctx, nc.timeout)
	defer cancel()

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}
