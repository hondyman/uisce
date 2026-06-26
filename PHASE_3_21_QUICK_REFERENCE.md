# Phase 3.21 Quick Reference Guide

**For:** ML Engineers, Data Engineers, Operations Team  
**Purpose:** Day-1 operational guide for feature engineering platform  
**Read Time:** 5 minutes

---

## 🚀 Quick Start (5 minutes)

### Step 1: Check Status
```bash
# Verify all services running
kubectl get pods -l app=drift-detection,app=importance-service
kubectl logs deployment/drift-detection | grep "Application startup"

# Health check
curl http://drift-detection:8000/health/ready
```

### Step 2: Detect Feature Drift
```bash
# POST request to detect drift
curl -X POST http://drift-detection:8000/api/v1/drift/detect \
  -H "Content-Type: application/json" \
  -d '{
    "feature_id": "feature:orders.monthly_revenue_v1",
    "baseline_window": "30d",
    "eval_window": "7d",
    "method": "ks"
  }'

# Response includes:
# - ks_score [0,1]
# - is_drifted: true/false
# - percentile_rank: [0,100]
# - recommendation: "High drift detected"
```

### Step 3: View Dashboard
```bash
# Open Grafana
open http://localhost:3000/d/phase-3-21-features

# Key panels:
# 1. Active Drifts (count in last 24h)
# 2. Drift Score Distribution (heatmap)
# 3. Feature Freshness Gauge (SLO: 1 hour)
# 4. Materialization Latency (p95 < 30s)
```

### Step 4: Check Alerts
```bash
# View active Prometheus alerts
curl http://prometheus:9090/api/v1/alerts

# Critical alerts:
# - HighFeatureDrift (KS > 0.15)
# - ExtremeFeatureDrift (KS > 0.3)
# - MultipleDriftsActive (>10 features)
# - FeatureFreshnessSLABreach (>2h old)
# - ComputationSLOBreach (<99% success)
```

---

## 📊 Key Endpoints

### Drift Detection API

```bash
# Detect drift for single feature
POST /api/v1/drift/detect
{
  "feature_id": "feature:name_v1",
  "baseline_window": "30d",
  "eval_window": "7d",
  "method": "ks"  # or "psi", "chi2", "classifier"
}
Response:
{
  "feature_id": "...",
  "ks_score": 0.15,
  "psi_score": 0.22,
  "chi2_score": 12.5,
  "classifier_score": 0.72,
  "is_drifted": true,
  "percentile_rank": 78,
  "recommendation": "High drift detected"
}

# Detect drift for multiple features (batch)
POST /api/v1/drift/batch
[ { "feature_id": "...", "method": "ks" }* ]

# Get current health status
GET /api/v1/drift/health/{feature_id}
Response: {
  "feature_id": "...",
  "last_computed": "2026-02-10T12:30:00Z",
  "active_drifts": 3,
  "recent_drifts_24h": 5,
  "recent_alerts": [...]
}

# Get all currently drifting features
GET /api/v1/drift/active
Response: [ { "feature_id": "...", "is_drifted": true, "score": 0.78 }* ]

# Get historical metrics for graphing  
GET /api/v1/drift/metrics/{feature_id}?days=30
Response: [ { "timestamp": "...", "ks_score": 0.15, "is_drifted": true }* ]

# Get feature metadata
GET /api/v1/features/metadata/{feature_id}
Response: {
  "feature_id": "...",
  "name": "Monthly Revenue",
  "owner": "data-team",
  "drift_config": { "method": "ks", "threshold": 0.15 },
  "test_cases": [...]
}
```

### Health & Monitoring

```bash
# Liveness probe (K8s)
GET /health/live
Response: 200 OK

# Readiness probe (K8s)
GET /health/ready  
Response: 200 OK (checks PostgreSQL, Iceberg)

# Prometheus metrics
GET /metrics
# Exports: drift_score, drift_alerts, drift_detection_duration, etc.
```

---

## 🎯 Common Tasks

### Task: Enable Drift Alerts for New Feature

```bash
# 1. Add feature to PostgreSQL
psql -U postgres -d semlayer << EOF
INSERT INTO feature_catalog 
(feature_id, name, owner, feature_type, expression, drift_config)
VALUES (
  'feature:new_feature_v1',
  'New Feature',
  'data-team',
  'numeric',
  'SELECT value FROM table WHERE date >= now() - interval 7 day',
  '{"method": "ks", "threshold": 0.15, "alert": true}'::jsonb
);
EOF

# 2. Trigger next drift detection run
# (Temporal workflow runs every 4 hours, or manually trigger)
curl -X POST http://temporal:7233/api/v1/workflows/start \
  -d '{"workflow": "DetectFeatureDrift", "feature_id": "feature:new_feature_v1"}'

# 3. Verify alert fires on next drift
kubectl logs deployment/drift-detection | grep "new_feature_v1"
```

### Task: Investigate High Feature Drift

```bash
# 1. Check drift alert
# (Grafana notification → Slack/email with feature_id)

# 2. Query drift details
curl http://drift-detection:8000/api/v1/drift/health/feature:problematic_v1

# 3. Review metrics over time
curl "http://drift-detection:8000/api/v1/drift/metrics/feature:problematic_v1?days=30"

# 4. Check upstream data quality
# Query feature_quality_checks table:
psql -U postgres -d semlayer << EOF
SELECT * FROM feature_quality_checks 
WHERE feature_id = 'feature:problematic_v1' 
ORDER BY computed_at DESC LIMIT 5;
EOF

# 5. Verify materialization freshness
psql -U postgres -d semlayer << EOF
SELECT feature_id, last_processed, 
  EXTRACT(EPOCH FROM (now() - last_processed))/3600 as hours_old
FROM feature_watermarks
WHERE feature_id = 'feature:problematic_v1';
EOF

# 6. If data is stale, manually trigger materialization
spark-submit spark_jobs/materialization.py \
  feature:problematic_v1 default us-east-1
```

### Task: Update Feature Importance Thresholds

```bash
# 1. Current configuration
psql -U postgres -d semlayer << EOF
SELECT feature_id, stability_score, trend, percentile_rank
FROM feature_importance
WHERE feature_id = 'feature:name_v1'
ORDER BY computed_at DESC LIMIT 1;
EOF

# 2. Edit alerting thresholds in importance_service/config.py
# - stability_alert_threshold = 0.6  (default)
# - importance_drop_threshold = 30   (percent)
# - importance_percentile_threshold = 10  (bottom 10%)

# 3. Rebuild and redeploy
docker build -t semlayer/importance-service:3.21.1 importance_service/
kubectl set image deployment/importance-service \
  importance-service=semlayer/importance-service:3.21.1
kubectl rollout status deployment/importance-service
```

### Task: Scale Drift Detection for More Features

```bash
# Current capacity: ~100 features with 4-hour cycle
# For 500+ features: increase replicas

kubectl scale deployment drift-detection --replicas=8

# HPA will auto-scale:
# - CPU >70% OR Memory >80% → scale up
# - CPU <30% AND Memory <50% → scale down
# - Max: 10 replicas

# Monitor scaling
kubectl get hpa drift-detection -w
```

---

## 🚨 Troubleshooting

### Problem: Drift Detection Latency >10s
```bash
# 1. Check service logs
kubectl logs -f deployment/drift-detection | grep "duration_ms"

# 2. Check PostgreSQL query performance
psql -U postgres -d semlayer << EOF
EXPLAIN ANALYZE
SELECT * FROM feature_drift_metrics 
WHERE feature_id = 'feature:name_v1' 
ORDER BY updated_at DESC LIMIT 100;
EOF

# 3. Check memory usage
kubectl top pod -l app=drift-detection
# If >1Gi, increase resource limits:
kubectl set resources deployment drift-detection --limits memory=2Gi

# 4. Reduce sample size if needed (config.py)
# - baseline_sample_size = 1000
# - eval_sample_size = 500
```

### Problem: Feature Freshness SLA Breach (>1h old)
```bash
# 1. Check materialization status
kubectl logs -f deployment/spark-driver | grep "ERROR"

# 2. Check watermark
psql -U postgres -d semlayer << EOF
SELECT feature_id, last_processed,
  EXTRACT(EPOCH FROM (now() - last_processed))/60 as minutes_old
FROM feature_watermarks
WHERE feature_id = 'feature:stale_v1';
EOF

# 3. Force materialization job
spark-submit spark_jobs/materialization.py \
  feature:stale_v1 default us-east-1

# 4. If still failing, check Spark cluster
kubectl get pods -l app=spark
```

### Problem: Quality Check Failures
```bash
# 1. Check which checks are failing
psql -U postgres -d semlayer << EOF
SELECT feature_id, check_type, check_value, result, fail_reason
FROM feature_quality_checks
WHERE result = 'FAIL'
ORDER BY computed_at DESC LIMIT 10;
EOF

# 2. Review feature expression/data
# Query the actual feature values:
SELECT * FROM analytics.{table} 
WHERE {your_feature_column} IS NULL 
LIMIT 10;

# 3. Update quality_config if thresholds too tight
psql -U postgres -d semlayer << EOF
UPDATE feature_catalog
SET quality_config = jsonb_set(
  quality_config, 
  '{null_rate_threshold}', 
  '0.10'::jsonb
)
WHERE feature_id = 'feature:name_v1';
EOF
```

### Problem: Alert Fatigue (Too Many Alerts)
```bash
# 1. Review alert thresholds in Prometheus
kubectl get configmap prometheus-rules -o yaml | grep threshold

# 2. Adjust sensitive alerts
# For false positives, increase:
# - baseline_window from 30d to 60d
# - alert_duration from 5m to 15m
# - drift_threshold from 0.15 to 0.20

# 3. Disable non-critical feature alerts
psql -U postgres -d semlayer << EOF
UPDATE feature_catalog
SET drift_config = jsonb_set(
  drift_config, 
  '{alert_enabled}', 
  'false'::jsonb
)
WHERE feature_id = 'feature:non_critical_v1';
EOF
```

---

## 📈 Key Metrics to Monitor

### Drift Detection Health
```
Metric                      Target              Alert
────────────────────────────────────────────────────────
Detection latency p99       <10s               >30s
Algorithms accuracy         >95%               <90%
Percentile ranking error    <5%                >10%
False positive rate         <5%                >10%
```

### Feature Importance Health
```
Metric                      Target              Alert
────────────────────────────────────────────────────────
Stability score             >0.7               <0.6
Computation time            <2m                >5m
Top-K accuracy              >90%               <80%
Trend correlation           >0.8               <0.7
```

### Materialization Health
```
Metric                      Target              Alert
────────────────────────────────────────────────────────
Feature freshness           ≤1h                >2h
Job success rate            ≥99%               <99%
Latency p95                 <30s               >60s
Data completeness           ≥99%               <95%
```

### Platform SLOs
```
Metric                      Target              Alert
────────────────────────────────────────────────────────
API availability            ≥99.9%             <99%
Database uptime             ≥99.95%            <99%
Computation success rate    ≥99%               <99%
Mean alert response time    <5min              >10min
```

---

## 🔍 Debugging Queries

### SQL Queries (PostgreSQL)

```sql
-- Find all drifting features (Right Now)
SELECT COUNT(*) as drifting_features
FROM active_drifts
WHERE updated_at > now() - interval 1 hour;

-- Top 10 most problematic features
SELECT feature_id, COUNT(*) as drift_count
FROM feature_drift_metrics
WHERE is_drifted = true AND updated_at > now() - interval 24 hours
GROUP BY feature_id
ORDER BY drift_count DESC
LIMIT 10;

-- Feature importance trends (30 days)
SELECT 
  DATE(computed_at) as day,
  feature_id,
  AVG(shap_mean) as avg_importance,
  STDDEV(shap_mean) as importance_var
FROM feature_importance
WHERE computed_at > now() - interval 30 days
GROUP BY DATE(computed_at), feature_id
ORDER BY day DESC;

-- Quality check failures (Last 24h)
SELECT feature_id, check_type, fail_reason, COUNT(*) as failures
FROM feature_quality_checks
WHERE result = 'FAIL' AND computed_at > now() - interval 24 hours
GROUP BY feature_id, check_type, fail_reason
ORDER BY failures DESC;

-- Feature lineage (upstream dependencies)
SELECT * FROM get_feature_ancestors('feature:name_v1');

-- Feature health report
SELECT * FROM get_feature_health('feature:name_v1');
```

---

## 📞 Getting Help

### Emergency Escalation (Critical Alert)
- **Page:** On-call ML Operations engineer (from Slack/PagerDuty)
- **Severity:** Red alerts (ExtremeFeatureDrift, ComputationSLOBreach, etc.)
- **Response:** <5 minutes

### Standard Issues
- **Slack Channel:** #feature-engineering
- **Documentation:** PHASE_3_21_COMPLETE.md
- **On-call:** Monday-Friday 9am-6pm PT

### Known Issues & Workarounds
```
Issue: KS test fails with small sample sizes (<50)
Workaround: Use PSI or Chi-square for small samples
Risk: May miss real drift

Issue: SHAP values slow with 100+ features
Workaround: Sample down to top 50 features by gain importance
Risk: Top-K accuracy reduced

Issue: Drift alert fired but data looks OK
Workaround: Check historical baseline (last 30 days may include anomaly)
Action: Manually approve as "false positive" (decreases future sensitivity)
```

---

## 🎓 Learning Resources

1. **Drift Detection Deep Dive**
   - Read: PHASE_3_21_COMPLETE.md § Algorithms
   - Video: "Statistical Drift Detection for ML" (internal training)

2. **Feature Importance Interpretation**
   - Read: SHAP documentation + model_explainability.md
   - Code: importance_service/pipeline.py (annotated)

3. **Operational Excellence**
   - Read: SLO Targets section in status doc
   - Runbook: PHASE_3_21_COMPLETE.md § Deployment Checklist

4. **Incident Response**
   - Read: Troubleshooting section above
   - Practice: Phase 3.21 post-mortems (weekly)

---

## 📋 Checklists

### Daily Operations
- [ ] Check Grafana dashboard (drift, freshness, SLO)
- [ ] Review active alerts (should be <5)
- [ ] Spot-check 2-3 feature metrics
- [ ] Verify all 3 services healthy (kubectl get pods)

### Weekly Operations
- [ ] Review feature drift trends (any patterns?)
- [ ] Update alert thresholds if needed
- [ ] Check materialization success rate
- [ ] Review importance score distributions

### Monthly Operations
- [ ] Full SLO compliance review (99%+ target)
- [ ] Capacity planning (scaling needed?)
- [ ] Update documentation
- [ ] Training/knowledge transfer session

---

**Quick Reference Version:** 3.21.0  
**Last Updated:** February 10, 2026  
**Next Review:** March 10, 2026  
**Owner:** ML Operations Team
