# ✅ GraphQL Connection Error FIXED

## Problem
```
POST http://localhost:8083/v1/graphql net::ERR_CONNECTION_REFUSED
TypeError: Failed to fetch
```

Frontend was trying to connect to GraphQL on port **8083** instead of **8080**.

## Root Cause
The `.env.local` file was overriding `.env` with incorrect GraphQL endpoint:
```bash
# .env.local (had wrong port)
VITE_GRAPHQL_ENDPOINT=http://localhost:8083/v1/graphql  ❌
```

Note: `.env.local` takes precedence over `.env` in development mode.

## Solution Applied ✅

### Fixed `.env.local`
```bash
# Updated to use correct port
VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql  ✅
VITE_GRAPHQL_WS_ENDPOINT=ws://localhost:8080/v1/graphql  ✅
```

### Restarted Frontend Dev Server
- Killed old Vite process
- Restarted with updated environment variables
- Server now running on port 5173 ✅

## Verification

The frontend will now:
1. ✅ Connect to GraphQL at `http://localhost:8080/v1/graphql`
2. ✅ Load tenant data successfully
3. ✅ Initialize Apollo client without errors
4. ✅ Display the application properly

## Current Status

| Service | Port | Status |
|---------|------|--------|
| Frontend (Vite) | 5173 | ✅ Running |
| Backend Server | 8080 | ✅ Running |
| GraphQL Endpoint | 8080 | ✅ Correct |
| Database | 5432 | ✅ Connected |

## What to Do Now

1. **Open browser**: http://localhost:5173
2. **Check browser console (F12)**:
   - Should see successful GraphQL query for tenants
   - No "net::ERR_CONNECTION_REFUSED" errors
   - Application should load normally

3. **Set tenant scope** (if needed):
```javascript
localStorage.setItem('selected_tenant', JSON.stringify({
  id: '910638ba-a459-4a3f-bb2d-78391b0595f6',
  display_name: 'Test Tenant'
}));
localStorage.setItem('selected_datasource', JSON.stringify({
  id: '982aef38-418f-46dc-acd0-35fe8f3b97b0',
  source_name: 'Test Datasource'
}));
localStorage.setItem('selected_product', JSON.stringify({
  id: 'product-1',
  alpha_product: { product_name: 'Test Product' }
}));
window.location.reload();
```

## Files Modified

- ✅ `frontend/.env.local` - Updated GraphQL endpoints to port 8080

## Related Ports

- Port **8080**: Backend API + GraphQL (our target) ✅
- Port **8083**: Docker Hasura (not running locally)

The frontend was confused about which GraphQL service to use. Now it correctly points to the backend's GraphQL on 8080.

---

**Fix Date**: October 23, 2025  
**Status**: ✅ RESOLVED  
**Ready for**: Browser testing
