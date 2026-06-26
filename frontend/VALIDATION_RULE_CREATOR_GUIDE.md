# ValidationRuleCreator - Advanced Condition Builder Guide

## Problem: Advanced Conditions Not Showing When Selecting Entity

If you select an entity in Step 2 but don't see advanced expressions (Looker patterns, relative dates) when adding conditions, the issue is that **field metadata is not being loaded for that entity**.

## Solution: Automatic Field Metadata Loading

The updated `ValidationRuleCreator` now supports **dynamic field metadata loading**. When you select an entity, it can automatically fetch the field information from your semantic objects/terms.

### How It Works Now

1. **When you select an entity in Step 2**
2. **Component updates `formData.target_entity`**
3. **useEffect triggers and can load field metadata for that entity**
4. **Field dropdown populates with semantic fields and their types**
5. **Advanced Condition Builder activates for type-aware expressions**

### Implementation: Connect to Your Semantic Objects API

Update the metadata loading effect in `ValidationRuleCreator.tsx`:

```typescript
// Load field metadata when target entity changes
useEffect(() => {
  // If initial fieldMetadata provided, use it
  if (Object.keys(fieldMetadata).length > 0) {
    setDynamicFieldMetadata(fieldMetadata);
    return;
  }

  // Fetch from your semantic objects API
  if (formData.target_entity) {
    _setIsLoadingMetadata(true);
    
    // Call your semantic objects endpoint
    fetchSemanticObjectFields(formData.target_entity)
      .then(fields => {
        // Transform to FieldTypeInfo format
        const metadata: Record<string, FieldTypeInfo> = {};
        fields.forEach(field => {
          metadata[field.name] = {
            type: mapDataTypeToFieldType(field.data_type),
            enumValues: field.enum_values,
            isNullable: !field.required
          };
        });
        setDynamicFieldMetadata(metadata);
      })
      .catch(err => console.error('Failed to load fields:', err))
      .finally(() => _setIsLoadingMetadata(false));
  }
}, [formData.target_entity, fieldMetadata]);
```

### Example: Fetching from Semantic Objects

```typescript
// In your API utils
async function fetchSemanticObjectFields(entityName: string) {
  const response = await fetch(
    `/api/semantic-objects/${entityName}/fields?tenant_id=${tenantId}&datasource_id=${datasourceId}`
  );
  if (!response.ok) throw new Error('Failed to fetch fields');
  return response.json();
}

// Map backend data types to frontend FieldTypeInfo types
function mapDataTypeToFieldType(backendType: string): FieldTypeInfo['type'] {
  const typeMap: Record<string, FieldTypeInfo['type']> = {
    'string': 'string',
    'varchar': 'string',
    'text': 'string',
    'integer': 'number',
    'bigint': 'number',
    'decimal': 'number',
    'float': 'number',
    'double': 'number',
    'date': 'date',
    'timestamp': 'date',
    'datetime': 'date',
    'boolean': 'boolean',
    'bool': 'boolean'
  };
  return typeMap[backendType.toLowerCase()] || 'unknown';
}
```

### Complete Example with Semantic Objects Integration

```typescript
import React, { useState, useEffect } from 'react';
import { ValidationRuleCreator, FieldTypeInfo } from './components/ValidationRuleCreator';
import type { ValidationRule } from './validation/types';

export function RuleBuilderWithSemanticObjects() {
  const [isOpen, setIsOpen] = useState(false);
  const tenantId = useContext(TenantContext)?.selected_tenant?.id;
  const datasourceId = useContext(TenantContext)?.selected_datasource?.id;
  const [availableEntities, setAvailableEntities] = useState<string[]>([]);

  // Load available entities on mount
  useEffect(() => {
    async function loadEntities() {
      try {
        const response = await fetch(
          `/api/semantic-objects?tenant_id=${tenantId}&datasource_id=${datasourceId}`
        );
        const data = await response.json();
        setAvailableEntities(data.map((obj: any) => obj.name));
      } catch (err) {
        console.error('Failed to load entities:', err);
      }
    }
    
    if (tenantId && datasourceId) {
      loadEntities();
    }
  }, [tenantId, datasourceId]);

  const handleSave = async (rule: ValidationRule) => {
    try {
      const response = await fetch('/api/validation-rules', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId
        },
        body: JSON.stringify(rule)
      });
      
      if (response.ok) {
        console.log('Rule saved successfully');
        setIsOpen(false);
      }
    } catch (err) {
      console.error('Failed to save rule:', err);
    }
  };

  return (
    <>
      <button onClick={() => setIsOpen(true)} className="px-4 py-2 bg-blue-600 text-white rounded">
        Create Validation Rule
      </button>

      <ValidationRuleCreator
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        onSave={handleSave}
        availableEntities={availableEntities}
        displayMode="modal"
        tenantId={tenantId}
        datasourceId={datasourceId}
        // Don't pass fieldMetadata here - it will be loaded dynamically
      />
    </>
  );
}
```

### What Happens Now (Step-by-Step)

1. **User clicks "Create Validation Rule"** → Opens modal

2. **Step 1: Basic Info** → User enters rule name and description

3. **Step 2: Configuration**
   - User selects entity (e.g., "Employee")
   - **Component automatically fetches fields** from semantic objects
   - Field list populates

4. **Step 3: Severity** → User selects error/warning/info

5. **Step 4: Conditions**
   - User clicks "Add Condition"
   - **Field dropdown shows all fields** with their types
   - User selects a field (e.g., "salary" - number type)
   - **AdvancedConditionBuilder activates** with:
     - Type-aware operators for numbers: `>=`, `<=`, `[50,100]`, `AND/OR`
     - Real-time validation
     - Expression examples

### Behavior Changes

**Before (Static fieldMetadata):**
- ❌ Field dropdown shows nothing or placeholder text
- ❌ Advanced Condition Builder can't activate properly
- ❌ User sees only basic operators

**After (Dynamic from Semantic Objects):**
- ✅ Field dropdown shows all fields with their types
- ✅ Selecting a field triggers Advanced Condition Builder
- ✅ Type-aware operators appear automatically
- ✅ Real-time validation and examples work
- ✅ Looker expressions available for all types

## Error Handling

The component gracefully handles scenarios where fields can't be loaded:

```typescript
{!fieldsAvailable && (
  <div className="px-3 py-2 bg-yellow-50 border border-yellow-200 rounded text-xs text-yellow-700">
    No field metadata available. Please configure field types for your entity.
  </div>
)}
```

## Common Issues

### Issue: Still seeing "No field metadata" message

**Check:**
1. ✅ Is tenantId and datasourceId being passed to the API?
2. ✅ Does the semantic-objects endpoint return field information?
3. ✅ Are field names matching the semantic object attribute names?

**Debug:**
```javascript
// In browser console
fetch('/api/semantic-objects/Employee/fields?tenant_id=XXX&datasource_id=YYY')
  .then(r => r.json())
  .then(console.log)
```

### Issue: Fields load but no Advanced options appear

**Check:**
1. ✅ Field data type is mapped correctly (string, number, date, etc.)
2. ✅ FieldTypeInfo type matches supported values
3. ✅ Operators exist for that type in AdvancedConditionBuilder

**Verify Types:**
```typescript
// Valid types
type: 'string' | 'number' | 'boolean' | 'date' | 'enum' | 'unknown'
```

## Field Type Mapping Reference

| Backend Type | Frontend Type | Advanced Options |
|---|---|---|
| VARCHAR, TEXT, STRING | string | %, -%,  EMPTY, NULL, expressions |
| INT, BIGINT, DECIMAL, FLOAT | number | [50,100], >=50, AND/OR, expressions |
| DATE, TIMESTAMP, DATETIME | date | today, last 7 days, 2024-01-15, expressions |
| BOOLEAN, BOOL | boolean | is_true, is_false |
| Other | unknown | Basic: equals, not_equals only |

## Next Steps

1. ✅ Update your semantic-objects API endpoint to return field metadata
2. ✅ Implement the field-fetching effect in ValidationRuleCreator
3. ✅ Test by selecting an entity in Step 2
4. ✅ Add a condition in Step 4
5. ✅ Verify Advanced Condition Builder shows up with proper operators

---

**The Advanced Condition Builder is now fully integrated with your semantic objects!** When you select an entity, its fields automatically populate with proper type information, enabling sophisticated expression-based validation rules.


## Field Types Supported

### String
- **Operators:** Equals, Not Equals, Contains, Starts With, Ends With, Is Empty, Is Not Empty, Advanced Expressions
- **Advanced Patterns:** `%pattern%`, `pattern%`, `%pattern`, `-pattern`, `EMPTY`, `NULL`

### Number
- **Operators:** Equals, Not Equals, Greater Than, Less Than, Is Empty, Is Not Empty, Advanced Expressions
- **Advanced Patterns:** `[50,100]`, `(50,100)`, `>=50`, `NOT [50,100]`, `>=50 AND <=100`

### Date
- **Operators:** Equals, Before, After, Is Empty, Is Not Empty, Relative Dates, Advanced Expressions
- **Relative Date Examples:** `today`, `yesterday`, `last 7 days`, `this month`, `Monday`
- **Absolute Dates:** `2024-01-15`

### Boolean
- **Operators:** Equals, Is True, Is False

### Enum
- **Operators:** Equals, Not Equals, In List, Advanced Expressions

### Unknown/Not Provided
- Shows basic operators only (Equals, Not Equals, Is Empty, Is Not Empty)

## Complete Example

```typescript
import React, { useState } from 'react';
import { ValidationRuleCreator, FieldTypeInfo } from './components/ValidationRuleCreator';
import type { ValidationRule } from './validation/types';

export function RuleManager() {
  const [isOpen, setIsOpen] = useState(false);

  // Define field metadata for your entity
  const fieldMetadata: Record<string, FieldTypeInfo> = {
    email: { type: 'string', isNullable: true },
    salary: { type: 'number', isNullable: false },
    hire_date: { type: 'date', isNullable: true },
    is_active: { type: 'boolean', isNullable: false },
    department: {
      type: 'enum',
      enumValues: ['Sales', 'Engineering', 'HR'],
      isNullable: true
    }
  };

  const handleSave = (rule: ValidationRule) => {
    console.log('Rule saved:', rule);
    // Call your backend API
    // await api.saveRule(rule);
  };

  return (
    <>
      <button onClick={() => setIsOpen(true)}>
        Create Rule
      </button>

      <ValidationRuleCreator
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        onSave={handleSave}
        availableEntities={['Employee', 'Department']}
        fieldMetadata={fieldMetadata}  // ← Required for advanced conditions!
        displayMode="modal"
      />
    </>
  );
}
```

## Workflow: Adding a Condition

1. **Click "Add Condition"** in Step 4
2. **Select a Field** from the dropdown (now shows field type)
3. **Select an Operator** (now shows type-aware operators)
4. **Enter a Value** 
   - **String:** Use patterns like `%admin%`, `-test`, `EMPTY`
   - **Number:** Use ranges like `[50,100]`, comparisons, or lists
   - **Date:** Use `today`, `last 7 days`, or `2024-01-15`
5. **See Real-Time Validation** - Helpers show valid syntax

## Advanced Expression Examples

### String Validation
- Email domain validation: `%@company.com`
- Exclude test data: `-%test%`
- Check if empty: `EMPTY`

### Numeric Validation
- Salary range: `[50000,150000]`
- Age restrictions: `>=18 AND <=65`
- Not in range: `NOT [0,100]`

### Date Validation
- Recent hires: `last 30 days`
- Specific date: `2024-01-15`
- Day of week: `Monday`
- Date range: `after 2024-01-01 AND before 2024-12-31`

## Troubleshooting

### Issue: "No field metadata available" message

**Cause:** You didn't pass `fieldMetadata` prop or it's empty

**Fix:**
```typescript
// ❌ Wrong
<ValidationRuleCreator
  availableEntities={entities}
/>

// ✅ Correct
<ValidationRuleCreator
  availableEntities={entities}
  fieldMetadata={fieldMetadata}  // Must provide this
/>
```

### Issue: Only basic operators show (no "Advanced Expressions")

**Cause:** Field is selected as `unknown` type

**Fix:** Ensure your field is defined in metadata with a specific type:
```typescript
// ❌ Wrong
const fieldMetadata = {
  email: { type: 'unknown' }  // Too generic
};

// ✅ Correct
const fieldMetadata = {
  email: { type: 'string' }  // Specific type
};
```

### Issue: Can't see Looker patterns in value field

**Cause:** Operator is not set to `expressions`

**Fix:** Select "Advanced Expressions" from the operator dropdown

## Getting Field Metadata from Your Entity Schema

If you're fetching field metadata from your backend:

```typescript
// From your API
async function loadFieldMetadata(entity: string) {
  const response = await fetch(`/api/entities/${entity}/fields`);
  const fields = await response.json();

  // Transform to FieldTypeInfo format
  const metadata: Record<string, FieldTypeInfo> = {};
  fields.forEach(field => {
    metadata[field.name] = {
      type: mapBackendTypeToFrontendType(field.data_type),
      enumValues: field.enum_values,
      isNullable: !field.required
    };
  });

  return metadata;
}

// Usage
const metadata = await loadFieldMetadata('Employee');
setFieldMetadata(metadata);
```

## Next Steps

1. ✅ Add `fieldMetadata` to your ValidationRuleCreator instance
2. ✅ Open the component and go to Step 4
3. ✅ Click "Add Condition"
4. ✅ Select a field from the dropdown
5. ✅ You should now see advanced operators and expression syntax helpers!

---

**The Advanced Condition Builder is now active and ready for Looker expressions!**
