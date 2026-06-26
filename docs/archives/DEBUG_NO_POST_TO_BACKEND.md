# DEBUG: Why "Create Edges" Is Not Posting to Backend

## Current State
- ✅ Frontend dev server running on `localhost:5173`
- ✅ Backend running on `localhost:8080` (in Docker)
- ❌ Clicking "Create Edges" does not send request to backend
- ❌ No logs appearing in backend

## Root Cause Analysis

### The Request Flow
```
User clicks "Create Edges"
    ↓
confirmCreate() filters selected mappings
    ↓
Calls createEdges(selected)
    ↓
fetch('/api/semantic-mappings/edges')
    ↓
setupTenantFetch intercepts
    ↓
Should forward to backend at localhost:8080
    ↓
❌ REQUEST NEVER ARRIVES
```

## Possible Issues

### Issue #1: Tenant Scope Not Set
**Symptom:** setupTenantFetch.ts rejects request before sending

**Check:**
Open browser console and run:
```javascript
localStorage.getItem('selected_tenant')
localStorage.getItem('selected_datasource')
```

**Expected:** Both should return JSON objects with IDs

**If NULL:** You need to select a tenant in the UI first!

**Solution:**
1. Look for tenant selector at top of page
2. Select a tenant and datasource
3. Try "Create Edges" again

---

### Issue #2: No Mappings Passing Filter
**Symptom:** `confirmCreate()` filters out all mappings

**Check:**
Look in browser console for:
```
[SemanticMapper] Creating edges for mappings: []
```

**If empty array:** Mappings are being filtered out

**Reasons:**
- `!m.edge_exists` - Edge already exists
- `m.ignored` - Mapping is ignored
- `!selectedMappings.has(id)` - Row not selected (checkbox not checked)

**Solution:**
1. Make sure row checkbox is checked
2. Make sure "Ready to Create Edge" chip shows (green)
3. Make sure `edge_exists` is false

---

### Issue #3: fetch() Throwing Error Before Request
**Symptom:** Request fails immediately without network call

**Check:**
Look in browser console for errors like:
```
Error: Tenant selection required
```
or
```
TypeError: Failed to fetch
```

**Solution:**
- Select tenant if scope error
- Check CORS if fetch fails
- Check network tab for failed requests

---

### Issue #4: Request Going to Wrong URL
**Symptom:** Request sent but not to backend

**Check:**
Look in browser Network tab (F12 → Network)
Filter by "edges"
Look for POST request to `/api/semantic-mappings/edges`

**Possible wrong URLs:**
- `http://localhost:5173/api/...` (frontend, returns HTML)
- `http://localhost:3000/api/...` (wrong port)
- `/semantic-mappings/edges` (missing /api prefix)

**Solution:**
The request SHOULD go to `http://localhost:8080/api/semantic-mappings/edges`
If not, check `VITE_BACKEND_TARGET` env variable

---

## How to Debug

### Step 1: Open Browser DevTools
Press F12 or Right-click → Inspect

### Step 2: Open Console Tab
You should see logs like:
```
[useSemanticMapper] Creating edges: {url: "/api/semantic-mappings/edges", count: 1, ...}
[useSemanticMapper] Request payload: {...}
```

### Step 3: Open Network Tab
Filter for "edges"
Look for POST request

### Step 4: Check Request Details
Click on the POST request in Network tab

**Check:**
- Request URL: Should be `http://localhost:8080/api/semantic-mappings/edges`
- Request Method: POST
- Status: Should be 200 or 201
- Request Payload: Should have `mappings` array with tenant IDs

**Common Problems:**
- Status 400: Bad request (missing tenant info)
- Status 401: Unauthorized (need auth)
- Status 404: Not found (wrong URL)
- Status 502: Bad gateway (backend not running)
- (failed) net::ERR_CONNECTION_REFUSED: Backend not accessible

---

## Quick Diagnostic Script

Run this in browser console:

```javascript
// Check tenant scope
const tenant = localStorage.getItem('selected_tenant');
const datasource = localStorage.getItem('selected_datasource');
console.log('Tenant:', tenant ? JSON.parse(tenant) : '❌ NOT SET');
console.log('Datasource:', datasource ? JSON.parse(datasource) : '❌ NOT SET');

// Try a test request
fetch('/api/semantic-mappings/edges', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  credentials: 'include',
  body: JSON.stringify({ mappings: [] })
})
.then(res => {
  console.log('Test request status:', res.status);
  return res.json();
})
.then(data => console.log('Test request data:', data))
.catch(err => console.error('Test request error:', err));
```

**Expected output:**
```
Tenant: {id: "910638ba-...", display_name: "..."}
Datasource: {id: "982aef38-...", source_name: "..."}
Test request status: 200
Test request data: {created_edges: 0, ...}
```

---

## Enhanced Logging Added

I've added comprehensive logging to `useSemanticMapper.ts`:

1. **Before request:**
   - URL being called
   - Number of mappings
   - Each mapping's column, term, IDs, tenant info

2. **After request:**
   - Response status
   - Response data or error text

3. **Errors:**
   - Full error message
   - HTTP status if available

**Look for these logs** when clicking "Create Edges"!

---

## Network Tab Investigation

If the request IS appearing in Network tab but failing:

### Status 400 Bad Request
**Cause:** Missing tenant_id or tenant_datasource_id in payload

**Check payload:**
```json
{
  "mappings": [{
    "database_column": {
      "tenant_id": "...",              ← Must be present
      "tenant_datasource_id": "..."    ← Must be present  
    }
  }]
}
```

**If missing:** The bug fix didn't work, database_column is still not being preserved

### Status 404 Not Found
**Cause:** Backend route not registered or wrong URL

**Check:**
- Backend logs should show route registration
- URL should be `/api/semantic-mappings/edges` exactly

### Status 502 Bad Gateway
**Cause:** Backend crashed or not responding

**Check:**
- Docker container status: `docker ps`
- Backend logs: `docker logs semlayer-backend-1`

---

## If Request Never Appears in Network Tab

This means the request is blocked BEFORE being sent.

**Most likely causes:**
1. **Tenant scope check fails** - setupTenantFetch rejects it
2. **JavaScript error** - confirmCreate() throws before calling createEdges()
3. **Button not wired** - onClick not connected to confirmCreate()

**Check:**
1. Console for errors
2. Tenant selection (see diagnostic script above)
3. React DevTools to verify button's onClick prop

---

## Expected Backend Logs

When request arrives at backend, you should see:

```json
{"level":"info","msg":"POST /api/semantic-mappings/edges"}
```

Followed by semantic mapping service logs about creating edges.

**If you don't see these logs:** Request is not reaching backend!

---

## Next Steps

1. ✅ Frontend dev server is running
2. ✅ Added enhanced logging
3. 🔍 Open browser at http://localhost:5173
4. 🔍 Open DevTools Console + Network tabs
5. 🔍 Navigate to Semantic Mapper
6. 🔍 Click override, apply term, select row
7. 🔍 Click "Create Edges"
8. 🔍 Watch Console for logs
9. 🔍 Watch Network for POST request
10. 📋 Report back with:
    - Console logs
    - Network request (if any)
    - Request payload (if any)
    - Response status (if any)
    - Any errors

---

## Most Likely Issue

Based on your description "it's not posting to the backend", I suspect:

**Either:**
1. **Tenant scope not set** - Request blocked by setupTenantFetch
2. **No mappings passing filter** - confirmCreate() filters to empty array
3. **Request going to wrong URL** - Hitting Vite dev server instead of backend

**Solution:**
Run the diagnostic script above to check tenant scope first!
