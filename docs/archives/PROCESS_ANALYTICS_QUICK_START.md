# Process Analytics Dashboard - Quick Start Guide

## 🚀 5-Minute Setup

### Step 1: Run Database Migration
```bash
cd /Users/eganpj/GitHub/semlayer
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable \
  -f backend/migrations/misc/process_analytics_schema.sql
```

**What it creates**:
- `process_execution_metrics` table
- `process_bottleneck_analysis` table
- `process_optimization_recommendations` table
- `process_performance_baselines` table

### Step 2: Add Dashboard to Navigation
Edit your Fabric Builder router (e.g., `frontend/src/App.tsx`):

```typescript
import { ProcessAnalyticsDashboard } from './components/BPBuilder/ProcessAnalyticsDashboard';

// Add route
<Route 
  path="/process-analytics" 
  element={
    <ProcessAnalyticsDashboard 
      tenant={selectedTenant!} 
      datasource={selectedDatasource!} 
    />
  } 
/>
```

### Step 3: Add Navigation Link
In your sidebar/header navigation:

```typescript
<Link to="/process-analytics" className="nav-link">
  <Brain className="w-5 h-5" />
  Process Analytics
</Link>
```

### Step 4: Start Collecting Metrics
In your workflow execution code:

```go
import bp "github.com/hondyman/semlayer/backend/internal/business_process"

// Initialize collector (do this once at startup)
collector := bp.NewProcessMetricsCollector(db, temporalClient)

// Start background worker (collects metrics every 5 minutes)
go collector.StartMetricsCollectionWorker(context.Background(), 5*time.Minute)

// Or manually record metrics
collector.RecordStepStart(ctx, bp.StepMetric{
    WorkflowID:   workflowID,
    WorkflowType: "ExpenseApproval",
    TenantID:     tenantID,
    StepName:     "Validate Data",
    StepType:     "validate",
    StartTime:    time.Now(),
    Status:       "running",
    Metadata:     map[string]interface{}{"user_id": "123"},
})

// When step completes
collector.RecordStepCompletion(ctx, workflowID, "Validate Data", 
    time.Now(), "completed", nil)
```

### Step 5: Generate Test Data (Optional)
```bash
# Create sample workflow metrics
curl -X POST http://localhost:8080/api/test/generate-workflow-metrics \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "00000000-0000-0000-0000-000000000000",
    "workflow_count": 100,
    "workflow_type": "ExpenseApproval"
  }'
```

---

## 📊 Using the Dashboard

### View 1: Overview
- **KPI Cards**: Total workflows, success rate, avg duration, active bottlenecks
- **Trend Chart**: 14-day workflow volume and success rate trends
- **Success Rate Pie**: Distribution of completed/active/failed workflows
- **Top Bottlenecks**: 5 worst performing steps with severity scores

**Actions**:
- Click "Analyze" to run ML bottleneck detection
- Toggle "Live" for 30s auto-refresh
- Click "Refresh" to update manually

### View 2: Bottlenecks
- **Detailed Bottleneck List**: All detected performance issues
- **Step Performance Table**: Per-step metrics (duration, success rate, execution count)
- **Severity Scores**: Red (>70%), Orange (40-70%), Yellow (<40%)

**Actions**:
- Select workflow type to see step-level analysis
- Review recommendations for each bottleneck
- Identify steps marked as "Bottleneck" in the table

### View 3: Recommendations
- **AI-Generated Optimizations**: Suggested improvements with priority and impact scores
- **Implementation Details**: Specific steps to optimize (e.g., "add parallel execution")
- **Priority Levels**: High (red), Medium (yellow), Low (blue)

**Actions**:
- Click "Implement" to accept recommendation
- Click "Dismiss" to ignore
- Review expected impact percentage

### View 4: Predictions
- **ML Duration Forecast**: Predicted workflow completion time
- **Confidence Interval**: 95% confidence range
- **Prediction Factors**: What's affecting the prediction (parallel execution, bottlenecks, time of day)

**Actions**:
- Select workflow type to see prediction
- Review factors and their impact (+/- percentages)
- Use prediction to set realistic SLAs

---

## 🔧 API Integration Examples

### Dashboard Stats
```typescript
// In your React component
const fetchDashboardStats = async () => {
  const response = await fetch(
    `/api/process-analytics/dashboard?tenant_id=${tenantId}&datasource_id=${datasourceId}`
  );
  const stats = await response.json();
  console.log(stats);
};
```

### Run Bottleneck Analysis
```typescript
const analyzeBottlenecks = async () => {
  await fetch(
    `/api/process-analytics/analyze-bottlenecks?tenant_id=${tenantId}`,
    { method: 'POST' }
  );
  // Refresh dashboard after analysis
  fetchDashboardStats();
};
```

### Get Predictions
```typescript
const getPrediction = async (workflowType: string) => {
  const response = await fetch(
    `/api/process-analytics/predict-duration?tenant_id=${tenantId}&workflow_type=${workflowType}`
  );
  const prediction = await response.json();
  console.log(`Predicted duration: ${prediction.predicted_minutes} minutes`);
};
```

---

## 🧪 Testing Workflow

1. **Create workflows** (manually or via test script)
2. **Record metrics** for each step execution
3. **Wait 5-10 minutes** for metrics to accumulate
4. **Run bottleneck analysis**: `POST /api/process-analytics/analyze-bottlenecks`
5. **View dashboard**: Navigate to `/process-analytics`
6. **Check predictions**: Select a workflow type to see ML forecast

---

## 🎯 What You Get

### Real-Time Monitoring
- Live workflow execution status
- 30-second auto-refresh
- Active bottleneck alerts

### Predictive Intelligence
- ML-based duration forecasting
- 95% confidence intervals
- Multi-factor impact analysis

### AI Optimization
- Automatic bottleneck detection
- Severity scoring (0-100%)
- Implementation recommendations

### Beautiful Visualizations
- Gradient KPI cards with trend indicators
- Interactive trend charts
- Success rate pie charts
- Step performance tables

---

## 🚨 Troubleshooting

### No data showing?
1. Ensure database migration ran successfully
2. Check that metrics are being recorded (query `process_execution_metrics` table)
3. Verify tenant_id matches between metrics and dashboard

### Bottlenecks not detected?
1. Run analysis: `POST /api/process-analytics/analyze-bottlenecks`
2. Ensure at least 5 executions per workflow type
3. Check that step durations are being recorded

### Predictions showing 0?
1. Need at least 1 completed workflow
2. Verify workflow_type matches exactly
3. Check that end_time is set for completed steps

---

## 📈 Next Steps

1. **Add to navigation** so users can access the dashboard
2. **Set up cron job** to run bottleneck analysis hourly:
   ```go
   c := cron.New()
   c.AddFunc("@hourly", func() {
       http.Post("http://localhost:8080/api/process-analytics/analyze-bottlenecks?tenant_id=...", "", nil)
   })
   c.Start()
   ```
3. **Configure alerts** for critical bottlenecks (severity >70%)
4. **Train users** on reading the dashboard
5. **Monitor impact** of optimizations

---

## ✅ You're Ready!

The Process Analytics Dashboard is **fully operational** and ready to give you enterprise-grade insights into your workflows.

**Time to value**: 5 minutes from setup to first insights! 🚀
