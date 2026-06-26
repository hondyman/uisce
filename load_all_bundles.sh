#!/bin/bash

# Load all new DAX-annotated bundles into the semantic registry
echo "🚀 Loading All New DAX-Annotated Bundles"
echo "========================================"

BUNDLES=(
    "banking_lending_bundle.json"
    "insurance_bundle.json"
    "capital_markets_bundle.json"
    "regulatory_compliance_bundle.json"
    "healthcare_bundle.json"
    "retail_bundle.json"
    "investment_accounting_pack.json"
    "currency_fx_pack.json"
    "unified_financial_services_super_bundle.json"
)

for bundle in "${BUNDLES[@]}"; do
    if [ -f "$bundle" ]; then
        echo ""
        echo "📦 Loading $bundle..."
        go run load_generic_bundle.go "$bundle"
        echo "✅ $bundle loaded successfully"
    else
        echo "⚠️  $bundle not found, skipping"
    fi
done

echo ""
echo "🎉 All bundles loaded!"
echo "======================"
echo "Your semantic layer now supports:"
echo "• Banking & Lending Analytics"
echo "• Insurance Risk Management"
echo "• Capital Markets Trading"
echo "• Regulatory Compliance"
echo "• Healthcare Operations"
echo "• Retail Performance"
echo ""
echo "All metrics are DAX-powered with governance and audience controls."
