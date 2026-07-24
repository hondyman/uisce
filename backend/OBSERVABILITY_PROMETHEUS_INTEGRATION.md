# Observability Prometheus Integration - Production Guide

**Status**: ✅ Production Ready (Priority A Complete)  
**Date**: February 9, 2026  
**Version**: 1.0

## Overview

This document describes the production-ready Prometheus integration for the SemLayer observability platform. All handlers now query real Prometheus metrics instead of returning hardcoded/mock data.

## Architecture

### Prometheus Connectivity

**Default Configuration:**
- URL: `http://prometheus:9090`
- Configurable via: `PROMETHEUS_URL` environment variable
- Query endpoint: `/api/v1/query`
- Default timeout: 5 seconds per query

```bash
# Set custom Prometheus URL (optional)
export PROMETHEUS_URL="http://prometheus.example.com:9090"
```

### Implementation Pattern

All Prometheus queries follow this pattern:

1. **Build PromQL query** with relevant labels and time windows
2. **Execute HTTP request** to Prometheus API with context timeout
3. **Parse JSON response** into typed result struct
4. **Extract values** using helper functions
5. **Return in API response** with proper timestamps

## API Endpoints

### 1. Global Metrics (`GET /api/metrics/global`)

Returns system-wide health KPIs.

**Response:**
```json
{
  "commitSuccessRate": 99.82,
  "s3Failures5m": 2,
  "idempotencyHits5m": 412,
  "regionsDegraded": 0,
  "avgCommitLatencyMs": 245.5,
  "p95CommitLatencyMs": 520.0,
  "activeRegions": 3,
  "timestamp": "2026-02-09T15:30:45Z"
}
```

**Prometheus Queries:**
- `(increase(commits_total{status="success"}[5m]) / increase(commits_total[5m])) * 100` → commitSuccessRate
- `increase(s3_failures_total[5m])` → s3Failures5m
- `increase(idempotency_hits_total[5m])` → idempotencyHits5m
- `count(up{job="region_health"} == 0)` → regionsDegraded
- `avg(rate(commit_latency_milliseconds_sum[5m]) / rate(commit_latency_milliseconds_count[5m]))` → avgCommitLatencyMs
- `histogram_quantile(0.95, rate(commit_latency_milliseconds_bucket[5m]))` → p95CommitLatencyMs
- `count(up{job="region_health"} == 1)` → activeRegions

**Cache-Control:** `max-age=30` (30 seconds)

### 2. Region Heatmap (`GET /api/observability/heatmap`)

Returns latency distribution by region.

**Response:**
```json
[
  {
    "region": "us-east",
    "bucket": "current",
    "value": 245.5
  },
  {
    "region": "eu-west",
    "bucket": "current",
    "value": 450.0
  }
]
```

**Prometheus Query:**
- `increase(commit_latency_milliseconds_sum{job="commits"}[5m]) / increase(commit_latency_milliseconds_count{job="commits"}[5m])` → latency by region

**Cache-Control:** `max-age=60`

### 3. Tenant Metrics (`GET /api/metrics/tenant/{tenantId}`)

Returns metrics for a specific tenant.

**Response:**
```json
{
  "tenantId": "tenant-123",
  "successRate": 99.8,
  "s3Failures": 3,
  "idempotencyHits": 127,
  "avgLatencyMs": 234.5,
  "timestamp": "2026-02-09T15:30:45Z"
}
```

**Prometheus Queries:**
- `(increase(commits_total{tenant_id="...",status="success"}[5m]) / increase(commits_total{tenant_id="..."}[5m])) * 100` → successRate
- `increase(s3_failures_total{tenant_id="..."}[5m])` → s3Failures
- `increase(idempotency_hits_total{tenant_id="..."}[5m])` → idempotencyHits
- `avg(rate(commit_latency_milliseconds_sum{tenant_id="..."}[5m]) / rate(commit_latency_milliseconds_count{tenant_id="..."}[5m]))` → avgLatencyMs

**Cache-Control:** `max-age=30`

### 4. Commit Metrics (`GET /api/metrics/commit?plan_id=...`)

Returns detailed metrics for a specific commit plan.

**Response:**
```json
{
  "planId": "plan-abc123",
  "commitLatencyMs": 245.5,
  "s3Failures": 0,
  "idempotencyHits": 5,
  "commitSuccessRate": 100.0,
  "table": "customer_ltv",
  "region": "us-east",
  "timestamp": "2026-02-09T15:30:45Z"
}
```

**Prometheus Queries:**
- `histogram_quantile(0.95, rate(commit_latency_milliseconds_bucket{plan_id="..."}[5m]))` → commitLatencyMs
- `increase(s3_failures_total{plan_id="..."}[5m])` → s3Failures
- `increase(idempotency_hits_total{plan_id="..."}[5m])` → idempotencyHits
- `(increase(commits_total{plan_id="...",status="success"}[5m]) / increase(commits_total{plan_id="..."}[5m])) * 100` → commitSuccessRate
- `max_over_time(commit_metadata{plan_id="..."}[1h])` → table, region from labels

**Returns `200 OK` or `500 Internal Server Error` (never defaults)**

### 5. Recent Plans (`GET /api/plans?tenant=... [&limit=100]`)

Returns recent plans for a tenant from Prometheus.

**Response:**
```json
[
  {
    "id": "plan-abc123",
    "table": "customer_ltv",
    "region": "us-east",
    "status": "success",
    "latency": 245.5,
    "timestamp": "2026-02-09T15:30:45Z"
  }
]
```

**Prometheus Query:**
- `topk(limit, group by (plan_id, table, region, status) (max_over_time(commit_status{tenant_id="..."}[1h])))`

**Cache-Control:** `max-age=60`

### 6. Plan Timeline (`GET /api/plans/timeline?limit=50`)

Returns chronological plan events from Prometheus.

**Response:**
```json
[
  {
    "planId": "plan-abc123",
    "table": "customer_ltv",
    "region": "us-east",
    "status": "success",
    "latency": 245.5,
    "timestamp": "2026-02-09T15:30:45Z"
  }
]
```

**Prometheus Query:**
- `topk(limit, max_over_time(commit_status[1h]))`

**Cache-Control:** `max-age=60`

### 7. Iceberg Lineage (`GET /api/iceberg/lineage?table=...`)

Returns Iceberg snapshot lineage for a table.

**Response:**
```json
[
  {
    "snapshotId": 1,
    "timestamp": "2026-02-09T15:25:00Z",
    "fileCount": 150,
    "dataBytes": 2500000
  },
  {
    "snapshotId": 2,
    "parentSnapshotId": 1,
    "timestamp": "2026-02-09T15:30:00Z",
    "fileCount": 152,
    "dataBytes": 2600000
  }
]
```

**Prometheus Query:**
- `max_over_time(iceberg_snapshot_metadata{table="..."}[1h])`

**Cache-Control:** `max-age=300`

## Helper Functions

### queryPrometheus()

Executes a PromQL query and returns the result.

```go
func queryPrometheus(ctx context.Context, query string) (*PrometheusQueryResult, error)
```

- **Timeout**: 5 seconds (configurable in implementation)
- **Error handling**: Returns error with context if query fails
- **URL handling**: Uses `PROMETHEUS_URL` env var (default: `http://prometheus:9090`)

### getFloatValue()

Extracts a single float value from Prometheus result.

```go
func getFloatValue(result *PrometheusQueryResult) float64
```

- Returns `0` if result is empty or malformed
- Handles both string and float64 value types
- Safe against nil pointers

### getMetricLabel()

Extracts a label value from query result.

```go
func getMetricLabel(result *PrometheusQueryResult, label string) string
```

- Returns empty string if label not found
- Used for metadata like table name, region, status

### sanitizePromQL()

Escapes special characters in PromQL string literals.

```go
func sanitizePromQL(s string) string
```

- Escapes double quotes to prevent PromQL injection
- Called on all user-provided inputs (plan_id, tenant_id, table)

## Error Handling

### No Hardcoded Defaults

All endpoints properly handle Prometheus query failures:

1. **Query Timeout**: Returns `500` with error message
2. **Invalid Response**: Returns `500` with parse error
3. **Empty Results**: Returns `0` for numeric fields, empty string for text
4. **Network Error**: Returns `500` with connection error

Example error response:

```json
{
  "error": "failed to query commit latency: context deadline exceeded"
}
```

### Required Parameters

All endpoints validate required query parameters:

- `plan_id` (commit metrics) - Returns `400` if missing
- `tenantId` (tenant metrics) - Returns `400` if missing  
- `tenant` (plans) - Returns `400` if missing
- `table` (Iceberg lineage) - Returns `400` if missing

## Metrics Requirements

### Required Prometheus Metrics

For the observability platform to function, these metrics must be exposed by your application:

1. **Commit Metrics**
   - `commits_total` (counter) - Labels: plan_id, tenant_id, status, region, table
   - `commit_latency_milliseconds` (histogram) - Labels: plan_id, tenant_id, region
   - `commit_metadata` (gauge) - Labels: plan_id, table, region

2. **S3 Metrics**
   - `s3_failures_total` (counter) - Labels: plan_id, tenant_id, region

3. **Idempotency Metrics**
   - `idempotency_hits_total` (counter) - Labels: plan_id, tenant_id, region

4. **Region Health**
   - `up` (gauge with job label) - Job: region_health, Labels: region

5. **Iceberg Metrics**
   - `iceberg_snapshot_metadata` (gauge) - Labels: table, snapshot_id, file_count, data_bytes

### Example Prometheus Recording Rules

```yaml
groups:
  - name: observability
    interval: 30s
    rules:
      - record: commit:success_rate_5m
        expr: (increase(commits_total{status="success"}[5m]) / increase(commits_total[5m])) * 100
      
      - record: commit:avg_latency_5m
        expr: avg(rate(commit_latency_milliseconds_sum[5m]) / rate(commit_latency_milliseconds_count[5m]))
      
      - record: commit:p95_latency_5m
        expr: histogram_quantile(0.95, rate(commit_latency_milliseconds_bucket[5m]))
```

## Performance Considerations

1. **Query Caching**
   - Global metrics: 30 seconds (dynamic, all tenants)
   - Tenant metrics: 30 seconds (per-tenant, plan_id specific)
   - Heatmap: 60 seconds (aggregated by region)
   - Timeline: 60 seconds (top-k aggregation)
   - Snapshots: 300 seconds (Iceberg metadata rarely changes)

2. **Query Optimization**
   - All queries limited to recent time windows (1h-1d)
   - Use `topk()` for pagination (no database offset)
   - Histogram quantiles computed at query time (not stored)
   - Aggregations performed in Prometheus (not application)

3. **Timeout Settings**
   - Individual query: 5 seconds
   - Endpoint total: 10 seconds (context timeout)
   - HTTP client: 5 seconds per request

## Monitoring & Debugging

### Check Prometheus Connectivity

```bash
curl http://prometheus:9090/api/v1/query?query=up
```

### Test Specific Metric

```bash
# Check if metrics are being scraped
curl "http://prometheus:9090/api/v1/query?query=commits_total"

# Check latest values
curl "http://prometheus:9090/api/v1/instant?query=rate(commits_total[5m])"
```

### Enable Verbose Logging

The implementation uses standard `http` client logging. Enable with:

```bash
export GO_DEBUG_HTTP=1
```

### Common Issues

| Issue | Solution |
|-------|----------|
| `connection refused` | Verify `PROMETHEUS_URL` is correct and Prometheus is running |
| `200 OK` but empty results | Check metrics are being scraped; verify label names match |
| `500` errors on all endpoints | Verify Prometheus API (/api/v1/query) responds to test queries |
| High latency | Check Prometheus server load; increase query timeout if needed |

## Configuration

### Environment Variables

```bash
# Prometheus server URL (default: http://prometheus:9090)
export PROMETHEUS_URL="http://prometheus:9090"

# Optional: Enable detailed logging
export API_DEBUG=1
```

### Docker Compose

```yaml
services:
  backend:
    environment:
      PROMETHEUS_URL: "http://prometheus:9090"
      TRACE_QUERY_URL: "http://tempo:3100"
```

## Migration from Mock Data

If migrating from hardcoded mock data:

1. **Verify metrics exist** in Prometheus
2. **Check label names** match queries (case-sensitive)
3. **Test queries manually** before deploying
4. **Monitor error rates** in first 24 hours
5. **Adjust time windows** if data is sparse

## Testing

### Unit Test Pattern

```go
// Mock Prometheus responses
mockResult := &PrometheusQueryResult{
    Status: "success",
    Data: struct {
        ResultType string
        Result []struct {
            Metric map[string]string
            Value [2]interface{}
        }
    }{
        Result: []struct {
            Metric map[string]string
            Value [2]interface{}
        }{{
            Value: [2]interface{}{
                int64(time.Now().Unix()),
                "234.5",
            },
        }},
    },
}

// Test handler
resp := httptest.NewRecorder()
req, _ := http.NewRequest("GET", "/api/metrics/global", nil)
s.globalMetricsHandler(resp, req)

// Verify response
assert.Equal(t, http.StatusOK, resp.Code)
```

### Integration Test Pattern

```bash
# Start Prometheus
docker-compose up -d prometheus

# Run backend
go run ./cmd/server

# Test endpoint
curl http://localhost:8080/api/metrics/global

# Should return real metrics, not zeros
```

## Future Enhancements

1. **Dashboard caching layer** - Cache aggregations locally for 60s
2. **Metric cardinality limits** - Alert on high cardinality labels
3. **Query optimization** - Replace instant queries with range queries where appropriate
4. **Exemplar support** - Link metrics to traces via Prometheus exemplars
5. **Custom recording rules** - Pre-compute complex aggregations

## Support

For issues with Prometheus integration:

1. Check `PROMETHEUS_URL` is accessible
2. Verify metrics are being collected (check Prometheus UI)
3. Review Prometheus query performance (check Prometheus /graph')
4. Check application logs for query errors
5. Monitor endpoint response times for timeout issues

---

**Last Updated**: February 9, 2026  
**Implemented By**: Priority A - Prometheus Wiring  
**Status**: Production Ready ✅
