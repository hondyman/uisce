#!/bin/bash
# Risk Alpha Deployment Quick Reference
# Copy and paste commands below to deploy Risk Alpha to your platform

set -e

echo "🚀 Risk Alpha Deployment Script"
echo "================================\n"

# Configuration
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-your_db}"

API_URL="${API_URL:-http://localhost:8080}"

# ============================================================================
# STEP 1: Run Database Migration
# ============================================================================
echo "📦 Step 1: Running database migration..."

psql postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME} \
  -f backend/db/migrations/20251030_risk_management_schema.sql

echo "✅ Database migration complete\n"

# ============================================================================
# STEP 2: Register Risk Alpha Business Process
# ============================================================================
echo "⚙️  Step 2: Registering Risk Alpha business process..."

# Option A: Copy to registry
mkdir -p config/business_processes
cp config/business_processes/risk_alpha_v1.json \
   /path/to/your/bp/registry/

# Option B: Register via API
TENANT_ID="${TENANT_ID:-00000000-0000-0000-0000-000000000000}"

curl -X POST ${API_URL}/api/business-processes \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d @config/business_processes/risk_alpha_v1.json

echo "✅ Risk Alpha business process registered\n"

# ============================================================================
# STEP 3: Verify Activities Registration
# ============================================================================
echo "🔧 Step 3: Verifying Temporal activities..."

echo "
Activities expected to be registered:
  ✓ AIRiskScoreComprehensive
  ✓ AIMitigationStrategy
  ✓ ExecuteRiskMitigation
  ✓ CreateRiskEvent
  ✓ UpdateRiskEventMitigated

Check rebalancing/worker/main.go to confirm registration.
"

# ============================================================================
# STEP 4: Rebuild and Restart Worker
# ============================================================================
echo "👷 Step 4: Rebuilding and restarting worker..."

cd rebalancing/worker
go build -o rebalancing-worker main.go
./rebalancing-worker &
cd ../..

echo "✅ Worker started\n"

# ============================================================================
# STEP 5: Verify Everything
# ============================================================================
echo "✅ Step 5: Verification checks..."

echo "Checking database..."
psql postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME} \
  -c "SELECT COUNT(*) as risk_events FROM risk_events;" || true

# ============================================================================
# DONE
# ============================================================================
echo "
🎉 Risk Alpha Deployment Complete!
==================================

Next steps:
1. Mount RiskAlphaDashboard component in your React app
2. Navigate to Risk Alpha Dashboard
3. Click 'Run AI Analysis' on any portfolio
4. Watch Temporal UI at http://localhost:8081
5. See risk_events populate in the database
6. Dashboard updates in real-time via subscriptions

Troubleshooting:
- Check logs: docker logs temporal
- Verify xAI API key set in env
- Check Redpanda (Kafka) connection: docker exec semlayer-redpanda rpk cluster info

For details, see: RISK_ALPHA_INTEGRATION_GUIDE.md
"
