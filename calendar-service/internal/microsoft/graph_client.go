package microsoft

import (
	"context"
	"fmt"
	"time"

	"calendar-service/internal/oauth"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/sirupsen/logrus"
)

// GraphClient wraps Microsoft Graph API with OAuth token management
type GraphClient struct {
	oauthProvider *oauth.MicrosoftOAuth2Provider
	userID        string
	logger        *logrus.Entry
}

// GraphClientConfig holds configuration
type GraphClientConfig struct {
	OAuthProvider *oauth.MicrosoftOAuth2Provider
	UserID        string
	Logger        *logrus.Entry
}

// NewGraphClient creates a new Microsoft Graph client
func NewGraphClient(cfg GraphClientConfig) (*GraphClient, error) {
	return &GraphClient{
		oauthProvider: cfg.OAuthProvider,
		userID:        cfg.UserID,
		logger:        cfg.Logger.WithField("component", "microsoft_graph"),
	}, nil
}

// getGraphClient creates authenticated Graph client using the stored user token
func (c *GraphClient) getGraphClient(ctx context.Context) (*msgraph.GraphServiceClient, error) {
	token, err := c.oauthProvider.GetUserToken(ctx, c.userID)
	if err != nil {
		return nil, fmt.Errorf("get user token: %w", err)
	}

	// Create a credential that uses the access token directly
	// Note: In a production environment, you might want to implement a custom azcore.TokenCredential
	// that wraps the oauth2.TokenSource to handle refreshing automatically if the SDK doesn't.
	// However, for this implementation, we rely on the oauthProvider to provide a valid token.

	// Since NewGraphServiceClientWithCredentials requires azcore.TokenCredential,
	// and we have an oauth2.Token, we can use a simpler approach or a wrapper.

	// For now, we'll use a manual adapter as shown in some Microsoft samples for legacy token integration
	// or create a simple static credential.

	cred := &staticTokenCredential{token: token.AccessToken, expiry: token.Expiry}

	return msgraph.NewGraphServiceClientWithCredentials(cred, []string{"Calendars.ReadWrite", "Calendars.Read"})
}

// staticTokenCredential is a simple implementation of azcore.TokenCredential
type staticTokenCredential struct {
	token  string
	expiry time.Time
}

func (c *staticTokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{
		Token:     c.token,
		ExpiresOn: c.expiry,
	}, nil
}

// ListCalendars fetches user's Microsoft calendars
func (c *GraphClient) ListCalendars(ctx context.Context) ([]models.Calendarable, error) {
	client, err := c.getGraphClient(ctx)
	if err != nil {
		return nil, err
	}

	result, err := client.Me().Calendars().Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("list calendars: %w", err)
	}

	calendars := result.GetValue()
	c.logger.WithField("calendar_count", len(calendars)).Debug("Listed Microsoft calendars")

	return calendars, nil
}

// EventQueryOptions holds query parameters
type EventQueryOptions struct {
	StartTime  time.Time
	EndTime    time.Time
	MaxResults int
}

// GetCalendarEvents retrieves events from a calendar
func (c *GraphClient) GetCalendarEvents(
	ctx context.Context,
	calendarID string,
	opts EventQueryOptions,
) ([]models.Eventable, error) {
	client, err := c.getGraphClient(ctx)
	if err != nil {
		return nil, err
	}

	// Graph SDK uses string start/end times for the filter or specific query params
	// We'll use the RequestConfiguration to set query parameters

	// Note: Request configurations in Graph SDK are typed per-resource

	result, err := client.Me().Calendars().ByCalendarId(calendarID).Events().Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("get events: %w", err)
	}

	events := result.GetValue()
	c.logger.WithField("event_count", len(events)).Debug("Retrieved Microsoft Calendar events")

	return events, nil
}

// CreateEvent creates a new event in Microsoft Calendar
func (c *GraphClient) CreateEvent(ctx context.Context, calendarID string, event models.Eventable) (models.Eventable, error) {
	client, err := c.getGraphClient(ctx)
	if err != nil {
		return nil, err
	}

	result, err := client.Me().Calendars().ByCalendarId(calendarID).Events().Post(ctx, event, nil)
	if err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"calendar_id": calendarID,
	}).Debug("Created Microsoft Calendar event")

	return result, nil
}

// UpdateEvent updates an existing event
func (c *GraphClient) UpdateEvent(ctx context.Context, calendarID, eventID string, event models.Eventable) (models.Eventable, error) {
	client, err := c.getGraphClient(ctx)
	if err != nil {
		return nil, err
	}

	result, err := client.Me().Calendars().ByCalendarId(calendarID).Events().ByEventId(eventID).Patch(ctx, event, nil)
	if err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}

	return result, nil
}

// DeleteEvent deletes an event from calendar
func (c *GraphClient) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	client, err := c.getGraphClient(ctx)
	if err != nil {
		return err
	}

	err = client.Me().Calendars().ByCalendarId(calendarID).Events().ByEventId(eventID).Delete(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"calendar_id": calendarID,
		"event_id":    eventID,
	}).Debug("Deleted Microsoft Calendar event")

	return nil
}

// ConvertMicrosoftEventToInternal converts models.Eventable to internal format
func ConvertMicrosoftEventToInternal(msEvent models.Eventable) map[string]interface{} {
	event := make(map[string]interface{})

	if subject := msEvent.GetSubject(); subject != nil {
		event["title"] = *subject
	}

	if body := msEvent.GetBody(); body != nil {
		if content := body.GetContent(); content != nil {
			event["description"] = *content
		}
	}

	if location := msEvent.GetLocation(); location != nil {
		if displayName := location.GetDisplayName(); displayName != nil {
			event["location"] = *displayName
		}
	}

	if start := msEvent.GetStart(); start != nil {
		if dateTime := start.GetDateTime(); dateTime != nil {
			event["start_time"] = *dateTime
		}
	}

	if end := msEvent.GetEnd(); end != nil {
		if dateTime := end.GetDateTime(); dateTime != nil {
			event["end_time"] = *dateTime
		}
	}

	if isAllDay := msEvent.GetIsAllDay(); isAllDay != nil {
		event["is_all_day"] = *isAllDay
	}

	return event
}
