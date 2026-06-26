# Enterprise BP Branching - Quick Start & Deployment Guide

## 🚀 5-Minute Quick Start

### Step 1: Database Schema (1 min)

```bash
# Apply the branching schema to your PostgreSQL database
psql -U postgres -d alpha << 'EOF'

-- Run everything from backend/pkg/bp/branching_schema.sql
-- Key tables created:
-- - bp_branch_executions (branch execution history)
-- - bp_branch_metrics (aggregated performance)
-- - bp_join_convergences (parallel join tracking)
-- - bp_ml_models (ML model configuration)
-- - bp_ab_tests (A/B test configurations)
-- - bp_branch_events (event-based routing)
-- - bp_branch_anomalies (anomaly detection)

EOF
```

### Step 2: Backend Integration (2 min)

```go
// In backend/cmd/server/main.go

import "github.com/eganpj/semlayer/backend/internal/api"

// Register branching routes
branchingHandlers := api.NewBranchingHandlers(db)
branchingHandlers.RegisterRoutes(r)  // r is chi.Router

// Your router should be configured like:
// r.Route("/api", func(r chi.Router) {
//     branchingHandlers.RegisterRoutes(r)
// })
```

### Step 3: Rebuild & Test (2 min)

```bash
# From backend directory
cd backend
go build -o ./bin/server ./cmd/server
./bin/server

# From another terminal, test an endpoint
curl -X GET http://localhost:8080/api/bp/branching/config/examples \
  -H "X-Tenant-ID: {your-tenant-id}" \
  -H "X-Tenant-Datasource-ID: {your-datasource-id}"
```

✅ **Done!** Your branching system is now live!

---

## 📊 Configuration Examples

### Example 1: Credit Application Approval Routing

```bash
curl -X POST http://localhost:8080/api/bp/branching/evaluate \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $(uuidgen)" \
  -H "X-Tenant-Datasource-ID: $(uuidgen)" \
  -d '{
    "workflow_id": "'$(uuidgen)'",
    "step_id": "'$(uuidgen)'",
    "branching_config": {
      "type": "exclusive",
      "branches": [
        {
          "id": "vip-fast-track",
          "priority": 1,
          "condition": {
            "type": "and",
            "rules": [
              {"field": "applicant.credit_score", "operator": "gte", "value": 750},
              {"field": "applicant.annual_income", "operator": "gte", "value": 100000}
            ]
          },
          "steps": ["auto-approve-vip"]
        },
        {
          "id": "standard-approval",
          "priority": 2,
          "condition": {
            "type": "and",
            "rules": [
              {"field": "applicant.credit_score", "operator": "gte", "value": 650}
            ]
          },
          "steps": ["manager-review"]
        },
        {
          "id": "manual-review",
          "priority": 3,
          "steps": ["risk-analyst-review", "compliance-check"]
        }
      ],
      "default_branch_id": "manual-review"
    },
    "data": {
      "applicant": {
        "credit_score": 720,
        "annual_income": 85000
      }
    }
  }'
```

Response:
```json
{
  "selected_branches": [
    {
      "id": "standard-approval",
      "label": "Standard Approval",
      "steps": ["manager-review"]
    }
  ],
  "evaluation_time_ms": 7,
  "selection_method": "exclusive"
}
```

### Example 2: Parallel Background Checks

```bash
curl -X POST http://localhost:8080/api/bp/branching/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "branching_config": {
      "type": "parallel",
      "branches": [
        {
          "id": "criminal-check",
          "label": "Criminal Background Check",
          "steps": ["call-sterling-api", "review-results"],
          "critical": true,
          "sla_hours": 48
        },
        {
          "id": "employment-verify",
          "label": "Employment Verification",
          "steps": ["verify-employment"],
          "critical": true,
          "sla_hours": 72
        },
        {
          "id": "education-verify",
          "label": "Education Verification",
          "condition": {"field": "position_level", "operator": "in", "value": ["senior", "executive"]},
          "steps": ["verify-education"],
          "critical": false
        }
      ],
      "join_config": {
        "strategy": "wait_all",
        "timeout_action": "proceed_with_warning",
        "timeout_hours": 168,
        "critical_only": true
      }
    },
    "data": {
      "position_level": "senior"
    }
  }'
```

### Example 3: ML-Powered Fraud Detection

```bash
# First, register your ML model
curl -X POST http://localhost:8080/api/bp/branching/ml-models \
  -H "Content-Type: application/json" \
  -d '{
    "model_id": "fraud-detector-v2",
    "model_endpoint": "https://ml-api.yourcompany.com/predict/fraud",
    "input_features": [
      "order.amount",
      "customer.account_age_days",
      "payment.card_velocity_24h",
      "shipping.address_match_score"
    ],
    "confidence_threshold": 0.75
  }'

# Then evaluate with ML-powered branching
curl -X POST http://localhost:8080/api/bp/branching/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "branching_config": {
      "type": "ml_powered",
      "ml_config": {
        "model_id": "fraud-detector-v2",
        "model_endpoint": "https://ml-api.yourcompany.com/predict/fraud",
        "input_features": [
          "order.amount",
          "customer.account_age_days",
          "payment.card_velocity_24h"
        ],
        "confidence_threshold": 0.75,
        "fallback_strategy": "conservative"
      },
      "branches": [
        {
          "id": "high-risk",
          "condition": {"type": "ml_score", "operator": "gte", "threshold": 0.8},
          "steps": ["fraud-analyst-review", "enhanced-verification"]
        },
        {
          "id": "medium-risk",
          "condition": {"type": "ml_score", "operator": "between", "threshold_min": 0.5, "threshold_max": 0.8},
          "steps": ["3ds-challenge"]
        },
        {
          "id": "low-risk",
          "condition": {"type": "ml_score", "operator": "lt", "threshold": 0.5},
          "steps": ["auto-approve"]
        }
      ]
    },
    "data": {
      "order": {"amount": 2500},
      "customer": {"account_age_days": 365},
      "payment": {"card_velocity_24h": 3}
    }
  }'
```

### Example 4: A/B Testing New Approval Workflow

```bash
# Start an A/B test
curl -X POST http://localhost:8080/api/bp/branching/ab-tests \
  -H "Content-Type: application/json" \
  -d '{
    "step_id": "'$(uuidgen)'",
    "test_name": "AI-Assisted Approval vs Standard",
    "control_branch_id": "standard-approval",
    "experiment_branch_id": "ai-assisted-approval",
    "control_weight": 0.7,
    "experiment_weight": 0.3,
    "duration_days": 30
  }'

# Check test status
curl -X GET http://localhost:8080/api/bp/branching/ab-tests/{testID}
```

---

## 📈 Monitoring & Analytics

### View Branch Metrics

```bash
# Get metrics for a specific step
curl -X GET "http://localhost:8080/api/bp/branching/metrics/{stepID}" \
  -H "X-Tenant-ID: {tenant-id}"

# Response:
{
  "metrics": [
    {
      "branch_id": "vip-fast-track",
      "branch_label": "VIP Fast Track",
      "execution_count": 1250,
      "completed_count": 1248,
      "timeout_count": 2,
      "completion_rate": 99.84,
      "avg_duration_ms": 150,
      "avg_ml_score": 0.92
    },
    ...
  ],
  "timestamp": "2025-10-21T15:30:00Z"
}
```

### View Branch Performance

```bash
curl -X GET "http://localhost:8080/api/bp/branching/branch-performance/{branchID}"

# Response:
{
  "branch_id": "vip-fast-track",
  "total_count": 1250,
  "success_rate": 99.84,
  "avg_duration_ms": 150,
  "p95_duration_ms": 320,
  "failed_count": 2,
  "timeout_count": 0
}
```

### View Process-Level Summary

```bash
curl -X GET "http://localhost:8080/api/bp/branching/metrics/summary/{processID}"

# Response:
{
  "total_executions": 5432,
  "avg_duration_ms": 175,
  "completion_rate": 99.92,
  "avg_ml_score": 0.88
}
```

### Check for Anomalies

```bash
curl -X GET "http://localhost:8080/api/bp/branching/anomalies" \
  -H "X-Tenant-ID: {tenant-id}"

# Response:
{
  "anomalies": [
    {
      "id": "uuid",
      "anomaly_type": "latency_spike",
      "severity": "high",
      "description": "Branch execution time increased by 250%",
      "affected_executions": 47,
      "detected_at": "2025-10-21T14:30:00Z",
      "investigation_status": "open"
    }
  ],
  "count": 1
}
```

---

## 🔧 Common Workflows

### Adding a New Branch Type

1. **Update schema** if new tables needed
2. **Add to BranchingConfig** JSON structure
3. **Implement evaluator** in `BranchEvaluator.EvaluateBranches()`
4. **Add tests** for new type
5. **Document** with examples
6. **Deploy** with feature flag (optional)

### Integrating ML Model

```bash
# 1. Register model
curl -X POST /api/bp/branching/ml-models \
  -d '{"model_id": "my-model", "model_endpoint": "...", ...}'

# 2. Update branching config to use it
{
  "type": "ml_powered",
  "ml_config": {
    "model_id": "my-model",
    ...
  }
}

# 3. Monitor performance
curl -X GET /api/bp/branching/ml-models/my-model/performance
```

### Setting Up Monitoring Alerts

```bash
# Check for high latency anomalies
SELECT * FROM bp_branch_anomalies 
WHERE anomaly_type = 'latency_spike' 
  AND severity IN ('high', 'critical')
  AND investigation_status IN ('open', 'investigating');

# Check for high failure rates
SELECT branch_id, completion_rate 
FROM bp_branch_metrics 
WHERE completion_rate < 95 
ORDER BY completion_rate;

# Check for selection bias
SELECT branch_id, COUNT(*) as selection_count
FROM bp_branch_executions
WHERE DATE(created_at) = CURRENT_DATE
GROUP BY branch_id
ORDER BY selection_count DESC;
```

---

## ✅ Verification Checklist

After deployment:

- [ ] Database schema applied without errors
- [ ] Backend compiles and starts successfully
- [ ] API endpoints respond to requests
- [ ] Branch execution is logged to database
- [ ] Metrics are being collected
- [ ] Dashboard displays branch data
- [ ] ML models (if used) are registered
- [ ] A/B tests can be created and monitored
- [ ] Anomalies are being detected
- [ ] No errors in application logs

---

## 🆘 Troubleshooting

### Issue: "No matching branch found"
**Solution**: 
- Check that at least one condition matches
- Verify condition field paths exist in data
- Add a `default_branch_id` as fallback

### Issue: High latency on parallel branches
**Solution**:
- Check slowest branch SLA
- Consider `first_complete` strategy instead
- Add timeouts to parallel gateway

### Issue: ML model predictions inconsistent
**Solution**:
- Check confidence_threshold setting
- Review fallback_strategy (conservative recommended)
- Monitor model performance and drift

### Issue: Join convergence hanging
**Solution**:
- Verify `required_branches` matches actual branch count
- Check join timeout configuration
- Review branch execution logs for failures

---

## 📚 Additional Resources

- **Architecture**: See `BP_BRANCHING_SYSTEM.md`
- **Database Schema**: `backend/pkg/bp/branching_schema.sql`
- **API Handlers**: `backend/internal/api/bp_branching_handlers.go`
- **Evaluator Engine**: `backend/pkg/bp/branch_evaluator.go`
- **Dashboard**: See React component documentation

---

## 🎯 Success Metrics

After 30 days, you should see:

- **Branch Accuracy**: 95%+ of decisions meeting business criteria
- **Processing Time**: <500ms average branch evaluation
- **Coverage**: All workflow steps have appropriate branching
- **Anomaly Detection**: Proactive alerts on 80%+ of issues
- **A/B Test Winners**: Statistical significance in 70%+ of tests

---

**Status**: ✅ Production Ready  
**Version**: 1.0  
**Last Updated**: October 21, 2025
