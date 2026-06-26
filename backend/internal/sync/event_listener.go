package sync

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/sirupsen/logrus"
)

// GoogleSyncListener listens for internal event changes and pushes to Google
type GoogleSyncListener struct {
	processor *SyncProcessor
	logger    *logrus.Entry
	consumer  *services.EventConsumer
}

// NewGoogleSyncListener creates a new listener
func NewGoogleSyncListener(processor *SyncProcessor, logger *logrus.Entry, consumer *services.EventConsumer) *GoogleSyncListener {
	return &GoogleSyncListener{
		processor: processor,
		logger:    logger.WithField("component", "google_sync_listener"),
		consumer:  consumer,
	}
}

// Start starts listening for events
func (l *GoogleSyncListener) Start() {
	if l.consumer == nil {
		l.logger.Warn("Event consumer not configured, skipping listener start")
		return
	}

	// Subscribe to internal_event.*
	events, err := l.consumer.Subscribe("internal_event.#", nil)
	if err != nil {
		l.logger.WithError(err).Error("Failed to subscribe to internal events")
		return
	}

	go func() {
		for event := range events {
			l.handleEvent(event)
		}
	}()
	l.logger.Info("Started Google Sync Listener")
}

func (l *GoogleSyncListener) handleEvent(event *services.BOEvent) {
	ctx := context.Background()

	switch event.Type {
	case services.EventInternalEventCreated, services.EventInternalEventUpdated:
		var internalEvent models.InternalEvent

		// Unmarshal data back to struct.
		// Note: event.Data is interface{}, might need careful handling if it's map[string]interface{} after JSON unmarshal
		dataBytes, _ := json.Marshal(event.Data)
		if err := json.Unmarshal(dataBytes, &internalEvent); err != nil {
			l.logger.WithError(err).Error("Failed to unmarshal internal event data")
			return
		}

		// Ensure IDs are valid
		if internalEvent.ID == uuid.Nil {
			// Try to parse from EntityID if Data didn't have ID populated correctly (e.g. if passed as map)
			if id, err := uuid.Parse(event.EntityID); err == nil {
				internalEvent.ID = id
			}
		}

		// Record metric
		internalEventsReceivedTotal.WithLabelValues(string(event.Type)).Inc()

		start := time.Now()
		if err := l.processor.PushEventToGoogle(ctx, event.UserID, event.TenantID, &internalEvent); err != nil {
			pushToGoogleDuration.WithLabelValues("error").Observe(time.Since(start).Seconds())
			l.logger.WithError(err).Errorf("Failed to push event %s to Google", event.EntityID)
		} else {
			pushToGoogleDuration.WithLabelValues("success").Observe(time.Since(start).Seconds())
			l.logger.Infof("Successfully pushed event %s to Google", event.EntityID)
		}

	case services.EventInternalEventDeleted:
		// Delete logic needs to be implemented in processor too
		// l.processor.DeleteEventFromGoogle(...)
		// For now, P3 mainly asked for Push (Create/Update).
		// Deletion is a good to have.
	}
}
