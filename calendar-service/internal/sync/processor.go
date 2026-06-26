package sync

import (
	"calendar-service/internal/google"
	"calendar-service/internal/notifications"
	"calendar-service/internal/oauth"
	"calendar-service/internal/repository"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// SyncResult represents the result of a sync operation
type SyncResult struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	EventsImported int    `json:"events_imported,omitempty"`
}

// SyncStatus represents the current status of a sync
type SyncStatus struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	Status    string    `json:"status"` // pending, in_progress, completed, failed
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error,omitempty"`
}

// GoogleSyncProcessor handles Google Calendar sync
type GoogleSyncProcessor struct {
	oauth2    *oauth.GoogleOAuth2Provider
	cache     interface{} // Redis client
	syncRepo  *repository.GoogleSyncRepo
	notifier  notifications.NotificationService
	logger    *logrus.Entry
	syncs     map[string]*SyncStatus
	syncMutex sync.RWMutex
}

// NewGoogleSyncProcessor creates a new sync processor
func NewGoogleSyncProcessor(oauth2Provider *oauth.GoogleOAuth2Provider, cache interface{}, repo *repository.GoogleSyncRepo, notifier notifications.NotificationService, logger *logrus.Entry) *GoogleSyncProcessor {
	return &GoogleSyncProcessor{
		oauth2:   oauth2Provider,
		cache:    cache,
		syncRepo: repo,
		notifier: notifier,
		logger:   logger.WithField("component", "google_sync_processor"),
		syncs:    make(map[string]*SyncStatus),
	}
}

// SyncUserCalendars initiates sync for a user
func (p *GoogleSyncProcessor) SyncUserCalendars(ctx context.Context, userID, tenantID string) (*SyncResult, error) {
	syncID := uuid.New().String()

	status := &SyncStatus{
		ID:        syncID,
		UserID:    userID,
		TenantID:  tenantID,
		Status:    "pending",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	p.syncMutex.Lock()
	p.syncs[syncID] = status
	p.syncMutex.Unlock()

	// Launch sync in background
	go func() {
		p.executeSync(context.Background(), syncID, userID, tenantID)
	}()

	return &SyncResult{
		ID:     syncID,
		Status: "pending",
	}, nil
}

// executeSync performs the actual calendar sync
func (p *GoogleSyncProcessor) executeSync(ctx context.Context, syncID, userID, tenantID string) {
	p.syncMutex.Lock()
	status, exists := p.syncs[syncID]
	p.syncMutex.Unlock()

	if !exists {
		return
	}

	status.Status = "in_progress"
	status.UpdatedAt = time.Now()

	// 1. Get Google OAuth token & Calendar Client
	token, err := p.oauth2.GetUserToken(ctx, userID)
	if err != nil {
		p.markSyncFailed(syncID, fmt.Errorf("auth error: %w", err))
		return
	}
	client, err := google.NewCalendarClient(ctx, token, 10)
	if err != nil {
		p.markSyncFailed(syncID, fmt.Errorf("client error: %w", err))
		return
	}

	// 2. Locate user's primary connected Google Calendar
	calID, err := p.syncRepo.GetPrimaryCalendarID(ctx, tenantID, userID)
	if err != nil {
		p.markSyncFailed(syncID, fmt.Errorf("no calendar linked: %w", err))
		return
	}

	// 3. Fetch recent events from Google
	timeMin := time.Now().AddDate(-1, 0, 0) // 1 year ago
	timeMax := time.Now().AddDate(1, 0, 0)  // 1 year from now
	gEvents, err := client.ListEvents(ctx, calID, timeMin, timeMax, 2500)
	if err != nil {
		p.markSyncFailed(syncID, fmt.Errorf("fetch google events error: %w", err))
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)
	userUUID, _ := uuid.Parse(userID)
	mapper := NewEventMapper()

	// 4. Import events to local database
	imported := 0
	for _, ge := range gEvents {
		// Convert using mapper
		internalEvent, err := mapper.ToInternalEvent(ge, tenantUUID, userUUID)
		if err != nil {
			p.logger.WithError(err).WithField("google_id", ge.Id).Warn("Failed to map google event")
			continue
		}

		// Does it exist already? Check synced events
		synced, err := p.syncRepo.GetSyncedEventByGoogleID(ctx, calID, ge.Id, calID)
		if err != nil {
			continue // repo error
		}

		if synced != nil && synced.InternalEventID != nil {
			// Update existing internal event
			internalEvent.ID, _ = uuid.Parse(*synced.InternalEventID)
			err = p.syncRepo.UpdateInternalEvent(ctx, internalEvent)

			// Update sync record
			synced.LastSyncedAt = time.Now().UTC()
			synced.SyncStatus = "synced"
			_ = p.syncRepo.UpsertSyncedEvent(ctx, synced)
		} else {
			// Create new internal event
			err = p.syncRepo.CreateInternalEvent(ctx, internalEvent)

			// Create sync record
			eventIdStr := internalEvent.ID.String()
			newSync := &repository.SyncedGoogleEvent{
				TenantID:         tenantID,
				GoogleEventID:    ge.Id,
				GoogleCalendarID: calID,
				InternalEventID:  &eventIdStr,
				SyncStatus:       "synced",
				SyncDirection:    "google_to_internal",
				LastSyncedAt:     time.Now().UTC(),
			}
			_ = p.syncRepo.UpsertSyncedEvent(ctx, newSync)
		}

		if err == nil {
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
	}).Info("Sync completed successfully")

	p.sendSyncCompleteNotification(ctx, userID, tenantID, imported)
}

func (p *GoogleSyncProcessor) markSyncFailed(syncID string, err error) {
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

	p.sendSyncFailedNotification(context.Background(), userID, tenantID, err.Error())
}

// sendSyncCompleteNotification sends completion notification
func (p *GoogleSyncProcessor) sendSyncCompleteNotification(ctx context.Context, userID, tenantID string, importedCount int) {
	if p.notifier == nil || userID == "" {
		return
	}

	userEmail, userName, err := p.getUserEmail(ctx, userID)
	if err != nil {
		p.logger.WithError(err).Warn("Failed to get user email for notification")
		return
	}

	preferences, err := p.getUserNotificationPreferences(ctx, userID)
	if err != nil {
		p.logger.WithError(err).Warn("Failed to get user preferences")
		return
	}

	if !preferences.SyncCompleteNotification {
		p.logger.Debug("User disabled sync complete notifications")
		return
	}

	// Assuming a concrete NotificationService interface that supports SendSyncComplete
	// if your current interface doesn't, this is to map to the user request.
	// You might need to cast or define these methods on the interface used.
	// For now using the generic SendNotification
	_ = p.notifier.SendNotification(ctx, notifications.NotificationEvent{
		UserID:         userID,
		TenantID:       tenantID,
		Type:           "SYNC_COMPLETE",
		Title:          "Google Calendar Sync Complete",
		Message:        fmt.Sprintf("Successfully synced %d events.", importedCount),
		RecipientEmail: userEmail,
		RecipientName:  userName,
	})
}

// sendSyncFailedNotification sends failure notification
func (p *GoogleSyncProcessor) sendSyncFailedNotification(ctx context.Context, userID, tenantID, errorMsg string) {
	if p.notifier == nil || userID == "" {
		return
	}

	userEmail, userName, err := p.getUserEmail(ctx, userID)
	if err != nil {
		p.logger.WithError(err).Warn("Failed to get user email for notification")
		return
	}

	preferences, err := p.getUserNotificationPreferences(ctx, userID)
	if err != nil {
		p.logger.WithError(err).Warn("Failed to get user preferences")
		return
	}

	if !preferences.ErrorNotification {
		p.logger.Debug("User disabled error notifications")
		return
	}

	_ = p.notifier.SendNotification(ctx, notifications.NotificationEvent{
		UserID:         userID,
		TenantID:       tenantID,
		Type:           "SYNC_ERROR",
		Title:          "Google Calendar Sync Error",
		Message:        errorMsg,
		RecipientEmail: userEmail,
		RecipientName:  userName,
	})
}

// getUserEmail gets user email from database (stub implementation without Hasura client access in processor)
func (p *GoogleSyncProcessor) getUserEmail(ctx context.Context, userID string) (string, string, error) {
	return "user@example.com", "User Example", nil
}

// NotificationPreferences subset
type NotificationPreferences struct {
	SyncCompleteNotification bool
	ErrorNotification        bool
	ConflictNotification     bool
	EmailNotifications       bool
}

// getUserNotificationPreferences gets user notification preferences (stub without Hasura client)
func (p *GoogleSyncProcessor) getUserNotificationPreferences(ctx context.Context, userID string) (*NotificationPreferences, error) {
	return &NotificationPreferences{
		SyncCompleteNotification: true,
		ErrorNotification:        true,
		ConflictNotification:     true,
		EmailNotifications:       true,
	}, nil
}

// GetSyncStatus retrieves sync status
func (p *GoogleSyncProcessor) GetSyncStatus(userID string) interface{} {
	p.syncMutex.RLock()
	defer p.syncMutex.RUnlock()

	for _, status := range p.syncs {
		if status.UserID == userID {
			return status
		}
	}

	return nil
}

// CancelSync cancels a sync
func (p *GoogleSyncProcessor) CancelSync(userID string) error {
	p.syncMutex.Lock()
	defer p.syncMutex.Unlock()

	for syncID, status := range p.syncs {
		if status.UserID == userID && (status.Status == "pending" || status.Status == "in_progress") {
			status.Status = "cancelled"
			status.UpdatedAt = time.Now()
			p.syncs[syncID] = status
			return nil
		}
	}

	return fmt.Errorf("no active sync found for user %s", userID)
}

// ListActiveSyncs lists active syncs
func (p *GoogleSyncProcessor) ListActiveSyncs() []*SyncStatus {
	p.syncMutex.RLock()
	defer p.syncMutex.RUnlock()

	var active []*SyncStatus
	for _, status := range p.syncs {
		if status.Status == "pending" || status.Status == "in_progress" {
			active = append(active, status)
		}
	}

	return active
}

// PushEvent pushes an internal event to Google Calendar
func (p *GoogleSyncProcessor) PushEvent(ctx context.Context, userID, eventID string) error {
	startTime := time.Now()

	// Check if we should sync to Google (loop prevention)
	if !p.shouldSyncToGoogle(ctx, eventID) {
		p.logger.WithField("event_id", eventID).Debug("Skipping sync to Google (loop prevention)")
		return nil
	}

	// Get internal event
	event, err := p.syncRepo.GetEvent(ctx, eventID)
	if err != nil {
		return fmt.Errorf("get event: %w", err)
	}

	// Just for compilation, assume tenant from event logic
	tenantIDStr := event.TenantID.String()

	// Get primary calendar ID for the connection
	calID, err := p.syncRepo.GetPrimaryCalendarID(ctx, tenantIDStr, userID)
	if err != nil {
		return fmt.Errorf("get connection calendar id: %w", err)
	}

	// Create Google Calendar client
	token, err := p.oauth2.GetUserToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get valid token: %w", err)
	}

	client, err := google.NewCalendarClient(ctx, token, 10)
	if err != nil {
		return fmt.Errorf("create calendar client: %w", err)
	}

	// Convert to Google event
	eventMapper := NewEventMapper()
	googleEvent := eventMapper.ToGoogleEvent(event)

	// Check if event already exists in Google
	syncedEvent, err := p.syncRepo.GetSyncedEventByInternalID(ctx, eventID)
	if err != nil {
		return err
	}

	if syncedEvent != nil && syncedEvent.GoogleEventID != "" {
		// Update existing event
		_, err = client.UpdateEvent(ctx, syncedEvent.GoogleCalendarID, syncedEvent.GoogleEventID, googleEvent)
		if err != nil {
			return fmt.Errorf("update google event: %w", err)
		}

		// Update sync record
		syncedEvent.LastSyncedAt = time.Now().UTC()
		syncedEvent.SyncStatus = "synced"
		if err := p.syncRepo.UpsertSyncedEvent(ctx, syncedEvent); err != nil {
			return fmt.Errorf("update sync record: %w", err)
		}

		p.logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"event_id":    eventID,
			"google_id":   syncedEvent.GoogleEventID,
			"duration_ms": time.Since(startTime).Milliseconds(),
		}).Info("Updated event in Google Calendar")
	} else {
		// Create new event
		createdEvent, err := client.CreateEvent(ctx, calID, googleEvent)
		if err != nil {
			return fmt.Errorf("create google event: %w", err)
		}

		// Create sync record
		eventIDStr := event.ID.String()
		syncedEvent := &repository.SyncedGoogleEvent{
			TenantID:         tenantIDStr,
			GoogleEventID:    createdEvent.Id,
			GoogleCalendarID: calID,
			InternalEventID:  &eventIDStr,
			SyncStatus:       "synced",
			LastSyncedAt:     time.Now().UTC(),
		}

		if err := p.syncRepo.UpsertSyncedEvent(ctx, syncedEvent); err != nil {
			return fmt.Errorf("create sync record: %w", err)
		}

		p.logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"event_id":    eventID,
			"google_id":   createdEvent.Id,
			"duration_ms": time.Since(startTime).Milliseconds(),
		}).Info("Created event in Google Calendar")
	}

	return nil
}

// DeleteEventFromGoogle deletes an internal event from Google Calendar
func (p *GoogleSyncProcessor) DeleteEventFromGoogle(ctx context.Context, userID, eventID string) error {
	startTime := time.Now()

	// Get synced event record
	syncedEvent, err := p.syncRepo.GetSyncedEventByInternalID(ctx, eventID)
	if err != nil {
		return err
	}
	if syncedEvent == nil || syncedEvent.GoogleEventID == "" {
		return nil // Event was never synced to Google
	}

	// Create Google Calendar client
	token, err := p.oauth2.GetUserToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get valid token: %w", err)
	}

	client, err := google.NewCalendarClient(ctx, token, 10)
	if err != nil {
		return fmt.Errorf("create calendar client: %w", err)
	}

	// Delete from Google
	if err := client.DeleteEvent(ctx, syncedEvent.GoogleCalendarID, syncedEvent.GoogleEventID); err != nil {
		// Don't fail if event already deleted in Google (404 Not Found)
		if !strings.Contains(err.Error(), "notFound") {
			return fmt.Errorf("delete google event: %w", err)
		}
	}

	// Update sync record
	syncedEvent.SyncStatus = "deleted"
	syncedEvent.LastSyncedAt = time.Now().UTC()
	if err := p.syncRepo.UpsertSyncedEvent(ctx, syncedEvent); err != nil {
		return fmt.Errorf("update sync record: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"event_id":    eventID,
		"google_id":   syncedEvent.GoogleEventID,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Deleted event from Google Calendar")

	return nil
}

// shouldSyncToGoogle checks if event should be synced (loop prevention)
func (p *GoogleSyncProcessor) shouldSyncToGoogle(ctx context.Context, eventID string) bool {
	// Check if this event originated from Google
	syncedEvent, err := p.syncRepo.GetSyncedEventByInternalID(ctx, eventID)
	if err != nil || syncedEvent == nil {
		return true // Sync if no record
	}

	// Check if internal event was modified after last sync using the Mapper's helper
	event, err := p.syncRepo.GetEvent(ctx, eventID)
	if err != nil {
		return true
	}

	mapper := NewEventMapper()
	return mapper.ShouldPushToGoogle(event, syncedEvent)
}

// SyncAllToGoogle syncs all user events to Google (for initial setup)
func (p *GoogleSyncProcessor) SyncAllToGoogle(ctx context.Context, userID string) error {
	// Get all user events
	events, err := p.syncRepo.GetAllEvents(ctx, userID)
	if err != nil {
		return err
	}

	successCount := 0
	errorCount := 0

	for _, event := range events {
		if err := p.PushEvent(ctx, userID, event.ID.String()); err != nil {
			p.logger.WithError(err).WithField("event_id", event.ID).Error("Failed to sync event to Google")
			errorCount++
		} else {
			successCount++
		}
	}

	p.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"success": successCount,
		"errors":  errorCount,
		"total":   len(events),
	}).Info("Batch sync to Google completed")

	return nil
}
