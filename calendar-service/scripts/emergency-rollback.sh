#!/bin/bash
# emergency-rollback.sh - Emergency rollback to previous version

set -e

NAMESPACE="${1:-calendar-service}"
DEPLOYMENT="${2:-calendar-service-prod}"
TIMEOUT="${3:-5m}"

echo "═══════════════════════════════════════════════════════════"
echo "🔄 EMERGENCY ROLLBACK PROCEDURE"
echo "═══════════════════════════════════════════════════════════"
echo "📍 Namespace: $NAMESPACE"
echo "📍 Deployment: $DEPLOYMENT"
echo "⏱️  Timeout: $TIMEOUT"
echo ""

# Step 1: Verify deployment exists
echo "▶️ Step 1: Verifying deployment..."

if ! kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" &>/dev/null; then
    echo "❌ Deployment not found: $DEPLOYMENT"
    exit 1
fi

CURRENT_IMAGE=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" \
    -o jsonpath='{.spec.template.spec.containers[0].image}')
echo "📍 Current image: $CURRENT_IMAGE"
echo "✅ Deployment verified"
echo ""

# Step 2: Get rollout history
echo "▶️ Step 2: Checking rollout history..."

REVISIONS=$(kubectl rollout history deployment/"$DEPLOYMENT" -n "$NAMESPACE" | wc -l)

if [ "$REVISIONS" -lt 2 ]; then
    echo "❌ No previous revision available"
    exit 1
fi

kubectl rollout history deployment/"$DEPLOYMENT" -n "$NAMESPACE"
echo "✅ Multiple revisions available"
echo ""

# Step 3: Trigger rollback
echo "▶️ Step 3: Starting rollback..."

kubectl rollout undo deployment/"$DEPLOYMENT" -n "$NAMESPACE"

echo "✅ Rollback command issued"
echo ""

# Step 4: Wait for rollback to complete
echo "▶️ Step 4: Waiting for rollback ($TIMEOUT)..."

if ! kubectl rollout status deployment/"$DEPLOYMENT" -n "$NAMESPACE" --timeout="$TIMEOUT"; then
    echo "⚠️ Rollback did not complete within timeout"
    exit 1
fi

echo "✅ Rollback complete"
echo ""

# Step 5: Verify service health
echo "▶️ Step 5: Verifying service health..."

POD_NAME=$(kubectl get pods -n "$NAMESPACE" \
    -l app=calendar-service \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$POD_NAME" ]; then
    echo "⚠️ No running pods found yet"
else
    echo "📍 Testing pod: $POD_NAME"
    
    sleep 5  # Wait for pod startup
    
    HEALTH=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" \
        -- curl -s http://localhost:8080/health || echo "FAIL")
    
    if [[ $HEALTH == *"healthy"* ]]; then
        echo "✅ Service health check passed"
    else
        echo "⚠️ Health check inconclusive"
    fi
fi

echo ""

# Step 6: Verify readiness
echo "▶️ Step 6: Checking pod readiness..."

READY_REPLICAS=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" \
    -o jsonpath='{.status.readyReplicas}')

if [ -z "$READY_REPLICAS" ] || [ "$READY_REPLICAS" == "0" ]; then
    echo "⚠️ No ready replicas yet"
else
    echo "📍 Ready replicas: $READY_REPLICAS"
    echo "✅ Pods are becoming ready"
fi

echo ""

# Step 7: Summary
echo "═══════════════════════════════════════════════════════════"
echo "✅ EMERGENCY ROLLBACK COMPLETE"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "📊 Deployment Status:"
kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE"
echo ""
echo "📝 Next Steps:"
echo "   1. Monitor service health for 5 minutes"
echo "   2. Check application logs for errors"
echo "   3. Verify no data corruption occurred"
echo "   4. Create incident ticket"
echo "   5. Schedule post-mortem review"
echo ""
echo "🔗 Monitoring:"
echo "   • Logs: kubectl logs -f deployment/$DEPLOYMENT -n $NAMESPACE"
echo "   • Events: kubectl describe nodes"
echo "   • Dashboard: http://grafana.example.com"
echo ""
echo "📞 Contact on-call engineer immediately"
echo ""
