#!/bin/bash
# Script to find all references to domain-specific metrics_registry and dax_functions tables
# Run from repo root: bash find_schema_references.sh

echo "🔍 Searching for references to domain-specific metrics and DAX functions..."
echo ""

SEARCH_DIRS=("backend" "frontend" "migrations")
DOMAIN_SCHEMAS=("banking" "capital_markets" "currency_fx" "financial_services" "fixed_income" "foffice" "hdb_catalog" "healthcare" "hld" "insurance" "investment_accounting" "regulatory" "report_sys" "retail" "semantic_layer" "sml" "unified_financial_services" "wealth_management")

echo "=== REFERENCES TO metrics_registry ==="
for dir in "${SEARCH_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo ""
        echo "📁 Searching in $dir/"
        grep -r "metrics_registry" "$dir" --include="*.go" --include="*.ts" --include="*.tsx" --include="*.js" --include="*.sql" 2>/dev/null | grep -v node_modules | grep -v ".next" | head -20
    fi
done

echo ""
echo ""
echo "=== REFERENCES TO dax_functions ==="
for dir in "${SEARCH_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo ""
        echo "📁 Searching in $dir/"
        grep -r "dax_functions" "$dir" --include="*.go" --include="*.ts" --include="*.tsx" --include="*.js" --include="*.sql" 2>/dev/null | grep -v node_modules | grep -v ".next" | head -20
    fi
done

echo ""
echo ""
echo "=== DOMAIN SCHEMA REFERENCES (SELECT COUNT) ==="
for schema in "${DOMAIN_SCHEMAS[@]}"; do
    echo ""
    echo "Schema: $schema"
    for dir in "${SEARCH_DIRS[@]}"; do
        if [ -d "$dir" ]; then
            count=$(grep -r "$schema\." "$dir" --include="*.go" --include="*.ts" --include="*.tsx" --include="*.js" --include="*.sql" 2>/dev/null | wc -l)
            if [ $count -gt 0 ]; then
                echo "  $dir: $count references"
            fi
        fi
    done
done

echo ""
echo "✅ Search complete!"
echo ""
echo "💡 To update code:"
echo "   1. Replace 'SCHEMA.metrics_registry' with 'public.metrics_registry WHERE schema_domain = \"SCHEMA\"'"
echo "   2. Replace 'SCHEMA.dax_functions' with 'public.dax_functions WHERE schema_domain = \"SCHEMA\"'"
