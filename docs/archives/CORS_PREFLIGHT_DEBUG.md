# CORS Preflight Issue: OPTIONS but No POST

## Problem
The browser is sending an OPTIONS request (CORS preflight), which succeeds with 204, but the actual POST request is never sent.

## Evidence
```
Request Method: OPTIONS
Status Code: 204 No Content
Request URL: http://localhost:8080/api/semantic-mappings/edges?tenant_id=...&datasource_id=...
```

Backend logs show OPTIONS was handled:
```
{"level":"info","resource_id":"/api/semantic-mappings/edges"}
Request completed in 2.138875ms
```

But NO POST request follows!

## Why This Happens

### CORS Preflight Flow
```
1. Browser sees: POST with custom headers + query params
2. Browser sends: OPTIONS (preflight check)
3. Backend responds: 204 with CORS headers ✅
4. Browser should send: POST with actual data
5. ❌ POST NEVER SENT
```

### Common Causes

#### 1. CORS Headers Missing in Preflight Response
**Check:** OPTIONS response must have:
- `Access-Control-Allow-Origin: http://localhost:5173`
- `Access-Control-Allow-Methods: POST`
- `Access-Control-Allow-Headers: Content-Type, X-Tenant-ID, X-Tenant-Datasource-ID`
- `Access-Control-Allow-Credentials: true` (if using credentials)

**Solution:** Backend CORS middleware needs all these headers in OPTIONS response.

#### 2. JavaScript Error After Preflight
**Check:** Browser console for errors between OPTIONS and POST

**Look for:**
- `TypeError: Failed to fetch`
- `NetworkError`
- Promise rejection
- setupTenantFetch errors

**Solution:** Fix the JavaScript error.

#### 3. Request Blocked by Browser Security
**Check:** Browser console for security warnings

**Look for:**
- `Blocked by CORS policy`
- `Mixed content`
- `Insecure request`

**Solution:** Ensure both frontend and backend use same protocol (both HTTP or both HTTPS).

#### 4. Credentials Mode Mismatch
**Check:** Request using `credentials: 'include'` but CORS not allowing it

**Solution:** Backend must return:
```
Access-Control-Allow-Credentials: true
Access-Control-Allow-Origin: http://localhost:5173  (NOT *)
```

## Debugging Steps

### Step 1: Check Browser Console
Look for errors logged between the OPTIONS and (missing) POST.

**With enhanced logging, you should see:**
```
[setupTenantFetch] Intercepted request: {url: "/api/semantic-mappings/edges", method: "POST", ...}
[setupTenantFetch] Making request: {finalUrl: "http://localhost:8080/...", method: "POST", hasBody: true}
```

**If you DON'T see "Making request":** The request is being blocked before fetch() is called.

**If you see "Making request" but no response:** The fetch() call is failing.

### Step 2: Check Network Tab Response Headers
Click on the OPTIONS request in Network tab.

**Check Response Headers:**
```
Access-Control-Allow-Origin: http://localhost:5173  ✅
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS  ✅
Access-Control-Allow-Headers: Content-Type, X-Tenant-ID, X-Tenant-Datasource-ID  ✅
Access-Control-Allow-Credentials: true  ✅
```

**If any are missing:** Backend CORS configuration needs updating.

### Step 3: Check Request Headers
Click on the OPTIONS request, look at Request Headers.

**Check:**
```
Origin: http://localhost:5173
Access-Control-Request-Method: POST
Access-Control-Request-Headers: content-type, x-tenant-id, x-tenant-datasource-id
```

**If headers look wrong:** Frontend is sending incorrect preflight.

### Step 4: Try Without setupTenantFetch
Temporarily bypass setupTenantFetch to see if it's the issue.

**In browser console:**
```javascript
// Direct fetch without interception
const originalFetch = window.fetch;
originalFetch('http://localhost:8080/api/semantic-mappings/edges?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-Tenant-ID': '910638ba-a459-4a3f-bb2d-78391b0595f6',
    'X-Tenant-Datasource-ID': '982aef38-418f-46dc-acd0-35fe8f3b97b0'
  },
  credentials: 'include',
  body: JSON.stringify({ mappings: [] })
})
.then(r => r.json())
.then(d => console.log('Direct fetch result:', d))
.catch(e => console.error('Direct fetch error:', e));
```

**If this works:** setupTenantFetch is the problem.
**If this fails too:** CORS or network issue.

## Enhanced Logging Added

I've added detailed logging to `setupTenantFetch.ts`:

**Before request:**
```javascript
[setupTenantFetch] Intercepted request: {url, method, tenantId, datasourceId}
[setupTenantFetch] Making request: {finalUrl, method, hasBody}
```

**After request:**
```javascript
[setupTenantFetch] Response received: {url, status, statusText}
```

**On error:**
```javascript
[setupTenantFetch] Tenant scope not set, rejecting request to: ...
```

## Most Likely Issue

Based on the symptoms, I suspect:

### Issue: Browser Silently Blocking POST After Successful Preflight

**Possible causes:**
1. JavaScript error in promise chain after preflight
2. setupTenantFetch rejecting the request
3. Body being consumed twice (cloning issue)
4. CORS header case sensitivity

## Solution: Check Console Logs

The enhanced logging will show exactly where the request is failing.

**Look for:**
1. Is `[setupTenantFetch] Making request` logged?
   - YES → fetch() is being called
   - NO → Request blocked before fetch()

2. Is `[setupTenantFetch] Response received` logged?
   - YES → Response came back
   - NO → fetch() threw an error

3. Any error messages between these logs?
   - This will tell us exactly what failed

## Backend CORS Configuration

The backend CORS middleware looks correct:

```go
w.Header().Set("Access-Control-Allow-Origin", origin)
w.Header().Set("Access-Control-Allow-Credentials", "true")
w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, X-Request-ID, X-Tenant-Datasource-ID, X-Tenant-ID, X-User-ID")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
```

This should allow the POST request after preflight.

## Next Steps

1. ✅ Enhanced logging added to setupTenantFetch
2. ✅ Dev server running with hot reload
3. 🔍 **Refresh browser** to load new code
4. 🔍 Open Console (F12)
5. 🔍 Try "Create Edges" again
6. 🔍 Look for the new `[setupTenantFetch]` logs
7. 📋 Report back with:
   - All `[setupTenantFetch]` console logs
   - Any errors between them
   - Network tab showing BOTH OPTIONS and POST (if POST appears)

The logs will tell us exactly where it's failing! 🎯
