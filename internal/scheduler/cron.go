package scheduler

import (
	"context"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/services"
)

type CronScheduler struct {
	cron        *cron.Cron
	monitor     *services.BootstrapMonitor
	grpcMonitor *services.GRPCMonitor
	logger      *logrus.Logger
}

func NewCronScheduler(monitor *services.BootstrapMonitor, grpcMonitor *services.GRPCMonitor, logger *logrus.Logger) *CronScheduler {
	return &CronScheduler{
		cron:        cron.New(),
		monitor:     monitor,
		grpcMonitor: grpcMonitor,
		logger:      logger,
	}
}

func (s *CronScheduler) Start() {

	// Schedule daily gRPC server checks at 7 AM UTC
	_, err := s.cron.AddFunc("0 7 * * *", func() {
		s.logger.Info("Starting scheduled gRPC server health check")

		ctx := context.Background()
		if err := s.grpcMonitor.CheckAllServers(ctx); err != nil {
			s.logger.WithError(err).Error("Failed to check gRPC servers")
		} else {
			s.logger.Info("Completed scheduled gRPC server health check")
		}
	})

	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule gRPC server checks")
	}

	// Schedule gRPC server sync every 6 hours
	_, err = s.cron.AddFunc("30 */6 * * *", func() {
		s.logger.Info("Starting gRPC server sync")
		if err := s.grpcMonitor.SyncGRPCServersFromFile(); err != nil {
			s.logger.WithError(err).Error("Failed to sync gRPC servers")
		} else {
			s.logger.Info("Completed gRPC server sync")
		}
	})

	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule gRPC sync")
	}

	// Schedule daily bootstrap node checks at 6 AM UTC
	_, err = s.cron.AddFunc("0 6 * * *", func() {
		s.logger.Info("Starting scheduled bootstrap node health check")

		ctx := context.Background()
		if err := s.monitor.CheckAllNodes(ctx); err != nil {
			s.logger.WithError(err).Error("Failed to check bootstrap nodes")
		} else {
			s.logger.Info("Completed scheduled bootstrap node health check")
		}
	})

	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule bootstrap node checks")
		return
	}

	_, err = s.cron.AddFunc("0 */6 * * *", func() {
		s.logger.Info("Starting GitHub bootstrap node sync")
		if err := s.monitor.SyncBootstrapNodesFromFile(); err != nil {
			s.logger.WithError(err).Error("Failed to sync bootstrap nodes from file")
		} else {
			s.logger.Info("Completed bootstrap node sync from file")
		}
	})

	if err != nil {
		s.logger.WithError(err).Error("Failed to schedule GitHub sync")
		return
	}

	s.cron.Start()
	s.logger.Info("Cron scheduler started")
}

func (s *CronScheduler) Stop() {
	s.cron.Stop()
	s.logger.Info("Cron scheduler stopped")
}
