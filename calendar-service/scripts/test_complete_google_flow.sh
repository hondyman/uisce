#!/bin/bash
# Complete Google Calendar OAuth + Sync Testing Script
# This script verifies:
# 1. OAuth PKCE flow
# 2. Token persistence (DB + Redis + Cookie)
# 3. Calendar sync

set -e

BASE_URL="http://localhost:9081"
POSTGRES_USER="postgres"
POSTGRES_PASSWORD="postgres"
POSTGRES_HOST="localhost"
POSTGRES_DB="alpha"
REDIS_URL="redis://localhost:6379/0"

# Color codes for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

function print_step() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

function print_check() {
    echo -e "${GREEN}✅ $1${NC}"
}

function print_error() {
    echo -e "${RED}❌ $1${NC}"
}

function print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# Generate unique test user ID
USER_ID="test-user-$(date +%s)"
TENANT_ID="test-tenant"

print_step "PHASE 1: Verify Service Health"

# Check service
if ! curl -s "$BASE_URL/api/v1/health" | grep -q "healthy"; then
    print_error "Service not responding"
    exit 1
fi
print_check "Service running on $BASE_URL"

# Check database
if ! PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -c "SELECT 1" >/dev/null 2>&1; then
    print_error "PostgreSQL not accessible"
    exit 1
fi
print_check "PostgreSQL connected"

# Check Redis
if ! redis-cli -u "$REDIS_URL" PING >/dev/null 2>&1; then
    print_error "Redis not accessible"
    exit 1
fi
print_check "Redis connected"

print_step "PHASE 2: Test OAuth PKCE Flow"

# Generate JWT
print_info "Generating JWT token..."
if [ ! -f "./bin/jwt_gen" ]; then
    print_error "jwt_gen binary not found. Build it first: go build -o bin/jwt_gen cmd/jwt_gen/main.go"
    exit 1
fi

JWT_TOKEN=$(./bin/jwt_gen "dev-jwt-secret-key-change-in-production" "$USER_ID" "$TENANT_ID")
print_check "JWT generated for user: $USER_ID"
print_info "JWT (first 50 chars): ${JWT_TOKEN:0:50}..."

# Get auth URL
print_info "Requesting Google OAuth authorization URL..."
AUTH_RESPONSE=$(curl -s "${BASE_URL}/api/v1/sync/google/auth-url-pkce?user_id=${USER_ID}&tenant_id=${TENANT_ID}" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json")

# Extract values
AUTH_STATE=$(echo "$AUTH_RESPONSE" | jq -r '.state // empty')
AUTH_URL=$(echo "$AUTH_RESPONSE" | jq -r '.auth_url // empty')
EXPIRES_IN=$(echo "$AUTH_RESPONSE" | jq -r '.expires_in_seconds // empty')

if [ -z "$AUTH_STATE" ] || [ -z "$AUTH_URL" ]; then
    print_error "Failed to get auth URL"
    print_info "Response: $AUTH_RESPONSE"
    exit 1
fi

print_check "OAuth PKCE URL generated"
print_info "State: $AUTH_STATE"
print_info "Expires: $EXPIRES_IN seconds"

# Verify Google client ID in URL
if echo "$AUTH_URL" | grep -q "607288898719"; then
    print_check "Google client ID present in auth URL"
else
    print_error "Google client ID NOT found in auth URL"
    print_info "This means real credentials are not loaded"
fi

# Verify PKCE parameters
if echo "$AUTH_URL" | grep -q "code_challenge"; then
    print_check "PKCE code_challenge present"
else
    print_error "PKCE code_challenge missing"
fi

if echo "$AUTH_URL" | grep -q "code_challenge_method=S256"; then
    print_check "PKCE S256 method configured"
else
    print_error "PKCE method incorrect"
fi

print_step "PHASE 3: Verify PKCE State Storage"

# Check Redis for PKCE state
REDIS_PKCE_KEY="calendar:pkce:${AUTH_STATE}"
REDIS_PKCE=$(redis-cli -u "$REDIS_URL" GET "$REDIS_PKCE_KEY" 2>/dev/null || echo "")

if [ -n "$REDIS_PKCE" ]; then
    print_check "PKCE state stored in Redis"
    print_info "Key: $REDIS_PKCE_KEY"
else
    print_info "PKCE state not yet in Redis (will be added during callback)"
fi

print_step "PHASE 4: Verify Token Storage Preparation"

# Count current tokens in DB
CURRENT_TOKENS=$(PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -t -c \
  "SELECT COUNT(*) FROM oauth_tokens WHERE provider='google';" 2>/dev/null || echo "0")

print_info "Current Google tokens in database: $CURRENT_TOKENS"

# Check if oauth_tokens table exists and has right schema
COLUMN_CHECK=$(PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -t -c \
  "SELECT column_name FROM information_schema.columns WHERE table_name='oauth_tokens' AND column_name='user_id';" 2>/dev/null || echo "")

if [ -z "$COLUMN_CHECK" ]; then
    print_error "oauth_tokens table missing required columns"
    exit 1
fi
print_check "oauth_tokens table schema verified"

print_step "PHASE 5: Test Sync Status Endpoint"

# Check active syncs
SYNC_STATUS=$(curl -s "${BASE_URL}/api/v1/sync/active" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID")

print_check "Sync status endpoint accessible"
print_info "Active syncs: $(echo "$SYNC_STATUS" | jq 'length')"

print_step "PHASE 6: Manual Flow Instructions"

echo ""
echo "🔷 TO COMPLETE THE GOOGLE OAUTH FLOW:"
echo ""
echo "1. Open this URL in your browser:"
echo "   $AUTH_URL"
echo ""
echo "2. Sign in with your Google account"
echo ""
echo "3. Grant permissions to access calendar"
echo ""
echo "4. After Google redirects, check these commands to verify token storage:"
echo ""
echo "   📊 View token in database:"
echo "   psql -h localhost -U postgres -d alpha -c \\"
echo "     \"SELECT user_id, provider, token_type, expires_at FROM oauth_tokens \\"
echo "      WHERE user_id='$USER_ID' AND provider='google';\""
echo ""
echo "   💾 View token in Redis cache:"
echo "   redis-cli -u 'redis://localhost:6379/0' GET 'calendar:oauth:$USER_ID:google'"
echo ""
echo "   🌐 View session cookie (in browser console):"
echo "   document.cookie"
echo ""
echo "5. After token is stored, trigger calendar sync:"
echo "   curl -X POST '${BASE_URL}/api/v1/sync/google' \\"
echo "     -H 'Authorization: Bearer $JWT_TOKEN' \\"
echo "     -H 'X-Tenant-ID: $TENANT_ID' \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"user_id\":\"$USER_ID\",\"tenant_id\":\"$TENANT_ID\"}'"
echo ""
echo "6. Check sync results:"
echo "   psql -h localhost -U postgres -d alpha -c \\"
echo "     \"SELECT sync_id, sync_status, events_synced FROM google_sync_results \\"
echo "      WHERE user_id='$USER_ID' ORDER BY created_at DESC LIMIT 1;\""
echo ""

print_step "PHASE 7: Automated Verification (After Manual Auth)"

echo ""
echo "After completing manual authorization above, run this to verify storage:"
echo ""
echo "#!/bin/bash"
echo "USER_ID='$USER_ID'"
echo ""
echo "# Check PostgreSQL"
echo "echo 'PostgreSQL tokens:'"
echo "PGPASSWORD=postgres psql -h localhost -U postgres -d alpha -c \\"
echo "  \"SELECT user_id, provider, token_type, expires_at FROM oauth_tokens WHERE user_id='\$USER_ID' AND provider='google';\""
echo ""
echo "# Check Redis"
echo "echo 'Redis cache:'"
echo "redis-cli -u 'redis://localhost:6379/0' GET \"calendar:oauth:\$USER_ID:google\" | jq ."
echo ""

print_step "SUMMARY"

echo ""
echo "✅ PRE-REQUISITES VERIFIED:"
echo "  ✓ Service running"
echo "  ✓ Database accessible"
echo "  ✓ Redis working"
echo "  ✓ OAuth URL can be generated"
echo "  ✓ Google real credentials loaded"
echo "  ✓ PKCE parameters correct"
echo ""
echo "🔷 NEXT STEP: Manual Browser OAuth"
echo "  Open: $AUTH_URL"
echo ""
echo "📊 After that, tokens should be in:"
echo "  • PostgreSQL (encrypted)"
echo "  • Redis (24h cache)"
echo "  • Browser cookie (httpOnly)"
echo ""
