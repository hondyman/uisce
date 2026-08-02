package sync

import (
	"time"
)

type Provider string

const (
	ProviderGoogle    Provider = "google"
	ProviderMicrosoft Provider = "microsoft"
	ProviderApple     Provider = "apple"
)

type CalendarInfo struct {
	ID          string
	Name        string
	Description string
	Color       string
	IsPrimary   bool
}

type NormalizedEvent struct {
	ID             string
	Title          string
	Description    string
	Location       string
	StartTime      time.Time
	EndTime        time.Time
	IsAllDay       bool
	IsRecurring    bool
	RecurrenceRule string
	RecurrenceID   string
	ExternalID     string
}

type EventQueryOptions struct {
	TimeMin      time.Time
	TimeMax      time.Time
	MaxResults   int
	SingleEvents bool
	OrderBy      string
}

type CalendarClient interface {
	ListCalendars() ([]CalendarInfo, error)
	GetEvents(calendarID string, opts EventQueryOptions) ([]NormalizedEvent, error)
	CreateEvent(calendarID string, event *NormalizedEvent) (*NormalizedEvent, error)
	UpdateEvent(calendarID, eventID string, event *NormalizedEvent) (*NormalizedEvent, error)
	DeleteEvent(calendarID, eventID string) error
}
