package clientportal

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error
}

// Preferences represents client portal preferences
type Preferences struct {
	PreferenceID        uuid.UUID       `db:"preference_id" json:"preference_id"`
	ClientID            uuid.UUID       `db:"client_id" json:"client_id"`
	TenantID            uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	DashboardLayout     json.RawMessage `db:"dashboard_layout" json:"dashboard_layout"`
	EnabledWidgets      []string        `db:"enabled_widgets" json:"enabled_widgets"`
	Theme               string          `db:"theme" json:"theme"`
	AccentColor         string          `db:"accent_color" json:"accent_color"`
	CompactMode         bool            `db:"compact_mode" json:"compact_mode"`
	Language            string          `db:"language" json:"language"`
	Currency            string          `db:"currency" json:"currency"`
	Timezone            string          `db:"timezone" json:"timezone"`
	DateFormat          string          `db:"date_format" json:"date_format"`
	EmailNotifications  bool            `db:"email_notifications" json:"email_notifications"`
	SMSNotifications    bool            `db:"sms_notifications" json:"sms_notifications"`
	PushNotifications   bool            `db:"push_notifications" json:"push_notifications"`
	DataRefreshInterval int             `db:"data_refresh_interval" json:"data_refresh_interval"`
	CreatedAt           time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at" json:"updated_at"`
}

// AnalyticsEvent represents a portal analytics event
type AnalyticsEvent struct {
	EventID        uuid.UUID       `db:"event_id" json:"event_id"`
	ClientID       uuid.UUID       `db:"client_id" json:"client_id"`
	TenantID       uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	EventType      string          `db:"event_type" json:"event_type"`
	EventData      json.RawMessage `db:"event_data" json:"event_data"`
	SessionID      *uuid.UUID      `db:"session_id" json:"session_id,omitempty"`
	PageURL        *string         `db:"page_url" json:"page_url,omitempty"`
	DeviceType     *string         `db:"device_type" json:"device_type,omitempty"`
	DeviceOS       *string         `db:"device_os" json:"device_os,omitempty"`
	Browser        *string         `db:"browser" json:"browser,omitempty"`
	BrowserVersion *string         `db:"browser_version" json:"browser_version,omitempty"`
	IPAddress      *string         `db:"ip_address" json:"ip_address,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}

// Service provides client portal operations
type Service struct {
	db           *sqlx.DB
	hasuraClient HasuraClient
}

// NewService creates a new client portal service
func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

// NewServiceWithHasura creates a new client portal service with Hasura support
func NewServiceWithHasura(db *sqlx.DB, hasuraClient HasuraClient) *Service {
	return &Service{db: db, hasuraClient: hasuraClient}
}

// GetPreferences retrieves portal preferences for a client
func (s *Service) GetPreferences(ctx context.Context, clientID uuid.UUID) (*Preferences, error) {
	prefs, err := s.getPreferencesRecord(ctx, clientID)
	if err == sql.ErrNoRows {
		// Initialize default preferences
		return s.InitializePreferences(ctx, clientID)
	}

	return prefs, err
}

// InitializePreferences creates default preferences for a new client
func (s *Service) InitializePreferences(ctx context.Context, clientID uuid.UUID) (*Preferences, error) {
	// Get tenant ID from client
	tenantID, err := s.getClientTenantIDRecord(ctx, clientID)
	if err != nil {
		return nil, err
	}

	_, err = s.initializePreferencesRecord(ctx, clientID, tenantID)
	if err != nil {
		return nil, err
	}

	return s.GetPreferences(ctx, clientID)
}

// UpdatePreferences updates portal preferences
func (s *Service) UpdatePreferences(ctx context.Context, clientID uuid.UUID, updates map[string]interface{}) error {
	return s.updatePreferencesRecord(ctx, clientID, updates)
}

// TrackEvent records a portal analytics event
func (s *Service) TrackEvent(ctx context.Context, event *AnalyticsEvent) error {
	return s.trackEventRecord(ctx, event)
}

// GetEngagementMetrics retrieves engagement metrics for a client
func (s *Service) GetEngagementMetrics(ctx context.Context, clientID uuid.UUID, days int) (map[string]interface{}, error) {
	return s.getEngagementMetricsRecord(ctx, clientID, days)
}

// Helper methods for SQL operations with Hasura fallback

// getPreferencesRecord retrieves portal preferences for a client
// TODO: Implement Hasura GraphQL query
// SQL fallback: GetContext SELECT * from client_portal_preferences
func (s *Service) getPreferencesRecord(ctx context.Context, clientID uuid.UUID) (*Preferences, error) {
	var prefs Preferences
	query := `
		SELECT * FROM client_portal_preferences
		WHERE client_id = $1
	`
	err := s.db.GetContext(ctx, &prefs, query, clientID)
	return &prefs, err
}

// getClientTenantIDRecord retrieves tenant ID for a client
// TODO: Implement Hasura GraphQL query
// SQL fallback: GetContext SELECT tenant_id from clients
func (s *Service) getClientTenantIDRecord(ctx context.Context, clientID uuid.UUID) (uuid.UUID, error) {
	var tenantID uuid.UUID
	query := `SELECT tenant_id FROM clients WHERE client_id = $1`
	err := s.db.GetContext(ctx, &tenantID, query, clientID)
	return tenantID, err
}

// initializePreferencesRecord creates default preferences using stored procedure
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: Stored procedure initialize_client_portal_preferences
func (s *Service) initializePreferencesRecord(ctx context.Context, clientID, tenantID uuid.UUID) (uuid.UUID, error) {
	var preferenceID uuid.UUID
	query := `SELECT initialize_client_portal_preferences($1, $2)`
	err := s.db.GetContext(ctx, &preferenceID, query, clientID, tenantID)
	return preferenceID, err
}

// updatePreferencesRecord updates portal preferences with dynamic fields
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: Dynamic UPDATE with programmatic SET clauses based on updates map
func (s *Service) updatePreferencesRecord(ctx context.Context, clientID uuid.UUID, updates map[string]interface{}) error {
	query := `
		UPDATE client_portal_preferences
		SET updated_at = NOW()
	`

	args := []interface{}{clientID}
	argIndex := 2

	if dashboardLayout, ok := updates["dashboard_layout"]; ok {
		query += `, dashboard_layout = $` + string(rune('0'+argIndex))
		args = append(args, dashboardLayout)
		argIndex++
	}

	if enabledWidgets, ok := updates["enabled_widgets"]; ok {
		query += `, enabled_widgets = $` + string(rune('0'+argIndex))
		args = append(args, enabledWidgets)
		argIndex++
	}

	if theme, ok := updates["theme"]; ok {
		query += `, theme = $` + string(rune('0'+argIndex))
		args = append(args, theme)
		argIndex++
	}

	query += ` WHERE client_id = $1`

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// trackEventRecord records a portal analytics event using stored procedure
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: Stored procedure track_portal_event with 6 parameters
func (s *Service) trackEventRecord(ctx context.Context, event *AnalyticsEvent) error {
	query := `
		SELECT track_portal_event($1, $2, $3, $4, $5, $6)
	`

	_, err := s.db.ExecContext(ctx, query,
		event.ClientID,
		event.TenantID,
		event.EventType,
		event.EventData,
		event.SessionID,
		event.DeviceType,
	)

	return err
}

// getEngagementMetricsRecord retrieves engagement metrics using stored procedure
// TODO: Implement Hasura GraphQL query
// SQL fallback: Stored procedure get_portal_engagement_metrics returns metrics struct
func (s *Service) getEngagementMetricsRecord(ctx context.Context, clientID uuid.UUID, days int) (map[string]interface{}, error) {
	query := `SELECT * FROM get_portal_engagement_metrics($1, $2)`

	var result struct {
		TotalLogins               int       `db:"total_logins"`
		AvgSessionDurationMinutes float64   `db:"avg_session_duration_minutes"`
		MostViewedWidget          string    `db:"most_viewed_widget"`
		WidgetViewCount           int       `db:"widget_view_count"`
		LastLogin                 time.Time `db:"last_login"`
	}

	err := s.db.GetContext(ctx, &result, query, clientID, days)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_logins":                 result.TotalLogins,
		"avg_session_duration_minutes": result.AvgSessionDurationMinutes,
		"most_viewed_widget":           result.MostViewedWidget,
		"widget_view_count":            result.WidgetViewCount,
		"last_login":                   result.LastLogin,
	}, nil
}
