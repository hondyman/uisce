package mdm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Client is the MDM Calendar Service API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Entry
	timeout    time.Duration
}

// NewClient creates a new MDM client
func NewClient(baseURL string, timeout time.Duration, logger *logrus.Entry) *Client {
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}

	return &Client{
		baseURL: baseURL,
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// ============================================================================
// Response Models
// ============================================================================

type GoldenCalendarRecord struct {
	ID              string  `json:"id"`
	CalendarDate    string  `json:"calendar_date"`
	IsBusinessDay   bool    `json:"is_business_day"`
	RegionCode      string  `json:"region_code"`
	ExchangeCode    *string `json:"exchange_code"`
	HolidayName     *string `json:"holiday_name"`
	SourceType      string  `json:"source_type"`
	ConfidenceScore int     `json:"confidence_score"`
}

type GetGoldenCalendarResponse struct {
	Records            []GoldenCalendarRecord `json:"records"`
	CoveragePercentage float64                `json:"coverage_percentage"`
	Conflicts          []ConflictRecord       `json:"conflicts,omitempty"`
}

type IsBusinessDayResponse struct {
	Date          string  `json:"date"`
	Region        string  `json:"region"`
	Exchange      *string `json:"exchange"`
	IsBusinessDay bool    `json:"is_business_day"`
}

type LineageRecord struct {
	ID               string  `json:"id"`
	SemanticTerm     string  `json:"semantic_term"`
	PreviousValue    *string `json:"previous_value"`
	WinningValue     string  `json:"winning_value"`
	WinningSourceID  *string `json:"winning_source_id"`
	RuleApplied      string  `json:"rule_applied"`
	ExecutionTime    string  `json:"execution_time"`
	ConflictDetected bool    `json:"conflict_detected"`
}

type LineageResponse struct {
	GoldenRecordID string          `json:"golden_record_id"`
	History        []LineageRecord `json:"history"`
	SourceStats    json.RawMessage `json:"source_stats"`
}

type ConflictRecord struct {
	ID             string `json:"id"`
	GoldenRecordID string `json:"golden_record_id"`
	ConflictType   string `json:"conflict_type"`
	Severity       string `json:"severity"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

type HealthCheckResponse struct {
	TenantID                  string  `json:"tenant_id"`
	CoveragePercentage        float64 `json:"coverage_percentage"`
	ConflictCount             int     `json:"conflict_count"`
	HighConfidencePercentage  float64 `json:"high_confidence_percentage"`
	DaysSinceLastOfficialFeed int     `json:"days_since_last_official_feed"`
	Status                    string  `json:"status"`
}

// ============================================================================
// API Methods
// ============================================================================

// GetGoldenCalendar fetches trusted calendar data from MDM
func (c *Client) GetGoldenCalendar(
	ctx context.Context,
	tenantID uuid.UUID,
	start, end time.Time,
	region string,
	exchange *string,
	token string,
) (*GetGoldenCalendarResponse, error) {

	url := fmt.Sprintf(
		"%s/api/v1/mdm/calendar/golden?start_date=%s&end_date=%s&region=%s",
		c.baseURL,
		start.Format("2006-01-02"),
		end.Format("2006-01-02"),
		region,
	)

	if exchange != nil && *exchange != "" {
		url += fmt.Sprintf("&exchange=%s", *exchange)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.WithError(err).Error("failed to create MDM request")
		return nil, err
	}

	c.setHeaders(req, tenantID, token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Error("failed to call MDM calendar endpoint")
		return nil, fmt.Errorf("mdm request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.WithField("status", resp.StatusCode).WithField("body", string(body)).Error("MDM returned error")
		return nil, fmt.Errorf("mdm returned status %d: %s", resp.StatusCode, string(body))
	}

	var result *GetGoldenCalendarResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.WithError(err).Error("failed to decode MDM response")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.
		WithField("record_count", len(result.Records)).
		WithField("coverage", result.CoveragePercentage).
		Info("fetched golden calendar from MDM")

	return result, nil
}

// IsBusinessDay checks if a specific date is a business day according to MDM
func (c *Client) IsBusinessDay(
	ctx context.Context,
	tenantID uuid.UUID,
	date time.Time,
	region string,
	exchange *string,
	token string,
) (bool, error) {

	url := fmt.Sprintf(
		"%s/api/v1/mdm/calendar/is-business-day?date=%s&region=%s",
		c.baseURL,
		date.Format("2006-01-02"),
		region,
	)

	if exchange != nil && *exchange != "" {
		url += fmt.Sprintf("&exchange=%s", *exchange)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.WithError(err).Error("failed to create IsBusinessDay request")
		return true, err // Default to true on error
	}

	c.setHeaders(req, tenantID, token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Warn("failed to call MDM IsBusinessDay endpoint")
		return true, fmt.Errorf("mdm check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.WithField("status", resp.StatusCode).Warn("MDM IsBusinessDay returned error")
		return true, fmt.Errorf("mdm returned status %d", resp.StatusCode)
	}

	var result *IsBusinessDayResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.WithError(err).Warn("failed to decode IsBusinessDay response")
		return true, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.IsBusinessDay, nil
}

// GetLineage retrieves the audit trail for a golden record
func (c *Client) GetLineage(
	ctx context.Context,
	tenantID uuid.UUID,
	goldenRecordID string,
	token string,
) (*LineageResponse, error) {

	url := fmt.Sprintf("%s/api/v1/mdm/calendar/lineage/%s", c.baseURL, goldenRecordID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	c.setHeaders(req, tenantID, token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mdm returned status %d", resp.StatusCode)
	}

	var result *LineageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetHealthMetrics retrieves health metrics from MDM
func (c *Client) GetHealthMetrics(
	ctx context.Context,
	tenantID uuid.UUID,
	token string,
) (*HealthCheckResponse, error) {

	url := fmt.Sprintf("%s/api/v1/mdm/calendar/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	c.setHeaders(req, tenantID, token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mdm returned status %d", resp.StatusCode)
	}

	var result *HealthCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// ============================================================================
// Health Check
// ============================================================================

// Health checks if MDM service is reachable
func (c *Client) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mdm health check failed: status %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// Helper Methods
// ============================================================================

func (c *Client) setHeaders(req *http.Request, tenantID uuid.UUID, token string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
}
