# Multi-Entity Validation System: Testing Guide

## Quick Start Test (5 minutes)

### Verify UI Components Loaded

1. **Open Browser Console** (F12)
2. **Navigate** to http://localhost:5173/catalog/validation-rules
3. **Verify** these elements exist:
   - "Apply to Entities (Optional)" field with dropdown
   - "Source Entity" dropdown (with Foreign Key rules)
   - "Source Field" autocomplete with suggestions
   - "Target Entity" dropdown
   - "Target Field" autocomplete

### Test Multi-Select Entity Picker

```javascript
// In browser console, verify the component mounts:
document.querySelector('[aria-label*="Apply to Entities"]')  // Should return element
```

**Manual Steps:**
1. Click "Create New Validation Rule"
2. Fill out form:
   - Rule Name: "Test Phone Validation"
   - Rule Type: "Field Format"
   - Target Entity: "Customer"
   - **NEW:** Click "Apply to Entities" field
3. **Verify dropdown appears** with options: Customer, Employee, Supplier, Product, Order, OrderDetail, Department, global
4. **Select multiple** entities (Customer, Employee, Supplier)
5. **Verify** selected chips appear below the field
6. **Click outside** to close dropdown

## Complete Test Suite

### Test 1: Create Single-Entity Rule (Backward Compatibility)

**Objective:** Ensure existing workflow still works

**Steps:**
1. Navigate to Validation Rules page
2. Click "Create New Validation Rule"
3. Fill form:
   - Rule Name: "Email Format - Backward Compat"
   - Rule Type: "Field Format"
   - Target Entity: "Customer"
   - **Leave "Apply to Entities" empty**
   - Field: "email"
   - Pattern: `^[^\s@]+@[^\s@]+\.[^\s@]+$`
   - Severity: "warning"
4. Click "Save Validation Rule"
5. **Verify** rule appears in table with status "Active"
6. In console, check API request:
   ```
   POST /api/validation-rules?tenant_id=...&datasource_id=...
   Body includes: "target_entities": []  ← Empty array
   ```

### Test 2: Create Multi-Entity Rule

**Objective:** Apply one rule to multiple entities

**Steps:**
1. Click "Create New Validation Rule"
2. Fill form:
   - Rule Name: "Phone Validation - Multi-Entity"
   - Rule Type: "Field Format"
   - Target Entity: "Customer"
   - **Apply to Entities:** Select [Customer, Employee, Supplier]
   - Field: "phone_number"
   - Pattern: `^\+?[1-9]\d{1,14}$`
   - Severity: "error"
3. Click "Save Validation Rule"
4. **Verify:**
   - Rule saved successfully
   - Toast notification shows success
   - Rule appears in table
5. In browser Network tab, inspect POST request:
   ```json
   {
     "rule_name": "Phone Validation - Multi-Entity",
     "target_entities": ["Customer", "Employee", "Supplier"],
     "target_entity": "Customer",
     ...
   }
   ```

### Test 3: Global Rule Creation

**Objective:** Create rule that applies to all entities

**Steps:**
1. Click "Create New Validation Rule"
2. Fill form:
   - Rule Name: "Data Quality - Non-Null Created Date"
   - Rule Type: "Cardinality"
   - Target Entity: "Product"
   - **Apply to Entities:** Select ["global"]
   - Field: "created_at"
   - Operator: ">"
   - Value: "0"
   - Severity: "warning"
3. Click "Save Validation Rule"
4. **Verify:**
   - Request includes: `"target_entities": ["global"]`
   - Rule applies to all entities in backend

### Test 4: Edit Multi-Entity Rule

**Objective:** Modify which entities a rule applies to

**Steps:**
1. Find "Phone Validation - Multi-Entity" rule (from Test 2)
2. Click ✏️ Edit icon
3. **Verify** form loads with:
   - Target Entities: [Customer, Employee, Supplier] selected
   - All other fields populated
4. **Modify** target entities:
   - Add "Product" to selection
   - Remove "Supplier" from selection
   - Result: [Customer, Employee, Product]
5. Click "Save Validation Rule"
6. **Verify:**
   - Toast shows "Rule updated successfully"
   - Table reflects the change
   - API PATCH request includes updated array

### Test 5: FK Picker Dropdown Functionality

**Objective:** Test enhanced foreign key picker UI

**Steps:**
1. Click "Create New Validation Rule"
2. Select Rule Type: "Referential Integrity"
3. **Verify UI changes:**
   - Info alert appears: "📌 Foreign Key (FK) Validation..."
   - Source Entity: Shows dropdown with entities
   - Source Field: Shows autocomplete with suggestions
   - Target Entity: Shows dropdown with entities
   - Target Field: Shows autocomplete with suggestions

4. **Test Source Entity Dropdown:**
   - Click "Source Entity" dropdown
   - **Verify** options appear: Customer, Employee, Supplier, Order, OrderDetail, Product, Department
   - Select "Order"
   - **Verify** field value changes to "Order"

5. **Test Source Field Autocomplete:**
   - Click "Source Field" field
   - **Verify** suggestions appear: id, customer_id, employee_id, supplier_id, order_id, product_id, department_id, email, phone
   - Type "cust" 
   - **Verify** filtered to "customer_id"
   - Select "customer_id"

6. **Test Target Entity Dropdown:**
   - Select "Customer"

7. **Test Target Field Autocomplete:**
   - Type "id"
   - Select "id"

8. **Fill remaining fields:**
   - Rule Name: "FK - Order to Customer"
   - Severity: "error"

9. **Click Save**
10. **Verify:** Rule saved with all FK fields populated

### Test 6: Search and Filter in Entity Picker

**Objective:** Test autocomplete filtering

**Steps:**
1. Click "Create New Validation Rule"
2. Click "Apply to Entities" field
3. **Type "cust"** in the input
4. **Verify:** Dropdown filtered to only show "Customer"
5. **Type "emp"**
6. **Verify:** Dropdown filtered to show "Employee"
7. **Clear** the search
8. **Verify:** All options appear again
9. **Type "order"**
10. **Verify:** Shows "Order" and "OrderDetail"

### Test 7: Data Persistence

**Objective:** Verify rules persist after page refresh

**Steps:**
1. Create multi-entity rule: "Persistence Test Rule" with [Customer, Employee]
2. **Close the form dialog** 
3. **Verify** rule appears in table
4. **Refresh page** (F5)
5. **Verify:**
   - Rule still appears in table
   - Multi-entity indicators visible
   - Data loaded from backend

### Test 8: Tenant Scoping

**Objective:** Ensure multi-entity rules respect tenant boundaries

**Steps:**
1. **Open browser console**
2. **Note current tenant:**
   ```javascript
   JSON.parse(localStorage.getItem('selected_tenant'))
   ```
3. **Create multi-entity rule** for current tenant
4. **Switch to different tenant** (if available) using Fabric Builder selector
5. **Navigate** to Validation Rules page
6. **Verify:**
   - Rules from previous tenant NOT visible
   - Different set of rules loads
7. **Switch back** to original tenant
8. **Verify:**
   - Original multi-entity rule appears again

### Test 9: API Response Format

**Objective:** Verify backend returns multi-entity field

**Steps:**
1. Open Network tab (F12)
2. Fetch validation rules with curl:
   ```bash
   curl -X GET "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&datasource_id=982aef38-418f-46dc-acd0-35fe8f3b97b0" \
     -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
     -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0"
   ```
3. **Verify response includes:**
   ```json
   [
     {
       "id": "rule-123",
       "rule_name": "Phone Validation - Multi-Entity",
       "target_entity": "Customer",
       "target_entities": ["Customer", "Employee", "Supplier"],
       ...
     }
   ]
   ```

## Component-Level Tests

### Test Autocomplete Component

```javascript
// In browser console, simulate selecting multiple values
const event = new Event('change', { bubbles: true });
const input = document.querySelector('input[aria-label*="Apply to Entities"]');
input.value = 'Customer';
input.dispatchEvent(event);
```

### Test Form Validation

**Steps:**
1. Click "Create New Validation Rule"
2. **Try to save** WITHOUT filling required fields
3. **Verify error messages appear:**
   - "Rule name is required"
   - "Target entity is required"
   - Type-specific errors (e.g., "Field name is required" for format rules)
4. **Fill only** Rule Name and Rule Type
5. **Click Save**
6. **Verify** target entity error appears
7. **Select Target Entity**
8. **Click Save**
9. **Verify** type-specific field errors appear

## Integration Tests

### Test: Multi-Entity Rule Validation

**Scenario:** Phone rule for Customer, Employee, Supplier

**Setup:**
1. Create multi-entity rule:
   ```json
   {
     "rule_name": "Phone Format Validation",
     "rule_type": "field_format",
     "target_entities": ["Customer", "Employee", "Supplier"],
     "condition_json": {
       "field": "phone_number",
       "pattern": "^\\+?[1-9]\\d{1,14}$"
     },
     "severity": "error"
   }
   ```

2. **Verify rule applies to all three entities:**
   - GET `/api/validation-rules?entity=Customer` → rule included
   - GET `/api/validation-rules?entity=Employee` → rule included
   - GET `/api/validation-rules?entity=Supplier` → rule included
   - GET `/api/validation-rules?entity=Product` → rule NOT included

### Test: Global Rule Application

**Setup:**
1. Create global rule:
   ```json
   {
     "target_entities": ["global"],
     "condition_json": { ... }
   }
   ```

2. **Verify rule applies to ALL entities:**
   ```bash
   for entity in Customer Employee Supplier Order Product; do
     curl -s "http://localhost:29080/api/validation-rules?entity=$entity&..." \
       | jq '.[] | select(.target_entities[] == "global")'
   done
   ```

## Performance Tests

### Test: Large Dataset Performance

**Setup:** Create 100+ multi-entity rules

**Steps:**
1. Write script to create rules via API
2. Measure page load time:
   ```javascript
   performance.measure('validation-rules-load', 'navigationStart', 'loadEventEnd');
   console.log(performance.getEntriesByName('validation-rules-load')[0].duration);
   ```
3. **Verify** page loads in < 2 seconds
4. **Test filtering:**
   - Search for entity
   - **Verify** filters respond in < 500ms

### Test: Index Query Performance

**Run in PostgreSQL:**
```sql
EXPLAIN ANALYZE
SELECT * FROM catalog_validation_rules
WHERE ('global' = ANY(target_entities) OR 'Customer' = ANY(target_entities))
  AND is_active = true;
```

**Verify:**
- Uses index scan (not sequential scan)
- Planning time < 1ms
- Execution time < 10ms (for < 10k rules)

## Error Handling Tests

### Test 1: Invalid Entity Name

**Steps:**
1. Create rule with custom entity name: "NonExistentEntity"
2. **Verify:**
   - Rule still saves (no validation on entity existence)
   - Warning message could be added for future improvement

### Test 2: Duplicate Entities in Selection

**Steps:**
1. Try to add "Customer" twice to target_entities
2. **Verify:**
   - Duplicate not added (Autocomplete prevents this)
   - Only single instance appears in array

### Test 3: Empty Entity Selection

**Steps:**
1. Create rule with empty target_entities array
2. **Verify:**
   - Falls back to single target_entity field
   - Rule applies only to specified entity

### Test 4: Database Migration Not Applied

**Steps:**
1. Comment out `target_entities` column in migrations
2. Try to create multi-entity rule
3. **Verify:**
   - Graceful error message (not raw SQL error)
   - Fallback to single-entity behavior

## Rollback Tests

### Test: Revert to Single-Entity Mode

**Steps:**
1. Create multi-entity rules
2. Remove `target_entities` column from database
3. Reload frontend
4. **Verify:**
   - UI still works
   - Uses target_entity field only
   - Multi-entity picker hidden or disabled

## Sign-Off Checklist

- [ ] UI components render without errors
- [ ] Multi-select autocomplete works correctly
- [ ] Single-entity rules still work (backward compatible)
- [ ] Multi-entity rules save and load correctly
- [ ] Global rules apply to all entities
- [ ] FK picker dropdowns and autocompletes work
- [ ] Search/filtering works in entity picker
- [ ] Data persists after page refresh
- [ ] Tenant scoping enforced
- [ ] API returns correct multi-entity format
- [ ] Validation errors display properly
- [ ] Performance acceptable (< 2s load time)
- [ ] Database migration script works
- [ ] No TypeScript errors
- [ ] All tests pass

## Test Report Template

```
TEST RESULTS - Multi-Entity Validation System
==============================================

Date: [DATE]
Tester: [NAME]
Environment: [DEV/STAGING/PROD]

Test 1: Create Single-Entity Rule
Status: ✅ PASS / ❌ FAIL / ⚠️ PARTIAL
Notes: 

Test 2: Create Multi-Entity Rule
Status: ✅ PASS / ❌ FAIL / ⚠️ PARTIAL
Notes:

... (continue for all tests)

Overall Result: ✅ READY FOR DEPLOYMENT / ❌ BLOCKERS FOUND

Blockers:
- 

Recommendations:
- 
```

## Quick Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| "Apply to Entities" field not visible | Autocomplete import missing | Add `Autocomplete` to MUI imports |
| Multi-entity not saved | Database column missing | Run migration: `ALTER TABLE catalog_validation_rules ADD COLUMN IF NOT EXISTS target_entities TEXT[]` |
| Dropdown shows no entities | Options array empty | Verify entity list in Autocomplete `options` prop |
| Rules not applying to multiple entities | Backend query not updated | Update backend engine to use `ANY()` operator |
| Autocomplete throwing errors | State mismatch | Verify `formData.target_entities` initialized as empty array `[]` |
| Search in entity picker not working | Filter logic missing | Check Autocomplete `filterOptions` prop |

## Next Steps After Testing

1. ✅ **UI Implementation Complete** - Multi-select and FK picker working
2. ⏳ **Database Migration** - Run ALTER TABLE command
3. ⏳ **Backend Engine Update** - Implement multi-entity query logic
4. ⏳ **Integration Tests** - Run full E2E tests
5. ⏳ **Performance Tests** - Measure with production data
6. ⏳ **User Acceptance Testing** - Validate with stakeholders
7. ⏳ **Deployment** - Roll out to production
