package utils

import (
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron *cron.Cron
}

// NewScheduler creates a new instance of the scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		cron: cron.New(), // Removed WithSeconds()
	}
}

// AddJob adds a new job to the scheduler.
func (s *Scheduler) AddJob(schedule string, job func()) error {
	_, err := s.cron.AddFunc(schedule, job)
	if err != nil {
		return err
	}
	return nil
}

// Start starts the scheduler.
func (s *Scheduler) Start() {
	s.cron.Start()
	getLogger().Info("Scheduler started")
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.cron.Stop()
	getLogger().Info("Scheduler stopped")
}
