# Semantic Mapper UI & Edge Creation Fixes

## Changes Made

### 1. Horizontal Stats Display in Business Term Mapper
**File**: `frontend/src/components/semantic-mapper/BusinessTermMapper.tsx` (lines 983-1024)

**Change**: Converted the stats display from vertical (3 columns in a Grid) to horizontal layout using flexbox
- **Before**: Used `Grid container spacing={2}` with 3 `Grid item xs={4}` children, taking up full width vertically
- **After**: Uses `Box sx={{ display: 'flex', gap: 2, mb: 3, flexWrap: 'wrap' }}` with Paper elements using `flex: '1 1 150px'`
- **Benefits**: 
  - Takes up much less vertical space
  - Stats display horizontally on a single line (or wraps on small screens)
  - More space-efficient for the dashboard
  - Still responsive and clickable

### 2. Fixed Create Edge Button Not Working
**File**: `frontend/src/components/semantic-mapper/useSemanticMapper.ts`

**Root Cause**: The `/api/semantic-mappings/edges` endpoint requires `X-Tenant-ID` and `X-Tenant-Datasource-ID` headers for tenant-scoped operations. The `createEdges` function was not including these required headers.

**Changes**:

#### In `createEdges` function (lines 281-308):
- Added code to retrieve tenant scope using `getRequiredTenantScope()`
- Added tenant ID and datasource ID to request headers
- Headers are now sent: `X-Tenant-ID` and `X-Tenant-Datasource-ID`
- Added detailed debug logging for tenant scope information

#### In `replaceMapping` function (lines 373-398):
- Applied the same fix for consistency
- Added tenant headers to the replace endpoint request

**Before**:
```typescript
const res = await fetch(url, {
  method: 'POST', 
  headers: { 'Content-Type': 'application/json' }, 
  credentials: 'include',
  body: JSON.stringify(payload)
});
```

**After**:
```typescript
const headers: Record<string, string> = { 'Content-Type': 'application/json' };
if (tenantId) headers['X-Tenant-ID'] = tenantId;
if (datasourceId) headers['X-Tenant-Datasource-ID'] = datasourceId;

const res = await fetch(url, {
  method: 'POST', 
  headers,
  credentials: 'include',
  body: JSON.stringify(payload)
});
```

## Testing the Fixes

### 1. Test Horizontal Stats Display
- Open the semantic mapper page
- Navigate to the Business Term Mapper tab
- Verify that the three stat cards (Total Semantic Terms, Mapped to Business Terms, Unmapped) are displayed horizontally in a single row
- On mobile/narrow screens, verify they wrap but still maintain horizontal flex layout

### 2. Test Create Edge Button
- Navigate to the semantic mapper page
- Select a datamart or other datasource from the tenant picker
- Wait for columns to load
- Select one or more mappings (check the checkboxes)
- Click the "Create Edges" button
- Verify that:
  - The confirmation dialog appears
  - After confirmation, edges are created successfully
  - Success message is displayed showing "Created X edges"
  - The mappings now show as "Mapped" (edge_exists = true)
  - Backend logs show proper tenant scope being used

## Key Technical Details

- **Tenant Scope Context**: The `getRequiredTenantScope()` function retrieves cached tenant context from localStorage
- **Header Requirements**: The backend `/api/semantic-mappings/edges` endpoint now (after previous fixes) properly handles datamart datasource mapping, but requires headers for proper scope validation
- **Error Handling**: If tenant scope cannot be retrieved, warnings are logged but requests still proceed (with mappings potentially containing the scope information)

## Related Files Modified
- `frontend/src/components/semantic-mapper/BusinessTermMapper.tsx` - Stats layout fix
- `frontend/src/components/semantic-mapper/useSemanticMapper.ts` - Tenant headers in edge creation
