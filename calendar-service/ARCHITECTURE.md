# Calendar Service - Architecture & Design

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Applications                      │
│              (Admin UI, API Clients, Mobile Apps)               │
└────────────────────────┬────────────────────────────────────────┘
                         │ HTTPS
                         ↓
┌─────────────────────────────────────────────────────────────────┐
│                      API Gateway Layer                           │
│              (Load Balancer, Rate Limiting, Auth)               │
└────────────────────────┬────────────────────────────────────────┘
                         │ Internal
                         ↓
╔═════════════════════════════════════════════════════════════════╗
║                  Calendar Service (Go)                          ║
║                                                                  ║
│  ┌─────────────────────────────────────────────────────────┐  │
│  │           HTTP Handler Layer (api/)                     │  │
│  │ ┌──────────┬──────────┬──────────┬──────────┬────────┐ │  │
│  │ │Availability Blackout Calendar Tenant Health │ │  │
│  │ └──────────┴──────────┴──────────┴──────────┴────────┘ │  │
│  └────────────────┬──────────────────────────────────────┘  │
│                   │                                           │
│  ┌────────────────▼───────────────────────────────────────┐  │
│  │        Business Logic Layer (availability/)  │  │
│  │ ┌───────────┬─────────────┬──────────────────┐ │  │
│  │ │ Checker   │  Blackout   │  SLACalculator  │ │  │
│  │ │ (Existing)│  (NEW)      │  (NEW)          │ │  │
│  │ └───────────┴─────────────┴──────────────────┘ │  │
│  └────────────────┬───────────────────────────────┘  │
│                   │                                           │
│  ┌────────────────▼───────────────────────────────────────┐  │
│  │         Data Access & External Services               │  │
│  │ ┌──────────┬─────────┬──────────┬──────────────────┐  │  │
│  │ │ Hasura   │ Redis   │ Database │ Temporal (CDC)   │  │  │
│  │ │ GraphQL  │ Cache   │ Queries  │ Events           │  │  │
│  │ └──────────┴─────────┴──────────┴──────────────────┘  │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                                  │
╚═════════════════════════════════════════════════════════════════╝
                         │
         ┌───────────────┼───────────────┐
         ↓               ↓               ↓
    PostgreSQL      Redis Cluster   Message Queue
    (Timescales)     (Cache)          (CDC, Events)
```

## Component Breakdown

### 1. API Handler Layer (`internal/api/`)

**availability_handlers.go**
- Endpoint: `POST /api/v1/availability`
- Purpose: Check if a time slot is available
- Returns: Availability status, SLA compliance, confidence

**blackout_handlers.go**
- Endpoint: `POST /api/v1/blackouts`
- Purpose: Create and manage blackout periods
- Features: RRULE support, recurring blackouts

**calendar_handlers.go**
- Endpoints: CRUD for calendars
- Purpose: Manage calendar definitions per tenant
- Features: Multi-region support, timezone handling

**tenant_handlers.go**
- Endpoints: Tenant configuration and CRUD
- Purpose: Multi-tenant configuration
- Features: Custom settings, localization preferences

**router.go**
- Purpose: Routes all endpoints through Gorilla Mux
- Features: Middleware hooks, error handling

### 2. Business Logic Layer (`internal/availability/`)

**checker.go** (Existing)
- Purpose: Core availability checking algorithm
- Integration: Uses Hasura for profile queries, Redis for caching

**blackout.go** (NEW - Sprint 1)
- Type: `RecurringBlackout`
- Methods: `ExpandOccurrences()` for RRULE expansion
- Features:
  - RFC 5545 recurrence rule parsing
  - Timezone-aware expansion
  - Efficient date range queries

**sla_calculator.go** (NEW - Sprint 1)
- Type: `SLACalculator`
- Methods:
  - `CalculateFulfillmentTime()` - Time to first available slot
  - `CalculateComplianceRate()` - Percentage availability
- Features: Accounts for blackouts in calculations

### 3. Server Layer (`internal/server/`)

**http.go**
- Type: `Server`
- Methods: `Start()`, `Stop()`
- Features:
  - Graceful shutdown
  - Timeout configuration
  - Signal handling

### 4. Integration Layers

**internal/hasura/**
- GraphQL client for calendar and profile queries
- Admin secret authentication

**internal/cache/**
- Redis caching for performance
- Pub/Sub for cache invalidation
- TTL-based expiration

**internal/config/**
- Configuration management
- Environment variables
- Default values

---

## Data Flow Diagrams

### Availability Check Flow

```
Client Request
    │
    ├─→ Parse Request
    │   └─→ Validate (tenant, calendar, time)
    │
    ├─→ Cache Lookup
    │   └─→ Hit: Return cached result
    │   └─→ Miss: Continue
    │
    ├─→ DB Query
    │   └─→ Fetch calendar profile
    │   └─→ Fetch availability windows
    │   └─→ Fetch active blackouts
    │
    ├─→ Availability Check
    │   └─→ Compare time slot with windows
    │   └─→ Check against blackouts
    │
    ├─→ SLA Calculation
    │   └─→ Calculate fulfillment time
    │   └─→ Determine compliance
    │
    ├─→ Cache Result
    │   └─→ Store with TTL
    │
    └─→ Return Response
        ├─→ is_available
        ├─→ sla_met
        ├─→ confidence
        └─→ reason (optional)
```

### Blackout Creation Flow

```
Admin Request: Create Blackout
    │
    ├─→ Parse Request
    │   └─→ Validate RRULE (if recurring)
    │   └─→ Validate timezone
    │
    ├─→ Persist to Database
    │   └─→ Store blackout record
    │   └─→ Generate UUID
    │
    ├─→ Emit CDC Event
    │   └─→ Message to event queue
    │   └─→ Trigger cache invalidation
    │
    ├─→ Notify Subscribers
    │   └─→ Cache invalidation across instances
    │   └─→ Update dependent services
    │
    └─→ Return Created Blackout
        ├─→ ID
        ├─→ Recurrence info
        └─→ Created timestamp
```

### Recurrence Expansion Flow

```
GET /api/v1/blackouts/{id}/occurrences
    │
    ├─→ Fetch Blackout Record
    │   └─→ Get RRULE, timezone, dates
    │
    ├─→ Parse RRULE
    │   └─→ Use rrule-go library
    │   └─→ Set time zone context
    │
    ├─→ Expand Occurrences
    │   └─→ Generate dates between start & end
    │   └─→ Calculate duration from original
    │   └─→ Apply timezone conversions
    │
    ├─→ Cache Results
    │   └─→ Store expanded dates (TTL: 1 day)
    │
    └─→ Return Occurrences Array
        └─→ Array of {start_time, end_time}
```

---

## Database Schema (PostgreSQL)

### calendars table
```sql
CREATE TABLE calendars (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    timezone VARCHAR(50),
    type VARCHAR(50),  -- 'fulfillment', 'support', 'custom'
    created_at TIMESTAMP DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMP DEFAULT NOW(),
    is_deleted BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);
```

### availability_events table
```sql
CREATE TABLE availability_events (
    id UUID PRIMARY KEY,
    calendar_id UUID NOT NULL,
    name VARCHAR(255),
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    recurrence_rule TEXT,  -- RFC 5545 RRULE
    recurrence_timezone VARCHAR(50),
    recurrence_end TIMESTAMP,
    is_recurring BOOLEAN,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by UUID,
    is_deleted BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (calendar_id) REFERENCES calendars(id)
);
```

### blackout_periods table (NEW)
```sql
CREATE TABLE blackout_periods (
    id UUID PRIMARY KEY,
    calendar_id UUID NOT NULL,
    name VARCHAR(255),
    description TEXT,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    recurrence_rule TEXT,  -- RFC 5545 RRULE
    recurrence_timezone VARCHAR(50),
    recurrence_end TIMESTAMP,
    is_recurring BOOLEAN,
    reason VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMP DEFAULT NOW(),
    is_deleted BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (calendar_id) REFERENCES calendars(id)
);
```

### sla_profiles table (NEW)
```sql
CREATE TABLE sla_profiles (
    id UUID PRIMARY KEY,
    calendar_id UUID NOT NULL,
    name VARCHAR(255),
    target_sla_hours INT,
    measurement_type VARCHAR(50),  -- 'fulfillment_time', 'compliance_rate'
    priority INT DEFAULT 5,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (calendar_id) REFERENCES calendars(id)
);
```

---

## Dependency Tree

### Direct Dependencies
```
calendar-service
├── github.com/google/uuid v1.6.0
├── github.com/gorilla/mux v1.8.0
├── github.com/hasura/go-graphql-client v0.10.0
├── github.com/sirupsen/logrus v1.9.4
├── github.com/teambition/rrule-go v1.8.2  (NEW - Sprint 1)
└── go.temporal.io/sdk v1.40.0
```

### Why Each Dependency?
- **uuid**: Generate unique IDs for calendars, blackouts, tenants
- **gorilla/mux**: HTTP routing with path parameters
- **hasura**: GraphQL queries for calendar data
- **logrus**: Structured JSON logging
- **rrule-go**: RFC 5545 recurrence rule expansion
- **temporal**: Workflow orchestration for complex operations

---

## Performance Characteristics

### Expected Performance

| Operation | Latency | Factor |
|-----------|---------|---------|
| Single Availability Check | 50-100ms | Database + cache lookup |
| Bulk Check (10 slots) | 100-150ms | Batched queries |
| Blackout Creation | 200-500ms | Database + cache invalidation |
| Recurrence Expansion | 10-50ms | In-memory RRULE | computation |
| Metrics Calculation | 100-200ms | Aggregation query |

### Optimization Strategies

1. **Caching** (Redis)
   - Cache availability windows (TTL: 1 hour)
   - Cache calculated compliance rates (TTL: 1 day)
   - Cache recurrence expansions (TTL: 1 day)

2. **Batching**
   - Bulk availability checks grouped
   - Batch cache invalidation
   - Batch CDC events

3. **Async Processing**
   - Async cache updates
   - Background recurrence expansions
   - Temporal workflow for complex calculations

4. **Indexing**
   - Index on (tenant_id, calendar_id)
   - Index on (calendar_id, start_time)
   - Index on is_deleted for soft deletes

---

## Error Handling Strategy

### Error Codes

| Code | Status | Category | Handling |
|------|--------|----------|----------|
| INVALID_TENANT | 400 | Client | Return error response |
| INVALID_RRULE | 400 | Client | Validation error |
| NOT_FOUND | 404 | Client | Resource not found |
| CONFLICT | 409 | Client | Duplicate/conflict |
| INTERNAL_ERROR | 500 | Server | Log & retry |
| SERVICE_UNAVAILABLE | 503 | Server | Circuit breaker |
| TIMEOUT | 508 | Server | Retry with backoff |

### Error Response Format
```json
{
  "error": "INVALID_RRULE",
  "message": "Invalid recurrence rule format",
  "details": "FREQ=INVALID is not supported",
  "request_id": "req-123abc"
}
```

---

## Security Considerations

### Authentication
- [TBD Sprint 2] Multi-tenant request validation via X-Hasura-Tenant-Id header
- [TBD Sprint 2] API key validation for external clients

### Authorization
- Tenant scope enforcement (users can only access their calendars)
- Role-based access control (admin, viewer, editor)
- Audit logging of all mutations

### Data Protection
- TLS for all network communication
- Encrypted storage in database
- Secure handling of Hasura admin secrets

---

## Monitoring & Observability

### Metrics to Track
- API response times
- Availability check accuracy
- Cache hit ratio
- SLA compliance rate
- Error rate by endpoint

### Logging
- JSON structured logging with request IDs
- Log levels: DEBUG, INFO, WARN, ERROR
- Correlation IDs for tracing

### Health Checks
- `/api/v1/health` - Service availability
- Database connectivity
- Redis connectivity
- Hasura connectivity

---

## Next Steps (Sprint 2)

1. **Persistence Layer** - Connect handlers to database
2. **Testing** - Unit and integration tests
3. **Caching Integration** - Full Redis integration
4. **Middleware** - Auth, logging, metrics
5. **Documentation** - OpenAPI/Swagger docs
6. **Deployment** - Docker, Kubernetes, monitoring
