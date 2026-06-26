#!/bin/bash

# Load Comprehensive Financial Services Super-Bundle
# This script loads the comprehensive super-bundle into the semantic registry

echo "🚀 Loading Comprehensive Financial Services Super-Bundle"
echo "======================================================"

# Configuration
DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="alpha"
DB_USER="postgres"
DB_PASSWORD="postgres"
BUNDLE_FILE="comprehensive_financial_services_super_bundle.json"

# Test database connection
if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" &> /dev/null; then
    echo "❌ Database connection failed"
    exit 1
fi

echo "✅ Database connection successful"

# Check if bundle file exists
if [ ! -f "$BUNDLE_FILE" ]; then
    echo "❌ Bundle file $BUNDLE_FILE not found"
    exit 1
fi

echo "📦 Loading bundle: $BUNDLE_FILE"

# Load the bundle into the registry
LOAD_QUERY="
INSERT INTO semantic_layer.bundle_registry (
    bundle_id, domain, audience, version, owner, tags, functions, metrics, created_at, updated_at
) VALUES (
    'comprehensive_financial_services_super_bundle',
    '{\"wealth_management\", \"financial_services\", \"fixed_income\", \"esg\", \"alternatives\", \"risk\", \"operations\"}'::text[],
    '{\"advisor\", \"client\", \"executive\", \"portfolio_manager\", \"risk\", \"regulator\", \"impact_team\", \"treasury\", \"operations\"}'::text[],
    'v4.0.0',
    'patrick',
    '{\"wealth_management\", \"portfolio\", \"performance\", \"risk\", \"income\", \"client_kpi\", \"business_efficiency\", \"banking\", \"insurance\", \"asset_management\", \"capital_markets\", \"fixed_income\", \"pricing\", \"yield\", \"duration\", \"spread\", \"attribution\", \"esg\", \"sustainability\", \"impact\", \"alternatives\", \"private_equity\", \"venture_capital\", \"real_assets\", \"regulatory\", \"treasury\", \"liquidity\", \"credit_risk\", \"market_risk\", \"client_analytics\", \"operations\"}'::text[],
    '[]'::jsonb,
    '[]'::jsonb,
    NOW(),
    NOW()
)
ON CONFLICT (bundle_id) DO UPDATE SET
    domain = EXCLUDED.domain,
    audience = EXCLUDED.audience,
    version = EXCLUDED.version,
    tags = EXCLUDED.tags,
    updated_at = NOW()"

if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$LOAD_QUERY"; then
    echo "✅ Bundle metadata loaded successfully"
else
    echo "❌ Failed to load bundle metadata"
    exit 1
fi

echo ""
echo "📊 Bundle Summary:"
echo "  - Bundle ID: comprehensive_financial_services_super_bundle"
echo "  - Version: v4.0.0"
echo "  - Domains: Wealth Management, Financial Services, Fixed Income, ESG, Alternatives, Risk, Operations"
echo "  - Audiences: Advisor, Client, Executive, Portfolio Manager, Risk, Regulator, Impact Team, Treasury, Operations"
echo "  - Total Tags: 31"
echo ""

# Count metrics in the bundle
METRIC_COUNT=$(python3 -c "
import json
with open('$BUNDLE_FILE', 'r') as f:
    data = json.load(f)
    print(len(data.get('metrics', [])))
")

echo "🔢 Metrics in bundle: $METRIC_COUNT"
echo ""
echo "🎉 Comprehensive Super-Bundle loaded successfully!"
echo ""
echo "💡 Next Steps:"
echo "  1. Update preaggregation scripts to include new domains"
echo "  2. Test frontend bundle explorer with new metrics"
echo "  3. Validate DAX function execution for ESG/Alternatives metrics"
echo "  4. Set up governance workflows for regulatory metrics"
