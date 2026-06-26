# Glossary Endpoints 404 Fix

## Problem
Frontend was getting 404 errors on glossary endpoints:
- `GET /api/glossary/semantic-terms` → 404
- `GET /api/glossary/edges` → 404
- `GET /api/admin/llm/config` → 404

### Root Cause
The frontend's `.env.local` was configured for **Docker-based development** with:
```env
VITE_BACKEND_TARGET=http://localhost:8001
VITE_API_BASE_URL=http://localhost:8001
```

However:
1. Backend API server was actually running on `http://localhost:8080` (local, not Docker)
2. API Gateway on port 8001 was not running
3. Vite proxy was forwarding `/api/*` requests to the wrong backend port
4. Requests were hitting a non-existent backend, returning 404

## Solution
Updated `.env.local` to point to the correct local backend port:

```diff
- VITE_BACKEND_TARGET=http://localhost:8001
+ VITE_BACKEND_TARGET=http://localhost:8080

- VITE_API_BASE_URL=http://localhost:8001
+ VITE_API_BASE_URL=http://localhost:8080
```

## Verification
After fix, backend logs show successful requests:
```
[REQ] GET /api/glossary/semantic-terms ... status=200
[REQ] GET /api/glossary/edges ... status=200
```

## Backend Route Status
All glossary routes are properly implemented and registered:
- ✅ `GET /api/glossary/semantic-terms` - Lists semantic terms
- ✅ `GET /api/glossary/business-terms` - Lists business terms
- ✅ `GET /api/glossary/edges` - Lists edges between terms
- ✅ `POST /api/glossary/edges` - Creates edges
- ✅ `PUT /api/glossary/edges/{id}` - Updates edges
- ✅ `DELETE /api/glossary/edges/{id}` - Deletes edges

## For Non-Docker Development
When developing locally WITHOUT Docker:
- Backend: `http://localhost:8080` (Go server)
- Hasura (if using): Update `VITE_GRAPHQL_ENDPOINT` separately if needed

## For Docker Development  
When using `docker-compose`:
- API Gateway: `http://localhost:8001`
- Hasura: `http://localhost:8085`

Update `.env.local` accordingly for your development environment.
