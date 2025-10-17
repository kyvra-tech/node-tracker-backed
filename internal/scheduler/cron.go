package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/services"
)

type CronScheduler struct {
	cron           *cron.Cron
	monitor        *services.BootstrapMonitor
	grpcMonitor    *services.GRPCMonitor
	logger         *logrus.Logger
	jobTimeout     time.Duration
	activeJobs     sync.WaitGroup
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

func NewCronScheduler(
	monitor *services.BootstrapMonitor,
	grpcMonitor *services.GRPCMonitor,
	logger *logrus.Logger,
) *CronScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &CronScheduler{
		cron:           cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger))),
		monitor:        monitor,
		grpcMonitor:    grpcMonitor,
		logger:         logger,
		jobTimeout:     30 * time.Minute, // Configurable timeout for jobs
		shutdownCtx:    ctx,
		shutdownCancel: cancel,
	}
}

func (s *CronScheduler) Start() {
	// Schedule daily gRPC server checks at 7 AM UTC
	_, err := s.cron.AddFunc("0 7 * * *", s.createJobWrapper("gRPC Health Check", func(ctx context.Context) error {
		return s.grpcMonitor.CheckAllServers(ctx)
	}))
	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule gRPC server checks")
	}

	// Schedule gRPC server sync every 6 hours
	_, err = s.cron.AddFunc("30 */6 * * *", s.createJobWrapper("gRPC Sync", func(ctx context.Context) error {
		return s.grpcMonitor.SyncGRPCServersFromFile(ctx)
	}))
	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule gRPC sync")
	}

	// Schedule daily bootstrap node checks at 6 AM UTC
	_, err = s.cron.AddFunc("0 6 * * *", s.createJobWrapper("Bootstrap Health Check", func(ctx context.Context) error {
		return s.monitor.CheckAllNodes(ctx)
	}))
	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule bootstrap node checks")
	}

	// Schedule bootstrap node sync every 6 hours
	_, err = s.cron.AddFunc("0 */6 * * *", s.createJobWrapper("Bootstrap Sync", func(ctx context.Context) error {
		return s.monitor.SyncBootstrapNodesFromFile(ctx)
	}))
	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule bootstrap sync")
	}

	s.cron.Start()
	s.logger.Info("Cron scheduler started successfully")
}

// createJobWrapper wraps a job with context, timeout, logging, and panic recovery
func (s *CronScheduler) createJobWrapper(jobName string, jobFunc func(context.Context) error) func() {
	return func() {
		s.activeJobs.Add(1)
		defer s.activeJobs.Done()

		// Create context with timeout
		ctx, cancel := context.WithTimeout(s.shutdownCtx, s.jobTimeout)
		defer cancel()

		// Track job execution time
		startTime := time.Now()

		s.logger.WithFields(logrus.Fields{
			"job":       jobName,
			"timestamp": startTime.UTC(),
		}).Info("Starting scheduled job")

		// Panic recovery
		defer func() {
			if r := recover(); r != nil {
				s.logger.WithFields(logrus.Fields{
					"job":   jobName,
					"panic": r,
				}).Error("Job panicked")
			}
		}()

		// Execute job
		err := jobFunc(ctx)

		duration := time.Since(startTime)

		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"job":      jobName,
				"duration": duration.String(),
				"error":    err.Error(),
			}).Error("Job failed")
		} else {
			s.logger.WithFields(logrus.Fields{
				"job":      jobName,
				"duration": duration.String(),
			}).Info("Job completed successfully")
		}

		// Check if context was cancelled
		if ctx.Err() == context.DeadlineExceeded {
			s.logger.WithFields(logrus.Fields{
				"job":     jobName,
				"timeout": s.jobTimeout.String(),
			}).Warn("Job timed out")
		}
	}
}

func (s *CronScheduler) Stop() {
	s.logger.Info("Stopping cron scheduler...")

	// Stop accepting new jobs
	ctx := s.cron.Stop()

	// Cancel all running jobs
	s.shutdownCancel()

	// Wait for running jobs to complete (with timeout)
	done := make(chan struct{})
	go func() {
		s.activeJobs.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("All jobs completed, cron scheduler stopped")
	case <-ctx.Done():
		s.logger.Info("Cron scheduler stopped")
	case <-time.After(1 * time.Minute):
		s.logger.Warn("Timeout waiting for jobs to complete, forcing shutdown")
	}
}

// GetSchedulerStatus returns the current status of the scheduler
func (s *CronScheduler) GetSchedulerStatus() map[string]interface{} {
	entries := s.cron.Entries()

	jobs := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		jobs = append(jobs, map[string]interface{}{
			"next_run": entry.Next,
			"prev_run": entry.Prev,
		})
	}

	return map[string]interface{}{
		"running":   len(entries) > 0,
		"job_count": len(entries),
		"jobs":      jobs,
	}
}
