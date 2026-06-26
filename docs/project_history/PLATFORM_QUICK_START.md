# SemLayer Platform - Quick Start Guide

## ✅ Prerequisites

Ensure these are running before starting the platform:

- **PostgreSQL**: Running on `localhost:5432` with user `postgres`/`postgres`
  - Database: `alpha` (metadata)
  - (Optional) Database: `northwinds` (aggregates)

```bash
# Check if Postgres is running
psql -U postgres -h localhost -d alpha -c "SELECT version();"
```

## 🚀 Quick Start (2 Steps)

### Step 1: Start Backend API Server

```bash
bash START_BACKEND.sh
```

Expected output:
```
✅ Backend server started
   URL: http://localhost:8080
   PID: XXXXX
   Logs: /Users/eganpj/GitHub/semlayer/logs/backend_*.log
```

**Verify backend is running:**
```bash
curl http://localhost:8080/health
# Should return: {"status":"healthy","timestamp":"..."}
```

### Step 2: Start Frontend Development Server (in another terminal)

```bash
bash START_FRONTEND.sh
```

Expected output:
```
✅ Frontend server started
   URL: http://localhost:5173
   PID: XXXXX
```

**Access the platform:**
- Frontend: http://localhost:5173
- Backend Swagger UI: http://localhost:8080/swagger/index.html

## 🔧 Configuration

### Backend Configuration
- **File**: `backend/config.yaml`
- **Database**: PostgreSQL on `localhost:5432`
- **API Port**: `8080`
- **Temporal**: Configured to connect to `localhost:7233` (optional - will warn if not running)

### Frontend Configuration
- **File**: `frontend/.env.local`
- **API Base URL**: `http://localhost:8080`
- **GraphQL Endpoint**: `/v1/graphql` (relative path, proxied through backend)
- **Dev Port**: `5173`

## 🎯 Features Available

All major APIs are running:
- ✅ REST APIs (100+ endpoints)
- ✅ Tenant-scoped operations (automatic via fetch interceptor)
- ✅ Bundle management
- ✅ Validation rules
- ✅ Fabric builder
- ✅ Semantic layer
- ⚠️  GraphQL (requires Hasura - not running by default)
- ⚠️  Temporal workflows (requires Temporal server - optional)

## 🚨 Common Issues & Solutions

### Issue: Port 8080 already in use
```bash
# Kill existing process
lsof -ti:8080 | xargs kill -9
# Then restart
bash START_BACKEND.sh
```

### Issue: Database connection failed
**Error**: `failed to connect to user=postgres database=alpha`

**Solution**:
1. Ensure PostgreSQL is running: `psql -U postgres -h localhost -c "SELECT 1;"`
2. Create `alpha` database if missing:
   ```sql
   CREATE DATABASE alpha;
   ```
3. Verify `backend/config.yaml` points to `localhost:5432`

### Issue: Frontend shows "Select a tenant" warning
This is normal on first load. The system automatically seeds a test tenant scope to `localStorage`.
- **Check**: Browser DevTools > Application > Local Storage
- **Keys**: `selected_tenant`, `selected_product`, `selected_datasource`

### Issue: GraphQL endpoint returns 404
**Expected** - Hasura GraphQL is not running. Most features work with REST APIs.
- GraphQL is optional for core functionality
- To enable GraphQL, start Hasura separately (not included in quick start)

### Issue: Temporal worker warnings
**Expected** - Temporal server is optional. The backend will warn but continue working.
- Temporal is needed only for advanced workflow features
- To enable: Start Temporal server on `localhost:7233` separately

## 📊 Tenant Context

The system uses **mandatory tenant scoping** for multi-tenant safety:

```javascript
// Auto-seeded test values (in localStorage):
localStorage.setItem('selected_tenant', JSON.stringify({
  id: '910638ba-a459-4a3f-bb2d-78391b0595f6',
  display_name: 'Test Tenant'
}));

localStorage.setItem('selected_datasource', JSON.stringify({
  id: '982aef38-418f-46dc-acd0-35fe8f3b97b0',
  source_name: 'Test Datasource'
}));
```

All API requests automatically include:
- Query parameters: `?tenant_id=...&datasource_id=...`
- Headers: `X-Tenant-ID: ...` and `X-Tenant-Datasource-ID: ...`

## 📁 Log Files

Logs are saved to `/Users/eganpj/GitHub/semlayer/logs/`:
- `backend_YYYYMMDD_HHMMSS.log` - Backend API logs
- `frontend_YYYYMMDD_HHMMSS.log` - Frontend dev server logs

## 🛑 Stopping the Platform

```bash
# Stop backend
lsof -ti:8080 | xargs kill -9

# Stop frontend
lsof -ti:5173 | xargs kill -9

# Or press Ctrl+C in each terminal
```

## 🔗 API Documentation

Once running, access Swagger UI:
- **URL**: http://localhost:8080/swagger/index.html
- **Features**: Try out all API endpoints, see request/response schemas

## ✨ Next Steps

After starting the platform:

1. **Explore the UI**: http://localhost:5173
2. **Test REST APIs**: Use Swagger UI at http://localhost:8080/swagger/index.html
3. **Check logs** if issues arise: `tail -f logs/backend_*.log`
4. **Configure** tenant scope in browser if needed (see Tenant Context section)

---

**Last Updated**: November 11, 2025
**Status**: ✅ Platform Ready for Development
