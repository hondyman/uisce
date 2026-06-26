package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ProactiveIntelligenceActivities contains activities for entity monitoring.
type ProactiveIntelligenceActivities struct {
	db         *sqlx.DB
	httpClient *http.Client
	config     PIConfig
}

// PIConfig holds configuration for proactive intelligence.
type PIConfig struct {
	NewsAPIKey      string
	SECAPIEndpoint  string
	MarketDataURL   string
	NotificationURL string
}

// NewProactiveIntelligenceActivities creates a new activities instance.
func NewProactiveIntelligenceActivities(db *sqlx.DB, config PIConfig) *ProactiveIntelligenceActivities {
	return &ProactiveIntelligenceActivities{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// EntityDataResult contains the result of fetching entity data.
type EntityDataResult struct {
	EntityID     string                 `json:"entity_id"`
	RiskScore    float64                `json:"risk_score"`
	PriceChange  float64                `json:"price_change,omitempty"`
	NewsCount    int                    `json:"news_count"`
	FilingsCount int                    `json:"filings_count"`
	Alerts       []RiskSignalData       `json:"alerts,omitempty"`
	Properties   map[string]interface{} `json:"properties"`
	FetchedAt    time.Time              `json:"fetched_at"`
}

// RiskSignalData represents a risk signal.
type RiskSignalData struct {
	EventType string                 `json:"event_type"`
	Severity  string                 `json:"severity"`
	Title     string                 `json:"title"`
	Details   map[string]interface{} `json:"details"`
	SourceURL string                 `json:"source_url,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// FetchEntityData fetches current data for an entity from various sources.
func (a *ProactiveIntelligenceActivities) FetchEntityData(
	ctx context.Context,
	tenantID string,
	entityID string,
	dataSources []string,
) (*EntityDataResult, error) {
	result := &EntityDataResult{
		EntityID:   entityID,
		Properties: make(map[string]interface{}),
		FetchedAt:  time.Now(),
	}

	// Fetch current entity data from database
	var entity struct {
		RiskScore   float64                `db:"risk_score"`
		Properties  map[string]interface{} `db:"properties"`
		CanonicalID string                 `db:"canonical_id"`
		Name        string                 `db:"name"`
	}

	err := a.db.GetContext(ctx, &entity, `
		SELECT COALESCE(risk_score, 0) as risk_score, properties, canonical_id, name
		FROM financial_entities 
		WHERE tenant_id = $1 AND entity_id = $2
	`, tenantID, entityID)
	if err != nil {
		return nil, fmt.Errorf("fetch entity: %w", err)
	}

	result.RiskScore = entity.RiskScore
	result.Properties = entity.Properties

	// Fetch from enabled data sources
	for _, source := range dataSources {
		switch source {
		case "news":
			if newsData, err := a.fetchNewsData(ctx, entity.Name); err == nil {
				result.NewsCount = newsData.Count
				if newsData.NegativeSentiment {
					result.Alerts = append(result.Alerts, RiskSignalData{
						EventType: "NEWS_SENTIMENT",
						Severity:  "medium",
						Title:     fmt.Sprintf("Negative news sentiment for %s", entity.Name),
						Details:   map[string]interface{}{"headlines": newsData.Headlines},
						Timestamp: time.Now(),
					})
				}
			}

		case "sec_filings":
			if entity.CanonicalID != "" {
				if filings, err := a.fetchSECFilings(ctx, entity.CanonicalID); err == nil {
					result.FilingsCount = filings.Count
					for _, filing := range filings.NewFilings {
						result.Alerts = append(result.Alerts, RiskSignalData{
							EventType: "NEW_FILING",
							Severity:  "low",
							Title:     fmt.Sprintf("New %s filing detected", filing.FormType),
							Details: map[string]interface{}{
								"form_type":   filing.FormType,
								"filing_date": filing.Date,
								"accession":   filing.AccessionNumber,
							},
							SourceURL: filing.URL,
							Timestamp: time.Now(),
						})
					}
				}
			}

		case "market_data":
			if priceData, err := a.fetchMarketData(ctx, entity.CanonicalID); err == nil {
				result.PriceChange = priceData.PercentChange
				result.Properties["last_price"] = priceData.Price
				result.Properties["volume"] = priceData.Volume

				// Check for significant price movement
				if priceData.PercentChange <= -10 || priceData.PercentChange >= 10 {
					severity := "medium"
					if priceData.PercentChange <= -20 || priceData.PercentChange >= 20 {
						severity = "high"
					}
					result.Alerts = append(result.Alerts, RiskSignalData{
						EventType: "PRICE_CHANGE",
						Severity:  severity,
						Title:     fmt.Sprintf("Significant price movement: %.2f%%", priceData.PercentChange),
						Details: map[string]interface{}{
							"percent_change": priceData.PercentChange,
							"current_price":  priceData.Price,
							"volume":         priceData.Volume,
						},
						Timestamp: time.Now(),
					})
				}
			}
		}
	}

	// Calculate updated risk score based on alerts
	if len(result.Alerts) > 0 {
		result.RiskScore = calculateAdjustedRiskScore(entity.RiskScore, result.Alerts)
	}

	return result, nil
}

// NewsData represents news API response.
type NewsData struct {
	Count             int      `json:"count"`
	Headlines         []string `json:"headlines"`
	NegativeSentiment bool     `json:"negative_sentiment"`
}

func (a *ProactiveIntelligenceActivities) fetchNewsData(ctx context.Context, entityName string) (*NewsData, error) {
	// Placeholder - would integrate with news API (e.g., NewsAPI, Bloomberg)
	// For now, return empty data
	return &NewsData{
		Count:             0,
		Headlines:         []string{},
		NegativeSentiment: false,
	}, nil
}

// SECFilings represents SEC filing data.
type SECFilings struct {
	Count      int         `json:"count"`
	NewFilings []SECFiling `json:"new_filings"`
}

// SECFiling represents a single SEC filing.
type SECFiling struct {
	FormType        string `json:"form_type"`
	Date            string `json:"date"`
	AccessionNumber string `json:"accession_number"`
	URL             string `json:"url"`
}

func (a *ProactiveIntelligenceActivities) fetchSECFilings(ctx context.Context, cik string) (*SECFilings, error) {
	// Placeholder - would integrate with SEC EDGAR API
	return &SECFilings{
		Count:      0,
		NewFilings: []SECFiling{},
	}, nil
}

// MarketData represents market price data.
type MarketData struct {
	Price         float64 `json:"price"`
	PercentChange float64 `json:"percent_change"`
	Volume        int64   `json:"volume"`
}

func (a *ProactiveIntelligenceActivities) fetchMarketData(ctx context.Context, symbol string) (*MarketData, error) {
	// Placeholder - would integrate with market data provider
	return &MarketData{
		Price:         0,
		PercentChange: 0,
		Volume:        0,
	}, nil
}

// calculateAdjustedRiskScore adjusts risk score based on alerts.
func calculateAdjustedRiskScore(baseScore float64, alerts []RiskSignalData) float64 {
	adjustment := 0.0
	for _, alert := range alerts {
		switch alert.Severity {
		case "critical":
			adjustment += 20
		case "high":
			adjustment += 10
		case "medium":
			adjustment += 5
		case "low":
			adjustment += 2
		}
	}

	newScore := baseScore + adjustment
	if newScore > 100 {
		newScore = 100
	}
	return newScore
}

// RecordMonitorCheck updates the monitor's last check timestamp.
func (a *ProactiveIntelligenceActivities) RecordMonitorCheck(ctx context.Context, monitorID string) error {
	_, err := a.db.ExecContext(ctx, `
		UPDATE entity_monitors 
		SET last_check_at = NOW(), check_count = check_count + 1
		WHERE monitor_id = $1
	`, monitorID)
	return err
}

// RecordMonitorError records an error for a monitor.
func (a *ProactiveIntelligenceActivities) RecordMonitorError(ctx context.Context, monitorID string, errMsg string) error {
	_, err := a.db.ExecContext(ctx, `
		UPDATE entity_monitors 
		SET last_error = $2, error_count = error_count + 1,
		    status = CASE WHEN error_count >= 5 THEN 'error' ELSE status END
		WHERE monitor_id = $1
	`, monitorID, errMsg)
	return err
}

// CreateRiskEvent creates a new risk event in the database.
func (a *ProactiveIntelligenceActivities) CreateRiskEvent(ctx context.Context, event map[string]interface{}) error {
	eventData, _ := json.Marshal(event["event_data"])

	_, err := a.db.ExecContext(ctx, `
		INSERT INTO risk_events (
			event_id, tenant_id, entity_id, monitor_id, event_type, severity,
			title, description, event_data, source_url, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'new')
	`,
		event["event_id"],
		event["tenant_id"],
		event["entity_id"],
		event["monitor_id"],
		event["event_type"],
		event["severity"],
		event["title"],
		event["description"],
		eventData,
		event["source_url"],
	)

	if err != nil {
		return fmt.Errorf("create risk event: %w", err)
	}

	// Also update the entity's risk score
	a.db.ExecContext(ctx, `
		UPDATE financial_entities 
		SET risk_score = LEAST(100, COALESCE(risk_score, 0) + 
			CASE $2 
				WHEN 'critical' THEN 20 
				WHEN 'high' THEN 10 
				WHEN 'medium' THEN 5 
				ELSE 2 
			END)
		WHERE entity_id = $1
	`, event["entity_id"], event["severity"])

	return nil
}

// SendNotification sends a notification through the specified channel.
func (a *ProactiveIntelligenceActivities) SendNotification(
	ctx context.Context,
	channel string,
	payload map[string]interface{},
) error {
	switch channel {
	case "email":
		return a.sendEmailNotification(ctx, payload)
	case "slack":
		return a.sendSlackNotification(ctx, payload)
	case "webhook":
		return a.sendWebhookNotification(ctx, payload)
	case "in_app":
		return a.createInAppNotification(ctx, payload)
	default:
		return fmt.Errorf("unknown notification channel: %s", channel)
	}
}

func (a *ProactiveIntelligenceActivities) sendEmailNotification(ctx context.Context, payload map[string]interface{}) error {
	// Placeholder - would integrate with email service
	return nil
}

func (a *ProactiveIntelligenceActivities) sendSlackNotification(ctx context.Context, payload map[string]interface{}) error {
	// Placeholder - would integrate with Slack API
	return nil
}

func (a *ProactiveIntelligenceActivities) sendWebhookNotification(ctx context.Context, payload map[string]interface{}) error {
	if a.config.NotificationURL == "" {
		return nil
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", a.config.NotificationURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook failed: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

func (a *ProactiveIntelligenceActivities) createInAppNotification(ctx context.Context, payload map[string]interface{}) error {
	eventID, _ := payload["event_id"].(string)
	entityID, _ := payload["entity_id"].(string)
	title, _ := payload["title"].(string)
	severity, _ := payload["severity"].(string)

	// Get users who should be notified (entity owners, risk managers)
	var userIDs []string
	a.db.SelectContext(ctx, &userIDs, `
		SELECT DISTINCT u.id 
		FROM app_user u
		JOIN entity_monitors em ON em.created_by = u.id
		WHERE em.entity_id = $1
	`, entityID)

	// Create in-app notifications
	for _, userID := range userIDs {
		a.db.ExecContext(ctx, `
			INSERT INTO engagement_notifications (
				user_id, type, title, message, priority, status, created_at
			) VALUES ($1, 'risk_alert', $2, $3, $4, 'draft', NOW())
		`, userID, title, fmt.Sprintf("Risk event %s detected", eventID), priorityFromSeverity(severity))
	}

	return nil
}

func priorityFromSeverity(severity string) int {
	switch severity {
	case "critical":
		return 1
	case "high":
		return 2
	case "medium":
		return 3
	default:
		return 4
	}
}

// StartEntityMonitor starts a new entity monitor workflow.
type StartMonitorInput struct {
	TenantID    uuid.UUID              `json:"tenant_id"`
	EntityID    uuid.UUID              `json:"entity_id"`
	MonitorType string                 `json:"monitor_type"`
	Config      map[string]interface{} `json:"config"`
}

// GetActiveMonitors retrieves all active monitors for a tenant.
func (a *ProactiveIntelligenceActivities) GetActiveMonitors(ctx context.Context, tenantID string) ([]map[string]interface{}, error) {
	var monitors []struct {
		MonitorID   string `db:"monitor_id"`
		EntityID    string `db:"entity_id"`
		MonitorType string `db:"monitor_type"`
		WorkflowID  string `db:"workflow_id"`
		Status      string `db:"status"`
	}

	err := a.db.SelectContext(ctx, &monitors, `
		SELECT monitor_id, entity_id, monitor_type, workflow_id, status
		FROM entity_monitors
		WHERE tenant_id = $1 AND status = 'active'
	`, tenantID)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(monitors))
	for i, m := range monitors {
		result[i] = map[string]interface{}{
			"monitor_id":   m.MonitorID,
			"entity_id":    m.EntityID,
			"monitor_type": m.MonitorType,
			"workflow_id":  m.WorkflowID,
			"status":       m.Status,
		}
	}
	return result, nil
}
