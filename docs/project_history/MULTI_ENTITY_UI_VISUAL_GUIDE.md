# Multi-Entity Validation System: UI Visual Guide

## Form Layout Overview

### Original Layout (Phase 1-2)
```
┌─────────────────────────────────────────┐
│ Create/Edit Validation Rule             │
├─────────────────────────────────────────┤
│ Rule Name: [________________]            │
│ Rule Type: [Dropdown] ▼                  │
│ Target Entity: [________________]        │
│ Description: [________________]          │
│                                          │
│ [Fields based on rule type]              │
│                                          │
│ [Save] [Cancel]                         │
└─────────────────────────────────────────┘
```

### Updated Layout (Phase 3 - NEW Features)
```
┌─────────────────────────────────────────────────────────────┐
│ Create/Edit Validation Rule                                 │
├─────────────────────────────────────────────────────────────┤
│ ★ Builder   ★ JSON Editor                                   │
├─────────────────────────────────────────────────────────────┤
│ Rule Name: [________________]                                │
│ Rule Type: [Dropdown] ▼                                      │
│ Target Entity: [________________]    ← Legacy field          │
│                                                              │
│ ★ NEW ★ Apply to Entities (Optional)                        │
│ [Search entities...]                     ← Multi-select      │
│ [Customer] [Employee] [Supplier] [+]     ← Selected chips    │
│                                                              │
│ Description: [________________]                              │
│                                                              │
│ ─────────────────────────────────────────────              │
│ [Type-Specific Fields]                                       │
│                                                              │
│ When Rule Type = "Field Format":                            │
│   Field Name: [________________]                             │
│   Regex Pattern: [________________]                          │
│                                                              │
│ When Rule Type = "Referential Integrity":  ← NEW            │
│   ℹ️ Foreign Key Validation: Verify values match...         │
│                                                              │
│   Source Entity: [Dropdown] ▼              ← NEW            │
│   Source Field: [Autocomplete ▼]           ← NEW            │
│                                                              │
│   Target Entity: [Dropdown] ▼              ← NEW            │
│   Target Field: [Autocomplete ▼]           ← NEW            │
│                                                              │
│ ─────────────────────────────────────────────              │
│ Severity: [Dropdown] ▼                                       │
│ [✓] Is Active                                                │
│                                                              │
│ [Save Validation Rule] [Cancel]                             │
└─────────────────────────────────────────────────────────────┘
```

## Feature Comparison

### Target Entity Selection

#### Before (Single Entity)
```
Target Entity: [Customer            ]
```
- ✅ Single text input
- ✅ Can type any entity name
- ❌ No search/filter
- ❌ Can't apply to multiple entities
- ❌ Duplication needed for multiple targets

#### After (Multi-Entity)
```
Target Entity: [Customer            ]

Apply to Entities (Optional):
[Search & select multiple entities...]
┌─────────────────────────────────┐
│ ☐ Customer     (selected) ✕     │ ← Shows selection
│ ☐ Employee     (selected) ✕     │   with chip view
│ ☐ Supplier     (selected) ✕     │
│ ☐ Product                        │
│ ☐ Order                          │
│ ☐ OrderDetail                    │
│ ☐ Department                     │
│ ☐ global                         │
└─────────────────────────────────┘
```
- ✅ Multi-select support
- ✅ Searchable dropdown
- ✅ Visual chip display
- ✅ One rule for all entities
- ✅ Backward compatible

### Foreign Key (Referential Integrity) Picker

#### Before (Text Input)
```
Source Entity: [_________________]
Source Field:  [_________________]
Target Entity: [_________________]
Target Field:  [_________________]
```
- ❌ No validation
- ❌ Hard to find field names
- ❌ No suggestions
- ❌ Easy to make mistakes

#### After (Smart Pickers)
```
ℹ️ Foreign Key Validation
   Verify that values in the source field match values in the 
   target field of the target entity. Example: Order.customer_id
   must match a Customer.id

┌─────────────────────────────┐ ┌─────────────────────────┐
│ Source Entity     ▼          │ │ Source Field      ▼     │
│ ┌──────────────────────────┐ │ │ ┌────────────────────┐ │
│ │ ☐ Customer               │ │ │ │ id                 │ │
│ │ ● Order           ←✓     │ │ │ │ order_id           │ │
│ │ ☐ OrderDetail           │ │ │ │ ✓ customer_id ←✓  │ │
│ │ ☐ Product               │ │ │ │ employee_id        │ │
│ │ ☐ Department            │ │ │ │ product_id         │ │
│ └──────────────────────────┘ │ │ └────────────────────┘ │
└─────────────────────────────┘ └─────────────────────────┘

┌─────────────────────────────┐ ┌─────────────────────────┐
│ Target Entity     ▼          │ │ Target Field      ▼     │
│ ┌──────────────────────────┐ │ │ ┌────────────────────┐ │
│ │ ☐ Order                  │ │ │ │ id           ←✓   │ │
│ │ ✓ Customer        ←✓     │ │ │ │ email              │ │
│ │ ☐ Employee               │ │ │ │ phone              │ │
│ │ ☐ Product               │ │ │ │ created_at         │ │
│ │ ☐ Department            │ │ │ └────────────────────┘ │
│ └──────────────────────────┘ │ └─────────────────────────┘
```
- ✅ Dropdown for entity selection (no typos)
- ✅ Autocomplete for field names (with suggestions)
- ✅ Searchable lists
- ✅ Free-form input allowed (for custom fields)
- ✅ Info alert explaining FK concept

## Component Details

### Multi-Select Autocomplete Component

```tsx
<Autocomplete
  multiple                              // Enable multiple selection
  options={[                            // Available options
    'Customer',
    'Employee',
    'Supplier',
    'Product',
    'Order',
    'OrderDetail',
    'Department',
    'global'
  ]}
  value={formData.target_entities}      // Currently selected
  onChange={(event, newValue) =>        // Handler
    handleFormChange('target_entities', newValue)
  }
  renderInput={(params) => (            // Input field
    <TextField
      {...params}
      label="Apply to Entities (Optional)"
      placeholder="Search & select entities..."
      helperText="Select multiple or leave empty for single entity"
    />
  )}
  filterOptions={(options, state) => {  // Custom filter
    return options.filter((option) =>
      option.toLowerCase().includes(state.inputValue.toLowerCase())
    );
  }}
  size="small"
/>
```

**Behavior:**
1. User clicks field → dropdown appears
2. User types "emp" → filtered to "Employee"
3. User clicks "Employee" → adds to selection as chip
4. User can add "Customer" → becomes ["Employee", "Customer"]
5. User clicks ✕ on chip → removes from selection
6. Form saves with array to backend

### FK Source Entity Dropdown

```tsx
<FormControl fullWidth>
  <InputLabel>Source Entity *</InputLabel>
  <Select
    value={formData.ref_source_entity}
    label="Source Entity *"
    onChange={(e) => {
      handleFormChange('ref_source_entity', e.target.value);
      setValidationErrors({...});
    }}
  >
    <MenuItem value="">-- Select Entity --</MenuItem>
    {['Customer', 'Employee', 'Supplier', 'Order', 'OrderDetail', 'Product', 'Department']
      .map((entity) => (
        <MenuItem key={entity} value={entity}>
          {entity}
        </MenuItem>
      ))}
  </Select>
  {validationErrors.ref_source_entity && (
    <Typography variant="caption" color="error">
      {validationErrors.ref_source_entity}
    </Typography>
  )}
</FormControl>
```

**Behavior:**
1. Dropdown shows all available entities
2. User selects "Order"
3. Field value changes to "Order"
4. Error message (if any) displays below

### FK Source Field Autocomplete

```tsx
<Autocomplete
  freeSolo                              // Allow custom values
  options={[                            // Suggested field names
    'id',
    'customer_id',
    'employee_id',
    'supplier_id',
    'order_id',
    'product_id',
    'department_id',
    'email',
    'phone'
  ]}
  value={formData.ref_source_field}
  onChange={(event, newValue) => {
    handleFormChange('ref_source_field', newValue || '');
    setValidationErrors({...});
  }}
  onInputChange={(event, newInputValue) => {
    handleFormChange('ref_source_field', newInputValue);
  }}
  renderInput={(params) => (
    <TextField
      {...params}
      label="Source Field *"
      placeholder="e.g., customer_id"
      error={!!validationErrors.ref_source_field}
      helperText={validationErrors.ref_source_field || 
                  'Field that contains the FK value'}
    />
  )}
/>
```

**Behavior:**
1. User clicks field → shows suggestions
2. User types "cust" → filtered to "customer_id"
3. User can select suggestion OR type custom field name
4. Both options are accepted (freeSolo mode)
5. Error/helper text shows below

## User Workflow Examples

### Scenario 1: Create Phone Validation Rule

**User Action → UI Response**

1. Click "Create New Validation Rule"
   → Form opens with empty fields

2. Enter Rule Name: "Phone Format Validation"
   → Name field updates

3. Select Rule Type: "Field Format"
   → Type-specific fields appear

4. Select Target Entity: "Customer"
   → Basic entity selection

5. **NEW:** Click "Apply to Entities" field
   → Dropdown appears with entity options

6. **NEW:** Type "emp" to search
   → Dropdown filters to "Employee"

7. **NEW:** Click "Employee" chip
   → "Employee" added to multi-select

8. **NEW:** Click search field again, type "supp"
   → Dropdown filters to "Supplier"

9. **NEW:** Click "Supplier"
   → Now shows [Customer, Employee, Supplier]

10. Enter Field: "phone_number"
    → Field updated

11. Enter Pattern: `^\+?[1-9]\d{1,14}$`
    → Pattern updated

12. Select Severity: "error"
    → Severity selected

13. Click "Save Validation Rule"
    → API request sent with:
    ```json
    {
      "rule_name": "Phone Format Validation",
      "rule_type": "field_format",
      "target_entity": "Customer",
      "target_entities": ["Customer", "Employee", "Supplier"],
      "condition_json": {
        "field": "phone_number",
        "pattern": "^\\+?[1-9]\\d{1,14}$"
      },
      "severity": "error"
    }
    ```

14. Success toast appears
    → Rule added to table

### Scenario 2: Create FK Validation Rule

**User Action → UI Response**

1. Click "Create New Validation Rule"
   → Form opens

2. Select Rule Type: "Referential Integrity"
   → Info alert appears:
   ```
   📌 Foreign Key (FK) Validation: Verify that values in the 
      source field match values in the target field of the 
      target entity.
   ```

3. Click "Source Entity" dropdown
   → Shows: [Customer, Employee, Supplier, Order, OrderDetail, Product, Department]

4. Select "Order"
   → "Order" appears in dropdown

5. Click "Source Field" autocomplete
   → Shows suggestions: [id, customer_id, employee_id, order_id, ...]

6. Type "cust"
   → Filtered to: [customer_id]

7. Click "customer_id"
   → "customer_id" selected

8. Click "Target Entity" dropdown
   → Shows all entities

9. Select "Customer"
   → "Customer" appears in dropdown

10. Click "Target Field" autocomplete
    → Shows suggestions: [id, email, phone, ...]

11. Select "id"
    → "id" selected

12. Fill other fields (Rule Name, Severity, etc.)
    → Fields updated

13. Click "Save Validation Rule"
    → Request sent with FK details

14. Rule saved
    → Appears in table with FK configuration

## State Management

### Form Data Structure
```javascript
{
  // Basic fields
  rule_name: "Phone Validation",
  rule_type: "field_format",
  description: "Validate phone numbers",
  target_entity: "Customer",
  
  // NEW: Multi-entity support
  target_entities: ["Customer", "Employee", "Supplier"],
  
  // Severity & status
  severity: "error",
  is_active: true,
  
  // Type-specific fields (format)
  format_pattern: "^\\+?[1-9]\\d{1,14}$",
  format_field: "phone_number",
  
  // Type-specific fields (cardinality)
  cardinality_field: "",
  cardinality_operator: ">",
  cardinality_value: "",
  
  // Type-specific fields (uniqueness)
  unique_field: "",
  
  // Type-specific fields (FK/referential_integrity)
  ref_source_entity: "Order",
  ref_source_field: "customer_id",
  ref_target_entity: "Customer",
  ref_target_field: "id",
  
  // Type-specific fields (business_logic)
  logic_condition: "{...}"
}
```

## Validation Rules Table

### Before (Single Entity)
```
┌─────────────────────────────────────────────────┐
│ Rule Name        │ Type     │ Entity   │ Status  │
├─────────────────────────────────────────────────┤
│ Phone Format     │ Format   │ Customer │ Active  │
│ Phone Format     │ Format   │ Employee │ Active  │ ← Duplicate!
│ Phone Format     │ Format   │ Supplier │ Active  │ ← Duplicate!
│ Email Format     │ Format   │ Customer │ Active  │
├─────────────────────────────────────────────────┤
```

### After (Multi-Entity)
```
┌───────────────────────────────────────────────────────────────┐
│ Rule Name        │ Type     │ Entities              │ Status  │
├───────────────────────────────────────────────────────────────┤
│ Phone Format     │ Format   │ 3 entities ●●●        │ Active  │
│ Email Format     │ Format   │ Customer              │ Active  │
│ FK Validation    │ FK       │ Order → Customer      │ Active  │
├───────────────────────────────────────────────────────────────┤
```

Hover over entity badges shows: "Customer, Employee, Supplier"

## Responsive Design

### Desktop (Full Width)
```
┌──────────────────────────────────────────────────────┐
│ Rule Name: [_________________] Rule Type: [▼]       │
├──────────────────────────────────────────────────────┤
│ Target Entity: [_____________]                       │
│ Apply to Entities: [search...] [entity1] [entity2]  │
├──────────────────────────────────────────────────────┤
│ Source Entity: [▼]  │  Source Field: [autocomplete] │
│ Target Entity: [▼]  │  Target Field: [autocomplete] │
└──────────────────────────────────────────────────────┘
```

### Tablet (Medium Width)
```
┌─────────────────────────────────────┐
│ Rule Name: [________________]        │
│ Rule Type: [▼]                      │
├─────────────────────────────────────┤
│ Target Entity: [____________]       │
│ Apply to Entities: [search...]      │
│ [entity1] [entity2] [entity3]      │
├─────────────────────────────────────┤
│ Source Entity: [▼]                  │
│ Source Field: [autocomplete]        │
│ Target Entity: [▼]                  │
│ Target Field: [autocomplete]        │
└─────────────────────────────────────┘
```

### Mobile (Small Width)
```
┌────────────────────────┐
│ Rule Name              │
│ [________________]     │
│ Rule Type              │
│ [▼]                    │
├────────────────────────┤
│ Target Entity          │
│ [________________]     │
│ Apply to Entities      │
│ [search...]            │
│ [entity1]              │
│ [entity2]              │
├────────────────────────┤
│ Source Entity: [▼]     │
│ Source Field: [...   ▼]
│ Target Entity: [▼]     │
│ Target Field: [... ▼] │
└────────────────────────┘
```

## Error States

### Validation Error Display
```
❌ Rule name is required
   [_________________]
   ^ Red underline + error message

❌ Field name is required
   [_________________]
   ^ Red underline + error message below
```

### FK Picker Error
```
Source Entity *
┌──────────────┐ ← Red border
│ ┌──────────┐ │
│ │ [Select] ▼│
│ └──────────┘ │
└──────────────┘
❌ Source entity is required
```

## Success States

### Successful Save
```
✅ Toast notification (top-right)
   "Validation rule saved successfully"
   
   ┌────────────────────────────────┐
   │ ✅ Validation rule saved       │ ← Auto-dismiss in 3s
   └────────────────────────────────┘
```

### Rule Added to Table
```
┌─────────────────────────────────────────────────┐
│ New Rule              │ Format   │ 3 entities    │
│ ✨ (highlight/pulse)  │          │ Active        │
└─────────────────────────────────────────────────┘
```

## Next Steps

After reviewing this visual guide:
1. Test the UI in your browser at `http://localhost:5173`
2. Create a multi-entity rule using the workflow shown
3. Verify the API request includes the new fields
4. Run database migration when ready
5. Implement backend engine changes
6. Run full integration tests

This completes the UI implementation for multi-entity validation system!
