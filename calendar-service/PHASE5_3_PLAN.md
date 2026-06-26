# 📋 Phase 5.3 - Microsoft Outlook Integration & Production Deployment

**Status**: Ready to Begin  
**Date**: February 20, 2026  
**Objectives**: 
1. Integrate Microsoft OAuth (Outlook Calendar)
2. Implement multi-provider sync
3. Deploy to remote infrastructure
4. Production readiness verification

---

## 🎯 Phase 5.3 Timeline

| Phase | Task | Est. Time | Status |
|-------|------|-----------|--------|
| 5.3.1 | Microsoft OAuth Setup | 45 min | 🔵 Pending |
| 5.3.2 | Multi-Provider Sync | 60 min | 🔵 Pending |
| 5.3.3 | Cross-Provider Testing | 30 min | 🔵 Pending |
| 5.3.4 | Remote Deployment | 45 min | 🔵 Pending |
| 5.3.5 | Production Verification | 30 min | 🔵 Pending |

**Total Estimated**: 3.5 hours

---

## Phase 5.3.1: Microsoft OAuth Setup

### Objective
Integrate Microsoft OAuth provider for Outlook calendar sync, mirroring the Google implementation.

### Configuration Needed

**Azure App Registration (Get from user or create):**
```
MICROSOFT_CLIENT_ID=<from-azure-portal>
MICROSOFT_CLIENT_SECRET=<from-azure-portal>
MICROSOFT_TENANT_ID=common
MICROSOFT_REDIRECT_URL=http://localhost:9081/api/v1/oauth/microsoft/callback
```

### Implementation Steps

#### Step 1: Verify Microsoft OAuth Provider
**File**: `internal/oauth/microsoft_provider.go`
- [x] Structure defined
- [ ] Check implementation completeness
- [ ] Test token exchange flow

```bash
# Verify the provider structure
grep -n "MicrosoftOAuth2Provider" internal/oauth/microsoft_provider.go | head -20
```

#### Step 2: Register Microsoft Routes
**File**: `internal/api/router.go`
- [ ] Check if Microsoft routes are registered
- [ ] Verify `GET /api/v1/sync/microsoft/auth-url-pkce`
- [ ] Verify `GET /api/v1/oauth/microsoft/callback`

```bash
# Check current routes
grep -n "microsoft\|Microsoft" internal/api/router.go
```

#### Step 3: Create Microsoft Sync Handler
**File**: `internal/api/microsoft_handlers.go`
- [ ] Check if handler exists
- [ ] Verify implementation mirrors Google handler
- [ ] Test POST `/api/v1/sync/microsoft`

```bash
# Check if file exists
ls -la internal/api/microsoft_handlers.go
```

### Testing Microsoft OAuth

```bash
# 1. Get Microsoft auth URL
TOKEN=$(./bin/jwt_gen)
curl -s "http://localhost:9081/api/v1/sync/microsoft/auth-url-pkce?user_id=user1&tenant_id=tenant1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant1" | jq '.auth_url'

# 2. User opens URL → Authenticates with Microsoft
# 3. Service receives callback
# 4. Token stored in database
```

**Expected Result**: Valid Microsoft OAuth auth URL with code_challenge, state, and proper scopes

---

## Phase 5.3.2: Multi-Provider Sync

### Objective
Enable users to sync from both Google and Microsoft calendar providers simultaneously.

### Implementation

#### Step 1: Update Sync Request Model
**File**: `internal/api/sync_handler.go`
```go
type MultiProviderSyncRequest struct {
    UserID      string   `json:"user_id" binding:"required"`
    TenantID    string   `json:"tenant_id" binding:"required"`
    Providers   []string `json:"providers" binding:"required"` // ["google", "microsoft"]
}

type MultiProviderSyncResponse struct {
    Google    *SyncResult `json:"google,omitempty"`
    Microsoft *SyncResult `json:"microsoft,omitempty"`
    Status    string      `json:"status"`
}
```

#### Step 2: Implement Concurrent Sync
**File**: `internal/sync/processor.go`
```go
func (p *MultiSyncProcessor) SyncAllProviders(ctx context.Context, userID, tenantID string, providers []string) (*MultiProviderSyncResponse, error) {
    results := &MultiProviderSyncResponse{}
    
    // Run syncs concurrently
    for _, provider := range providers {
        switch provider {
        case "google":
            go p.syncGoogle(ctx, userID, tenantID, results)
        case "microsoft":
            go p.syncMicrosoft(ctx, userID, tenantID, results)
        }
    }
    
    return results, nil
}
```

#### Step 3: Register Multi-Provider Endpoints
**File**: `internal/api/router.go`
```go
// POST /api/v1/sync/all - Sync all configured providers
v1.POST("/sync/all", jwtMiddleware, multiSyncHandler.SyncAll)

// GET /api/v1/sync/providers - List available providers for user
v1.GET("/sync/providers", jwtMiddleware, multiSyncHandler.ListProviders)

// GET /api/v1/sync/history - Get sync history for all providers
v1.GET("/sync/history", jwtMiddleware, multiSyncHandler.GetHistory)
```

### Testing Multi-Provider Sync

```bash
# 1. Login with both Google and Microsoft
# 2. Trigger multi-provider sync
curl -X POST http://localhost:9081/api/v1/sync/all \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant1" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user1", "tenant_id":"tenant1", "providers":["google","microsoft"]}'

# 3. Check results
curl http://localhost:9081/api/v1/sync/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant1"
```

**Expected Result**: Both Google and Microsoft calendars synced concurrently with merged results

---

## Phase 5.3.3: Cross-Provider Testing

### Test Scenarios

#### Test 1: Google-Only Sync
```bash
# Prerequisites: User authenticated with Google
TOKEN=$(./bin/jwt_gen)

curl -X POST http://localhost:9081/api/v1/sync/google \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: test" | jq .

# Verify in database
psql -h localhost -U postgres -d alpha -c \
  "SELECT * FROM google_sync_results ORDER BY created_at DESC LIMIT 1;"
```

**Expected**: 
- ✅ Sync completes successfully
- ✅ Events stored in database
- ✅ Results show event count

#### Test 2: Microsoft-Only Sync
```bash
curl -X POST http://localhost:9081/api/v1/sync/microsoft \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: test" | jq .

# Verify in database
psql -h localhost -U postgres -d alpha -c \
  "SELECT * FROM microsoft_sync_results ORDER BY created_at DESC LIMIT 1;"
```

**Expected**: 
- ✅ Microsoft sync completes
- ✅ Outlook events stored
- ✅ Results show event count

#### Test 3: Dual-Provider Sync
```bash
# Sync both providers
curl -X POST http://localhost:9081/api/v1/sync/all \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: test" \
  -d '{"user_id":"test","tenant_id":"test","providers":["google","microsoft"]}'

# Verify both providers synced
psql -h localhost -U postgres -d alpha -c \
  "SELECT provider, sync_status, events_synced FROM sync_results ORDER BY created_at DESC LIMIT 2;"
```

**Expected**: 
- ✅ Both syncs start concurrently
- ✅ Both complete successfully
- ✅ Events from both providers available

#### Test 4: Token Refresh
```bash
# Check token age and refresh if needed
psql -h localhost -U postgres -d alpha -c \
  "SELECT user_id, provider, expires_at FROM oauth_tokens WHERE user_id='test';"

# Verify refresh token works
curl http://localhost:9081/api/v1/oauth/refresh \
  -H "Authorization: Bearer $TOKEN" | jq .
```

**Expected**: 
- ✅ Tokens shown with expiration times
- ✅ Auto-refresh works before expiry
- ✅ New token silently obtained

#### Test 5: Error Recovery
```bash
# Test with invalid/expired token
curl http://localhost:9081/api/v1/sync/google \
  -H "Authorization: Bearer invalid-token"

# Should get 401 with helpful error message
```

**Expected**: 
- ✅ Returns 401 Unauthorized
- ✅ Clear error message
- ✅ Suggests re-authentication

---

## Phase 5.3.4: Remote Deployment

### Deployment Target
```
Server: 100.84.126.19
Port: 9081
Environment: production
Protocol: HTTPS (with valid cert)
```

### Pre-Deployment Checklist

- [ ] Production TLS certificate ready
- [ ] Remote PostgreSQL credentials configured
- [ ] Remote Redis instance operational
- [ ] Secrets management setup (if any)
- [ ] Monitoring/logging configured
- [ ] Rollback plan documented

### Deployment Steps

#### Step 1: Build Production Binary
```bash
cd /Users/eganpj/GitHub/semlayer/calendar-service

# Full rebuild with optimization
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w" \
  -o bin/calendar-service-prod ./cmd/server

# Verify binary
file bin/calendar-service-prod
ls -lh bin/calendar-service-prod
```

#### Step 2: Configure Remote Environment
```bash
# Create .env.production
cat > .env.production << 'EOF'
# Production Configuration
SERVER_PORT=9081
LOG_LEVEL=info

# Google OAuth
GOOGLE_CLIENT_ID=<production-client-id>
GOOGLE_CLIENT_SECRET=<production-client-secret>
GOOGLE_REDIRECT_URL=https://api.calendar.yourdomain.com/api/v1/oauth/google/callback

# Microsoft OAuth
MICROSOFT_CLIENT_ID=<production-client-id>
MICROSOFT_CLIENT_SECRET=<production-client-secret>
MICROSOFT_REDIRECT_URL=https://api.calendar.yourdomain.com/api/v1/oauth/microsoft/callback

# Database (Remote)
POSTGRES_HOST=db.production.internal
POSTGRES_PORT=5432
POSTGRES_USER=calendar_user
POSTGRES_PASSWORD=<secure-password>
POSTGRES_DB=calendar_service

# Redis (Remote)
REDIS_URL=redis://:password@cache.production.internal:6379/0

# JWT Secret
JWT_SECRET=<production-jwt-secret>

# Hasura
HASURA_ENDPOINT=http://hasura.production.internal:8080/v1/graphql
HASURA_ADMIN_SECRET=<hasura-secret>
EOF
```

#### Step 3: SCP to Remote Server
```bash
# Copy binary and config
scp bin/calendar-service-prod user@100.84.126.19:/opt/calendar-service/
scp .env.production user@100.84.126.19:/opt/calendar-service/.env

# Verify permissions
ssh user@100.84.126.19 'chmod +x /opt/calendar-service/calendar-service-prod'
```

#### Step 4: Start on Remote Server
```bash
ssh user@100.84.126.19 << 'EOF'
cd /opt/calendar-service

# Create systemd service (or use supervisor/docker)
cat > /tmp/calendar-service.service << 'SYSCTL'
[Unit]
Description=Calendar Service
After=network.target

[Service]
Type=simple
User=calendar
ExecStart=/opt/calendar-service/calendar-service-prod -port 9081
Environment="HOME=/opt/calendar-service"
EnvironmentFile=/opt/calendar-service/.env
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
SYSCTL

sudo mv /tmp/calendar-service.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable calendar-service
sudo systemctl start calendar-service

# Check status
sudo systemctl status calendar-service
EOF
```

#### Step 5: Setup TLS Proxy (Nginx)
```bash
ssh user@100.84.126.19 << 'EOF'
# Create nginx config
cat > /etc/nginx/sites-available/calendar-api << 'NGINX'
server {
    listen 443 ssl http2;
    server_name api.calendar.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/api.calendar.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.calendar.yourdomain.com/privkey.pem;

    location /api/v1/ {
        proxy_pass http://localhost:9081/api/v1/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 80;
    server_name api.calendar.yourdomain.com;
    return 301 https://$server_name$request_uri;
}
NGINX

ln -s /etc/nginx/sites-available/calendar-api /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
EOF
```

---

## Phase 5.3.5: Production Verification

### Health Checks

```bash
# 1. Service is running
curl -X GET https://100.84.126.19/api/v1/health

# Expected: {"status":"healthy"}

# 2. Database connection
curl -X GET https://100.84.126.19/api/v1/health/db

# Expected: {"status":"connected"}

# 3. OAuth providers initialized
curl -X GET https://100.84.126.19/api/v1/health/providers

# Expected: {"google":true,"microsoft":true}

# 4. Cache operational
curl -X GET https://100.84.126.19/api/v1/health/cache

# Expected: {"status":"connected"}
```

### Load Testing

```bash
# Generate some load
ab -n 100 -c 10 https://100.84.126.19/api/v1/health

# Expected: 100% success rate, <100ms response time
```

### Security Verification

```bash
# Check TLS certificate
echo | openssl s_client -servername 100.84.126.19 -connect 100.84.126.19:443 2>/dev/null | openssl x509 -text

# Verify JWT authentication required
curl https://100.84.126.19/api/v1/sync/google/auth-url-pkce

# Expected: 401 Unauthorized (no JWT)
```

### Monitoring Setup

- [ ] Configure log aggregation (ELK/Splunk/etc)
- [ ] Setup alerts for errors
- [ ] Monitor response times
- [ ] Track sync success/failure rates
- [ ] Monitor resource usage (CPU, memory, disk)

---

## 📊 Verification Checklist

### Google OAuth (Already Complete) ✅
- [x] Real credentials integrated
- [x] Auth URL generation working
- [x] JWT token generation working
- [x] PKCE flow verified
- [x] Service running locally

### Microsoft OAuth (In Progress)
- [ ] Credentials obtained from Azure
- [ ] Microsoft provider implementation verified
- [ ] Auth URL generation working
- [ ] Token exchange working
- [ ] Callback handling working

### Multi-Provider Support (Pending)
- [ ] Concurrent sync implementation
- [ ] Error handling across providers
- [ ] Results aggregation
- [ ] UI/API supports dual providers

### Production Ready (Pending)
- [ ] Binary builds for Linux
- [ ] Environment configuration templates
- [ ] Deployment scripts
- [ ] Monitoring configured
- [ ] Health checks passing
- [ ] TLS certificate valid

---

## 🚀 Quick Commands

### Build & Deploy
```bash
# 1. Build for production
cd /Users/eganpj/GitHub/semlayer/calendar-service
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o bin/calendar-service-prod ./cmd/server

# 2. Copy to remote
scp bin/calendar-service-prod user@100.84.126.19:/opt/calendar-service/

# 3. Start on remote
ssh user@100.84.126.19 '/opt/calendar-service/calendar-service-prod -port 9081'
```

### Testing
```bash
# Generate JWT
TOKEN=$(./bin/jwt_gen)

# Test Google
curl -s http://localhost:9081/api/v1/sync/google/auth-url-pkce \
  -H "Authorization: Bearer $TOKEN" | jq '.auth_url'

# Test Microsoft (when ready)
curl -s http://localhost:9081/api/v1/sync/microsoft/auth-url-pkce \
  -H "Authorization: Bearer $TOKEN" | jq '.auth_url'

# Check sync status
curl -s http://localhost:9081/api/v1/sync/status \
  -H "Authorization: Bearer $TOKEN" | jq .
```

---

## 📝 Success Criteria

Phase 5.3 is complete when:

✅ Microsoft OAuth fully integrated and tested  
✅ Users can sync from both Google and Microsoft simultaneously  
✅ Service running on remote server (100.84.126.19)  
✅ HTTPS working with valid certificate  
✅ Production health checks passing  
✅ Deployment automated and documented  
✅ Monitoring and alerts configured  

---

## 🎯 Next: Phase 5.4 (Post-Launch)

After Phase 5.3 completes:
- User acceptance testing
- Performance optimization
- Security audit
- Compliance verification (if needed)
- User training & documentation

---

*Phase 5.3 Implementation Plan - Ready to Execute*
