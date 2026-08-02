package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hondyman/uisce/backend/internal/repository"
	"github.com/sirupsen/logrus"
)

type ConflictType string

const (
	ConflictTimeOverlap       ConflictType = "time_overlap"
	ConflictTitleMismatch     ConflictType = "title_mismatch"
	ConflictDeletedVsModified ConflictType = "deleted_vs_modified"
	ConflictRecurringChange   ConflictType = "recurring_change"
	ConflictAttendeeChange    ConflictType = "attendee_change"
)

type ConflictSeverity string

const (
	SeverityInfo     ConflictSeverity = "info"
	SeverityWarning  ConflictSeverity = "warning"
	SeverityError    ConflictSeverity = "error"
	SeverityCritical ConflictSeverity = "critical"
)

type ResolutionStatus string

const (
	ResolutionPending          ResolutionStatus = "pending"
	ResolutionAutoResolved     ResolutionStatus = "auto_resolved"
	ResolutionManuallyResolved ResolutionStatus = "manually_resolved"
	ResolutionSkipped          ResolutionStatus = "skipped"
	ResolutionEscalated        ResolutionStatus = "escalated"
)

type ResolutionStrategy string

const (
	StrategyKeepExternal ResolutionStrategy = "keep_external"
	StrategyKeepInternal ResolutionStrategy = "keep_internal"
	StrategyMerge        ResolutionStrategy = "merge"
	StrategySkip         ResolutionStrategy = "skip"
	StrategyManual       ResolutionStrategy = "manual"
)

type Conflict struct {
	ID                 string              `json:"id"`
	TenantID           string              `json:"tenant_id"`
	UserID             string              `json:"user_id"`
	ConnectionID       string              `json:"connection_id"`
	Provider           repository.Provider `json:"provider"`
	ExternalEventID    string              `json:"external_event_id"`
	ExternalCalendarID string              `json:"external_calendar_id"`
	InternalEventID    *string             `json:"internal_event_id"`
	ConflictType       ConflictType        `json:"conflict_type"`
	Severity           ConflictSeverity    `json:"severity"`
	Description        string              `json:"description"`
	ExternalEventData  interface{}         `json:"external_event_data"`
	InternalEventData  interface{}         `json:"internal_event_data"`
	ResolutionStatus   ResolutionStatus    `json:"resolution_status"`
	ResolutionStrategy *ResolutionStrategy `json:"resolution_strategy"`
	DetectedAt         time.Time           `json:"detected_at"`
}

type ConflictDetector struct {
	syncRepo *repository.CalendarSyncRepo
	logger   *logrus.Entry
}

type ConflictDetectorConfig struct {
	SyncRepo               *repository.CalendarSyncRepo
	Logger                 *logrus.Entry
	AutoResolveTimeOverlap bool
	AutoResolveThreshold   time.Duration
}

func NewConflictDetector(cfg ConflictDetectorConfig) *ConflictDetector {
	return &ConflictDetector{
		syncRepo: cfg.SyncRepo,
		logger:   cfg.Logger.WithField("component", "conflict_detector"),
	}
}

func (cd *ConflictDetector) DetectConflicts(
	ctx context.Context,
	tenantID, userID string,
	provider repository.Provider,
	externalEvent *NormalizedEvent,
	externalCalendarID string,
) ([]Conflict, error) {
	var conflicts []Conflict

	startTime := externalEvent.StartTime
	endTime := externalEvent.EndTime

	syncedEvent, err := cd.syncRepo.GetSyncedEventByExternalID(ctx, "", provider, externalEvent.ID, externalCalendarID)
	if err != nil {
		cd.logger.WithError(err).Warn("Failed to get synced event")
	}

	conflictingEvents, err := cd.syncRepo.FindConflictingEvents(ctx, tenantID, startTime, endTime, nil)
	if err != nil {
		return nil, fmt.Errorf("find conflicting events: %w", err)
	}

	for _, internalEvent := range conflictingEvents {
		if internalEvent.ExternalEventID == externalEvent.ID {
			continue
		}

		conflict := cd.analyzeConflict(provider, externalEvent, internalEvent, syncedEvent)
		if conflict != nil {
			conflict.TenantID = tenantID
			conflict.UserID = userID
			conflict.Provider = provider
			conflict.ExternalEventID = externalEvent.ID
			conflict.ExternalCalendarID = externalCalendarID
			conflict.DetectedAt = time.Now().UTC()

			conflicts = append(conflicts, *conflict)

			conflictsDetectedTotal.WithLabelValues(string(conflict.ConflictType), string(conflict.Severity)).Inc()
		}
	}

	if syncedEvent != nil && externalEvent.ID == "" {
		conflict := cd.detectDeletedVsModified(provider, externalEvent, syncedEvent)
		if conflict != nil {
			conflict.TenantID = tenantID
			conflict.UserID = userID
			conflict.DetectedAt = time.Now().UTC()
			conflicts = append(conflicts, *conflict)

			conflictsDetectedTotal.WithLabelValues(string(conflict.ConflictType), string(conflict.Severity)).Inc()
		}
	}

	return conflicts, nil
}

func (cd *ConflictDetector) analyzeConflict(
	provider repository.Provider,
	externalEvent *NormalizedEvent,
	internalEvent repository.SyncedCalendarEvent,
	syncedEvent *repository.SyncedCalendarEvent,
) *Conflict {
	if externalEvent.Title != internalEvent.Title {
		desc := fmt.Sprintf("Time overlap with different event: '%s' vs '%s'", externalEvent.Title, internalEvent.Title)
		return &Conflict{
			InternalEventID:    internalEvent.InternalEventID,
			ConflictType:       ConflictTimeOverlap,
			Severity:           SeverityWarning,
			Description:         desc,
			ExternalEventData:  externalEvent,
			InternalEventData:  internalEvent,
			ResolutionStatus:   ResolutionPending,
		}
	}

	if syncedEvent != nil {
		timeDiff := externalEvent.StartTime.Sub(syncedEvent.StartTime)
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}

		if timeDiff > 15*time.Minute {
			return &Conflict{
				InternalEventID:   internalEvent.InternalEventID,
				ConflictType:      ConflictTitleMismatch,
				Severity:          SeverityWarning,
				Description:        fmt.Sprintf("Event time shifted by %v", timeDiff),
				ExternalEventData:  externalEvent,
				InternalEventData: internalEvent,
				ResolutionStatus:   ResolutionPending,
			}
		}
	}

	if externalEvent.IsRecurring && !internalEvent.IsRecurring {
		return &Conflict{
			InternalEventID:   internalEvent.InternalEventID,
			ConflictType:      ConflictRecurringChange,
			Severity:           SeverityError,
			Description:        "Event changed from single to recurring",
			ExternalEventData:  externalEvent,
			InternalEventData:  internalEvent,
			ResolutionStatus:   ResolutionPending,
		}
	}

	return nil
}

func (cd *ConflictDetector) detectDeletedVsModified(
	provider repository.Provider,
	externalEvent *NormalizedEvent,
	syncedEvent *repository.SyncedCalendarEvent,
) *Conflict {
	if syncedEvent.UpdatedAt.After(syncedEvent.LastSyncedAt) {
		return &Conflict{
			InternalEventID:   syncedEvent.InternalEventID,
			ConflictType:       ConflictDeletedVsModified,
			Severity:           SeverityCritical,
			Description:         "Event deleted externally but modified internally after last sync",
			ExternalEventData:   externalEvent,
			InternalEventData:   syncedEvent,
			ResolutionStatus:     ResolutionPending,
		}
	}

	st := StrategyKeepExternal
	return &Conflict{
		InternalEventID:     syncedEvent.InternalEventID,
		ConflictType:        ConflictDeletedVsModified,
		Severity:            SeverityWarning,
		Description:         "Event deleted externally",
		ExternalEventData:   externalEvent,
		InternalEventData:   syncedEvent,
		ResolutionStatus:     ResolutionAutoResolved,
		ResolutionStrategy:  &st,
	}
}

func (cd *ConflictDetector) AutoResolveConflicts(ctx context.Context, conflicts []Conflict) []Conflict {
	var autoResolved []Conflict

	for i, conflict := range conflicts {
		if conflict.ConflictType == ConflictTitleMismatch && conflict.Severity == SeverityWarning {
			if strings.Contains(conflict.Description, "shifted by") {
				st := StrategyKeepExternal
				conflicts[i].ResolutionStatus = ResolutionAutoResolved
				conflicts[i].ResolutionStrategy = &st
				autoResolved = append(autoResolved, conflicts[i])
			}
		} else if conflict.ConflictType == ConflictDeletedVsModified && conflict.Severity == SeverityWarning {
			st := StrategyKeepExternal
			conflicts[i].ResolutionStatus = ResolutionAutoResolved
			conflicts[i].ResolutionStrategy = &st
			autoResolved = append(autoResolved, conflicts[i])
		}
	}

	return autoResolved
}

func (cd *ConflictDetector) SaveConflict(ctx context.Context, conflict Conflict) error {
	conflictMap := map[string]interface{}{
		"tenant_id":             conflict.TenantID,
		"user_id":               conflict.UserID,
		"connection_id":         conflict.ConnectionID,
		"provider":              string(conflict.Provider),
		"external_event_id":     conflict.ExternalEventID,
		"external_calendar_id":  conflict.ExternalCalendarID,
		"internal_event_id":     conflict.InternalEventID,
		"conflict_type":         string(conflict.ConflictType),
		"severity":              string(conflict.Severity),
		"description":           conflict.Description,
		"external_event_data":    conflict.ExternalEventData,
		"internal_event_data":   conflict.InternalEventData,
		"resolution_status":     string(conflict.ResolutionStatus),
		"detected_at":           conflict.DetectedAt,
	}

	if conflict.ResolutionStrategy != nil {
		conflictMap["resolution_strategy"] = string(*conflict.ResolutionStrategy)
	}

	return cd.syncRepo.SaveConflict(ctx, conflictMap)
}

func (cd *ConflictDetector) ResolveConflict(ctx context.Context, conflictID string, strategy ResolutionStrategy) error {
	record, err := cd.syncRepo.GetConflict(ctx, conflictID)
	if err != nil {
		return fmt.Errorf("get conflict: %w", err)
	}
	if record == nil {
		return fmt.Errorf("conflict not found")
	}

	if strategy == StrategyKeepExternal {
		if record.InternalEventID != nil {
			cd.logger.Infof("Resolving conflict %s with %s (Data patch pending)", conflictID, strategy)
		}
	}

	stratStr := string(strategy)
	conflictResolvedTotal.WithLabelValues(stratStr).Inc()
	return cd.syncRepo.UpdateConflictStatus(ctx, conflictID, string(ResolutionManuallyResolved), &stratStr)
}
