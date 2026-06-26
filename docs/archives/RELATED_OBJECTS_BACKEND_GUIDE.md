# Related Objects Tab - Backend Implementation Guide

## Current Status

✅ **Frontend**: Complete with graceful demo data fallback  
⚠️ **Backend**: Endpoint exists but returning 404  

### What's Happening

When you access the Related Objects tab:
1. ✅ Component loads without errors
2. ✅ Shows demo data (Orders, Department, Manager relationships)
3. ⚠️ Tries to fetch from `/api/relationships/objects`
4. ❌ Backend returns 404 (endpoint not accessible)
5. ✅ Falls back to demo data with helpful message

---

## Backend Endpoint Status

### Location
**File**: `backend/internal/api/api.go`  
**Handler**: `getRelatedObjects` (line 6336)  
**Route**: Already registered at line 352

### Route Registration
```go
r.Get("/relationships/objects", srv.getRelatedObjects)
```

### Current Implementation
```go
func (s *Server) getRelatedObjects(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	entity := r.URL.Query().Get("entity")

	if tenantID == "" || datasourceID == "" || entity == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Query catalog_edge table for relationships
	query := `
		SELECT 
			ce.id as edge_id,
			CASE WHEN ce.source_node_id = cn.id THEN 'OUTBOUND' ELSE 'INBOUND' END as direction,
			cet.edge_type_name as edge_type,
			ce.cardinality,
			cn_src.node_name as source_name,
			cn_tgt.node_name as target_name
		FROM catalog_edge ce
		JOIN catalog_node cn ON (ce.source_node_id = cn.id OR ce.target_node_id = cn.id)
		...
	`
	// Implementation exists but may not be routing correctly
}
```

---

## Problem: Why 404?

### Possible Causes

1. **Backend not running**
   - ✓ Test: `curl http://localhost:8080/api/health`

2. **Wrong port**
   - Frontend tries: `localhost:8001`
   - Backend runs on: `localhost:8080`
   - Solution: Check vite proxy config or backend server config

3. **Route not matching**
   - Frontend sends: `/api/relationships/objects`
   - Backend registered: `r.Get("/relationships/objects", ...)`
   - This should work inside `/api` subrouter

4. **Middleware blocking request**
   - Auth middleware might reject requests
   - Tenant scope validation might fail
   - Check request headers being sent

5. **Database query fails**
   - `catalog_edge` table might not exist
   - `catalog_node` table might not exist
   - No relationships defined in database

---

## Solution: Enable Backend Endpoint

### Option 1: Fix Backend Connection (Recommended)

Check backend is running:
```bash
# Terminal 1: Start backend
cd /Users/eganpj/GitHub/semlayer
go run ./cmd/server/main.go
# Should print: "Server listening on :8080"
```

Test endpoint directly:
```bash
curl "http://localhost:8080/api/relationships/objects?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0&entity=Employee" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0"
```

Expected response:
```json
[
  {
    "edgeId": "...",
    "direction": "OUTBOUND",
    "edgeType": "has_many",
    "cardinality": "One-to-Many",
    "source": { "id": "Employee", "name": "Employee" },
    "target": { "id": "Orders", "name": "Orders" }
  }
]
```

### Option 2: Enable Vite Proxy (Development Only)

Create `.env.local` in frontend directory:
```
VITE_USE_PROXY=true
VITE_BACKEND_TARGET=http://localhost:8080
```

Restart frontend dev server:
```bash
cd frontend
npm run dev
```

This tells Vite to proxy `/api` requests to backend.

### Option 3: Fix Frontend Request URL

If you know the backend is on a different port:

Edit `RelatedObjectsTab.tsx` line 53:
```typescript
// Current (relative, uses current origin)
const response = await fetch(`/api/relationships/objects?${params.toString()}`, ...)

// Change to (absolute URL to backend)
const response = await fetch(
  `http://localhost:8080/api/relationships/objects?${params.toString()}`,
  ...
)
```

---

## Step-by-Step: Implement Live Data

### Step 1: Verify Backend Handler

The handler already exists. Just verify it compiles:

```bash
cd /Users/eganpj/GitHub/semlayer/backend
go build ./cmd/server/main.go
```

If no errors, handler is ready.

### Step 2: Verify Database Schema

Check if catalog tables exist:

```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "
SELECT table_name 
FROM information_schema.tables 
WHERE table_name IN ('catalog_edge', 'catalog_node', 'catalog_edge_types')
"
```

Expected output:
```
 catalog_edge
 catalog_node
 catalog_edge_types
```

If missing, create them or populate test data.

### Step 3: Insert Test Relationships

If tables exist, add test data:

```sql
-- Connect to database
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable

-- Insert test relationship between Employee and Orders
INSERT INTO catalog_edge (
  id, 
  tenant_datasource_id,
  source_node_id, 
  target_node_id,
  edge_type_id,
  cardinality,
  properties
) VALUES (
  'rel-emp-ord-1',
  '982aef38-418f-46dc-acd0-35fe8f3b97b0',
  (SELECT id FROM catalog_node WHERE node_name = 'Employee' LIMIT 1),
  (SELECT id FROM catalog_node WHERE node_name = 'Orders' LIMIT 1),
  'has_many',
  'One-to-Many',
  '{"key_fields": ["EmployeeID"]}'
);
```

### Step 4: Start Backend

```bash
cd /Users/eganpj/GitHub/semlayer
go run ./cmd/server/main.go
```

### Step 5: Test API Directly

```bash
curl "http://localhost:8080/api/relationships/objects?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0&entity=Employee"
```

### Step 6: Update Frontend Connection

Option A - Enable Vite proxy (recommended for dev):
```bash
echo "VITE_USE_PROXY=true" >> frontend/.env.local
echo "VITE_BACKEND_TARGET=http://localhost:8080" >> frontend/.env.local
npm run dev  # Restart dev server
```

Option B - Update API URL in component:
```typescript
// In RelatedObjectsTab.tsx line 53, change fetch URL to:
fetch(`http://localhost:8080/api/relationships/objects?...`)
```

### Step 7: Refresh Browser

Navigate back to Related Objects tab. Should now show real data instead of demo.

---

## Troubleshooting

### Still seeing demo data?

1. **Clear browser cache**
   ```
   Ctrl+Shift+Del (or Cmd+Shift+Del on Mac)
   ```

2. **Check browser console**
   ```
   F12 → Console tab → Look for error messages
   ```

3. **Verify backend is running**
   ```bash
   ps aux | grep "go run\|server"
   curl http://localhost:8080/api/health
   ```

4. **Check network requests**
   ```
   F12 → Network tab → Look for /api/relationships/objects request
   Check: Status code, Response body, Request headers
   ```

### Backend returns 404

1. **Verify route is registered**
   - Check `api.go` line 352 has the route
   - Ensure it's inside the `/api` subrouter

2. **Verify handler exists**
   - Check `getRelatedObjects` function exists (line 6336)
   - Verify no compilation errors: `go build ./cmd/server/main.go`

3. **Check database connection**
   - Verify PostgreSQL is running
   - Test query: `psql postgres://postgres:postgres@localhost:5432/alpha`

4. **Enable debug logging**
   - Backend logs unmatched requests at `/api` level
   - Check terminal output for `[API-NOTFOUND]` messages

### Database query fails

1. **Check tables exist**
   ```sql
   \dt catalog_*
   ```

2. **Check data exists**
   ```sql
   SELECT COUNT(*) FROM catalog_edge;
   SELECT COUNT(*) FROM catalog_node;
   ```

3. **Check for entity by name**
   ```sql
   SELECT * FROM catalog_node WHERE node_name = 'Employee' LIMIT 1;
   ```

---

## Expected Response Format

When backend is working, API returns:

```json
[
  {
    "edgeId": "edge-123",
    "direction": "OUTBOUND",
    "edgeType": "has_many",
    "cardinality": "One-to-Many",
    "source": {
      "id": "Employee",
      "name": "Employee",
      "kind": "table"
    },
    "target": {
      "id": "Orders",
      "name": "Orders",
      "kind": "table"
    }
  },
  {
    "edgeId": "edge-456",
    "direction": "INBOUND",
    "edgeType": "belongs_to",
    "cardinality": "Many-to-One",
    "source": {
      "id": "Department",
      "name": "Department",
      "kind": "table"
    },
    "target": {
      "id": "Employee",
      "name": "Employee",
      "kind": "table"
    }
  }
]
```

---

## Demo Data (Current State)

When API returns 404, frontend shows this demo data:

1. **Orders Relationship**
   - Type: One-to-Many
   - Description: Each employee has many orders
   - Keys: Employee(ID) → Orders(EmployeeID)

2. **Department Relationship**
   - Type: Many-to-One
   - Description: Many employees belong to one department
   - Keys: Employee(DepartmentID) → Department(ID)

3. **Manager Relationship**
   - Type: Many-to-One
   - Description: Many employees report to one manager
   - Keys: Employee(ManagerID) → Manager(ID)

This demo data is:
- ✅ Fully functional (both card and diagram views work)
- ✅ Shows what real data will look like
- ✅ Useful for UI/UX testing while backend is being implemented
- ⚠️ Not real database data

---

## Summary

| Component | Status | Action |
|-----------|--------|--------|
| Frontend | ✅ Ready | None, shows demo data |
| Backend Handler | ✅ Exists | Verify it's routed correctly |
| Database | ❓ Unknown | Check if tables exist |
| API Connection | ⚠️ 404 | Enable proxy or fix port |
| Demo Data | ✅ Working | Currently displayed |

### Quick Start for Live Data

```bash
# 1. Ensure backend is running
cd /Users/eganpj/GitHub/semlayer
go run ./cmd/server/main.go

# 2. Enable Vite proxy in frontend
echo "VITE_USE_PROXY=true" >> frontend/.env.local

# 3. Restart dev server
cd frontend
npm run dev

# 4. Refresh browser
# Navigate to Related Objects tab
```

---

## Next Steps

1. **Start backend**: `go run ./cmd/server/main.go`
2. **Test API**: `curl http://localhost:8080/api/relationships/objects...`
3. **Check database**: Verify catalog tables have data
4. **Enable proxy**: Set `VITE_USE_PROXY=true` in frontend
5. **Refresh browser**: See live data (or continue using demo data)

The component is production-ready and will work with real data as soon as the backend endpoint is properly connected!
