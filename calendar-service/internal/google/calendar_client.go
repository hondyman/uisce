package google

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type CalendarClient struct {
	service    *calendar.Service
	rpsLimit   int
	maxRetries int
	retryDelay time.Duration
	userID     string
	logger     *logrus.Entry
}

type CalendarClientConfig struct {
	OAuthProvider interface{} // Used to resolve user token if needed, or pass token directly
	UserID        string
	TenantID      string
	Logger        *logrus.Entry
}

func NewCalendarClient(ctx context.Context, token *oauth2.Token, rpsLimit int) (*CalendarClient, error) {
	service, err := calendar.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(token)))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	return &CalendarClient{
		service:    service,
		rpsLimit:   rpsLimit,
		maxRetries: 3,
		retryDelay: 1 * time.Second,
		logger:     logrus.WithField("component", "google_calendar_client"),
	}, nil
}

// ListCalendars lists all calendars for the authenticated user
func (c *CalendarClient) ListCalendars(ctx context.Context) ([]*calendar.CalendarListEntry, error) {
	call := c.service.CalendarList.List()
	result, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	return result.Items, nil
}

// GetCalendarEvents retrieves events from a specific calendar
func (c *CalendarClient) GetCalendarEvents(ctx context.Context, calendarID string, pageToken string) (*calendar.Events, error) {
	call := c.service.Events.List(calendarID)
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	result, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar events: %w", err)
	}

	return result, nil
}

func (c *CalendarClient) getCalendarService(ctx context.Context) (*calendar.Service, error) {
	return c.service, nil
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if error is a Google API error
	var googleErr *googleapi.Error
	if errors.As(err, &googleErr) {
		// Retry on 429 Too Many Requests, 500, 502, 503, 504
		if googleErr.Code == 429 || googleErr.Code >= 500 {
			return true
		}
	}

	// Check for network errors like timeouts or connection resets
	errString := err.Error()
	if strings.Contains(errString, "timeout") ||
		strings.Contains(errString, "connection reset") ||
		strings.Contains(errString, "context deadline exceeded") {
		return true
	}

	return false
}

func getStatus(err error) string {
	if err != nil {
		return "error"
	}
	return "success"
}

func getErrorType(err error) string {
	if err == nil {
		return "none"
	}
	return "unknown"
}

// CreateEvent creates a new event in Google Calendar
func (c *CalendarClient) CreateEvent(ctx context.Context, calendarID string, event *calendar.Event) (*calendar.Event, error) {
	startTime := time.Now()

	svc, err := c.getCalendarService(ctx)
	if err != nil {
		return nil, err
	}

	attempts := 0
	var createdEvent *calendar.Event

	for {
		attempts++
		createdEvent, err = svc.Events.Insert(calendarID, event).Do()

		if err != nil {
			if isRetryableError(err) && attempts < c.maxRetries {
				c.logger.WithError(err).WithField("attempt", attempts).Warn("Retrying CreateEvent")
				time.Sleep(c.retryDelay * time.Duration(attempts))
				continue
			}
			return nil, fmt.Errorf("create event: %w", err)
		}

		break
	}

	c.logger.WithFields(logrus.Fields{
		"user_id":     c.userID,
		"calendar_id": calendarID,
		"event_id":    createdEvent.Id,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Debug("Created Google Calendar event")

	return createdEvent, nil
}

// UpdateEvent updates an existing event in Google Calendar
func (c *CalendarClient) UpdateEvent(ctx context.Context, calendarID, eventID string, event *calendar.Event) (*calendar.Event, error) {
	startTime := time.Now()

	svc, err := c.getCalendarService(ctx)
	if err != nil {
		return nil, err
	}

	attempts := 0
	var updatedEvent *calendar.Event

	for {
		attempts++
		updatedEvent, err = svc.Events.Update(calendarID, eventID, event).Do()

		if err != nil {
			if isRetryableError(err) && attempts < c.maxRetries {
				c.logger.WithError(err).WithField("attempt", attempts).Warn("Retrying UpdateEvent")
				time.Sleep(c.retryDelay * time.Duration(attempts))
				continue
			}
			return nil, fmt.Errorf("update event: %w", err)
		}

		break
	}

	c.logger.WithFields(logrus.Fields{
		"user_id":     c.userID,
		"calendar_id": calendarID,
		"event_id":    eventID,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Debug("Updated Google Calendar event")

	return updatedEvent, nil
}

// DeleteEvent deletes an event from Google Calendar
func (c *CalendarClient) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	startTime := time.Now()

	svc, err := c.getCalendarService(ctx)
	if err != nil {
		return err
	}

	attempts := 0

	for {
		attempts++
		err = svc.Events.Delete(calendarID, eventID).Do()

		if err != nil {
			if isRetryableError(err) && attempts < c.maxRetries {
				c.logger.WithError(err).WithField("attempt", attempts).Warn("Retrying DeleteEvent")
				time.Sleep(c.retryDelay * time.Duration(attempts))
				continue
			}
			return fmt.Errorf("delete event: %w", err)
		}

		break
	}

	c.logger.WithFields(logrus.Fields{
		"user_id":     c.userID,
		"calendar_id": calendarID,
		"event_id":    eventID,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Debug("Deleted Google Calendar event")

	return nil
}

// GetEvent retrieves a single event from Google Calendar
func (c *CalendarClient) GetEvent(ctx context.Context, calendarID, eventID string) (*calendar.Event, error) {
	svc, err := c.getCalendarService(ctx)
	if err != nil {
		return nil, err
	}

	event, err := svc.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	return event, nil
}

// ListEvents retrieves events from a given calendar within a time range
func (c *CalendarClient) ListEvents(ctx context.Context, calendarID string, timeMin, timeMax time.Time, maxResults int64) ([]*calendar.Event, error) {
	svc, err := c.getCalendarService(ctx)
	if err != nil {
		return nil, err
	}

	eventsList, err := svc.Events.List(calendarID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(timeMin.Format(time.RFC3339)).
		TimeMax(timeMax.Format(time.RFC3339)).
		MaxResults(maxResults).
		OrderBy("startTime").Do()

	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	return eventsList.Items, nil
}

// Close cleans up resources
func (c *CalendarClient) Close() error {
	// Calendar service doesn't require explicit cleanup
	return nil
}
