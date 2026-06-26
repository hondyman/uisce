# ValidationRuleCreator Quick Start Guide

## 5-Minute Setup

### Step 1: Import the Component

```typescript
import { ValidationRuleCreator, type FieldTypeInfo } from './ValidationRuleCreator';
```

### Step 2: Define Field Metadata

```typescript
const fieldMetadata: Record<string, FieldTypeInfo> = {
  // String fields
  employee_id: { type: 'string', isNullable: false },
  name: { type: 'string', isNullable: false },
  email: { type: 'string', isNullable: true },
  
  // Number fields
  salary: { type: 'number', isNullable: false },
  years_experience: { type: 'number', isNullable: true },
  
  // Date fields
  hire_date: { type: 'date', isNullable: true },
  termination_date: { type: 'date', isNullable: true },
  
  // Boolean fields
  is_active: { type: 'boolean', isNullable: false },
  
  // Enum fields
  department: { 
    type: 'enum', 
    enumValues: ['HR', 'Engineering', 'Sales', 'Finance'],
    isNullable: false 
  },
};
```

### Step 3: Use in Component

```typescript
import React, { useState } from 'react';
import { ValidationRuleCreator } from './ValidationRuleCreator';
import type { ValidationRule } from './validation/types';

export const MyRuleBuilder = () => {
  const [isOpen, setIsOpen] = useState(false);
  const [rules, setRules] = useState<ValidationRule[]>([]);

  const handleSave = (rule: ValidationRule) => {
    setRules(prev => [rule, ...prev]);
    setIsOpen(false);
  };

  return (
    <>
      <button onClick={() => setIsOpen(true)}>
        Create Validation Rule
      </button>

      <ValidationRuleCreator
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        onSave={handleSave}
        availableEntities={['Employee', 'Department', 'Position']}
        fieldMetadata={fieldMetadata}  // ← Pass metadata here!
        displayMode="modal"
      />
    </>
  );
};
```

## Common Patterns

### Pattern 1: Filter Rules by Entity

```typescript
const getFieldMetadataForEntity = (entity: string) => {
  const allMetadata = {
    'Employee': {
      employee_id: { type: 'string' },
      salary: { type: 'number' },
      hire_date: { type: 'date' },
    },
    'Department': {
      dept_code: { type: 'string' },
      budget: { type: 'number' },
    },
  };
  return allMetadata[entity] || {};
};

// In component:
const currentEntity = formData.target_entity;
const relevantMetadata = getFieldMetadataForEntity(currentEntity);

<ValidationRuleCreator
  fieldMetadata={relevantMetadata}
  // ...
/>
```

### Pattern 2: Fetch Metadata from Backend

```typescript
const [fieldMetadata, setFieldMetadata] = useState({});

useEffect(() => {
  const fetchMetadata = async () => {
    const response = await fetch(`/api/entities/${entityId}/field-schema`);
    const metadata = await response.json();
    setFieldMetadata(metadata);
  };
  fetchMetadata();
}, [entityId]);

<ValidationRuleCreator
  fieldMetadata={fieldMetadata}
  // ...
/>
```

### Pattern 3: Editing Existing Rules

```typescript
const [selectedRule, setSelectedRule] = useState<ValidationRule | null>(null);

const handleEdit = (rule: ValidationRule) => {
  setSelectedRule(rule);  // ← Pass to initialRule prop
  setIsOpen(true);
};

const handleSave = (updatedRule: ValidationRule) => {
  // Update in list
  setRules(prev => 
    prev.map(r => r.id === updatedRule.id ? updatedRule : r)
  );
  setIsOpen(false);
  setSelectedRule(null);
};

<ValidationRuleCreator
  initialRule={selectedRule}  // ← Edit mode
  onSave={handleSave}
  // ...
/>
```

### Pattern 4: Multi-Tenant Support

```typescript
<ValidationRuleCreator
  tenantId={currentTenant.id}
  datasourceId={currentDatasource.id}
  fieldMetadata={getFieldsForTenant(currentTenant.id)}
  // ...
/>
```

## Prop Reference

### Required Props

| Prop | Type | Description |
|------|------|-------------|
| `onSave` | `(rule: ValidationRule) => void` | Called when rule is created/updated |
| `availableEntities` | `string[]` | List of entities to validate against |

### Optional Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `isOpen` | `boolean` | `true` | Modal visibility (modal mode only) |
| `onClose` | `() => void` | – | Called when modal closes |
| `tenantId` | `string` | – | Tenant scope ID |
| `datasourceId` | `string` | – | Datasource scope ID |
| `displayMode` | `'modal' \| 'inline'` | `'modal'` | Display mode |
| `className` | `string` | `''` | Additional CSS classes |
| `initialRule` | `ValidationRule \| null` | `null` | Rule to edit (enables edit mode) |
| `fieldMetadata` | `Record<string, FieldTypeInfo>` | `{}` | Field type information |

## Condition Structure

### Creating Conditions Programmatically

```typescript
const conditions = [
  {
    field: 'salary',
    operator: 'greater_than',
    value: '50000',
  },
  {
    field: 'hire_date',
    operator: 'is_not_empty',
    value: '',  // Empty for stateless operators
  },
];

const rule: ValidationRule = {
  id: 'rule_1',
  rule_name: 'High Earners Check',
  description: 'Validates high-earning employees',
  rule_type: 'business_logic',
  target_entity: 'Employee',
  severity: 'warning',
  is_active: true,
  conditions,
};
```

### Rendering Conditions

```typescript
{rule.conditions?.map((cond, i) => (
  <div key={i}>
    <strong>{cond.field}</strong> {cond.operator}
    {cond.value && ` "${cond.value}"`}
  </div>
))}
```

## Operators by Type

### String Type Operators
- `equals` – Exact match (requires value)
- `not_equals` – Not equal (requires value)
- `contains` – Contains substring (requires value)
- `starts_with` – Starts with (requires value)
- `ends_with` – Ends with (requires value)
- `in_list` – In comma-separated list (requires value)
- `is_empty` – Empty/null (no value)
- `is_not_empty` – Has value (no value)

### Number Type Operators
- `equals` – Exact match (requires value)
- `not_equals` – Not equal (requires value)
- `greater_than` – Greater than (requires value)
- `less_than` – Less than (requires value)
- `is_empty` – Empty/null (no value)
- `is_not_empty` – Has value (no value)

### Date Type Operators
- `equals` – Same date (requires value)
- `not_equals` – Different date (requires value)
- `greater_than` – Later than (requires value)
- `less_than` – Earlier than (requires value)
- `is_empty` – Empty/null (no value)
- `is_not_empty` – Has value (no value)

### Boolean Type Operators
- `equals` – Exact match (requires value)
- `not_equals` – Not equal (requires value)
- `is_empty` – Empty/null (no value)
- `is_not_empty` – Has value (no value)

### Enum Type Operators
- `equals` – Exact match (requires value)
- `not_equals` – Not equal (requires value)
- `in_list` – In list (requires value)
- `is_empty` – Empty/null (no value)
- `is_not_empty` – Has value (no value)

## Testing

### Test Scenario 1: Type Filtering Works

```typescript
test('shows only string operators for string fields', () => {
  render(
    <ValidationRuleCreator
      fieldMetadata={{ name: { type: 'string' } }}
      // ...
    />
  );
  
  // Add condition with 'name' field
  // Verify dropdown shows: equals, contains, starts_with, etc.
  // Verify NO: greater_than, less_than
});
```

### Test Scenario 2: Value Hidden for Stateless Operators

```typescript
test('hides value input for is_empty operator', () => {
  render(
    <ValidationRuleCreator
      fieldMetadata={{ email: { type: 'string' } }}
      // ...
    />
  );
  
  // Add condition
  // Select operator 'is_empty'
  // Verify value input is NOT in DOM
  // Verify message shows: "doesn't require a value"
});
```

### Test Scenario 3: Save with Conditions

```typescript
test('saves rule with conditions', async () => {
  const onSave = jest.fn();
  
  render(
    <ValidationRuleCreator
      onSave={onSave}
      fieldMetadata={{ salary: { type: 'number' } }}
      // ...
    />
  );
  
  // Create rule with conditions
  // Click save
  // Verify onSave called with rule containing conditions
  
  expect(onSave).toHaveBeenCalledWith(
    expect.objectContaining({
      conditions: [
        { field: 'salary', operator: 'greater_than', value: '50000' }
      ]
    })
  );
});
```

## Troubleshooting

### Q: Operators not filtering for my field type
**A:** Ensure you passed `fieldMetadata` prop with the field's type:
```typescript
fieldMetadata={{ myField: { type: 'string' } }}  // ← Required
```

### Q: Value field still showing for "is_empty"
**A:** Check the operator's `requiresValue` property. Make sure:
```typescript
// In ALL_OPERATORS config
{ value: 'is_empty', label: 'Is Empty', requiresValue: false, ... }
```

### Q: Type detection shows "unknown"
**A:** Field not in metadata. Either:
1. Add to fieldMetadata: `{ myField: { type: 'string' } }`
2. Or accept "unknown" – all operators will show

### Q: Conditions not saving
**A:** Ensure `onSave` callback properly handles the rule object with conditions:
```typescript
onSave={(rule) => {
  console.log(rule.conditions);  // Should have array of conditions
  // Save to backend or state
}}
```

## API Integration Example

```typescript
const handleSave = async (rule: ValidationRule) => {
  try {
    const response = await fetch(
      `/api/bundles/${bundleId}/validation-rules`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify(rule),
      }
    );
    
    const savedRule = await response.json();
    setRules(prev => [savedRule, ...prev]);
    setIsOpen(false);
  } catch (error) {
    console.error('Failed to save rule:', error);
    // Show error toast
  }
};
```

## Next Steps

1. **Define your field metadata** - Determine field types for your domain
2. **Pass metadata to component** - Update ValidationRuleCreator props
3. **Test the UX** - Verify operators filter correctly
4. **Connect to backend** - Save rules via your API
5. **Display saved rules** - List and allow editing of rules

See `ValidationRuleCreatorDemo.tsx` for a complete working example!
