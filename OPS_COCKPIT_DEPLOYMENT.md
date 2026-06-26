# Global Ops Cockpit - Deployment & Testing Guide

## ✅ Completion Status

All components have been successfully implemented, integrated, and tested for compilation.

### Components Delivered
- ✅ **Backend Go Modules** (8 modules): alerts, health, latency, errors, store interface, postgres store, handlers
- ✅ **SQL Migrations** (3 migrations): ops_alerts, ops_health_cache, ops_error_fingerprints
- ✅ **React Components** (4 components): HealthBadge, HeatmapChart, AlertList, ErrorFingerprints
- ✅ **React Hooks** (18 hooks): Full React Query integration for all ops endpoints
- ✅ **Metrics Integration**: Real queries against public.metrics table
- ✅ **UI Integration**: Mounted ops components in GlobalOpsDashboard
- ✅ **Navigation**: Added Ops Cockpit nav item to AdminLayout

---

## 🚀 Deployment Steps

### 1. Start the Backend Server

The server now includes Ops Cockpit initialization and will:
- Apply SQL migrations automatically on startup
- Schedule alert evaluator (runs every 5 minutes)
- Register all 18+ HTTP endpoints under `/admin/*`

```bash
cd /Users/eganpj/GitHub/semlayer/backend
go run ./cmd/server
```

**Expected Output:**
```
✅ Global Ops Cockpit schemas initialized
✅ Global Ops Cockpit initialized at /admin/alerts, /admin/health, /admin/errors
Alert evaluation complete
```

### 2. Create Sample Alerts

Run the sample alert creation script:

```bash
bash /Users/eganpj/GitHub/semlayer/create-sample-alerts.sh
```

This creates 4 sample alerts:
1. **High Error Rate** - triggers when error_rate > 0.05 (5%)
2. **High P95 Latency** - triggers when p95 latency > 500ms
3. **Low Availability** - triggers when availability < 95%
4. **High Rate Limit Hits** - triggers when rate_limited > 100

### 3. Test API Endpoints

Run the comprehensive test suite:

```bash
bash /Users/eganpj/GitHub/semlayer/test-ops-api.sh
```

**Tested Endpoints:**
- `GET /admin/alerts` - List all alerts
- `POST /admin/alerts/evaluate` - Trigger manual alert evaluation
- `GET /admin/tenants/{tenantID}/health` - Get tenant health score
- `GET /admin/endpoints/health` - Get all endpoint health
- `GET /admin/latency/heatmap?window=3600` - Get latency heatmap
- `GET /admin/errors/fingerprints?limit=20` - Get error fingerprints

---

## 📊 Frontend Integration

### GlobalOpsDashboard

The ops cockpit is now integrated into the main ops dashboard with:

1. **Real-Time Intelligence Section** (Top)
   - Active Alerts Table: Shows enabled/disabled status, evaluates on demand
   - System Health Badge: Displays 0-100 score with color-coding (green/yellow/red)
   - Latency Heatmap: Matrix grid visualization of p95 latency by region/tenant/endpoint
   - Error Fingerprints: Grouped errors with drill-down capability

2. **Historical Analytics Section** (Bottom)
   - Usage trends, error trends, latency trends
   - Top tenants and endpoints
   - Recent errors timeline

### Navigation

Added "Ops Cockpit" (⚡) to admin sidebar navigation:
- Maps to `/admin/ops-cockpit` route
- Quick access to real-time intelligence

### Component Details

#### HealthBadge
- Displays health score (0-100) with percentage
- Color-coded: green ≥80, yellow 50-79, red <50
- Includes component breakdown (availability, latency, error_rate, rate_limits)
- Pulse animation for degraded/critical states

#### HeatmapChart
- Matrix grid: dimensions (region/tenant/endpoint) × time buckets
- Color gradient: green (fast) → yellow → red (slow)
- Based on p95 latency values
- Responsive and scrollable for mobile

#### AlertList
- Table view: name, metric, scope, threshold, window, status
- Toggle controls for enabling/disabling alerts
- "Evaluate Now" button for manual trigger
- Status badges (enabled/disabled)

#### ErrorFingerprints
- Main table: path, status code, sample message, count, last seen
- Click to expand and see recent occurrences
- Detail view: tenant, endpoint, message, timestamp
- Status code color-coding (4xx yellow, 5xx red)

---

## 🔧 Metrics Integration

The metrics methods now query from `public.metrics` table:

### GetMetricValue(metric, scope, since)
- Averages metric values for a given scope
- Supports any metric_type ("availability", "latency", "error_rate", etc.)
- Returns 0 if no data found

### GetTenantMetrics(tenantID, since)
- Aggregates: availability, p50/p95/p99 latency, request count, rate limit hits, error rate
- Returns default healthy metrics if no data

### GetEndpointMetrics(endpoint, since)
- Aggregates: p50/p95/p99 latency, request count, error rate
- Filters by endpoint tag
- Returns default healthy metrics if no data

### GetGlobalMetrics(since)
- Global averages across all tenants and endpoints
- Same metrics as tenant metrics
- Used for system-wide health assessment

---

## 🎯 Alert Evaluation

### Automatic Evaluation
- **Schedule**: Every 5 minutes (background goroutine)
- **Process**: For each enabled alert:
  1. Fetch metric using GetMetricValue()
  2. Compare current value against threshold
  3. Record AlertEvent if triggered
  4. Log success/failure

### Manual Evaluation
- **Endpoint**: `POST /admin/alerts/evaluate`
- **Response**: Number of alerts evaluated
- **Use**: Immediate testing during development

---

## 📈 Health Score Calculation

Health scores are weighted composites:

```
TenantHealth = (0.40 × AvailabilityScore) +
               (0.30 × LatencyScore) +
               (0.20 × ErrorRateScore) +
               (0.10 × RateLimitScore)
```

Where each component is normalized to 0-100:
- **Availability**: target 99% (99% = 100)
- **Latency**: target <200ms p95
- **Error Rate**: target <1%
- **Rate Limits**: target <100 hits/hour

---

## 🗄️ Database Schema

### ops_alerts
```sql
id (UUID), name, scope, metric, threshold, comparison, 
window_secs, enabled, created_at, updated_at
```

### ops_alert_events
```sql
id, alert_id, tenant_id (nullable), scope_id, endpoint (nullable),
value, triggered_at
```

### ops_tenant_health_cache
```sql
tenant_id, health_score, components (JSON), computed_at, updated_at
```

### ops_endpoint_health_cache
```sql
endpoint, health_score, error_rate, p95_ms, requests_1h, 
computed_at, updated_at
```

### ops_latency_heatmap
```sql
id, dimension (region/tenant_id/endpoint), time_bucket,
p50_ms, p95_ms, p99_ms, created_at
```

### ops_error_fingerprints
```sql
id, fingerprint (SHA256), path, status_code, sample_message,
first_seen, last_seen, count, created_at
```

### ops_error_events
```sql
id, fingerprint_id, tenant_id (nullable), endpoint, status_code,
message, request_id (nullable), occurred_at
```

---

## ✨ Production Checklist

- [x] Backend compiles without errors
- [x] SQL migrations are idempotent
- [x] Alert evaluator scheduled
- [x] Metrics methods implemented
- [x] React components wired to hooks
- [x] AdminLayout navigation updated
- [x] GlobalOpsDashboard integrated
- [ ] Database migrations applied to production
- [ ] Frontend built and deployed
- [ ] API authentication configured
- [ ] Monitoring/logging enabled
- [ ] Performance tested under load

---

## 🔍 Testing Checklist

Run these in order:

```bash
# 1. Start server (requires DATABASE_URL set)
cd /Users/eganpj/GitHub/semlayer/backend
go run ./cmd/server

# 2. In new terminal: Create sample alerts
bash /Users/eganpj/GitHub/semlayer/create-sample-alerts.sh

# 3. Test API endpoints
bash /Users/eganpj/GitHub/semlayer/test-ops-api.sh

# 4. Verify alert evaluation ran (check server logs)
# Should see: "Alert evaluation complete" every 5 minutes

# 5. Build frontend
cd /Users/eganpj/GitHub/semlayer/frontend
npm run build

# 6. Visit http://localhost:3000/admin/ops-cockpit
# Should see ops dashboard with real-time components
```

---

## 📋 API Reference

### Alerts

**Create Alert**
```bash
POST /admin/alerts
Content-Type: application/json

{
  "name": "High Error Rate",
  "scope": "global|tenant|endpoint",
  "metric": "error_rate|latency_p95|availability|rate_limited",
  "threshold": 0.05,
  "comparison": ">|<|>=|<=|==",
  "window_secs": 300,
  "enabled": true
}
```

**List Alerts**
```bash
GET /admin/alerts?enabled=true&limit=50
```

**Evaluate Alerts**
```bash
POST /admin/alerts/evaluate
```

### Health

**Get Tenant Health**
```bash
GET /admin/tenants/{tenantID}/health
```

**List Endpoint Health**
```bash
GET /admin/endpoints/health?limit=50
```

### Metrics

**Get Latency Heatmap**
```bash
GET /admin/latency/heatmap?window=3600&group_by=region
```

**Get Error Fingerprints**
```bash
GET /admin/errors/fingerprints?limit=20
```

---

## 🐛 Troubleshooting

### Migrations Not Applied
- Check `DATABASE_URL` environment variable is set
- Verify PostgreSQL is running
- Check logs for "Global Ops Cockpit schemas initialized"

### Alert Evaluator Not Running
- Check server logs for "Alert evaluation complete" every 5 minutes
- Verify metrics table has data
- Check alert comparison operators (>, <, >=, <=, ==)

### Components Not Rendering
- Verify React hooks have correct endpoint URL
- Check browser DevTools network tab for failed requests
- Ensure authentication headers are being sent

### No Metrics Data
- Insert sample metrics into public.metrics table
- Check metric_type and tags match query expectations
- Verify time window includes current timestamp

---

## 📖 Additional Documentation

See comprehensive implementation guide: `OPS_COCKPIT_COMPLETE.md`

Key sections:
- Architecture Decision Log
- Data Flow Examples
- API Integration Patterns
- Performance Optimization
- Success Metrics

