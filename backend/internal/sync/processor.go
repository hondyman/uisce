package sync

import (
	"context"
	"fmt"
	stdsync "sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/repository"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	syncJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "calendar_sync_jobs_total",
			Help: "Total number of calendar sync jobs",
		},
		[]string{"status"},
	)

	syncDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "calendar_sync_duration_seconds",
			Help:    "Duration of calendar sync operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)

	syncEventsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "calendar_sync_events_processed_total",
			Help: "Total number of events processed during sync",
		},
		[]string{"status"},
	)
)

type SyncStatus struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	TenantID        string     `json:"tenant_id"`
	Provider        string     `json:"provider"`
	Status          string     `json:"status"`
	Progress        int        `json:"progress"`
	TotalEvents     int        `json:"total_events"`
	ProcessedEvents int        `json:"processed_events"`
	Errors          []string   `json:"errors"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	TimeRange       TimeRange  `json:"time_range"`
}

type SyncProcessor struct {
	clients       map[repository.Provider]CalendarClient
	syncRepo      *repository.CalendarSyncRepo
	logger        *logrus.Entry
	activeSyncs   map[string]*SyncStatus
	mu            stdsync.RWMutex
	maxConcurrent int

	recurringService *RecurringEventService
}

func NewSyncProcessor(
	clients map[repository.Provider]CalendarClient,
	syncRepo *repository.CalendarSyncRepo,
	logger *logrus.Entry,
	maxConcurrent int,
) *SyncProcessor {
	if maxConcurrent == 0 {
		maxConcurrent = 10
	}
	return &SyncProcessor{
		clients:         clients,
		syncRepo:        syncRepo,
		logger:          logger.WithField("component", "sync_processor"),
		activeSyncs:     make(map[string]*SyncStatus),
		maxConcurrent:   maxConcurrent,
		recurringService: NewRecurringEventService(),
	}
}

func (p *SyncProcessor) StartSync(ctx context.Context, userID, tenantID string, provider repository.Provider, externalCalendarID, internalCalendarID string, timeRange TimeRange) (*SyncStatus, error) {
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
		Provider:  string(provider),
		Status:    "pending",
		Progress:  0,
		StartedAt: &now,
		Errors:    make([]string, 0),
		TimeRange: timeRange,
	}

	p.mu.Lock()
	p.activeSyncs[syncID] = status
	p.mu.Unlock()

	go p.runSync(context.Background(), status, provider, externalCalendarID, internalCalendarID, timeRange)

	p.logger.WithFields(logrus.Fields{
		"sync_id":     syncID,
		"user_id":     userID,
		"provider":    provider,
		"calendar_id": externalCalendarID,
	}).Info("Started calendar sync")

	return status, nil
}

func (p *SyncProcessor) GetSyncStatus(syncID string) (*SyncStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status, exists := p.activeSyncs[syncID]
	if !exists {
		return nil, fmt.Errorf("sync job not found: %s", syncID)
	}
	return status, nil
}

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
	provider repository.Provider,
	externalCalendarID, internalCalendarID string,
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

	client, ok := p.clients[provider]
	if !ok {
		p.addError(status, fmt.Sprintf("no calendar client for provider: %s", provider))
		status.Status = "failed"
		return
	}

	events, err := client.GetEvents(externalCalendarID, EventQueryOptions{
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

	status.TotalEvents = len(events)
	processedCount := 0

	for _, event := range events {
		select {
		case <-ctx.Done():
			status.Status = "cancelled"
			return
		default:
		}

		err := p.syncEventToDB(ctx, status, provider, &event, externalCalendarID, internalCalendarID)
		if err != nil {
			p.addError(status, fmt.Sprintf("Failed to sync event %s: %v", event.ID, err))
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
	}).Info("Calendar sync completed")
}

func (p *SyncProcessor) syncEventToDB(
	ctx context.Context,
	status *SyncStatus,
	provider repository.Provider,
	event *NormalizedEvent,
	externalCalendarID, internalCalendarID string,
) error {
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
		provider,
		event,
		externalCalendarID,
	)
	if err == nil && len(conflicts) > 0 {
		autoResolved := cd.AutoResolveConflicts(ctx, conflicts)
		for _, c := range autoResolved {
			cd.SaveConflict(ctx, c)
		}
		hasPending := false
		for _, c := range conflicts {
			if c.ResolutionStatus == ResolutionPending {
				hasPending = true
				break
			}
		}
		if hasPending {
			return nil
		}
	}

	mapper := NewEventMapper()
	syncedEvent, err := mapper.ToSyncedEvent(provider, event, status.TenantID, externalCalendarID, nil)
	if err != nil {
		return fmt.Errorf("map event: %w", err)
	}

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

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func (p *SyncProcessor) PushEventToGoogle(ctx context.Context, userID, tenantID string, event *models.InternalEvent) error {
	calendarID, err := p.syncRepo.GetPrimaryCalendarID(ctx, tenantID, userID, repository.ProviderGoogle)
	if err != nil {
		return fmt.Errorf("get primary calendar: %w", err)
	}

	client, ok := p.clients[repository.ProviderGoogle]
	if !ok {
		return fmt.Errorf("no google calendar client")
	}

	syncedEvent, err := p.syncRepo.GetSyncedEventByInternalID(ctx, event.ID.String())
	if err != nil {
		return fmt.Errorf("check synced event: %w", err)
	}

	externalEvent := NewEventMapper().ToProviderEvent(event)

	if syncedEvent != nil {
		if !event.UpdatedAt.After(syncedEvent.UpdatedAt) {
			p.logger.WithField("event_id", event.ID).Info("Skipping push: event not updated since last sync")
			return nil
		}

		updatedEvent, err := client.UpdateEvent(syncedEvent.ExternalCalendarID, syncedEvent.ExternalEventID, externalEvent)
		if err != nil {
			return fmt.Errorf("update external event: %w", err)
		}

		syncedEvent.LastSyncedAt = time.Now().UTC()
		syncedEvent.Title = updatedEvent.Title

		return p.syncRepo.UpsertSyncedEvent(ctx, syncedEvent)
	}

	createdEvent, err := client.CreateEvent(calendarID, externalEvent)
	if err != nil {
		return fmt.Errorf("create external event: %w", err)
	}

	newSyncedEvent, err := NewEventMapper().ToSyncedEvent(repository.ProviderGoogle, createdEvent, tenantID, calendarID, nil)
	if err != nil {
		return fmt.Errorf("map created event: %w", err)
	}

	internalID := event.ID.String()
	newSyncedEvent.InternalEventID = &internalID

	return p.syncRepo.UpsertSyncedEvent(ctx, newSyncedEvent)
}
