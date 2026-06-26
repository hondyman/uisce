# Add Relationship Feature - Validation & Testing Guide

## Pre-Deployment Validation

### ✅ Code Compilation

```bash
# Backend: Should have NO errors
cd backend
go build -o api-gateway ./cmd/api-gateway
# Expected: Binary created without errors

# Frontend: Should have NO errors (only CSS lint warnings are acceptable)
cd ../frontend
npm run build
# Expected: Build succeeds, only pre-existing CSS warnings
```

### ✅ Git Changes

```bash
# Review what changed
git diff backend/internal/api/api.go
git diff frontend/src/api/relationships.ts
git diff frontend/src/components/relationship/RelatedObjectsTab.tsx

# Expected: See all changes from ADD_RELATIONSHIP_CHANGES_SUMMARY.md
```

---

## Unit Test Cases

### Test Case 1: Apply Valid Relationship

**Preconditions:**
- Backend running at `http://localhost:8080`
- Frontend running at `http://localhost:3000`
- Tenant and datasource selected
- Entity has discoverable relationships

**Steps:**
```
1. Navigate to EntityDetailsPage
2. Click on an entity name to load details
3. Scroll to or click "Related Objects" tab
4. Wait for relationships to load
5. Click blue "Apply" button on first relationship
```

**Expected Results:**
- Button changes text from "Apply" to "Applying..." with loading icon
- After 1-2 seconds, button turns green
- Button text changes from "Applying..." to "Applied" with checkmark icon
- Button becomes disabled (cursor-default)
- No errors in browser console (F12)

**Validation Queries:**
```sql
-- Check new edge was created
SELECT id, source_node_id, target_node_id, relationship_type, created_by, created_at
FROM catalog_edge
WHERE created_by = 'user'
ORDER BY created_at DESC
LIMIT 5;

-- Expected: New row with created_by = 'user' and today's date
```

---

### Test Case 2: Apply Multiple Relationships

**Preconditions:**
- Entity has 3+ discoverable relationships

**Steps:**
```
1. Navigate to Related Objects tab
2. Apply relationship #1 (click button, wait for success)
3. Apply relationship #2 (click button, wait for success)
4. Apply relationship #3 (click button, wait for success)
```

**Expected Results:**
- Each button independently shows "Applying..." then "Applied"
- First relationship remains green while applying others
- All 3 edges created in database
- No conflicts or errors

**Validation:**
```sql
SELECT COUNT(*) as applied_count
FROM catalog_edge
WHERE created_by = 'user' 
  AND created_at > NOW() - INTERVAL '5 minutes';
-- Expected: 3 (or however many you applied)
```

---

### Test Case 3: Error Handling - Invalid Tenant

**Preconditions:**
- Tenant scope NOT selected (localStorage cleared)

**Steps:**
```
1. Clear browser localStorage: localStorage.clear()
2. Reload page
3. Navigate to Related Objects tab
4. Try to click Apply button (if available)
```

**Expected Results:**
- Error alert appears explaining invalid request
- Button does NOT turn green
- Error message is clear and actionable
- Browser console shows detailed error

---

### Test Case 4: Error Handling - Invalid Entity

**Preconditions:**
- Entity name in database doesn't match frontend

**Steps:**
```
1. Manually edit entity name in catalog_node (only for testing!)
2. Navigate to Related Objects tab
3. Click Apply button
```

**Expected Results:**
- Error message appears: "Failed to apply relationship: ..."
- Button does NOT turn green
- Database shows no new edge created

---

### Test Case 5: No Relationships Available

**Preconditions:**
- Entity has NO semantic terms or FK mappings

**Steps:**
```
1. Navigate to an entity with no relationships
2. Click Related Objects tab
```

**Expected Results:**
- Shows message: "No entities available to relate to"
- Subtext: "Verify that semantic terms are mapped to columns..."
- No error styling
- No buttons to click

---

### Test Case 6: Loading State

**Preconditions:**
- Slow network (can simulate with DevTools throttling)

**Steps:**
```
1. Open DevTools (F12)
2. Go to Network tab
3. Set throttle to "Slow 3G" or "Slow 4G"
4. Navigate to Related Objects tab
5. Click Apply button while relationships are loading
```

**Expected Results:**
- Tab shows spinner while loading
- Apply buttons disabled until relationships load
- Button correctly changes through states even with slow network

---

## Console Log Validation

Open browser DevTools (F12 → Console) and look for these expected logs:

**When fetching relationships:**
```
🔗 Fetching relationships for entity: {entityName, tenantId, datasourceId}
✅ Relationships fetched: [array of relationships]
```

**When applying relationship:**
```
🔗 Applying relationship: {sourceEntity, targetEntity, edgeType, cardinality}
✅ Relationship applied: {status: "applied", edge_id: "..."}
```

**When errors occur:**
```
Error fetching relationships: [error message]
Error applying relationship: [error message]
```

---

## Database State Validation

After applying relationships, verify database state:

```sql
-- 1. Check edges were created
SELECT 
    ce.id,
    src.node_name as source,
    tgt.node_name as target,
    cet.edge_type_name as edge_type,
    ce.cardinality,
    ce.created_by,
    ce.created_at
FROM catalog_edge ce
JOIN catalog_node src ON ce.source_node_id = src.id
JOIN catalog_node tgt ON ce.target_node_id = tgt.id
JOIN catalog_edge_type cet ON ce.edge_type_id = cet.id
WHERE ce.created_by = 'user'
  AND ce.created_at > NOW() - INTERVAL '1 hour'
ORDER BY ce.created_at DESC;

-- Expected: Rows for each applied relationship

-- 2. Check tenant scoping
SELECT DISTINCT cd.id, cd.tenant_id, COUNT(*) as edge_count
FROM catalog_edge ce
JOIN catalog_datasource cd ON ce.tenant_datasource_id = cd.id
WHERE ce.created_by = 'user'
GROUP BY cd.id, cd.tenant_id;

-- Expected: Edges only in correct tenant/datasource

-- 3. Verify data integrity
SELECT 
    ce.id,
    (src.id IS NOT NULL) as source_exists,
    (tgt.id IS NOT NULL) as target_exists,
    (cet.id IS NOT NULL) as edge_type_exists
FROM catalog_edge ce
LEFT JOIN catalog_node src ON ce.source_node_id = src.id
LEFT JOIN catalog_node tgt ON ce.target_node_id = tgt.id
LEFT JOIN catalog_edge_type cet ON ce.edge_type_id = cet.id
WHERE ce.created_by = 'user'
  AND ce.created_at > NOW() - INTERVAL '1 hour';

-- Expected: All TRUE (all foreign keys resolve correctly)
```

---

## Performance Baseline

Measure and record these metrics:

### Load Times
```javascript
// In browser console while loading relationships
console.time('load-relationships');
// [wait for load]
console.timeEnd('load-relationships');
// Expected: 200-800ms for typical datasets
```

### Apply Times
```javascript
console.time('apply-relationship');
// [click Apply and wait]
console.timeEnd('apply-relationship');
// Expected: 300-1000ms including network round-trip
```

### Database Query Performance
```sql
-- Measure query time
EXPLAIN ANALYZE
SELECT src.id, tgt.id, cet.id
FROM catalog_node src, catalog_node tgt, catalog_edge_type cet
WHERE src.node_name = 'YourEntity'
  AND src.tenant_datasource_id = '123'
  AND tgt.node_name = 'RelatedEntity'
  AND tgt.tenant_datasource_id = '123'
  AND cet.edge_type_name = 'entity_relationship';

-- Expected: < 50ms for typical datasets
```

---

## Regression Testing

Ensure existing functionality still works:

### Test 1: Discovery Still Works

```
1. Navigate to Related Objects tab
2. Should see list of relationships (if they exist)
3. No errors
```

### Test 2: Diagram View Still Works

```
1. Navigate to Related Objects tab
2. Switch to "Diagram View" toggle
3. Should see SVG visualization
4. Zoom/pan should work (if implemented)
```

### Test 3: Card View Pagination (if implemented)

```
1. With 10+ relationships
2. Card view should show all or paginate properly
3. Apply buttons should work on all cards
```

### Test 4: Tenant Switching

```
1. Select different tenant in dropdown
2. Related Objects should update
3. Apply relationships should work in new tenant
```

---

## Security Validation

### Tenant Isolation

```bash
# Get two different tenant IDs
T1=$(psql -tc "SELECT id FROM tenant LIMIT 1")
T2=$(psql -tc "SELECT id FROM tenant WHERE id != '$T1' LIMIT 1")

# Try to apply relationship from tenant 1 as tenant 2
curl -X POST http://localhost:8080/api/relationships/apply \
  -H "X-Tenant-ID: $T2" \
  -H "X-Tenant-Datasource-ID: <some-id-from-T1>" \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "'$T1'",
    "datasourceId": "<T1-datasource>",
    "sourceEntity": "Entity1",
    "targetEntity": "Entity2",
    "edgeType": "entity_relationship",
    "cardinality": "One-to-Many",
    "fkColumn": "",
    "confidence": 0.8
  }'

# Expected: 400 error "Invalid tenant or datasource"
# Edge should NOT be created in database
```

### Field Validation

```bash
# Try with missing required field
curl -X POST http://localhost:8080/api/relationships/apply \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "'$TENANT_ID'",
    "datasourceId": "$DATASOURCE_ID",
    # Missing sourceEntity
    "targetEntity": "Entity2"
  }'

# Expected: 400 error "Missing required fields"
```

---

## Browser Compatibility

Test in these browsers:

- [ ] Chrome/Edge (latest)
- [ ] Firefox (latest)
- [ ] Safari (latest)
- [ ] Mobile Safari (iOS)

Expected: All show correct styling and functionality.

---

## Accessibility Validation

```javascript
// In browser console
// 1. Check button has accessible label
const btn = document.querySelector('button[title]');
console.log(btn.title); // Should not be empty

// 2. Check color contrast
// Visual: Blue button should be readable
// Use: https://webaim.org/resources/contrastchecker/

// 3. Check keyboard navigation
// Manual: Can you tab to Apply button and press Enter?
// Expected: Yes
```

---

## Final Sign-Off Checklist

- [ ] All code changes compile without errors
- [ ] Browser console has no errors or warnings (CSS lint is okay)
- [ ] Can apply relationship and see button change to green
- [ ] Database shows new edges created with correct data
- [ ] Multiple relationships can be applied independently
- [ ] Error handling works for invalid requests
- [ ] "No relationships" message shows for entities without relationships
- [ ] Tenant scoping prevents cross-tenant data leaks
- [ ] No performance degradation compared to baseline
- [ ] Existing features (discovery, view switching) still work
- [ ] Mobile/responsive layout looks good
- [ ] Accessibility is maintained

---

## Known Limitations

1. **Editing Applied Relationships:** Currently cannot edit or delete applied relationships through the UI
2. **Batch Operations:** Cannot apply multiple relationships at once
3. **Undo:** No undo functionality once relationship is applied
4. **Suggestions:** ML-based suggestions not yet implemented (stub only)

These are acceptable for MVP and can be added in future iterations.

---

## Support & Escalation

If issues found during validation:

1. **Check logs first**
   - Browser console (F12)
   - Backend service logs
   - Database query logs

2. **Verify data**
   - Run SQL queries above
   - Ensure semantic terms exist
   - Ensure FKs exist

3. **Test isolation**
   - Clear browser cache
   - Restart services
   - Try different entity

4. **Escalate if needed**
   - Document exact steps to reproduce
   - Include error messages and logs
   - Provide test data/tenant IDs

---

## Rollback Procedure

If critical issues found:

```bash
# 1. Revert changes
git checkout backend/internal/api/api.go \
            frontend/src/api/relationships.ts \
            frontend/src/components/relationship/RelatedObjectsTab.tsx

# 2. Rebuild
cd backend && go build -o api-gateway ./cmd/api-gateway
cd ../frontend && npm run build

# 3. Restart services
systemctl restart semlayer-api
# Deploy frontend build

# 4. Verify
curl http://localhost:8080/health
# Should return to previous behavior
```

---

## Next Steps After Approval

1. **Merge to main branch**
2. **Tag release** (e.g., v1.2.3)
3. **Update CHANGELOG** with this fix
4. **Deploy to staging** for final integration tests
5. **Deploy to production** with monitoring enabled
6. **Monitor logs** for first 24 hours

