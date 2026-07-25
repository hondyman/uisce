# MDM Calendar Service - Calendar Gold Copy Implementation

This is a comprehensive Master Data Management (MDM) service for managing calendar data across your organization, implementing the **Usice Architecture** principles for semantic terms, business objects, multi-tenant row-level security, and survivorship rules.

## Architecture Overview

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                    External Data Sources                     │
│         (Bloomberg, ExchangeFeed, Internal Steward)          │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ HTTP REST/GraphQL
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                  MDM Calendar Service                        │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Ingestion Pipeline                                  │  │
│  │  1. Store Source Records                             │  │
│  │  2. Match to Existing Golden Records                 │  │
│  │  3. Execute Survivorship Rules (Priority Hierarchy)  │  │
│  │  4. Upsert Golden Record                             │  │
│  │  5. Record Lineage/Audit Trail                       │  │
│  │  6. Flag Conflicts for Stewardship                   │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ Multi-Tenant RLS
                      ▼
┌─────────────────────────────────────────────────────────────┐
│              PostgreSQL + Row-Level Security                 │
│  ┌──────────────────────┐  ┌──────────────────────┐         │
│  │ Golden Records       │  │ Source Records       │         │
│  │ (Trusted Data)       │  │ (Raw Ingestion)      │         │
│  └──────────────────────┘  └──────────────────────┘         │
│  ┌──────────────────────┐  ┌──────────────────────┐         │
│  │ Lineage Tracking     │  │ Conflict Flags       │         │
│  │ (Audit Trail)        │  │ (Stewardship Queue)  │         │
│  └──────────────────────┘  └──────────────────────┘         │
│  ┌──────────────────────┐  ┌──────────────────────┐         │
│  │ Version History      │  │ Health Metrics       │         │
│  │ (Time-Travel)        │  │ (Operational Intel)  │         │
│  └──────────────────────┘  └──────────────────────┘         │
└─────────────────────────────────────────────────────────────┘
                      │
                      │ Query APIs
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                Calendar Module (Consumer)                    │
│         Uses Golden Calendar for all calculations            │
└─────────────────────────────────────────────────────────────┘
```

## Semantic Model

### Semantic Terms (Atomic Meanings)

| Term | Type | Definition |
|------|------|-----------|
| `CalendarDate` | DATE | The specific day being defined (UTC) |
| `IsBusinessDay` | BOOLEAN | True if markets/offices are open |
| `RegionCode` | STRING | ISO-3166 Country Code (e.g., "US", "GB") |
| `ExchangeCode` | STRING | ISO-10383 MIC Code (e.g., "XNYS", "XLON") |
| `HolidayName` | STRING | Human-readable name (e.g., "Independence Day") |
| `SourceType` | STRING | Origin of data (e.g., "Bloomberg", "ExchangeFeed") |
| `ConfidenceScore` | INT | 0-100 score based on survivorship rules |

### Business Object: `HolidaySchedule`

```go
type HolidaySchedule struct {
    ID              uuid.UUID  // Unique identifier
    TenantID        uuid.UUID  // Multi-tenant isolation
    CalendarDate    time.Time  // The date
    IsBusinessDay   bool       // Semantic term
    RegionCode      string     // ISO-3166 code
    ExchangeCode    *string    // ISO-10383 code (optional)
    HolidayName     *string    // Holiday name (optional)
    SourceType      string     // Where data came from
    ConfidenceScore int        // 0-100 quality indicator
    VersionID       int        // For time-travel queries
    CreatedAt       time.Time  // Audit
    UpdatedAt       time.Time  // Audit
}
```

## Survivorship Rules Engine

The rules engine implements a **Priority Hierarchy** to select the "winning" data source:

### Priority 1: Official Exchange Data (Confidence: 100)
- Most authoritative source
- Example: NYSE official holiday schedule

### Priority 2: Premium Vendors (Confidence: 80-90)
- Bloomberg or Refinitiv with latency < 24 hours
- Highly trusted, but secondary to official

### Priority 3: Internal Steward Override (Confidence: 85)
- Manual corrections by stewards
- Used for custom business rules

### Priority 4: Regional Default (Confidence: 50)
- Fallback when no higher priority exists
- Generic regional patterns

**Conflict Detection:**
- Flags when multiple high-confidence (>90) sources disagree
- Creates stewardship tasks for manual review

## Database Schema

### Tables

1. **mdm_calendar_golden** - The trusted Golden Record
   - Unique constraint: `(tenant_id, calendar_date, region_code, exchange_code)`
   - Versioned for time-travel

2. **mdm_calendar_source** - Raw ingestion staging
   - Stores every candidate from every source
   - Links to winning golden record

3. **mdm_calendar_lineage** - Full audit trail
   - Tracks how each semantic term was resolved
   - Captures rule applied, priority level, conflicts

4. **mdm_calendar_conflicts** - Stewardship queue
   - Open conflicts awaiting steward review
   - Escalation workflows

5. **mdm_calendar_versions** - Time-travel snapshots
   - Complete historical snapshots
   - Query "what was the calendar on 2024-01-01?"

### Row-Level Security (RLS)

All tables enforce tenant isolation via PostgreSQL RLS:
```sql
CREATE POLICY tenant_isolation ON mdm_calendar_golden
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
```

Application sets tenant context:
```go
// Before queries
conn.Exec(ctx, "SET app.current_tenant_id = $1", tenantID)
```

## API Endpoints

### REST API

#### Ingest Calendar Data
```bash
POST /api/v1/mdm/calendar/ingest
X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440001

{
  "source_system": "Bloomberg",
  "data": [
    {
      "date": "2024-12-25",
      "region": "US",
      "exchange": "XNYS",
      "is_business_day": false,
      "holiday_name": "Christmas"
    }
  ]
}
```

Response: `HTTP 202 Accepted`
```json
{
  "golden_record_ids": ["550e8400-e29b-41d4-a716-446655440099"],
  "count": 1,
  "ingested_at": "2024-02-20T15:30:00Z"
}
```

#### Get Golden Calendar
```bash
GET /api/v1/mdm/calendar/golden?start_date=2024-01-01&end_date=2024-12-31&region=US&exchange=XNYS
X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440001
```

Response:
```json
{
  "records": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440099",
      "calendar_date": "2024-12-25",
      "is_business_day": false,
      "region_code": "US",
      "exchange_code": "XNYS",
      "holiday_name": "Christmas",
      "source_type": "ExchangeFeed",
      "confidence_score": 100
    }
  ],
  "coverage_percentage": 98.5
}
```

#### Check if Date is Business Day
```bash
GET /api/v1/mdm/calendar/is-business-day?date=2024-12-25&region=US&exchange=XNYS
X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440001
```

Response:
```json
{
  "date": "2024-12-25",
  "region": "US",
  "exchange": "XNYS",
  "is_business_day": false
}
```

#### Get Lineage (Audit Trail)
```bash
GET /api/v1/mdm/calendar/lineage/550e8400-e29b-41d4-a716-446655440099
X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440001
```

Response:
```json
{
  "golden_record_id": "550e8400-e29b-41d4-a716-446655440099",
  "history": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440088",
      "semantic_term": "IsBusinessDay",
      "winning_value": "false",
      "winning_source_id": "550e8400-e29b-41d4-a716-446655440077",
      "rule_applied": "Priority 1: ExchangeOfficial (Confidence 100)",
      "execution_time": "2024-02-20T15:30:00Z",
      "conflict_detected": false
    }
  ]
}
```

#### Health Metrics
```bash
GET /api/v1/mdm/calendar/health
X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440001
```

Response:
```json
{
  "tenant_id": "550e8400-e29b-41d4-a716-446655440001",
  "coverage_percentage": 98.5,
  "conflict_count": 3,
  "high_confidence_percentage": 95.2,
  "days_since_last_official_feed": 0,
  "status": "healthy"
}
```

### GraphQL API

```graphql
query {
  getGoldenCalendar(
    startDate: "2024-01-01",
    endDate: "2024-12-31",
    region: "US",
    exchange: "XNYS"
  ) {
    id
    calendar_date
    is_business_day
    holiday_name
    confidence_score
  }
}
```

## Integration with Calendar Module

Your Calendar Module consumes the Golden Calendar:

```typescript
// src/services/CalendarService.ts
import { gql, useQuery } from '@apollo/client';

const GET_GOLDEN_CALENDAR = gql`
  query GetGoldenCalendar($start: date!, $end: date!, $region: String!) {
    getGoldenCalendar(start: $start, end: $end, region: $region) {
      calendar_date
      is_business_day
      holiday_name
      confidence_score
    }
  }
`;

export const useGoldenCalendar = (region: string, start: string, end: string) => {
  const { data, loading, error } = useQuery(GET_GOLDEN_CALENDAR, {
    variables: { start, end, region }
  });

  return {
    isBusinessDay: (date: string) => {
      const record = data?.getGoldenCalendar.find(d => d.calendar_date === date);
      return record ? record.is_business_day : true;
    },
    holidays: data?.getGoldenCalendar || [],
    loading,
    error
  };
};
```

## Deployment

### Local Development

```bash
# 1. Set up database
createdb semlayer
psql semlayer < database/migrations/mdm_calendar_schema.sql

# 2. Configure environment
cp mdm-service/.env.example mdm-service/.env
# Edit .env with local database URL

# 3. Build and run
cd mdm-service
go mod download
go run ./cmd/mdm-service/main.go
```

### Docker

```dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o mdm-service ./cmd/mdm-service

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/mdm-service /
EXPOSE 8080
CMD ["/mdm-service"]
```

### Kubernetes (Helm)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mdm-calendar-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mdm-calendar-service
  template:
    metadata:
      labels:
        app: mdm-calendar-service
    spec:
      containers:
      - name: mdm-service
        image: semlayer/mdm-calendar-service:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: mdm-secrets
              key: database-url
        resources:
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## Multi-Tenancy Strategy

### Tenant Isolation

1. **Application Layer**: All operations scoped by `X-Tenant-ID` header
2. **Database Layer**: PostgreSQL RLS policies enforce tenant boundaries
3. **Data Partitioning**: High-volume tenants can use table partitioning by region

```sql
ALTER TABLE mdm_calendar_golden
PARTITION BY LIST (region_code);

CREATE TABLE mdm_calendar_golden_us
  PARTITION OF mdm_calendar_golden
  FOR VALUES IN ('US');
```

### Tenant Provisioning

```bash
# Create new tenant
curl -X POST /api/v1/mdm/tenants \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "tenant_id": "550e8400-e29b-41d4-a716-446655440042",
    "name": "Acme Corp",
    "regions": ["US", "GB", "HK"]
  }'
```

## Operational Intelligence

### Health Scoring

**Coverage:** % of calendar days with records (target: >95%)

**Conflict Rate:** % flagged for stewardship (target: <5%)

**Staleness:** Days since last official feed (target: 0)

**Confidence Distribution:**
- High (80-100): >90%
- Medium (50-79): 5-10%
- Low (0-49): <5%

### Monitoring & Alerting

```promql
# Alert: Missing official data
mdm_calendar_staleness_days > 3

# Alert: High conflict rate
(mdm_calendar_conflicts_open / mdm_calendar_golden_count) > 0.05

# Alert: Low coverage
mdm_calendar_coverage_percentage < 0.90
```

## Extensibility

### Adding Custom Rules

```go
// Implement custom rule in rules/engine.go
func (re *RulesEngine) ExecuteCustomBusinessRule(ctx context.Context, execCtx ExecutionContext) (*RuleResult, error) {
    // Your logic here
}
```

### Loading DSL Rules (Future WASM Support)

```go
// Load external DSL
err := engine.LoadRuleFromDSL("marketing_fiscal_calendar", `
    RULE MarketingFiscalYear FOR HolidaySchedule.IsBusinessDay {
        PRIORITY 1: IF Department == "Marketing" AND FiscalQuarter == Q4
            THEN USE BusinessDay(12/15) CONFIDENCE 95;
    }
`)
```

## Troubleshooting

### Q: Why isn't my data appearing in the golden calendar?

A: Check:
1. Tenant ID matches (`X-Tenant-ID` header)
2. Date is in valid format (YYYY-MM-DD)
3. Region code is 2-character ISO-3166
4. No conflicts preventing adoption

```bash
curl /api/v1/mdm/calendar/lineage/{golden_id}
```

### Q: How do I resolve a conflict?

A: Steward reviews via conflict management UI:
```bash
# View conflicts
curl /api/v1/mdm/calendar/conflicts?status=open

# Resolve (coming soon)
curl -X POST /api/v1/mdm/calendar/conflicts/{id}/resolve \
  -d '{"winning_value": true, "notes": "Official exchange confirmed"}'
```

### Q: Can I query historical calendar data?

A: Yes! Use version history:
```bash
GET /api/v1/mdm/calendar/golden?start=2024-01-01&end=2024-12-31&version=5
```

## Contributing

1. Add tests to `internal/**/test.go`
2. Run: `go test ./...`
3. Update schema migrations if db changes
4. Submit PR with design doc

## License

Internal use only - Usice Architecture framework
