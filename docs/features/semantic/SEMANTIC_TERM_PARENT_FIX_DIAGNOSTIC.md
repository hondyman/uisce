# Semantic Term Parent ID Fix - Diagnostic Guide

## Issue Summary
When creating or editing semantic terms with a parent business term, the parent_id should:
1. Be saved to the database
2. Be returned in API responses
3. Display in the UI under "Parent Business Term"
4. Modal should close after successful save

## Implementation Status

### ✅ Backend Implementation (glossary_handler.go)
- **CreateTerm**: Lines 355-560
  - Accepts `parent_id` in request JSON
  - Stores in `catalog_node.parent_id` column
  - Creates `glossary_edges` entry for tracking parent relationship
  - Returns `parent_id` in response

- **UpdateTerm**: Lines 565-730
  - Accepts `parent_id` in updates map
  - Handles both SET and NULL operations
  - Manages edge creation/deletion based on parent_id changes
  - Returns updated `parent_id`

### ✅ Frontend Form Implementation (TermForm.tsx)
- **Parent Selector**: Lines 337-360
  - Autocomplete dropdown for selecting business term parent
  - Shows only when `catalog_type === 'semantic_term'`
  - Properly handles null/undefined values
  - **NEW**: Debug logging added:
    - Line 338: Logs current parent_id and business terms count
    - Line 344: Logs parent selection changes with ID and name

- **Save Payload**: Lines 231-237
  - Includes `parent_id` in termData for semantic terms
  - Converts null to empty string for backend
  - **NEW**: Enhanced logging:
    - Logs node_name, catalog_type, parent_id, and hasParent flag

- **Modal Close**: Line 244
  - Calls `handleClose()` after successful `await onSave()`
  - Should close modal immediately after save

### ✅ Frontend Display Implementation (SemanticTermsTab.tsx)
- **Parent Display**: Lines 250-287
  - Shows parent business term in detail view
  - Conditionally renders only if parent_id exists
  - Provides clickable navigation to parent term
  - **DEBUG LOGGING**: Lines 252-267
    - Logs parent_id value from selectedAsset.node
    - Logs business_terms array details
    - Logs ID comparison results for troubleshooting

## Testing Checklist

### Step 1: Start the Application
```bash
# Start backend
export DATABASE_URL='postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable'
go run ./backend/cmd/server

# In another terminal, start frontend (if not already running)
npm start
```

### Step 2: Access Business Glossary
1. Navigate to http://localhost:3000/glossary (or your frontend URL)
2. Select a tenant and datasource from the tenant picker
3. Verify no "Select a tenant" warning appears

### Step 3: Create Semantic Term with Parent
1. Click the "Semantic Terms" tab
2. Click the "+" button to create new semantic term
3. Fill in:
   - **Name**: "Test Semantic Term" (any name)
   - **Description**: (optional)
   - **Parent Business Term**: Select any business term from dropdown
   - **Properties**: Fill in any required fields
4. Click "Save"

### Step 4: Check Browser Console
Open DevTools (F12) → Console tab and look for logs starting with:

#### From TermForm.tsx:
```
[TermForm] Parent selector - formData.parent_id: <UUID or null>
[TermForm] Parent selection changed to: <UUID> (name: <Business Term Name>)
[TermForm.handleSave] {...}
[TermForm] Parent selector render - formData.parent_id: <UUID>
```

#### Expected flow for new semantic term with parent selected:
1. Parent selector renders with parent_id showing the selected UUID
2. User clicks Save
3. `[TermForm.handleSave]` logs with parent_id showing the UUID and hasParent: true
4. Modal should close (look for form to disappear)
5. Verify success snackbar message appears

### Step 5: Check Backend Console
Look for logs in the backend terminal starting with:
```
[DEBUG CreateTerm] catalog_type=semantic_term, parent_id=<pointer>, provided ParentID=<UUID>
[DEBUG CreateTerm Response] ID=<semantic-term-id>, ParentID=<parent-id>, CatalogType=semantic_term
```

### Step 6: Verify Parent Displays
1. Click on the newly created semantic term in the list
2. Scroll to "Metadata" section
3. Should see "Parent Business Term: <Business Term Name>" as a clickable link
4. Browser console should show:
```
[DEBUG PARENT] selectedAsset.node.parent_id: <UUID>
[DEBUG PARENT] First 5 business terms: [...]
[DEBUG PARENT] Found parentBusinessTerm: <Business Term Name>
```

### Step 7: Test Edit Flow
1. Click the created semantic term to view its details
2. Click Edit (if available) or create another term
3. Select the same parent business term again
4. Click Save
5. Verify modal closes and parent still displays

## Debugging: What to Look For

### Issue: Modal stays open after save
- **Check**: Browser console for `[TermForm.handleSave]` logs
- **Expected**: Should see successful save log, then form should close
- **If stuck**: Check if `onSave` promise is being awaited correctly

### Issue: Parent not showing in UI
- **Check 1**: Browser console for `[TermForm] Parent selector` logs
  - If parent_id is null/empty, user didn't select a parent
  - If parent_id exists but UI shows "Reference not found", parent UUID doesn't match any business_term.id

- **Check 2**: Backend console for `[DEBUG CreateTerm]` logs
  - Verify parent_id is not null when creating
  - Verify response includes parent_id

- **Check 3**: Backend database directly
  ```bash
  psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable
  SELECT node_id, node_name, catalog_type, parent_id FROM catalog_node 
  WHERE catalog_type='semantic_term' LIMIT 10;
  ```
  - Verify parent_id column has UUID values (not NULL/empty)

### Issue: Parent selector shows no options
- **Check**: Browser console for `[TermForm] Parent selector` logs
- **If businessTerms count is 0**: Business terms not loaded
  - Verify business terms exist in database
  - Check useBusinessTerms() hook is fetching data

## Data Flow Diagram

```
User fills form with parent business term
       ↓
TermForm.onChange updates formData.parent_id
       ↓
User clicks Save → handleSave()
       ↓
termData = { ..., parent_id: <UUID or ''> }
       ↓
POST /api/glossary/terms with parent_id in body
       ↓
Backend CreateTerm/UpdateTerm receives parent_id
       ↓
INSERT/UPDATE catalog_node.parent_id column
       ↓
Response includes parent_id
       ↓
Apollo cache invalidated (query refetches)
       ↓
GET_ALL_SEMANTIC_DATA returns updated semantic_term with parent_id
       ↓
SemanticTermsTab displays parent if parent_id exists in node
```

## Files Modified (This Session)

1. **frontend/src/components/TermForm.tsx**
   - Added enhanced debug logging to parent selector
   - Improved parent_id value handling in handleSave
   - Added selection change logging

2. **backend/internal/api/glossary_handler.go** (from previous session)
   - CreateTerm: Persists parent_id in INSERT
   - UpdateTerm: Persists parent_id in UPDATE
   - Both include debug logging

3. **frontend/src/pages/glossary/SemanticTermsTab.tsx** (from previous session)
   - Displays parent business term with clickable link
   - Includes debug logging for parent lookup

## Quick Commands for Troubleshooting

### Clear browser cache and reload
```bash
Ctrl+Shift+Delete (or Cmd+Shift+Delete on Mac) → Clear all → Reload page
```

### Check backend server logs in real-time
```bash
cd /Users/eganpj/GitHub/semlayer
export DATABASE_URL='postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable'
go run ./backend/cmd/server 2>&1 | grep -E "DEBUG|Error"
```

### Query recent semantic terms in database
```bash
export PGPASSWORD=postgres
psql -h host.docker.internal -U postgres -d alpha -c "
  SELECT cn.id, cn.node_name, cn.catalog_type_name, cn.parent_id, cn.created_at 
  FROM catalog_node cn 
  WHERE cn.catalog_type_name='semantic_term' 
  ORDER BY cn.created_at DESC 
  LIMIT 5;
" | cat
```

### Check glossary_edges for parent relationships
```bash
export PGPASSWORD=postgres
psql -h host.docker.internal -U postgres -d alpha -c "
  SELECT ge.id, ge.subject_node_id, ge.object_node_id, ge.relationship_type 
  FROM glossary_edges ge 
  WHERE ge.relationship_type='business_term_to_semantic_term' 
  LIMIT 10;
" | cat
```

## Expected Success Criteria

✅ All of these should pass:
1. Modal closes immediately after clicking Save
2. New semantic term appears in the list
3. Clicking the term shows parent business term in metadata
4. Parent term is clickable and navigates to business term detail
5. Backend logs show parent_id with UUID (not empty/null)
6. Database query shows semantic_term.parent_id populated
7. Edit existing semantic term preserves parent_id relationship
