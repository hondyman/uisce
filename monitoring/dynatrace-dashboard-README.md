# Dynatrace Dashboard for Semlayer Governance Platform

This dashboard provides real-time observability for your governance-native semantic platform, giving SREs, stewards, and product leads a unified view of performance, governance health, and conversational adoption.

## 📊 Dashboard Overview

The dashboard is organized into 5 key sections:

### 1. Hot-Path Performance & Concurrency
**Purpose**: Monitor the Access Intelligence Service's real-time evaluation loop under peak load.

**Key Metrics**:
- `semlayer.access_evaluation.duration` - Service method latency (p50/p95/p99)
- `semlayer.access_evaluation.active` - Concurrent evaluations vs concurrency limit
- `semlayer.cache.hit_ratio` - Cache health and performance
- `semlayer.cache.invalidations` - Cache invalidation rate

**SLO Targets**:
- p95 Evaluate() latency < 8ms
- Cache hit ratio > 90%
- Error rate < 1%

### 2. Conversational Layer Load & Responsiveness
**Purpose**: Monitor multi-turn NL→query sessions for latency, guardrail activity, and adoption.

**Key Metrics**:
- `semlayer.conversational.turn.duration` - Turn latency by conversation length
- `semlayer.guardrail.interventions` - Guardrail intervention rate
- `semlayer.conversations.active` - Active concurrent conversations

**Business Events**:
- "Guardrail Triggered" - Links to Decision Trace
- Conversation start/end events

### 3. Governance KPIs
**Purpose**: Live view of governance posture for stewards.

**Key Metrics**:
- `semlayer.governance.manual_grants` - Manual grant rate trend
- `semlayer.governance.drift_backlog` - Active drifted claims
- `semlayer.governance.conflict_resolution_time` - Mean time to resolve conflicts

**Davis AI Alerts**:
- Drift backlog spikes
- Manual grant rate anomalies

### 4. Adoption & Change-Management Metrics
**Purpose**: Track rollout success and training impact.

**Key Metrics**:
- `semlayer.conversational.success_rate` - NL query success rate
- Feature uptake metrics
- Pilot vs control cohort comparisons

### 5. Alert & Incident Panel
**Purpose**: Centralized governance and performance alerts.

**Features**:
- Active alerts grouped by severity
- Recent incidents with root cause
- SLO breach notifications

## 🚀 Importing the Dashboard

### Option 1: Via Dynatrace UI
1. Go to **Dashboards** in your Dynatrace environment
2. Click **Upload dashboard**
3. Select the `dynatrace-dashboard.json` file
4. Review and adjust any entity selectors as needed

### Option 2: Via Dynatrace API
```bash
curl -X POST "https://your-environment.live.dynatrace.com/api/config/v1/dashboards" \
  -H "Authorization: Api-Token YOUR_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d @monitoring/dynatrace-dashboard.json
```

## 🔧 Configuration Requirements

### Environment Variables
Ensure your application is configured with:
```bash
export DT_ENDPOINT="https://your-dynatrace-environment.live.dynatrace.com/api/v2/otlp"
export DT_API_TOKEN="your-dynatrace-api-token"
export OTEL_SERVICE_NAME="semlayer.backend"
export OTEL_SERVICE_VERSION="1.0.0"
```

### Custom Metrics Setup
The dashboard expects these custom metrics to be available:

#### Performance Metrics
```javascript
// Example metric ingestion
{
  "metric": "semlayer.access_evaluation.duration",
  "value": 5.2,
  "dimensions": {
    "tenant_id": "tenant_123",
    "cache.hit": "true",
    "decision": "allow"
  }
}
```

#### Business Events
```javascript
// Governance action event
{
  "event.type": "semlayer.governance.action",
  "action.type": "approve",
  "steward.id": "hashed_user_id",
  "tenant.id": "tenant_123",
  "duration.ms": 150
}
```

## 📈 Metric Mapping

| Dashboard Tile | Backend Implementation | Metric Source |
|---|---|---|
| Service Method Latency | `DynatraceManager.TraceAccessEvaluation` | OpenTelemetry spans |
| Cache Health | `ShardedCache` hit/miss counters | Custom metrics |
| Guardrail Interventions | `NLQueryEngine` compliance checks | Business events |
| Governance KPIs | `AccessIntelligenceService` audit logs | Database queries |
| Conversational Success | `ConversationalLoadTester` results | Trace attributes |

## 🎯 SLO Configuration

### Access Intelligence SLO
```json
{
  "name": "Access Intelligence Latency",
  "metric": "semlayer.access_evaluation.duration",
  "target": 8,
  "warning": 15,
  "critical": 30,
  "timeframe": "-1h"
}
```

### Cache Performance SLO
```json
{
  "name": "Cache Hit Ratio",
  "metric": "semlayer.cache.hit_ratio",
  "target": 90,
  "warning": 80,
  "critical": 70,
  "timeframe": "-1h"
}
```

## 🔍 Drill-Down Capabilities

The dashboard supports multi-dimensional analysis:

- **By Tenant**: Filter by `tenant_id` for multi-tenant isolation
- **By User**: Hashed user IDs for privacy-compliant analysis
- **By Asset**: Asset-level performance and governance metrics
- **By Conversation**: Trace conversational flows end-to-end
- **By Feature**: Compare pilot vs control cohorts

## 📋 Maintenance

### Regular Tasks
- Review SLO targets quarterly
- Update metric names if backend changes
- Add new tiles for emerging KPIs
- Archive old incident data

### Alert Configuration
Set up Davis AI anomaly detection for:
- Latency spikes
- Error rate increases
- Cache performance degradation
- Governance backlog growth

## 🤝 Team Access

### Recommended Permissions
- **SRE Team**: Full edit access
- **Governance Stewards**: View access + dashboard sharing
- **Product Leads**: View access for adoption metrics
- **Developers**: View access for debugging

### Dashboard Sharing
- Save as shared dashboard
- Use management zones for tenant isolation
- Set up automated exports for compliance

## 📞 Support

For dashboard customization or metric issues:
1. Check Dynatrace logs for ingestion errors
2. Verify OpenTelemetry configuration
3. Review backend tracing implementation
4. Contact platform SRE team
