package services

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNodeChecker_ParseAddress(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	nc := NewNodeChecker(5*time.Second, 3, logger)

	tests := []struct {
		name        string
		address     string
		expectHost  string
		expectPort  string
		expectError bool
	}{
		{
			name:        "Valid DNS address",
			address:     "/dns/bootstrap1.pactus.org/tcp/21888/p2p/12D3KooWPxG5TnY",
			expectHost:  "bootstrap1.pactus.org",
			expectPort:  "21888",
			expectError: false,
		},
		{
			name:        "Valid IP4 address",
			address:     "/ip4/65.108.211.187/tcp/21888/p2p/12D3KooWPxG5TnY",
			expectHost:  "65.108.211.187",
			expectPort:  "21888",
			expectError: false,
		},
		{
			name:        "Invalid address format",
			address:     "invalid-address",
			expectError: true,
		},
		{
			name:        "Empty address",
			address:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := nc.parseAddress(tt.address)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if host != tt.expectHost {
				t.Errorf("Expected host %s, got %s", tt.expectHost, host)
			}

			if port != tt.expectPort {
				t.Errorf("Expected port %s, got %s", tt.expectPort, port)
			}
		})
	}
}

func TestNodeChecker_CheckNode(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	nc := NewNodeChecker(2*time.Second, 2, logger)

	ctx := context.Background()

	t.Run("Invalid address format", func(t *testing.T) {
		result := nc.CheckNode(ctx, "invalid-address")

		if result.Success {
			t.Error("Expected failure for invalid address")
		}

		if result.ErrorMsg == "" {
			t.Error("Expected error message for invalid address")
		}
	})

	t.Run("Valid address format", func(t *testing.T) {
		// Use a valid address format (will likely fail to connect but should parse correctly)
		result := nc.CheckNode(ctx, "/ip4/127.0.0.1/tcp/99999/p2p/test")

		// We expect this to fail to connect, but the address should parse
		if result.Attempts == 0 {
			t.Error("Expected at least one connection attempt")
		}
	})
}
