# Calendar Service

A robust, scalable calendar and availability management service for handling business availability windows, blackout periods, and SLA tracking.

## Overview

The Calendar Service handles:
- **Multi-tenant calendar management** - Manage multiple calendars per tenant
- **Availability windows** - Define available time slots with timezone support
- **Blackout periods** - Mark periods where services are unavailable (one-time or recurring)
- **Availability checking** - Query whether a given time slot aligns with availability
- **Bulk operations** - Check multiple slots in a single request
- **SLA tracking** - Monitor SLA compliance and fulfillment metrics
- **Audit logging** - Full audit trail for compliance

## Architecture

### Core Components

1. **Availability Engine** (`internal/availability/`)
   - `availability.go` - Core availability window and recurrence logic
   - `blackout.go` - Blackout period management with RRULE expansion
   - `rrule_expander.go` - RFC 5545 recurrence rule expansion
   - `sla_calculator.go` - SLA and fulfillment time calculations

2. **API Handlers** (`internal/api/`)
   - `availability_handlers.go` - Availability checking endpoints
   - `blackout_handlers.go` - Blackout management endpoints
   - `calendar_handlers.go` - Calendar CRUD endpoints
   - `tenant_handlers.go` - Tenant configuration endpoints
   - `router.go` - Route registration and setup

3. **Server** (`internal/server/`)
   - `http.go` - HTTP server lifecycle management

4. **Command** (`cmd/calendar-service/`)
   - `main.go` - Service entry point

## API Endpoints

### Availability

```
POST   /api/v1/availability              - Check availability for a single slot
POST   /api/v1/availability/bulk         - Check availability for multiple slots
GET    /api/v1/availability/metrics      - Get availability metrics
```

### Blackouts

```
POST   /api/v1/blackouts                 - Create blackout (one-time or recurring)
GET    /api/v1/blackouts/{id}/occurrences - Get blackout occurrences in date range
DELETE /api/v1/blackouts/{id}            - Delete blackout
```

### Calendars

```
GET    /api/v1/calendars                 - List calendars for tenant
POST   /api/v1/calendars                 - Create new calendar
GET    /api/v1/calendars/{id}            - Get calendar details
PUT    /api/v1/calendars/{id}            - Update calendar
DELETE /api/v1/calendars/{id}            - Delete calendar
```

### Tenants

```
POST   /api/v1/tenants                   - Create tenant
GET    /api/v1/tenants/{id}              - Get tenant
PUT    /api/v1/tenants/{id}              - Update tenant
GET    /api/v1/tenants/{id}/config       - Get tenant config
PUT    /api/v1/tenants/{id}/config       - Update tenant config
```

### Health

```
GET    /api/v1/health                    - Health check
```

## Building

```bash
cd /Users/eganpj/GitHub/semlayer

# Build the calendar service
go build -o bin/calendar-service ./cmd/calendar-service

# With verbose output
cd backend && go build -v -o ../bin/calendar-service ./cmd/calendar-service
```

## Running

```bash
# With defaults (port 8080, info logging)
./bin/calendar-service

# With custom settings
./bin/calendar-service -port 9090 -loglevel debug

# Check health
curl http://localhost:8080/api/v1/health
```

## Configuration

### Command-line Flags

- `-port` (default: 8080) - HTTP server port
- `-loglevel` (default: info) - Log level (debug, info, warn, error)

### Environment Variables (Future)

- `CALENDAR_PORT` - HTTP server port
- `CALENDAR_LOG_LEVEL` - Log level
- `DATABASE_URL` - Database connection string
- `CACHE_REDIS_URL` - Redis cache URL

## Request/Response Examples

### Check Availability

**Request:**
```json
{
  "tenant_id": "tenant-123",
  "calendar_id": "cal-456",
  "start_time": "2024-01-15T09:00:00Z",
  "duration_secs": 3600,
  "include_reason": true
}
```

**Response:**
```json
{
  "is_available": true,
  "start_time": "2024-01-15T09:00:00Z",
  "end_time": "2024-01-15T10:00:00Z",
  "sla_met": true,
  "confidence": 1.0
}
```

### Create Blackout (Recurring)

**Request:**
```json
{
  "tenant_id": "tenant-123",
  "calendar_id": "cal-456",
  "name": "Weekend Closure",
  "start_time": "2024-01-13T00:00:00Z",
  "end_time": "2024-01-14T23:59:59Z",
  "recurrence_rule": "FREQ=WEEKLY;BYDAY=SA,SU",
  "recurrence_timezone": "America/New_York",
  "reason": "Weekend maintenance",
  "actor_id": "user-789"
}
```

**Response:**
```json
{
  "id": "blackout-20240115120000",
  "tenant_id": "tenant-123",
  "calendar_id": "cal-456",
  "name": "Weekend Closure",
  "start_time": "2024-01-13T00:00:00Z",
  "end_time": "2024-01-14T23:59:59Z",
  "is_recurring": true,
  "created_at": "2024-01-15T12:00:00Z",
  "created_by": "user-789"
}
```

## Testing

### Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test file
go test ./internal/availability -v
```

### Test Starlark Files

The workspace includes a task to run Starlark tests:

```bash
# Using VS Code task
Task: Starlark: Run tests in current file

# Or manually
go run ./cmd/starlarktest -file <test_file>
```

## Future Enhancements

### Phase 2
- [ ] CDC integration for real-time updates
- [ ] Redis caching layer
- [ ] Time slot optimization algorithm
- [ ] Holiday calendar support

### Phase 3
- [ ] GraphQL API
- [ ] WebSocket support for real-time changes
- [ ] Bulk import/export (CSV, iCal)
- [ ] Calendar synchronization (Google Calendar, Outlook)

### Phase 4
- [ ] AI-powered availability optimization
- [ ] Predictive maintenance windows
- [ ] Custom availability rules engine
- [ ] Mobile app support

## Database Schema (PostgreSQL - Future)

### Calendars Table
```sql
CREATE TABLE calendars (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  timezone VARCHAR(50),
  type VARCHAR(50),
  created_at TIMESTAMP,
  created_by UUID,
  created_at_utc TIMESTAMP,
  is_deleted BOOLEAN
);
```

### Availability Events Table
```sql
CREATE TABLE availability_events (
  id UUID PRIMARY KEY,
  calendar_id UUID NOT NULL,
  start_time TIMESTAMP,
  end_time TIMESTAMP,
  recurrence_rule TEXT,
  recurrence_timezone VARCHAR(50),
  recurrence_end TIMESTAMP,
  is_recurring BOOLEAN,
  created_at TIMESTAMP,
  created_by UUID,
  FOREIGN KEY (calendar_id) REFERENCES calendars(id)
);
```

### Blackout Periods Table
```sql
CREATE TABLE blackout_periods (
  id UUID PRIMARY KEY,
  calendar_id UUID NOT NULL,
  name VARCHAR(255),
  start_time TIMESTAMP,
  end_time TIMESTAMP,
  recurrence_rule TEXT,
  recurrence_timezone VARCHAR(50),
  recurrence_end TIMESTAMP,
  is_recurring BOOLEAN,
  reason TEXT,
  created_at TIMESTAMP,
  created_by UUID,
  is_deleted BOOLEAN,
  FOREIGN KEY (calendar_id) REFERENCES calendars(id)
);
```

## Development Standards

### Code Organization
- Keep handler logic thin, move business logic to service layer
- Use dependency injection for services
- Log all significant operations for audit trail

### Testing
- Unit tests for availability calculations
- Integration tests for API endpoints
- Test with various timezones and edge cases

### Performance
- Cache availability calculations
- Batch operations where possible
- Index calendar lookups by tenant_id and calendar_id

## Troubleshooting

### Server Won't Start
```
error: address already in use
```
Solution: Use different port with `-port` flag or kill process on port 8080

### High Memory Usage
Check for unclosed database connections or caches. Review rrule expansion for large date ranges.

### Timezone Issues
Always use IANA timezone identifiers (e.g., "America/New_York", not "EST").
Validate recurrence_timezone before processing.

## Contributing

1. Create feature branch
2. Make changes with proper tests
3. Run full test suite
4. Submit PR with description

## License

Internal - Addepar

## Support

For issues or questions:
- Check logs: `loglevel debug`
- Review recent changes in git
- Post in #calendar-service Slack channel
