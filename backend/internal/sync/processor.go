package sync

import (
	"context"
	"fmt"
	stdsync "sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/google"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/oauth"
	"github.com/hondyman/semlayer/backend/internal/repository"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/calendar/v3"
)

// Prometheus metrics for sync operations
var (
	syncJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "google_sync_jobs_total",
			Help: "Total number of Google Calendar sync jobs",
		},
		[]string{"status"},
	)

	syncDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "google_sync_duration_seconds",
			Help:    "Duration of Google Calendar sync operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)

	syncEventsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "google_sync_events_processed_total",
			Help: "Total number of events processed during sync",
		},
		[]string{"status"},
	)
)

// SyncStatus represents the status of a sync job
type SyncStatus struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	TenantID        string     `json:"tenant_id"`
	Status          string     `json:"status"` // pending, running, completed, failed, cancelled
	Progress        int        `json:"progress"`
	TotalEvents     int        `json:"total_events"`
	ProcessedEvents int        `json:"processed_events"`
	Errors          []string   `json:"errors"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	TimeRange       TimeRange  `json:"time_range"`
}

// SyncProcessor handles background Google Calendar sync operations
type SyncProcessor struct {
	oauthProvider *oauth.GoogleOAuth2Provider
	syncRepo      *repository.GoogleSyncRepo
	logger        *logrus.Entry
	activeSyncs   map[string]*SyncStatus
	mu            stdsync.RWMutex
	maxConcurrent int

	recurringService *RecurringEventService
}

// NewSyncProcessor creates a new sync processor
func NewSyncProcessor(
	oauthProvider *oauth.GoogleOAuth2Provider,
	syncRepo *repository.GoogleSyncRepo,
	logger *logrus.Entry,
	maxConcurrent int,
) *SyncProcessor {
	if maxConcurrent == 0 {
		maxConcurrent = 10
	}

	return &SyncProcessor{
		oauthProvider:    oauthProvider,
		syncRepo:         syncRepo,
		logger:           logger.WithField("component", "sync_processor"),
		activeSyncs:      make(map[string]*SyncStatus),
		maxConcurrent:    maxConcurrent,
		recurringService: NewRecurringEventService(),
	}
}

// StartSync initiates a sync job for a user's Google Calendar
func (p *SyncProcessor) StartSync(ctx context.Context, userID, tenantID, googleCalendarID, internalCalendarID string, timeRange TimeRange) (*SyncStatus, error) {
	p.mu.RLock()
	activeCount := len(p.activeSyncs)
	p.mu.RUnlock()

	if activeCount >= p.maxConcurrent {
		return nil, fmt.Errorf("max concurrent syncs reached (%d)", p.maxConcurrent)
	}

	syncID := uuid.New().String()
	now := time.Now().UTC()

	status := &SyncStatus{
		ID:        syncID,
		UserID:    userID,
		TenantID:  tenantID,
		Status:    "pending",
		Progress:  0,
		StartedAt: &now,
		Errors:    make([]string, 0),
		TimeRange: timeRange,
	}

	p.mu.Lock()
	p.activeSyncs[syncID] = status
	p.mu.Unlock()

	go p.runSync(context.Background(), status, googleCalendarID, internalCalendarID, timeRange)

	p.logger.WithFields(logrus.Fields{
		"sync_id":     syncID,
		"user_id":     userID,
		"calendar_id": googleCalendarID,
	}).Info("Started Google Calendar sync")

	return status, nil
}

// GetSyncStatus returns the status of a sync job
func (p *SyncProcessor) GetSyncStatus(syncID string) (*SyncStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status, exists := p.activeSyncs[syncID]
	if !exists {
		return nil, fmt.Errorf("sync job not found: %s", syncID)
	}
	return status, nil
}

// CancelSync cancels an active sync job
func (p *SyncProcessor) CancelSync(syncID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	status, exists := p.activeSyncs[syncID]
	if !exists {
		return fmt.Errorf("sync job not found: %s", syncID)
	}

	if status.Status == "completed" || status.Status == "failed" {
		return fmt.Errorf("sync job already finished: %s", status.Status)
	}

	status.Status = "cancelled"
	now := time.Now().UTC()
	status.CompletedAt = &now

	delete(p.activeSyncs, syncID)
	return nil
}

// ListActiveSyncs returns all active sync jobs for a user
func (p *SyncProcessor) ListActiveSyncs(userID string) []*SyncStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var active []*SyncStatus
	for _, status := range p.activeSyncs {
		if status.UserID == userID && status.Status == "running" {
			active = append(active, status)
		}
	}
	return active
}

func (p *SyncProcessor) runSync(
	ctx context.Context,
	status *SyncStatus,
	googleCalendarID, internalCalendarID string,
	timeRange TimeRange,
) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		syncDuration.WithLabelValues(status.Status).Observe(duration.Seconds())
		syncJobsTotal.WithLabelValues(status.Status).Inc()

		p.mu.Lock()
		delete(p.activeSyncs, status.ID)
		p.mu.Unlock()
	}()

	status.Status = "running"

	client, err := google.NewCalendarClient(google.CalendarClientConfig{
		OAuthProvider: p.oauthProvider,
		UserID:        status.UserID,
		TenantID:      status.TenantID,
		Logger:        p.logger,
	})
	if err != nil {
		p.addError(status, fmt.Sprintf("Failed to create calendar client: %v", err))
		status.Status = "failed"
		return
	}

	events, err := client.GetCalendarEvents(ctx, googleCalendarID, google.EventQueryOptions{
		TimeMin:      timeRange.Start,
		TimeMax:      timeRange.End,
		SingleEvents: true,
		OrderBy:      "startTime",
	})
	if err != nil {
		p.addError(status, fmt.Sprintf("Failed to fetch events: %v", err))
		status.Status = "failed"
		return
	}

	status.TotalEvents = len(events.Items)
	processedCount := 0

	for _, event := range events.Items {
		select {
		case <-ctx.Done():
			status.Status = "cancelled"
			return
		default:
		}

		if event.Status == "cancelled" {
			continue
		}

		err := p.syncEventToDB(ctx, status, event, googleCalendarID, internalCalendarID)
		if err != nil {
			p.addError(status, fmt.Sprintf("Failed to sync event %s: %v", event.Id, err))
			syncEventsProcessed.WithLabelValues("failed").Inc()
		} else {
			syncEventsProcessed.WithLabelValues("success").Inc()
			processedCount++
		}

		status.ProcessedEvents = processedCount
		if status.TotalEvents > 0 {
			status.Progress = int(float64(processedCount) / float64(status.TotalEvents) * 100)
		}
	}

	now := time.Now().UTC()
	status.CompletedAt = &now
	status.Status = "completed"

	p.logger.WithFields(logrus.Fields{
		"sync_id":       status.ID,
		"events_synced": processedCount,
		"duration_ms":   time.Since(startTime).Milliseconds(),
	}).Info("Google Calendar sync completed")
}

func (p *SyncProcessor) syncEventToDB(
	ctx context.Context,
	status *SyncStatus,
	event *calendar.Event,
	googleCalendarID, internalCalendarID string,
) error {
	// Conflict Detection
	cd := NewConflictDetector(ConflictDetectorConfig{
		SyncRepo:               p.syncRepo,
		Logger:                 p.logger,
		AutoResolveTimeOverlap: true,
		AutoResolveThreshold:   15 * time.Minute,
	})

	conflicts, err := cd.DetectConflicts(
		ctx,
		status.TenantID,
		status.UserID,
		"", // ConnectionID would be needed here
		event,
		googleCalendarID,
	)
	if err == nil && len(conflicts) > 0 {
		autoResolved := cd.AutoResolveConflicts(ctx, conflicts)
		for _, c := range autoResolved {
			cd.SaveConflict(ctx, c)
		}
		// If there are pending conflicts, skip sync
		hasPending := false
		for _, c := range conflicts {
			if c.ResolutionStatus == ResolutionPending {
				hasPending = true
				break
			}
		}
		if hasPending {
			return nil // Skip
		}
	}

	// Use EventMapper to create SyncedGoogleEvent
	mapper := NewEventMapper()
	syncedEvent, err := mapper.ToSyncedEvent(event, status.TenantID, googleCalendarID, nil) // InternalEventID nil for now
	if err != nil {
		return fmt.Errorf("map event: %w", err)
	}

	// If internalCalendarID provided, set it
	if internalCalendarID != "" {
		syncedEvent.InternalCalendarID = &internalCalendarID
	}

	return p.syncRepo.UpsertSyncedEvent(ctx, syncedEvent)
}

func (p *SyncProcessor) addError(status *SyncStatus, errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	status.Errors = append(status.Errors, errMsg)
	p.logger.WithField("sync_id", status.ID).Warnf("Sync error: %s", errMsg)
}

// TimeRange represents a time range for sync
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// PushEventToGoogle pushes an internal event to Google Calendar
func (p *SyncProcessor) PushEventToGoogle(ctx context.Context, userID, tenantID string, event *models.InternalEvent) error {
	// 1. Get Primary Calendar ID
	calendarID, err := p.syncRepo.GetPrimaryCalendarID(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("get primary calendar: %w", err)
	}

	// 2. Create Client
	client, err := google.NewCalendarClient(google.CalendarClientConfig{
		OAuthProvider: p.oauthProvider,
		UserID:        userID,
		TenantID:      tenantID,
		Logger:        p.logger,
	})
	if err != nil {
		return fmt.Errorf("create calendar client: %w", err)
	}

	// 3. Check if event is already synced
	syncedEvent, err := p.syncRepo.GetSyncedEventByInternalID(ctx, event.ID.String())
	if err != nil {
		return fmt.Errorf("check synced event: %w", err)
	}

	googleEvent := NewEventMapper().ToGoogleEvent(event)

	if syncedEvent != nil {
		// Update existing event
		// Avoid loop: if the event was just synced *from* Google, we might shouldn't push back?
		// But here we assume this is called when internal event changed.
		// We should update Google.

		// Avoid loop: if the event was just synced from Google, LastSyncedAt should be >= InternalEvent.UpdatedAt
		// But we need to be careful with clock skew.
		// If InternalEvent.UpdatedAt is significantly newer than LastSyncedAt, it's a local change.
		if !event.UpdatedAt.After(syncedEvent.LastSyncedAt) {
			p.logger.WithField("event_id", event.ID).Info("Skipping push: event not updated since last sync")
			return nil
		}

		updatedEvent, err := client.UpdateEvent(ctx, syncedEvent.GoogleCalendarID, syncedEvent.GoogleEventID, googleEvent)
		if err != nil {
			return fmt.Errorf("update google event: %w", err)
		}

		// Update synced event record
		syncedEvent.LastSyncedAt = time.Now().UTC()
		syncedEvent.Title = updatedEvent.Summary // Update local record with latest from Google? Or trust internal?
		// Actually we just pushed internal to Google, so they should match.

		return p.syncRepo.UpsertSyncedEvent(ctx, syncedEvent)
	}

	// Insert new event
	createdEvent, err := client.CreateEvent(ctx, calendarID, googleEvent)
	if err != nil {
		return fmt.Errorf("create google event: %w", err)
	}

	// Create synced event record
	newSyncedEvent, err := NewEventMapper().ToSyncedEvent(createdEvent, tenantID, calendarID, nil)
	if err != nil {
		return fmt.Errorf("map created event: %w", err)
	}

	internalID := event.ID.String()
	newSyncedEvent.InternalEventID = &internalID

	return p.syncRepo.UpsertSyncedEvent(ctx, newSyncedEvent)
}
