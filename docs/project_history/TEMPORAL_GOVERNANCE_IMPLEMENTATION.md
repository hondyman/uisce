# Temporal Workflow Governance Implementation Guide

## Overview

This guide implements Workday-grade workflow governance, monitoring, and reporting using Temporal, integrated into your Fabric Builder platform. You now have:

1. **Search Attributes** - queryable business context for filtering workflows
2. **Admin Controls** - signal, update, cancel, terminate, reset operations
3. **History Export** - audit trails and analytics-ready data
4. **Frontend Dashboard** - Workday-like operations control panel
5. **Prometheus/Grafana Monitoring** - real-time metrics and alerting

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Frontend (React)                               │
│        TemporalAdminDashboard with Saved Views & Filters            │
└────────────────────────┬────────────────────────────────────────────┘
                         │ HTTP REST API
┌────────────────────────▼────────────────────────────────────────────┐
│                   Backend (Go)                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  /api/temporal/workflows/{id}                               │   │
│  │    - /signal    (send workflow signal)                      │   │
│  │    - /update    (update workflow)                           │   │
│  │    - /cancel    (graceful cancellation)                     │   │
│  │    - /terminate (immediate termination)                     │   │
│  │    - /reset     (replay from decision point)                │   │
│  │  /api/temporal/search-attributes (definitions)              │   │
│  │  /api/temporal/setup-cli-script (CLI instructions)          │   │
│  └──────────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Services:                                                   │   │
│  │    - SearchAttributeInitializer (registers attributes)      │   │
│  │    - WorkflowAdminService (control operations)              │   │
│  │    - HistoryExportService (audit & analytics)               │   │
│  └──────────────────────────────────────────────────────────────┘   │
└──────────────────────┬───────────────────────────────────────────────┘
                       │ gRPC
┌──────────────────────▼───────────────────────────────────────────────┐
│                  Temporal Server (Docker)                            │
│            (7233: gRPC, 8233: Metrics)                              │
├──────────────────────────────────────────────────────────────────────┤
│   - Workflow Executions (with Search Attributes)                    │
│   - Event History                                                    │
│   - Worker Health                                                    │
│   - Metrics Endpoint (Prometheus format)                            │
└───────────────┬────────────────────────────────┬───────────────────┘
                │ Metrics                        │ Temporal CLI
       ┌────────▼────────┐         ┌─────────────▼──────────┐
       │  Prometheus     │         │  Grafana Dashboards    │
       │  (9090)         │         │  (3000)                │
       └─────────────────┘         └────────────────────────┘
```

## Implementation Steps

### Step 1: Initialize Search Attributes

Search Attributes enable filtering and querying workflows by business context.

#### Backend Setup (Go)

```go
// backend/internal/temporal/search_attributes.go
// Already implemented with StandardSearchAttributes()

// In your server startup (cmd/server/main.go):
import "github.com/eganpj/semlayer/backend/internal/temporal"

func init() {
    // Call this at startup to log what attributes to register
    searchAttrInit := temporal.NewSearchAttributeInitializer(temporalClient, "default")
    searchAttrInit.InitializeSearchAttributes(context.Background())
}
```

#### Register Attributes via Temporal CLI

Download the setup script from `/api/temporal/setup-cli-script` or run directly:

```bash
#!/bin/bash
# Setup Temporal Search Attributes

temporal operator search-attribute create \
  --name BusinessUnit \
  --type Keyword \
  --yes

temporal operator search-attribute create \
  --name SlaDeadline \
  --type Datetime \
  --yes

temporal operator search-attribute create \
  --name Priority \
  --type Int \
  --yes

temporal operator search-attribute create \
  --name ProcessOwner \
  --type Keyword \
  --yes

temporal operator search-attribute create \
  --name CustomerID \
  --type Keyword \
  --yes

temporal operator search-attribute create \
  --name ProcessStatus \
  --type Keyword \
  --yes

temporal operator search-attribute create \
  --name ComplianceRisk \
  --type Keyword \
  --yes

temporal operator search-attribute create \
  --name EscalationLevel \
  --type Int \
  --yes

temporal operator search-attribute create \
  --name StartTime \
  --type Datetime \
  --yes

temporal operator search-attribute create \
  --name TenantID \
  --type Keyword \
  --yes
```

### Step 2: Enable Admin Control API Endpoints

#### Register Routes (backend/internal/api/api.go)

Add this to your route registration in the `/api` block:

```go
import (
    "go.temporal.io/sdk/client"
    httpapi "github.com/eganpj/semlayer/backend/internal/api"
)

// In your Server.RegisterRoutes or similar:
func (s *Server) RegisterRoutes(r chi.Router, temporalClient client.Client) {
    r.Route("/api", func(r chi.Router) {
        // ... existing routes ...

        // Temporal workflow admin endpoints
        httpapi.RegisterTemporalAdminRoutes(r, temporalClient)
    })
}
```

#### Test Admin Endpoints

```bash
# Signal a workflow
curl -X POST http://localhost:8080/api/temporal/workflows/order-123/signal \
  -H "Content-Type: application/json" \
  -d '{
    "signal_name": "unblock",
    "input": {"reason": "manual override"},
    "reason": "escalation required"
  }'

# Cancel a workflow
curl -X POST http://localhost:8080/api/temporal/workflows/order-123/cancel \
  -H "Content-Type: application/json" \
  -d '{"reason": "customer requested cancellation"}'

# Terminate a workflow
curl -X POST http://localhost:8080/api/temporal/workflows/order-123/terminate \
  -H "Content-Type: application/json" \
  -d '{"reason": "stuck workflow", "details": "no progress in 24 hours"}'

# List Search Attributes
curl http://localhost:8080/api/temporal/search-attributes
```

### Step 3: Integrate Frontend Dashboard

#### Add Route (frontend/src/AppRoutes.tsx or similar)

```tsx
import TemporalAdminDashboard from './pages/TemporalAdminDashboard';

export const routes = [
  // ... existing routes ...
  {
    path: '/temporal-admin',
    element: <TemporalAdminDashboard />,
    label: 'Temporal Admin',
    icon: 'Activity',
  },
];
```

#### Dashboard Features

- **Saved Views**: Pre-configured queries for common operational patterns
  - "Failed Last 24h": `status = 'failed' AND start_time > '-24h'`
  - "Pending > 2h": `status = 'pending' AND elapsed_time > 7200`
  - "High Priority": `Priority > 2`

- **Search Attributes**: Quick reference for available filter dimensions

- **Workflow List**: Real-time view of all executions with inline controls

- **Admin Actions**: Signal, Update, Cancel, Terminate, Reset workflows

- **Action History**: Audit trail of all admin operations

- **Workflow Details**: Deep-dive into individual workflow state

### Step 4: Export Histories for Analytics

Use the `HistoryExportService` to pull data for BI and compliance.

#### REST API

```bash
# Export single workflow history
curl http://localhost:8080/api/temporal/workflows/order-123/history

# Batch export (for reporting)
curl -X POST http://localhost:8080/api/temporal/workflows/batch-export \
  -H "Content-Type: application/json" \
  -d '{
    "query": "status = '\''completed'\'' AND start_time > '\''-7d'\''",
    "format": "jsonl"
  }'
```

#### Go Usage

```go
import "github.com/eganpj/semlayer/backend/internal/temporal"

// In your handler or service:
historyService := temporal.NewHistoryExportService(temporalClient, "default")

// Export for analytics
records, err := historyService.ExportHistoryForAnalytics(ctx, temporal.HistoryExportRequest{
    WorkflowID: "order-123",
})

// Export for compliance audit
auditTrail, err := historyService.ExportAuditTrail(ctx, "order-123", "run-001")
```

### Step 5: Setup Prometheus & Grafana Monitoring

#### Update docker-compose.yml

Add Prometheus and Grafana services to your `docker-compose.yml`:

```yaml
services:
  # ... existing services ...

  prometheus:
    image: prom/prometheus:latest
    container_name: semlayer-prometheus
    restart: always
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=30d'
    depends_on:
      - temporal

  grafana:
    image: grafana/grafana:latest
    container_name: semlayer-grafana
    restart: always
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning
    depends_on:
      - prometheus

volumes:
  prometheus_data:
  grafana_data:
```

#### Create prometheus/prometheus.yml

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    monitor: 'temporal-monitor'

scrape_configs:
  - job_name: 'temporal-server'
    static_configs:
      - targets: ['temporal:8233']
    metric_path: '/metrics'

  - job_name: 'backend-service'
    static_configs:
      - targets: ['backend:8080']
    metrics_path: '/metrics'

  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
```

#### Create Grafana Dashboards

##### File: `grafana/provisioning/dashboards/temporal-workflows.json`

```json
{
  "dashboard": {
    "title": "Temporal Workflows - Real-time Monitor",
    "panels": [
      {
        "title": "Workflow Executions (Last 24h)",
        "targets": [
          {
            "expr": "increase(temporal_workflow_completed_total[24h])"
          }
        ]
      },
      {
        "title": "Failed Workflows",
        "targets": [
          {
            "expr": "increase(temporal_workflow_failed_total[24h])"
          }
        ]
      },
      {
        "title": "Workflow Duration (Percentiles)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, temporal_workflow_execution_latency_bucket)"
          }
        ]
      },
      {
        "title": "Long-Running Workflows (>2h)",
        "targets": [
          {
            "expr": "temporal_workflow_running{elapsed_seconds > '7200'}"
          }
        ]
      },
      {
        "title": "Workflow Status Distribution",
        "targets": [
          {
            "expr": "temporal_workflow_running_total"
          },
          {
            "expr": "temporal_workflow_completed_total"
          },
          {
            "expr": "temporal_workflow_failed_total"
          }
        ]
      },
      {
        "title": "Activity Success Rate",
        "targets": [
          {
            "expr": "temporal_activity_executed_total / temporal_activity_completed_total * 100"
          }
        ]
      },
      {
        "title": "Worker Heartbeat Health",
        "targets": [
          {
            "expr": "temporal_worker_task_queue_lag"
          }
        ]
      },
      {
        "title": "Workflow SLA Compliance",
        "targets": [
          {
            "expr": "temporal_workflow_completed_on_time_total / temporal_workflow_completed_total * 100"
          }
        ]
      }
    ],
    "refresh": "30s",
    "time": {
      "from": "now-24h",
      "to": "now"
    }
  }
}
```

#### Alerting Rules

Create `prometheus/alert-rules.yml`:

```yaml
groups:
  - name: temporal_alerts
    interval: 5m
    rules:
      - alert: HighWorkflowFailureRate
        expr: sum(rate(temporal_workflow_failed_total[5m])) > 0.05
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High workflow failure rate (>5%)"
          description: "{{ $value | humanizePercentage }} of workflows failing"

      - alert: LongRunningWorkflows
        expr: |
          count(temporal_workflow_running_seconds > 7200) > 10
        for: 5m
        labels:
          severity: info
        annotations:
          summary: "{{ $value }} workflows running >2 hours"

      - alert: WorkerBacklog
        expr: temporal_worker_task_queue_lag > 1000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Task queue lag > 1000 tasks"

      - alert: TemporalServerDown
        expr: up{job="temporal-server"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Temporal server is down"
```

### Step 6: Using the Dashboard

#### Typical Workflows

**Incident Response:**
1. User reports stuck workflow in Temporal UI
2. Ops team opens Temporal Admin Dashboard
3. Filter: `status = 'pending'` and `elapsed_time > '2h'`
4. Select workflow → click "Terminate"
5. Reason logged in action history

**Escalation:**
1. SLA violation detected by Prometheus alert
2. Dashboard shows "Pending > 2h" view
3. Select workflow → "Send Signal" with `priority: 1`
4. Workflow picks up signal and escalates

**Audit & Compliance:**
1. Monthly audit requested
2. API: `GET /api/temporal/workflows/audit-trail?start=2024-01-01&end=2024-01-31`
3. Export to CSV/JSON for compliance report
4. Grafana dashboard shows SLA compliance %, cycle time, error rates

## Comparison with Workday

| Capability | Workday BPF | Temporal + Platform |
|---|---|---|
| Always-on audit | ✅ 100% capture | ✅ Full event history export |
| Configurable visibility | ✅ Role-based | ✅ Saved Views + Search Attributes |
| Real-time dashboards | ✅ 170+ prebuilt | ✅ Grafana + custom metrics |
| Process reports | ✅ 5,000+ prebuilt | ✅ History export + BI queries |
| Admin controls | ✅ Approve/reject | ✅ Signal/Update/Cancel/Terminate/Reset |
| SLA tracking | ✅ Built-in | ✅ Priority + SlaDeadline attributes |
| Escalation | ✅ Configurable | ✅ Workflow signals + alerts |

## Next Steps

1. **Deploy to production**: Update docker-compose, test with production workloads
2. **Add more Search Attributes**: CustomerSegment, Region, RiskProfile based on your needs
3. **Build custom Grafana dashboards**: KPIs specific to your business (cycle time, approval rate, etc.)
4. **Integrate with incident tracking**: Wire Prometheus alerts to PagerDuty/Slack
5. **Train operations teams**: Document runbooks for common scenarios
6. **Enable audit logging**: Configure Temporal audit logging integration for regulatory compliance

## File Manifest

### Backend (Go)

- `backend/internal/temporal/search_attributes.go` - Search Attribute definitions and registration
- `backend/internal/temporal/workflow_admin.go` - Admin operations (Signal, Update, Cancel, Terminate, Reset)
- `backend/internal/temporal/history_export.go` - History export for audit & analytics
- `backend/internal/api/temporal_admin.go` - REST API endpoints

### Frontend (React/TypeScript)

- `frontend/src/pages/TemporalAdminDashboard.tsx` - Admin dashboard component
- `frontend/src/pages/TemporalAdminDashboard.css` - Dashboard styling

### Infrastructure (Docker/Config)

- `prometheus/prometheus.yml` - Prometheus scrape config
- `prometheus/alert-rules.yml` - Alert rules
- `grafana/provisioning/dashboards/temporal-workflows.json` - Dashboard template
- docker-compose.yml updates: prometheus, grafana services

## Support & Troubleshooting

**Q: How do I reset a workflow to a specific decision point?**
A: Use the CLI: `temporal workflow reset --workflow-id <id> --reset-type LastWorkflowTask`

**Q: Can I batch-signal 100 workflows?**
A: Use Temporal's ListWorkflow API to get the list, then loop with SignalWorkflow calls.

**Q: How do I know if a Search Attribute is registered?**
A: Check Temporal UI → Namespace → Search Attributes, or run `temporal operator search-attribute list`

**Q: Can I customize the Grafana dashboard?**
A: Yes! Login as admin (default: admin/admin) and modify dashboards in Grafana UI. Changes are persisted.

---

**Implementation Date**: October 22, 2025
**Status**: Ready for Deployment
**Tech Stack**: Go 1.21+, React 18+, Temporal 1.37+, Prometheus, Grafana
