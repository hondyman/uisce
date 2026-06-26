# Incident Timeline Feature Documentation

## Overview

The **Incident Timeline** is a real-time event correlation and incident management system integrated into the SemLayer Ops Cockpit. It provides unified visibility into all operational events (alerts, health changes, errors, anomalies) and automatically correlates related events into incidents for trend analysis.

## Architecture

### Components

#### 1. **Database Schema** (`ops_events` & `ops_incidents` tables)
- **ops_events**: Unified event log capturing all operational events
  - 14 columns storing event metadata, severity, scope, and typed JSON details
  - Optimized indexes for time-range queries and incident correlation
  - Foreign key to ops_incidents for event-to-incident relationships

- **ops_incidents**: Incident grouping with lifecycle management
  - Tracks open/closed status, severity, and incident details
  - Stores summary and root cause analysis
  - Timestamps for started_at, ended_at, audit tracking

#### 2. **Backend Services**

**TimelineService** (`internal/ops/timeline.go`)
- Centralized event recording with automatic incident correlation
- Methods:
  - `RecordAlertEvent()` - Alert threshold violations
  - `RecordTenantHealthChange()` - Tenant health score transitions
  - `RecordEndpointHealthChange()` - Endpoint health changes
  - `RecordErrorFingerprint()` - Error pattern discovery
  - `RecordLatencyAnomaly()` - Latency spike detection

**Store Interface** (`internal/ops/store.go`)
- Methods for event/incident CRUD:
  - `InsertEvent()` - Add event to timeline
  - `ListEvents()` - Query events by time range
  - `GetIncident()` - Retrieve incident with all related events
  - `UpsertIncidentForEvent()` - Correlation logic
  - `CloseIncident()` - Mark incident as resolved

**PostgreSQL Store** (`internal/ops/store_postgres_timeline.go`)
- Full SQL implementation with parameterized queries (SQL injection safe)
- Incident correlation: 60-minute lookback window with scope-based matching
  - Tenant scope: Matches by tenant_id OR TenantHealth event type
  - Endpoint scope: Matches by endpoint_path OR EndpointHealth event type
  - Region scope: Matches by region field
  - Global scope: Matches by alert_id

#### 3. **API Handlers** (`internal/ops/handlers_timeline.go`)

**GET /admin/ops/timeline**
```bash
curl "http://localhost:8080/admin/ops/timeline?since=1h&limit=100"
```
- Query parameters:
  - `since`: Duration string (e.g., "1h", "24h", "7d") - default: 1 hour
  - `limit`: Max events to return (1-1000) - default: 200
- Response: `TimelineResponse` with events array and total count
- Ordering: Descending by occurred_at (newest first)

**GET /admin/ops/incidents/{incidentID}**
```bash
curl "http://localhost:8080/admin/ops/incidents/550e8400-e29b-41d4-a716-446655440000"
```
- Path parameter: `incidentID` (UUID)
- Response: `IncidentResponse` with incident details and all related events
- Ordering: Events sorted ascending by occurred_at (oldest first)

**POST /admin/ops/incidents/{incidentID}/close**
```bash
curl -X POST "http://localhost:8080/admin/ops/incidents/550e8400-e29b-41d4-a716-446655440000/close" \
  -H "Content-Type: application/json" \
  -d '{
    "summary": "Alert threshold misconfiguration",
    "root_cause": "Tenant limits set below actual usage"
  }'
```
- Path parameter: `incidentID` (UUID)
- Request body (optional): `{summary?: string, root_cause?: string}`
- Response: `{closed: true}`
- Updates: status→"closed", ended_at, summary, root_cause

#### 4. **Frontend Components**

**useOpsTimeline Hook** (`frontend/src/admin-v2/hooks/useOpsTimeline.ts`)
```typescript
interface UseOpsTimelineResult {
  isLoading: boolean;
  isError: boolean;
  data?: TimelineResponse;
  error?: Error;
}

const timeline = useOpsTimeline(since="1h", limit=200);
```
- React Query integration with 30-second auto-refetch
- Memoized queries keyed by [since, limit]
- Enabled flag for conditional fetching

**useOpsIncident Hook**
```typescript
const incident = useOpsIncident("550e8400-e29b-41d4-a716-446655440000");
```
- Fetches single incident with all events
- Enabled flag for conditional loading

**useCloseIncident Hook**
```typescript
const closeIncident = useCloseIncident();
await closeIncident.mutateAsync({
  incidentId: "550e8400-e29b-41d4-a716-446655440000",
  summary: "Issue resolved",
  rootCause: "Configuration corrected"
});
```
- Mutation hook for incident closure
- Integrates with React Query cache

**OpsTimeline Component** (`frontend/src/admin-v2/components/OpsTimeline.tsx`)
- Real-time timeline UI with filtering
- Features:
  - Severity-based filtering (All, Critical, Error, Warning, Info)
  - Dynamic event count badges per severity
  - Event type icons (🚨 alert, 👆 fingerprint, 🏢 health, 📈 anomaly, etc.)
  - Timeline layout with left markers + right content
  - Click handlers for incident navigation
  - Responsive design (mobile-friendly)
  - Empty state handling
  - Responsive CSS with color-coded severity indicators

## Data Model

### Event Types
```
alert                  - Alert threshold violation
fingerprint           - Error pattern discovery
tenant_health         - Tenant health score change
endpoint_health       - Endpoint health change
latency_anomaly      - P95 latency spike detection
incident_opened      - Incident creation event
incident_closed      - Incident closure event
```

### Severity Mapping (Health Score → Severity)
```
Health < 30   → CRITICAL (red #ef4444)
Health < 50   → ERROR    (orange #f87171)
Health < 70   → WARNING  (yellow #eab308)
Health ≥ 70   → INFO     (blue #3b82f6)
```

### Event Details (JSON field)
Each event type stores type-specific details in JSON:

**Alert Event**
```json
{
  "metric": "cpu",
  "threshold": 80,
  "current_value": 92,
  "exceeded_by": 12
}
```

**Health Change Event**
```json
{
  "old_score": 85,
  "new_score": 45,
  "degradation": 40
}
```

**Error Fingerprint Event**
```json
{
  "error_message": "connection timeout",
  "stack_trace": "...",
  "affected_requests": 42
}
```

**Latency Anomaly Event**
```json
{
  "p95_latency": 850,
  "baseline": 200,
  "increase_percent": 325
}
```

## Integration Points

### For Alert System
```go
import "github.com/semlayer/backend/internal/ops"

// In alert evaluation
ts := ops.NewTimelineService(store)
err := ts.RecordAlertEvent(ctx, alert, overagePercentage)
```

### For Health Calculator
```go
// When tenant health score changes
oldScore := 82
newScore := 45
ts.RecordTenantHealthChange(ctx, tenantID, oldScore, newScore)
```

### For Endpoint Health
```go
// When endpoint health changes
ts.RecordEndpointHealthChange(ctx, "/api/users", oldScore, newScore)
```

### For Error Fingerprinting
```go
// When fingerprint is discovered
ts.RecordErrorFingerprint(ctx, fingerprint)
```

### For Anomaly Detection
```go
// When latency anomaly detected
ts.RecordLatencyAnomaly(ctx, endpoint, p95Latency, baselineLatency)
```

## Incident Correlation

### Correlation Strategy
The system implements **scope-aware incident grouping** with a 60-minute lookback window:

1. **Query scope matching** based on event type:
   - **Tenant scope**: Events with same tenant_id OR TenantHealth type
   - **Endpoint scope**: Events with same endpoint_path OR EndpointHealth type
   - **Region scope**: Events with same region field
   - **Global scope**: Events with same alert_id

2. **Matching logic**:
   ```
   IF open incident exists in scope with matching field
     THEN attach event to existing incident
     ELSE create new incident
   ```

3. **Incident lifecycle**:
   - **Created**: When first event occurs (status="open")
   - **Updated**: When new event arrives within 60 minutes
   - **Closed**: Via API with optional summary/root_cause

### Example: Tenant Incident Correlation
```
T=0:05    → Alert: CPU > 80%        → Incident #1 (open)
T=0:18    → TenantHealth degraded   → Correlated to Incident #1
T=0:42    → Error spike             → Correlated to Incident #1
T=1:15    → New anomaly             → New Incident #2 (>60min elapsed)
```

## Usage Examples

### Frontend: Display Timeline in Dashboard
```typescript
<OpsTimeline 
  since="24h" 
  limit={150} 
  onEventClick={(event) => navigateToIncident(event.incident_id)}
/>
```

### Frontend: Check Specific Incident
```typescript
const incident = useOpsIncident(incidentId);
if (incident.data?.incident) {
  console.log(`Incident: ${incident.data.incident.title}`);
  console.log(`Events: ${incident.data.incident.events?.length}`);
}
```

### Frontend: Close Incident with Analysis
```typescript
const closeIncident = useCloseIncident();
await closeIncident.mutateAsync({
  incidentId: "550e8400-e29b-41d4-a716-446655440000",
  summary: "CPU spike resolved by autoscaling",
  rootCause: "Tenant workload spike triggered by daily batch job"
});
```

### API: Stream Recent Critical Events
```bash
curl -s "http://localhost:8080/admin/ops/timeline?since=1h&limit=50" | jq '.events[] | select(.severity=="critical")'
```

## Performance Considerations

### Indexes
- `ops_events(occurred_at DESC)` - Primary query path for timeline
- `ops_events(incident_id)` - Fast incident event lookup
- `ops_events(event_type, severity)` - Filtering support
- `ops_incidents(status, started_at DESC)` - Incident queries
- `ops_incidents(severity)` - Severity-based filtering

### Query Optimization
- Time-range queries limited by `since` parameter (default 1h)
- Event limit capped at 1000
- Incident queries use indexed lookups
- React Query caching with 30s refetch interval

### Data Retention
- Consider archiving events older than 90 days
- Keep open incidents in hot storage
- Consider partitioning ops_events table by date for large deployments

## Troubleshooting

### Events Not Appearing
1. Verify timeline service is initialized in server startup
2. Check database connectivity: `select count(*) from ops_events;`
3. Verify event recording is called in relevant services
4. Check event timestamps are within query range

### Incidents Not Correlating
1. Check incident lookup window: 60 minutes by default
2. Verify scope matching in UpsertIncidentForEvent
3. Check existing incident status is "open"
4. Review PostgreSQL query results in store_postgres_timeline.go

### API Timeouts
1. Reduce `limit` parameter or increase `since` time range
2. Check database indexes on `ops_events(occurred_at)`
3. Consider partitioning for very large event tables
4. Add database connection pool monitoring

## Future Enhancements

1. **Alerting on Incidents**: Notify teams on incident creation
2. **Incident Severity Auto-Update**: Escalate based on correlated events
3. **Custom Correlation Rules**: Allow per-tenant correlation logic
4. **Analytics**: Timeline trend analysis and incident patterns
5. **Export/Integration**: Send to external incident tracking systems
6. **Machine Learning**: Anomaly detection for correlation tuning

## Related Files

**Backend**
- `migrations/20260208_create_ops_timeline.up.sql` - Schema
- `internal/ops/event.go` - Domain types
- `internal/ops/timeline.go` - Service layer
- `internal/ops/handlers_timeline.go` - HTTP handlers
- `internal/ops/store_postgres_timeline.go` - Store implementation

**Frontend**
- `frontend/src/admin-v2/hooks/useOpsTimeline.ts` - React hooks
- `frontend/src/admin-v2/components/OpsTimeline.tsx` - Timeline component
- `frontend/src/admin-v2/components/OpsTimeline.css` - Styling
- `frontend/src/admin-v2/pages/GlobalOpsDashboard.tsx` - Integration

## API Reference

### Event Type Constants
```go
const (
  EventTypeAlert           = "alert"
  EventTypeFingerprint     = "fingerprint"
  EventTypeTenantHealth    = "tenant_health"
  EventTypeEndpointHealth  = "endpoint_health"
  EventTypeLatencyAnomaly  = "latency_anomaly"
  EventTypeIncidentOpened  = "incident_opened"
  EventTypeIncidentClosed  = "incident_closed"
)
```

### Severity Constants
```go
const (
  SeverityInfo     = "info"
  SeverityWarning  = "warning"
  SeverityError    = "error"
  SeverityCritical = "critical"
)
```

### Response Structures

**TimelineResponse**
```typescript
{
  events: OpsEvent[],
  total: number
}
```

**IncidentResponse**
```typescript
{
  incident: OpsIncident,
  events: OpsEvent[]
}
```

**OpsEvent**
```typescript
{
  id: string
  incident_id?: string
  event_type: string
  scope: "global" | "tenant" | "endpoint" | "region"
  tenant_id?: string
  endpoint_path?: string
  region?: string
  fingerprint_id?: string
  alert_id?: string
  severity: "info" | "warning" | "error" | "critical"
  title: string
  details: Record<string, any>
  occurred_at: string
  created_at: string
}
```

**OpsIncident**
```typescript
{
  id: string
  status: "open" | "closed"
  severity: "info" | "warning" | "error" | "critical"
  title: string
  summary?: string
  root_cause?: string
  started_at: string
  ended_at?: string
  created_at: string
  updated_at: string
  events?: OpsEvent[]
}
```
