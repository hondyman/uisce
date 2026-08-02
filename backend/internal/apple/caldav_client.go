package apple

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hondyman/uisce/backend/internal/sync"
	"github.com/sirupsen/logrus"
)

type CalendarClient struct {
	baseURL  string
	username string
	password string
	client   *http.Client
	logger   *logrus.Entry
}

type CalDAVClientConfig struct {
	BaseURL  string
	Username string
	Password string
}

func NewCalendarClient(cfg CalDAVClientConfig) *CalendarClient {
	return &CalendarClient{
		baseURL:  cfg.BaseURL,
		username: cfg.Username,
		password: cfg.Password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logrus.WithField("component", "apple_caldav_client"),
	}
}

func (c *CalendarClient) ListCalendars() ([]sync.CalendarInfo, error) {
	ctx := context.Background()
	propfind := `<?xml version="1.0" encoding="UTF-8"?>
<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
    <D:prop>
        <D:displayname/>
        <C:calendar-description/>
        <C:calendar-color/>
    </D:prop>
</D:propfind>`

	req, err := http.NewRequestWithContext(ctx, "PROPFIND", c.baseURL, bytes.NewReader([]byte(propfind)))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("caldav unexpected status: %d body=%s", resp.StatusCode, string(body))
	}

	_ /*body*/, _ = io.ReadAll(resp.Body)

	return []sync.CalendarInfo{}, nil
}

func (c *CalendarClient) GetEvents(calendarID string, opts sync.EventQueryOptions) ([]sync.NormalizedEvent, error) {
	ctx := context.Background()
	report := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
    <D:prop>
        <D:getetag/>
        <C:calendar-data/>
    </D:prop>
    <C:filter>
        <C:comp-filter name="VCALENDAR">
            <C:comp-filter name="VEVENT">
                <C:time-range start="%s" end="%s"/>
            </C:comp-filter>
        </C:comp-filter>
    </C:filter>
</C:calendar-query>`,
		opts.TimeMin.Format("20060102T150400Z"),
		opts.TimeMax.Format("20060102T150400Z"))

	req, err := http.NewRequestWithContext(ctx, "REPORT", calendarID, bytes.NewReader([]byte(report)))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/xml")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("caldav unexpected status: %d body=%s", resp.StatusCode, string(body))
	}

	_ /*body*/, _ = io.ReadAll(resp.Body)

	return []sync.NormalizedEvent{}, nil
}

func (c *CalendarClient) CreateEvent(calendarID string, event *sync.NormalizedEvent) (*sync.NormalizedEvent, error) {
	ctx := context.Background()
	ical := c.generateICal(event)
	eventURL := fmt.Sprintf("%s/%s.ics", calendarID, event.ID)

	req, err := http.NewRequestWithContext(ctx, "PUT", eventURL, bytes.NewReader([]byte(ical)))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "text/calendar")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("caldav create unexpected status: %d body=%s", resp.StatusCode, string(body))
	}

	return event, nil
}

func (c *CalendarClient) UpdateEvent(calendarID, eventID string, event *sync.NormalizedEvent) (*sync.NormalizedEvent, error) {
	return c.CreateEvent(calendarID, event)
}

func (c *CalendarClient) DeleteEvent(calendarID, eventID string) error {
	ctx := context.Background()
	eventURL := fmt.Sprintf("%s/%s.ics", calendarID, eventID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", eventURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caldav delete unexpected status: %d body=%s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *CalendarClient) generateICal(event *sync.NormalizedEvent) string {
	return fmt.Sprintf(`BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Calendar Sync//EN
BEGIN:VEVENT
UID:%s
DTSTART:%s
DTEND:%s
SUMMARY:%s
DESCRIPTION:%s
LOCATION:%s
END:VEVENT
END:VCALENDAR`,
		event.ID,
		event.StartTime.Format("20060102T150405Z"),
		event.EndTime.Format("20060102T150405Z"),
		event.Title,
		event.Description,
		event.Location,
	)
}
