# Observability Console Integration Guide

## Overview

The SemLayer Observability Console is a comprehensive, Material UI-based observability suite that provides:

- **Commit Path Trace Explorer** — Visualize distributed traces (Planner → Temporal → Commit → Trino)
- **Region Health Heatmap** — Monitor regional performance degradation
- **Per-Tenant Observability Dashboard** — Track tenant-level metrics and health
- **Plan Timeline** — Chronological view of ingestion activity
- **Snapshot Lineage** — Iceberg snapshot DAG visualization
- **Compare Plans** — Side-by-side plan analysis for debugging
- **RBAC Guards** — Role-based access control integration

---

## Architecture

### Directory Structure

```
ui/
├── pages/
│   ├── ObservabilityConsole.tsx      # Main search + tabs page
│   ├── ComparePlansView.tsx          # Side-by-side plan diff
│   ├── RegionHeatmapPage.tsx         # Region health heatmap
│   ├── TenantObservabilityPage.tsx   # Per-tenant metrics dashboard
│   ├── PlanTimelinePage.tsx          # Chronological plan timeline
│   ├── SnapshotLineagePage.tsx       # Iceberg snapshot lineage
│   └── GlobalHealthOverview.tsx      # System-wide health KPIs
│
├── components/
│   ├── observability/
│   │   ├── CommitPathTraceExplorer.tsx  # Existing (reused)
│   │   ├── RecentPlansTable.tsx         # MUI DataGrid with recent plans
│   │   ├── RegionHeatmap.tsx            # Heatmap visualization
│   │   ├── PlannerDecisionGraph.tsx     # Planner routing visualization
│   │   ├── PlannerExplainPanel.tsx      # Existing (reused)
│   │   ├── MetricsPanel.tsx             # Existing (reused)
│   │   ├── LogsPanel.tsx                # Existing (reused)
│   │   ├── TracesPanel.tsx              # Existing (reused)
│   │   │
│   │   └── api/
│   │       ├── fetchRegionHeatmap.ts
│   │       ├── fetchPlanTimeline.ts
│   │       ├── fetchGlobalMetrics.ts
│   │       ├── tenant.ts
│   │       ├── fetchExplain.ts           # Existing (reused)
│   │       ├── fetchMetrics.ts           # Existing (reused)
│   │       └── fetchTrace.ts             # Existing (reused)
│   │
│   ├── rbac/
│   │   └── RequireRole.tsx           # RBAC wrapper component
│   │
│   └── navigation/
│       └── ObservabilityNav.tsx      # Sidebar navigation items
│
├── routes/
│   └── ObservabilityRoutes.tsx       # Route definitions with RBAC
│
└── stories/
    ├── ObservabilityConsole.stories.tsx
    ├── ObservabilityPages.stories.tsx
    └── ObservabilityComponents.stories.tsx
```

### Route Structure

```
/admin/observability/                  # Global health overview (requires: sre, admin)
/admin/observability/plan/:planId      # Plan detail with tabs (requires: sre, admin)
/admin/observability/compare            # Compare plans side-by-side (requires: sre, admin)
/admin/observability/regions            # Region health heatmap (requires: sre, admin)
/admin/observability/tenant/:tenantId   # Tenant-specific metrics (requires: admin, tenant_admin)
/admin/observability/timeline           # Plan timeline view (requires: sre, admin)
```

---

## Integration Steps

### Step 1: Add Routes to Admin App

In your main `App.tsx` or routing file:

```tsx
import { ObservabilityRoutes } from "./routes/ObservabilityRoutes";
import { useAuth } from "./hooks/useAuth"; // Your auth hook

export function AdminApp() {
  const { user } = useAuth();
  
  return (
    <Routes>
      {/* ... other admin routes */}
      
      <Route path="/observability/*" element={<ObservabilityRoutes currentUser={user} />} />
    </Routes>
  );
}
```

### Step 2: Add Navigation

In your admin sidebar/drawer component:

```tsx
import { ObservabilityNav } from "./components/navigation/ObservabilityNav";

export function AdminSidebar() {
  return (
    <Drawer>
      {/* ... other nav items */}
      <ObservabilityNav />
    </Drawer>
  );
}
```

### Step 3: Implement Backend API Endpoints

The Observability Console expects these backend endpoints:

#### Global Metrics
```
GET /api/metrics/global
Response: {
  commitSuccessRate: string,
  s3Failures5m: number,
  idempotencyHits5m: number,
  regionsDegraded: number,
  avgCommitLatencyMs: number,
  p95CommitLatencyMs: number,
  activeRegions: number
}
```

#### Region Heatmap
```
GET /api/metrics/region-heatmap
Response: Array<{
  region: string,
  bucket: string,        // e.g., "now", "5m ago", "10m ago"
  value: number          // latency in ms
}>
```

#### Tenant Metrics
```
GET /api/metrics/tenant/{tenantId}
Response: {
  successRate: string,
  s3Failures: number,
  idempotencyHits: number,
  avgLatencyMs: number
}
```

#### Tenant Plans
```
GET /api/plans?tenant={tenantId}&limit={limit}
Response: Array<{
  id: string,
  table: string,
  region: string,
  status: "success" | "degraded" | "failed",
  latency: number,
  timestamp: string
}>
```

#### Plan Timeline
```
GET /api/plans/timeline?limit={limit}
Response: Array<{
  planId: string,
  table: string,
  region: string,
  status: "pending" | "success" | "degraded" | "failed",
  latency: number,
  timestamp: string
}>
```

#### Snapshot Lineage
```
GET /api/iceberg/lineage?table={table}
Response: Array<{
  snapshotId: number,
  parentSnapshotId?: number,
  timestamp: string,
  fileCount: number,
  dataBytes: number
}>
```

### Step 4: Configure User Context

Update `RequireRole.tsx` to integrate with your auth system:

```tsx
// In your auth hook or context
export interface UserContext {
  role: "admin" | "sre" | "tenant_admin" | "viewer";
  tenantId?: string;
}

// Pass user to ObservabilityRoutes
<ObservabilityRoutes currentUser={currentUser} />
```

### Step 5: Add Storybook Stories (Optional)

For local development and component documentation:

```bash
cd ui
npm run storybook
```

Stories are located in `stories/` directory:
- `ObservabilityConsole.stories.tsx` — Main page
- `ObservabilityPages.stories.tsx` — Overview pages
- `ObservabilityComponents.stories.tsx` — Individual components

---

## Features Breakdown

### 1. ObservabilityConsole (Main Page)

**Location:** `/admin/observability/plan/:planId`

**Features:**
- Search bar (plan_id, table, region, tenant_id)
- Recent plans table with quick view/compare
- Tabbed interface:
  - Commit Path Trace Explorer
  - Planner Explain
  - Metrics
  - Logs
  - Traces
  - Planner Decision Graph

**Usage:**
```tsx
<ObservabilityConsole />
```

### 2. ComparePlansView

**Location:** `/admin/observability/compare?left=PLAN_A&right=PLAN_B`

**Features:**
- Side-by-side Commit Path Trace Explorer
- Same tabs as single plan view
- Perfect for debugging regressions

### 3. RegionHeatmapPage

**Location:** `/admin/observability/regions`

**Features:**
- Color-coded heatmap (green/yellow/orange/red)
- Regions × time buckets
- Responsive to commit latency metric

**Color Scheme:**
- Green: < 50ms (ok)
- Yellow: 50-200ms (warning)
- Orange: 200-500ms (degraded)
- Red: > 500ms (critical)

### 4. TenantObservabilityPage

**Location:** `/admin/observability/tenant/:tenantId`

**Features:**
- 4-card KPI dashboard (success rate, S3 failures, idempotency hits, latency)
- Recent plans for tenant
- RBAC restricted to admin + tenant_admin roles

### 5. PlanTimelinePage

**Location:** `/admin/observability/timeline`

**Features:**
- Vertical timeline of recent plans
- Status-colored dots (green/yellow/red)
- Hover for latency details
- Chronological incident review

### 6. SnapshotLineagePage

**Location:** `/admin/observability/lineage/:table`

**Features:**
- Iceberg snapshot DAG
- Parent/child snapshot relationships
- Snapshot metadata (file count, data size)
- Foundation for future React Flow upgrade

### 7. GlobalHealthOverview

**Location:** `/admin/observability` (landing page)

**Features:**
- System-wide KPI dashboard
- 7-card overview (success rate, failures, idempotency, regions degraded, latencies)
- Quick health check

---

## RBAC Configuration

### Role Definitions

| Role | Access |
|------|--------|
| `admin` | All observability features |
| `sre` | Observability Console, Compare Plans, Region Heatmap, Timeline |
| `tenant_admin` | Only their tenant's observability dashboard |
| `viewer` | Read-only access (future: logs, traces only) |

### Usage

```tsx
<RequireRole role={["sre", "admin"]} user={currentUser}>
  <ObservabilityConsole />
</RequireRole>
```

---

## API Integration Checklist

- [ ] Backend `/api/metrics/global` endpoint
- [ ] Backend `/api/metrics/region-heatmap` endpoint
- [ ] Backend `/api/metrics/tenant/{tenantId}` endpoint
- [ ] Backend `/api/plans?tenant={tenantId}` endpoint
- [ ] Backend `/api/plans/timeline` endpoint
- [ ] Backend `/api/iceberg/lineage?table={table}` endpoint
- [ ] Prometheus queries for global/tenant/region metrics
- [ ] RBAC enforcement on all endpoints

---

## Customization

### Change Color Scheme

Edit `RegionHeatmap.tsx` `getColor()` function:

```tsx
const getColor = (value: number) => {
  if (value > 600) return "#c62828";   // Custom red
  if (value > 300) return "#ff6f00";   // Custom orange
  if (value > 100) return "#f57f17";   // Custom yellow
  return "#2e7d32";                    // Custom green
};
```

### Add Custom Tabs to ObservabilityConsole

Edit `ObservabilityConsole.tsx`:

```tsx
<Tab label="Custom Tab" />

// In content section:
{activeTab === 6 && <YourCustomComponent planId={planId} />}
```

### Extend RBAC

Update `RequireRole.tsx` and `UserContext`:

```tsx
export type Role = "admin" | "sre" | "tenant_admin" | "viewer" | "analyst";

// Add new routes with analyst role
<RequireRole role={["analyst", "admin"]} user={user}>
  <AnalyticsPage />
</RequireRole>
```

---

## Performance Considerations

### Pagination

Recent plans table supports server-side pagination:

```
GET /api/plans?page=0&pageSize=50&filter=table:orders
```

### Caching

Consider caching endpoints with `SWR` or `React Query`:

```tsx
import useSWR from "swr";

function TenantObservabilityPage({ tenantId }) {
  const { data: metrics } = useSWR(
    `/api/metrics/tenant/${tenantId}`,
    fetcher,
    { revalidateOnFocus: false }
  );
  // ...
}
```

### Lazy Loading

Pages are already lazy-loaded via React Router code splitting.

---

## Troubleshooting

### CORS Issues

If observability pages fail to load cross-origin APIs:
- Verify backend CORS headers
- Use backend proxy pattern (proxy through `/api/` routes)

### Missing Metrics

If heatmap/timeline shows no data:
- Check Prometheus PromQL queries
- Verify backend is configured with `PROMETHEUS_URL`
- Review mock data fallbacks

### RBAC Not Working

- Confirm `UserContext` is passed to `ObservabilityRoutes`
- Check user role value matches `RequireRole` expectations
- Review browser console for 401/403 errors

---

## Next Steps (Future Priorities)

### Priority 1: Multi-Tenant Isolation
- Filter all views by current user's tenant
- Add tenant selector for admins

### Priority 2: Alerting Integration
- Wire alerts to `/api/alerts` endpoint
- Show active alerts in global health overview

### Priority 3: Query Lineage Visualization
- Display query DAG (Planner → Temporal → Commit → Trino)
- Add interactive drill-down

### Priority 4: SLO/SLA Dashboards
- Error budgets per service
- Trend visualization (latency, success rate, etc.)

### Priority 5: Snapshot Lineage Upgrade
- Integrate React Flow for full DAG visualization
- Add snapshot comparison view

---

## Support

For questions or issues:
1. Check mock data in page components (fallback behavior)
2. Review Storybook stories for component usage
3. Verify backend API endpoint contracts match documented schemas
