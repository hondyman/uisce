#!/bin/bash
# Post-Authentication Verification Script
# Run this AFTER you've completed the Google OAuth flow in the browser

set -e

POSTGRES_USER="postgres"
POSTGRES_PASSWORD="postgres"
POSTGRES_HOST="localhost"
POSTGRES_DB="alpha"
REDIS_URL="redis://localhost:6379/0"

# Get user ID (will be provided or use default)
USER_ID="${1:-test-user}"

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}POST-AUTHENTICATION VERIFICATION${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# 1. Check PostgreSQL for token
echo -e "${YELLOW}📊 Checking PostgreSQL oauth_tokens table...${NC}"
DB_TOKEN=$(PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -t -c \
  "SELECT COUNT(*) FROM oauth_tokens WHERE user_id LIKE '%$USER_ID%' AND provider='google';" 2>/dev/null || echo "0")

if [ "$DB_TOKEN" -gt "0" ]; then
    echo -e "${GREEN}✅ Token found in PostgreSQL ($DB_TOKEN records)${NC}"
    
    # Show details
    echo ""
    echo "Token details:"
    PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -c \
      "SELECT user_id, provider, token_type, expires_at, LENGTH(access_token) as token_length FROM oauth_tokens 
       WHERE user_id LIKE '%$USER_ID%' AND provider='google' 
       ORDER BY created_at DESC LIMIT 1" || true
else
    echo -e "${RED}❌ No token found in PostgreSQL${NC}"
    echo -e "${YELLOW}   This means the OAuth callback didn't complete successfully${NC}"
fi
echo ""

# 2. Check Redis cache
echo -e "${YELLOW}💾 Checking Redis cache...${NC}"
for REDIS_KEY in $(redis-cli -u "$REDIS_URL" KEYS "calendar:oauth:*:google" 2>/dev/null); do
    echo -e "${GREEN}✅ Found cached token: $REDIS_KEY${NC}"
    REDIS_TOKEN=$(redis-cli -u "$REDIS_URL" GET "$REDIS_KEY" 2>/dev/null | head -c 100)
    echo "   Value (first 100 chars): $REDIS_TOKEN..."
    
    TTL=$(redis-cli -u "$REDIS_URL" TTL "$REDIS_KEY" 2>/dev/null)
    echo "   TTL: $TTL seconds (~$(( TTL / 3600 )) hours)"
done

if [ -z "$(redis-cli -u "$REDIS_URL" KEYS "calendar:oauth:*:google" 2>/dev/null)" ]; then
    echo -e "${RED}❌ No cached tokens in Redis${NC}"
    echo -e "${YELLOW}   Note: Redis cache is lazy - it's populated on first sync request${NC}"
fi
echo ""

# 3. Check for sync results
echo -e "${YELLOW}🔄 Checking calendar sync results...${NC}"
SYNC_COUNT=$(PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -t -c \
  "SELECT COUNT(*) FROM google_sync_results WHERE user_id LIKE '%$USER_ID%';" 2>/dev/null || echo "0")

if [ "$SYNC_COUNT" -gt "0" ]; then
    echo -e "${GREEN}✅ Found $SYNC_COUNT sync results${NC}"
    echo ""
    PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -c \
      "SELECT sync_id, sync_status, events_synced, started_at, completed_at FROM google_sync_results 
       WHERE user_id LIKE '%$USER_ID%' 
       ORDER BY created_at DESC LIMIT 3" || true
else
    echo -e "${YELLOW}ℹ️  No sync results yet (run sync to generate)${NC}"
fi
echo ""

# 4. Show what database tables exist
echo -e "${YELLOW}📋 Database schema summary:${NC}"
PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -t -c \
  "SELECT tablename FROM pg_tables WHERE schemaname='public' 
   AND tablename IN ('oauth_tokens', 'google_sync_results', 'microsoft_sync_results');" || true
echo ""

# 5. Summary
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}VERIFICATION COMPLETE${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if [ "$DB_TOKEN" -gt "0" ]; then
    echo -e "${GREEN}✅ Authentication appears successful!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Test calendar sync:"
    echo "   curl -X POST 'http://localhost:9081/api/v1/sync/google' \\"
    echo "     -H 'Authorization: Bearer \$(./bin/jwt_gen)' \\"
    echo "     -H 'Content-Type: application/json' \\"
    echo "     -d '{\"user_id\":\"$USER_ID\",\"tenant_id\":\"test-tenant\"}'"
    echo ""
    echo "2. Check sync results again:"
    echo "   scripts/test_verify_auth.sh '$USER_ID'"
else
    echo -e "${RED}❌ Authentication setup incomplete${NC}"
    echo ""
    echo "Troubleshooting:"
    echo "1. Check service logs: tail -50 /tmp/calendar-service.log"
    echo "2. Verify service is running: curl http://localhost:9081/api/v1/health"
    echo "3. Make sure you completed the browser OAuth flow"
fi
echo ""
