# Quick Test: Direct API Call

Run this in your browser console to test if the backend accepts the request:

```javascript
// Test 1: Direct fetch bypassing setupTenantFetch wrapper
console.log('=== TEST 1: Direct Fetch ===');
fetch('http://localhost:8080/api/semantic-mappings/edges?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-Tenant-ID': '910638ba-a459-4a3f-bb2d-78391b0595f6',
    'X-Tenant-Datasource-ID': '982aef38-418f-46dc-acd0-35fe8f3b97b0'
  },
  credentials: 'include',
  body: JSON.stringify({
    mappings: [{
      database_column: {
        schema: 'public',
        table: 'test_table',
        column: 'test_column',
        node_id: 'test-node-123',
        tenant_id: '910638ba-a459-4a3f-bb2d-78391b0595f6',
        tenant_datasource_id: '982aef38-418f-46dc-acd0-35fe8f3b97b0'
      },
      semantic_term: 'TEST_TERM',
      semantic_term_id: 'test-term-123',
      is_new_term: false,
      confidence: 1.0,
      override: true
    }]
  })
})
.then(response => {
  console.log('Response status:', response.status, response.statusText);
  return response.json();
})
.then(data => {
  console.log('Response data:', data);
  console.log('✅ Direct fetch WORKED! Backend is accessible.');
})
.catch(error => {
  console.error('❌ Direct fetch FAILED:', error);
  console.error('Error details:', error.message, error.stack);
});

// Wait 2 seconds, then test through the wrapper
setTimeout(() => {
  console.log('\n=== TEST 2: Through setupTenantFetch Wrapper ===');
  
  // This should use the patched fetch
  fetch('/api/semantic-mappings/edges', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    credentials: 'include',
    body: JSON.stringify({
      mappings: [{
        database_column: {
          schema: 'public',
          table: 'test_table',
          column: 'test_column',
          node_id: 'test-node-123',
          tenant_id: '910638ba-a459-4a3f-bb2d-78391b0595f6',
          tenant_datasource_id: '982aef38-418f-46dc-acd0-35fe8f3b97b0'
        },
        semantic_term: 'TEST_TERM',
        semantic_term_id: 'test-term-123',
        is_new_term: false,
        confidence: 1.0,
        override: true
      }]
    })
  })
  .then(response => {
    console.log('Response status:', response.status, response.statusText);
    return response.json();
  })
  .then(data => {
    console.log('Response data:', data);
    console.log('✅ Wrapped fetch WORKED! setupTenantFetch is OK.');
  })
  .catch(error => {
    console.error('❌ Wrapped fetch FAILED:', error);
    console.error('Error details:', error.message, error.stack);
    console.error('🔍 Check logs above for [setupTenantFetch] messages');
  });
}, 2000);
```

## Expected Output

### If Direct Fetch Works:
```
=== TEST 1: Direct Fetch ===
Response status: 200 OK
Response data: {created_edges: 0, created_terms: 0, ...}
✅ Direct fetch WORKED! Backend is accessible.

=== TEST 2: Through setupTenantFetch Wrapper ===
[setupTenantFetch] Intercepted request: {url: "/api/semantic-mappings/edges", method: "POST", ...}
[setupTenantFetch] Making request: {finalUrl: "http://localhost:8080/...", method: "POST", hasBody: true}
[setupTenantFetch] Response received: {url: "...", status: 200, statusText: "OK"}
Response status: 200 OK
Response data: {created_edges: 0, created_terms: 0, ...}
✅ Wrapped fetch WORKED! setupTenantFetch is OK.
```

### If Direct Fetch Fails with CORS:
```
=== TEST 1: Direct Fetch ===
❌ Direct fetch FAILED: TypeError: Failed to fetch
Access to fetch at 'http://localhost:8080/...' from origin 'http://localhost:5173' 
has been blocked by CORS policy: Response to preflight request doesn't pass access control check
```
**Meaning:** Backend CORS configuration issue

### If Wrapped Fetch Fails:
```
=== TEST 2: Through setupTenantFetch Wrapper ===
[setupTenantFetch] Tenant scope not set, rejecting request to: /api/semantic-mappings/edges
❌ Wrapped fetch FAILED: Error: Tenant selection required
```
**Meaning:** Tenant scope not cached in localStorage

## What To Look For

1. **Both tests work** → Original issue might be fixed, try UI again
2. **Direct works, wrapped fails** → setupTenantFetch is blocking it
3. **Both fail with CORS** → Backend CORS config needs fixing
4. **Both fail with network error** → Backend not accessible

Run this test and report the results! 🎯
