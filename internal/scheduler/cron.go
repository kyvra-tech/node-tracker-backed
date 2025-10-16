package scheduler

import (
	"context"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/services"
)

type CronScheduler struct {
	cron    *cron.Cron
	monitor *services.BootstrapMonitor
	logger  *logrus.Logger
}

func NewCronScheduler(monitor *services.BootstrapMonitor, logger *logrus.Logger) *CronScheduler {
	return &CronScheduler{
		cron:    cron.New(),
		monitor: monitor,
		logger:  logger,
	}
}

func (s *CronScheduler) Start() {
	// Schedule daily bootstrap node checks at 6 AM UTC
	_, err := s.cron.AddFunc("0 6 * * *", func() {
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

	s.cron.Start()
	s.logger.Info("Cron scheduler started")
}

func (s *CronScheduler) Stop() {
	s.cron.Stop()
	s.logger.Info("Cron scheduler stopped")
}
