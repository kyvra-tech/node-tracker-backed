package services

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pactus "github.com/pactus-project/pactus/www/grpc/gen/go"
)

type GRPCChecker struct {
	timeout    time.Duration
	maxRetries int
	logger     *logrus.Logger
}

func NewGRPCChecker(timeout time.Duration, maxRetries int, logger *logrus.Logger) *GRPCChecker {
	return &GRPCChecker{
		timeout:    timeout,
		maxRetries: maxRetries,
		logger:     logger,
	}
}

type GRPCCheckResult struct {
	Success        bool
	Attempts       int
	ErrorMsg       string
	ResponseTimeMs int
}

// CheckGRPCServer checks if a gRPC server is healthy using Ping API
func (gc *GRPCChecker) CheckGRPCServer(ctx context.Context, address string) *GRPCCheckResult {
	result := &GRPCCheckResult{}

	for attempt := 1; attempt <= gc.maxRetries; attempt++ {
		result.Attempts = attempt

		start := time.Now()
		success, err := gc.attemptGRPCPing(ctx, address)
		duration := time.Since(start)

		if success {
			result.Success = true
			result.ResponseTimeMs = int(duration.Milliseconds())
			gc.logger.WithFields(logrus.Fields{
				"address":  address,
				"attempts": attempt,
				"latency":  duration,
			}).Info("gRPC server ping successful")
			return result
		}

		result.ErrorMsg = err.Error()

		if attempt < gc.maxRetries {
			time.Sleep(time.Second * 2) // Wait between retries
		}
	}

	gc.logger.WithFields(logrus.Fields{
		"address":  address,
		"attempts": result.Attempts,
		"error":    result.ErrorMsg,
	}).Warn("gRPC server ping failed")

	return result
}

// attemptGRPCPing attempts to connect and call Ping API
func (gc *GRPCChecker) attemptGRPCPing(ctx context.Context, address string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, gc.timeout)
	defer cancel()

	// Create gRPC connection
	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return false, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Create Network client and call GetNetworkInfo (this acts as a ping)
	client := pactus.NewNetworkClient(conn)
	_, err = client.GetNetworkInfo(ctx, &pactus.GetNetworkInfoRequest{})
	if err != nil {
		return false, fmt.Errorf("ping failed: %w", err)
	}

	return true, nil
}
