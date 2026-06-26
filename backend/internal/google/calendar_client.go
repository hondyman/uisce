package google

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hondyman/semlayer/backend/internal/oauth"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// Prometheus metrics for Google Calendar API
// Prometheus metrics variables are now centralized, or we can use the ones defined here if they don't collide.
// Metrics definitions here are fine as they are package private to google.
var (
	googleAPICallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "google_calendar_api_calls_total",
			Help: "Total number of Google Calendar API calls",
		},
		[]string{"method", "status"},
	)

	googleAPILatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "google_calendar_api_latency_seconds",
			Help:    "Latency of Google Calendar API calls",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	googleAPIErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "google_calendar_api_errors_total",
			Help: "Total number of Google Calendar API errors",
		},
		[]string{"method", "error_type"},
	)
)

// CalendarClient wraps the Google Calendar API with OAuth token management
type CalendarClient struct {
	oauthProvider *oauth.GoogleOAuth2Provider
	userID        string
	tenantID      string
	logger        *logrus.Entry
	httpClient    *http.Client
	maxRetries    int
	retryDelay    time.Duration
}

// CalendarClientConfig holds configuration for CalendarClient
type CalendarClientConfig struct {
	OAuthProvider *oauth.GoogleOAuth2Provider
	UserID        string
	TenantID      string
	Logger        *logrus.Entry
	MaxRetries    int
	RetryDelay    time.Duration
}

// NewCalendarClient creates a new client for a specific user
func NewCalendarClient(cfg CalendarClientConfig) (*CalendarClient, error) {
	client := &CalendarClient{
		oauthProvider: cfg.OAuthProvider,
		userID:        cfg.UserID,
		tenantID:      cfg.TenantID,
		logger:        cfg.Logger.WithField("component", "google_calendar_client"),
		maxRetries:    cfg.MaxRetries,
		retryDelay:    cfg.RetryDelay,
	}

	if client.maxRetries == 0 {
		client.maxRetries = 3
	}
	if client.retryDelay == 0 {
		client.retryDelay = 1 * time.Second
	}

	return client, nil
}

// getHTTPClient creates an HTTP client with OAuth2 token (with auto-refresh)
func (c *CalendarClient) getHTTPClient(ctx context.Context) (*http.Client, error) {
	token, err := c.oauthProvider.GetUserToken(ctx, c.userID)
	if err != nil {
		googleAPIErrors.WithLabelValues("get_token", "auth_error").Inc()
		return nil, fmt.Errorf("get user token: %w", err)
	}

	// Create OAuth2 HTTP client (token source handles refresh automatically)
	oauth2Client := c.oauthProvider.Config().Client(ctx, token)

	// Add custom transport for metrics and retry logic
	oauth2Client.Transport = &metricsTransport{
		base:   oauth2Client.Transport,
		logger: c.logger,
	}

	c.httpClient = oauth2Client
	return oauth2Client, nil
}

// getCalendarService creates a Calendar API service instance with retry logic
func (c *CalendarClient) getCalendarService(ctx context.Context) (*calendar.Service, error) {
	httpClient, err := c.getHTTPClient(ctx)
	if err != nil {
		return nil, err
	}

	svc, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create calendar service: %w", err)
	}

	return svc, nil
}

// ListCalendars fetches the user's Google Calendars with pagination
func (c *CalendarClient) ListCalendars(ctx context.Context) ([]*calendar.CalendarListEntry, error) {
	startTime := time.Now()

	svc, err := c.getCalendarService(ctx)
	if err != nil {
		return nil, err
	}

	var allCalendars []*calendar.CalendarListEntry
	pageToken := ""
	attempts := 0

	for {
		attempts++
		call := svc.CalendarList.List().PageToken(pageToken)

		result, err := call.Do()
		googleAPICallsTotal.WithLabelValues("list_calendars", getStatus(err)).Inc()
		googleAPILatency.WithLabelValues("list_calendars").Observe(time.Since(startTime).Seconds())

		if err != nil {
			if isRetryableError(err) && attempts < c.maxRetries {
				c.logger.WithError(err).WithField("attempt", attempts).Warn("Retrying ListCalendars")
				time.Sleep(c.retryDelay * time.Duration(attempts))
				continue
			}
			googleAPIErrors.WithLabelValues("list_calendars", getErrorType(err)).Inc()
			return nil, fmt.Errorf("list calendars: %w", err)
		}

		allCalendars = append(allCalendars, result.Items...)

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	c.logger.WithFields(logrus.Fields{
		"user_id":     c.userID,
		"calendars":   len(allCalendars),
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Listed Google Calendars")

	return allCalendars, nil
}

// GetCalendarEvents retrieves events from a specific calendar with options
func (c *CalendarClient) GetCalendarEvents(
	ctx context.Context,
	calendarID string,
	opts EventQueryOptions,
) (*calendar.Events, error) {
	startTime := time.Now()

	svc, err := c.getCalendarService(ctx)
	if err != nil {
		return nil, err
	}

	call := svc.Events.List(calendarID)

	// Apply query options
	if !opts.TimeMin.IsZero() {
		call = call.TimeMin(opts.TimeMin.Format(time.RFC3339))
	}
	if !opts.TimeMax.IsZero() {
		call = call.TimeMax(opts.TimeMax.Format(time.RFC3339))
	}
	if opts.MaxResults > 0 {
		call = call.MaxResults(int64(opts.MaxResults))
	}
	if opts.SingleEvents {
		call = call.SingleEvents(true)
	}
	if opts.OrderBy != "" {
		call = call.OrderBy(opts.OrderBy)
	}
	if opts.ShowDeleted {
		call = call.ShowDeleted(true)
	}

	attempts := 0
	var result *calendar.Events

	for {
		attempts++
		result, err = call.Do()
		googleAPICallsTotal.WithLabelValues("get_events", getStatus(err)).Inc()
		googleAPILatency.WithLabelValues("get_events").Observe(time.Since(startTime).Seconds())

		if err != nil {
			if isRetryableError(err) && attempts < c.maxRetries {
				c.logger.WithError(err).WithField("attempt", attempts).Warn("Retrying GetCalendarEvents")
				time.Sleep(c.retryDelay * time.Duration(attempts))
				continue
			}
			googleAPIErrors.WithLabelValues("get_events", getErrorType(err)).Inc()
			return nil, fmt.Errorf("get events: %w", err)
		}

		break
	}

	c.logger.WithFields(logrus.Fields{
		"user_id":      c.userID,
		"calendar_id":  calendarID,
		"events_count": len(result.Items),
		"duration_ms":  time.Since(startTime).Milliseconds(),
	}).Debug("Retrieved Google Calendar events")

	return result, nil
}

// CreateEvent creates a new event in the specified calendar
func (c *CalendarClient) CreateEvent(ctx context.Context, calendarID string, event *calendar.Event) (*calendar.Event, error) {
	startTime := time.Now()
	svc, err := c.getCalendarService(ctx)
	if err != nil {
		return nil, err
	}

	var result *calendar.Event
	attempts := 0

	for {
		attempts++
		call := svc.Events.Insert(calendarID, event)
		result, err = call.Do()

		googleAPICallsTotal.WithLabelValues("create_event", getStatus(err)).Inc()
		googleAPILatency.WithLabelValues("create_event").Observe(time.Since(startTime).Seconds())

		if err != nil {
			if isRetryableError(err) && attempts < c.maxRetries {
				c.logger.WithError(err).WithField("attempt", attempts).Warn("Retrying CreateEvent")
				time.Sleep(c.retryDelay * time.Duration(attempts))
				continue
			}
			googleAPIErrors.WithLabelValues("create_event", getErrorType(err)).Inc()
			return nil, fmt.Errorf("create event: %w", err)
		}
		break
	}

	return result, nil
}

// UpdateEvent updates an existing event
func (c *CalendarClient) UpdateEvent(ctx context.Context, calendarID, eventID string, event *calendar.Event) (*calendar.Event, error) {
	startTime := time.Now()
	svc, err := c.getCalendarService(ctx)
	if err != nil {
		return nil, err
	}

	var result *calendar.Event
	attempts := 0

	for {
		attempts++
		call := svc.Events.Update(calendarID, eventID, event)
		result, err = call.Do()

		googleAPICallsTotal.WithLabelValues("update_event", getStatus(err)).Inc()
		googleAPILatency.WithLabelValues("update_event").Observe(time.Since(startTime).Seconds())

		if err != nil {
			if isRetryableError(err) && attempts < c.maxRetries {
				c.logger.WithError(err).WithField("attempt", attempts).Warn("Retrying UpdateEvent")
				time.Sleep(c.retryDelay * time.Duration(attempts))
				continue
			}
			googleAPIErrors.WithLabelValues("update_event", getErrorType(err)).Inc()
			return nil, fmt.Errorf("update event: %w", err)
		}
		break
	}

	return result, nil
}

// DeleteEvent deletes an event
func (c *CalendarClient) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	startTime := time.Now()
	svc, err := c.getCalendarService(ctx)
	if err != nil {
		return err
	}

	attempts := 0
	for {
		attempts++
		call := svc.Events.Delete(calendarID, eventID)
		err = call.Do()

		googleAPICallsTotal.WithLabelValues("delete_event", getStatus(err)).Inc()
		googleAPILatency.WithLabelValues("delete_event").Observe(time.Since(startTime).Seconds())

		if err != nil {
			if isRetryableError(err) && attempts < c.maxRetries {
				c.logger.WithError(err).WithField("attempt", attempts).Warn("Retrying DeleteEvent")
				time.Sleep(c.retryDelay * time.Duration(attempts))
				continue
			}
			googleAPIErrors.WithLabelValues("delete_event", getErrorType(err)).Inc()
			return fmt.Errorf("delete event: %w", err)
		}
		break
	}

	return nil
}

// EventQueryOptions holds query parameters for event listing
type EventQueryOptions struct {
	TimeMin      time.Time
	TimeMax      time.Time
	MaxResults   int
	SingleEvents bool
	OrderBy      string // "startTime", "updated", etc.
	ShowDeleted  bool
}

// metricsTransport wraps http.RoundTripper to add metrics
type metricsTransport struct {
	base   http.RoundTripper
	logger *logrus.Entry
}

func (t *metricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	if t.base == nil {
		t.base = http.DefaultTransport
	}
	resp, err := t.base.RoundTrip(req)

	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	if resp != nil {
		status = fmt.Sprintf("%d", resp.StatusCode)
	}

	// Record metrics
	googleAPILatency.WithLabelValues(status).Observe(duration)

	return resp, err
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	gerr, ok := err.(*googleapi.Error)
	if !ok {
		// Network errors are retryable
		return true
	}

	// Retry on 5xx server errors and 429 rate limit
	return gerr.Code >= 500 || gerr.Code == 429
}

// getStatus returns HTTP status code as string
func getStatus(err error) string {
	if err == nil {
		return "200"
	}

	gerr, ok := err.(*googleapi.Error)
	if ok {
		return fmt.Sprintf("%d", gerr.Code)
	}

	return "error"
}

// getErrorType categorizes error for metrics
func getErrorType(err error) string {
	if err == nil {
		return "none"
	}

	gerr, ok := err.(*googleapi.Error)
	if ok {
		if gerr.Code >= 500 {
			return "server_error"
		}
		if gerr.Code == 429 {
			return "rate_limit"
		}
		if gerr.Code >= 400 {
			return "client_error"
		}
	}

	return "unknown"
}
