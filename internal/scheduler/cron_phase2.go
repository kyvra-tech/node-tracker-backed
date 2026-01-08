package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/services"
)

// CronSchedulerPhase2 extends CronScheduler with Phase 2 functionality
type CronSchedulerPhase2 struct {
	cron              *cron.Cron
	bootstrapMonitor  *services.BootstrapMonitor
	grpcMonitor       *services.GRPCMonitor
	jsonrpcMonitor    *services.JSONRPCMonitorService
	networkStats      *services.NetworkStatsService
	geoService        *services.GeoLocationService
	logger            *logrus.Logger
	jobTimeout        time.Duration
	activeJobs        sync.WaitGroup
	shutdownCtx       context.Context
	shutdownCancel    context.CancelFunc
}

// NewCronSchedulerPhase2 creates a new Phase 2 scheduler
func NewCronSchedulerPhase2(
	bootstrapMonitor *services.BootstrapMonitor,
	grpcMonitor *services.GRPCMonitor,
	jsonrpcMonitor *services.JSONRPCMonitorService,
	networkStats *services.NetworkStatsService,
	geoService *services.GeoLocationService,
	logger *logrus.Logger,
) *CronSchedulerPhase2 {
	ctx, cancel := context.WithCancel(context.Background())

	return &CronSchedulerPhase2{
		cron:             cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger))),
		bootstrapMonitor: bootstrapMonitor,
		grpcMonitor:      grpcMonitor,
		jsonrpcMonitor:   jsonrpcMonitor,
		networkStats:     networkStats,
		geoService:       geoService,
		logger:           logger,
		jobTimeout:       30 * time.Minute,
		shutdownCtx:      ctx,
		shutdownCancel:   cancel,
	}
}

func (s *CronSchedulerPhase2) Start() {
	// ============ PHASE 1 JOBS ============

	// Schedule daily gRPC server checks at 2 AM UTC
	_, err := s.cron.AddFunc("0 2 * * *", s.createJobWrapper("gRPC Health Check", func(ctx context.Context) error {
		return s.grpcMonitor.CheckAllServers(ctx)
	}))
	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule gRPC server checks")
	}

	// Schedule gRPC server sync every 6 hours
	_, err = s.cron.AddFunc("30 */6 * * *", s.createJobWrapper("gRPC Sync", func(ctx context.Context) error {
		return s.grpcMonitor.SyncGRPCServers(ctx)
	}))
	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule gRPC sync")
	}

	// Schedule daily bootstrap node checks at 1 AM UTC
	_, err = s.cron.AddFunc("0 1 * * *", s.createJobWrapper("Bootstrap Health Check", func(ctx context.Context) error {
		return s.bootstrapMonitor.CheckAllNodes(ctx)
	}))
	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule bootstrap node checks")
	}

	// Schedule bootstrap node sync every 6 hours
	_, err = s.cron.AddFunc("0 */6 * * *", s.createJobWrapper("Bootstrap Sync", func(ctx context.Context) error {
		return s.bootstrapMonitor.SyncBootstrapNodes(ctx)
	}))
	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule bootstrap sync")
	}

	// ============ PHASE 2 JOBS ============

	// Schedule daily JSON-RPC server checks at 3 AM UTC
	_, err = s.cron.AddFunc("0 3 * * *", s.createJobWrapper("JSON-RPC Health Check", func(ctx context.Context) error {
		return s.jsonrpcMonitor.CheckAllServers(ctx)
	}))
	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule JSON-RPC server checks")
	}

	// Schedule geo location updates every 12 hours
	_, err = s.cron.AddFunc("0 */12 * * *", s.createJobWrapper("Geo Location Update", func(ctx context.Context) error {
		return s.jsonrpcMonitor.UpdateServerGeoLocations(ctx)
	}))
	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule geo location updates")
	}

	// Schedule network snapshots every 6 hours
	_, err = s.cron.AddFunc("0 */6 * * *", s.createJobWrapper("Network Snapshot", func(ctx context.Context) error {
		return s.networkStats.CreateSnapshot(ctx)
	}))
	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule network snapshots")
	}

	s.cron.Start()
	s.logger.Info("Phase 2 Cron scheduler started successfully")

	// Log scheduled jobs
	entries := s.cron.Entries()
	s.logger.WithField("job_count", len(entries)).Info("Scheduled jobs:")
	for i, entry := range entries {
		s.logger.WithFields(logrus.Fields{
			"job_index": i,
			"next_run":  entry.Next,
		}).Debug("Job scheduled")
	}
}

// createJobWrapper wraps a job with context, timeout, logging, and panic recovery
func (s *CronSchedulerPhase2) createJobWrapper(jobName string, jobFunc func(context.Context) error) func() {
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

func (s *CronSchedulerPhase2) Stop() {
	s.logger.Info("Stopping Phase 2 cron scheduler...")

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
func (s *CronSchedulerPhase2) GetSchedulerStatus() map[string]interface{} {
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
