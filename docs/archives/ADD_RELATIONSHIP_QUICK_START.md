# Quick Start: Test Add Relationship Feature

## Prerequisites

- Backend running: `go run ./backend/cmd/api-gateway`
- Frontend running: `npm start` (in frontend directory)
- PostgreSQL running with semlayer database
- Tenant and datasource selected in UI

## 5-Minute Test

### Step 1: Verify Backend is Running

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{"status": "healthy"}
```

### Step 2: Select Tenant in UI

1. Open browser and navigate to your app (usually http://localhost:3000)
2. Look for tenant selector in top-right corner
3. Select:
   - **Tenant**: Choose any available tenant
   - **Product**: Choose associated product
   - **Datasource**: Choose associated datasource
4. Page should refresh and load data

### Step 3: Navigate to Related Objects Tab

1. Go to any Entity Details Page (or browse entities)
2. Find and click the entity you want to test
3. Look for **Related Objects** tab
4. Wait for content to load (should show spinner then cards or "no entities" message)

### Step 4: Test Apply Relationship (If Available)

**If you see relationship cards:**
1. Look at one of the cards
2. Find the blue "Apply" button at the bottom
3. Click it
4. **Expected:** Button changes to "Applying..." then "Applied" (green with checkmark)

**If you see "No entities available to relate to":**
1. This is normal - the entity has no discoverable relationships
2. Verify by checking that semantic terms are mapped to columns in the database
3. Try a different entity that has semantic terms

### Step 5: Verify Success

**Check Browser Console (F12):**
```
🔗 Fetching relationships for entity: {...}
✅ Relationships fetched: [...]
🔗 Applying relationship: {...}
✅ Relationship applied: {...}
```

**Check Database:**
```bash
psql postgresql://postgres:postgres@localhost:5432/semlayer

SELECT * FROM catalog_edge 
WHERE created_by = 'user' 
ORDER BY created_at DESC LIMIT 1;
```

Should see a new edge row with your applied relationship.

---

## Common Issues & Fixes

### Issue: "Cannot read properties of null (reading 'relationships')"

**Fix:**
1. Refresh the page
2. Ensure tenant is selected
3. Check backend logs for SQL errors
4. Open browser console (F12) for detailed error message

### Issue: Apply button doesn't change to "Applying..."

**Check:**
1. Is backend running? `curl http://localhost:8080/health`
2. Is tenant selected? Check localStorage
3. Are there network errors? Check Network tab in DevTools

### Issue: Applied but edge not in database

**Possible causes:**
1. Entity names don't match exactly (case-sensitive)
2. Edge type doesn't exist for relationship type
3. SQL permissions issue

**Debug:**
```bash
# Check if nodes exist
SELECT id, node_name FROM catalog_node 
WHERE node_name IN ('YourEntity', 'RelatedEntity');

# Check if edge_type exists
SELECT * FROM catalog_edge_type 
WHERE edge_type_name = 'entity_relationship';
```

---

## Test Scenarios

### Scenario 1: Entity with Many Relationships

**Expected:**
- Card view shows 3-10 relationship cards
- Diagram view shows central entity with related entities arranged in circle
- Can apply each one independently

**Test:**
```
1. Find entity with semantic terms mapped to multiple columns
2. Navigate to Related Objects tab
3. Apply 2-3 relationships
4. All should succeed without conflicts
```

### Scenario 2: Entity with No Relationships

**Expected:**
- Shows "No entities available to relate to" message
- No errors displayed
- Helpful diagnostic message visible

**Test:**
```
1. Find entity with NO semantic terms or FK mappings
2. Navigate to Related Objects tab
3. Should see appropriate message (not error)
```

### Scenario 3: Missing Tenant Scope

**Expected:**
- Error message about invalid request parameters
- Clear feedback about what's wrong

**Test:**
```
1. Open DevTools → Application → Storage → Local Storage
2. Delete 'selected_tenant' key
3. Navigate to Related Objects tab
4. Should see helpful error
5. Reload page to fix
```

---

## Performance Baseline

**Normal load times:**
- Load relationships: 200-500ms
- Apply relationship: 300-800ms
- Database insert: 50-200ms

**If slower, check:**
1. Backend database query performance
2. Network latency
3. Browser performance (DevTools → Performance)

---

## Next Steps After Testing

If everything works:
1. ✅ Commit changes to branch
2. ✅ Deploy to dev environment
3. ✅ Run integration tests
4. ✅ Get QA approval
5. ✅ Merge to main

If issues found:
1. Check backend logs: `journalctl -u semlayer-api`
2. Check database logs: `tail -f /var/log/postgresql/postgresql.log`
3. Run queries above to debug data issues
4. Refer to `ADD_RELATIONSHIP_FIX.md` for detailed troubleshooting

---

## Support

For detailed documentation, see:
- `ADD_RELATIONSHIP_FIX.md` - Complete technical details
- `RELATED_OBJECTS_TROUBLESHOOTING.md` - Diagnostic checklist
- `RELATED_OBJECTS_IMPLEMENTATION_GUIDE.md` - Architecture details

