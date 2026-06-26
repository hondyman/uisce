# Entity Save Not Persisting - Debugging Guide

## Root Cause Analysis

The entity save endpoint (`/entity-schema`) requires tenant-scoped headers to persist data to the backend:

- ✅ **Backend endpoint exists** at `/backend/internal/api/api.go:711`
- ✅ **Database table exists** at `/backend/migrations/000028_create_entity_schema_table.sql`
- ❓ **Question**: Are the required headers being sent?

The backend endpoint **requires**:
```
X-Tenant-ID: <uuid>
X-Tenant-Datasource-ID: <uuid>
```

Without these headers, the request returns `400 Bad Request: X-Tenant-ID and X-Tenant-Datasource-ID headers are required`.

## Debugging Steps

### Step 1: Verify Tenant Scope is Cached

Open the browser DevTools Console and run:
```javascript
console.log('Tenant:', JSON.parse(localStorage.getItem('selected_tenant')));
console.log('Datasource:', JSON.parse(localStorage.getItem('selected_datasource')));
```

**Expected Output**: Both should have an `id` field. If either is null or missing, the tenant scope is not selected.

**If scope is missing**: Use the tenant picker in the Fabric Builder shell to select a tenant, product, and datasource.

### Step 2: Check Network Request Headers

1. Open DevTools → **Network** tab
2. Click **SAVE & APPLY** button on the Entity Config page
3. Find the `POST` request to `/api/entity-schema`
4. Click on it and check **Request Headers**

**Expected Headers**:
```
X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6
X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0
```

**If headers are missing**: The tenant scope fetch patch in `setupTenantFetch.ts` is not working correctly. Check browser console for `[setupTenantFetch]` log messages.

### Step 3: Check Browser Console Logs

In the DevTools Console, you should see:
```
[setupTenantFetch] Intercepted request: {...}
[setupTenantFetch] Making request: {...}
[setupTenantFetch] Response received: {...}
[saveEntitySchema] Saving schema: {...}
[saveEntitySchema] Save successful: {...}
```

**If logs are missing or show errors**: Check for error messages about tenant scope.

### Step 4: Verify Backend Received the Request

Check the backend logs:
```bash
docker compose logs backend | grep entity-schema
```

**Expected**: Backend logs should show the INSERT query being executed.

### Step 5: Verify Data Persisted to Database

Connect to the database and check:
```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable
```

Then query:
```sql
SELECT * FROM public.entity_schema 
WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
AND tenant_datasource_id = '982aef38-418f-46dc-acd0-35fe8f3b97b0';
```

**Expected**: Should show a row with your saved schema JSON in the `schema_data` column.

## Common Issues & Solutions

### Issue 1: "Tenant scope not set, rejecting request"

**Cause**: Tenant/datasource not selected before visiting `/config` page.

**Solution**: 
1. Go back to the main dashboard
2. Use the tenant picker to select tenant, product, and datasource
3. Navigate to `/config` again
4. Try saving

### Issue 2: Request returns 400 "X-Tenant-ID and X-Tenant-Datasource-ID headers are required"

**Cause**: Headers are not being sent.

**Solution**:
- Verify localStorage has cached selection (Step 1)
- Check browser console for `[setupTenantFetch]` logs
- Restart browser to ensure `setupTenantFetch.ts` is loaded before first API call

### Issue 3: Request succeeds but data doesn't appear in database

**Cause**: Request reached backend but database transaction failed.

**Solution**:
- Check backend logs for SQL errors
- Verify the referenced tenant and datasource IDs exist in the database:
```sql
SELECT * FROM public.tenants WHERE id = '910638ba-a459-4a3f-bb2d-78391b0595f6';
SELECT * FROM public.tenant_product_datasource WHERE id = '982aef38-418f-46dc-acd0-35fe8f3b97b0';
```
- Check if there are foreign key constraint issues

## Recent Changes

I've added enhanced logging to:
- `frontend/src/api/entitySchema.ts`: Logs what schema is being saved and results
- `frontend/src/pages/EntityConfigPage.tsx`: Logs tenant scope checks before saving

Run through the debugging steps above with these new logs to identify where the issue is.

## Key Files

- Frontend save function: `frontend/src/api/entitySchema.ts`
- Page component: `frontend/src/pages/EntityConfigPage.tsx`
- Tenant scope utilities: `frontend/src/utils/tenantScope.ts`
- Fetch patch: `frontend/src/setupTenantFetch.ts`
- Backend endpoint: `backend/internal/api/api.go:711`
- Database migration: `backend/migrations/000028_create_entity_schema_table.sql`
