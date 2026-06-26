# Metrics Console Frontend Integration

**Production-Ready React + Vite + TypeScript UX** for metric registry discovery, CRUD operations, PoP analysis, anomaly triage, and Temporal job monitoring.

## 📋 Overview

This frontend component suite delivers a complete metrics console integrated into your Semlayer Fabric Builder stack, consuming the Dual-Path Metric Calculation Engine API.

### Key Features

✅ **Metric Registry** — List, search, create, edit, delete metrics with full CRUD  
✅ **Detail Views** — Semantic metadata, comparison periods, SLA thresholds  
✅ **PoP Trends** — Period-over-period analysis with delta/percent-change visualizations  
✅ **Anomaly Triage** — Severity classification, confidence scoring, status tracking  
✅ **Job Run Monitoring** — Temporal workflow status, execution history, error details  
✅ **Multi-Tenant Ready** — Tenant-scoped requests via X-Tenant-ID header  
✅ **Dark Mode Support** — Tailwind CSS with light/dark theme  
✅ **Type-Safe** — Full TypeScript coverage with validation  

---

## 🗂️ File Structure

```
frontend/src/
├── api/
│   └── metrics-console.ts          # Axios client + tenant-aware endpoints
├── types/
│   └── metrics-console.ts          # TypeScript interfaces (MetricRegistry, PopRow, etc.)
├── hooks/
│   └── useMetricsConsole.ts        # TanStack Query hooks for CRUD & compute
├── components/
│   └── MetricForm.tsx              # Reusable create/edit form component
├── pages/
│   ├── MetricsConsolePage.tsx      # List page with filters & search
│   ├── MetricDetailPage.tsx        # Detail page (PoP, Anomalies, Runs tabs)
│   ├── MetricCreatePage.tsx        # Create new metric
│   └── MetricEditPage.tsx          # Edit existing metric
└── AppRoutes.tsx                    # Routes integrated into main app menu
```

---

## 📦 Installation & Setup

### 1. Dependencies (Already Included)

The frontend already has all required packages:

```bash
npm list @tanstack/react-query axios zod react-hook-form
```

**Core packages:**
- `@tanstack/react-query` — Data fetching, caching, synchronization
- `axios` — HTTP client with interceptors
- `react-router-dom` — Routing
- `tailwindcss` — Styling

### 2. Environment Variables

Ensure your `.env.local` has:

```env
VITE_API_URL=http://localhost:8080
```

This routes all metric API calls to your backend.

### 3. Tenant Setup

The console respects the tenant context from `TenantContext`:

```typescript
// In API client (metrics-console.ts)
setMetricsTenant(tenantId);  // Sets X-Tenant-ID header globally
```

When users select a tenant in Fabric Builder, the metrics console automatically inherits that scope.

---

## 🚀 Usage

### Accessing the Metrics Console

In the top navigation menu, click **📊 Metrics Console** or navigate to `/metrics`

### List Page (`/metrics`)

Browse all metrics for current tenant with:
- **Search** — Filter by metric name
- **Domain Filter** — Find metrics by domain (Marketing, Finance, etc.)
- **Golden Metrics Only** — Show SLA-critical metrics  
- **Create/Edit/Delete** — CRUD actions

**Sample Row:**
```
Name: Daily Active Users | Domain: Marketing | Granularity: 1d | Golden: ⭐ | SLA: 24hrs | Updated: 2025-11-01 10:00
```

### Detail Page (`/metrics/:metricId`)

View complete metric with three tabs:

#### **PoP Trend Tab**
- **Chart** — Visualize trend over selected date range
- **Table** — Period-label, current value, previous value, delta, percent change, record count, status
- **Date Range Picker** — Filter PoP results
- **Recompute Button** — Trigger real-time lane refresh

**Example Output:**
```
Period: 2025-10-31 | Current: 1,520 | Previous: 1,480 | Δ: +40 | %: +2.7% | Records: 1,520 | Status: Success
```

#### **Anomalies Tab**
- **Detected At** — ISO timestamp
- **Type** — z_score, threshold, trend, etc.
- **Severity** — low, medium, high, critical (color-coded)
- **Confidence** — 0-1 probability score
- **Actual/Expected** — Values for comparison
- **Status** — open, resolved, acknowledged
- **Analyze Button** — Trigger anomaly detection with configurable threshold & window

**Example Output:**
```
Detected: 2025-11-01 03:15 | Type: z_score | Severity: HIGH | Confidence: 95% | Actual: 1,250 | Expected: 1,500 | Status: Open
```

#### **Runs Tab**
- **Run ID** — Temporal workflow ID
- **Type** — pop, anomaly, comparison
- **Status** — pending, running, success, failed
- **Period Label** — Time window of computation
- **Started/Ended** — Execution timeline

**Example Output:**
```
Run: a1b2c3d4 | Type: pop | Status: Success | Period: 2025-10-31 | Started: 2025-11-01 12:00:05 UTC | Ended: 2025-11-01 12:05:10 UTC
```

### Create Metric (`/metrics/create`)

Form captures:

**Basic Information:**
- Name (required) — Identifier for metric
- Display Name — User-friendly label
- Domain (required) — Organizing category
- Category — Sub-category

**Technical Configuration:**
- Granularity — day, month, quarter, year
- Aggregation Function — SUM, COUNT, AVG, MAX, MIN, RATIO
- Base Query — SQL template with {{ date_start }}, {{ date_end }}
- Comparison Periods — Comma-separated list (e.g., "previous_period, year_over_year, quarter_over_quarter")

**SLAs & Ownership:**
- SLA Freshness (hours) — Max staleness threshold
- SLA Completeness (0-1) — Min data point ratio
- Owner User — Steward email
- Steward Group — RBAC group
- Golden Path — Mark as critical metric

**Validation:**
- Name & Domain required
- Completeness threshold must be 0-1

### Edit Metric (`/metrics/:metricId/edit`)

Same form as create, pre-populated with existing definition. Only non-required fields are editable (name is read-only to prevent orphaning).

---

## 🔗 API Integration

### Endpoint Mapping

| Page | Method | Endpoint | Purpose |
|------|--------|----------|---------|
| List | GET | `/api/metrics?q=&domain=&golden=` | Browse with filters |
| Detail | GET | `/api/metrics/:id` | Get definition |
| Create | POST | `/api/metrics` | Create new |
| Edit | PUT | `/api/metrics/:id` | Update definition |
| Delete | DELETE | `/api/metrics/:id` | Remove metric |
| PoP | GET | `/api/pop/metrics/:id?from=&to=` | Period-over-period data |
| Anomalies | GET | `/api/pop/anomalies/:id?from=&to=&status=` | Detected anomalies |
| Runs | GET | `/api/runs?metric_id=&calc_type=&status=` | Job run history |
| Compute PoP | POST | `/api/pop/metrics/:id/analyze-pop` | Trigger batch lane |
| Detect Anomaly | POST | `/api/pop/metrics/:id/analyze` | Trigger anomaly detection |
| Golden Path | POST | `/api/metrics/:id/promote-golden` | Governance action |

### Request Headers

All requests include:
```http
X-Tenant-ID: <current-tenant-id>
Authorization: Bearer <auth-token>
Content-Type: application/json
```

### Sample Requests

```bash
# List metrics with filters
curl -H "X-Tenant-ID: tenant-1" \
  "http://localhost:8080/api/metrics?domain=Marketing&golden=true"

# Get single metric
curl -H "X-Tenant-ID: tenant-1" \
  "http://localhost:8080/api/metrics/d123"

# Get PoP for date range
curl -H "X-Tenant-ID: tenant-1" \
  "http://localhost:8080/api/pop/metrics/d123?from=2025-10-01&to=2025-10-31"

# Trigger PoP computation
curl -X POST -H "X-Tenant-ID: tenant-1" \
  -d '{"period_label":"2025-10"}' \
  "http://localhost:8080/api/pop/metrics/d123/analyze-pop"

# Detect anomalies with threshold
curl -X POST -H "X-Tenant-ID: tenant-1" \
  -d '{"threshold":2.5,"window_days":90}' \
  "http://localhost:8080/api/pop/metrics/d123/analyze"
```

---

## 📊 Data Model Reference

### MetricRegistry

```typescript
{
  metric_id: UUID;
  name: string;                    // e.g., "user_signups"
  display_name?: string;           // e.g., "User Signups (Daily)"
  domain: string;                  // e.g., "Marketing"
  category?: string;               // e.g., "Acquisition"
  granularity: 'day'|'month'|..;   // Aggregation level
  aggregation_function: string;    // "SUM", "COUNT", etc.
  base_query?: string;             // SQL template
  comparison_periods?: string[];   // ["previous_period", "year_over_year"]
  sla_freshness_hours?: number;    // e.g., 24
  sla_completeness_threshold?: number; // 0.95
  golden_path: boolean;            // Critical metric flag
  owner_user_id?: string;          // alice@company.com
  steward_group?: string;          // "Marketing Analytics"
  created_at: string;              // ISO timestamp
  updated_at: string;              // ISO timestamp
}
```

### PopRow

```typescript
{
  metric_id: UUID;
  period_label: string;            // "2025-10-31" or "2025-10"
  period_start: string;            // YYYY-MM-DD
  period_end: string;              // YYYY-MM-DD
  current_value: string;           // Decimal as string
  previous_value?: string;         // For comparison
  delta?: string;                  // current - previous
  percent_change?: number;         // (delta / previous) * 100
  record_count: number;            // Underlying records
  computation_status: string;      // "success" | "failed" | "running"
  last_updated: string;            // ISO timestamp
}
```

### AnomalyRow

```typescript
{
  metric_id: UUID;
  anomaly_type: string;            // "z_score", "threshold"
  detected_at: string;             // ISO timestamp
  severity: string;                // "low" | "medium" | "high" | "critical"
  confidence?: number;             // 0-1 (e.g., 0.95)
  actual_value?: string;
  expected_value?: string;
  z_score?: number;                // For z_score anomalies
  status: string;                  // "open" | "resolved" | "acknowledged"
}
```

### JobRun

```typescript
{
  run_id: UUID;
  metric_id: UUID;
  calc_type: string;               // "pop" | "anomaly" | "comparison"
  status: string;                  // "pending" | "running" | "success" | "failed"
  period_label?: string;           // e.g., "2025-10"
  started_at: string;              // ISO timestamp
  ended_at?: string;               // ISO timestamp
}
```

---

## 🎨 UI/UX Highlights

### Dark Mode
- Automatic light/dark theme based on OS preference or user setting
- All components use `dark:` Tailwind classes
- Primary color: `#5048e5` (Indigo)

### Responsive Design
- Mobile-first grid layouts
- Collapsible menus on small screens
- Touch-friendly button sizes (h-10, h-11 = 40-44px)

### Accessibility
- Form labels with `htmlFor` binding
- ARIA labels on interactive elements
- Semantic HTML5 structure
- Color contrast ratios meet WCAG AA

### Status Indicators
- **Success** — Green badge with checkmark
- **Failed** — Red badge with X
- **Running** — Blue badge with spinner
- **Golden** — Yellow star icon ⭐

### Date/Time Formatting
- ISO 8601 for storage
- Locale-aware display (e.g., "2025-11-01 10:30 UTC")
- Relative time for recent events (e.g., "2 hours ago")

---

## 🧪 Testing

### Manual Test Scenarios

**Scenario 1: Create a Metric**
1. Navigate to `/metrics/create`
2. Fill form: name="test_metric", domain="Test", granularity="day"
3. Click "Save Metric"
4. Verify redirect to detail page

**Scenario 2: Edit Metric**
1. On detail page, click "Edit"
2. Update display_name
3. Click "Save Changes"
4. Verify update reflected on detail page

**Scenario 3: View PoP Results**
1. On detail page, select date range (e.g., 2025-10-01 to 2025-10-31)
2. Tab to "PoP Trend"
3. Verify table displays period_label, current_value, percent_change
4. Verify chart renders if data present

**Scenario 4: Trigger Anomaly Detection**
1. On detail page, click "Analyze Anomalies"
2. Wait for request (shows loading state)
3. Verify new run appears in "Runs" tab
4. Check "Anomalies" tab for detected issues

---

## 🔐 Security & Multi-Tenancy

### Tenant Isolation

Every request includes `X-Tenant-ID` header:

```typescript
// In api/metrics-console.ts
setMetricsTenant(tenantId);  // Called when tenant changes

// In metricsApi interceptor
metricsApi.defaults.headers.common['X-Tenant-ID'] = tenantId;
```

Backend validates tenant scope on every request. Frontend cannot override.

### Authentication

Uses existing Fabric Builder auth context. Protected routes enforce login via `<ProtectedRoute>`.

### Data Privacy

- Metric definitions are scoped to tenant
- PoP results visible only to tenant users
- Audit trail tracks all modifications (owner, timestamp)

---

## 🚀 Deployment

### Step 1: Verify Routes

Check `/frontend/src/AppRoutes.tsx` has metrics routes:

```tsx
<Route path="/metrics" element={<ProtectedRoute><MetricsConsolePage /></ProtectedRoute>} />
<Route path="/metrics/create" element={<ProtectedRoute><MetricCreatePage /></ProtectedRoute>} />
<Route path="/metrics/:metricId" element={<ProtectedRoute><MetricDetailPage /></ProtectedRoute>} />
<Route path="/metrics/:metricId/edit" element={<ProtectedRoute><MetricEditPage /></ProtectedRoute>} />
```

### Step 2: Build Frontend

```bash
cd frontend
npm run build
```

Output: `dist/` folder ready for deployment.

### Step 3: Start Dev Server

```bash
npm run dev
```

Console available at `http://localhost:5173/metrics` (or your Vite dev server URL).

### Step 4: Connect Backend

Ensure backend API running at `VITE_API_URL`:

```bash
# Backend
cd backend
go run cmd/server/main.go

# Frontend will call http://localhost:8080/api/metrics/*
```

---

## 📚 Integration Points

### With Dual-Path Engine

The console consumes all outputs from the backend:

- **Registry CRUD** → `semantic_layer.metric_registry` table
- **PoP Results** → `public.pop_computations` & `public.metrics_comparison_periods` tables
- **Anomalies** → `public.pop_anomalies` table  
- **Runs** → Job run tracking from Temporal workflows
- **SLA Violations** → `public.sla_violations` table

### With Fabric Builder

- Inherits tenant context from `TenantContext`
- Shares auth token from `ProtectedRoute`
- Uses same API base URL and headers
- Follows Fabric Builder styling conventions

### With Temporal

- Triggers workflows via POST endpoints
- Polls run status via GET `/api/runs`
- Displays Temporal IDs and execution timelines

---

## 📖 Documentation

For backend integration details, see:

- `/DELIVERY_INDEX_DUAL_PATH_ENGINE.md` — Architecture & API reference
- `/DUAL_PATH_ENGINE_GUIDE.md` — Complete system design
- `/DUAL_PATH_ENGINE_QUICK_START.md` — Deployment checklist

---

## ✨ Next Steps

1. **Deploy backend migrations** → Apply `000013_*.sql` and `000014_*.sql`
2. **Start backend** → Run orchestrator and job schedulers
3. **Register initial metrics** → Load from existing catalog
4. **Configure alerts** → Hook up Slack/PagerDuty  
5. **Train users** → Share runbook for stewards

---

## 🆘 Troubleshooting

### Issue: Metrics list shows "no data"

**Solution:**
1. Verify backend API running: `curl http://localhost:8080/api/metrics`
2. Check tenant selector — ensure tenant is selected
3. Inspect Network tab — verify X-Tenant-ID header

### Issue: "Unauthorized" error on API calls

**Solution:**
1. Check auth token in localStorage
2. Verify user has "Metrics Read" RBAC permission
3. Clear browser cache and re-login

### Issue: PoP results not showing

**Solution:**
1. Run compute trigger via "Recompute PoP" button
2. Wait for job to complete (check "Runs" tab)
3. Refresh page or wait for cache invalidation

### Issue: Anomalies never detected

**Solution:**
1. Ensure > 90 days of historical data
2. Lower z-score threshold (try 2.0 instead of 3.0)
3. Check anomaly detection scheduler running

---

**Status**: ✅ Production Ready  
**Version**: 1.0  
**Last Updated**: November 1, 2025
