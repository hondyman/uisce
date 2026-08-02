package sync

import (
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/repository"
)

type EventMapper struct{}

func NewEventMapper() *EventMapper {
	return &EventMapper{}
}

func (m *EventMapper) ToInternalEvent(
	provider repository.Provider,
	event *NormalizedEvent,
	tenantID, userID uuid.UUID,
) (*models.InternalEvent, error) {
	return &models.InternalEvent{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		Title:     event.Title,
		StartTime: event.StartTime,
		EndTime:   event.EndTime,
		IsAllDay:  event.IsAllDay,
		UpdatedAt: time.Now(),
	}, nil
}

func (m *EventMapper) ToSyncedEvent(
	provider repository.Provider,
	event *NormalizedEvent,
	tenantID, externalCalendarID string,
	internalEventID *string,
) (*repository.SyncedCalendarEvent, error) {
	return &repository.SyncedCalendarEvent{
		TenantID:           tenantID,
		Provider:           provider,
		ExternalEventID:    event.ID,
		ExternalCalendarID: externalCalendarID,
		InternalEventID:    internalEventID,
		Title:              event.Title,
		Description:        &event.Description,
		Location:           &event.Location,
		StartTime:          event.StartTime,
		EndTime:            event.EndTime,
		IsAllDay:           event.IsAllDay,
		IsRecurring:        event.IsRecurring,
		RecurrenceRule:     &event.RecurrenceRule,
		RecurrenceID:       &event.RecurrenceID,
		SyncStatus:         "synced",
		LastSyncedAt:       time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}, nil
}

func (m *EventMapper) ToProviderEvent(internalEvent *models.InternalEvent) *NormalizedEvent {
	return &NormalizedEvent{
		Title:          internalEvent.Title,
		Description:    derefString(internalEvent.Description),
		Location:       derefString(internalEvent.Location),
		StartTime:      internalEvent.StartTime,
		EndTime:        internalEvent.EndTime,
		IsAllDay:       internalEvent.IsAllDay,
		IsRecurring:    internalEvent.IsRecurring,
		RecurrenceRule: derefString(internalEvent.RecurrenceRule),
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
