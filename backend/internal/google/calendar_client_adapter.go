package google

import (
	"context"
	"fmt"
	"time"

	"github.com/hondyman/uisce/backend/internal/sync"
	"google.golang.org/api/calendar/v3"
)

type GoogleCalendarAdapter struct {
	client *CalendarClient
}

func NewGoogleCalendarAdapter(client *CalendarClient) *GoogleCalendarAdapter {
	return &GoogleCalendarAdapter{client: client}
}

func (a *GoogleCalendarAdapter) ListCalendars() ([]sync.CalendarInfo, error) {
	ctx := context.Background()
	calendars, err := a.client.ListCalendars(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]sync.CalendarInfo, 0, len(calendars))
	for _, c := range calendars {
		result = append(result, sync.CalendarInfo{
			ID:          c.Id,
			Name:        c.Summary,
			Description: c.Description,
			Color:       c.BackgroundColor,
			IsPrimary:   c.Primary,
		})
	}
	return result, nil
}

func (a *GoogleCalendarAdapter) GetEvents(calendarID string, opts sync.EventQueryOptions) ([]sync.NormalizedEvent, error) {
	ctx := context.Background()
	events, err := a.client.GetCalendarEvents(ctx, calendarID, EventQueryOptions{
		TimeMin:      opts.TimeMin,
		TimeMax:      opts.TimeMax,
		MaxResults:   opts.MaxResults,
		SingleEvents: opts.SingleEvents,
		OrderBy:      opts.OrderBy,
	})
	if err != nil {
		return nil, err
	}
	result := make([]sync.NormalizedEvent, 0, len(events.Items))
	for _, e := range events.Items {
		result = append(result, googleEventToNormalized(e))
	}
	return result, nil
}

func (a *GoogleCalendarAdapter) CreateEvent(calendarID string, event *sync.NormalizedEvent) (*sync.NormalizedEvent, error) {
	ctx := context.Background()
	googleEvent := normalizedToGoogleEvent(event)
	created, err := a.client.CreateEvent(ctx, calendarID, googleEvent)
	if err != nil {
		return nil, err
	}
	e := googleEventToNormalized(created)
	return &e, nil
}

func (a *GoogleCalendarAdapter) UpdateEvent(calendarID, eventID string, event *sync.NormalizedEvent) (*sync.NormalizedEvent, error) {
	ctx := context.Background()
	googleEvent := normalizedToGoogleEvent(event)
	updated, err := a.client.UpdateEvent(ctx, calendarID, eventID, googleEvent)
	if err != nil {
		return nil, err
	}
	e := googleEventToNormalized(updated)
	return &e, nil
}

func (a *GoogleCalendarAdapter) DeleteEvent(calendarID, eventID string) error {
	ctx := context.Background()
	return a.client.DeleteEvent(ctx, calendarID, eventID)
}

func googleEventToNormalized(e *calendar.Event) sync.NormalizedEvent {
	ne := sync.NormalizedEvent{
		ID:          e.Id,
		Title:       e.Summary,
		Description: e.Description,
		Location:    e.Location,
		ExternalID:  e.Id,
	}
	if e.Start != nil {
		if e.Start.DateTime != "" {
			if t, err := parseEventTime(e.Start); err == nil {
				ne.StartTime = t
			}
		} else if e.Start.Date != "" {
			if t, err := parseEventDate(e.Start); err == nil {
				ne.StartTime = t
				ne.IsAllDay = true
			}
		}
	}
	if e.End != nil {
		if t, err := parseEventTime(e.End); err == nil {
			ne.EndTime = t
		} else if t, err := parseEventDate(e.End); err == nil {
			ne.EndTime = t
		}
	}
	if len(e.Recurrence) > 0 {
		ne.IsRecurring = true
		ne.RecurrenceRule = e.Recurrence[0]
	}
	if e.RecurringEventId != "" {
		ne.RecurrenceID = e.RecurringEventId
	}
	return ne
}

func normalizedToGoogleEvent(ne *sync.NormalizedEvent) *calendar.Event {
	e := &calendar.Event{
		Summary:     ne.Title,
		Description: ne.Description,
		Location:    ne.Location,
	}
	if ne.IsAllDay {
		e.Start = &calendar.EventDateTime{Date: ne.StartTime.Format("2006-01-02")}
		e.End = &calendar.EventDateTime{Date: ne.EndTime.Format("2006-01-02")}
	} else {
		e.Start = &calendar.EventDateTime{DateTime: ne.StartTime.Format("2006-01-02T15:04:05Z07:00")}
		e.End = &calendar.EventDateTime{DateTime: ne.EndTime.Format("2006-01-02T15:04:05Z07:00")}
	}
	if ne.IsRecurring && ne.RecurrenceRule != "" {
		e.Recurrence = []string{ne.RecurrenceRule}
	}
	return e
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

func parseEventDate(et *calendar.EventDateTime) (time.Time, error) {
	if et == nil {
		return time.Time{}, fmt.Errorf("event time is nil")
	}
	if et.Date != "" {
		return time.Parse("2006-01-02", et.Date)
	}
	return time.Time{}, fmt.Errorf("no date in event")
}
