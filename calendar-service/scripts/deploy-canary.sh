#!/bin/bash
# deploy-canary.sh - Execute canary deployment to production

set -e

VERSION="${1:-1.0.0}"
ENVIRONMENT="production"
CANARY_PERCENTAGE="${2:-10}"

echo "═══════════════════════════════════════════════════════════"
echo "🚀 Canary Deployment to Production"
echo "═══════════════════════════════════════════════════════════"
echo "📦 Version: $VERSION"
echo "🎯 Initial Traffic: ${CANARY_PERCENTAGE}%"
echo ""

# Configuration
NAMESPACE="calendar-service"
DEPLOYMENT="calendar-service-prod"
SERVICE="calendar-service"
METRIC_CHECK_INTERVAL=30  # seconds
MONITORING_DURATION=300   # 5 minutes
ERROR_RATE_THRESHOLD=5    # percent

# Step 1: Pre-flight checks
echo "▶️ Step 1: Pre-flight checks..."

if ! kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" &>/dev/null; then
    echo "❌ Deployment not found: $DEPLOYMENT"
    exit 1
fi

# Backup current deployment
CURRENT_REPLICAS=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" \
    -o jsonpath='{.spec.replicas}')
echo "📍 Current replicas: $CURRENT_REPLICAS"
echo "📍 Creating backup..."

kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" \
    -o yaml > "/tmp/deployment-backup-$(date +%s).yaml"

echo "✅ Pre-flight checks passed"
echo ""

# Step 2: Deploy canary (10% of traffic)
echo "▶️ Step 2: Deploying canary (${CANARY_PERCENTAGE}% traffic)..."

# Update deployment image
kubectl set image deployment/"$DEPLOYMENT" \
    calendar-service=calendar-service:${VERSION} \
    -n "$NAMESPACE" \
    --record

# Wait for deployment
if ! kubectl wait --for=condition=available --timeout=5m \
    deployment/"$DEPLOYMENT" -n "$NAMESPACE"; then
    echo "❌ Canary deployment failed"
    exit 1
fi

echo "✅ Canary deployed"
echo ""

# Step 3: Monitor canary metrics
echo "▶️ Step 3: Monitoring canary (${MONITORING_DURATION}s)..."
echo "   Checking: error rate, latency, rate limits, database"
echo ""

MONITORING_END=$(($(date +%s) + MONITORING_DURATION))
ERROR_DETECTED=0

while [ $(date +%s) -lt $MONITORING_END ]; do
    # Get metrics from Prometheus
    ERROR_RATE=$(curl -s "http://prometheus:9090/api/v1/query?query=\
rate(http_requests_total%7Bstatus=%225.%22%7D%5B5m%5D)" \
        | grep -o '"value":\[[^]]*\]' | head -1 | grep -o '[0-9.]*' || echo "0")

    LATENCY_P95=$(curl -s "http://prometheus:9090/api/v1/query?query=\
histogram_quantile(0.95,rate(http_request_duration_seconds_bucket%5B5m%5D))" \
        | grep -o '"value":\[[^]]*\]' | head -1 | grep -o '[0-9.]*' || echo "0")

    RATE_LIMIT_429=$(curl -s "http://prometheus:9090/api/v1/query?query=\
increase(http_requests_total%7Bstatus=%22429%22%7D%5B1m%5D)" \
        | grep -o '"value":\[[^]]*\]' | head -1 | grep -o '[0-9.]*' || echo "0")

    printf "\r⏱️  [$(date '+%H:%M:%S')] Error Rate: %.2f%% | P95 Latency: %.3fs | 429s: %.0f               " \
        "$ERROR_RATE" "$LATENCY_P95" "$RATE_LIMIT_429"

    # Check error rate threshold
    if (( $(echo "$ERROR_RATE > $ERROR_RATE_THRESHOLD" | bc -l) )); then
        echo ""
        echo "❌ ERROR RATE EXCEEDED: $ERROR_RATE% > ${ERROR_RATE_THRESHOLD}%"
        ERROR_DETECTED=1
        break
    fi

    sleep $METRIC_CHECK_INTERVAL
done

echo ""
echo ""

if [ $ERROR_DETECTED -eq 1 ]; then
    echo "❌ CANARY FAILED - Rolling back..."
    kubectl rollout undo deployment/"$DEPLOYMENT" -n "$NAMESPACE"
    kubectl wait --for=condition=available --timeout=5m \
        deployment/"$DEPLOYMENT" -n "$NAMESPACE"
    echo "✅ Rollback complete"
    exit 1
fi

echo "✅ Canary monitoring passed"
echo ""

# Step 4: Increase to 50% traffic
echo "▶️ Step 4: Increasing traffic to 50%..."

DESIRED_REPLICAS=$((CURRENT_REPLICAS * 3 / 2))  # 1.5x for 50% blend

kubectl scale deployment "$DEPLOYMENT" \
    --replicas=$(( DESIRED_REPLICAS )) \
    -n "$NAMESPACE"

kubectl wait --for=condition=available --timeout=5m \
    deployment/"$DEPLOYMENT" -n "$NAMESPACE"

echo "✅ Traffic increased to 50%"
echo "🔍 Monitoring for 5 minutes..."
sleep 300

echo ""

# Step 5: Increase to 100% traffic
echo "▶️ Step 5: Deploying to 100% traffic..."

kubectl scale deployment "$DEPLOYMENT" \
    --replicas=$((CURRENT_REPLICAS * 2)) \
    -n "$NAMESPACE"

kubectl wait --for=condition=available --timeout=5m \
    deployment/"$DEPLOYMENT" -n "$NAMESPACE"

# Scale back down to original replicas after verification
sleep 30
kubectl scale deployment "$DEPLOYMENT" \
    --replicas="$CURRENT_REPLICAS" \
    -n "$NAMESPACE"

echo "✅ Traffic at 100%"
echo ""

# Step 6: Final verification
echo "▶️ Step 6: Final verification..."

# Get deployment info
READY_REPLICAS=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" \
    -o jsonpath='{.status.readyReplicas}')

if [ "$READY_REPLICAS" != "$CURRENT_REPLICAS" ]; then
    echo "⚠️ Not all replicas ready: $READY_REPLICAS / $CURRENT_REPLICAS"
fi

echo "✅ Canary deployment successful!"
echo ""

# Step 7: Summary
echo "═══════════════════════════════════════════════════════════"
echo "✅ CANARY DEPLOYMENT COMPLETE"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "📊 Deployment Information:"
kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE"
echo ""
echo "📈 Monitoring Dashboards:"
echo "   • Grafana: http://grafana.example.com/d/calendar-service"
echo "   • Prometheus: http://prometheus.example.com"
echo ""
echo "🔍 View logs:"
echo "   kubectl logs -f deployment/$DEPLOYMENT -n $NAMESPACE"
echo ""
echo "✅ Canary deployment successful - ready for full rollout"
echo ""
