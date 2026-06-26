#!/bin/bash
# Rollback Deployment Script
# One-command rollback to previous version using blue-green pattern
#
# Usage: ./rollback.sh [staging|production]

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Parameters
ENVIRONMENT="${1:-staging}"
NAMESPACE="calendar"
SERVICE_NAME="calendar-service"

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

# Get current active environment
get_current_env() {
  kubectl get svc "$SERVICE_NAME" -n "$NAMESPACE" \
    -o jsonpath='{.spec.selector.version}' 2>/dev/null || echo "blue"
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

# Main rollback
main() {
  echo -e "${BLUE}"
  echo "╔════════════════════════════════════════════════════════════╗"
  echo "║          Rollback Script                                   ║"
  echo "║          Environment: $ENVIRONMENT (Namespace: $NAMESPACE)"
  echo "╚════════════════════════════════════════════════════════════╝"
  echo -e "${NC}"
  
  local current=$(get_current_env)
  local previous=$(get_inactive_env)
  
  log_info "Current active environment: $current"
  log_info "Previous environment (to rollback to): $previous"
  
  # Confirmation
  read -p "Are you sure you want to rollback to $previous? (y/N): " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_warning "Rollback cancelled"
    exit 0
  fi
  
  log_warning "Rolling back..."
  
  # Switch service selector back
  kubectl patch service "$SERVICE_NAME" \
    -n "$NAMESPACE" \
    -p "{\"spec\":{\"selector\":{\"version\":\"$previous\"}}}" \
    --type merge
  
  log_success "Service selector updated to: $previous"
  
  # Wait for connections to drain
  sleep 5
  
  # Verify rollback
  local new_current=$(get_current_env)
  if [ "$new_current" == "$previous" ]; then
    log_success "Rollback completed successfully!"
    log_success "Now running: $new_current"
  else
    log_error "Rollback verification failed"
    exit 1
  fi
  
  # Show pod info
  echo -e "\n${BLUE}Pod Status:${NC}"
  kubectl get pods -n "$NAMESPACE" -l "app=$SERVICE_NAME"
}

main "$@"
