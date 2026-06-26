# Calendar Service - Developer Quick Reference

## Quick Links
- 📋 [Sprint 1 Summary](./SPRINT1_SUMMARY.md) - High-level overview
- 📖 [Delivery Document](./SPRINT1_DELIVERY.md) - Complete specification
- 🏗️ [Architecture](./ARCHITECTURE.md) - System design
- 🚀 [Build Script](./build.sh) - Automated build & test

## Directory Structure

```
calendar-service/
├── cmd/server/main.go              # Service entry point
├── internal/
│   ├── api/                        # HTTP handlers
│   │   ├── *_handlers.go           # Endpoint handlers
│   │   └── router.go               # Route registration
│   ├── availability/               # Business logic
│   │   ├── checker.go              # Existing checker
│   │   ├── blackout.go             # NEW: RRULE expansion
│   │   └── sla_calculator.go       # NEW: Metrics
│   ├── server/http.go              # Server lifecycle
│   ├── hasura/client.go            # NEW: GraphQL client
│   ├── cache/                      # Redis caching
│   └── config/                     # Configuration
├── go.mod                          # Dependencies
└── docs/                           # Documentation
```

## Building

### Standard Build
```bash
cd calendar-service
go mod tidy
go build -o bin/calendar-service ./cmd/server
```

### Using Build Script
```bash
chmod +x build.sh
./build.sh
```

### Fast Development Build
```bash
go build -o /tmp/calendar-service ./cmd/server
/tmp/calendar-service -port 8080 -loglevel debug
```

## Running

### Local
```bash
./bin/calendar-service -port 8080 -loglevel info
```

### With Custom Options
```bash
./bin/calendar-service \
  -port 9090 \
  -loglevel debug
```

### With Docker (Coming Sprint 2)
```bash
docker run -p 8080:8080 calendar-service:latest
```

## Testing Endpoints

### Health Check
```bash
curl http://localhost:8080/api/v1/health
```

### Create Calendar
```bash
curl -X POST http://localhost:8080/api/v1/calendars \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "tenant-1",
    "name": "Support Calendar",
    "timezone": "UTC",
    "type": "support",
    "actor_id": "admin"
  }'
```

### Check Availability
```bash
curl -X POST http://localhost:8080/api/v1/availability \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "tenant-1",
    "calendar_id": "cal-123",
    "start_time": "2024-01-15T09:00:00Z",
    "duration_secs": 3600,
    "include_reason": true
  }'
```

### Create Blackout
```bash
curl -X POST http://localhost:8080/api/v1/blackouts \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "tenant-1",
    "calendar_id": "cal-123",
    "name": "Maintenance",
    "start_time": "2024-01-20T23:00:00Z",
    "end_time": "2024-01-20T23:30:00Z",
    "recurrence_rule": "FREQ=DAILY",
    "reason": "Nightly maintenance"
  }'
```

## Key Types

### AvailabilityResult
```go
type AvailabilityResult struct {
    IsAvailable bool
    StartTime   time.Time
    EndTime     time.Time
    Reason      string  // If not available
    SLAMet      bool
    Confidence  float32
}
```

### RecurringBlackout (NEW)
```go
type RecurringBlackout struct {
    ID                 string
    StartTime          time.Time
    EndTime            time.Time
    RecurrenceRule     string      // RRULE format
    RecurrenceTimezone string      // IANA timezone
    RecurrenceEnd      *time.Time
    IsRecurring        bool
}

// Method
func (rb *RecurringBlackout) ExpandOccurrences(
    rangeStart, rangeEnd time.Time,
) ([]Occurrence, error)
```

### SLACalculator (NEW)
```go
// Methods
func (s *SLACalculator) CalculateFulfillmentTime(
    startTime time.Time,
    availabilityOccurrences []Occurrence,
    blackoutOccurrences []Occurrence,
) time.Duration

func (s *SLACalculator) CalculateComplianceRate(
    availabilityOccurrences []Occurrence,
    blackoutOccurrences []Occurrence,
    periodStart, periodEnd time.Time,
) float32
```

## Dependencies

### Direct Dependencies
```
github.com/google/uuid v1.6.0          - ID generation
github.com/gorilla/mux v1.8.0           - HTTP routing
github.com/hasura/go-graphql-client     - GraphQL client
github.com/sirupsen/logrus v1.9.4       - Logging
github.com/teambition/rrule-go v1.8.2   - RRULE parsing (NEW)
go.temporal.io/sdk v1.40.0              - Workflows
```

### Adding New Dependencies
```bash
go get github.com/owner/package@v1.0.0
go mod tidy
```

## Common Tasks

### Adding a New Endpoint
1. Create handler method in appropriate `*_handlers.go`
2. Register route in `internal/api/router.go`
3. Add request/response types with JSON tags
4. Test with curl

**Example:**
```go
// In calendar_handlers.go
func (h *CalendarHandler) GetCalendars(w http.ResponseWriter, r *http.Request) {
    // Implementation
}

// In router.go
api.HandleFunc("/calendars", h.calendarHandler.GetCalendars).Methods("GET")
```

### Adding Business Logic
1. Create type/method in relevant `internal/availability/*.go`
2. Document with comments
3. Add error handling
4. Call from appropriate handler
5. Add tests in Sprint 2

### Debugging
```bash
# Run with debug logging
./bin/calendar-service -loglevel debug

# Check what's listening
lsof -i :8080

# Kill process on port
kill -9 $(lsof -t -i:8080)

# View logs in real-time
./bin/calendar-service | jq .
```

## Common RRULE Patterns

```
FREQ=DAILY                          # Every day
FREQ=WEEKLY;BYDAY=MO,WE,FR         # Every Monday, Wednesday, Friday
FREQ=MONTHLY;BYMONTHDAY=15         # Every 15th of month
FREQ=YEARLY;BYMONTH=12;BYMONTHDAY=25  # Every Christmas
FREQ=WEEKLY;INTERVAL=2;BYDAY=TU    # Every other Tuesday
```

## Timezone Handling

Always use IANA timezone identifiers:
```
✓ America/New_York
✓ Europe/London
✓ Asia/Tokyo
✓ UTC

✗ EST (ambiguous - daylight savings)
✗ PST (abbreviation)
✗ GMT (not accurate)
```

## Error Handling Pattern

```go
if err != nil {
    h.logger.WithError(err).WithField("tenant_id", tenantID).Warn("Operation failed")
    http.Error(w, "Failed to process request", http.StatusInternalServerError)
    return
}
```

## Logging Pattern

```go
h.logger.WithFields(logrus.Fields{
    "tenant_id": tenantID,
    "calendar_id": calendarID,
    "duration_secs": durationSecs,
}).Info("Availability check performed")
```

## JSON Tag Conventions

```go
type Request struct {
    TenantID    string    `json:"tenant_id"`           // Required
    CalendarID  string    `json:"calendar_id"`         // Required
    Description string    `json:"description,omitempty"` // Optional
    Count       int       `json:"count"`               // Omit if zero
    CreatedAt   time.Time `json:"created_at"`          // ISO8601
}
```

## Sprint 1 → Sprint 2 Transition

### What's Ready for Sprint 2
- ✅ All API endpoints specified
- ✅ Business logic implemented
- ✅ Error handling framework
- ✅ Logging infrastructure
- ✅ Handler stubs

### What Sprint 2 Will Add
- 🔨 Database integration
- 🧪 Comprehensive tests
- 💾 Actual data persistence
- 🚀 Performance optimization
- 🔐 Authentication
- 📦 Deployment configs

## Troubleshooting

### Build Fails with "package not found"
```bash
go mod tidy
go mod download
```

### Import error on calendar-service
Ensure calendar-service is in `/go.work`:
```bash
cat /Users/eganpj/GitHub/semlayer/go.work
# Should include: ./calendar-service
```

### Port Already in Use
```bash
kill -9 $(lsof -t -i:8080)
./bin/calendar-service -port 9090
```

### JSON Marshal Error
Check JSON tags - ensure all fields are properly tagged or unexported:
```go
type Response struct {
    ID       string `json:"id"`       // ✓ Exported + tagged
    internal string `json:"-"`        // ✓ Unexported
    Debug    string                   // ✗ Exported but untagged
}
```

## Git Workflow

```bash
# Pull latest
git pull origin main

# Create feature branch
git checkout -b feature/calendar-feature

# Make changes, test
go test ./...

# Commit
git commit -m "feat: new calendar feature"

# Push for review
git push origin feature/calendar-feature
```

## Code Review Checklist

- [ ] Error handling present
- [ ] Logging at appropriate levels
- [ ] JSON tags correct
- [ ] Comments for exported functions
- [ ] No hardcoded values
- [ ] Consistent with code style
- [ ] Handles edge cases
- [ ] No goroutine leaks

## Performance Tips

1. **Use Bulk Operations**
   - `POST /api/v1/availability/bulk` for multiple checks

2. **Enable Caching**
   - Redis caching for calendars (Sprint 2)
   - Cache RRULE expansions

3. **Batch Database Queries**
   - Group IDs in IN clauses
   - Use database batch inserts

4. **Optimize Timezones**
   - Pre-calculate UTC where possible
   - Avoid repeated timezone conversions

## Resources

### Go Documentation
- https://golang.org/doc/
- https://pkg.go.dev/

### Libraries Used
- https://github.com/gorilla/mux - HTTP routing
- https://github.com/sirupsen/logrus - Logging
- https://github.com/teambition/rrule-go - Recurrence rules

### Time/Date Best Practices
- Always store in UTC
- Parse with time.RFC3339
- Display in user's timezone

---

**Last Updated:** February 17, 2025  
**Sprint:** 1 (Complete)  
**Next Sprint:** Sprint 2 - Database Integration
