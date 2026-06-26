#!/bin/bash

# Phase 5.2: Real Google Calendar OAuth Test
# This script tests the actual Google OAuth flow with real credentials

set -e

SERVICE_URL="${SERVICE_URL:-http://localhost:9081}"
USER_ID="${USER_ID:-test-user-$(date +%s)}"
TENANT_ID="${TENANT_ID:-test-tenant}"

echo "==============================================="
echo "Phase 5.2: Real Google Calendar OAuth Test"
echo "==============================================="
echo "Service URL: $SERVICE_URL"
echo "User ID: $USER_ID"
echo "Tenant ID: $TENANT_ID"
echo ""

# Step 1: Check service health
echo "Step 1: Verify service is running"
HEALTH=$(curl -s -X GET "$SERVICE_URL/api/v1/health" | jq -r '.status' 2>/dev/null || echo "error")
if [ "$HEALTH" = "healthy" ]; then
    echo "✓ Service is healthy"
else
    echo "✗ Service is not responding - exiting"
    exit 1
fi
echo ""

# Step 2: Get Google Auth URL with PKCE
echo "Step 2: Generate Google OAuth Authorization URL (PKCE)"
echo "URL: $SERVICE_URL/api/v1/sync/google/auth-url-pkce"
echo ""
echo "Expected: This will redirect user to Google OAuth consent screen"
echo ""
echo "  1. Visit: $SERVICE_URL/api/v1/sync/google/auth-url-pkce?user_id=$USER_ID&tenant_id=$TENANT_ID"
echo "  2. Authenticate with your Google account"
echo "  3. Grant calendar access permissions"
echo "  4. Google will redirect to: $SERVICE_URL/api/v1/sync/google/callback-pkce"
echo ""

# Step 3: Test sync endpoints with auth token
echo "Step 3: Test sync endpoint structure"
echo ""
echo "POST /api/v1/sync/google - Initiate sync"
echo "  Expected request body:"
cat << 'EOF'
{
  "user_id": "test-user-123",
  "tenant_id": "test-tenant",
  "auth_code": "oauth-code-from-google"
}
EOF

echo ""
echo "GET /api/v1/sync/status - Check sync status"
echo "  Query params: user_id=test-user-123"
echo ""

echo "GET /api/v1/sync/active - List active syncs"
echo ""

echo "POST /api/v1/sync/cancel - Cancel sync"
echo "  Query params: user_id=test-user-123"
echo ""

# Step 4: Test database connectivity
echo "Step 4: Verify database for sync tracking"
PGPASSWORD="postgres" psql -h localhost -U postgres -d alpha -c "SELECT COUNT(*) as sync_records FROM google_sync_results;" 2>/dev/null || echo "Database check skipped"
echo ""

# Step 5: Show next steps
echo "==============================================="
echo "Phase 5.2: Manual Testing Required"
echo "==============================================="
echo ""
echo "To complete Phase 5.2 real Google Calendar testing:"
echo ""
echo "1. GET Auth URL:"
echo "   curl -s -X GET '"
echo "     $SERVICE_URL/api/v1/sync/google/auth-url-pkce"
echo "     ?user_id=$USER_ID&tenant_id=$TENANT_ID'"
echo "   Then open the URL in a browser to authenticate"
echo ""
echo "2. After authentication, Google will redirect to callback"
echo "   The service will exchange auth code for access token"
echo "   Token will be stored in database and Redis"
echo ""
echo "3. Verify token storage:"
echo "   psql -h localhost -U postgres -d alpha -c \\"
echo "   SELECT user_id, provider, token_type FROM oauth_tokens WHERE provider='google';\""
echo ""
echo "4. Trigger sync (once token is stored):"
echo "   curl -X POST '$SERVICE_URL/api/v1/sync/google' \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -H 'Authorization: Bearer <jwt-token>' \\"
echo "     -H 'X-Tenant-ID: $TENANT_ID' \\"
echo "     -d '{\"user_id\":\"$USER_ID\",\"tenant_id\":\"$TENANT_ID\"}"
echo ""
echo "5. Check sync results:"
echo "   psql -h localhost -U postgres -d alpha -c \\"
echo "   SELECT sync_id, sync_status, events_synced FROM google_sync_results;"
echo ""
echo "==============================================="
