# Audit Explorer - Complete Implementation Guide

## Overview

The Audit Explorer is a unified, role-aware audit surface that answers the question: **"What actually happened?"** across your entire platform.

### Key Features

- **Multi-Role Support**: Global Admin, Global Ops, Tenant Admin, Tenant Ops with tailored views
- **Unified Timeline**: Single query showing all events (jobs, DAGs, changes, compliance)
- **AI Explanations**: Tenant-scoped root cause analysis and recommendations
- **Incident Clustering**: Groups related failures with blast radius and SLO impact
- **Compliance Tracking**: Violation detection, remediation path, and audit trail
- **Entity Audit**: Event history for specific semantic terms, jobs, or DAGs
- **Multi-Tenant Safe**: Tenant scope enforced at service, handler, and query layers

## Architecture

### Backend Stack (Go)

```
backend/internal/audit/
├── explorer_models.go          # Domain models (AuditEvent, EntityAudit, etc.)
├── explorer_repository.go      # Data access abstraction + Trino implementation
├── explorer_service.go         # Business logic + role enforcement
├── explorer_handler.go         # HTTP handlers
├── explorer_rbac.go            # Role-based access control
└── trino_queries.go            # Trino query builders with UNION queries
```

**Key Models:**
- `AuditEvent`: Unified audit record with semantic/compliance context
- `EntityAudit`: All events touching a specific entity
- `IncidentCluster`: Grouped failures with AI root cause
- `ComplianceEvent`: Compliance violation with remediation path
- `Dashboard`: Role-specific metrics and summaries

### Frontend Stack (React + TypeScript)

```
frontend/src/
├── components/audit/
│   ├── AuditExplorer.tsx       # Main container with role-aware tabs
│   ├── FilterBar.tsx           # Unified filter component
│   ├── tabs/
│   │   ├── TimelineView.tsx    # Unified timeline (5-way UNION)
│   │   ├── EntitiesView.tsx    # Entity-centric audit trail
│   │   ├── IncidentsView.tsx   # Grouped failures with AI analysis
│   │   └── ComplianceView.tsx  # Compliance violations
│   └── panels/
│       └── AIPanel.tsx         # AI explanation side panel
├── hooks/
│   └── useAuditExplorer.ts    # Custom hook for data fetching
└── contexts/
    ├── AuthContext.tsx        # User role + permissions
    └── TenantContext.tsx      # Selected tenant scope
```

## API Endpoints

### Timeline Events
```http
POST /api/audit-explorer/events
Content-Type: application/json
X-Tenant-ID: <tenant_id>

{
  "tenantFilter": ["tenant-001", "tenant-002"],
  "from": "2026-01-10T00:00:00Z",
  "to": "2026-01-17T00:00:00Z",
  "artifactTypes": ["job_run", "changeset"],
  "statuses": ["failed", "success"],
  "riskLevels": ["high", "medium"],
  "limit": 50,
  "offset": 0
}

Response:
{
  "events": [
    {
      "id": "evt-001",
      "type": "job_run",
      "tenantId": "tenant-001",
      "timestamp": "2026-01-10T10:30:00Z",
      "status": "failed",
      "riskLevel": "high",
      "artifactId": "job-123",
      "artifactType": "job",
      "actor": "scheduler",
      "message": "Job failed after 5 retries",
      "semanticContext": { },
      "complianceContext": { },
      "aiNarrative": "Database connection timeout..."
    }
  ],
  "total": 245
}
```

### Entity Audit Trail
```http
GET /api/audit-explorer/entities/{entityType}/{entityID}?from=...&to=...&limit=50&offset=0
X-Tenant-ID: <tenant_id>

Response:
{
  "entityId": "semantic-term-42",
  "entityType": "semantic_term",
  "firstSeen": "2025-12-01T00:00:00Z",
  "lastSeen": "2026-01-17T14:22:00Z",
  "timeline": [ /* all events touching this entity */ ],
  "changes": [ /* changesets affecting this entity */ ],
  "compliance": [ /* compliance events */ ],
  "aiInsights": {
    "summary": "This semantic term was modified 12 times...",
    "riskFactors": [ ]
  }
}
```

### Incidents (Grouped Failures)
```http
GET /api/audit-explorer/incidents?from=...&to=...&limit=50&offset=0
X-Tenant-ID: <tenant_id>

Response:
{
  "incidents": [
    {
      "id": "incident-001",
      "timeWindow": {
        "start": "2026-01-15T08:00:00Z",
        "end": "2026-01-15T10:30:00Z"
      },
      "affectedTenants": ["tenant-001", "tenant-002"],
      "affectedJobs": ["job-123", "job-456"],
      "affectedDAGs": ["dag-78"],
      "failureCount": 47,
      "aiRootCause": "Database replication lag exceeded 30s threshold...",
      "blastRadius": "2 tenants, 15 jobs, 1 DAG",
      "sloImpact": {
        "failed_slas": 8,
        "estimated_impact": 0.15
      }
    }
  ]
}
```

### Compliance Events
```http
GET /api/audit-explorer/compliance-events?from=...&to=...&violationTypes=pii_exposure&limit=50&offset=0
X-Tenant-ID: <tenant_id>

Response:
{
  "events": [
    {
      "id": "compliance-001",
      "tenantId": "tenant-001",
      "timestamp": "2026-01-10T14:22:00Z",
      "violationType": "pii_exposure",
      "severity": "critical",
      "affectedRecords": 1250,
      "status": "open",
      "artifact": {
        "type": "database_table",
        "id": "customers"
      },
      "narrative": "Unmasked email column detected in customer data export",
      "remediationPath": "Apply PII masking policy to customers table"
    }
  ]
}
```

### AI Explanation
```http
POST /api/audit-explorer/explain
Content-Type: application/json
X-Tenant-ID: <tenant_id>

{
  "tenantScope": ["tenant-001"],
  "auditRecords": [ /* selected events */ ],
  "entityContext": { /* optional context */ }
}

Response:
{
  "rootCause": "Database connection pool exhaustion due to unoptimized query...",
  "timeline": "15:20 - Job starts\n15:22 - Database connections climb...",
  "affectedSystems": ["database-primary", "cache-redis"],
  "recommendations": [
    "Add connection pooling timeout",
    "Optimize query to use index on timestamp"
  ],
  "riskAssessment": {
    "level": "high",
    "description": "Affects 2 critical jobs, risk of SLA miss"
  },
  "relatedEvents": [ /* correlated events */ ]
}
```

### Dashboards (Role-Specific)
```http
GET /api/audit-explorer/dashboard/global-admin?from=...&to=...
GET /api/audit-explorer/dashboard/global-ops?from=...&to=...
GET /api/audit-explorer/dashboard/tenant-admin/{tenantID}?from=...&to=...
GET /api/audit-explorer/dashboard/tenant-ops/{tenantID}?from=...&to=...

Response:
{
  "dashboardType": "global_admin",
  "metrics": {
    "totalEvents": 5420,
    "failureRate": 0.08,
    "complianceViolations": 12,
    "incidentCount": 3,
    "avgResolutionTime": "2.5h"
  },
  "topFailingJobs": [ ],
  "complianceTrend": { },
  "tenantHealthMatrix": [ ]
}
```

## Role-Based Access Control

### Permission Matrix

| Feature | Global Admin | Global Ops | Tenant Admin | Tenant Ops |
|---------|---|---|---|---|
| View all tenants | ✅ | ❌ | ❌ | ❌ |
| View Timeline | ✅ | ✅ | ✅ | ✅ |
| View Entities | ✅ | ✅ | ✅ | ❌ |
| View Incidents | ✅ | ✅ | ✅ | ✅ |
| View Compliance | ✅ | ✅ | ✅ | ❌ |
| AI Explain (cross-tenant) | ✅ | assigned tenants | ❌ | ❌ |
| Approve Changes | ✅ | medium-risk | ✅ | ❌ |
| Access Dashboard | global_admin | global_ops | tenant_admin | tenant_ops |

### TenantScope Enforcement

Every request:
1. Extracts `allowed_tenants` from context (auth.AllowedTenantsFromContext)
2. Intersects with request `tenantFilter`
3. Rejects if no overlap (HTTP 403)
4. Enforces intersection at query layer (WHERE tenant_id IN (...))

**Example:**
```go
// Service layer
func (s *ExplorerService) ListEvents(ctx context.Context, req *ListEventsRequest) {
    allowedTenants := auth.AllowedTenantsFromContext(ctx)
    req.TenantFilter = allowedTenants.Intersect(TenantScope(req.TenantFilter))
    
    if len(req.TenantFilter) == 0 {
        return nil, fmt.Errorf("no accessible tenants for this request")
    }
    
    // Proceed with query using filtered tenant list
}
```

## Tab Visibility Rules

### Global Admin
- Timeline, Entities, Incidents, Compliance
- No tenant filter restriction (can see all)
- Can correlate events across tenants

### Global Ops
- Timeline, Entities, Incidents, Compliance
- Limited to assigned tenants
- Can approve medium-risk changes

### Tenant Admin
- Timeline, Entities, Incidents, Compliance
- Single tenant only
- Can approve all changes within tenant

### Tenant Ops
- Timeline, Incidents only
- Single tenant only
- Read-only view

## Frontend Integration

### Setup

1. **Install Dependencies** (already done)
```bash
npm install @mui/material @mui/icons-material
```

2. **Import Contexts** in App.tsx:
```tsx
import { AuthProvider } from './contexts/AuthContext';
import { TenantProvider } from './contexts/TenantContext';

function App() {
  return (
    <AuthProvider>
      <TenantProvider>
        {/* App content */}
      </TenantProvider>
    </AuthProvider>
  );
}
```

3. **Add Route** in Router:
```tsx
import AuditExplorer from './components/audit/AuditExplorer';

<Route path="/audit-explorer" element={<AuditExplorer />} />
```

4. **Add Navigation Link** in MainNavigation:
```tsx
{hasRole('global_admin', 'global_ops', 'tenant_admin', 'tenant_ops') && (
  <NavLink to="/audit-explorer">Audit Explorer</NavLink>
)}
```

### Component Usage

```tsx
import AuditExplorer from '@/components/audit/AuditExplorer';

// Component automatically:
// - Reads user role from auth context
// - Reads selected tenant from tenant context
// - Enforces role-based visibility rules
// - Fetches role-specific data
// - Scopes AI explanations to tenant

export function AuditPage() {
  return <AuditExplorer />;
}
```

## Backend Integration

### 1. Register Routes in api.go

```go
func (a *APIServer) registerAuditExplorerRoutes(r chi.Router) {
    // Initialize repository
    trinoRepo := audit.NewTrinoRepository(a.db)
    
    // Initialize service
    svc := audit.NewExplorerService(trinoRepo, a.aiClient)
    
    // Initialize handler
    handler := audit.NewExplorerHandler(svc)
    
    // Register routes under /api/audit-explorer
    r.Route("/audit-explorer", func(r chi.Router) {
        r.Use(audit.TenantScopeMiddleware)
        handler.RegisterRoutes(r)
    })
}
```

### 2. Middleware Registration

```go
// In main router initialization
r.Use(auth.AuthMiddleware)
r.Use(audit.RoleBasedAccessMiddleware("global_admin", "global_ops", "tenant_admin", "tenant_ops"))
```

### 3. Trino Connection

The `explorer_repository.go` assumes:
- Existing `sql.DB` with "trino" driver
- Database: `alpha`
- Catalog: `iceberg`
- Schema: `audit`

Tables used:
- `scheduler_job_runs`
- `scheduler_dag_runs`
- `governance_changesets`
- `semantic_snapshots`
- `compliance_violations`
- `orchestration_events` (optional)

### 4. AI Client Setup

```go
// In explorer_service.go
type ExplorerService struct {
    repo     Repository
    aiClient AIClient  // Interface for any AI vendor
}

interface AIClient {
    ExplainAuditEvents(ctx context.Context, req *ExplainRequest) (*ExplainResponse, error)
}
```

## Query Optimization

### Partition Pruning

All Trino queries use partition keys:
- `tenant_id` - Always filtered
- `date` - Always filtered by time range

Example:
```sql
WHERE tenant_id IN (?, ?, ?)
  AND date BETWEEN CAST(? AS date) AND CAST(? AS date)
```

### UNION Query Strategy

Timeline combines 5 sources in single query:
```sql
SELECT ... FROM scheduler_job_runs WHERE ...
UNION ALL
SELECT ... FROM scheduler_dag_runs WHERE ...
UNION ALL
SELECT ... FROM governance_changesets WHERE ...
UNION ALL
SELECT ... FROM semantic_snapshots WHERE ...
UNION ALL
SELECT ... FROM compliance_violations WHERE ...
ORDER BY timestamp DESC
LIMIT ? OFFSET ?
```

This is more efficient than:
- 5 separate queries + client merge
- Queries without partition keys
- Unfiltered tenant scopes

## Testing

### Unit Tests

```go
// Test role enforcement
func TestExplorerServiceRoleEnforcement(t *testing.T) {
    // Create tenant_ops context
    ctx := auth.WithTenants(context.Background(), []string{"tenant-001"})
    
    // Verify service rejects access to other tenants
    req := &ListEventsRequest{TenantFilter: []string{"tenant-002"}}
    _, err := svc.ListEvents(ctx, req)
    assert.Error(t, err)  // Should be forbidden
}
```

### Integration Tests

```go
// Test API endpoint with tenant scope
func TestListEventsEndpoint(t *testing.T) {
    // Setup
    req := httptest.NewRequest("POST", "/api/audit-explorer/events", ...)
    req.Header.Set("X-Tenant-ID", "tenant-001")
    
    // Execute
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)
    
    // Verify response contains only tenant-001 data
    assert.Equal(t, http.StatusOK, w.Code)
}
```

### Frontend Tests

```tsx
// Test role-based tab visibility
it('should show only Timeline and Incidents for tenant_ops', () => {
  const { getByText } = render(
    <AuditExplorer role="tenant_ops" />
  );
  
  expect(getByText('Timeline')).toBeInTheDocument();
  expect(getByText('Incidents')).toBeInTheDocument();
  expect(queryByText('Entities')).not.toBeInTheDocument();
  expect(queryByText('Compliance')).not.toBeInTheDocument();
});
```

## Security Considerations

### 1. Tenant Scope Validation

✅ Every layer validates tenant scope:
- HTTP handlers check permissions
- Service enforces tenant intersection
- Repository applies WHERE tenant_id IN (...)
- Trino query scoped to specific partitions

### 2. AI Explanation Safety

✅ AI prompts include:
- Explicit tenant scope constraints
- Instruction not to correlate across tenants
- Semantic context only (no raw data)
- Compliance context obfuscated

### 3. Field Masking

- Compliance violations obfuscate affected record details
- Audit narratives redact sensitive paths
- Error messages sanitized for non-admin users

### 4. Rate Limiting

- Apply per-tenant rate limits
- Dashboard queries cached (5min TTL)
- Explain requests limited (1/sec per tenant)

## Performance Tuning

### Caching Strategy

```
Timeline (7-day range): 1-minute cache
Incidents (24-hour): 5-minute cache
Dashboards (hourly): 1-hour cache
Entity Audit (on-demand): No cache (real-time)
```

### Query Optimization Checklist

- [x] Partition pruning by tenant_id + date
- [x] UNION all sources in single query
- [x] Use LIMIT/OFFSET for pagination
- [x] Index on (tenant_id, timestamp)
- [x] Materialized view for SLO summary (future)

## Future Enhancements

### Phase 2 (Planned)

1. **Real-Time Updates**
   - WebSocket for incident stream
   - Live update to timeline

2. **Advanced Search**
   - Full-text search across narratives
   - Regex filtering on artifact IDs
   - Saved search filters per role

3. **Custom Dashboards**
   - Tenant admins can create custom metrics
   - Drag-and-drop widget builder
   - Export to Grafana

4. **Audit Export**
   - Export timeline to CSV
   - Integration with SIEM tools
   - Compliance report generation

5. **Drill-Down**
   - Click event → jump to source logs
   - Integration with trace viewer
   - SQL query replay

## Troubleshooting

### Common Issues

**Issue: "No accessible tenants for this request"**
- Check auth context has tenants set
- Verify user role has access
- Check request doesn't specify restricted tenants

**Issue: AI explanations are slow**
- AI client timeout? Check network
- Model is busy? Implement queue
- Token budget exceeded? Reduce context size

**Issue: Trino queries timeout**
- Check partition filters are applied
- Verify index exists on (tenant_id, timestamp)
- Check if LIMIT is too high (keep ≤ 100)

**Issue: No data appears for tenant_ops role**
- Verify incidents tab is visible (only tab for tenant_ops besides timeline)
- Check incidents exist in time range
- Verify tenant_id matches user's scope
