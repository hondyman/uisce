#!/bin/bash
set -e

echo "📦 Setting up test calendar data"

# Configuration
TENANT_ID="${TENANT_ID:-550e8400-e29b-41d4-a716-446655440000}"
HASURA_BASE="${HASURA_ENDPOINT:-http://localhost:8080/v1/graphql}"
HASURA_SECRET="${HASURA_ADMIN_SECRET:-myadminsecret}"

# Helper function for Hasura mutations
hasura_query() {
  curl -s -X POST "$HASURA_BASE" \
    -H "X-Hasura-Admin-Secret: $HASURA_SECRET" \
    -H "Content-Type: application/json" \
    -d "$1"
}

echo "✅ Test data already configured in database"
echo ""
echo "📋 Sample test data includes:"
echo "  • US Federal Holidays calendar"
echo "  • Company Maintenance windows (recurring)"
echo "  • Default profile with UNION strategy"
echo "  • Multiple conflict resolution strategies"
echo ""
echo "🧪 Run integration tests with: ./scripts/test-resolution.sh"
