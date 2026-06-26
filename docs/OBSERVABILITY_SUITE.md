# SemLayer End-to-End Observability Suite

## Overview

This document describes the complete, production-grade observability stack for SemLayer's global ingestion and query platform (Planner → Temporal → Commit Service → Trino).

**Status**: ✅ Phase 6.1 Complete — Full End-to-End Tracing, Metrics, Logs, and Dashboards

---

## 🏗️ Architecture Overview

### Trace Flow

```
┌─────────────────────────────────────────────────────────────┐
│ API Request (GET /plan?table=incidents&region=us-east-1)   │
└──────────────────────┬──────────────────────────────────────┘
                       │ plan_id = "8f2c3a4b-..."
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ Planner Service (OpenTelemetry)                             │
│ └─ span: planner_generate_plan                              │
│    ├─ attributes: plan_id, table, regions, strategy         │
│    └─ tags: region, tenant_id                               │
└──────────────────────┬──────────────────────────────────────┘
                       │ planID propagated in X-Plan-ID header
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ Temporal Workflow (Go + OTEL Instrumentation)               │
│ └─ span: workflow_drift_start                               │
│    ├─ attributes: plan_id, table, num_regions               │
│    └─ child spans: activity_regional_drift (per region)     │
└──────────────────────┬──────────────────────────────────────┘
                       │ plan_id propagated in activity context
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ Commit Service (Spring Boot + Micrometer + OTEL)            │
│ └─ span: commit_manifest                                    │
│    ├─ attributes: plan_id, manifest_id, table, snapshot_id  │
│    ├─ child spans: iceberg_commit, s3_validation            │
│    └─ metrics: commit_service_commit_latency_ms (histogram)  │
└──────────────────────┬──────────────────────────────────────┘
                       │ snapshot_id propagated in response
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ Trino Query Engine                                          │
│ └─ span: query_trino                                        │
│    ├─ attributes: plan_id, query_id, table, duration_ms     │
│    └─ metrics: trino_query_latency_ms (if instrumented)     │
└─────────────────────────────────────────────────────────────┘
```

All tied together by distributed trace context: **plan_id** → **manifest_id** → **snapshot_id** → **query_id**

---

## 📊 Observability Components

### 1. Logs — Structured JSON (Logstash Format)

**Provider**: Logback + Logstash Encoder

**Location**: All services emit JSON to stdout/stderr

**Example Output**:
```json
{
  "timestamp": "2026-02-09T14:32:10Z",
  "level": "INFO",
  "logger": "com.example.icebergcommit.IcebergCommitService",
  "message": "Iceberg commit successful",
  "plan_id": "8f2c3a4b-1234-5678-abcd-ef1234567890",
  "manifest_id": "m-1707480730000",
  "table": "ops.incidents",
  "snapshot_id": "1234567890123456",
  "commit_duration_ms": 145,
  "thread_name": "http-nio-8080-exec-1"
}
```

**Queryable by**: `plan_id`, `manifest_id`, `table`, `commit_duration_ms`, error codes

**Tools**: ELK Stack, Datadog, Google Cloud Logging, CloudWatch

---

### 2. Metrics — Prometheus + Micrometer

**Provider**: Micrometer + Prometheus Registry

**Endpoint**: `GET /actuator/prometheus` (Spring Boot)

**Metrics Exposed** (Commit Service):

| Metric Name | Type | Tags | Description |
|---|---|---|---|
| `commit_service_commits_success_total` | Counter | `table`, `region` | Total successful commits |
| `commit_service_commits_failed_total` | Counter | `table`, `region` | Total failed commits |
| `commit_service_commit_latency_ms` | Histogram | `table`, `region` | Commit operation latency (ms) |
| `commit_service_s3_validation_failures_total` | Counter | `bucket` | S3 object existence check failures |
| `commit_service_idempotency_hits_total` | Counter | `table` | Duplicate manifest detections |

**Scrape Configuration** (for Prometheus):
```yaml
scrape_configs:
  - job_name: 'iceberg-commit-service'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/actuator/prometheus'
    scrape_interval: 15s
```

**Tools**: Prometheus, Grafana, Datadog, New Relic

---

### 3. Traces — OpenTelemetry OTLP

**Provider**: OpenTelemetry SDK + OTLP Exporter (gRPC)

**Collector**: OpenTelemetry Collector Contrib (in Testcontainers for CI, in production)

**Collector Endpoint**: `localhost:4317` (default OTLP/gRPC)

**Collector Exporters**:
- Logging (stdout, debug)
- File (JSON, `/tmp/spans.json` for CI artifacts)
- Jaeger/Tempo (for trace visualization)

**Span Attributes Propagated**:

| Attribute | Set By | Description |
|---|---|---|
| `planner.plan_id` | Planner + Commit Service | Distributed trace context key |
| `manifest.id` | Commit Service | Manifest identifier |
| `iceberg.snapshot_id` | Commit Service | Iceberg snapshot ID after commit |
| `region` | All services | Geographic region |
| `table` | All services | Data table name |
| `tenant_id` | All services | Tenant identifier |

**Trace Visualization**: Jaeger UI (http://localhost:16686), Grafana Tempo

---

### 4. Dashboards — Grafana

**Dashboard File**: [commit_path_dashboard.json](./tools/iceberg-commit-service/grafana/commit_path_dashboard.json)

**Panels**:

**Row 1 — Global Health**
- Commit Success Rate (%)
- Commit Failure Rate (failures/sec)
- S3 Validation Failures (failures/sec)
- Idempotency Hits (hits/sec)

**Row 2 — Latency Breakdown**
- Commit Latency (p50/p95, ms)
- Commit Latency by Table (top 5)
- Commit Latency by Region (p95)

**Row 3 — Volume & Failures**
- Commits by Table (top 10, requests/sec)
- Commit Failures by Table (top 10)
- S3 Failures by Region (top 10)

**Import Instructions**:
1. Log into Grafana
2. Create Data Source → Prometheus (http://localhost:9090)
3. Dashboards → Import → Paste JSON from `commit_path_dashboard.json`
4. Select Prometheus as data source
5. Save and view

---

## 🧪 Integration Testing Infrastructure

### Testcontainers Stack

**File**: [IcebergTestEnvironment.java](./tools/iceberg-commit-service/src/test/java/com/example/icebergcommit/test/IcebergTestEnvironment.java)

**Containers**:
- PostgreSQL 15 (idempotency store)
- MinIO (S3-compatible object storage)
- Hive Metastore 3.1.2 (Iceberg catalog)
- Trino latest (query engine)
- OpenTelemetry Collector Contrib (OTLP receiver)

**Network**: Shared Docker network for inter-container communication

**Test Execution**:
```bash
mvn -f tools/iceberg-commit-service/pom.xml -P integration-tests verify
```

**Test Coverage**:
1. ✅ Happy path: Upload parquet, POST manifest, verify Iceberg snapshot, query via Trino
2. ✅ Error path: Missing S3 object returns 4xx
3. ✅ Idempotency: Same manifest twice produces one snapshot
4. ✅ Tracing: Spans exported to OTEL collector, queryable in `/tmp/spans.json`

**CI Artifacts on Failure**:
- Failsafe reports
- Application logs
- OTEL spans JSON
- Prometheus metrics

---

## 📔 Configuration Reference

### Application Configuration

**File**: `application.yml` (Spring Boot config for Commit Service)

```yaml
server:
  port: 8080

management:
  endpoints:
    web:
      exposure:
        include: prometheus,health,info
  endpoint:
    prometheus:
      enabled: true

spring:
  application:
    name: iceberg-commit-service

# OpenTelemetry exporter
otel:
  exporter:
    otlp:
      endpoint: http://localhost:4317  # OTEL Collector gRPC endpoint
  resource:
    attributes:
      service.name: iceberg-commit-service
      service.version: 1.0.0

# Structured logging (Logstash format)
logging:
  pattern:
    console: "%d{yyyy-MM-dd HH:mm:ss} %-5level %logger{36} - %msg%n"
```

### Environment Variables

```bash
# OTLP Exporter
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc

# Tracing sampler
export OTEL_TRACES_SAMPLER=always_on  # Sample 100%

# Database (for idempotency store)
export MANIFEST_DB_URL=jdbc:postgresql://localhost:5432/ops
export MANIFEST_DB_USER=ops
export MANIFEST_DB_PASSWORD=ops_pass

# S3 (or MinIO)
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export S3_ENDPOINT=http://localhost:9000

# Iceberg catalog
export ICEBERG_CATALOG_TYPE=hive
export ICEBERG_CATALOG_URI=thrift://localhost:9083
export ICEBERG_CATALOG_WAREHOUSE=s3a://iceberg-warehouse/
```

---

## 🚀 Deployment Checklist

### Pre-Production

- [ ] OTEL collector deployed and metrics/traces flowing
- [ ] Prometheus scraping commit service metrics
- [ ] Grafana dashboard imported and verified
- [ ] Structured logs shipped to ELK/Datadog/CloudWatch
- [ ] Sampling ratio set (start with 1.0, reduce to 0.1 once stable)
- [ ] Alerting rules created (see section below)
- [ ] Team trained on Grafana dashboard + trace UI

### Alert Rules (Prometheus)

#### Critical

```
# Commit failure rate > 5% over 5 minutes
alert: HighCommitFailureRate
expr: sum(rate(commit_service_commits_failed_total[5m])) / sum(rate(commit_service_commits_total[5m])) > 0.05
for: 5m
annotations:
  summary: "High commit failure rate ({{ $value | humanizePercentage }})"
```

#### Warning

```
# Commit latency p95 > 500ms
alert: HighCommitLatency
expr: histogram_quantile(0.95, sum(rate(commit_service_commit_latency_ms_bucket[5m])) by (le)) > 500
for: 10m
annotations:
  summary: "High commit latency p95 ({{ $value }}ms)"

# S3 validation failures > 1/sec
alert: HighS3ValidationFailures
expr: sum(rate(commit_service_s3_validation_failures_total[5m])) > 1
for: 5m
annotations:
  summary: "High S3 validation failures ({{ $value }}/sec)"
```

---

## 🔍 Debugging Guide

### Scenario 1: Slow Commit

1. Open Grafana dashboard
2. Check "Commit Latency (p50/p95)" panel
3. If p95 > 500ms:
   - Check "Commit Latency by Table" → which table?
   - Check "S3 Validation Failures" → object store issue?
   - Check logs: `grep "commit_duration_ms.*1000" /path/to/logs`
4. In Jaeger: Search for recent commits by `table=<value>`
5. Inspect spans: S3 validation? Iceberg snapshot creation? Network latency?

### Scenario 2: Commit Failures

1. Check Grafana "Commit Failure Rate" → when did it spike?
2. Check logs: `grep "error\|FAILED" | head -100`
3. In Jaeger: Search for `status=ERROR` in recent spans
4. Inspect error event on span → S3 bucket not found? Hive metastore down?
5. Check infrastructure: S3/MinIO availability, Hive Metastore health, Network connectivity

### Scenario 3: Idempotent Commits

1. In Grafana: "Idempotency Hits (5m)" → trending up?
2. In logs: `grep "idempotency.*hit" | wc -l` → count
3. In Jaeger: Search for `duplicate.*manifest` spans
4. Analyze root cause: Retry loop in planner? Duplicate API requests?

### Scenario 4: End-to-End Slow Path

1. Have the `plan_id` from API response headers or logs
2. Open Jaeger UI → search for `plan_id=<value>`
3. Inspect trace DAG:
   - Planner latency (should be < 100ms typically)
   - Temporal workflow latency (depends on region count)
   - Commit service latency per region
   - Trino query latency
4. Identify bottleneck (longest span in DAG)
5. Drill into that service's logs using `plan_id` filter

---

## 📚 Documentation Files

| File | Purpose |
|---|---|
| [TEMPORAL_TRACE_INJECTION.md](./docs/TEMPORAL_TRACE_INJECTION.md) | How to integrate OpenTelemetry with Temporal workflows |
| [otel_worker.go](./backend/internal/temporal/otel_worker.go) | Example Go code for Temporal worker with OTEL |
| [commit_path_dashboard.json](./tools/iceberg-commit-service/grafana/commit_path_dashboard.json) | Grafana dashboard for commit path observability |
| [CommitServiceIntegrationTest.java](./tools/iceberg-commit-service/src/test/java/com/example/icebergcommit/CommitServiceIntegrationTest.java) | Integration tests including OTEL span assertions |

---

## 🎯 What's Next?

### Option A — Trace Explorer React Component

Build a custom trace explorer in your admin UI:
- Search traces by `plan_id`, `table`, `region`
- Visualize trace DAG (Planner → Commit → Trino)
- Link to Jaeger for full details
- Show latency breakdown per span

### Option B — Temporal Metrics Integration

Add Temporal-specific metrics to Grafana:
- Workflow execution latency by name
- Activity execution latency by region
- Workflow failure rate by type
- Retry rate by activity

### Option C — Per-Tenant Observability Dashboard

Create a dashboard that breaks down all metrics by `tenant_id`:
- Commit success rate per tenant
- Idempotency hits per tenant
- S3 failures per tenant
- Can identify noisy tenants

### Option D — Multi-Region Heatmap

Build a 2D heatmap visualization:
- X-axis: Regions (us-east-1, eu-west-1, ap-southeast-1, etc.)
- Y-axis: Time
- Color intensity: Commit latency or failure rate
- Identify regional hotspots

---

## 📞 Support & Runbooks

### Runbook: "Commit Service Won't Start"

1. Check OTEL collector reachability: `curl http://localhost:4317`
2. Check Postgres: `psql -h localhost -U ops -d ops`
3. Check Hive Metastore: `telnet localhost:9083`
4. Check app logs: `docker logs <commit-service-container>`
5. Verify `application.yml` OTEL endpoint is correct

### Runbook: "Spans Not Appearing in Jaeger"

1. Verify OTEL collector is running: `docker ps | grep otel`
2. Check collector config has OTLP receiver: `cat /etc/otel/config.yaml`
3. Check collector logs: `docker logs <otel-collector-container>`
4. Verify commit service can reach collector: `curl http://collector:4317`
5. Check `otel.exporter.otlp.endpoint` in `application.yml`

### Runbook: "Alert: HighCommitFailureRate"

1. Open Grafana → Commit Path Dashboard
2. Check which tables are failing: "Commit Failures by Table"
3. Open logs: `grep "commits_failed_total" | tail -20`
4. Check S3 availability: `aws s3 ls s3://iceberg-warehouse/ --recursive | head`
5. Check Hive Metastore: `hive --debug -e "show tables;"`
6. Escalate to data platform team if infrastructure is healthy

---

## 📊 Metrics Reference

### Commit Service Metrics

```
# Counter: Total successful commits
commit_service_commits_success_total{table="ops.incidents",region="us-east-1"} 1234

# Histogram: Commit operation latency
commit_service_commit_latency_ms_bucket{table="ops.incidents",le="100"} 450
commit_service_commit_latency_ms_bucket{table="ops.incidents",le="500"} 980
commit_service_commit_latency_ms_bucket{table="ops.incidents",le="+Inf"} 1000

# Counter: S3 validation failures
commit_service_s3_validation_failures_total{bucket="iceberg-staging"} 5

# Counter: Idempotent commits (duplicates detected)
commit_service_idempotency_hits_total{table="ops.incidents"} 23
```

### Key PromQL Queries

```
# Success rate (%)
sum(rate(commit_service_commits_success_total[5m])) / 
(sum(rate(commit_service_commits_success_total[5m])) + sum(rate(commit_service_commits_failed_total[5m])))

# Latency p95
histogram_quantile(0.95, sum(rate(commit_service_commit_latency_ms_bucket[5m])) by (le))

# Per-table volume
{sum(rate(commit_service_commits_total[5m])) by (table)} | topk(10)

# Regional latency breakdown
histogram_quantile(0.95, sum(rate(commit_service_commit_latency_ms_bucket[5m])) by (le, region))
```

---

## ✅ Phase 6.1 Complete

| Component | Status | File |
|---|---|---|
| Structured Logging | ✅ | logback-spring.xml |
| Metrics (Prometheus) | ✅ | pom.xml (Micrometer), application.yml |
| Traces (OTEL/gRPC) | ✅ | IcebergCommitService.java, CommitController.java |
| Dashboard (Grafana) | ✅ | commit_path_dashboard.json |
| Integration Testing | ✅ | CommitServiceIntegrationTest.java + Testcontainers |
| OTEL Collector (CI) | ✅ | OtelCollectorContainer.java, otel-collector-config.yaml |
| Temporal Integration | ✅ | TEMPORAL_TRACE_INJECTION.md, otel_worker.go |
| Runbooks | ✅ | This document |

**Next Phase**: Add Temporal workflow instrumentation and per-tenant observability dashboard.

---

**Last Updated**: February 9, 2026  
**Authors**: Patrick + GitHub Copilot  
**Maintainers**: Data Platform Team
