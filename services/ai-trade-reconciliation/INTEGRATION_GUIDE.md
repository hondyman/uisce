# ATR Integration Guide

This guide shows how to integrate the AI Trade Reconciliation (ATR) module with your existing Fabric Builder stack.

---

## 1. Database Integration

### Add ATR Tables to Hasura

After migrations run, expose the tables in Hasura:

```bash
# In Hasura console:
# Data → Databases → alpha → Track tables
# Track:
#   - trades
#   - trade_confirms
#   - reconciliation_results
#   - discrepancies
#   - reconciliation_tasks
#   - reconciliation_rules
#   - reconciliation_audit_logs
```

### GraphQL Subscriptions (Real-Time)

```graphql
subscription OnReconciliationUpdate {
  reconciliation_results(order_by: {run_date: desc}, limit: 1) {
    id
    run_date
    match_rate
    matched_count
    unmatched_count
    status
    discrepancies
  }
}
```

---

## 2. Temporal Workflow Integration

### Register ATR Workflow with Your Orchestrator

```go
// In your main Temporal setup
worker := worker.New(c, "your-namespace", worker.Options{})

// Register ATR workflows
worker.RegisterWorkflow(workflows.AIReconciliationWorkflow)

// Register all ATR activities
worker.RegisterActivity(activities.FetchYesterdaysTrades)
worker.RegisterActivity(activities.FetchTradeConfirms)
// ... register all activities ...

// Schedule daily run
err := worker.Start()
```

### Cron Schedule

ATR automatically runs via:
```go
// In AIReconciliationWorkflow
// Cron: "0 6 * * *" → Every day at 6 AM
```

### Link to Rebalancing

```go
// In RebalanceOrchestrator
var reconResult *ReconciliationResult
client.ExecuteWorkflow(ctx, reconciliationWorkflowID).Get(ctx, &reconResult)

if reconResult.MatchRate < 0.95 {
    // Wait 24 hours before rebalancing
    workflow.Sleep(ctx, 24*time.Hour)
}
```

---

## 3. API Integration

### Expose ATR Endpoints Through Your API Gateway

```yaml
# Your API Gateway Config (Kong, Envoy, etc.)
routes:
  - path: /api/reconciliation/*
    destination: atr-service:8080
    auth: require-token
    rate-limit: 100/minute
```

### Frontend Integration

```tsx
// In your React app
import AIReconciliationDashboard from '@/pages/reconciliation/Dashboard';
import RuleBuilder from '@/components/reconciliation/RuleBuilder';

// Add to navigation
const routes = [
  { path: '/reconciliation', component: AIReconciliationDashboard },
  { path: '/reconciliation/rules', component: RuleBuilder },
];
```

---

## 4. ABAC Policy Integration

### Define Access Policies

```json
{
  "id": "atr_ops_access",
  "effect": "allow",
  "subject": {
    "role": ["operations_manager", "compliance_officer"]
  },
  "action": [
    "view_reconciliation_results",
    "view_reconciliation_tasks",
    "update_reconciliation_task"
  ],
  "resource": "reconciliation:*",
  "condition": {
    "time_of_day": {
      "after": "06:00",
      "before": "22:00"
    },
    "portfolio_scope": {
      "in": ["portfolio.assigned_portfolios"]
    }
  }
}
```

### Enforce in Your Middleware

```go
// Your auth middleware
func RequireATRAccess(c *gin.Context) {
    // Check ABAC policy
    allowed, err := abacClient.Evaluate(ctx, &abac.Request{
        Subject:   getUserFromToken(c),
        Action:    c.Request.Method + ":" + c.Request.URL.Path,
        Resource:  "reconciliation:*",
    })
    
    if !allowed {
        c.JSON(403, gin.H{"error": "Access denied"})
        return
    }
    c.Next()
}
```

---

## 5. Notification Integration

### Integrate with Your Notification System

```go
// In CreateReconciliationTask activity
func CreateReconciliationTask(ctx context.Context, db *sql.DB, ...) error {
    // Create task...
    
    // Send notification
    client := notification.NewClient()
    client.SendAlert(&notification.Alert{
        Type:      "reconciliation_discrepancy",
        Severity:  discrepancy.Severity,
        Title:     fmt.Sprintf("Trade mismatch: %s", discrepancy.Field),
        Message:   discrepancy.SuggestedFix,
        TargetRole: "operations_manager",
    })
    
    return nil
}
```

### RabbitMQ Integration (Optional)

```go
// Publish events for other services
func PublishReconciliationEvent(ch *amqp.Channel, result *ReconciliationResult) error {
    body, _ := json.Marshal(result)
    return ch.Publish(
        "reconciliation_exchange",
        "reconciliation.completed",
        false, false,
        amqp.Publishing{ContentType: "application/json", Body: body},
    )
}
```

---

## 6. Semantic Object Integration

### Connect to Semantic Cubes

```go
// In your semantic layer
type TradeSemanticObject struct {
    Trade         models.Trade
    Reconciliation *ReconciliationResult
    Status        string
}

// Register with semantic engine
func RegisterTradeSemanticObject(engine *semantic.Engine) {
    engine.RegisterObject("trade", func(id string) interface{} {
        trade := getTradeByID(id)
        result := getLatestReconciliationForTrade(id)
        return &TradeSemanticObject{
            Trade:          trade,
            Reconciliation: result,
        }
    })
}
```

---

## 7. Metrics & Monitoring

### Export Prometheus Metrics

```go
// In your metrics setup
var (
    reconMatchRate = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "reconciliation_match_rate",
        },
        []string{"portfolio_id"},
    )
    reconTasksOpen = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "reconciliation_tasks_open",
        },
        []string{"severity"},
    )
)

// Update after each reconciliation
reconMatchRate.WithLabelValues(portfolioID).Set(result.MatchRate)
reconTasksOpen.WithLabelValues("high").Set(float64(highSeverityCount))
```

### Grafana Dashboards

Import the ATR dashboard template:
- Match rate trend
- Task aging analysis
- Rule application success rate
- API latency percentiles

---

## 8. Data Synchronization

### Load Historical Trades

```bash
# Bulk import trades from custodian files
go run backend/cmd/data-import/main.go \
  --source=sftp://custodian.example.com/trades \
  --format=csv \
  --date-range="2025-01-01..2025-10-30"
```

### Reconcile Historical Data

```bash
# Run reconciliation for past dates (backfill)
curl -X POST http://localhost:8080/api/reconciliation/backfill \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "2025-10-01",
    "end_date": "2025-10-30",
    "parallel_workers": 4
  }'
```

---

## 9. Testing Integration

### End-to-End Test

```bash
#!/bin/bash
# e2e-test.sh

# 1. Insert test data
psql $DATABASE_URL << EOF
INSERT INTO trades (...) VALUES (...);
INSERT INTO trade_confirms (...) VALUES (...);
EOF

# 2. Trigger reconciliation
curl -X POST http://localhost:8080/api/reconciliation/run-now

# 3. Poll for results
for i in {1..30}; do
  RESULT=$(curl http://localhost:8080/api/reconciliation/results/latest)
  if [ "$(echo $RESULT | jq -r '.status')" == "completed" ]; then
    echo "✅ Reconciliation completed"
    exit 0
  fi
  sleep 1
done

echo "❌ Reconciliation timeout"
exit 1
```

---

## 10. Tenant Scoping (Critical!)

### Multi-Tenant Setup

Per the `agents.md` runbook, all ATR endpoints must respect tenant scope:

```go
// In API middleware
func EnforceTenantScope(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    datasourceID := c.GetString("datasource_id")
    
    if tenantID == "" || datasourceID == "" {
        c.JSON(400, gin.H{"error": "tenant_id and datasource_id required"})
        return
    }
    
    c.Set("tenant_id", tenantID)
    c.Set("datasource_id", datasourceID)
    c.Next()
}

// Apply to all routes
api := router.Group("/api/reconciliation")
api.Use(EnforceTenantScope)
```

### Tenant-Scoped Queries

```go
// All queries MUST filter by tenant
func GetReconciliationResults(tenantID, datasourceID string) {
    db.Query(`
        SELECT * FROM reconciliation_results
        WHERE tenant_id = $1 AND datasource_id = $2
    `, tenantID, datasourceID)
}
```

---

## 11. Deployment Integration

### Add to Your Docker Compose

```yaml
# In your main docker-compose.yml
services:
  atr-service:
    image: your-registry/atr-service:latest
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - TEMPORAL_HOST=${TEMPORAL_HOST}
      - XAI_API_KEY=${XAI_API_KEY}
    depends_on:
      - postgres
      - temporal
    networks:
      - fabric-network
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atr-service
spec:
  replicas: 2
  selector:
    matchLabels:
      app: atr-service
  template:
    metadata:
      labels:
        app: atr-service
    spec:
      containers:
      - name: atr-service
        image: your-registry/atr-service:latest
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: url
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
```

---

## 12. Troubleshooting Integration Issues

### Issue: "AI reconciliation failing"

```bash
# 1. Check XAI API connectivity
curl -H "Authorization: Bearer $XAI_API_KEY" https://api.x.ai/v1/models

# 2. Check Temporal workflow execution
tctl wf show -w atr-reconciliation-workflow-id

# 3. Review activity logs
docker logs atr-service | grep "AIReconcile"
```

### Issue: "Tasks not being created"

```sql
-- Check discrepancies table
SELECT severity, COUNT(*) FROM discrepancies 
WHERE created_at > NOW() - INTERVAL '1 day'
GROUP BY severity;

-- Check tasks table
SELECT status, COUNT(*) FROM reconciliation_tasks 
GROUP BY status;
```

### Issue: "Performance degradation"

```sql
-- Analyze reconciliation_results table
ANALYZE reconciliation_results;
CREATE INDEX idx_results_tenant_date 
  ON reconciliation_results(tenant_id, run_date DESC);

-- Check slow queries
SELECT query, mean_exec_time FROM pg_stat_statements 
WHERE query LIKE '%reconciliation%' 
ORDER BY mean_exec_time DESC LIMIT 5;
```

---

## Next Steps

1. ✅ Deploy ATR service
2. ✅ Configure Hasura relationships
3. ✅ Set up Temporal scheduling
4. ✅ Create ABAC policies
5. ✅ Add monitoring/alerting
6. ✅ Run E2E tests
7. ✅ Train ops team
8. ✅ Go live!

---

**Questions?** Check the main README.md or DEPLOYMENT_CHECKLIST.md
