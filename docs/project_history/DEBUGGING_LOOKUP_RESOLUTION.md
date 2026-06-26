# Debugging Category Lookup Resolution in Business Terms Tree

## Issue Summary
Business terms are showing UUIDs (e.g., `1f6dc23e-2253-4106-add1-168b23b4247d`) instead of friendly lookup names in the category hierarchy.

## Root Cause Analysis
The lookup resolution depends on several components working together:

### 1. Node Type Detection
- **File**: `BusinessTermsTree.tsx` line 73-79
- **Current Logic**: Matches business term node type case-insensitively
- **Status**: ✅ Enhanced to handle "business_term", "Business Term", and variations

### 2. Property Metadata Fetching
- **File**: `usePropertyLookupMaps.ts`
- **What it does**: Fetches node type properties and identifies which have `lookup_id`
- **Expected**: Properties like `category_1`, `category_2`, `category_3` should have `lookup_id` values
- **Debug**: Look at browser DevTools → Network → calls to `/api/node-types/{id}/properties`

### 3. Lookup Values Fetching
- **File**: `usePropertyLookupMaps.ts` line 19-23
- **What it does**: For each property with `lookup_id`, calls `/api/lookups/{lookup_id}/values`
- **Expected Response**: Array of `{id, label}` objects
- **Debug**: Check Network tab for `/api/lookups/{id}/values` requests

### 4. Map Building and Resolution
- **File**: `BusinessTermsTree.tsx` line 111-121 (`resolveValue` function)
- **What it does**: 
  1. Checks if value is a UUID
  2. Searches for lookup map by property key name
  3. Falls back to nodeNameMap (maps business term IDs to names)
  4. Returns the UUID if no match found

## How to Debug

### Step 1: Check Browser Console Logs
1. Open Firefox/Chrome DevTools (F12)
2. Go to **Console** tab
3. Look for messages starting with `[BusinessTermsTree]`:
   - "Business term node type:" → Shows detected node type
   - "Lookup maps keys:" → Shows available property lookups
   - "Could not resolve UUID:" → Shows which UUIDs couldn't be resolved
   - "Resolved {key}[UUID] -> NAME" → Shows successful resolutions

### Step 2: Check Network Requests
In DevTools **Network** tab, filter by XHR and look for:

#### a) Node Type Properties
```
GET /api/node-types/{nodeTypeId}/properties?tenant_id=...
```
**Expected response**: Array with objects like:
```json
[
  {"id": "...", "name": "category_1", "lookup_id": "lookup-uuid-1", ...},
  {"id": "...", "name": "category_2", "lookup_id": "lookup-uuid-2", ...}
]
```

**Issue if**: 
- Property names are different (e.g., `category` instead of `category_1`)
- `lookup_id` fields are `null` or missing

#### b) Lookup Values
```
GET /api/lookups/{lookup_id}/values?tenant_id=...
```
**Expected response**: Array like:
```json
{
  "items": [
    {"id": "409e2ab2-e6b2-44b6-bff7-2634614f5a30", "label": "Customer", "name": "Customer"},
    {"id": "1f6dc23e-2253-4106-add1-168b23b4247d", "label": "Product", "name": "Product"}
  ]
}
```

**Issue if**:
- Response is empty or `items` is missing
- `id` and `label` fields have different names

### Step 3: Verify Business Term Data Structure
In the console, run:
```javascript
// Find a business term in the component state
const term = businessTerms[0];
console.log('Business Term Properties:', term.properties);
// Output should show: {category_1: "uuid", category_2: "uuid", ...}
```

## Common Causes & Fixes

### ❌ Lookup IDs not in properties response
- **Backend issue**: Node type properties don't have `lookup_id` fields set
- **Fix**: Check database: 
  ```sql
  SELECT id, name, lookup_id FROM node_type_property 
  WHERE node_type_id = '...' AND name LIKE 'category%';
  ```
- If `lookup_id` is NULL, update the records in the database

### ❌ Category property names mismatch
- **Issue**: Properties are named `category`, `level_1`, `main_category` instead of `category_1`, `category_2`, etc.
- **Fix**: Update the candidate keys list in `BusinessTermsTree.tsx` line 122-123:
  ```typescript
  const level1 = resolveValue(
    ['category_1', 'category', 'level_1', 'main_category'],  // Add your property names here
    props.category_level_1 || props.category1 || props.category_1 || props.category || 'Uncategorized'
  );
  ```

### ❌ Lookup values not being fetched
- **Issue**: API endpoint not working or returns wrong format
- **Fix**: Test directly via curl:
  ```bash
  curl -X GET "http://localhost:8080/api/lookups/lookup-uuid/values?tenant_id=tenant-uuid" \
    -H "X-Tenant-ID: tenant-uuid"
  ```

### ❌ React Query caching too aggressively
- **Issue**: Old data cached, new lookups not shown
- **Fix**: In DevTools, clear cache and refresh:
  ```javascript
  // In browser console
  localStorage.clear();
  location.reload();
  ```
- Or check `usePropertyLookupMaps.ts` cache settings (currently 30min stale, 1hr cache)

## Semantic Terms Datasource Grouping

### What's New
✅ Added view toggle in SemanticTermsTab header:
- **📋 Flat** - Shows all terms in a list (original view)
- **🗂 Datasource** - Groups terms by datasource with edit/delete actions

### How It Works
- **File**: `SemanticTermsTab.tsx` line 29 (viewMode state)
- **Component**: `DatasourceGroupedTerms` (lines 656-756)
- **Grouping**: Uses `tenant_datasource_id` from term or `properties.datasource_id`
- **Datasource Names**: Fetched via `useDataSources()` hook and resolved to display names

### If Datasource View Doesn't Work
1. Check browser console for errors
2. Verify datasources are being fetched: Check Network tab for `/api/datasources` calls
3. Verify terms have `tenant_datasource_id` field populated:
   ```javascript
   const term = semanticTerms[0];
   console.log('Datasource ID:', term.tenant_datasource_id);
   ```

## Quick Validation Checklist

- [ ] Open Business Glossary
- [ ] Navigate to **Business Terms** tab
- [ ] Open browser DevTools → Console
- [ ] Look for `[BusinessTermsTree]` debug messages
- [ ] Check if categories show UUIDs or names
- [ ] Go to **Semantic Terms** tab
- [ ] Look for **🗂 Datasource** button in header
- [ ] Click it and verify terms are grouped by datasource

## Next Steps

1. **Check console logs** - Follow Step 1-3 above
2. **Report findings** with screenshots of:
   - Console logs showing UUID resolution attempts
   - Network request/response for node-types and lookups endpoints
   - Term data structure
3. **Update backend** if lookup_id fields are missing
4. **Clear cache** and test again if changes made

---

**Technical Debt**: Consider moving lookup resolution to backend to reduce client-side complexity and improve performance.
