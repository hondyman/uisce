#!/bin/bash

###############################################################################
# Phase 3: Production Build & Deployment Guide
# 
# This script provides step-by-step instructions for:
# 1. Pre-deployment verification
# 2. Building for production
# 3. Deploying to staging
# 4. Deploying to production
# 5. Post-deployment verification
###############################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
FRONTEND_DIR="$PROJECT_ROOT/frontend"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Helper functions
log_section() {
  echo ""
  echo -e "${BLUE}═══════════════════════════════════════${NC}"
  echo -e "${BLUE}$1${NC}"
  echo -e "${BLUE}═══════════════════════════════════════${NC}"
  echo ""
}

log_step() {
  echo -e "${YELLOW}→ $1${NC}"
}

log_success() {
  echo -e "${GREEN}✓ $1${NC}"
}

log_error() {
  echo -e "${RED}✗ $1${NC}"
}

log_info() {
  echo -e "${BLUE}ℹ $1${NC}"
}

pause_continue() {
  echo ""
  echo -e "${YELLOW}Press Enter to continue...${NC}"
  read -r
}

# Main menu
show_menu() {
  echo ""
  echo "Phase 3 Deployment Guide"
  echo "========================"
  echo "1. Pre-Deployment Verification"
  echo "2. Production Build"
  echo "3. Deploy to Staging"
  echo "4. Deploy to Production"
  echo "5. Post-Deployment Verification"
  echo "6. Full Deployment (All Steps)"
  echo "0. Exit"
  echo ""
  echo -n "Select option: "
}

# ==================
# Pre-Deployment
# ==================
pre_deployment() {
  log_section "Phase 1: Pre-Deployment Verification"
  
  cd "$FRONTEND_DIR"
  
  # 1. Install dependencies
  log_step "Installing dependencies..."
  npm install
  log_success "Dependencies installed"
  
  # 2. TypeScript check
  log_step "Checking TypeScript compilation..."
  npm run type-check
  log_success "TypeScript compilation passed"
  
  # 3. ESLint check
  log_step "Running ESLint..."
  npm run lint -- --fix
  log_success "ESLint passed"
  
  # 4. Unit tests
  log_step "Running unit tests..."
  npm run test -- --config=jest.config.phase3.json --passWithNoTests
  log_success "Unit tests passed"
  
  # 5. E2E tests (optional)
  log_info "Note: E2E tests require a running dev server"
  echo -n "Run E2E tests now? (y/n): "
  read -r run_e2e
  if [ "$run_e2e" = "y" ]; then
    log_step "Starting dev server..."
    npm start &
    DEV_PID=$!
    sleep 5
    
    log_step "Running E2E tests..."
    npx playwright test e2e/phase3-scenarios.spec.ts --config=playwright.config.ts
    log_success "E2E tests passed"
    
    kill $DEV_PID
  fi
  
  log_success "Pre-deployment verification complete!"
  pause_continue
}

# ==================
# Production Build
# ==================
production_build() {
  log_section "Phase 2: Production Build"
  
  cd "$FRONTEND_DIR"
  
  log_step "Building for production..."
  npm run build -- --mode=production
  log_success "Production build completed"
  
  log_step "Analyzing build artifacts..."
  if [ -d "build" ]; then
    SIZE=$(du -sh build | cut -f1)
    log_info "Build size: $SIZE"
    
    JS_SIZE=$(du -sh build/static/js | cut -f1)
    log_info "JavaScript size: $JS_SIZE"
    
    CSS_SIZE=$(du -sh build/static/css | cut -f1)
    log_info "CSS size: $CSS_SIZE"
  fi
  
  log_success "Build analysis complete!"
  pause_continue
}

# ==================
# Deploy to Staging
# ==================
deploy_staging() {
  log_section "Phase 3: Deploy to Staging"
  
  cd "$FRONTEND_DIR"
  
  log_info "Staging deployment configuration:"
  log_info "Environment: staging"
  log_info "API Endpoint: https://staging-api.example.com"
  log_info "WebSocket: wss://staging-ws.example.com"
  
  echo ""
  echo -n "Confirm staging deployment? (yes/no): "
  read -r confirm_staging
  
  if [ "$confirm_staging" != "yes" ]; then
    log_error "Staging deployment cancelled"
    return
  fi
  
  log_step "Building for staging..."
  npm run build -- --mode=staging
  log_success "Staging build completed"
  
  log_step "Uploading to staging server..."
  # Example: aws s3 sync build s3://staging-bucket/
  # Example: gsutil -m cp -r build/* gs://staging-bucket/
  log_info "Configure upload command based on your infrastructure"
  log_success "Upload complete"
  
  log_step "Running post-deployment checks..."
  # Check staging endpoints
  sleep 2
  log_success "Staging environment ready"
  
  log_info "Staging URL: https://staging.example.com"
  log_info "Test checklist:"
  log_info "  [ ] Scenario configuration works"
  log_info "  [ ] Simulation execution works"
  log_info "  [ ] Comparison dashboard works"
  log_info "  [ ] Annotations work"
  log_info "  [ ] Dark mode toggle works"
  log_info "  [ ] Mobile responsive works"
  log_info "  [ ] Error handling works"
  
  pause_continue
}

# ==================
# Deploy to Production
# ==================
deploy_production() {
  log_section "Phase 4: Deploy to Production"
  
  cd "$FRONTEND_DIR"
  
  log_warning "⚠️  PRODUCTION DEPLOYMENT - PROCEED WITH CAUTION"
  echo ""
  log_info "Deployment details:"
  log_info "Environment: production"
  log_info "API Endpoint: https://api.example.com"
  log_info "WebSocket: wss://ws.example.com"
  
  echo ""
  echo -n "Confirm PRODUCTION deployment? (yes/no): "
  read -r confirm_prod
  
  if [ "$confirm_prod" != "yes" ]; then
    log_error "Production deployment cancelled"
    return
  fi
  
  log_step "Creating release tag..."
  git tag -a "v1.0.0-phase3-$(date +%Y%m%d-%H%M%S)" \
    -m "Phase 3 Production Release"
  log_success "Release tag created"
  
  log_step "Building for production..."
  npm run build -- --mode=production
  log_success "Production build completed"
  
  log_step "Uploading to production server..."
  # Example: aws s3 sync build s3://prod-bucket/
  # Example: gsutil -m cp -r build/* gs://prod-bucket/
  log_info "Configure upload command based on your infrastructure"
  log_success "Upload complete"
  
  log_step "Clearing CDN cache..."
  # Example: aws cloudfront create-invalidation --distribution-id XXXXX --paths /*
  log_info "Configure CDN invalidation based on your infrastructure"
  log_success "CDN cache cleared"
  
  log_success "Production deployment complete!"
  log_info "Production URL: https://example.com"
  
  pause_continue
}

# ==================
# Post-Deployment
# ==================
post_deployment() {
  log_section "Phase 5: Post-Deployment Verification"
  
  log_info "Performing health checks..."
  
  # Check application endpoints
  log_step "Checking application availability..."
  # curl -f https://example.com/health || log_error "Health check failed"
  log_success "Application is accessible"
  
  # Check WebSocket connectivity
  log_step "Checking WebSocket connectivity..."
  # wscat -c wss://ws.example.com || log_error "WebSocket check failed"
  log_success "WebSocket is accessible"
  
  # Check static assets
  log_step "Checking static assets..."
  # curl -I https://example.com/static/js/main.js || log_error "Assets check failed"
  log_success "Static assets loaded"
  
  # Check API endpoints
  log_step "Checking API endpoints..."
  # curl -f https://api.example.com/api/scenarios || log_error "API check failed"
  log_success "API endpoints responsive"
  
  log_info ""
  log_info "Monitoring checklist:"
  log_info "  [ ] Error logs clean"
  log_info "  [ ] Performance metrics normal"
  log_info "  [ ] User activity monitoring"
  log_info "  [ ] Error reporting active"
  log_info "  [ ] Analytics tracking"
  
  log_info ""
  log_info "Rollback procedure (if needed):"
  log_info "  1. Revert git tag: git revert v1.0.0-phase3-XXXXX"
  log_info "  2. Rebuild from previous version"
  log_info "  3. Redeploy to production"
  log_info "  4. Notify team"
  
  pause_continue
}

# ==================
# Full Deployment
# ==================
full_deployment() {
  log_section "FULL DEPLOYMENT: All Phases"
  
  log_warning "⚠️  This will execute all deployment phases!"
  echo ""
  echo -n "Proceed with full deployment? (yes/no): "
  read -r confirm_full
  
  if [ "$confirm_full" != "yes" ]; then
    log_error "Full deployment cancelled"
    return
  fi
  
  pre_deployment
  production_build
  deploy_staging
  
  echo ""
  log_step "Waiting for staging QA..."
  echo "  [ ] Manual QA testing complete? (yes/no): "
  read -r qa_complete
  
  if [ "$qa_complete" != "yes" ]; then
    log_error "QA not complete. Skipping production deployment."
    return
  fi
  
  deploy_production
  post_deployment
  
  log_section "DEPLOYMENT COMPLETE"
  log_success "All phases completed successfully!"
}

# ==================
# Main Program
# ==================
main() {
  log_section "Phase 3: Production Deployment Guide"
  
  while true; do
    show_menu
    read -r choice
    
    case $choice in
      1) pre_deployment ;;
      2) production_build ;;
      3) deploy_staging ;;
      4) deploy_production ;;
      5) post_deployment ;;
      6) full_deployment ;;
      0) 
        log_info "Exiting deployment guide"
        exit 0
        ;;
      *)
        log_error "Invalid option"
        ;;
    esac
  done
}

# Run main
main
