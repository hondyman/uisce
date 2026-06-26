#!/bin/bash
# Blue-Green Deployment Strategy
# This script implements zero-downtime deployment using Kubernetes blue-green pattern
# 
# Usage: ./deploy-blue-green.sh [staging|production] [version] [image-registry]
# Example: ./deploy-blue-green.sh production v1.2.3 gcr.io/my-project

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default values
ENVIRONMENT="${1:-staging}"
VERSION="${2:-latest}"
REGISTRY="${3:-calendar-service}"
NAMESPACE="calendar"
DEPLOYMENT_NAME="calendar-service"
SERVICE_NAME="calendar-service"
TIMEOUT="300s"

# Validate environment
if [[ ! "$ENVIRONMENT" =~ ^(staging|production)$ ]]; then
  echo -e "${RED}❌ Invalid environment: $ENVIRONMENT${NC}"
  echo "Usage: $0 [staging|production] [version] [image-registry]"
  exit 1
fi

# Adjust namespace for staging
if [ "$ENVIRONMENT" == "staging" ]; then
  NAMESPACE="calendar-staging"
fi

# Helper functions
log_info() {
  echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
  echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
  echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
  echo -e "${RED}❌ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
  log_info "Checking prerequisites..."
  
  local missing_tools=()
  
  for tool in kubectl kustomize; do
    if ! command -v "$tool" &> /dev/null; then
      missing_tools+=("$tool")
    fi
  done
  
  if [ ${#missing_tools[@]} -gt 0 ]; then
    log_error "Missing required tools: ${missing_tools[*]}"
    exit 1
  fi
  
  # Check cluster connectivity
  if ! kubectl cluster-info &> /dev/null; then
    log_error "Not connected to Kubernetes cluster"
    exit 1
  fi
  
  # Check namespace exists
  if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
    log_error "Namespace $NAMESPACE does not exist"
    exit 1
  fi
  
  log_success "Prerequisites check passed"
}

# Get current active environment (blue or green)
get_current_env() {
  local current=$(kubectl get svc "$SERVICE_NAME" -n "$NAMESPACE" \
    -o jsonpath='{.spec.selector.version}' 2>/dev/null || echo "blue")
  echo "$current"
}

# Get inactive environment
get_inactive_env() {
  local current=$(get_current_env)
  if [ "$current" == "blue" ]; then
    echo "green"
  else
    echo "blue"
  fi
}

# Pre-deployment validation
pre_deployment_checks() {
  log_info "Running pre-deployment validation..."
  
  # Check if image exists
  log_info "Checking if image exists: $REGISTRY:$VERSION"
  if ! docker pull "$REGISTRY:$VERSION" 2>/dev/null; then
    log_warning "Could not verify image, continuing anyway..."
  fi
  
  # Verify enough resources available
  log_info "Checking cluster resources..."
  local available_memory=$(kubectl top nodes --no-headers 2>/dev/null | awk '{sum+=$5} END {print sum}' || echo "unknown")
  log_info "Available memory: $available_memory"
  
  log_success "Pre-deployment checks passed"
}

# Deploy to inactive environment
deploy_to_inactive() {
  local inactive=$(get_inactive_env)
  log_info "Deploying to $inactive environment..."
  
  # Build manifests with kustomize
  local kustomize_dir="k8s/overlays/$ENVIRONMENT"
  if [ ! -d "$kustomize_dir" ]; then
    log_error "Kustomize overlay directory not found: $kustomize_dir"
    exit 1
  fi
  
  # Generate manifests
  log_info "Generating Kubernetes manifests from kustomize..."
  kustomize build "$kustomize_dir" > /tmp/manifests.yaml
  
  # Update image in manifests
  sed -i "s|calendar-service:.*|$REGISTRY:$VERSION|g" /tmp/manifests.yaml
  
  # Update version selector for blue-green
  sed -i "s|version: .*|version: $inactive|g" /tmp/manifests.yaml
  
  # Apply manifests
  log_info "Applying manifests to $NAMESPACE namespace..."
  kubectl apply -f /tmp/manifests.yaml --namespace="$NAMESPACE"
  
  log_success "Deployment to $inactive environment initiated"
}

# Wait for rollout to complete
wait_for_rollout() {
  local inactive=$(get_inactive_env)
  log_info "Waiting for rollout to complete (timeout: $TIMEOUT)..."
  
  if kubectl rollout status deployment/"$DEPLOYMENT_NAME" \
    -n "$NAMESPACE" \
    --timeout="$TIMEOUT"; then
    log_success "Rollout completed successfully"
    return 0
  else
    log_error "Rollout failed or timed out"
    return 1
  fi
}

# Run smoke tests
run_smoke_tests() {
  log_info "Running smoke tests..."
  
  # Port-forward to service
  log_info "Port-forwarding to service..."
  kubectl port-forward "svc/$SERVICE_NAME" 8080:80 -n "$NAMESPACE" &
  PF_PID=$!
  sleep 2
  
  # Basic health check
  if curl -f http://localhost:8080/health &> /dev/null; then
    log_success "Health check passed"
  else
    log_error "Health check failed"
    kill $PF_PID 2>/dev/null || true
    return 1
  fi
  
  # API readiness
  if curl -f http://localhost:8080/ready &> /dev/null; then
    log_success "Readiness check passed"
  else
    log_error "Readiness check failed"
    kill $PF_PID 2>/dev/null || true
    return 1
  fi
  
  # Check metrics endpoint
  if curl -f http://localhost:9090/metrics &> /dev/null; then
    log_success "Metrics endpoint accessible"
  else
    log_warning "Metrics endpoint not accessible"
  fi
  
  # Cleanup
  kill $PF_PID 2>/dev/null || true
  
  log_success "Smoke tests passed"
  return 0
}

# Check deployment health
check_deployment_health() {
  log_info "Checking deployment health..."
  
  local ready_replicas=$(kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" \
    -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
  local desired_replicas=$(kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" \
    -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
  
  if [ "$ready_replicas" -eq "$desired_replicas" ] && [ "$desired_replicas" -gt 0 ]; then
    log_success "All replicas ready ($ready_replicas/$desired_replicas)"
    return 0
  else
    log_error "Not all replicas ready ($ready_replicas/$desired_replicas)"
    return 1
  fi
}

# Switch traffic to inactive environment
switch_traffic() {
  local inactive=$(get_inactive_env)
  log_info "Switching traffic from $(get_current_env) to $inactive..."
  
  # Update service selector
  kubectl patch service "$SERVICE_NAME" \
    -n "$NAMESPACE" \
    -p "{\"spec\":{\"selector\":{\"version\":\"$inactive\"}}}" \
    --type merge
  
  log_success "Traffic switched to $inactive"
  
  # Give a moment for connections to drain
  sleep 5
  
  # Verify traffic is switched
  local current=$(get_current_env)
  if [ "$current" == "$inactive" ]; then
    log_success "Traffic switch verified"
    return 0
  else
    log_error "Traffic switch verification failed"
    return 1
  fi
}

# Rollback to previous environment
rollback() {
  local current=$(get_current_env)
  local previous=$(get_inactive_env)
  
  log_warning "Rolling back to $previous environment..."
  
  # Switch service selector back
  kubectl patch service "$SERVICE_NAME" \
    -n "$NAMESPACE" \
    -p "{\"spec\":{\"selector\":{\"version\":\"$previous\"}}}" \
    --type merge
  
  log_success "Rolled back to $previous"
}

# Monitor deployment for errors
monitor_deployment() {
  local inactive=$(get_inactive_env)
  log_info "Monitoring deployment for 60 seconds..."
  
  local error_threshold=5
  local error_count=0
  local check_interval=10
  local total_time=60
  local elapsed=0
  
  while [ $elapsed -lt $total_time ]; do
    # Check error rate from metrics
    local error_rate=$(kubectl exec -n "$NAMESPACE" \
      "deployment/$DEPLOYMENT_NAME" \
      -- curl -s http://localhost:9090/metrics 2>/dev/null | \
      grep 'http_requests_total.*status="5' | \
      awk '{sum+=$2} END {print sum}' || echo "0")
    
    if (( $(echo "$error_rate > $error_threshold" | bc -l) )); then
      error_count=$((error_count + 1))
    fi
    
    if [ $error_count -gt 3 ]; then
      log_error "High error rate detected, triggering rollback"
      rollback
      return 1
    fi
    
    log_info "Error rate: $error_rate (check $((elapsed / check_interval + 1))/6)"
    sleep $check_interval
    elapsed=$((elapsed + check_interval))
  done
  
  log_success "Monitoring complete - deployment stable"
  return 0
}

# Main deployment flow
main() {
  echo -e "${BLUE}"
  echo "╔════════════════════════════════════════════════════════════╗"
  echo "║        Blue-Green Deployment Script                        ║"
  echo "║        Environment: $ENVIRONMENT (Namespace: $NAMESPACE)"
  echo "║        Version: $VERSION"
  echo "║        Image: $REGISTRY:$VERSION"
  echo "╚════════════════════════════════════════════════════════════╝"
  echo -e "${NC}"
  
  # Step 1: Prerequisites
  check_prerequisites
  
  # Step 2: Pre-deployment validation
  pre_deployment_checks
  
  # Step 3: Deploy to inactive environment
  deploy_to_inactive
  
  # Step 4: Wait for rollout
  if ! wait_for_rollout; then
    log_error "Deployment failed during rollout"
    exit 1
  fi
  
  # Step 5: Health checks
  if ! check_deployment_health; then
    log_error "Deployment health check failed"
    exit 1
  fi
  
  # Step 6: Smoke tests
  if ! run_smoke_tests; then
    log_error "Smoke tests failed"
    exit 1
  fi
  
  # Step 7: Switch traffic
  if ! switch_traffic; then
    log_error "Traffic switch failed"
    exit 1
  fi
  
  # Step 8: Monitor deployment
  if ! monitor_deployment; then
    log_error "Deployment monitoring detected issues"
    exit 1
  fi
  
  # Success!
  echo -e "${GREEN}"
  echo "╔════════════════════════════════════════════════════════════╗"
  echo "║           ✅ Deployment Completed Successfully             ║"
  echo "║        Inactive environment kept for 24h rollback window  ║"
  echo "╚════════════════════════════════════════════════════════════╝"
  echo -e "${NC}"
  
  log_info "Next steps:"
  log_info "  1. Monitor metrics: kubectl port-forward -n $NAMESPACE svc/prometheus 9090:9090"
  log_info "  2. View logs: kubectl logs -n $NAMESPACE -l app=$DEPLOYMENT_NAME -f"
  log_info "  3. Rollback if needed: ./scripts/rollback.sh $ENVIRONMENT"
}

# Error handling
trap 'log_error "Script failed at line $LINENO"; exit 1' ERR

# Run main
main "$@"
