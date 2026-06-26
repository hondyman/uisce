package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	GraphAPIEndpoint = "https://graph.microsoft.com/v1.0"
)

type CalendarClient struct {
	client  *http.Client
	logger  *logrus.Entry
	baseURL string
}

// GraphCalendarList represents a list of MS calendars
type GraphCalendarList struct {
	Value []*GraphCalendar `json:"value"`
}

// GraphCalendar represents a MS calendar instance
type GraphCalendar struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Color               string `json:"color"`
	IsDefaultCalendar   bool   `json:"isDefaultCalendar"`
	CanShare            bool   `json:"canShare"`
	CanViewPrivateItems bool   `json:"canViewPrivateItems"`
	CanEdit             bool   `json:"canEdit"`
}

// GraphEventList represents a list of MS events
type GraphEventList struct {
	Value    []*GraphEvent `json:"value"`
	NextLink string        `json:"@odata.nextLink"`
}

// GraphEvent represents a MS event instance
type GraphEvent struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Body    struct {
		ContentType string `json:"contentType"`
		Content     string `json:"content"`
	} `json:"body"`
	Start                DateTimeTimeZone `json:"start"`
	End                  DateTimeTimeZone `json:"end"`
	Location             Location         `json:"location"`
	IsAllDay             bool             `json:"isAllDay"`
	IsCancelled          bool             `json:"isCancelled"`
	CreatedDateTime      string           `json:"createdDateTime"`
	LastModifiedDateTime string           `json:"lastModifiedDateTime"`
	Recurrence           interface{}      `json:"recurrence"`
}

type DateTimeTimeZone struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

type Location struct {
	DisplayName string `json:"displayName"`
}

// NewCalendarClient creates a new MS Graph calendar client
func NewCalendarClient(ctx context.Context, token *oauth2.Token) (*CalendarClient, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	return &CalendarClient{
		client:  client,
		logger:  logrus.WithField("component", "microsoft_calendar_client"),
		baseURL: GraphAPIEndpoint,
	}, nil
}

// ListCalendars fetches the authenticated user's calendars
func (c *CalendarClient) ListCalendars(ctx context.Context) ([]*GraphCalendar, error) {
	url := fmt.Sprintf("%s/me/calendars", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("microsoft graph api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result GraphCalendarList
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Value, nil
}

// GetCalendarEvents fetches the authenticated user's calendar events
func (c *CalendarClient) GetCalendarEvents(ctx context.Context, calendarID string, nextLink string) (*GraphEventList, error) {
	url := fmt.Sprintf("%s/me/calendars/%s/events", c.baseURL, calendarID)

	// If pagination link is supplied, use it explicitly instead of rebuilding
	if nextLink != "" {
		url = nextLink
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("microsoft graph api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result GraphEventList
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateEvent pushes a new event to the Microsoft calendar
func (c *CalendarClient) CreateEvent(ctx context.Context, calendarID string, event *GraphEvent) (*GraphEvent, error) {
	url := fmt.Sprintf("%s/me/calendars/%s/events", c.baseURL, calendarID)

	bodyData, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("microsoft graph api create error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var created GraphEvent
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, err
	}

	return &created, nil
}

// UpdateEvent pushes an update to a Microsoft calendar event
func (c *CalendarClient) UpdateEvent(ctx context.Context, calendarID string, eventID string, event *GraphEvent) (*GraphEvent, error) {
	url := fmt.Sprintf("%s/me/calendars/%s/events/%s", c.baseURL, calendarID, eventID)

	bodyData, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("microsoft graph api update error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var updated GraphEvent
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		return nil, err
	}

	return &updated, nil
}

// DeleteEvent deletes a Microsoft calendar event
func (c *CalendarClient) DeleteEvent(ctx context.Context, calendarID string, eventID string) error {
	url := fmt.Sprintf("%s/me/calendars/%s/events/%s", c.baseURL, calendarID, eventID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("microsoft graph api delete error: status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}
