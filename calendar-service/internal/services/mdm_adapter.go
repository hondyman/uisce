package services

import (
	"context"
	"sync"
	"time"

	"calendar-service/internal/mdm"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// MDMAdapter integrates MDM Calendar Service with Calendar Module
type MDMAdapter struct {
	mdmClient   *mdm.Client
	logger      *logrus.Logger
	cache       *CalendarCache
	isEnabled   bool
	failureMode string // "fallback" or "strict"
}

// NewMDMAdapter creates a new MDM adapter
func NewMDMAdapter(mdmClient *mdm.Client, logger *logrus.Logger, cacheTTL time.Duration) *MDMAdapter {
	if logger == nil {
		logger = logrus.New()
	}

	return &MDMAdapter{
		mdmClient:   mdmClient,
		logger:      logger,
		cache:       NewCalendarCache(cacheTTL),
		isEnabled:   true,
		failureMode: "fallback", // Safe default
	}
}

// ============================================================================
// Business Day Queries
// ============================================================================

// GetBusinessDays returns business days for a date range using MDM
func (a *MDMAdapter) GetBusinessDays(
	ctx context.Context,
	tenantID uuid.UUID,
	start time.Time,
	end time.Time,
	region string,
	exchange *string,
	token string,
) ([]time.Time, error) {

	if !a.isEnabled {
		a.logger.Debug("MDM adapter disabled, using fallback")
		return nil, nil
	}

	// Try cache first
	cacheKey := a.cacheKey(tenantID, start, end, region, exchange)
	if cached, ok := a.cache.Get(cacheKey); ok {
		a.logger.WithField("cache_key", cacheKey).Debug("cache hit for business days")
		return cached.([]time.Time), nil
	}

	// Fetch from MDM
	golden, err := a.mdmClient.GetGoldenCalendar(ctx, tenantID, start, end, region, exchange, token)
	if err != nil {
		a.logger.WithError(err).Warn("failed to fetch from MDM, using fallback")
		// In fallback mode, return empty (let caller use embedded logic)
		return nil, err
	}

	// Convert to business days
	var businessDays []time.Time
	for _, record := range golden.Records {
		if record.IsBusinessDay {
			if parsedDate, err := time.Parse("2006-01-02", record.CalendarDate); err == nil {
				businessDays = append(businessDays, parsedDate)
			}
		}
	}

	// Cache result
	a.cache.Set(cacheKey, businessDays)

	a.logger.WithField("count", len(businessDays)).
		WithField("region", region).
		Info("fetched business days from MDM")

	return businessDays, nil
}

// IsBusinessDay checks if a specific date is a business day
func (a *MDMAdapter) IsBusinessDay(
	ctx context.Context,
	tenantID uuid.UUID,
	date time.Time,
	region string,
	exchange *string,
	token string,
) (bool, error) {

	if !a.isEnabled {
		return true, nil // Safe default
	}

	// Quick check cache (if we've retrieved calendar for this date)
	cacheKey := a.cacheDateKey(tenantID, date, region, exchange)
	if cached, ok := a.cache.Get(cacheKey); ok {
		return cached.(bool), nil
	}

	// Call MDM for this specific date
	isBusinessDay, err := a.mdmClient.IsBusinessDay(ctx, tenantID, date, region, exchange, token)
	if err != nil {
		a.logger.WithError(err).WithField("date", date.Format("2006-01-02")).
			Warn("failed to check business day with MDM")
		return true, err // Default to true
	}

	// Cache result (with shorter TTL for single dates)
	a.cache.SetWithTTL(cacheKey, isBusinessDay, 1*time.Minute)

	return isBusinessDay, nil
}

// GetHolidays returns holidays for a date range
func (a *MDMAdapter) GetHolidays(
	ctx context.Context,
	tenantID uuid.UUID,
	start time.Time,
	end time.Time,
	region string,
	exchange *string,
	token string,
) ([]Holiday, error) {

	if !a.isEnabled {
		return nil, nil
	}

	// Fetch from MDM
	golden, err := a.mdmClient.GetGoldenCalendar(ctx, tenantID, start, end, region, exchange, token)
	if err != nil {
		a.logger.WithError(err).Warn("failed to fetch holidays from MDM")
		return nil, err
	}

	// Convert to holidays
	var holidays []Holiday
	for _, record := range golden.Records {
		if !record.IsBusinessDay && record.HolidayName != nil {
			if parsedDate, err := time.Parse("2006-01-02", record.CalendarDate); err == nil {
				holidays = append(holidays, Holiday{
					Date:       parsedDate,
					Name:       *record.HolidayName,
					Region:     region,
					Exchange:   exchange,
					Confidence: record.ConfidenceScore,
					Source:     record.SourceType,
				})
			}
		}
	}

	a.logger.WithField("count", len(holidays)).Info("fetched holidays from MDM")
	return holidays, nil
}

// ============================================================================
// Audit & Lineage
// ============================================================================

// GetAuditTrail retrieves the decision history for why a date was marked as holiday/business day
func (a *MDMAdapter) GetAuditTrail(
	ctx context.Context,
	tenantID uuid.UUID,
	goldenRecordID string,
	token string,
) (*MDMLineageTrail, error) {

	if !a.isEnabled {
		return nil, nil
	}

	lineage, err := a.mdmClient.GetLineage(ctx, tenantID, goldenRecordID, token)
	if err != nil {
		a.logger.WithError(err).Warn("failed to fetch lineage from MDM")
		return nil, err
	}

	// Convert to audit trail
	trail := &MDMLineageTrail{
		GoldenRecordID: goldenRecordID,
		History:        make([]MDMLineageEntry, len(lineage.History)),
	}

	for i, entry := range lineage.History {
		trail.History[i] = MDMLineageEntry{
			SemanticTerm:     entry.SemanticTerm,
			WinningValue:     entry.WinningValue,
			RuleApplied:      entry.RuleApplied,
			ExecutionTime:    entry.ExecutionTime,
			ConflictDetected: entry.ConflictDetected,
		}
	}

	return trail, nil
}

// GetHealthStatus retrieves operational health metrics from MDM
func (a *MDMAdapter) GetHealthStatus(
	ctx context.Context,
	tenantID uuid.UUID,
	token string,
) (*HealthStatus, error) {

	if !a.isEnabled {
		return &HealthStatus{Status: "disabled"}, nil
	}

	health, err := a.mdmClient.GetHealthMetrics(ctx, tenantID, token)
	if err != nil {
		a.logger.WithError(err).Warn("failed to fetch health from MDM")
		return nil, err
	}

	return &HealthStatus{
		CoveragePercentage:       health.CoveragePercentage,
		ConflictCount:            health.ConflictCount,
		HighConfidencePercentage: health.HighConfidencePercentage,
		DaysStaleness:            health.DaysSinceLastOfficialFeed,
		Status:                   health.Status,
	}, nil
}

// ============================================================================
// Control Methods
// ============================================================================

// Enable enables MDM adapter
func (a *MDMAdapter) Enable() {
	a.logger.Info("enabling MDM adapter")
	a.isEnabled = true
}

// Disable disables MDM adapter (falls back to embedded logic)
func (a *MDMAdapter) Disable() {
	a.logger.Info("disabling MDM adapter")
	a.isEnabled = false
}

// IsEnabled returns whether MDM adapter is active
func (a *MDMAdapter) IsEnabled() bool {
	return a.isEnabled
}

// ClearCache clears all cached data
func (a *MDMAdapter) ClearCache() {
	a.logger.Info("clearing MDM cache")
	a.cache.Clear()
}

// ============================================================================
// Domain Models
// ============================================================================

// Holiday represents a holiday entry
type Holiday struct {
	Date       time.Time
	Name       string
	Region     string
	Exchange   *string
	Confidence int    // 0-100
	Source     string // e.g., "ExchangeFeed", "Bloomberg"
}

// MDMLineageTrail provides the full decision history from MDM
type MDMLineageTrail struct {
	GoldenRecordID string
	History        []MDMLineageEntry
}

// MDMLineageEntry represents a single lineage entry from MDM
type MDMLineageEntry struct {
	SemanticTerm     string // e.g., "IsBusinessDay"
	WinningValue     string // The decided value
	RuleApplied      string // e.g., "Priority 1: ExchangeOfficial"
	ExecutionTime    string // ISO 8601 timestamp
	ConflictDetected bool   // If disagreement was detected
}

// HealthStatus provides operational metrics
type HealthStatus struct {
	CoveragePercentage       float64 // % of days with records
	ConflictCount            int     // Open conflicts
	HighConfidencePercentage float64 // % with score 80-100
	DaysStaleness            int     // Days since last official update
	Status                   string  // healthy, warning, critical, disabled
}

// ============================================================================
// Cache Implementation
// ============================================================================

// CalendarCache stores cached calendar data with TTL
type CalendarCache struct {
	data    map[string]interface{}
	expires map[string]time.Time
	mu      sync.RWMutex
	ttl     time.Duration
}

// NewCalendarCache creates a new calendar cache
func NewCalendarCache(ttl time.Duration) *CalendarCache {
	return &CalendarCache{
		data:    make(map[string]interface{}),
		expires: make(map[string]time.Time),
		ttl:     ttl,
	}
}

// Get retrieves a cached value if it hasn't expired
func (c *CalendarCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	expiresAt, exists := c.expires[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(expiresAt) {
		// Expired, but don't delete here (RLock)
		return nil, false
	}

	return c.data[key], true
}

// Set stores a value with default TTL
func (c *CalendarCache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value with custom TTL
func (c *CalendarCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = value
	c.expires[key] = time.Now().Add(ttl)
}

// Clear removes all cached entries
func (c *CalendarCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]interface{})
	c.expires = make(map[string]time.Time)
}

// ============================================================================
// Helper Methods
// ============================================================================

func (a *MDMAdapter) cacheKey(tenantID uuid.UUID, start, end time.Time, region string, exchange *string) string {
	exKey := ""
	if exchange != nil {
		exKey = *exchange
	}
	return tenantID.String() + "|" + start.Format("20060102") + "|" + end.Format("20060102") + "|" + region + "|" + exKey
}

func (a *MDMAdapter) cacheDateKey(tenantID uuid.UUID, date time.Time, region string, exchange *string) string {
	exKey := ""
	if exchange != nil {
		exKey = *exchange
	}
	return tenantID.String() + "|" + date.Format("20060102") + "|" + region + "|" + exKey + "|SINGLE"
}
