#!/bin/bash

# Live Process Monitoring Dashboard - Setup Script
# Delivered: January 1, 2026

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔴 Live Process Monitoring Dashboard - Setup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Step 1: Database Migration
echo "📊 Step 1: Running database migration..."
psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" \
  -f backend/migrations/misc/process_monitoring_schema.sql > /dev/null 2>&1

if [ $? -eq 0 ]; then
  echo "   ✅ Database migration completed"
  echo "      - Created process_interventions table"
  echo "      - Created 3 indexes"
else
  echo "   ❌ Database migration failed"
  exit 1
fi

echo ""

# Step 2: Verify Table Exists
echo "🔍 Step 2: Verifying database tables..."
TABLE_COUNT=$(psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" \
  -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'process_interventions';" 2>/dev/null | tr -d ' ')

if [ "$TABLE_COUNT" = "1" ]; then
  echo "   ✅ process_interventions table exists"
else
  echo "   ❌ Table verification failed"
  exit 1
fi

echo ""

# Step 3: Install gorilla/websocket
echo "📦 Step 3: Installing Go dependencies..."
cd backend
go get github.com/gorilla/websocket > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo "   ✅ gorilla/websocket installed"
else
  echo "   ⚠️  gorilla/websocket installation warning (may already exist)"
fi
cd ..

echo ""

# Step 4: Compile Backend
echo "🔨 Step 4: Compiling backend code..."
cd backend
go build ./internal/api/process_monitor_handlers.go > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo "   ✅ process_monitor_handlers.go compiles"
else
  echo "   ❌ Backend compilation failed"
  exit 1
fi

go build ./internal/business_process/event_broadcaster.go > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo "   ✅ event_broadcaster.go compiles"
else
  echo "   ❌ Event broadcaster compilation failed"
  exit 1
fi
cd ..

echo ""

# Step 5: Check Frontend Files
echo "🎨 Step 5: Checking frontend files..."
if [ -f "frontend/src/hooks/useProcessMonitorWebSocket.ts" ]; then
  echo "   ✅ useProcessMonitorWebSocket.ts exists"
else
  echo "   ❌ WebSocket hook missing"
  exit 1
fi

if [ -f "frontend/src/components/BPBuilder/ProcessMonitorDashboard.tsx" ]; then
  echo "   ✅ ProcessMonitorDashboard.tsx exists"
else
  echo "   ❌ Dashboard component missing"
  exit 1
fi

echo ""

# Step 6: Check Integration
echo "🔗 Step 6: Verifying BP Builder integration..."
if grep -q "ProcessMonitorDashboard" frontend/src/components/BPBuilder/BusinessProcessBuilderEnhanced.tsx; then
  echo "   ✅ Dashboard imported in BP Builder"
else
  echo "   ❌ Integration not found"
  exit 1
fi

if grep -q "'monitor'" frontend/src/components/BPBuilder/BusinessProcessBuilderEnhanced.tsx; then
  echo "   ✅ Monitor view mode added"
else
  echo "   ❌ View mode not added"
  exit 1
fi

if grep -q "Live Monitor" frontend/src/components/BPBuilder/BusinessProcessBuilderEnhanced.tsx; then
  echo "   ✅ Live Monitor button exists"
else
  echo "   ❌ Button not found"
  exit 1
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Live Process Monitoring Dashboard Setup Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📋 Next Steps:"
echo ""
echo "   1. Restart backend server:"
echo "      cd backend && go run cmd/server/main.go"
echo ""
echo "   2. Open BP Builder:"
echo "      http://localhost:3000/business-processes"
echo ""
echo "   3. Click green 'Live Monitor' button"
echo ""
echo "   4. Execute test workflows to see real-time updates"
echo ""
echo "   5. Test intervention actions:"
echo "      - Select running instance"
echo "      - Click Skip Step / Reassign / Retry / Cancel"
echo "      - Enter reason and execute"
echo ""
echo "📚 Documentation:"
echo "   - Setup Guide: LIVE_MONITORING_COMPLETE.md"
echo "   - User Guide: LIVE_MONITORING_GUIDE.md"
echo ""
echo "🎯 WebSocket Endpoint:"
echo "   ws://localhost:8080/api/process-monitor/ws?tenant_id=xxx&datasource_id=yyy"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
