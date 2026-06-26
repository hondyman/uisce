package services

import (
	"calendar-service/internal/sync"
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// EventListener listens for internal event changes and pushes to connected integrators
type EventListener struct {
	syncProcessor   *sync.GoogleSyncProcessor
	msSyncProcessor *sync.MicrosoftSyncProcessor
	logger          *logrus.Entry
}

// EventListenerConfig holds configuration for EventListener
type EventListenerConfig struct {
	SyncProcessor   *sync.GoogleSyncProcessor
	MsSyncProcessor *sync.MicrosoftSyncProcessor
	Logger          *logrus.Entry
}

// NewEventListener creates a new event listener
func NewEventListener(cfg EventListenerConfig) *EventListener {
	return &EventListener{
		syncProcessor:   cfg.SyncProcessor,
		msSyncProcessor: cfg.MsSyncProcessor,
		logger:          cfg.Logger.WithField("component", "event_listener"),
	}
}

// OnEventCreated is called when a new event is created internally
func (el *EventListener) OnEventCreated(ctx context.Context, userID, eventID string) {
	el.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"event_id": eventID,
	}).Info("Event created, pushing to external calendars")

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if el.syncProcessor != nil {
			err := el.withRetry(bgCtx, "Google Push", func() error {
				return el.syncProcessor.PushEvent(bgCtx, userID, eventID)
			})
			if err != nil {
				el.logger.WithError(err).Error("Final failure pushing to Google")
			}
		}

		if el.msSyncProcessor != nil {
			err := el.withRetry(bgCtx, "Microsoft Push", func() error {
				return el.msSyncProcessor.PushEvent(bgCtx, userID, eventID)
			})
			if err != nil {
				el.logger.WithError(err).Error("Final failure pushing to Microsoft")
			}
		}
	}()
}

// OnEventUpdated is called when an event is updated internally
func (el *EventListener) OnEventUpdated(ctx context.Context, userID, eventID string) {
	el.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"event_id": eventID,
	}).Info("Event updated, pushing to external calendars")

	el.OnEventCreated(ctx, userID, eventID) // Same logic for update
}

// OnEventDeleted is called when an event is deleted internally
func (el *EventListener) OnEventDeleted(ctx context.Context, userID, eventID string) {
	el.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"event_id": eventID,
	}).Info("Event deleted, removing from external calendars")

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if el.syncProcessor != nil {
			err := el.withRetry(bgCtx, "Google Delete", func() error {
				return el.syncProcessor.DeleteEventFromGoogle(bgCtx, userID, eventID)
			})
			if err != nil {
				el.logger.WithError(err).Error("Final failure deleting from Google")
			}
		}

		if el.msSyncProcessor != nil {
			err := el.withRetry(bgCtx, "Microsoft Delete", func() error {
				return el.msSyncProcessor.DeleteEventFromMicrosoft(bgCtx, userID, eventID)
			})
			if err != nil {
				el.logger.WithError(err).Error("Final failure deleting from Microsoft")
			}
		}
	}()
}

// OnBatchEventsCreated is called when multiple events are created (e.g., import)
func (el *EventListener) OnBatchEventsCreated(ctx context.Context, userID string, eventIDs []string) {
	el.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"event_count": len(eventIDs),
	}).Info("Batch events created, pushing to external calendars")

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		successCount := 0
		errorCount := 0

		for _, eventID := range eventIDs {
			failed := false
			if el.syncProcessor != nil {
				if err := el.syncProcessor.PushEvent(bgCtx, userID, eventID); err != nil {
					el.logger.WithError(err).WithField("event_id", eventID).Error("Failed to push event to Google")
					failed = true
				}
			}

			if el.msSyncProcessor != nil {
				if err := el.msSyncProcessor.PushEvent(bgCtx, userID, eventID); err != nil {
					el.logger.WithError(err).WithField("event_id", eventID).Error("Failed to push event to Microsoft")
					failed = true
				}
			}

			if failed {
				errorCount++
			} else {
				successCount++
			}
		}

		el.logger.WithFields(logrus.Fields{
			"success": successCount,
			"errors":  errorCount,
		}).Info("Batch push to external calendars completed")
	}()
}

func (el *EventListener) withRetry(ctx context.Context, opName string, fn func() error) error {
	maxRetries := 3
	backoff := 2 * time.Second

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err
			el.logger.WithError(err).WithFields(logrus.Fields{
				"operation": opName,
				"attempt":   i + 1,
			}).Warn("Operation failed, retrying...")

			select {
			case <-time.After(backoff):
				backoff *= 2
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}
		return nil
	}
	return lastErr
}
