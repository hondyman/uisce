# 404 Error Fix: /api/entity-schema Endpoint

## Problem

The frontend was receiving a **404 Not Found** error when attempting to call the `/api/entity-schema` endpoint:

```
Request URL: http://localhost:8001/api/entity-schema?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0
Status Code: 404 Not Found
```

## Root Cause

The issue was in the **API Gateway** (`api-gateway/main.go`), not the backend. Here's why:

1. **Backend Implementation**: The `/api/entity-schema` endpoint is properly implemented in `backend/internal/api/api.go` (lines 1195-1340):
   - `GET /api/entity-schema` - Retrieves entity schema for a tenant/datasource
   - `POST /api/entity-schema` - Saves entity schema configuration

2. **API Gateway**: The local development setup routes requests through an API Gateway running on port 8001, which proxies to the backend on port 8080.

3. **Missing Proxy Route**: The API Gateway (`api-gateway/main.go`) explicitly registers which backend endpoints to proxy. The `/entity-schema` routes were **not registered**, causing the gateway to return 404 instead of forwarding the request to the backend.

## Architecture Flow

```
Frontend (http://localhost:5173)
    ↓
API Gateway (http://localhost:8001)  ← Returns 404 because route not registered
    ↓
Backend (http://localhost:8080)      ← Has the endpoint, but never receives request
```

## Solution

Added the `/entity-schema` endpoint routes to the API Gateway's proxy configuration and allowed unauthenticated access during development.

### Changes Made

**File**: `api-gateway/main.go`

#### 1. Added proxy routes (around line 1160):

```go
// Entity Schema endpoints - forward to backend service (backend exposes /api/entity-schema)
api.GET("/entity-schema", proxy)
api.POST("/entity-schema", proxy)
```

#### 2. Added authentication bypass for development (line 208):

Updated the `DEV_ALLOW_UNAUTH_FABRIC` check to include `/api/entity-schema`:
```go
if strings.HasPrefix(p, "/api/policies") || strings.HasPrefix(p, "/api/bundles") || 
   strings.HasPrefix(p, "/api/semantic") || strings.HasPrefix(p, "/api/business") || 
   strings.HasPrefix(p, "/api/data-domains") || strings.HasPrefix(p, "/api/profiler") || 
   strings.HasPrefix(p, "/api/entity-schema") {
    c.Next()
    return
}
```

This ensures:
- `GET /api/entity-schema` requests are forwarded to the backend
- `POST /api/entity-schema` requests are forwarded to the backend
- Both endpoints are accessible without authentication during local development
- The proxy handler automatically includes necessary headers like `X-Tenant-ID` and `X-Tenant-Datasource-ID`

## Testing

After rebuilding the API Gateway, the endpoint responds correctly:

### GET Request (Retrieve schema)
```bash
curl -X GET "http://localhost:8001/api/entity-schema" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0"
```

### POST Request (Save schema)
```bash
curl -X POST "http://localhost:8001/api/entity-schema" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  -H "Content-Type: application/json" \
  -d '{"changed": {...}, "deleted": [...]}'
```

## Related Context

- See `agents.md` for Tenant-Scoped Fabric Bundles requirements
- The backend endpoint requires `X-Tenant-ID` and `X-Tenant-Datasource-ID` headers (or query parameters)
- The API Gateway's `createProxyHandler` automatically copies headers and forwards them to the backend

## Build & Restart

To apply this fix in your local environment:

```bash
cd /Users/eganpj/GitHub/semlayer
docker compose build api-gateway
docker compose up -d
```

## Verification Results

✅ **Fixed!** The endpoint now responds with HTTP 200 OK:

```bash
curl -s http://localhost:8001/api/entity-schema \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0"
```

**Response**: HTTP 200 with full entity schema data

The frontend can now successfully fetch entity schema data without encountering 404 errors.
