# Quick Start: Temporal Workflow Governance

## 5-Minute Setup

### 1. Register Search Attributes

```bash
# Run from any directory with Temporal CLI
temporal operator search-attribute create --name BusinessUnit --type Keyword --yes
temporal operator search-attribute create --name SlaDeadline --type Datetime --yes
temporal operator search-attribute create --name Priority --type Int --yes
temporal operator search-attribute create --name ProcessOwner --type Keyword --yes
temporal operator search-attribute create --name CustomerID --type Keyword --yes
```

Or download script: `curl http://localhost:8080/api/temporal/setup-cli-script > setup.sh && bash setup.sh`

### 2. Add Routes to API

Edit `backend/internal/api/api.go` in the `/api` route block:

```go
import "go.temporal.io/sdk/client"
import httpapi "github.com/eganpj/semlayer/backend/internal/api"

// In your Server.RegisterRoutes:
r.Route("/api", func(r chi.Router) {
    // ... existing routes ...
    httpapi.RegisterTemporalAdminRoutes(r, temporalClient)
})
```

### 3. Add Frontend Dashboard

Edit `frontend/src/AppRoutes.tsx`:

```tsx
import TemporalAdminDashboard from './pages/TemporalAdminDashboard';

export const routes = [
  // ...
  {
    path: '/temporal-admin',
    element: <TemporalAdminDashboard />,
    label: 'Temporal Admin',
  },
];
```

### 4. Update docker-compose.yml

Add Prometheus and Grafana:

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
    depends_on:
      - temporal

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    depends_on:
      - prometheus
```

### 5. Create prometheus/prometheus.yml

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'temporal-server'
    static_configs:
      - targets: ['temporal:8233']
```

### 6. Start Services

```bash
docker-compose up -d
```

### 7. Test APIs

```bash
# View Search Attributes
curl http://localhost:8080/api/temporal/search-attributes

# Signal a workflow
curl -X POST http://localhost:8080/api/temporal/workflows/{workflow-id}/signal \
  -H "Content-Type: application/json" \
  -d '{"signal_name":"unblock","reason":"test"}'

# Access dashboards
# Frontend: http://localhost:5173/temporal-admin
# Grafana: http://localhost:3000 (admin/admin)
# Prometheus: http://localhost:9091
```

## Common Operations

### Filter Workflows

**Failed last 24 hours**: Use "Failed Last 24h" saved view
**Pending > 2 hours**: Use "Pending > 2h" saved view
**High priority**: Use "High Priority" saved view
**Custom filter**: Enter SQL query in search box

### Control a Workflow

1. Find workflow in list
2. Click one of: ▶️ (Signal), ⏸️ (Cancel), ⏹️ (Terminate), ↻ (Reset)
3. Enter reason → Confirm
4. View in "Recent Admin Actions"

### Export History

```bash
curl http://localhost:8080/api/temporal/workflows/{id}/history > history.json
```

### View Metrics

Grafana: http://localhost:3000 → Temporal Workflows dashboard

## Architecture

```
Frontend (React)
    ↓ HTTP
Backend (Go) → Temporal Server (gRPC)
    ↓            ↓
  Logs        Metrics (Prometheus)
              ↓
            Grafana Dashboards
```

## Environment Variables

```bash
TEMPORAL_ADDRESS=temporal:7233
TEMPORAL_NAMESPACE=default
PROMETHEUS_SCRAPE_INTERVAL=15s
GRAFANA_ADMIN_PASSWORD=admin
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Search Attributes not found | Run setup script: `/api/temporal/setup-cli-script` |
| API endpoints 404 | Verify routes registered in `/api` block |
| Grafana no data | Check Prometheus targets: http://localhost:9091/targets |
| Dashboard empty | Check filters and date range (default: last 24h) |

## Next Level

- Add more Search Attributes for your domain
- Create custom Grafana dashboards for your KPIs
- Wire Prometheus alerts to Slack/PagerDuty
- Export history to S3 for long-term archival
- Build custom reports from history JSON

---

**Questions?** Check `TEMPORAL_GOVERNANCE_IMPLEMENTATION.md` for full documentation.
