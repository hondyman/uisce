# Phase 5.2 Google Calendar Integration - Testing Guide

## ✅ Phase 5.2 Status: READY FOR MANUAL TESTING

**Real Google OAuth Credentials Configured:**
- ✅ GOOGLE_CLIENT_ID: 607288898719-qkpbcdrdjshm55112pr9ld7h29c33u73.apps.googleusercontent.com
- ✅ GOOGLE_CLIENT_SECRET: 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
- ✅ GOOGLE_REDIRECT_URL: http://localhost:9081/api/v1/oauth/google/callback

## Manual Testing Steps

### Step 1: Generate JWT Token
```bash
cd /Users/eganpj/GitHub/semlayer/calendar-service

# Use Go to generate a proper JWT with all required claims
go build -o bin/jwt_gen cmd/jwt_gen/main.go
TOKEN=$(./bin/jwt_gen "dev-jwt-secret-key-change-in-production" "your-user-id" "your-tenant-id")
echo $TOKEN  # Save this for subsequent API calls
```

### Step 2: Get Google OAuth Authorization URL
```bash
# Get the authorization URL (user needs to visit this in browser)
curl -s "http://localhost:9081/api/v1/sync/google/auth-url-pkce?user_id=your-user-id&tenant_id=your-tenant-id" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: your-tenant-id" | jq .auth_url

# Copy the auth_url and open in browser to authenticate
```

### Step 3: Authenticate with Google
1. Visit the auth_url from Step 2
2. Sign in with your Google account
3. Grant calendar access permissions
4. You'll be redirected to the callback endpoint
5. The service will exchange the auth_code for tokens

### Step 4: Verify Token Storage
```bash
# Check if token was saved in database
psql -h localhost -U postgres -d alpha -c \
  "SELECT user_id, provider, token_type, expires_at FROM oauth_tokens WHERE provider='google';"

# Verify in Redis (if available)
redis-cli -u "redis://localhost:6379/0" GET "calendar:oauth:user-id:google"
```

### Step 5: Trigger Calendar Sync
```bash
# Generate new token (with proper user_id if different)
TOKEN=$(./bin/jwt_gen "dev-jwt-secret-key-change-in-production" "your-user-id" "your-tenant-id")

# Initiate sync
curl -X POST "http://localhost:9081/api/v1/sync/google" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: your-tenant-id" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "your-user-id",
    "tenant_id": "your-tenant-id"
  }'
```

### Step 6: Check Sync Results
```bash
# View sync results
psql -h localhost -U postgres -d alpha -c \
  "SELECT sync_id, sync_status, events_synced, errors, completed_at FROM google_sync_results ORDER BY created_at DESC LIMIT 5;"

# Check active syncs
curl -s "http://localhost:9081/api/v1/sync/active" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: your-tenant-id" | jq .
```

## JWT Token Generation Helper

The service requires JWT tokens with these claims:
- `user_id`: Unique user identifier
- `tenant_id`: Multi-tenant identifier  
- `tenant_ids[]`: Array of tenant IDs user can access
- `email`: User email
- `iat`: Issued at timestamp
- `exp`: Expiration timestamp

Use the JWT generator:
```bash
./bin/jwt_gen <secret> <user_id> <tenant_id>
```

**Default values:**
- Secret: `dev-jwt-secret-key-change-in-production`
- User ID: `test-user-phase5-2`
- Tenant ID: `test-tenant`

## Phase 5.2 Architecture

```
User Browser
    ↓ (clicks auth URL)
Google Consent Screen
    ↓ (grants access)
OAuth Callback (localhost:9081/api/v1/oauth/google/callback)
    ↓
Exchange Auth Code for Token
    ↓
Save Token (Database + Redis)
    ↓
Service Can Now Access Google Calendar
    ↓
Sync Events (creates google_sync_results records)
    ↓
Store in Database (for persistence)
```

## Config Verification

To verify Phase 5.2 is properly configured:

```bash
# Check env file
grep GOOGLE .env.local

# Verify service health
curl http://localhost:9081/api/v1/health

# Test OAuth endpoint (requires JWT)
curl "http://localhost:9081/api/v1/sync/google/auth-url-pkce?user_id=test&tenant_id=test" \
  -H "Authorization: Bearer <JWT>" \
  -H "X-Tenant-ID: test"
```

## Troubleshooting

**"Invalid token" response:**
- Ensure JWT secret matches: `dev-jwt-secret-key-change-in-production`
- Verify all required claims are present: `user_id`, `tenant_id`, `tenant_ids` array
- Check token hasn't expired

**"Authorization header required":**
- Make sure to include `Authorization: Bearer <token>` header
- Token must be from `./bin/jwt_gen` with matching secret

**Google OAuth errors:**
- Verify credentials in `.env.local`:
  - GOOGLE_CLIENT_ID=607288898719-qkpbcdrdjshm55112pr9ld7h29c33u73.apps.googleusercontent.com
  - GOOGLE_CLIENT_SECRET=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
- Ensure redirect URL is registered in Google Cloud Console

**No events synced:**
- Check if token is valid (not expired)
- Verify calendar has events
- Check logs for sync errors in database

## Next Steps for Phase 5.3

1. **Microsoft Outlook Integration**
   - Complete MICROSOFT_CLIENT_ID configuration
   - Test Outlook calendar sync
   - Verify token persistence for Microsoft

2. **Token Encryption**
   - Generate AES-256 key: `openssl rand -base64 32`
   - Set OAUTH_TOKEN_ENCRYPTION_KEY in .env
   - Enable encryption in OAuth provider

3. **Performance Optimization**
   - Deploy Redis Cluster for high availability
   - Set up Redpanda for CDC event streaming
   - Configure cache invalidation strategies

4. **Production Deployment**
   - Deploy to 100.84.126.19 remote infrastructure
   - Set up remote PostgreSQL and Redis
   - Configure cross-network sync operations
   - Monitor sync metrics and performance

## References

- **Google Calendar API**: https://developers.google.com/calendar/api
- **OAuth 2.0 PKCE**: https://tools.ietf.org/html/rfc7636
- **Calendar Service Architecture**: [PHASE5_INTEGRATION_CHECKLIST.md](PHASE5_INTEGRATION_CHECKLIST.md)
- **Database Migrations**: [GOOGLE_SYNC_MIGRATIONS.md](GOOGLE_SYNC_MIGRATIONS.md)
