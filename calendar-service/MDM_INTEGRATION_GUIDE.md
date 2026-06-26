# MDM Integration Guide for Calendar Service

This document provides step-by-step instructions for integrating the MDM (Master Data Management) service with the existing Calendar Service.

## Overview

The MDM integration adds enterprise-grade data quality, conflict resolution, and audit capabilities to the calendar service. The integration is designed to be:

- **Non-breaking**: Existing calendar functionality remains unchanged
- **Configurable**: Can be enabled/disabled via environment variables
- **Resilient**: Gracefully degrades if MDM service is unavailable
- **Multi-tenant aware**: Respects tenant boundaries through X-Tenant-ID headers

## Architecture

```
┌─────────────────────────────────────────────────────┐
│          Calendar Service (Existing)                │
├─────────────────────────────────────────────────────┤
│                                                       │
│  ┌─────────────────────────────────────────────┐   │
│  │     Calendar HTTP Handlers (API Layer)      │   │
│  │  - GET /calendar/business-days              │   │
│  │  - GET /calendar/is-business-day            │   │
│  │  - GET /calendar/holidays                   │   │
│  └──────────────────┬──────────────────────────┘   │
│                     │                                │
│                     ▼                                │
│  ┌─────────────────────────────────────────────┐   │
│  │     Calendar Services (Business Logic)      │   │
│  │  - CalendarService                          │   │
│  │  - AuditService                             │   │
│  └──────────────────┬──────────────────────────┘   │
│                     │                                │
│                     ▼ (NEW)                          │
│  ┌─────────────────────────────────────────────┐   │
│  │        MDM Adapter (Integration Layer)      │   │
│  │  - GetBusinessDays()                        │   │
│  │  - IsBusinessDay()                          │   │
│  │  - GetHolidays()                            │   │
│  │  - GetAuditTrail()                          │   │
│  │  - Caching + Graceful Degradation           │   │
│  └──────────────────┬──────────────────────────┘   │
│                     │                                │
│                     ▼                                │
│  ┌─────────────────────────────────────────────┐   │
│  │       MDM HTTP Client (Network Layer)       │   │
│  │  - HTTP requests to MDM service             │   │
│  │  - JWT token and tenant ID injection        │   │
│  │  - Error handling and retries               │   │
│  └──────────────────┬──────────────────────────┘   │
│                     │                                │
└─────────────────────┼────────────────────────────────┘
                      │
                      ▼
        ┌──────────────────────────────┐
        │     MDM Service (External)   │
        │  - Golden calendar records   │
        │  - Lineage tracking          │
        │  - Conflict detection        │
        │  - Health metrics            │
        └──────────────────────────────┘
```

## Integration Steps

### Step 1: Environment Configuration

Add these environment variables to your `.env` or deployment configuration:

```bash
# Enable MDM integration
MDM_ENABLED=true

# MDM service base URL
MDM_SERVICE_URL=http://localhost:8080

# Cache TTL (how long to cache MDM responses)
MDM_CACHE_TTL=5m

# Request timeout
MDM_TIMEOUT=10s

# Failure mode: "fallback" (safe defaults) or "strict" (fail fast)
MDM_FAILURE_MODE=fallback

# Health check interval
MDM_HEALTH_CHECK_INTERVAL=30s
```

### Step 2: Initialize MDM Module in main.go

```go
package main

import (
	"context"
	"log"

	"calendar-service/internal/mdm"
	"github.com/sirupsen/logrus"
)

func main() {
	// ... existing setup code ...

	logger := logrus.New()

	// Load MDM configuration from environment
	mdmConfig := mdm.LoadFromEnv()
	logger.WithField("config", mdmConfig).Info("Loaded MDM configuration")

	// Initialize MDM module
	mdmModule, err := mdm.NewModule(context.Background(), mdmConfig, logger)
	if err != nil {
		log.Fatalf("Failed to initialize MDM module: %v", err)
	}
	defer mdmModule.Shutdown(context.Background())

	// Get the MDM adapter for dependency injection
	mdmAdapter := mdmModule.GetAdapter()

	// ... rest of setup code ...
	// Pass mdmAdapter to your services/handlers
}
```

### Step 3: Inject MDM Adapter into Services

Update your existing services to accept the MDM adapter:

```go
type CalendarService struct {
	db         *sql.DB
	mdmAdapter *mdm.Adapter  // NEW
	logger     *logrus.Entry
}

func NewCalendarService(db *sql.DB, mdmAdapter *mdm.Adapter, logger *logrus.Logger) *CalendarService {
	return &CalendarService{
		db:         db,
		mdmAdapter: mdmAdapter,
		logger:     logger.WithField("service", "calendar"),
	}
}

// Update existing methods to use MDM when available
func (s *CalendarService) GetBusinessDays(ctx context.Context, tenantID string, start, end time.Time) ([]time.Time, error) {
	// Try MDM first if available
	if s.mdmAdapter != nil && s.mdmAdapter.IsEnabled() {
		businessDays, err := s.mdmAdapter.GetBusinessDays(
			ctx,
			tenantID,
			start,
			end,
			"", "", // region, exchange (optional filters)
			"",     // JWT token (optional)
		)
		if err == nil {
			return businessDays, nil
		}
		// Log but continue to fallback
		s.logger.WithError(err).Debug("MDM lookup failed, falling back to local cache")
	}

	// Fallback to existing local logic
	return s.getBusinessDaysFromLocalCache(ctx, tenantID, start, end)
}
```

### Step 4: Register MDM-Enhanced Handlers

Use the provided example handler to create MDM-aware HTTP endpoints:

```go
// In your router setup
calendarHandler := examples.NewCalendarHandlerWithMDM(mdmAdapter, logger)
calendarHandler.RegisterRoutes(router)
```

Or integrate into existing handlers:

```go
// In your existing handler
func (h *CalendarHandler) HandleGetBusinessDays(w http.ResponseWriter, r *http.Request) {
	// ... parameter extraction ...

	// Check MDM first
	if h.mdmAdapter != nil && h.mdmAdapter.IsEnabled() {
		businessDays, err := h.mdmAdapter.GetBusinessDays(ctx, tenantID, start, end, region, exchange, token)
		if err == nil {
			// Format and return MDM results
			// ...
			return
		}
	}

	// Existing local logic as fallback
	businessDays, err := h.localService.GetBusinessDays(ctx, tenantID, start, end)
	// ...
}
```

### Step 5: Update docker-compose.yml

Ensure both services are running together:

```yaml
version: '3.8'

services:
  mdm-service:
    build:
      context: ./mdm-service
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://user:password@postgres:5432/mdm
      LOG_LEVEL: info
    depends_on:
      - postgres

  calendar-service:
    build:
      context: ./calendar-service
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      MDM_ENABLED: "true"
      MDM_SERVICE_URL: http://mdm-service:8080
      MDM_CACHE_TTL: 5m
      MDM_FAILURE_MODE: fallback
      DATABASE_URL: postgres://user:password@postgres:5432/calendar
    depends_on:
      - mdm-service
      - postgres

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

## Features

### 1. Caching with TTL

MDM responses are cached for configured TTL duration. This:
- Reduces load on MDM service
- Improves response times
- Provides resilience during temporary MDM outages

```go
// Cache will serve results for 5 minutes (default)
// After TTL expires, fresh results are fetched from MDM
businessDays, _ := adapter.GetBusinessDays(ctx, tenantID, start, end, region, exchange, token)
```

### 2. Graceful Degradation

If MDM service fails and `FAILURE_MODE=fallback`:
- `GetBusinessDays()` returns empty list
- `IsBusinessDay()` returns `true` (safe default - assume business day)
- Application continues without interruption

```go
// Safe defaults on MDM failure:
// - IsBusinessDay defaults to true
// - GetBusinessDays returns empty slice  
// - Application continues working
```

### 3. Multi-Tenant Support

MDM requests automatically include tenant ID:

```go
// X-Tenant-ID header is automatically injected
businessDays, _ := adapter.GetBusinessDays(ctx, "tenant-123", start, end, region, exchange, token)
// MDM receives: X-Tenant-ID: tenant-123
```

### 4. Audit Trail Integration

Retrieve MDM lineage information for compliance and debugging:

```go
auditTrail, _ := adapter.GetAuditTrail(ctx, tenantID, recordID, token)
// Returns: []AuditEntry with timestamps, source, user info
```

### 5. Health Monitoring

Check MDM service health:

```go
healthStatus, _ := adapter.GetHealthStatus(ctx, tenantID, token)
// Returns: {
//   "status": "healthy",
//   "uptime": "24h",
//   "records": 1000,
//   "conflict_queue_size": 5
// }
```

## Testing

### Unit Tests for MDM Adapter

```go
func TestMDMAdapterCaching(t *testing.T) {
	// Mock MDM client
	mockClient := &MockMDMClient{}
	mockClient.On("GetGoldenCalendar", mock.Anything, mock.Anything).Return(
		&GetGoldenCalendarResponse{
			Records: []GoldenCalendarRecord{...},
		}, nil,
	)

	// Create adapter
	adapter := NewAdapter(mockClient, 5*time.Minute, logger)

	// First call should hit MDM
	result1, _ := adapter.GetBusinessDays(ctx, "tenant-1", start, end, "", "", "")
	assert.Equal(t, 1, mockClient.CallCount)

	// Second call should use cache
	result2, _ := adapter.GetBusinessDays(ctx, "tenant-1", start, end, "", "", "")
	assert.Equal(t, 1, mockClient.CallCount) // Still 1, not 2

	// Different tenant should not use cache
	result3, _ := adapter.GetBusinessDays(ctx, "tenant-2", start, end, "", "", "")
	assert.Equal(t, 2, mockClient.CallCount) // Now 2
}
```

### Integration Tests

```bash
# Start services
docker-compose -f docker-compose.yml up -d

# Run tests
go test ./... -v -tags=integration

# Check health
curl -X GET http://localhost:3000/api/v1/calendar/health
```

## Troubleshooting

### MDM Service Not Responding

```bash
# 1. Verify MDM_SERVICE_URL environment variable
echo $MDM_SERVICE_URL

# 2. Check MDM service is running
curl http://localhost:8080/health

# 3. Check network connectivity
telnet localhost 8080

# 4. Review logs
docker logs calendar-service | grep MDM
```

### Poor Performance / Cache Not Working

```bash
# 1. Increase cache TTL
MDM_CACHE_TTL=10m  # From 5m to 10m

# 2. Clear cache (if monitoring endpoint available)
curl -X POST http://localhost:3000/api/v1/calendar/cache/clear

# 3. Check MDM response times
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8080/api/v1/mdm/calendar/golden?start_date=2024-01-01&end_date=2024-01-31
```

### Multi-Tenant Isolation Issues

Ensure X-Tenant-ID header is set in all requests:

```bash
# Correct
curl -H "X-Tenant-ID: tenant-123" http://localhost:3000/api/v1/calendar/business-days?start_date=2024-01-01&end_date=2024-01-31

# Wrong - no tenant header
curl http://localhost:3000/api/v1/calendar/business-days?start_date=2024-01-01&end_date=2024-01-31
```

## API Response Examples

### GET /api/v1/calendar/business-days

Request:
```bash
curl -H "X-Tenant-ID: tenant-123" \
  "http://localhost:3000/api/v1/calendar/business-days?start_date=2024-01-01&end_date=2024-01-31"
```

Response:
```json
{
  "start_date": "2024-01-01",
  "end_date": "2024-01-31",
  "business_days": [
    "2024-01-01",
    "2024-01-02",
    "2024-01-03"
  ],
  "count": 22
}
```

### GET /api/v1/calendar/is-business-day

Request:
```bash
curl -H "X-Tenant-ID: tenant-123" \
  "http://localhost:3000/api/v1/calendar/is-business-day?date=2024-01-01"
```

Response:
```json
{
  "date": "2024-01-01",
  "is_business_day": true,
  "region": "US",
  "exchange": "NYSE"
}
```

### GET /api/v1/calendar/holidays

Request:
```bash
curl -H "X-Tenant-ID: tenant-123" \
  "http://localhost:3000/api/v1/calendar/holidays?start_date=2024-01-01&end_date=2024-12-31"
```

Response:
```json
{
  "start_date": "2024-01-01",
  "end_date": "2024-12-31",
  "holidays": [
    {
      "date": "2024-01-01",
      "name": "New Year's Day",
      "region": "US",
      "exchange": "NYSE"
    },
    {
      "date": "2024-07-04",
      "name": "Independence Day",
      "region": "US",
      "exchange": "NYSE"
    }
  ],
  "count": 9
}
```

### GET /api/v1/calendar/health

Request:
```bash
curl -H "X-Tenant-ID: tenant-123" \
  "http://localhost:3000/api/v1/calendar/health"
```

Response:
```json
{
  "mdm_health": {
    "status": "healthy",
    "uptime": "24h30m15s",
    "total_records": 1250,
    "conflict_queue_size": 3,
    "last_check": "2024-01-15T10:30:00Z"
  },
  "timestamp": "2024-01-15T10:30:05Z"
}
```

## Monitoring & Metrics

### Key Metrics to Monitor

1. **MDM Response Time** - Average time for MDM requests
   ```
   histogram_quantile(0.95, mdm_request_duration_seconds)
   ```

2. **Cache Hit Rate** - Percentage of requests served from cache
   ```
   rate(mdm_cache_hits[5m]) / rate(mdm_cache_requests[5m])
   ```

3. **MDM Service Availability** - Uptime percentage
   ```
   up{job="mdm-service"}
   ```

4. **Conflict Queue Size** - Outstanding data conflicts
   ```
   mdm_conflict_queue_size
   ```

### Prometheus Configuration

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'calendar-service'
    static_configs:
      - targets: ['localhost:3000']
    metrics_path: '/metrics'

  - job_name: 'mdm-service'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

## Next Steps

1. **Deploy MDM Service** - Follow [MDM Service README](../mdm-service/README.md)
2. **Configure Calendar Service** - Set environment variables and wire dependencies
3. **Run Integration Tests** - Validate data flows end-to-end
4. **Setup Monitoring** - Configure Prometheus and alerting
5. **Plan Migration** - Schedule gradual rollout to production

## Support & Documentation

- Full MDM Service Docs: [README.md](../mdm-service/README.md)
- MDM Integration Guide: [INTEGRATION_GUIDE.md](../mdm-service/INTEGRATION_GUIDE.md)
- Example Code: [mdm_handler.go](./examples/mdm_handler.go)
- Configuration: [config.go](./internal/mdm/config.go)
