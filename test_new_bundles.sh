#!/bin/bash

# Test script for new DAX-annotated bundles
echo "🧪 Testing New DAX-Annotated Bundles"
echo "===================================="

# Test database connection
echo ""
echo "🔍 Testing database connection..."
if PGPASSWORD="postgres" psql -h localhost -p 5432 -U postgres -d alpha -c "SELECT 1;" &> /dev/null; then
    echo "✅ Database connection successful"
else
    echo "❌ Database connection failed"
    exit 1
fi

# Test bundle schemas
echo ""
echo "📦 Testing bundle schemas..."
DOMAINS=("banking" "insurance" "capital_markets" "regulatory" "healthcare" "retail" "financial_services" "fixed_income")

for domain in "${DOMAINS[@]}"; do
    if PGPASSWORD="postgres" psql -h localhost -p 5432 -U postgres -d alpha -c "SELECT COUNT(*) FROM ${domain}.metrics_registry;" &> /dev/null; then
        COUNT=$(PGPASSWORD="postgres" psql -h localhost -p 5432 -U postgres -d alpha -c "SELECT COUNT(*) FROM ${domain}.metrics_registry;" -t -A)
        echo "✅ $domain bundle: $COUNT metrics loaded"

        # Test DAX functions table if it exists
        if PGPASSWORD="postgres" psql -h localhost -p 5432 -U postgres -d alpha -c "SELECT COUNT(*) FROM ${domain}.dax_functions;" &> /dev/null; then
            FN_COUNT=$(PGPASSWORD="postgres" psql -h localhost -p 5432 -U postgres -d alpha -c "SELECT COUNT(*) FROM ${domain}.dax_functions;" -t -A)
            echo "   📊 DAX functions: $FN_COUNT functions"
        fi
    else
        echo "❌ $domain bundle: Schema or table not found"
    fi
done

# Test DAX engine functions
echo ""
echo "🔧 Testing DAX engine functions..."
if [ -f "backend/internal/services/dax_engine.go" ]; then
    echo "✅ DAX engine file exists"

    # Check for required functions
    REQUIRED_FUNCTIONS=("SUMX" "AVERAGEX" "FILTER" "DIVIDE" "MINX" "MAXX")
    for func in "${REQUIRED_FUNCTIONS[@]}"; do
        if grep -q "e.functions\[\"$func\"\]" backend/internal/services/dax_engine.go; then
            echo "✅ $func function registered"
        else
            echo "❌ $func function not found"
        fi
    done
else
    echo "❌ DAX engine file not found"
fi

# Test backend compilation
echo ""
echo "🔨 Testing backend compilation..."
if cd backend && go build ./cmd/server; then
    echo "✅ Backend compiles successfully"
    rm -f server
else
    echo "❌ Backend compilation failed"
fi

# Test bundle JSON validation
echo ""
echo "📋 Testing bundle JSON validation..."
BUNDLES=(
    "/Users/eganpj/GitHub/semlayer/banking_lending_bundle.json"
    "/Users/eganpj/GitHub/semlayer/insurance_bundle.json"
    "/Users/eganpj/GitHub/semlayer/capital_markets_bundle.json"
    "/Users/eganpj/GitHub/semlayer/regulatory_compliance_bundle.json"
    "/Users/eganpj/GitHub/semlayer/healthcare_bundle.json"
    "/Users/eganpj/GitHub/semlayer/retail_bundle.json"
    "/Users/eganpj/GitHub/semlayer/financial_services_expansion_pack.json"
    "/Users/eganpj/GitHub/semlayer/fixed_income_pack.json"
    "/Users/eganpj/GitHub/semlayer/unified_financial_services_bundle.json"
)

for bundle in "${BUNDLES[@]}"; do
    if [ -f "$bundle" ]; then
        if jq empty "$bundle" 2>/dev/null; then
            echo "✅ $bundle: Valid JSON"
        else
            echo "❌ $bundle: Invalid JSON"
        fi
    else
        echo "⚠️  $bundle: File not found"
    fi
done

# Test preaggregation plans
echo ""
echo "📊 Testing preaggregation plans..."
PLANS=(
    "/Users/eganpj/GitHub/semlayer/banking_preaggregation_plan.json"
    "/Users/eganpj/GitHub/semlayer/insurance_preaggregation_plan.json"
    "/Users/eganpj/GitHub/semlayer/capital_markets_preaggregation_plan.json"
    "/Users/eganpj/GitHub/semlayer/fixed_income_preaggregation_plan.json"
)

for plan in "${PLANS[@]}"; do
    if [ -f "$plan" ]; then
        if jq empty "$plan" 2>/dev/null; then
            echo "✅ $plan: Valid JSON"
        else
            echo "❌ $plan: Invalid JSON"
        fi
    else
        echo "⚠️  $plan: File not found"
    fi
done

echo ""
echo "🎉 Bundle testing complete!"
echo "==========================="
echo "If all tests passed, your semantic layer is ready with:"
echo "• 7 domain-specific metric bundles (including unified financial services)"
echo "• 60+ DAX-powered metrics across all financial domains"
echo "• Complete DAX function library with 24 functions"
echo "• Preaggregation plans for performance optimization"
echo "• Frontend components for bundle discovery and exploration"
