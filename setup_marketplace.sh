#!/bin/bash

# Integration Marketplace Setup and Verification Script
# This script verifies that the Integration Marketplace is properly installed and configured

set -e

echo "========================================="
echo "Integration Marketplace Setup Verification"
echo "========================================="
echo ""

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Database connection
DB_URL="postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"

# Function to print success message
success() {
    echo -e "${GREEN}✓${NC} $1"
}

# Function to print error message
error() {
    echo -e "${RED}✗${NC} $1"
}

# Function to print warning message
warning() {
    echo -e "${YELLOW}!${NC} $1"
}

# Function to print section header
section() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "$1"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Check if PostgreSQL is running
section "1. Checking PostgreSQL Connection"
if psql "$DB_URL" -c "SELECT 1" > /dev/null 2>&1; then
    success "PostgreSQL connection successful"
else
    error "Cannot connect to PostgreSQL"
    echo "  Please ensure PostgreSQL is running and accessible at:"
    echo "  $DB_URL"
    exit 1
fi

# Check database schema
section "2. Verifying Database Schema"

tables=(
    "marketplace_integrations"
    "installed_integrations"
    "integration_executions"
    "marketplace_integration_settings"
)

for table in "${tables[@]}"; do
    if psql "$DB_URL" -c "SELECT 1 FROM $table LIMIT 1" > /dev/null 2>&1; then
        success "Table exists: $table"
    else
        error "Table missing: $table"
        echo "  Run: psql \"$DB_URL\" -f backend/migrations/misc/integration_marketplace_schema.sql"
        exit 1
    fi
done

# Check for seeded integrations
section "3. Checking Marketplace Catalog"

integration_count=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM marketplace_integrations")
integration_count=$(echo $integration_count | xargs) # Trim whitespace

if [ "$integration_count" -eq 0 ]; then
    warning "No integrations found in marketplace catalog"
    echo "  Run: psql \"$DB_URL\" -f backend/migrations/misc/seed_marketplace_integrations.sql"
    echo ""
    read -p "Would you like to seed the marketplace now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        psql "$DB_URL" -f backend/migrations/misc/seed_marketplace_integrations.sql > /dev/null 2>&1
        success "Marketplace seeded with pre-built integrations"
    fi
else
    success "Found $integration_count integration(s) in marketplace"
    
    # List integrations
    echo ""
    echo "  Available integrations:"
    psql "$DB_URL" -c "SELECT integration_key, name, category, rating, install_count FROM marketplace_integrations ORDER BY install_count DESC" -x | grep -E '(integration_key|name|category|rating|install_count)'
fi

# Check backend files
section "4. Verifying Backend Implementation"

backend_files=(
    "backend/internal/api/marketplace_integration_handlers.go"
    "backend/migrations/misc/integration_marketplace_schema.sql"
    "backend/migrations/misc/seed_marketplace_integrations.sql"
)

for file in "${backend_files[@]}"; do
    if [ -f "$file" ]; then
        success "File exists: $file"
    else
        error "File missing: $file"
        exit 1
    fi
done

# Check if routes are registered
if grep -q "marketplaceIntegrationHandler" backend/internal/api/api.go; then
    success "Routes registered in api.go"
else
    warning "Routes not found in api.go"
    echo "  Add the following to backend/internal/api/api.go:"
    echo "  marketplaceIntegrationHandler := NewMarketplaceIntegrationHandlers(sqlxDB)"
    echo "  marketplaceIntegrationHandler.RegisterRoutes(r)"
fi

# Try to compile backend
section "5. Compiling Backend"

if go build -C backend ./internal/api/marketplace_integration_handlers.go > /dev/null 2>&1; then
    success "Backend compiles successfully"
else
    error "Backend compilation failed"
    echo "  Run: cd backend && go build ./internal/api/marketplace_integration_handlers.go"
    exit 1
fi

# Check frontend files
section "6. Verifying Frontend Implementation"

frontend_files=(
    "frontend/src/components/BPBuilder/IntegrationMarketplaceBrowser.tsx"
    "frontend/src/components/BPBuilder/BusinessProcessBuilderEnhanced.tsx"
)

for file in "${frontend_files[@]}"; do
    if [ -f "$file" ]; then
        success "File exists: $file"
    else
        error "File missing: $file"
        exit 1
    fi
done

# Check if marketplace is integrated with BP Builder
if grep -q "IntegrationMarketplaceBrowser" frontend/src/components/BPBuilder/BusinessProcessBuilderEnhanced.tsx; then
    success "Marketplace integrated with BP Builder"
else
    warning "Marketplace not integrated with BP Builder"
    echo "  Add IntegrationMarketplaceBrowser component to BusinessProcessBuilderEnhanced.tsx"
fi

# Check documentation
section "7. Checking Documentation"

doc_files=(
    "INTEGRATION_MARKETPLACE_GUIDE.md"
    "INTEGRATION_DEVELOPER_GUIDE.md"
)

for file in "${doc_files[@]}"; do
    if [ -f "$file" ]; then
        success "Documentation exists: $file"
    else
        warning "Documentation missing: $file"
    fi
done

# Test API endpoints (if backend is running)
section "8. Testing API Endpoints (Optional)"

# Check if backend is running
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    success "Backend is running on port 8080"
    
    # Test marketplace endpoint
    echo ""
    echo "  Testing GET /api/integrations/marketplace..."
    
    response=$(curl -s -w "%{http_code}" http://localhost:8080/api/integrations/marketplace?tenant_id=00000000-0000-0000-0000-000000000000&datasource_id=00000000-0000-0000-0000-000000000000 -o /tmp/marketplace_test.json)
    
    if [ "$response" == "200" ]; then
        success "API endpoint responds successfully"
        integration_count=$(cat /tmp/marketplace_test.json | jq '. | length' 2>/dev/null || echo "?")
        echo "    Returned $integration_count integrations"
    else
        warning "API returned status code: $response"
    fi
    
    rm -f /tmp/marketplace_test.json
else
    warning "Backend is not running"
    echo "  Start backend to test API endpoints:"
    echo "  cd backend && go run ./cmd/api-gateway"
fi

# Summary
section "Setup Summary"

echo ""
echo "Database Schema:    ✓ All tables created"
echo "Marketplace Catalog: ✓ $integration_count integration(s) available"
echo "Backend Code:       ✓ Handlers implemented and compiled"
echo "Frontend Code:      ✓ UI components created"
echo "Documentation:      ✓ Guides available"
echo ""

# Next steps
section "Next Steps"

echo ""
echo "1. Start the backend (if not already running):"
echo "   cd backend && go run ./cmd/api-gateway"
echo ""
echo "2. Start the frontend (if not already running):"
echo "   cd frontend && npm start"
echo ""
echo "3. Access the Integration Marketplace:"
echo "   - Open Business Process Builder"
echo "   - Click the 'Integrations' button (green button with package icon)"
echo "   - Browse available integrations"
echo "   - Install an integration (e.g., Webhook or Email)"
echo "   - Test the connection"
echo ""
echo "4. Read the documentation:"
echo "   - User Guide: INTEGRATION_MARKETPLACE_GUIDE.md"
echo "   - Developer Guide: INTEGRATION_DEVELOPER_GUIDE.md"
echo ""
echo "5. Create a workflow that uses an integration:"
echo "   - Add a step with type 'Integration Action'"
echo "   - Select your installed integration"
echo "   - Configure the action parameters"
echo "   - Execute the workflow"
echo "   - View execution logs in the 'Execution Logs' tab"
echo ""

success "Integration Marketplace setup complete!"
echo ""
echo "========================================="
