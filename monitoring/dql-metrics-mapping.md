# Dynatrace DQL Queries & Custom Metrics for Semlayer Dashboard

This file contains the exact DQL queries, custom metric definitions, and SLO configurations needed to make the Semlayer governance dashboard fully functional.

## 📊 Custom Metrics Setup

First, ensure these custom metrics are defined in your Dynatrace environment:

### Performance Metrics
```json
{
  "name": "semlayer.access_evaluation.duration",
  "description": "Duration of access evaluation in milliseconds",
  "unit": "MILLISECOND",
  "dimensions": ["tenant_id", "cache_hit", "decision"],
  "aggregation": "AVG"
}
```

```json
{
  "name": "semlayer.access_evaluation.active",
  "description": "Number of active concurrent evaluations",
  "unit": "COUNT",
  "dimensions": ["tenant_id"],
  "aggregation": "MAX"
}
```

```json
{
  "name": "semlayer.cache.hit_ratio",
  "description": "Cache hit ratio percentage",
  "unit": "PERCENT",
  "dimensions": ["tenant_id"],
  "aggregation": "AVG"
}
```

```json
{
  "name": "semlayer.cache.invalidations",
  "description": "Cache invalidation rate per minute",
  "unit": "COUNT",
  "dimensions": ["tenant_id"],
  "aggregation": "SUM"
}
```

### Conversational Metrics
```json
{
  "name": "semlayer.conversational.turn.duration",
  "description": "Duration of conversational turn processing",
  "unit": "MILLISECOND",
  "dimensions": ["conversation_id", "tenant_id", "conversation.length"],
  "aggregation": "AVG"
}
```

```json
{
  "name": "semlayer.guardrail.interventions",
  "description": "Number of guardrail interventions",
  "unit": "COUNT",
  "dimensions": ["tenant_id", "trigger_reason"],
  "aggregation": "SUM"
}
```

```json
{
  "name": "semlayer.conversations.active",
  "description": "Number of active conversations",
  "unit": "COUNT",
  "dimensions": ["tenant_id"],
  "aggregation": "MAX"
}
```

### Governance Metrics
```json
{
  "name": "semlayer.governance.manual_grants",
  "description": "Number of manual governance grants",
  "unit": "COUNT",
  "dimensions": ["tenant_id", "steward_id"],
  "aggregation": "SUM"
}
```

```json
{
  "name": "semlayer.governance.drift_backlog",
  "description": "Number of drifted claims awaiting action",
  "unit": "COUNT",
  "dimensions": ["tenant_id"],
  "aggregation": "MAX"
}
```

```json
{
  "name": "semlayer.governance.conflict_resolution_time",
  "description": "Time to resolve governance conflicts",
  "unit": "MILLISECOND",
  "dimensions": ["tenant_id"],
  "aggregation": "AVG"
}
```

### Adoption Metrics
```json
{
  "name": "semlayer.conversational.success_rate",
  "description": "Conversational query success rate",
  "unit": "PERCENT",
  "dimensions": ["tenant_id"],
  "aggregation": "AVG"
}
```

## 🔍 DQL Queries for Dashboard Tiles

### 1. Hot-Path Performance & Concurrency

#### Service Method Latency (p50/p95/p99)
```dql
fetch metrics
| filter metric.name == "semlayer.access_evaluation.duration"
| fieldsAdd percentile(50) as p50, percentile(95) as p95, percentile(99) as p99
| fields timestamp, p50, p95, p99, cache_hit
| sort timestamp desc
```

#### Concurrent Evaluations
```dql
fetch metrics
| filter metric.name == "semlayer.access_evaluation.active"
| fields timestamp, value as active_evaluations
| sort timestamp desc
| limit 100
```

#### Cache Health
```dql
fetch metrics
| filter metric.name == "semlayer.cache.hit_ratio" or metric.name == "semlayer.cache.invalidations"
| fields timestamp, value, metric.name
| sort timestamp desc
| limit 200
```

### 2. Conversational Layer Load & Responsiveness

#### Turn Latency Distribution
```dql
fetch metrics
| filter metric.name == "semlayer.conversational.turn.duration"
| fieldsAdd percentile(50) as p50, percentile(95) as p95
| fields timestamp, p50, p95, conversation.length
| sort timestamp desc
```

#### Guardrail Intervention Rate
```dql
fetch metrics
| filter metric.name == "semlayer.guardrail.interventions" or metric.name == "semlayer.conversational.turns"
| fields timestamp, value, metric.name
| sort timestamp desc
| limit 200
```

#### Active Conversations
```dql
fetch metrics
| filter metric.name == "semlayer.conversations.active"
| fields timestamp, value as active_conversations
| sort timestamp desc
| limit 100
```

### 3. Governance KPIs

#### Manual Grant Rate (30-day trend)
```dql
fetch metrics
| filter metric.name == "semlayer.governance.manual_grants"
| summarize sum(value) by {bin(timestamp, 1d)}
| fields timestamp, sum as daily_grants
| sort timestamp desc
| limit 30
```

#### Drift Backlog
```dql
fetch metrics
| filter metric.name == "semlayer.governance.drift_backlog"
| fields timestamp, value as backlog_count
| sort timestamp desc
| limit 100
```

#### Conflict MTTR
```dql
fetch metrics
| filter metric.name == "semlayer.governance.conflict_resolution_time"
| fieldsAdd percentile(50) as median_resolution_time
| fields timestamp, median_resolution_time
| sort timestamp desc
```

### 4. Adoption & Change-Management Metrics

#### NL Query Success Rate
```dql
fetch metrics
| filter metric.name == "semlayer.conversational.success_rate"
| fields timestamp, value as success_rate
| sort timestamp desc
| limit 100
```

### 5. Alert & Incident Panel

#### Active Alerts
```dql
fetch events
| filter event.type == "semlayer.alert" and event.status == "ACTIVE"
| fields timestamp, event.name, event.severity, tenant_id
| sort timestamp desc
| limit 50
```

#### Recent Incidents
```dql
fetch logs
| filter content contains "semlayer" and (content contains "error" or content contains "incident" or content contains "breach")
| fields timestamp, content, loglevel, service.name
| sort timestamp desc
| limit 100
```

## 🎯 SLO Definitions

### Access Intelligence Latency SLO
```json
{
  "name": "Access Intelligence Latency SLO",
  "description": "p95 latency for access evaluation should be under 8ms",
  "metricExpression": "semlayer.access_evaluation.duration:percentile(95)",
  "target": 8,
  "warning": 15,
  "critical": 30,
  "timeframe": "-1h",
  "evaluationType": "AGGREGATE"
}
```

### Cache Hit Ratio SLO
```json
{
  "name": "Cache Hit Ratio SLO",
  "description": "Cache hit ratio should be above 90%",
  "metricExpression": "semlayer.cache.hit_ratio",
  "target": 90,
  "warning": 80,
  "critical": 70,
  "timeframe": "-1h",
  "evaluationType": "AGGREGATE"
}
```

### Conversational Success Rate SLO
```json
{
  "name": "Conversational Success Rate SLO",
  "description": "NL query success rate should be above 95%",
  "metricExpression": "semlayer.conversational.success_rate",
  "target": 95,
  "warning": 90,
  "critical": 85,
  "timeframe": "-1h",
  "evaluationType": "AGGREGATE"
}
```

## 🚨 Alert Rules

### High Latency Alert
```json
{
  "name": "High Access Evaluation Latency",
  "description": "Access evaluation latency is too high",
  "query": "semlayer.access_evaluation.duration:percentile(95) > 30",
  "severity": "CRITICAL",
  "tags": ["performance", "hot-path", "semlayer"],
  "threshold": 30,
  "timeframe": "-5m"
}
```

### Cache Performance Degradation
```json
{
  "name": "Cache Performance Degradation",
  "description": "Cache hit ratio has dropped significantly",
  "query": "semlayer.cache.hit_ratio < 70",
  "severity": "WARNING",
  "tags": ["cache", "performance", "semlayer"],
  "threshold": 70,
  "timeframe": "-10m"
}
```

### Governance Drift Backlog Alert
```json
{
  "name": "Governance Drift Backlog",
  "description": "Too many drifted claims awaiting action",
  "query": "semlayer.governance.drift_backlog > 100",
  "severity": "WARNING",
  "tags": ["governance", "drift", "semlayer"],
  "threshold": 100,
  "timeframe": "-1h"
}
```

### Conversational Error Spike
```json
{
  "name": "Conversational Error Spike",
  "description": "Conversational query success rate has dropped",
  "query": "semlayer.conversational.success_rate < 85",
  "severity": "CRITICAL",
  "tags": ["conversational", "adoption", "semlayer"],
  "threshold": 85,
  "timeframe": "-5m"
}
```

## 📊 Business Events

### Access Evaluation Event
```json
{
  "eventType": "semlayer.access_evaluation",
  "title": "Access Evaluation Completed",
  "description": "An access evaluation decision has been made",
  "properties": {
    "tenant_id": {
      "type": "string",
      "displayName": "Tenant ID"
    },
    "user_id": {
      "type": "string",
      "displayName": "User ID (Hashed)"
    },
    "asset_id": {
      "type": "string",
      "displayName": "Asset ID"
    },
    "decision": {
      "type": "string",
      "displayName": "Decision",
      "enum": ["allow", "deny", "partial"]
    },
    "cache_hit": {
      "type": "boolean",
      "displayName": "Cache Hit"
    },
    "duration_ms": {
      "type": "number",
      "displayName": "Duration (ms)"
    }
  }
}
```

### Governance Action Event
```json
{
  "eventType": "semlayer.governance.action",
  "title": "Governance Action Performed",
  "description": "A governance steward has performed an action",
  "properties": {
    "action_type": {
      "type": "string",
      "displayName": "Action Type",
      "enum": ["approve", "reject", "simulate", "grant", "revoke"]
    },
    "steward_id": {
      "type": "string",
      "displayName": "Steward ID (Hashed)"
    },
    "tenant_id": {
      "type": "string",
      "displayName": "Tenant ID"
    },
    "target_type": {
      "type": "string",
      "displayName": "Target Type",
      "enum": ["claim", "policy", "user", "asset"]
    },
    "target_id": {
      "type": "string",
      "displayName": "Target ID"
    },
    "duration_ms": {
      "type": "number",
      "displayName": "Duration (ms)"
    }
  }
}
```

### Conversational Query Event
```json
{
  "eventType": "semlayer.conversational.query",
  "title": "Conversational Query Processed",
  "description": "A conversational NL query has been processed",
  "properties": {
    "conversation_id": {
      "type": "string",
      "displayName": "Conversation ID"
    },
    "tenant_id": {
      "type": "string",
      "displayName": "Tenant ID"
    },
    "user_id": {
      "type": "string",
      "displayName": "User ID (Hashed)"
    },
    "query_length": {
      "type": "number",
      "displayName": "Query Length"
    },
    "turns_count": {
      "type": "number",
      "displayName": "Turns in Conversation"
    },
    "success": {
      "type": "boolean",
      "displayName": "Success"
    }
  }
}
```

## 🚀 Deployment Script

Create this script to deploy all monitoring configuration:

```bash
#!/bin/bash
# deploy-monitoring.sh

DYNATRACE_ENV="${DYNATRACE_ENV:-your-environment}"
API_TOKEN="${DYNATRACE_API_TOKEN:-your-token}"

echo "🚀 Deploying Semlayer Monitoring Configuration..."

# Deploy custom metrics
echo "📊 Creating custom metrics..."
curl -X POST "https://$DYNATRACE_ENV.live.dynatrace.com/api/v2/metrics/ingest" \
  -H "Authorization: Api-Token $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d @monitoring/custom-metrics.json

# Deploy SLOs
echo "🎯 Creating SLOs..."
curl -X POST "https://$DYNATRACE_ENV.live.dynatrace.com/api/v2/slo" \
  -H "Authorization: Api-Token $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d @monitoring/slos.json

# Deploy alert rules
echo "🚨 Creating alert rules..."
curl -X POST "https://$DYNATRACE_ENV.live.dynatrace.com/api/v2/settings/objects" \
  -H "Authorization: Api-Token $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d @monitoring/alerts.json

# Deploy business events
echo "📋 Creating business events..."
curl -X POST "https://$DYNATRACE_ENV.live.dynatrace.com/api/v2/events/ingest" \
  -H "Authorization: Api-Token $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d @monitoring/business-events.json

# Import dashboard
echo "📊 Importing dashboard..."
curl -X POST "https://$DYNATRACE_ENV.live.dynatrace.com/api/config/v1/dashboards" \
  -H "Authorization: Api-Token $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d @monitoring/dynatrace-dashboard.json

echo "✅ Monitoring configuration deployed successfully!"
```

## 🔧 Backend Integration

Update your DynatraceManager to send these metrics:

```go
// In your DynatraceManager
func (dm *DynatraceManager) SendCustomMetric(ctx context.Context, name string, value float64, dimensions map[string]string) {
    if !dm.enabled {
        return
    }

    // Send to Dynatrace Metrics API
    metric := map[string]interface{}{
        "metric": name,
        "value": value,
        "dimensions": dimensions,
        "timestamp": time.Now().UnixMilli(),
    }

    // Implementation depends on your Dynatrace setup
    dm.sendToMetricsAPI(metric)
}
```

This configuration makes your dashboard completely turnkey - your engineers can deploy it with a single script and have live observability in minutes! 🎉
