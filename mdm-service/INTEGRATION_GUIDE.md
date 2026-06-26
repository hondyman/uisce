# Calendar Module → MDM Service Integration Guide

This guide explains how your existing Calendar Module integrates with the new MDM Calendar Service to consume trusted, versioned calendar data.

## Architecture: Decoupling Data Quality from Consumption

### Before (Tight Coupling)
```
Calendar Module
  └─ Embedded holiday logic
  └─ Data quality issues propagate to consumers
  └─ No audit trail
  └─ Hard to fix errors (requires redeployment)
```

### After (MDM Separation)
```
External Sources (Bloomberg, Exchange, Internal)
    │
    ├──→ MDM Service (Golden Record)
    │      ├─ Data validation
    │      ├─ Survivorship rules
    │      ├─ Conflict detection
    │      └─ Full lineage audit trail
    │
    └──→ Calendar Module (Consumer)
           └─ Simple query: "Is 2024-12-25 a business day?"
           └─ Always gets the trusted answer
           └─ Can access lineage if needed
```

## Integration Steps

### Step 1: Add MDM Client to Calendar Service

```bash
cd /Users/eganpj/GitHub/semlayer/calendar-service

# Add MDM service as dependency
go get github.com/hondyman/semlayer/mdm-service
```

Create `internal/mdm/client.go`:

```go
package mdm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Client wraps MDM API calls
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new MDM client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// GoldenCalendarRecord represents a calendar entry from MDM
type GoldenCalendarRecord struct {
	ID              string `json:"id"`
	CalendarDate    string `json:"calendar_date"`
	IsBusinessDay   bool   `json:"is_business_day"`
	RegionCode      string `json:"region_code"`
	ExchangeCode    *string `json:"exchange_code"`
	HolidayName     *string `json:"holiday_name"`
	SourceType      string `json:"source_type"`
	ConfidenceScore int    `json:"confidence_score"`
}

// GetGoldenCalendar fetches trusted calendar data from MDM
func (c *Client) GetGoldenCalendar(ctx context.Context, tenantID uuid.UUID, start, end time.Time, region string, exchange *string) ([]GoldenCalendarRecord, error) {
	url := fmt.Sprintf("%s/api/v1/mdm/calendar/golden?start_date=%s&end_date=%s&region=%s",
		c.baseURL,
		start.Format("2006-01-02"),
		end.Format("2006-01-02"),
		region,
	)

	if exchange != nil {
		url += fmt.Sprintf("&exchange=%s", *exchange)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set tenant header
	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("Authorization", "Bearer " + c.getToken(ctx))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mdm request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mdm returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse records
	recordsData, _ := json.Marshal(result["records"])
	var records []GoldenCalendarRecord
	json.Unmarshal(recordsData, &records)

	return records, nil
}

// IsBusinessDay checks if a specific date is a business day
func (c *Client) IsBusinessDay(ctx context.Context, tenantID uuid.UUID, date time.Time, region string, exchange *string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/mdm/calendar/is-business-day?date=%s&region=%s",
		c.baseURL,
		date.Format("2006-01-02"),
		region,
	)

	if exchange != nil {
		url += fmt.Sprintf("&exchange=%s", *exchange)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return true, err // Default to true if check fails
	}

	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("Authorization", "Bearer " + c.getToken(ctx))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return true, fmt.Errorf("mdm check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return true, fmt.Errorf("mdm returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	isBusinessDay, _ := result["is_business_day"].(bool)
	return isBusinessDay, nil
}

// GetLineage retrieves the audit trail for a calendar entry
func (c *Client) GetLineage(ctx context.Context, tenantID uuid.UUID, goldenRecordID string) (interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/mdm/calendar/lineage/%s", c.baseURL, goldenRecordID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("Authorization", "Bearer " + c.getToken(ctx))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

// Helper to get JWT token (implement based on your auth scheme)
func (c *Client) getToken(ctx context.Context) string {
	// Retrieve from context or environment
	// For now, assume it's set in Authorization middleware
	return ""
}
```

### Step 2: Update Calendar Service to Use MDM

Modify `internal/services/calendar_service.go`:

```go
package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/calendar-service/internal/mdm"
	"go.uber.org/zap"
)

// CalendarService now uses MDM for trusted data
type CalendarService struct {
	mdmClient *mdm.Client
	logger    *zap.Logger
	// Cache for performance
	cache *CalendarCache
}

// NewCalendarService creates service with MDM integration
func NewCalendarService(mdmClient *mdm.Client, logger *zap.Logger) *CalendarService {
	return &CalendarService{
		mdmClient: mdmClient,
		logger:    logger,
		cache:     NewCalendarCache(5 * time.Minute),
	}
}

// GetBusinessDays returns business days for a date range (from MDM)
func (s *CalendarService) GetBusinessDays(
	ctx context.Context,
	tenantID uuid.UUID,
	start time.Time,
	end time.Time,
	region string,
	exchange *string,
) ([]time.Time, error) {
	// Try cache first
	cacheKey := s.getCacheKey(tenantID, start, end, region, exchange)
	if cached, ok := s.cache.Get(cacheKey); ok {
		s.logger.Debug("cache hit", zap.String("key", cacheKey))
		return cached.([]time.Time), nil
	}

	// Fetch from MDM
	records, err := s.mdmClient.GetGoldenCalendar(ctx, tenantID, start, end, region, exchange)
	if err != nil {
		s.logger.Error("mdm fetch failed", zap.Error(err))
		return nil, err
	}

	// Convert to business days
	var businessDays []time.Time
	for _, record := range records {
		if record.IsBusinessDay {
			dateTime, _ := time.Parse("2006-01-02", record.CalendarDate)
			businessDays = append(businessDays, dateTime)
		}
	}

	// Cache result
	s.cache.Set(cacheKey, businessDays)

	s.logger.Info("fetched business days from MDM",
		zap.Int("count", len(businessDays)),
		zap.String("region", region))

	return businessDays, nil
}

// IsBusinessDay checks if a date is a business day (from MDM)
func (s *CalendarService) IsBusinessDay(
	ctx context.Context,
	tenantID uuid.UUID,
	date time.Time,
	region string,
	exchange *string,
) (bool, error) {
	return s.mdmClient.IsBusinessDay(ctx, tenantID, date, region, exchange)
}

// GetHolidays returns holidays for a date range (from MDM)
func (s *CalendarService) GetHolidays(
	ctx context.Context,
	tenantID uuid.UUID,
	start time.Time,
	end time.Time,
	region string,
	exchange *string,
) ([]Holiday, error) {
	records, err := s.mdmClient.GetGoldenCalendar(ctx, tenantID, start, end, region, exchange)
	if err != nil {
		return nil, err
	}

	var holidays []Holiday
	for _, record := range records {
		if !record.IsBusinessDay && record.HolidayName != nil {
			dateTime, _ := time.Parse("2006-01-02", record.CalendarDate)
			holidays = append(holidays, Holiday{
				Date:       dateTime,
				Name:       *record.HolidayName,
				Region:     region,
				Exchange:   exchange,
				Confidence: record.ConfidenceScore,
				Source:     record.SourceType,
			})
		}
	}

	return holidays, nil
}

// Holiday represents a holiday record
type Holiday struct {
	Date       time.Time
	Name       string
	Region     string
	Exchange   *string
	Confidence int
	Source     string
}

func (s *CalendarService) getCacheKey(tenantID uuid.UUID, start, end time.Time, region string, exchange *string) string {
	exKey := ""
	if exchange != nil {
		exKey = *exchange
	}
	return tenantID.String() + "|" + start.Format("20060102") + "|" + end.Format("20060102") + "|" + region + "|" + exKey
}

// CalendarCache for intermediate caching
type CalendarCache struct {
	data    map[string]interface{}
	ttl     time.Duration
	expires map[string]time.Time
}

func NewCalendarCache(ttl time.Duration) *CalendarCache {
	return &CalendarCache{
		data:    make(map[string]interface{}),
		ttl:     ttl,
		expires: make(map[string]time.Time),
	}
}

func (c *CalendarCache) Get(key string) (interface{}, bool) {
	if time.Now().After(c.expires[key]) {
		delete(c.data, key)
		return nil, false
	}
	val, ok := c.data[key]
	return val, ok
}

func (c *CalendarCache) Set(key string, value interface{}) {
	c.data[key] = value
	c.expires[key] = time.Now().Add(c.ttl)
}
```

### Step 3: Update API Handlers

Modify `internal/api/calendar_handlers.go`:

```go
package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/calendar-service/internal/services"
)

// CalendarHandler provides HTTP endpoints
type CalendarHandler struct {
	service *services.CalendarService
}

// GetBusinessDaysHandler GET /api/v1/calendar/business-days
func (h *CalendarHandler) GetBusinessDaysHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant ID
	tenantID := uuid.MustParse(r.Header.Get("X-Tenant-ID"))

	// Parse query params
	start, _ := time.Parse("2006-01-02", r.URL.Query().Get("start"))
	end, _ := time.Parse("2006-01-02", r.URL.Query().Get("end"))
	region := r.URL.Query().Get("region")
	exchange := r.URL.Query().Get("exchange")

	// Fetch from MDM (via CalendarService)
	businessDays, err := h.service.GetBusinessDays(ctx, tenantID, start, end, region, exchangePtr(exchange))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return as JSON
	w.Header().Set("Content-Type", "application/json")
	// ... marshal businessDays ...
}

func exchangePtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
```

### Step 4: Wire MDM Client in Dependency Injection

Update `main.go`:

```go
package main

import (
	"github.com/hondyman/semlayer/calendar-service/internal/mdm"
	"github.com/hondyman/semlayer/calendar-service/internal/services"
)

func main() {
	// Initialize MDM client
	mdmClient := mdm.NewClient(os.Getenv("MDM_SERVICE_URL"))

	// Create calendar service with MDM integration
	calendarService := services.NewCalendarService(mdmClient, logger)

	// Register handlers
	calendarHandler := &api.CalendarHandler{Service: calendarService}
	mux.HandleFunc("GET /api/v1/calendar/business-days", calendarHandler.GetBusinessDaysHandler)

	// Start server
	http.ListenAndServe(":8081", mux)
}
```

### Step 5: Configure Service Discovery

In `docker-compose.yml`:

```yaml
version: '3.8'

services:
  mdm-service:
    image: semlayer/mdm-calendar-service:latest
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: "postgres://postgres:password@postgres:5432/semlayer"
      PORT: "8080"
    depends_on:
      - postgres

  calendar-service:
    image: semlayer/calendar-service:latest
    ports:
      - "8081:8081"
    environment:
      MDM_SERVICE_URL: "http://mdm-service:8080"
      PORT: "8081"
    depends_on:
      - mdm-service

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: semlayer
      POSTGRES_PASSWORD: password
    volumes:
      - ./database/migrations:/docker-entrypoint-initdb.d
```

## Query Examples

### Get Next N Business Days for Rebalancing

```go
businessDays, _ := calendarService.GetBusinessDays(
	ctx, tenantID,
	time.Now(),
	time.Now().AddDate(0, 3, 0),
	"US",
	nil,
)

// businessDays = [2024-02-20, 2024-02-21, 2024-02-22, ...]
```

### Check Portfolio Settlement Date

```go
settlementDate := tradeDate.AddDate(0, 0, 2)

// Check if settlement date is a business day
isBusinessDay, _ := calendarService.IsBusinessDay(
	ctx, tenantID,
	settlementDate,
	"US",
	ptr("XNYS"),
)

if !isBusinessDay {
	// Push to next business day
	// Use golden calendar to find next business day
}
```

### Get Holiday Schedule for Payroll

```go
holidays, _ := calendarService.GetHolidays(
	ctx, tenantID,
	time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	"US",
	nil,
)

// holidays = [
//   {Date: 2024-01-01, Name: "New Year's Day", ...},
//   {Date: 2024-12-25, Name: "Christmas", ...},
// ]
```

### Audit Why a Date Was Marked as Holiday

```go
lineage, _ := mdmClient.GetLineage(ctx, tenantID, goldenRecordID)

// lineage.History shows:
// - Priority 1: ExchangeFeed marked it as holiday (Confidence: 100)
// - Two alternative sources (Bloomberg, Internal) also agree
// - No conflicts detected
```

## Testing

Create `internal/services/calendar_service_test.go`:

```go
package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/calendar-service/internal/mdm"
	"github.com/hondyman/semlayer/calendar-service/internal/services"
)

// MockMDMClient for testing
type MockMDMClient struct{}

func (m *MockMDMClient) GetGoldenCalendar(ctx context.Context, tenantID uuid.UUID, start, end time.Time, region string, exchange *string) ([]mdm.GoldenCalendarRecord, error) {
	return []mdm.GoldenCalendarRecord{
		{
			CalendarDate:  "2024-12-25",
			IsBusinessDay: false,
			RegionCode:    "US",
		},
	}, nil
}

func TestIsBusinessDay(t *testing.T) {
	service := services.NewCalendarService(&MockMDMClient{}, nil)

	isBusinessDay, _ := service.IsBusinessDay(
		context.Background(),
		uuid.New(),
		time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC),
		"US",
		nil,
	)

	if isBusinessDay {
		t.Error("expected Christmas to not be a business day")
	}
}
```

## Troubleshooting

### Q: MDM service not reachable

A: Check:
```bash
curl http://localhost:8080/health

# In docker-compose:
docker logs mdm-service
```

### Q: Wrong tenant data returned

A: Verify `X-Tenant-ID` header:
```bash
curl -H "X-Tenant-ID: wrong-id" http://localhost:8080/api/v1/mdm/calendar/golden
# Should return 403 if RLS properly enforced
```

### Q: Stale cache issue

A: Clear cache and check MDM:
```bash
# Force refresh in CalendarService
s.cache.Clear()

# Check MDM directly
curl http://mdm-service:8080/api/v1/mdm/calendar/health
```

## Benefits of Integration

✅ **Single Source of Truth** - All calendar data flows through MDM
✅ **Full Auditability** - Every decision is tracked with lineage
✅ **Data Quality** - Automatic conflict detection & stewardship workflows
✅ **Decoupling** - Calendar module needs no holiday logic
✅ **Scalability** - MDM handles multi-region, multi-tenant at database level
✅ **Governance** - Role-based access control, audit trails for compliance

Your Calendar Module now consumes **Governed, Trusted, Audited Calendar Data**! 🎉
