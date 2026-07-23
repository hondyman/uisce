#!/bin/bash
# Phase 3.24 Grafana Dashboards Setup Script
# Deploy multi-region dashboards to Grafana

# Global Multi-Region Dashboard (Dashboard ID: 3001)
cat > /tmp/semlayer-global-overview.json << 'EOF'
{
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": "-- Grafana --",
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "type": "dashboard"
      }
    ]
  },
  "editable": true,
  "gnetId": null,
  "graphTooltip": 0,
  "id": 3001,
  "links": [
    {"asDropdown": true, "icon": "external link", "includeVars": true, "tags": ["semlayer", "regions"], "title": "Region Dashboards", "type": "dashlist"},
    {"asDropdown": false, "icon": "doc", "title": "Phase 3.24 Architecture", "url": "/docs/phase-3.24-architecture.md", "type": "link"}
  ],
  "panels": [
    {
      "title": "Global Region Health",
      "type": "stat",
      "gridPos": {"h": 8, "w": 24, "x": 0, "y": 0},
      "targets": [
        {
          "expr": "count(up{job=~\"semlayer-.*\",region=~\"$region\"}) by (region)",
          "legendFormat": "{{region}} - Healthy Services",
          "refId": "A"
        }
      ],
      "thresholds": {"mode": "absolute", "steps": [{"color": "red", "value": null}, {"color": "yellow", "value": 4}, {"color": "green", "value": 6}]},
      "unit": "short"
    },
    {
      "title": "Materialization Latency by Region",
      "type": "graph",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 8},
      "targets": [
        {
          "expr": "histogram_quantile(0.99, rate(semlayer_materialization_latency_seconds_bucket{region=~\"$region\"}[5m]))",
          "legendFormat": "{{region}} - P99",
          "refId": "A"
        },
        {
          "expr": "histogram_quantile(0.95, rate(semlayer_materialization_latency_seconds_bucket{region=~\"$region\"}[5m]))",
          "legendFormat": "{{region}} - P95",
          "refId": "B"
        }
      ],
      "yaxes": [{"format": "s", "label": "Latency"}]
    },
    {
      "title": "Drift Detection - Features with Drift",
      "type": "stat",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 8},
      "targets": [
        {
          "expr": "count(semlayer_drift_score{region=~\"$region\",score > 0.05}) by (region)",
          "legendFormat": "{{region}} - Drifted",
          "refId": "A"
        }
      ],
      "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": null}, {"color": "yellow", "value": 5}, {"color": "red", "value": 20}]}
    },
    {
      "title": "API Gateway Requests (Global)",
      "type": "graph",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 16},
      "targets": [
        {
          "expr": "sum(rate(semlayer_api_requests_total{region=~\"$region\"}[1m])) by (region)",
          "legendFormat": "{{region}} - RPS",
          "refId": "A"
        }
      ],
      "yaxes": [{"format": "short", "label": "Requests/sec"}]
    },
    {
      "title": "Feature Discovery - Candidates by Region",
      "type": "piechart",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 16},
      "targets": [
        {
          "expr": "sum(semlayer_discovery_candidates{status=\"approved\",region=~\"$region\"}) by (region)",
          "legendFormat": "{{region}}",
          "refId": "A"
        }
      ]
    },
    {
      "title": "Region Status Matrix",
      "type": "table",
      "gridPos": {"h": 8, "w": 24, "x": 0, "y": 24},
      "targets": [
        {
          "expr": "max(semlayer_region_health_percentage{region=~\"$region\"}) by (region)",
          "format": "table",
          "instant": true,
          "legendFormat": "Health %",
          "refId": "A"
        }
      ]
    }
  ],
  "refresh": "30s",
  "schemaVersion": 35,
  "style": "dark",
  "tags": ["semlayer", "global", "multi-region"],
  "templating": {
    "list": [
      {
        "allValue": ".*",
        "current": {"selected": true, "text": "All", "value": "$__all"},
        "datasource": "Prometheus",
        "definition": "label_values(up{job=~\"semlayer-.*\"}, region)",
        "description": null,
        "error": null,
        "hide": 0,
        "includeAll": true,
        "label": "Region",
        "multi": true,
        "name": "region",
        "options": [],
        "query": {"query": "label_values(up{job=~\"semlayer-.*\"}, region)", "refId": "Prometheus-region-Variable-Query"},
        "refresh": "on time range change",
        "sort": 0,
        "tagValuesQuery": "",
        "tagsQuery": "",
        "type": "query",
        "useTags": false
      }
    ]
  },
  "time": {"from": "now-6h", "to": "now"},
  "timepicker": {},
  "timezone": "UTC",
  "title": "Phase 3.24 Global Multi-Region Overview",
  "uid": "semlayer-global-overview",
  "version": 1
}
EOF

# Region-Specific Dashboard (Template for us-east, eu-west, apac)
cat > /tmp/semlayer-region-detailed.json << 'EOF'
{
  "title": "Phase 3.24 Region Detail - {{REGION_NAME}}",
  "uid": "semlayer-region-{{REGION_CODE}}",
  "tags": ["semlayer", "region", "{{REGION_CODE}}"],
  "timezone": "UTC",
  "schemaVersion": 35,
  "version": 1,
  "refresh": "30s",
  "editable": true,
  "panels": [
    {
      "title": "Materialization Service - Metrics",
      "type": "graph",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 0},
      "targets": [
        {
          "expr": "histogram_quantile(0.99, rate(semlayer_materialization_latency_seconds_bucket{region=\"{{REGION_CODE}}\"}[5m]))",
          "legendFormat": "P99 Latency",
          "refId": "A"
        },
        {
          "expr": "rate(semlayer_materialization_errors_total{region=\"{{REGION_CODE}}\"}[5m])",
          "legendFormat": "Errors/sec",
          "refId": "B"
        }
      ],
      "yaxes": [
        {"format": "s", "label": "Latency"},
        {"format": "short", "label": "Errors"}
      ]
    },
    {
      "title": "Materialization Success Rate",
      "type": "gauge",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 0},
      "targets": [
        {
          "expr": "100 * (1 - rate(semlayer_materialization_errors_total{region=\"{{REGION_CODE}}\"}[5m]) / rate(semlayer_materialization_total{region=\"{{REGION_CODE}}\"}[5m]))",
          "legendFormat": "Success %",
          "refId": "A"
        }
      ],
      "thresholds": {"mode": "absolute", "steps": [{"color": "red", "value": null}, {"color": "yellow", "value": 95}, {"color": "green", "value": 99}]}
    },
    {
      "title": "Drift Detection - Score Distribution",
      "type": "histogram",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 8},
      "targets": [
        {
          "expr": "histogram_quantile(0.50, semlayer_drift_score{region=\"{{REGION_CODE}}\"})",
          "legendFormat": "Median",
          "refId": "A"
        },
        {
          "expr": "histogram_quantile(0.95, semlayer_drift_score{region=\"{{REGION_CODE}}\"})",
          "legendFormat": "P95",
          "refId": "B"
        }
      ]
    },
    {
      "title": "Drift Detection - Methods Performance",
      "type": "table",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 8},
      "targets": [
        {
          "expr": "avg(rate(semlayer_drift_duration_seconds{region=\"{{REGION_CODE}}\"}[5m])) by (method)",
          "format": "table",
          "instant": true,
          "refId": "A"
        }
      ]
    },
    {
      "title": "Feature Discovery - Approval Rate",
      "type": "stat",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 16},
      "targets": [
        {
          "expr": "100 * (semlayer_discovery_candidates{status=\"approved\",region=\"{{REGION_CODE}}\",location=\"final\"} / semlayer_discovery_candidates{status=~\"approved|rejected\",region=\"{{REGION_CODE}}\",location=\"final\"})",
          "legendFormat": "Approval %",
          "refId": "A"
        }
      ],
      "thresholds": {"mode": "absolute", "steps": [{"color": "red", "value": null}, {"color": "yellow", "value": 30}, {"color": "green", "value": 50}]}
    },
    {
      "title": "Time-Series Anomalies",
      "type": "stat",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 16},
      "targets": [
        {
          "expr": "count(semlayer_ts_anomaly{region=\"{{REGION_CODE}}\",anomaly=\"true\"})",
          "legendFormat": "Active Anomalies",
          "refId": "A"
        }
      ]
    },
    {
      "title": "API Gateway - Request Rate",
      "type": "graph",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 24},
      "targets": [
        {
          "expr": "sum(rate(semlayer_api_requests_total{region=\"{{REGION_CODE}}\"}[1m])) by (method)",
          "legendFormat": "{{method}}",
          "refId": "A"
        }
      ],
      "yaxes": [{"format": "short", "label": "RPS"}]
    },
    {
      "title": "API Gateway - Error Rate",
      "type": "graph",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 24},
      "targets": [
        {
          "expr": "sum(rate(semlayer_api_errors_total{region=\"{{REGION_CODE}}\"}[1m])) by (status_code)",
          "legendFormat": "{{status_code}}",
          "refId": "A"
        }
      ]
    },
    {
      "title": "Temporal Worker Tasks",
      "type": "graph",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 32},
      "targets": [
        {
          "expr": "sum(rate(temporal_workflow_execute_total{region=\"{{REGION_CODE}}\"}[1m])) by (workflow_type)",
          "legendFormat": "{{workflow_type}}",
          "refId": "A"
        }
      ]
    },
    {
      "title": "Region Resource Utilization",
      "type": "gauge",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 32},
      "targets": [
        {
          "expr": "avg(rate(container_cpu_usage_seconds_total{pod=~\"semlayer-.*\",region=\"{{REGION_CODE}}\"}[5m]))",
          "legendFormat": "CPU Avg",
          "refId": "A"
        }
      ]
    }
  ]
}
EOF

# SLO Dashboard
cat > /tmp/semlayer-slo-dashboard.json << 'EOF'
{
  "title": "Phase 3.24 SLO Status",
  "uid": "semlayer-slos",
  "tags": ["semlayer", "slo", "compliance"],
  "timezone": "UTC",
  "schemaVersion": 35,
  "refresh": "60s",
  "panels": [
    {
      "title": "Materialization Latency SLO (Target: <5s P99)",
      "type": "stat",
      "gridPos": {"h": 6, "w": 6, "x": 0, "y": 0},
      "targets": [
        {
          "expr": "histogram_quantile(0.99, rate(semlayer_materialization_latency_seconds_bucket[30m])) < 5",
          "legendFormat": "SLO Met",
          "refId": "A"
        }
      ],
      "thresholds": {"mode": "boolean", "steps": [{"color": "red", "value": 0}, {"color": "green", "value": 1}]},
      "colorMode": "background"
    },
    {
      "title": "Drift Detection Latency SLO (Target: <3s P99)",
      "type": "stat",
      "gridPos": {"h": 6, "w": 6, "x": 6, "y": 0},
      "targets": [
        {
          "expr": "histogram_quantile(0.99, rate(semlayer_drift_duration_seconds_bucket[30m])) < 3",
          "legendFormat": "SLO Met",
          "refId": "A"
        }
      ],
      "thresholds": {"mode": "boolean", "steps": [{"color": "red", "value": 0}, {"color": "green", "value": 1}]},
      "colorMode": "background"
    },
    {
      "title": "API Latency SLO (Target: <2s P99)",
      "type": "stat",
      "gridPos": {"h": 6, "w": 6, "x": 12, "y": 0},
      "targets": [
        {
          "expr": "histogram_quantile(0.99, rate(semlayer_api_latency_seconds_bucket[30m])) < 2",
          "legendFormat": "SLO Met",
          "refId": "A"
        }
      ],
      "thresholds": {"mode": "boolean", "steps": [{"color": "red", "value": 0}, {"color": "green", "value": 1}]},
      "colorMode": "background"
    },
    {
      "title": "Service Health SLO (Target: >99%)",
      "type": "stat",
      "gridPos": {"h": 6, "w": 6, "x": 18, "y": 0},
      "targets": [
        {
          "expr": "100 * (1 - (rate(semlayer_errors_total[30m]) / rate(semlayer_requests_total[30m]))) > 99",
          "legendFormat": "SLO Met",
          "refId": "A"
        }
      ],
      "thresholds": {"mode": "boolean", "steps": [{"color": "red", "value": 0}, {"color": "green", "value": 1}]},
      "colorMode": "background"
    },
    {
      "title": "SLO Compliance Report (30 days)",
      "type": "table",
      "gridPos": {"h": 10, "w": 24, "x": 0, "y": 6},
      "targets": [
        {
          "expr": "max by (slo_name) (slo_compliance_percentage)",
          "format": "table",
          "instant": true,
          "refId": "A"
        }
      ]
    }
  ]
}
EOF

# Deploy dashboards to Grafana
echo "Deploying Grafana dashboards..."

# Global overview dashboard
curl -X POST http://grafana.semlayer.internal:3000/api/dashboards/db \
  -H "Authorization: Bearer ${GRAFANA_API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "dashboard": '"$(cat /tmp/semlayer-global-overview.json)"'
  }'

# Deploy region-specific dashboards
for REGION_CODE in us-east eu-west apac; do
  REGION_NAME=$(echo "$REGION_CODE" | sed 's/-/ /g; s/\b./\u&/g')
  
  sed -e "s/{{REGION_CODE}}/$REGION_CODE/g" \
      -e "s/{{REGION_NAME}}/$REGION_NAME/g" \
      /tmp/semlayer-region-detailed.json > /tmp/semlayer-region-${REGION_CODE}.json
  
  curl -X POST http://grafana.semlayer.internal:3000/api/dashboards/db \
    -H "Authorization: Bearer ${GRAFANA_API_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
      "dashboard": '"$(cat /tmp/semlayer-region-${REGION_CODE}.json)"'
    }'
  
  echo "Deployed dashboard for region: $REGION_CODE"
done

# Deploy SLO dashboard
curl -X POST http://grafana.semlayer.internal:3000/api/dashboards/db \
  -H "Authorization: Bearer ${GRAFANA_API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "dashboard": '"$(cat /tmp/semlayer-slo-dashboard.json)"'
  }'

echo "All Grafana dashboards deployed successfully!"
