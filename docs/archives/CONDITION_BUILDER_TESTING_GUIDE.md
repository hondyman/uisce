# Manual Testing Guide: Advanced Condition Builder

## Prerequisites
- Frontend running at `http://localhost:5173`
- Backend running at `http://localhost:8001`
- Tenant selected in Fabric Builder (stored in localStorage)
- API Gateway forwarding requests properly

## Testing Steps

### Step 1: Navigate to Validation Rules
1. Open browser to `http://localhost:5173`
2. Ensure tenant is selected (see tenant picker in UI)
3. Navigate to Validation Rules page (from main menu)

### Step 2: Create New Validation Rule
1. Click "+ Create New Rule" or similar button
2. ValidationRuleCreator modal opens

### Step 3: Fill Basic Information (Step 1)
- Rule Name: "Test Customer Name Rule"
- Description: "Validate customer names are not empty"
- Click "Next"

### Step 4: Configure Rule (Step 2)
- Rule Type: Select "Field Format"
- Target Entity: Select "Customer"
- Click "Next"

### Step 5: Set Severity (Step 3)
- Severity: Select "Warning"
- Click "Next"

### Step 6: Add Conditions (Step 4) - Main Test Area
1. Click "+ Add Condition"
2. **Field Dropdown Test:**
   - Click "Field" dropdown
   - Verify it shows fields from Customer entity:
     - Customer ID
     - Company Name
     - Contact Name
     - Contact Title
     - Address
     - City
     - (etc.)
   - Select "Company Name"

3. **Operator Filtering Test:**
   - Notice operator dropdown now shows TEXT operators:
     - Equals
     - Not Equals
     - Contains
     - Starts With
     - Ends With
     - Is Empty
     - Is Not Empty
   - Select "Contains"

4. **Value Input Test:**
   - Value input shows as text input (not date/number/boolean)
   - Enter "ABC" as value

5. **Add Second Condition (Number Type Test):**
   - Click "+ Add Condition" again
   - Field: Select an entity that has a number field (if available)
   - Or use test entity "Order" which might have quantity/amount fields
   - Verify operator shows NUMBER operators (Greater Than, Less Than, etc.)
   - Verify value input shows number input

6. **Subtype Test (Optional):**
   - Go back to Step 2 (click Back button)
   - Select subtype "VIP Customer" for Customer entity
   - Go forward to Step 4
   - Add condition and verify fields still populate from Customer + VIP Customer

### Step 7: Verify Save
1. Click "Save" or "Create Rule"
2. Verify rule saves successfully
3. Check browser network tab for POST to `/api/validation-rules`
4. Verify conditions are sent with correct structure

## Expected Behaviors

### Field Dropdown
✓ Shows all available fields from selected entity
✓ Fields show business names (e.g., "Customer ID") not technical names
✓ Dynamically updates when entity selection changes
✓ Dynamically updates when subtype selection changes (adds subtype fields)

### Operator Dropdown
✓ Shows different operators based on field type:
  - Text fields: 7 operators (equals, contains, starts_with, etc.)
  - Number fields: 6 operators (equals, >, <, is_empty, etc.)
  - Date fields: 6 operators (equals, after, before, is_empty, etc.)
  - Boolean fields: 2 operators (equals, not_equals)
✓ Automatically resets to valid operator if field type changes
✓ Cannot select incompatible operators

### Value Input
✓ Text fields: text input
✓ Number fields: number input with spinner controls
✓ Date fields: date picker (calendar)
✓ Boolean fields: dropdown with True/False options

### Accessibility
✓ All inputs have labels
✓ All inputs have title attributes
✓ Keyboard navigation works
✓ Screen readers can identify elements

## Debugging

If issues occur:

### Field dropdown is empty
- Check if entity is selected in Step 2
- Verify entitySchema is passed correctly to component
- Check browser console for errors
- Verify entity schema API endpoint returns data

### Operators not changing
- Check getOperatorsForFieldType function is being called
- Verify selected field type is detected correctly
- Check browser console for errors in field onChange

### Value input not changing type
- Verify selectedField is found in availableFields
- Check field.type property has correct value
- Verify conditional rendering logic is correct

### Console Errors
- Check for undefined entitySchema
- Verify availableFields array is populated
- Check for proper key-value pairs in field objects

## Expected Network Calls

### Create Rule Request
```javascript
POST /api/validation-rules?tenant_id=<ID>&datasource_id=<ID>
Headers:
  - X-Tenant-ID: <ID>
  - X-Tenant-Datasource-ID: <ID>
  - Content-Type: application/json

Body:
{
  "rule_name": "Test Customer Name Rule",
  "rule_type": "field_format",
  "target_entity": "customer",
  "description": "Validate customer names are not empty",
  "severity": "warning",
  "is_active": true,
  "condition_json": {
    "conditions": [
      {
        "field": "company_name",
        "fieldType": "text",
        "operator": "contains",
        "value": "ABC",
        "fieldLabel": "Company Name"
      }
    ]
  }
}
```

## Success Criteria

- [ ] Field dropdown populates with entity fields
- [ ] Operators change based on field type
- [ ] Value input type changes based on field type
- [ ] Conditions can be added and removed
- [ ] Rule saves successfully to database
- [ ] No console errors
- [ ] Conditions persist when editing rule
- [ ] Subtype fields merge with entity fields when applicable
- [ ] Accessibility features work (labels, titles, keyboard nav)

## Next Steps After Testing

1. Test validation rule execution with conditions
2. Test editing existing rules with conditions
3. Test conditions with different entity/subtype combinations
4. Verify conditions are used in validation enforcement
5. Add expression builder for AND/OR logic
