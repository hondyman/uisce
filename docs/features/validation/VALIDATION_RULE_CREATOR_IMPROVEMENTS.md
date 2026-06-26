# ValidationRuleCreator: Smart Condition Builder

## Overview

The improved `ValidationRuleCreator` component now features an intelligent condition-building experience that adapts to your data types. When you select a field, the available operators and UI elements automatically adjust based on the field's data type.

## Key Improvements

### 1. **Type-Aware Operator Filtering**

When you select a field and its type is known, only relevant operators are displayed:

- **String fields**: `equals`, `not_equals`, `contains`, `starts_with`, `ends_with`, `is_empty`, `is_not_empty`, `in_list`
- **Number fields**: `equals`, `not_equals`, `greater_than`, `less_than`, `is_empty`, `is_not_empty`
- **Date fields**: `equals`, `not_equals`, `greater_than`, `less_than`, `is_empty`, `is_not_empty`
- **Boolean fields**: `equals`, `not_equals`, `is_empty`, `is_not_empty`
- **Enum fields**: `equals`, `not_equals`, `in_list`, `is_empty`, `is_not_empty`

This prevents the user from selecting invalid operators for their data type.

### 2. **Smart Value Input Visibility**

Operators that don't require a value input (like `is_empty` and `is_not_empty`) automatically hide the value field:

```
Field: employee_id (string)
Operator: is_empty        ← No value needed
✓ Operator 'is_empty' doesn't require a value — it checks the field state only
```

When you select a stateless operator, the component shows a helpful inline message and the value input is hidden.

### 3. **Field Type Detection Hints**

Each condition row displays the detected field type:

```
Field (string) ← Auto-detected type hint
employee_id
ℹ️ Available operators for string type shown below
```

If the field type is unknown, the user is informed and all operators are shown as options.

### 4. **Better Visual Hierarchy**

Condition rows are now structured with:
- Clear section headers for each input (Field, Operator, Value)
- Type hints next to labels
- Helpful guidance text and feedback
- Distinct visual grouping with better spacing

### 5. **Operator Labels Include State Info**

In the operator dropdown, operators that don't require values show a hint:

```
Equals
Not Equals
Contains
Is Empty (no value needed)     ← Visual hint in dropdown
Is Not Empty (no value needed) ← Visual hint in dropdown
```

## Usage

### Basic Usage

```tsx
import { ValidationRuleCreator } from './ValidationRuleCreator';
import type { FieldTypeInfo } from './ValidationRuleCreator';

const MyComponent = () => {
  const [isOpen, setIsOpen] = useState(false);

  const fieldMetadata: Record<string, FieldTypeInfo> = {
    employee_id: { type: 'string', isNullable: false },
    salary: { type: 'number', isNullable: false },
    hire_date: { type: 'date', isNullable: true },
    department: {
      type: 'enum',
      enumValues: ['HR', 'Engineering', 'Sales'],
      isNullable: false,
    },
  };

  return (
    <>
      <button onClick={() => setIsOpen(true)}>Create Rule</button>
      
      <ValidationRuleCreator
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        onSave={(rule) => console.log(rule)}
        availableEntities={['Employee']}
        fieldMetadata={fieldMetadata}
      />
    </>
  );
};
```

### Field Metadata Structure

```typescript
interface FieldTypeInfo {
  type: 'string' | 'number' | 'boolean' | 'date' | 'enum' | 'unknown';
  enumValues?: string[];  // Required for 'enum' type
  isNullable?: boolean;
}
```

### Props

```typescript
interface ValidationRuleCreatorProps {
  isOpen?: boolean;                                    // Modal visibility
  onClose?: () => void;                               // Called when modal closes
  onSave: (rule: ValidationRule) => void;             // Called with created rule
  tenantId?: string;                                  // Optional tenant scope
  datasourceId?: string;                              // Optional datasource scope
  availableEntities: string[];                        // List of entities to validate against
  displayMode?: 'modal' | 'inline';                   // Display mode (default: 'modal')
  className?: string;                                 // Optional CSS class
  initialRule?: ValidationRule | null;                // For edit mode
  fieldMetadata?: Record<string, FieldTypeInfo>;     // NEW: Type metadata for fields
}
```

## Condition Structure

Each condition follows this structure:

```typescript
interface Condition {
  field: string;      // Field name to validate
  operator: string;   // Operator: 'equals', 'is_empty', etc.
  value: string;      // Value to compare (empty for stateless operators)
}
```

## Operator Reference

| Operator | Requires Value | Supported Types | Description |
|----------|----------------|-----------------|------------|
| `equals` | Yes | All types | Exact match |
| `not_equals` | Yes | All types | Not matching |
| `contains` | Yes | string | Substring match |
| `starts_with` | Yes | string | String prefix |
| `ends_with` | Yes | string | String suffix |
| `greater_than` | Yes | number, date | Greater than |
| `less_than` | Yes | number, date | Less than |
| `is_empty` | No | All types | Field is empty/null |
| `is_not_empty` | No | All types | Field has value |
| `in_list` | Yes | string, number, enum | Value in comma-separated list |

## Example: Creating a Rule with Conditions

```tsx
const rule = {
  id: 'rule_123',
  rule_name: 'Active Employees Only',
  description: 'Validates that only active employees are processed',
  rule_type: 'business_logic',
  target_entity: 'Employee',
  severity: 'error',
  is_active: true,
  is_global: false,
  conditions: [
    {
      field: 'employee_status',
      operator: 'equals',
      value: 'ACTIVE',
    },
    {
      field: 'termination_date',
      operator: 'is_empty',
      value: '',  // Empty because operator doesn't need it
    },
  ],
};
```

## UX Workflow

1. **Add Condition** → Click "Add Condition" button
2. **Select Field** → Type field name (e.g., "salary")
   - Component detects field type (e.g., "number")
   - Shows hint: "Available operators for number type shown below"
3. **Select Operator** → Choose from filtered list
   - Only operators valid for that type are shown
   - Operators that don't need values are marked
4. **Enter Value** (if needed)
   - Value field hidden for stateless operators
   - Inline message explains why no value is needed
5. **Continue or Remove** → Add more conditions or delete this one

## Integration Notes

### With Backend APIs

When sending conditions to the backend:

```typescript
// Backend receives conditions as:
{
  field: 'salary',
  operator: 'greater_than',
  value: '100000'
}

// Backend applies:
WHERE salary > 100000
```

For stateless operators, value is ignored:

```typescript
{
  field: 'hire_date',
  operator: 'is_not_empty',
  value: ''  // Backend ignores this
}

// Backend applies:
WHERE hire_date IS NOT NULL
```

### Type Inference

If `fieldMetadata` is not provided or a field type is not known:
- All operators are shown as available
- User has full flexibility
- Type detection happens at runtime

When type information is available:
- Operators are filtered by supported types
- UI provides guidance on the detected type
- Better user experience with fewer invalid choices

## Performance Considerations

- Operator filtering is computed inline (fast, no API calls)
- Type metadata should be cached on the client
- Minimal re-renders: operators only update when field changes

## Accessibility

- All inputs have proper labels and ARIA attributes
- Operators marked with "(no value needed)" for screen readers
- Keyboard navigation fully supported
- Clear visual feedback for invalid states

## Future Enhancements

1. **Enum Value Suggestions**: Show dropdown with enum values for enum fields
2. **Advanced Conditions**: Support AND/OR logic between conditions
3. **Type Coercion**: Auto-format values based on field type
4. **Validation Rules**: Real-time validation of value format against field type
5. **Condition Templates**: Pre-built templates for common scenarios
