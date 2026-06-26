# Database Persistence Layer

## Overview

The Calendar Service implements a comprehensive PostgreSQL-based persistence layer for managing:
- Multi-tenant isolation
- Calendar profiles (named collections of holidays and blackout rules)
- Holiday dates
- Blackout windows (including recurring windows with RFC 5545 RRULE support)
- Resolved calendar metadata for cache validation
- Audit logs for compliance

## Architecture

### Database Client (`internal/database/client.go`)
- **Connection Pooling**: pgxpool with configurable min/max connections
- **Health Checks**: Automatic connectivity verification
- **Pool Statistics**: Monitoring and diagnostics
- **Graceful Shutdown**: Clean connection cleanup

### Repository Pattern (`internal/repository/`)

Repositories provide data access abstraction with:
- **TenantRepository**: Multi-tenant management
- **CalendarProfileRepository**: Calendar profile CRUD + cache invalidation
- **HolidayRepository**: Holiday date management with upsert support
- **BlackoutWindowRepository**: Blackout window management
- **MetadataRepository**: Cache validation metadata
- **AuditLogRepository**: Change tracking for compliance

### Schema

#### Core Tables

**tenants**
```sql
- id (UUID, PK)
- name (VARCHAR, UNIQUE)
- region (VARCHAR) - Global distribution support
- created_at, updated_at (TIMESTAMP WITH TZ)
- metadata (JSONB)
- deleted_at (soft deletes)
```

**calendar_profiles**
```sql
- id (UUID, PK)
- tenant_id (UUID, FK)
- name, description
- timezone (e.g., "America/New_York")
- region (distribution region)
- is_active (BOOLEAN)
- version (cache invalidation hash)
- created_at, updated_at
- deleted_at (soft deletes)
```

**holidays**
```sql
- id (UUID, PK)
- profile_id (UUID, FK)
- tenant_id (UUID, FK)
- holiday_date (DATE)
- name, region
- created_at
- Unique constraint on (profile_id, holiday_date)
```

**blackout_windows**
```sql
- id (UUID, PK)
- profile_id (UUID, FK)
- tenant_id (UUID, FK)
- start_time, end_time (TIMESTAMP WITH TZ)
- title, reason
- rrule (RFC 5545 recurrence rule)
- is_recurring (BOOLEAN)
- recurrence_start, recurrence_end (DATE)
- created_at, updated_at
```

**resolved_calendar_metadata**
```sql
- id (UUID, PK)
- tenant_id (UUID, FK)
- profile_id (UUID, FK)
- region (VARCHAR)
- resolved_at, version
- holidays_count, blackouts_count
- content_hash (for change detection)
- created_at, updated_at
- Unique constraint on (tenant_id, profile_id, region)
```

**audit_logs**
```sql
- id (UUID, PK)
- tenant_id (UUID, FK)
- entity_type (VARCHAR) - e.g., "calendar_profile", "blackout"
- entity_id (UUID) - Optional references
- action (VARCHAR) - "CREATE", "UPDATE", "DELETE"
- changes (JSONB) - Change details
- performed_by (VARCHAR) - User/service performing action
- created_at
```

#### Indexes

- Composite indexes on `tenant_id, deleted_at` for soft delete queries
- BRIN indexes on time ranges for efficient range queries
- GIN index on profile names for text search
- Composite indexes on foreign keys for join performance
- Descending indexes on created_at for ordering

## Setup

### 1. Create PostgreSQL Database

```bash
createdb calendar_service

# Create application user
psql calendar_service -c "CREATE USER calendar_user WITH PASSWORD 'calendar_password';"
psql calendar_service -c "GRANT ALL PRIVILEGES ON DATABASE calendar_service TO calendar_user;"
```

### 2. Run Migrations

```bash
cd calendar-service

# Check migration status
go run ./cmd/migrate -host localhost -port 5432 -user calendar_user -password calendar_password -db calendar_service -action status

# Apply migrations
go run ./cmd/migrate -host localhost -port 5432 -user calendar_user -password calendar_password -db calendar_service -action up
```

### 3. Docker Compose (Development)

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: calendar_user
      POSTGRES_DB: calendar_service
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
volumes:
  postgres_data:
```

## Integration

### Application Initialization

```go
// Create database client
dbClient, err := database.NewClient(ctx, database.Config{
    Host:            "localhost",
    Port:            5432,
    User:            "calendar_user",
    Password:        "calendar_password",
    Database:        "calendar_service",
    SSLMode:         "disable",
    MaxConnections:  20,
    ConnMaxLifetime: 15 * time.Minute,
    ConnMaxIdleTime: 5 * time.Minute,
}, logger)

// Create repositories
repos := repository.NewPostgresRepositories(dbClient.Pool(), logger)

// Use repositories
profile, err := repos.Profile.GetByName(ctx, tenantID, "work-calendar")
holidays, err := repos.Holiday.ListByProfile(ctx, profileID)
```

## Features

### Multi-tenancy
- All queries include tenant_id to ensure isolation
- No cross-tenant data visibility
- Tenant-scoped audit logs

### Cache Validation
- `resolved_calendar_metadata` table tracks cache state
- Content hash detection for invalidation
- Automatic invalidation on profile/holiday/blackout changes

### Soft Deletes
- `deleted_at` column for historical tracking
- Compliance-friendly retention
- Logical deletion without data loss

### RFC 5545 Support
- RRULE column for recurring blackout patterns
- Recurrence date ranges (start/end)
- Examples: `FREQ=WEEKLY;BYDAY=SA,SU`, `FREQ=DAILY;UNTIL=20261231`

### Audit Trail
- Complete change tracking
- JSON-based change details
- User/service attribution
- Compliance reporting

### Performance Optimizations
- Connection pooling with configurable parameters
- BRIN indexes for time range queries
- Composite indexes for common access patterns
- Prepared statements support via pgx

## Transactions

### Atomic Operations

Repositories support transaction management:

```go
// Multi-step atomic operation
err := dbClient.Transaction(ctx, func(ctx context.Context, tx interface{}) error {
    pgTx := tx.(pgx.Tx)
    
    // Create profile
    // Add holidays
    // Update metadata
    
    return nil
})
```

## Monitoring & Diagnostics

### Pool Statistics

```go
stats := dbClient.GetPoolStats()
// Returns: total_conns, acquire_count, acquire_duration, idle_conns
```

### Health Checks

```go
// Health endpoint integration
err := dbClient.Health(ctx)
if err != nil {
    // Database unavailable
}
```

## Configuration

### Environment Variables (Recommended)

```bash
# Create .env or use system environment
export CALENDAR_DB_HOST=localhost
export CALENDAR_DB_PORT=5432
export CALENDAR_DB_USER=calendar_user
export CALENDAR_DB_PASSWORD=calendar_password
export CALENDAR_DB_NAME=calendar_service
export CALENDAR_DB_SSL_MODE=disable
export CALENDAR_DB_MAX_CONNS=20
export CALENDAR_DB_CONN_MAX_LIFETIME=15m
export CALENDAR_DB_CONN_MAX_IDLE_TIME=5m
```

### Flags

```bash
./bin/calendar-service \
    -db-host localhost \
    -db-port 5432 \
    -db-user calendar_user \
    -db-password calendar_password \
    -db-name calendar_service \
    -port 8080
```

## Query Examples

### Get Calendar Profile with Holidays

```go
// Fetch profile
profile, err := repos.Profile.GetByName(ctx, tenantID, "work-calendar")

// Get all holidays in profile
holidays, err := repos.Holiday.ListByProfile(ctx, profile.ID)

// Get holidays in date range
holidays, err := repos.Holiday.ListByDateRange(ctx, profile.ID, startDate, endDate)
```

### Manage Blackouts

```go
// Create recurring blackout
blackout := &repository.BlackoutWindow{
    ProfileID:       profileID,
    TenantID:        tenantID,
    Title:           "Weekend Hours",
    RRULE:           "FREQ=WEEKLY;BYDAY=SA,SU",
    IsRecurring:     true,
    RecurrenceStart: &startDate,
    RecurrenceEnd:   &endDate,
}
err := repos.Blackout.Create(ctx, blackout)

// List blackouts in range
blackouts, err := repos.Blackout.ListByDateRange(ctx, profileID, rangeStart, rangeEnd)
```

### Audit Logging

```go
// Log profile creation
err := repos.AuditLog.Log(ctx, &repository.AuditLog{
    TenantID:    tenantID,
    EntityType:  "calendar_profile",
    EntityID:    &profileID,
    Action:      "CREATE",
    PerformedBy: userEmail,
    Changes: map[string]interface{}{
        "name":     "work-calendar",
        "timezone": "America/New_York",
    },
})
```

## Migration Files

**001_initial_schema.sql**
- Creates all core tables
- Sets up indexes for performance
- Defines foreign key relationships
- Adds automatic timestamp triggers
- Enables required PostgreSQL extensions (uuid-ossp, pg_trgm)

## Next Steps

- [ ] Integrate repository layer with API handlers
- [ ] Add transaction support for multi-step operations
- [ ] Implement query result caching decorator
- [ ] Add database connection monitoring/metrics
- [ ] Create backup strategy documentation
- [ ] Add read replica support configuration
