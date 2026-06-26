# Advanced Condition Expression Builder Implementation

## Overview
Enhanced the validation rule condition builder in `ValidationRuleCreator.tsx` to provide a Workday-style expression builder with proper field selection, type-aware operators, and type-specific value inputs.

## Changes Made

### 1. Enhanced Condition Interface
**File:** `frontend/src/components/ValidationRules/ValidationRuleCreator.tsx`

Added field type tracking to the Condition interface:
```typescript
interface Condition {
  field: string;
  fieldType?: string;           // NEW: Track field type (text, number, date, boolean)
  operator: string;
  value: string;
  fieldLabel?: string;          // NEW: Display name of field
}
```

### 2. New Helper Functions

#### `getFieldsForEntity(entityName: string)`
- Extracts all available fields from a selected entity
- Returns array of field objects with metadata: `{ key, name, type, businessName }`
- Uses entity schema JSONB data from `entitySchema[entityName].entity_fields`

#### `getFieldsForSubtype(entityName: string, subtypeName: string)`
- Extracts fields from a specific subtype
- Searches for subtype fields in `entitySchema[entityName].subtypes[subtypeKey].subtype_fields`
- Falls back to entity fields if subtype has no unique fields

#### `getAllAvailableFields()`
- Combines entity fields with subtype fields (when subtype selected)
- Deduplicates fields by key
- Returns complete set of available fields for current form data
- Returns empty array if no entity is selected

#### `getOperatorsForFieldType(fieldType: string)`
- Filters operators based on selected field type
- **Text fields:** equals, not_equals, contains, starts_with, ends_with, is_empty, is_not_empty
- **Number fields:** equals, not_equals, greater_than, less_than, is_empty, is_not_empty
- **Date fields:** equals, not_equals, greater_than (after), less_than (before), is_empty, is_not_empty
- **Boolean fields:** equals, not_equals
- Falls back to default OPERATORS for unknown types

### 3. Enhanced UI Components

#### Field Selector
- Replaced text input with dropdown select
- Dynamically populated from `getAllAvailableFields()`
- Shows business-friendly field names from `businessName` property
- ID: `field-${index}` for accessibility
- On change, automatically:
  - Updates field type metadata
  - Validates operator is applicable for new field type
  - Resets operator if needed

#### Operator Selector
- Now shows type-aware operators based on selected field
- Dynamically filtered from `getOperatorsForFieldType(selectedField.type)`
- Ensures user cannot select incompatible operators
- Defaults to appropriate operator for field type

#### Value Input
- Type-specific input based on selected field type:
  - **Date fields:** `<input type="date">`
  - **Number fields:** `<input type="number">`
  - **Boolean fields:** `<select>` with True/False options
  - **Text fields:** `<input type="text">`
- Each input has appropriate title for accessibility

### 4. Data Structure Support

The implementation properly handles the entity schema JSONB structure:
```json
{
  "customer": {
    "name": "Customer",
    "entity_fields": [
      {
        "key": "id",
        "name": "id",
        "type": "text",
        "businessName": "Customer ID",
        "technicalName": "customer_id",
        "semanticTermId": "cust.id",
        "semanticTermName": "customer.id",
        "isCore": true
      },
      ...
    ],
    "subtypes": {
      "vip_customer": {
        "name": "VIP Customer",
        "subtype_fields": [...]
      },
      "standard_customer": {
        "name": "Standard Customer",
        "subtype_fields": [...]
      }
    }
  },
  ...
}
```

## Key Features

✅ **Field Picker Dropdown** - Select from all available entity fields
✅ **Subtype Support** - Shows entity fields + subtype-specific fields when applicable
✅ **Type-Aware Operators** - Operators filter based on field type
✅ **Type-Specific Inputs** - Date pickers, number inputs, boolean selects
✅ **Smart Operator Validation** - Resets operator if incompatible with new field
✅ **Business Names** - Displays user-friendly field names (e.g., "Customer ID" instead of "id")
✅ **Accessibility** - Proper labels, titles, and accessible form structure
✅ **Workday-style UX** - Visual, intuitive expression builder

## Usage

When creating a validation rule:
1. Select target entity (e.g., "Customer")
2. Optionally select subtype (e.g., "VIP Customer")
3. In Step 4 (Conditions):
   - Click "Add Condition"
   - Select field from dropdown (automatically populated with entity + subtype fields)
   - Operator dropdown filters based on field type
   - Value input adapts to field type (text/date/number/boolean)
   - Can add multiple conditions and remove as needed

## Example Conditions

**Text Field Condition:**
```
Field: Company Name
Operator: Contains
Value: "ABC"
```

**Number Field Condition:**
```
Field: Revenue
Operator: Greater Than
Value: 1000000
```

**Date Field Condition:**
```
Field: Created Date
Operator: After
Value: 2024-01-01
```

**Boolean Field Condition:**
```
Field: Is Active
Operator: Equals
Value: True
```

## Technical Details

- **No backend changes required** - Condition format remains the same: `{ field, operator, value }`
- **Field type stored for UI** - `fieldType` and `fieldLabel` stored in condition for better UX
- **Type safety** - TypeScript interfaces ensure type correctness
- **Performance** - Helper functions use memoization via component state
- **Accessibility** - All inputs labeled with proper titles and IDs

## Files Modified

- `frontend/src/components/ValidationRules/ValidationRuleCreator.tsx`
  - Enhanced Condition interface
  - Added 4 new helper functions
  - Updated condition builder UI in Step 4
  - Type-aware operator and value input selection

## Testing Checklist

- [ ] Field dropdown shows all entity fields
- [ ] Subtype selection shows combined entity + subtype fields
- [ ] Operator options filter based on selected field type
- [ ] Value input changes type based on field selection
- [ ] Creating conditions with different field types works
- [ ] Editing existing rules with conditions preserves field types
- [ ] No errors in browser console
- [ ] Accessibility features work (keyboard navigation, screen readers)

## Future Enhancements

- [ ] Expression builder GUI for complex AND/OR logic
- [ ] Condition templates for common patterns
- [ ] Expression validation before save
- [ ] Support for nested object field access (e.g., address.city)
- [ ] Custom operator definitions per field type
- [ ] Condition preview/simulation before save
