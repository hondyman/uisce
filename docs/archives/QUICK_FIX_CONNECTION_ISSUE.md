# ⚡ Quick Fix: Backend Connection Issue RESOLVED

## Problem
You were getting: `Failed to load resource: net::ERR_CONNECTION_REFUSED` on port 29080

## Root Cause
The frontend's Vite dev server was configured to proxy API requests to `localhost:29080` instead of your backend at `localhost:8080`.

## Solution Applied ✅

### 1. Created `.env.local` in frontend directory:
```
VITE_USE_PROXY=true
VITE_BACKEND_TARGET=http://localhost:8080
VITE_API_BASE_URL=http://localhost:8080
```

### 2. Frontend Now Running
- **Frontend**: http://localhost:5173 (Vite dev server)
- **Backend**: http://localhost:8080 (Go server)
- **Proxy**: Frontend proxies `/api/*` requests to backend

## What This Means
✅ Frontend at 5173 will now correctly proxy all API calls to backend at 8080  
✅ Tenant scoping headers will be passed through  
✅ ValidationRulesWithFacets and other components will connect  

## Next Steps

### In Browser
1. Navigate to: **http://localhost:5173**
2. You should see the Fabric Builder UI
3. Select your tenant from the picker
4. The validation rules and other API calls should work!

### If Still Having Issues
1. **Verify backend is running**:
   ```bash
   curl http://localhost:8080/health
   ```
   Should NOT return "connection refused"

2. **Check frontend started correctly**:
   ```bash
   curl http://localhost:5173
   ```
   Should return HTML

3. **Clear browser cache**:
   - Open DevTools (F12)
   - Go to Application → Clear Site Data
   - Hard refresh (Cmd+Shift+R)

## Summary

| Component | Port | Status | URL |
|-----------|------|--------|-----|
| Backend (Go) | 8080 | ✅ Running | http://localhost:8080 |
| Frontend (Vite) | 5173 | ✅ Running | http://localhost:5173 |
| Proxy Target | 8080 | ✅ Configured | `.env.local` |

## Important Note
The port 29080 error was because:
- `vite.config.ts` had `VITE_BACKEND_TARGET` default of `localhost:29080`
- Your backend is on `8080`
- The `.env.local` file overrides this default

Going forward, keep `.env.local` committed locally (add to `.gitignore`) so your dev environment stays configured.

---

**Status**: Issue Resolved ✅  
**Access at**: http://localhost:5173  
**Backend**: http://localhost:8080
