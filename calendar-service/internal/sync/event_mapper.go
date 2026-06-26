package sync

import (
	"fmt"
	"time"

	"calendar-service/internal/models"
	"calendar-service/internal/repository"

	"github.com/google/uuid"
	msgraphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
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

// ToMicrosoftEvent converts an InternalEvent to a Microsoft Graph Event
func (m *EventMapper) ToMicrosoftEvent(internalEvent *models.InternalEvent) msgraphmodels.Eventable {
	event := msgraphmodels.NewEvent()

	subject := internalEvent.Title
	event.SetSubject(&subject)

	event.SetIsAllDay(&internalEvent.IsAllDay)

	if internalEvent.Description != nil {
		body := msgraphmodels.NewItemBody()
		contentType := msgraphmodels.TEXT_BODYTYPE
		body.SetContentType(&contentType)
		body.SetContent(internalEvent.Description)
		event.SetBody(body)
	}

	if internalEvent.Location != nil {
		location := msgraphmodels.NewLocation()
		location.SetDisplayName(internalEvent.Location)
		event.SetLocation(location)
	}

	start := msgraphmodels.NewDateTimeTimeZone()
	startTimeStr := internalEvent.StartTime.Format(time.RFC3339)
	start.SetDateTime(&startTimeStr)
	utc := "UTC"
	start.SetTimeZone(&utc)
	event.SetStart(start)

	end := msgraphmodels.NewDateTimeTimeZone()
	endTimeStr := internalEvent.EndTime.Format(time.RFC3339)
	end.SetDateTime(&endTimeStr)
	end.SetTimeZone(&utc)
	event.SetEnd(end)

	return event
}

// FromMicrosoftEvent converts a Microsoft Graph Event to an InternalEvent model
func (m *EventMapper) FromMicrosoftEvent(
	me msgraphmodels.Eventable,
	tenantID, userID uuid.UUID,
) (*models.InternalEvent, error) {
	startStr := ""
	if me.GetStart() != nil && me.GetStart().GetDateTime() != nil {
		startStr = *me.GetStart().GetDateTime()
	}
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		// Try fallback if MS gives us a partial format
		start, _ = time.Parse("2006-01-02T15:04:05.9999999", startStr)
	}

	endStr := ""
	if me.GetEnd() != nil && me.GetEnd().GetDateTime() != nil {
		endStr = *me.GetEnd().GetDateTime()
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		end, _ = time.Parse("2006-01-02T15:04:05.9999999", endStr)
	}

	var description *string
	if me.GetBody() != nil && me.GetBody().GetContent() != nil {
		description = me.GetBody().GetContent()
	}

	var location *string
	if me.GetLocation() != nil && me.GetLocation().GetDisplayName() != nil {
		location = me.GetLocation().GetDisplayName()
	}

	subject := ""
	if me.GetSubject() != nil {
		subject = *me.GetSubject()
	}

	isAllDay := false
	if me.GetIsAllDay() != nil {
		isAllDay = *me.GetIsAllDay()
	}

	return &models.InternalEvent{
		ID:          uuid.New(),
		TenantID:    tenantID,
		UserID:      userID,
		Title:       subject,
		Description: description,
		Location:    location,
		StartTime:   start,
		EndTime:     end,
		IsAllDay:    isAllDay,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// ToSyncedMicrosoftEvent creates a SyncedMicrosoftEvent from a Microsoft Graph Event
func (m *EventMapper) ToSyncedMicrosoftEvent(
	me msgraphmodels.Eventable,
	tenantID, microsoftCalendarID string,
	internalEventID *string,
) (*repository.SyncedMicrosoftEvent, error) {
	startStr := ""
	if me.GetStart() != nil && me.GetStart().GetDateTime() != nil {
		startStr = *me.GetStart().GetDateTime()
	}
	start, _ := time.Parse(time.RFC3339, startStr)

	endStr := ""
	if me.GetEnd() != nil && me.GetEnd().GetDateTime() != nil {
		endStr = *me.GetEnd().GetDateTime()
	}
	end, _ := time.Parse(time.RFC3339, endStr)

	var description *string
	if me.GetBody() != nil && me.GetBody().GetContent() != nil {
		description = me.GetBody().GetContent()
	}

	var location *string
	if me.GetLocation() != nil && me.GetLocation().GetDisplayName() != nil {
		location = me.GetLocation().GetDisplayName()
	}

	subject := ""
	if me.GetSubject() != nil {
		subject = *me.GetSubject()
	}

	isAllDay := false
	if me.GetIsAllDay() != nil {
		isAllDay = *me.GetIsAllDay()
	}

	eventID := ""
	if me.GetId() != nil {
		eventID = *me.GetId()
	}

	return &repository.SyncedMicrosoftEvent{
		TenantID:            tenantID,
		MicrosoftEventID:    eventID,
		MicrosoftCalendarID: microsoftCalendarID,
		InternalEventID:     internalEventID,
		Title:               subject,
		Description:         description,
		Location:            location,
		StartTime:           start,
		EndTime:             end,
		IsAllDay:            isAllDay,
		SyncStatus:          "synced",
		LastSyncedAt:        time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}, nil
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

// ShouldPushToGoogle checks if an event should be pushed based on sync direction
func (m *EventMapper) ShouldPushToGoogle(event *models.InternalEvent, syncedEvent *repository.SyncedGoogleEvent) bool {
	if syncedEvent == nil {
		return true
	}

	// Only push if sync direction permits
	if syncedEvent.SyncDirection == "google_to_internal" {
		return false
	}

	// Push if the event is newer than the last push
	// We use UpdatedAt to see if the internal event changed since we last pushed to Google
	if syncedEvent.LastPushedToGoogle != nil {
		if event.UpdatedAt.After(*syncedEvent.LastPushedToGoogle) {
			return true
		}
	} else if event.UpdatedAt.After(syncedEvent.LastSyncedAt) {
		return true
	}

	return false
}
