package services

import (
	"context"
	"time"

	"calendar-service/internal/database"

	"github.com/sirupsen/logrus"
)

type NotificationScheduler struct {
	dbClient  *database.Client
	notifier interface{}
	logger   *logrus.Entry
	ticker   *time.Ticker
	quit     chan struct{}
}

func NewNotificationScheduler(db *database.Client, notifier interface{}, logger *logrus.Entry) *NotificationScheduler {
	return &NotificationScheduler{
		dbClient:  db,
		notifier: notifier,
		logger:   logger.WithField("component", "notification_scheduler"),
		quit:     make(chan struct{}),
	}
}

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
	ctx := context.Background()

	query := `
		SELECT user_id, email, digest_frequency
		FROM user_notification_settings
		WHERE digest_frequency = 'weekly'
	`

	rows, err := s.dbClient.Pool().Query(ctx, query)
	if err != nil {
		s.logger.WithError(err).Error("Failed to fetch users for notifications")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var userID, email, digestFreq string
		if err := rows.Scan(&userID, &email, &digestFreq); err != nil {
			continue
		}
		s.logger.WithField("user_id", userID).Debug("Processing weekly digest notification")
	}

	s.logger.Debug("Running scheduled notification tasks")
}
