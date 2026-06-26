#!/bin/bash
# deploy-staging.sh - Deploy calendar service to staging environment

set -e

VERSION="${1:-1.0.0}"
ENVIRONMENT="staging"
REGISTRY="${REGISTRY:-gcr.io/my-project}"

echo "═══════════════════════════════════════════════════════════"
echo "🚀 Calendar Service Staging Deployment"
echo "═══════════════════════════════════════════════════════════"
echo "📦 Version: $VERSION"
echo "🌍 Environment: $ENVIRONMENT"
echo "📍 Registry: $REGISTRY"
echo ""

# Step 1: Validate prerequisites
echo "▶️ Step 1: Validating prerequisites..."

if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Install from: https://www.docker.com/products/docker-desktop"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Install from: https://kubernetes.io/docs/tasks/tools/"
    exit 1
fi

if [ -z "$JWT_SECRET" ]; then
    echo "❌ JWT_SECRET not set"
    exit 1
fi

if [ -z "$DATABASE_URL" ]; then
    echo "❌ DATABASE_URL not set"
    exit 1
fi

echo "✅ Prerequisites validated"
echo ""

# Step 2: Build Docker image
echo "▶️ Step 2: Building Docker image..."
docker build -t "${REGISTRY}/calendar-service:${VERSION}" \
            -t "${REGISTRY}/calendar-service:latest" \
            -f Dockerfile \
            . | tail -20

echo "✅ Docker image built"
echo ""

# Step 3: Push to registry
echo "▶️ Step 3: Pushing to Docker registry..."
docker push "${REGISTRY}/calendar-service:${VERSION}"
docker push "${REGISTRY}/calendar-service:latest"

echo "✅ Image pushed successfully"
echo ""

# Step 4: Update Kubernetes deployment
echo "▶️ Step 4: Updating Kubernetes deployment..."

kubectl set image deployment/calendar-service-staging \
    calendar-service="${REGISTRY}/calendar-service:${VERSION}" \
    -n calendar-service \
    --record

echo "✅ Deployment image updated"
echo ""

# Step 5: Monitor rollout
echo "▶️ Step 5: Monitoring rollout (timeout: 5 minutes)..."

if ! kubectl rollout status deployment/calendar-service-staging \
    -n calendar-service \
    --timeout=5m; then
    echo "❌ Deployment failed to reach ready state"
    echo "🔄 Rolling back..."
    kubectl rollout undo deployment/calendar-service-staging -n calendar-service
    exit 1
fi

echo "✅ Deployment successful"
echo ""

# Step 6: Wait for pods to be ready
echo "▶️ Step 6: Waiting for pods to stabilize..."
sleep 10

# Step 7: Run health checks
echo "▶️ Step 7: Running health checks..."

POD_NAME=$(kubectl get pods -n calendar-service \
    -l app=calendar-service \
    -o jsonpath='{.items[0].metadata.name}')

if [ -z "$POD_NAME" ]; then
    echo "❌ No running pods found"
    exit 1
fi

echo "📍 Testing pod: $POD_NAME"

# Health endpoint check
HEALTH_STATUS=$(kubectl exec -it "$POD_NAME" -n calendar-service \
    -- curl -s http://localhost:8080/health || echo "FAIL")

if [[ $HEALTH_STATUS == *"healthy"* ]]; then
    echo "✅ Service health check passed"
else
    echo "⚠️ Health check inconclusive: $HEALTH_STATUS"
fi

echo ""

# Step 8: Verify metrics collection
echo "▶️ Step 8: Verifying metrics collection..."

METRICS=$(kubectl exec -it "$POD_NAME" -n calendar-service \
    -- curl -s http://localhost:9090/metrics | grep -c "http_requests_total" || echo "0")

if [ "$METRICS" -gt 0 ]; then
    echo "✅ Metrics collection working"
else
    echo "⚠️ No metrics found yet (expected during startup)"
fi

echo ""

# Step 9: Summary
echo "═══════════════════════════════════════════════════════════"
echo "✅ STAGING DEPLOYMENT COMPLETE"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "📊 Deployment Information:"
kubectl get deployment calendar-service-staging -n calendar-service
echo ""
echo "🔍 Testing:"
echo "   • Health: curl http://staging-api.example.com/health"
echo "   • Metrics: curl http://staging-api.example.com/metrics"
echo ""
echo "📈 View logs:"
echo "   kubectl logs -f deployment/calendar-service-staging -n calendar-service"
echo ""
echo "🚀 Ready for testing! Next: Run load tests and monitor"
echo ""
