#!/bin/bash

# Process Analytics Dashboard - Complete Setup Script
# Run this script to fully integrate the analytics dashboard

set -e  # Exit on any error

echo "🚀 Setting up Process Analytics Dashboard..."
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Run database migration
echo -e "${BLUE}Step 1: Running database migration...${NC}"
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable \
  -f backend/migrations/misc/process_analytics_schema.sql

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Database migration completed${NC}"
else
    echo -e "${YELLOW}⚠️  Database migration may have already run${NC}"
fi
echo ""

# Step 2: Verify tables
echo -e "${BLUE}Step 2: Verifying analytics tables...${NC}"
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable \
  -c "SELECT COUNT(*) FROM process_execution_metrics" > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ process_execution_metrics table exists${NC}"
else
    echo -e "${YELLOW}❌ process_execution_metrics table not found${NC}"
    exit 1
fi

psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable \
  -c "SELECT COUNT(*) FROM process_bottleneck_analysis" > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ process_bottleneck_analysis table exists${NC}"
else
    echo -e "${YELLOW}❌ process_bottleneck_analysis table not found${NC}"
    exit 1
fi
echo ""

# Step 3: Check backend compilation
echo -e "${BLUE}Step 3: Checking backend code...${NC}"
cd backend
go build -o /dev/null ./internal/api/process_analytics_handlers.go 2>&1 | head -5

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Backend analytics code compiles${NC}"
else
    echo -e "${YELLOW}⚠️  Check compilation errors above${NC}"
fi
cd ..
echo ""

# Step 4: Check frontend compilation
echo -e "${BLUE}Step 4: Checking frontend code...${NC}"
if [ -d "frontend/node_modules" ]; then
    cd frontend
    npx tsc --noEmit --skipLibCheck \
      src/components/BPBuilder/ProcessAnalyticsDashboard.tsx 2>&1 | head -5
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Frontend analytics code compiles${NC}"
    else
        echo -e "${YELLOW}⚠️  Check TypeScript errors above${NC}"
    fi
    cd ..
else
    echo -e "${YELLOW}⚠️  Frontend node_modules not found, skipping check${NC}"
fi
echo ""

# Step 5: Print next steps
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ Process Analytics Dashboard Setup Complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo ""
echo "1. Restart your backend server to load new routes"
echo "   $ cd backend && go run cmd/server/main.go"
echo ""
echo "2. Access the analytics dashboard:"
echo "   - Navigate to BP Builder"
echo "   - Click 'View Analytics' button in the sidebar"
echo "   - Or click the Analytics tab in the view modes"
echo ""
echo "3. Generate test data (optional):"
echo "   $ curl -X POST http://localhost:8080/api/process-analytics/analyze-bottlenecks?tenant_id=YOUR_TENANT_ID"
echo ""
echo "4. View dashboard at: /business-processes (Analytics tab)"
echo ""
echo -e "${YELLOW}📖 Documentation:${NC}"
echo "   - Complete Guide: PROCESS_ANALYTICS_COMPLETE.md"
echo "   - Quick Start: PROCESS_ANALYTICS_QUICK_START.md"
echo "   - Delivery Summary: PROCESS_ANALYTICS_DELIVERY_SUMMARY.md"
echo ""
echo -e "${GREEN}🎉 Happy Analyzing!${NC}"
