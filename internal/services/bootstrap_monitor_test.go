package services

import (
	"testing"
)

func TestSyncStats(t *testing.T) {
	stats := &SyncStats{
		Added:       5,
		Updated:     3,
		Deactivated: 2,
		Errors:      0,
	}

	if stats.Added != 5 {
		t.Errorf("Expected 5 added, got %d", stats.Added)
	}

	if stats.Updated != 3 {
		t.Errorf("Expected 3 updated, got %d", stats.Updated)
	}

	if stats.Deactivated != 2 {
		t.Errorf("Expected 2 deactivated, got %d", stats.Deactivated)
	}

	if stats.Errors != 0 {
		t.Errorf("Expected 0 errors, got %d", stats.Errors)
	}
}
