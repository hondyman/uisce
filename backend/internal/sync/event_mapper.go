package sync

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/repository"
	"google.golang.org/api/calendar/v3"
)

// EventMapper handles conversion between Google Calendar events and internal events
type EventMapper struct {
	// Dependencies if needed, e.g. for user lookup
}

func NewEventMapper() *EventMapper {
	return &EventMapper{}
}

// ToInternalEvent converts a Google Event to an InternalEvent model
func (m *EventMapper) ToInternalEvent(
	googleEvent *calendar.Event,
	tenantID, userID uuid.UUID,
) (*models.InternalEvent, error) {

	start, err := parseEventTime(googleEvent.Start)
	if err != nil {
		return nil, fmt.Errorf("invalid start time: %w", err)
	}
	end, err := parseEventTime(googleEvent.End)
	if err != nil {
		return nil, fmt.Errorf("invalid end time: %w", err)
	}

	isAllDay := googleEvent.Start.Date != "" && googleEvent.Start.TimeZone == ""

	// Check recurrence
	isRecurring := googleEvent.Recurrence != nil && len(googleEvent.Recurrence) > 0
	var rrule *string
	if isRecurring {
		r := googleEvent.Recurrence[0] // taking first rule for now
		rrule = &r
	}

	desc := googleEvent.Description
	loc := googleEvent.Location

	return &models.InternalEvent{
		ID:             uuid.New(), // Generate new ID for new events
		TenantID:       tenantID,
		UserID:         userID,
		Title:          googleEvent.Summary,
		Description:    &desc,
		Location:       &loc,
		StartTime:      start,
		EndTime:        end,
		IsAllDay:       isAllDay,
		IsRecurring:    isRecurring,
		RecurrenceRule: rrule,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

// ToSyncedEvent creates a SyncedGoogleEvent from a Google Event
func (m *EventMapper) ToSyncedEvent(
	googleEvent *calendar.Event,
	tenantID, googleCalendarID string,
	internalEventID *string,
) (*repository.SyncedGoogleEvent, error) {
	start, err := parseEventTime(googleEvent.Start)
	if err != nil {
		return nil, fmt.Errorf("invalid start time: %w", err)
	}
	end, err := parseEventTime(googleEvent.End)
	if err != nil {
		return nil, fmt.Errorf("invalid end time: %w", err)
	}

	isAllDay := googleEvent.Start.Date != "" && googleEvent.Start.TimeZone == ""
	isRecurring := googleEvent.Recurrence != nil && len(googleEvent.Recurrence) > 0
	var rrule *string
	if isRecurring {
		r := googleEvent.Recurrence[0]
		rrule = &r
	}

	var recurrenceID *string
	if googleEvent.RecurringEventId != "" {
		recurrenceID = &googleEvent.RecurringEventId
	}

	desc := googleEvent.Description
	loc := googleEvent.Location

	return &repository.SyncedGoogleEvent{
		TenantID:         tenantID,
		GoogleEventID:    googleEvent.Id,
		GoogleCalendarID: googleCalendarID,
		InternalEventID:  internalEventID,
		Title:            googleEvent.Summary,
		Description:      &desc,
		Location:         &loc,
		StartTime:        start,
		EndTime:          end,
		IsAllDay:         isAllDay,
		IsRecurring:      isRecurring,
		RecurrenceRule:   rrule,
		RecurrenceID:     recurrenceID,
		SyncStatus:       "synced",
		LastSyncedAt:     time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}, nil
}

// ToGoogleEvent converts an InternalEvent to a Google Calendar Event
func (m *EventMapper) ToGoogleEvent(internalEvent *models.InternalEvent) *calendar.Event {
	event := &calendar.Event{
		Summary: internalEvent.Title,
	}

	if internalEvent.Description != nil {
		event.Description = *internalEvent.Description
	}
	if internalEvent.Location != nil {
		event.Location = *internalEvent.Location
	}

	if internalEvent.IsAllDay {
		event.Start = &calendar.EventDateTime{
			Date: internalEvent.StartTime.Format("2006-01-02"),
		}
		// Google Calendar end date for all-day events is exclusive, so we might need check if EndTime is inclusive/exclusive in InternalEvent.
		// Usually all-day events are stored as midnight to midnight.
		// If InternalEvent.EndTime is the start of the next day, this is correct.
		event.End = &calendar.EventDateTime{
			Date: internalEvent.EndTime.Format("2006-01-02"),
		}
	} else {
		event.Start = &calendar.EventDateTime{
			DateTime: internalEvent.StartTime.Format(time.RFC3339),
		}
		event.End = &calendar.EventDateTime{
			DateTime: internalEvent.EndTime.Format(time.RFC3339),
		}
	}

	if internalEvent.IsRecurring && internalEvent.RecurrenceRule != nil {
		event.Recurrence = []string{*internalEvent.RecurrenceRule}
	}

	return event
}

func parseEventTime(et *calendar.EventDateTime) (time.Time, error) {
	if et == nil {
		return time.Time{}, fmt.Errorf("event time is nil")
	}
	if et.DateTime != "" {
		return time.Parse(time.RFC3339, et.DateTime)
	}
	if et.Date != "" {
		return time.Parse("2006-01-02", et.Date)
	}
	return time.Time{}, fmt.Errorf("no valid time in event")
}
