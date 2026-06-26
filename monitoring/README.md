# Semlayer Dynatrace Monitoring - Complete As-Code Setup

This directory contains everything needed to deploy comprehensive observability for your governance-native semantic platform in Dynatrace.

## 📁 Files Overview

| File | Purpose |
|------|---------|
| `dynatrace-dashboard.json` | Main dashboard configuration |
| `dql-metrics-mapping.md` | Complete DQL queries and metric definitions |
| `custom-metrics.json` | Custom metric definitions |
| `slos.json` | SLO configurations |
| `alerts.json` | Alert rule definitions |
| `business-events.json` | Business event schemas |
| `deploy-monitoring.sh` | One-click deployment script |
| `import-dashboard.sh` | Dashboard-only import script |
| `dynatrace-dashboard-README.md` | Detailed documentation |

## 🚀 Quick Start

### 1. Set Environment Variables
```bash
export DYNATRACE_ENV="your-environment-id"
export DYNATRACE_API_TOKEN="your-api-token"
```

### 2. Deploy Everything
```bash
cd monitoring
./deploy-monitoring.sh
```

That's it! Your dashboard will be live in minutes.

## 🎯 What Gets Deployed

### Custom Metrics (11 total)
- **Performance**: Access evaluation latency, cache metrics, concurrency
- **Conversational**: Turn latency, guardrail interventions, active conversations
- **Governance**: Manual grants, drift backlog, conflict resolution time
- **Adoption**: Success rates, feature usage

### SLOs (3 total)
- Access Intelligence Latency (<8ms p95)
- Cache Hit Ratio (>90%)
- Conversational Success Rate (>95%)

### Alert Rules (4 total)
- High latency alerts
- Cache performance degradation
- Governance drift backlog
- Conversational error spikes

### Business Events (3 total)
- Access evaluation decisions
- Governance steward actions
- Conversational query processing

### Dashboard (1 complete)
- 12 tiles covering all aspects
- Smart layout for SRE and governance teams
- Real-time and historical views

## 🔧 Backend Integration Required

Your application needs to send metrics to Dynatrace. Update your `DynatraceManager`:

```go
// Send custom metrics
dm.SendCustomMetric(ctx, "semlayer.access_evaluation.duration", duration, map[string]string{
    "tenant_id": tenantID,
    "cache_hit": cacheHit,
    "decision": decision,
})

// Send business events
dm.SendBusinessEvent(ctx, "semlayer.access_evaluation", map[string]interface{}{
    "tenant_id": tenantID,
    "user_id": hashString(userID),
    "asset_id": hashString(assetID),
    "decision": response.Decision,
    "cache_hit": response.CacheHit,
    "duration_ms": duration.Milliseconds(),
})
```

## 📊 Dashboard Sections

### 1. Hot-Path Performance & Concurrency
- Service method latency (p50/p95/p99)
- Concurrent evaluations vs limits
- Cache health and invalidations
- SLO compliance monitoring

### 2. Conversational Layer Load & Responsiveness
- Turn latency by conversation length
- Guardrail intervention rates
- Active conversation tracking
- Success rate monitoring

### 3. Governance KPIs
- Manual grant rate trends
- Drift backlog monitoring
- Conflict mean time to resolution
- Policy simulation volume

### 4. Adoption & Change-Management Metrics
- NL query success rates
- Feature adoption tracking
- Pilot vs control cohort analysis
- Training impact measurement

### 5. Alert & Incident Panel
- Active alerts by severity
- Recent incidents with root cause
- SLO breach notifications
- Governance and performance alerts

## 🎨 Customization

### Adding New Metrics
1. Add to `custom-metrics.json`
2. Update dashboard tiles in `dynatrace-dashboard.json`
3. Add DQL queries to `dql-metrics-mapping.md`
4. Update backend to send the metric

### Modifying SLOs
1. Edit `slos.json`
2. Adjust targets based on baseline performance
3. Update alert thresholds accordingly

### Dashboard Layout
1. Edit `dynatrace-dashboard.json`
2. Adjust tile positions and sizes
3. Add new tiles for emerging KPIs

## 🔍 DQL Query Examples

### Performance Analysis
```dql
fetch metrics
| filter metric.name == "semlayer.access_evaluation.duration"
| fieldsAdd percentile(95) as p95_latency
| filter p95_latency > 10
| sort p95_latency desc
```

### Governance Insights
```dql
fetch events
| filter event.type == "semlayer.governance.action"
| summarize count() by {action_type, bin(timestamp, 1d)}
| sort timestamp desc
```

### Conversational Analytics
```dql
fetch metrics
| filter metric.name == "semlayer.conversational.success_rate"
| fields timestamp, value as success_rate, tenant_id
| filter success_rate < 95
| sort timestamp desc
```

## 🚨 Alert Configuration

Alerts are configured with smart thresholds:
- **Critical**: Immediate action required (latency >30ms, success <85%)
- **Warning**: Monitor closely (cache <70%, backlog >100)
- **Info**: Track for trends (performance degradation)

## 📈 Scaling Considerations

### Multi-Tenant Support
- All metrics include `tenant_id` dimension
- Dashboard supports tenant filtering
- Management zones can isolate tenant data

### High Volume
- Metrics are aggregated efficiently
- DQL queries are optimized for performance
- Alert rules use appropriate time windows

### Custom Dimensions
- Add business-specific dimensions as needed
- Update metric definitions accordingly
- Modify DQL queries to leverage new dimensions

## 🤝 Team Access

### Recommended Permissions
- **SRE Team**: Full edit access to monitoring config
- **Governance Stewards**: Dashboard view + alert access
- **Product Leads**: Adoption metrics view
- **Developers**: Debug access to traces and logs

### Sharing
- Dashboard is shared by default
- Use management zones for tenant isolation
- Set up automated exports for compliance

## 📞 Support

### Troubleshooting
1. **Metrics not appearing**: Check OpenTelemetry configuration
2. **Dashboard errors**: Verify entity selectors match your environment
3. **SLOs not calculating**: Ensure metric names match exactly
4. **Alerts not firing**: Check threshold values and time windows

### Getting Help
- Review `dql-metrics-mapping.md` for query examples
- Check Dynatrace logs for ingestion errors
- Verify backend tracing implementation
- Contact platform SRE team

---

## 🎉 Ready to Deploy!

Your governance platform now has enterprise-grade observability. Run the deployment script and you'll have:

- ✅ **Real-time performance monitoring**
- ✅ **Governance health visibility**
- ✅ **Conversational adoption tracking**
- ✅ **Automated alerting and SLOs**
- ✅ **Business event correlation**
- ✅ **Multi-dimensional analysis**

**One command, fully observable!** 🚀

```bash
./monitoring/deploy-monitoring.sh
```
