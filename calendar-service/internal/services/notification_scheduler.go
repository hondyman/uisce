package services

import (
	"time"

	"github.com/sirupsen/logrus"

	"calendar-service/internal/hasura"
)

type NotificationScheduler struct {
	hasuraClient *hasura.Client
	notifier     interface{}
	logger       *logrus.Entry
	ticker       *time.Ticker
	quit         chan struct{}
}

func NewNotificationScheduler(client *hasura.Client, notifier interface{}, logger *logrus.Entry) *NotificationScheduler {
	return &NotificationScheduler{
		hasuraClient: client,
		notifier:     notifier,
		logger:       logger.WithField("component", "notification_scheduler"),
		quit:         make(chan struct{}),
	}
}

// Start starts the scheduler loop. In production this would use cron or a dedicated task queue.
func (s *NotificationScheduler) Start(interval time.Duration) {
	s.ticker = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.runScheduledTasks()
			case <-s.quit:
				s.ticker.Stop()
				return
			}
		}
	}()
	s.logger.Info("Notification scheduler started")
}

func (s *NotificationScheduler) Stop() {
	if s.quit != nil {
		close(s.quit)
	}
	s.logger.Info("Notification scheduler stopped")
}

func (s *NotificationScheduler) runScheduledTasks() {
	// For demonstration, processing "weekly digests".
	// In a complete implementation, we would check if it's the start of the week.

	// In a real scenario, this would perform a Hasura query to get all users where digest_frequency = 'weekly'
	s.logger.Debug("Running scheduled notification tasks")
}
