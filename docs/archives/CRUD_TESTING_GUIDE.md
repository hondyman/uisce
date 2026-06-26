# CRUD Operations Testing Guide

## Critical Fixes Applied

### 1. Fixed URL Routing (Tenant Scope Issue)
**Problem**: All API calls were using hardcoded `http://localhost:8080/api/...` URLs which bypassed the tenant scope shim.

**Solution**: Changed all fetch calls to use relative URLs (`/api/...`) so they go through the tenant scope middleware that automatically adds tenant_id and datasource_id.

**Files Changed**:
- `frontend/src/components/semantic-mapper/BusinessTermMapper.tsx`

**Affected Functions**:
- `handleAcceptSuggestion` - Create business term and edge
- `handleCreateBusinessTerm` - Create custom business term
- `handleGenerateAllSuggestions` - Fetch suggestions
- `initializeData` - Load existing edges

### 2. Enhanced Error Logging
Added comprehensive console logging at every step:
- `[handleCreateBusinessTerm]` - Business term creation
- `[handleAcceptSuggestion]` - Accept suggestion flow
- `[handleSave]` - Save mapping (create edge)
- `[handleRejectSuggestion]` - Reject suggestion with feedback
- `[handleGenerateAllSuggestions]` - Suggestion generation
- `[initializeData]` - Initial data loading

### 3. Reject Suggestion Persistence
**Problem**: Rejected suggestions might reappear on next suggestion generation.

**Solution**: 
- Enhanced `handleRejectSuggestion` to properly record feedback
- Added try-catch and error handling
- Changed toast message to confirm: "This suggestion won't appear again"
- Backend uses suggestion feedback to filter out rejected pairs

## Testing Checklist

### Prerequisites
1. Start services:
```bash
cd /Users/eganpj/GitHub/semlayer
docker compose up -d
cd frontend
npm run dev
```

2. Open browser console (F12) to see detailed logs
3. Navigate to Business Term Mapper at http://localhost:5173

### Test 1: Create Business Term (Custom Entry)
**Steps**:
1. Expand any unmapped semantic term row
2. Click "Create Custom" button
3. Fill in:
   - Business Term Name: `test_customer_email`
   - Category: `Customer Data`
   - Description: `Test customer email address`
4. Click "Create & Map"

**Expected Console Logs**:
```
[handleCreateCustomTerm] Creating business term: ...
[handleCreateBusinessTerm] Creating: { termName, category, description }
[handleCreateBusinessTerm] Created successfully: { node_id, term_name, ... }
```

**Expected Results**:
- ✅ Success toast: "Created business term: Test Customer Email"
- ✅ Business term appears in the selected term chip
- ✅ Form clears
- ✅ Term appears in dropdown for other rows

**If It Fails**:
- Check console for error messages
- Look for tenant scope errors (missing tenant_id/datasource_id)
- Verify backend is running: `curl http://localhost:8080/api/roles`

### Test 2: Save Mapping (Create Edge)
**Steps**:
1. Select a business term from dropdown (or use one from Test 1)
2. Click "Save Mapping" button

**Expected Console Logs**:
```
[handleSave] Creating edge: { semanticTermId, businessTermId, businessTermName }
[handleSave] Edge created successfully: { edge_id, success, ... }
```

**Expected Results**:
- ✅ Button shows spinner briefly
- ✅ Success toast: "Created business term mapping for ..."
- ✅ Row status changes to "Mapped" (green)
- ✅ Save button disappears
- ✅ Mapping persists on page refresh

**If It Fails**:
- Check console for "Failed to create business term edge"
- Look for 400/500 HTTP errors
- Verify payload format in Network tab (should have subject_node_id, object_node_id, edge_type_id)

### Test 3: Generate Suggestions
**Steps**:
1. Click "Generate Suggestions" button at top
2. Wait for suggestions to load

**Expected Console Logs**:
```
[handleGenerateAllSuggestions] Got X suggestions for <semantic_term_id>
```

**Expected Results**:
- ✅ Info toast: "Generating suggestions for X terms..."
- ✅ Success toast: "Generated X suggestions for Y terms"
- ✅ "Suggestions (N)" chip appears on rows with suggestions
- ✅ Click chip to expand row and see suggestions

**If It Fails**:
- Check console for fetch errors
- Verify suggestions endpoint: `/api/semantic-terms/{id}/suggest-business-terms`
- Ensure tenant scope is set (should see in Network tab headers)

### Test 4: Accept Suggestion
**Steps**:
1. Generate suggestions (Test 3)
2. Expand a row with suggestions
3. Click "Accept" on a suggestion

**Expected Console Logs**:
```
[handleAcceptSuggestion] Creating new business term: ... (if term doesn't exist)
[handleAcceptSuggestion] Business term created: ...
[handleAcceptSuggestion] Creating edge: { businessTermId, semanticTermId }
[handleAcceptSuggestion] Edge created successfully: ...
```

**Expected Results**:
- ✅ If business term doesn't exist, it's created automatically
- ✅ Edge is created automatically
- ✅ Row status changes to "Mapped"
- ✅ Other suggestions for that term are auto-rejected
- ✅ Success toast appears

**If It Fails**:
- Check for business term creation errors
- Check for edge creation errors
- Verify both steps complete (create term, then create edge)

### Test 5: Reject Suggestion (Critical - Must Never Reappear)
**Steps**:
1. Generate suggestions
2. Expand a row with suggestions
3. Click "Reject" on a specific suggestion
4. Note the semantic term and business term names
5. Click "Generate Suggestions" again
6. Expand the same row

**Expected Console Logs**:
```
[handleRejectSuggestion] Rejecting: { semantic_term_id, business_term_name, business_term_id }
[handleRejectSuggestion] Feedback recorded successfully
[handleRejectSuggestion] Removed suggestion, remaining: X
```

**Expected Results**:
- ✅ Success toast: "Rejected: ... - This suggestion won't appear again"
- ✅ Suggestion disappears from UI immediately
- ✅ **When generating suggestions again, the rejected suggestion does NOT appear**
- ✅ Backend filters out rejected pairs based on feedback

**How to Verify Backend Filtering**:
```bash
# Get suggestions for a semantic term (replace IDs)
curl -i 'http://localhost:8080/api/semantic-terms/{semantic_term_id}/suggest-business-terms?tenant_id=...&datasource_id=...' \
  -H 'X-Tenant-ID: ...' \
  -H 'X-Tenant-Datasource-ID: ...'
```

The rejected business term should NOT appear in the response.

**If Rejection Doesn't Persist**:
- Check suggestion_feedback table in database
- Verify feedback was recorded (check console logs)
- Test backend endpoint directly to see if it filters
- Check backend suggestion service code

### Test 6: Load Existing Mappings on Refresh
**Steps**:
1. Create some mappings (Tests 2 or 4)
2. Refresh the page (F5)

**Expected Console Logs**:
```
[initializeData] Loading semantic and business terms...
[initializeData] Loaded X semantic terms, Y business terms
[initializeData] Loaded Z existing edges
[initializeData] Created edge map with Z entries
```

**Expected Results**:
- ✅ Mapped rows show "Mapped" status
- ✅ Business terms are displayed in mapped rows
- ✅ Statistics counts are correct

**If It Fails**:
- Check if edges are loading (console logs)
- Verify GET /api/business-term-edges returns data
- Check edge map creation logic

## Common Issues and Solutions

### Issue: "Failed to create business term: Request blocked"
**Cause**: Tenant scope not set or request bypassed tenant shim
**Solution**: 
- Ensure you've selected tenant/product/datasource in UI
- Check localStorage: `selected_tenant`, `selected_product`, `selected_datasource`
- All URLs should be relative (`/api/...`) not absolute

### Issue: "Failed to create edge: 400 Bad Request"
**Cause**: Invalid payload format or missing required fields
**Solution**:
- Check Network tab payload
- Should have: `subject_node_id`, `object_node_id`, `edge_type_id`, `relationship_type`
- Verify UUIDs are valid

### Issue: Rejected suggestions reappear
**Cause**: Feedback not recorded or backend not filtering
**Solution**:
- Check console for `[handleRejectSuggestion]` logs
- Verify POST to `/api/business-term/suggestion-feedback` succeeds
- Check database table `suggestion_feedback`
- Backend should use feedback to filter suggestions

### Issue: Network errors or CORS issues
**Cause**: Services not running or proxy misconfigured
**Solution**:
```bash
# Check services
docker compose ps

# Check backend health
curl http://localhost:8080/api/roles

# Check frontend proxy
curl http://localhost:5175/api/roles
```

## Database Queries for Verification

### Check Business Terms Created
```sql
SELECT id, term_name, properties, created_at 
FROM catalog_node 
WHERE data_type = 'business_term' 
ORDER BY created_at DESC 
LIMIT 10;
```

### Check Edges Created
```sql
SELECT 
  ce.id,
  ce.source_node_id,
  ce.target_node_id,
  ce.edge_type_id,
  cn_source.term_name as business_term,
  cn_target.term_name as semantic_term,
  ce.created_at
FROM catalog_edge ce
JOIN catalog_node cn_source ON cn_source.id = ce.source_node_id
JOIN catalog_node cn_target ON cn_target.id = ce.target_node_id
WHERE ce.edge_type_id = '3be9d6ae-1598-4628-a3dd-b606921a9193'
ORDER BY ce.created_at DESC
LIMIT 10;
```

### Check Suggestion Feedback
```sql
SELECT 
  semantic_term_id,
  business_term_name,
  action,
  reason,
  created_at
FROM suggestion_feedback
ORDER BY created_at DESC
LIMIT 20;
```

## Success Criteria

All CRUD operations must:
1. ✅ Complete without errors (check console logs)
2. ✅ Show appropriate success/error toasts
3. ✅ Update UI state immediately
4. ✅ Persist after page refresh
5. ✅ Work with tenant scope (relative URLs)
6. ✅ Rejected suggestions never reappear

## Next Steps After Testing

If all tests pass:
- Document any edge cases discovered
- Consider adding automated E2E tests
- Add user documentation

If tests fail:
- Paste console logs in issue tracker
- Include Network tab screenshots
- Note exact steps to reproduce
