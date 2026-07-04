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

HASURA_URL="${HASURA_URL:-http://localhost:8080}"
HASURA_ADMIN_SECRET="${HASURA_ADMIN_SECRET:-admin_secret_key}"

# ============================================================================
# STEP 1: Run Database Migration
# ============================================================================
echo "📦 Step 1: Running database migration..."

psql postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME} \
  -f backend/db/migrations/20251030_risk_management_schema.sql

echo "✅ Database migration complete\n"

# ============================================================================
# STEP 2: Track Tables in Hasura (via CLI)
# ============================================================================
echo "📊 Step 2: Tracking tables in Hasura..."

hasura metadata apply \
  --endpoint ${HASURA_URL} \
  --admin-secret ${HASURA_ADMIN_SECRET}

echo "✅ Hasura tables tracked\n"

# ============================================================================
# STEP 3: Register Risk Alpha Business Process
# ============================================================================
echo "⚙️  Step 3: Registering Risk Alpha business process..."

# Option A: Copy to registry
mkdir -p config/business_processes
cp config/business_processes/risk_alpha_v1.json \
   /path/to/your/bp/registry/

# Option B: Register via API
TENANT_ID="${TENANT_ID:-00000000-0000-0000-0000-000000000000}"

curl -X POST ${HASURA_URL}/api/business-processes \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d @config/business_processes/risk_alpha_v1.json

echo "✅ Risk Alpha business process registered\n"

# ============================================================================
# STEP 4: Verify Activities Registration
# ============================================================================
echo "🔧 Step 4: Verifying Temporal activities..."

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
# STEP 5: Rebuild and Restart Worker
# ============================================================================
echo "👷 Step 5: Rebuilding and restarting worker..."

cd rebalancing/worker
go build -o rebalancing-worker main.go
./rebalancing-worker &
cd ../..

echo "✅ Worker started\n"

# ============================================================================
# STEP 6: Verify Everything
# ============================================================================
echo "✅ Step 6: Verification checks..."

echo "Checking database..."
psql postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME} \
  -c "SELECT COUNT(*) as risk_events FROM risk_events;" || true

echo ""
echo "Checking Hasura GraphQL..."
curl -s -X POST ${HASURA_URL}/v1/graphql \
  -H "X-Hasura-Admin-Secret: ${HASURA_ADMIN_SECRET}" \
  -H "Content-Type: application/json" \
  -d '{"query": "{ risk_events { id } }"}' | head -c 100
echo "\n"

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
5. See risk_events populate in Hasura
6. Dashboard updates in real-time via subscriptions

Troubleshooting:
- Check logs: docker logs temporal
- Verify xAI API key set in env
- Ensure Hasura tables are tracked
- Check Redpanda (Kafka) connection: docker exec semlayer-redpanda rpk cluster info

For details, see: RISK_ALPHA_INTEGRATION_GUIDE.md
"
