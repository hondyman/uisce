# ACTION PLAN: Fix Missing POST After OPTIONS

## Current Situation

### ✅ What's Working:
- Frontend dev server: Running on localhost:5173
- Backend server: Running on localhost:8080 (Docker)
- OPTIONS request: Succeeds (204 No Content)
- CORS headers: Present and correct

### ❌ What's Broken:
- POST request: Never sent after successful OPTIONS
- Backend logs: Show only OPTIONS, no POST
- Edge creation: Fails silently

## Root Cause

The browser successfully completes the CORS preflight (OPTIONS), but something in the frontend JavaScript is preventing the actual POST request from being sent. This is typically caused by:

1. **setupTenantFetch wrapper** rejecting the request
2. **JavaScript error** after OPTIONS completes
3. **Promise rejection** in the fetch chain
4. **Body cloning issue** causing the request to fail

## Immediate Action Required

### Step 1: Refresh Browser & Check Console
**You MUST refresh your browser to get the new logging code!**

1. Go to http://localhost:5173
2. Press **Cmd+Shift+R** (Mac) or **Ctrl+Shift+R** (Windows) for hard refresh
3. Open DevTools Console (F12)
4. Navigate to Semantic Mapper page

### Step 2: Try "Create Edges" Again

1. Click override icon on a mapping
2. Type or select a semantic term (e.g., "LAST_UPDATE")
3. Click "Create & Apply New Term" or "Apply Existing Term"
4. Verify green "Ready to Create Edge" chip appears
5. Verify checkbox is checked
6. Click "Create Edges (1)" button
7. **WATCH THE CONSOLE**

### Step 3: Look for These Specific Logs

You should see this sequence in console:

```javascript
[SemanticMapper] Creating edges for mappings: [{...}]
[SemanticMapper] Request payload: {...}
[setupTenantFetch] Intercepted request: {url: "/api/semantic-mappings/edges", method: "POST", ...}
[setupTenantFetch] Making request: {finalUrl: "http://localhost:8080/...", method: "POST", hasBody: true}
[setupTenantFetch] Response received: {url: "...", status: 200, statusText: "OK"}
[useSemanticMapper] Response status: 200 OK
[useSemanticMapper] Response data: {created_edges: 1, ...}
```

### Step 4: Identify Where It Stops

**If you see:**
- ✅ `[SemanticMapper] Creating edges` → confirmCreate() is working
- ✅ `[useSemanticMapper] Creating edges` → createEdges() is being called
- ❌ **STOPS HERE** → Problem is in setupTenantFetch or fetch() call

**If you see:**
- ✅ `[setupTenantFetch] Intercepted request` → Tenant scope is set
- ❌ **STOPS HERE** → setupTenantFetch is rejecting the request

**If you see:**
- ✅ `[setupTenantFetch] Making request` → fetch() is being called
- ❌ **STOPS HERE** → fetch() is throwing an error

### Step 5: Check for Errors

Look for any error messages in console between the logs above. Common errors:

```
Error: Tenant selection required
TypeError: Failed to fetch
NetworkError when attempting to fetch resource
CORS policy: No 'Access-Control-Allow-Origin' header
```

## Alternative: Run Quick Test Script

If you don't see the logs above, run this in console:

```javascript
// Quick diagnostic
console.log('=== Checking Tenant Scope ===');
const tenant = localStorage.getItem('selected_tenant');
const datasource = localStorage.getItem('selected_datasource');
console.log('Tenant:', tenant ? JSON.parse(tenant).id : '❌ NOT SET');
console.log('Datasource:', datasource ? JSON.parse(datasource).id : '❌ NOT SET');

if (!tenant || !datasource) {
  console.error('⚠️ TENANT SCOPE NOT SET!');
  console.log('Solution: Use tenant selector at top of page to select tenant and datasource');
} else {
  console.log('✅ Tenant scope is set');
  console.log('Proceeding with test request...');
  
  // Test the exact request that should be sent
  fetch('/api/semantic-mappings/edges', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ mappings: [] })
  })
  .then(r => {
    console.log('✅ Request reached backend!', r.status);
    return r.json();
  })
  .then(d => console.log('Response:', d))
  .catch(e => console.error('❌ Request failed:', e));
}
```

## What To Report Back

Please provide:

1. **Console logs** showing the sequence of `[setupTenantFetch]` and `[useSemanticMapper]` messages
2. **Any error messages** that appear
3. **Network tab screenshot** showing:
   - The OPTIONS request (exists)
   - The POST request (missing)
4. **Result of test script** if you ran it

## Likely Scenarios

### Scenario A: Tenant Scope Not Set
**Console shows:**
```
[setupTenantFetch] Tenant scope not set, rejecting request to: /api/semantic-mappings/edges
```

**Solution:**
1. Look for tenant selector at top of page
2. Select a tenant from dropdown
3. Select a datasource from dropdown
4. Try "Create Edges" again

### Scenario B: JavaScript Error After OPTIONS
**Console shows:**
```
[setupTenantFetch] Intercepted request: {...}
TypeError: Cannot read property 'xxx' of undefined
```

**Solution:**
- Report the error message
- I'll fix the JavaScript bug

### Scenario C: fetch() Silently Failing
**Console shows:**
```
[setupTenantFetch] Making request: {...}
(no response log)
(no error message)
```

**Solution:**
- This suggests fetch() is being blocked by browser
- Check if browser security settings are blocking requests
- Try in incognito mode

### Scenario D: Network Error
**Console shows:**
```
[setupTenantFetch] Making request: {...}
TypeError: Failed to fetch
```

**Solution:**
- Backend might not be accessible
- Check Docker container is running: `docker ps`
- Check backend logs: `docker logs semlayer-backend-1`

## Expected Fix

Once we identify where it's failing from the console logs, the fix will be one of:

1. **Tenant scope issue** → Ensure tenant selector is used
2. **JavaScript error** → Fix the bug in code
3. **CORS issue** → Update backend CORS headers
4. **Network issue** → Fix backend accessibility

The enhanced logging will tell us exactly which one! 🎯

## Summary

**YOU NEED TO:**
1. ✅ Hard refresh browser (Cmd+Shift+R)
2. ✅ Open console (F12)
3. ✅ Try "Create Edges"
4. ✅ Copy all console logs
5. ✅ Report back with logs and any errors

**Without the console logs, I can't tell you exactly what's wrong!**
