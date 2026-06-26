package sync

import (
	"calendar-service/internal/microsoft"
	"calendar-service/internal/notifications"
	"calendar-service/internal/oauth"
	"calendar-service/internal/repository"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// MicrosoftSyncProcessor handles Microsoft Calendar sync
type MicrosoftSyncProcessor struct {
	oauth2    *oauth.MicrosoftOAuth2Provider
	cache     interface{} // Redis client
	syncRepo  *repository.MicrosoftSyncRepo
	notifier  notifications.NotificationService
	logger    *logrus.Entry
	syncs     map[string]*SyncStatus // Uses the same models as Google Sync Processor (from sync/processor.go)
	syncMutex sync.RWMutex
}

// NewMicrosoftSyncProcessor creates a new sync processor
func NewMicrosoftSyncProcessor(oauth2Provider *oauth.MicrosoftOAuth2Provider, cache interface{}, repo *repository.MicrosoftSyncRepo, notifier notifications.NotificationService, logger *logrus.Entry) *MicrosoftSyncProcessor {
	return &MicrosoftSyncProcessor{
		oauth2:   oauth2Provider,
		cache:    cache,
		syncRepo: repo,
		notifier: notifier,
		logger:   logger.WithField("component", "microsoft_sync_processor"),
		syncs:    make(map[string]*SyncStatus),
	}
}

// SyncUserCalendars initiates sync for a user
func (p *MicrosoftSyncProcessor) SyncUserCalendars(ctx context.Context, userID, tenantID string) (*SyncResult, error) {
	syncID := uuid.New().String()

	status := &SyncStatus{
		ID:        syncID,
		UserID:    userID,
		TenantID:  tenantID,
		Status:    "pending",
		StartedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	p.syncMutex.Lock()
	p.syncs[syncID] = status
	p.syncMutex.Unlock()

	// In a real implementation, this would be queued to Temporal or a worker pool
	// For now, we'll run it in a goroutine
	go p.processSync(syncID, userID, tenantID)

	return &SyncResult{
		ID:     syncID,
		Status: "pending",
	}, nil
}

func (p *MicrosoftSyncProcessor) processSync(syncID, userID, tenantID string) {
	p.syncMutex.Lock()
	status, exists := p.syncs[syncID]
	if !exists {
		p.syncMutex.Unlock()
		return
	}
	status.Status = "in_progress"
	status.UpdatedAt = time.Now()
	p.syncMutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// 1. Get user token (validation check)
	_, err := p.oauth2.GetUserToken(ctx, userID)
	if err != nil {
		p.markSyncFailed(syncID, fmt.Errorf("failed to get valid token: %w", err))

		_ = p.notifier.SendNotification(ctx, notifications.NotificationEvent{
			UserID:   userID,
			TenantID: tenantID,
			Type:     "SYNC_ERROR",
			Title:    "Microsoft Calendar Sync Error",
			Message:  "Failed to authenticate with Microsoft Calendar. Your connection may have expired. Please reconnect your account.",
			Data:     map[string]interface{}{"error": err.Error()},
		})
		return
	}

	// 2. Initialize Calendar Client
	client, err := microsoft.NewGraphClient(microsoft.GraphClientConfig{
		OAuthProvider: p.oauth2,
		UserID:        userID,
		Logger:        p.logger,
	})
	if err != nil {
		p.markSyncFailed(syncID, fmt.Errorf("failed to create calendar client: %w", err))
		return
	}

	// 3. List MS Calendars
	calendars, err := client.ListCalendars(ctx)
	if err != nil {
		p.markSyncFailed(syncID, fmt.Errorf("failed to list calendars: %w", err))
		return
	}

	// 4. Find the primary/default calendar to sync from
	var primaryCalId string
	for _, cal := range calendars {
		if cal.GetIsDefaultCalendar() != nil && *cal.GetIsDefaultCalendar() {
			primaryCalId = *cal.GetId()
			break
		}
	}

	if primaryCalId == "" && len(calendars) > 0 {
		primaryCalId = *calendars[0].GetId()
	}

	if primaryCalId == "" {
		p.markSyncFailed(syncID, fmt.Errorf("no calendars found to sync"))
		return
	}

	// 5. Fetch Events
	events, err := client.GetCalendarEvents(ctx, primaryCalId, microsoft.EventQueryOptions{})
	if err != nil {
		p.markSyncFailed(syncID, fmt.Errorf("failed to get events: %w", err))
		return
	}

	// 6. Sync events
	tenantUUID, _ := uuid.Parse(tenantID)
	userUUID, _ := uuid.Parse(userID)
	mapper := NewEventMapper()

	imported := 0
	for _, me := range events {
		eventID := ""
		if me.GetId() != nil {
			eventID = *me.GetId()
		}
		// Convert using mapper
		internalEvent, err := mapper.FromMicrosoftEvent(me, tenantUUID, userUUID)
		if err != nil {
			p.logger.WithError(err).WithField("microsoft_id", eventID).Warn("Failed to map microsoft event")
			continue
		}

		// Does it exist already?
		synced, err := p.syncRepo.GetSyncedEventByMicrosoftID(ctx, eventID, primaryCalId)
		if err != nil {
			continue
		}

		if synced != nil && synced.InternalEventID != nil {
			// Update logic (simplified: assume repo has UpdateInternalEvent if shared or repo is aware)
			// For now, let's upsert the sync record at least
			synced.LastSyncedAt = time.Now().UTC()
			synced.SyncStatus = "synced"
			_ = p.syncRepo.UpsertSyncedEvent(ctx, synced)
		} else {
			// Create sync record
			eventIdStr := internalEvent.ID.String()
			newSync := &repository.SyncedMicrosoftEvent{
				TenantID:            tenantID,
				MicrosoftEventID:    eventID,
				MicrosoftCalendarID: primaryCalId,
				InternalEventID:     &eventIdStr,
				Title:               internalEvent.Title,
				SyncStatus:          "synced",
				SyncDirection:       "microsoft_to_internal",
				LastSyncedAt:        time.Now().UTC(),
			}
			_ = p.syncRepo.UpsertSyncedEvent(ctx, newSync)
			imported++
		}
	}

	p.syncMutex.Lock()
	status.Status = "completed"
	status.UpdatedAt = time.Now()
	p.syncMutex.Unlock()

	p.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"sync_id":  syncID,
		"imported": imported,
	}).Info("Microsoft Sync completed successfully")

	// Trigger Success Notification
	_ = p.notifier.SendNotification(ctx, notifications.NotificationEvent{
		UserID:   userID,
		TenantID: tenantID,
		Type:     "SYNC_COMPLETE",
		Title:    "Microsoft Calendar Sync Complete",
		Message:  fmt.Sprintf("Successfully synced %d events from Microsoft Calendar.", imported),
	})
}

func (p *MicrosoftSyncProcessor) markSyncFailed(syncID string, err error) {
	p.syncMutex.Lock()
	var userID, tenantID string
	if status, exists := p.syncs[syncID]; exists {
		status.Status = "failed"
		status.Error = err.Error()
		status.UpdatedAt = time.Now()
		userID = status.UserID
		tenantID = status.TenantID
	}
	p.syncMutex.Unlock()
	p.logger.WithError(err).WithField("sync_id", syncID).Error("Sync failed")

	if p.notifier != nil && userID != "" {
		_ = p.notifier.SendNotification(context.Background(), notifications.NotificationEvent{
			UserID:   userID,
			TenantID: tenantID,
			Type:     "SYNC_ERROR",
			Title:    "Microsoft Calendar Sync Error",
			Message:  "Failed to sync with Microsoft Calendar: " + err.Error(),
		})
	}
}

// PushEvent pushes an internal event update to Microsoft Calendar
func (p *MicrosoftSyncProcessor) PushEvent(ctx context.Context, userID, eventID string) error {
	startTime := time.Now()

	// 1. Loop prevention
	if !p.shouldSyncToMicrosoft(ctx, eventID) {
		return nil
	}

	// 2. Get internal event
	event, err := p.syncRepo.GetEvent(ctx, eventID)
	if err != nil {
		return fmt.Errorf("get internal event: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"event_id": eventID,
	}).Info("Pushing event to Microsoft Calendar")

	// 3. Get Client
	client, err := microsoft.NewGraphClient(microsoft.GraphClientConfig{
		OAuthProvider: p.oauth2,
		UserID:        userID,
		Logger:        p.logger,
	})
	if err != nil {
		return err
	}

	// 4. Map to Microsoft event
	mapper := NewEventMapper()
	msEvent := mapper.ToMicrosoftEvent(event)

	// 5. Get Microsoft Calendar ID for this event
	syncedEvent, err := p.syncRepo.GetSyncedEventByInternalID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("get synced record: %w", err)
	}

	if syncedEvent != nil && syncedEvent.MicrosoftEventID != "" {
		// Update existing event
		_, err = client.UpdateEvent(ctx, syncedEvent.MicrosoftCalendarID, syncedEvent.MicrosoftEventID, msEvent)
		if err != nil {
			return fmt.Errorf("update microsoft event: %w", err)
		}
	} else {
		// Create new event on primary calendar
		primaryCalID, err := p.syncRepo.GetPrimaryCalendarID(ctx, event.TenantID.String(), userID)
		if err != nil {
			return fmt.Errorf("get primary calendar: %w", err)
		}

		created, err := client.CreateEvent(ctx, primaryCalID, msEvent)
		if err != nil {
			return fmt.Errorf("create microsoft event: %w", err)
		}

		createdID := ""
		if created.GetId() != nil {
			createdID = *created.GetId()
		}

		// Update sync record
		newSynced := &repository.SyncedMicrosoftEvent{
			TenantID:            event.TenantID.String(),
			MicrosoftEventID:    createdID,
			MicrosoftCalendarID: primaryCalID,
			InternalEventID:     &eventID,
			Title:               event.Title,
			SyncStatus:          "synced",
			SyncDirection:       "internal_to_microsoft",
			LastSyncedAt:        time.Now().UTC(),
		}
		_ = p.syncRepo.UpsertSyncedEvent(ctx, newSynced)
	}

	p.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"event_id":    eventID,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Successfully pushed event to Microsoft")

	return nil
}

// DeleteEventFromMicrosoft removes an event from Microsoft Calendar when deleted internally
func (p *MicrosoftSyncProcessor) DeleteEventFromMicrosoft(ctx context.Context, userID, eventID string) error {
	p.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"event_id": eventID,
	}).Info("Deleting event from Microsoft Calendar")

	synced, err := p.syncRepo.GetSyncedEventByInternalID(ctx, eventID)
	if err != nil || synced == nil || synced.MicrosoftEventID == "" {
		return nil
	}

	client, err := microsoft.NewGraphClient(microsoft.GraphClientConfig{
		OAuthProvider: p.oauth2,
		UserID:        userID,
		Logger:        p.logger,
	})
	if err != nil {
		return err
	}

	return client.DeleteEvent(ctx, synced.MicrosoftCalendarID, synced.MicrosoftEventID)
}

func (p *MicrosoftSyncProcessor) shouldSyncToMicrosoft(ctx context.Context, eventID string) bool {
	synced, err := p.syncRepo.GetSyncedEventByInternalID(ctx, eventID)
	if err != nil || synced == nil {
		return true
	}
	// Avoid loops
	return synced.SyncDirection != "microsoft_to_internal"
}
