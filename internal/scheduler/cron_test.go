package scheduler

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewCronScheduler(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	scheduler := NewCronScheduler(nil, nil, logger)

	if scheduler == nil {
		t.Fatal("Expected non-nil scheduler")
	}

	if scheduler.jobTimeout != 30*time.Minute {
		t.Errorf("Expected job timeout of 30 minutes, got %v", scheduler.jobTimeout)
	}

	if scheduler.cron == nil {
		t.Error("Expected non-nil cron instance")
	}
}

func TestCronScheduler_GetSchedulerStatus(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	scheduler := NewCronScheduler(nil, nil, logger)
	status := scheduler.GetSchedulerStatus()

	if status == nil {
		t.Fatal("Expected non-nil status")
	}

	if _, ok := status["running"]; !ok {
		t.Error("Expected 'running' key in status")
	}

	if _, ok := status["job_count"]; !ok {
		t.Error("Expected 'job_count' key in status")
	}

	if _, ok := status["jobs"]; !ok {
		t.Error("Expected 'jobs' key in status")
	}
}
