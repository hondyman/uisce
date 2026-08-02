package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hondyman/uisce/backend/internal/sync"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const GraphAPIEndpoint = "https://graph.microsoft.com/v1.0"

type CalendarClient struct {
	token  *oauth2.Token
	client *http.Client
	logger *logrus.Entry
}

type GraphCalendar struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Color             string `json:"color"`
	IsDefaultCalendar bool   `json:"isDefaultCalendar"`
}

type GraphEvent struct {
	ID            string            `json:"id"`
	Subject       string            `json:"subject"`
	Body          Body               `json:"body"`
	Start         DateTimeTimeZone   `json:"start"`
	End           DateTimeTimeZone   `json:"end"`
	Location      Location           `json:"location"`
	IsAllDay      bool               `json:"isAllDay"`
	IsCancelled   bool               `json:"isCancelled"`
	Recurrence    interface{}        `json:"recurrence"`
}

type Body struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type DateTimeTimeZone struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

type Location struct {
	DisplayName string `json:"displayName"`
}

func NewCalendarClient(token *oauth2.Token) *CalendarClient {
	return &CalendarClient{
		token: token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logrus.WithField("component", "microsoft_calendar_client"),
	}
}

func (c *CalendarClient) ListCalendars() ([]sync.CalendarInfo, error) {
	ctx := context.Background()
	url := fmt.Sprintf("%s/me/calendars", GraphAPIEndpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("microsoft graph api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Value []struct {
			ID                string `json:"id"`
			Name              string `json:"name"`
			Color             string `json:"color"`
			IsDefaultCalendar bool   `json:"isDefaultCalendar"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	infos := make([]sync.CalendarInfo, 0, len(result.Value))
	for _, cal := range result.Value {
		infos = append(infos, sync.CalendarInfo{
			ID:        cal.ID,
			Name:      cal.Name,
			Color:     cal.Color,
			IsPrimary: cal.IsDefaultCalendar,
		})
	}
	return infos, nil
}

func (c *CalendarClient) GetEvents(calendarID string, opts sync.EventQueryOptions) ([]sync.NormalizedEvent, error) {
	ctx := context.Background()
	url := fmt.Sprintf("%s/me/calendars/%s/events", GraphAPIEndpoint, calendarID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	if !opts.TimeMin.IsZero() {
		q.Add("startDateTime", opts.TimeMin.Format(time.RFC3339))
	}
	if !opts.TimeMax.IsZero() {
		q.Add("endDateTime", opts.TimeMax.Format(time.RFC3339))
	}
	if opts.MaxResults > 0 {
		q.Add("$top", fmt.Sprintf("%d", opts.MaxResults))
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("microsoft graph api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Value []GraphEvent `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	events := make([]sync.NormalizedEvent, 0, len(result.Value))
	for _, e := range result.Value {
		events = append(events, graphEventToNormalized(&e))
	}
	return events, nil
}

func (c *CalendarClient) CreateEvent(calendarID string, event *sync.NormalizedEvent) (*sync.NormalizedEvent, error) {
	ctx := context.Background()
	url := fmt.Sprintf("%s/me/calendars/%s/events", GraphAPIEndpoint, calendarID)

	ge := normalizedToGraphEvent(event)
	bodyData, err := json.Marshal(ge)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
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

	e := graphEventToNormalized(&created)
	return &e, nil
}

func (c *CalendarClient) UpdateEvent(calendarID, eventID string, event *sync.NormalizedEvent) (*sync.NormalizedEvent, error) {
	ctx := context.Background()
	url := fmt.Sprintf("%s/me/calendars/%s/events/%s", GraphAPIEndpoint, calendarID, eventID)

	ge := normalizedToGraphEvent(event)
	bodyData, err := json.Marshal(ge)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
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

	e := graphEventToNormalized(&updated)
	return &e, nil
}

func (c *CalendarClient) DeleteEvent(calendarID, eventID string) error {
	ctx := context.Background()
	url := fmt.Sprintf("%s/me/calendars/%s/events/%s", GraphAPIEndpoint, calendarID, eventID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)

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

func graphEventToNormalized(e *GraphEvent) sync.NormalizedEvent {
	ne := sync.NormalizedEvent{
		ID:          e.ID,
		Title:       e.Subject,
		Description: e.Body.Content,
		Location:    e.Location.DisplayName,
		ExternalID:  e.ID,
		IsAllDay:    e.IsAllDay,
	}
	if e.Start.DateTime != "" {
		if t, err := time.Parse(time.RFC3339, e.Start.DateTime); err == nil {
			ne.StartTime = t
		}
	}
	if e.End.DateTime != "" {
		if t, err := time.Parse(time.RFC3339, e.End.DateTime); err == nil {
			ne.EndTime = t
		}
	}
	return ne
}

func normalizedToGraphEvent(ne *sync.NormalizedEvent) *GraphEvent {
	e := &GraphEvent{
		Subject: ne.Title,
		Body: Body{
			ContentType: "text",
			Content:     ne.Description,
		},
	}
	if ne.IsAllDay {
		e.Start = DateTimeTimeZone{DateTime: ne.StartTime.Format("2006-01-02"), TimeZone: "UTC"}
		e.End = DateTimeTimeZone{DateTime: ne.EndTime.Format("2006-01-02"), TimeZone: "UTC"}
	} else {
		e.Start = DateTimeTimeZone{DateTime: ne.StartTime.Format(time.RFC3339), TimeZone: "UTC"}
		e.End = DateTimeTimeZone{DateTime: ne.EndTime.Format(time.RFC3339), TimeZone: "UTC"}
	}
	if ne.Location != "" {
		e.Location = Location{DisplayName: ne.Location}
	}
	return e
}
