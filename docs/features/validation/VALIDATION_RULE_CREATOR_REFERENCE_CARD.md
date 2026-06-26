# ValidationRuleCreator: Feature Reference Card

## Quick Reference

### Component Props

```typescript
<ValidationRuleCreator
  isOpen={boolean}                                    // Modal visibility
  onClose={() => void}                               // Close handler
  onSave={(rule: ValidationRule) => void}            // Save handler [REQUIRED]
  tenantId="uuid"                                    // Optional tenant scope
  datasourceId="uuid"                                // Optional datasource scope
  availableEntities={['Entity1', 'Entity2']}        // Entity list [REQUIRED]
  displayMode="modal" | "inline"                     // Display mode
  className="..."                                    // Custom CSS
  initialRule={rule}                                 // For edit mode
  fieldMetadata={{                                   // NEW: Type metadata
    fieldName: { type: 'string|number|date|boolean|enum|unknown' }
  }}
/>
```

### Field Metadata Pattern

```typescript
const fieldMetadata: Record<string, FieldTypeInfo> = {
  // String: text, email, codes
  name: { type: 'string', isNullable: false },
  
  // Number: int, float, amounts
  salary: { type: 'number', isNullable: false },
  
  // Date: timestamps, dates
  hire_date: { type: 'date', isNullable: true },
  
  // Boolean: flags, states
  is_active: { type: 'boolean', isNullable: false },
  
  // Enum: fixed set of values
  department: { 
    type: 'enum', 
    enumValues: ['HR', 'Sales', 'Eng'],
    isNullable: false 
  },
  
  // Unknown: no type info (shows all operators)
  custom_field: { type: 'unknown' }
};
```

## Operator Quick Lookup

### Operators by Name

| Name | Icon | RequiresValue | Types | Example |
|------|------|---------------|-------|---------|
| Equals | = | ✓ | All | `salary = 100000` |
| Not Equals | ≠ | ✓ | All | `status ≠ inactive` |
| Contains | ∋ | ✓ | string | `name contains "John"` |
| Starts With | → | ✓ | string | `code starts_with "EMP"` |
| Ends With | ← | ✓ | string | `email ends_with "@company.com"` |
| Greater Than | > | ✓ | number, date | `salary > 50000` |
| Less Than | < | ✓ | number, date | `hire_date < 2020` |
| Is Empty | ∅ | ✗ | All | `email is_empty` |
| Is Not Empty | ∅⁻¹ | ✗ | All | `phone is_not_empty` |
| In List | ∈ | ✓ | string, number, enum | `dept in "HR,Sales"` |

### Type → Operators Matrix

```
                 equals  contains  >/<  empty  in_list
string            ✓        ✓       ✗      ✓      ✓
number            ✓        ✗       ✓      ✓      ✓
date              ✓        ✗       ✓      ✓      ✗
boolean           ✓        ✗       ✗      ✓      ✗
enum              ✓        ✗       ✗      ✓      ✓
unknown           ✓        ✓       ✓      ✓      ✓
```

## Condition Structure

```typescript
interface Condition {
  field: string;      // e.g., "salary"
  operator: string;   // e.g., "greater_than"
  value: string;      // e.g., "50000", "" for empty
}

// Examples:
{ field: 'salary', operator: 'greater_than', value: '100000' }
{ field: 'email', operator: 'is_empty', value: '' }
{ field: 'dept', operator: 'in_list', value: 'HR,Sales' }
```

## Condition Step-by-Step

```
1. User clicks "Add Condition"
   ↓
2. Input field name
   ↓ [Component detects type from metadata]
3. Select operator
   ↓ [Only relevant operators shown]
4. Enter value (if operator requires it)
   ↓ [Value field shown/hidden based on operator]
5. Remove or add more conditions
   ↓
6. Save rule with all conditions
```

## Code Examples

### Basic Implementation

```tsx
import { ValidationRuleCreator } from './ValidationRuleCreator';

const metadata = {
  name: { type: 'string' },
  salary: { type: 'number' },
};

export const RuleBuilder = () => {
  const handleSave = (rule) => console.log('Rule:', rule);
  
  return (
    <ValidationRuleCreator
      onSave={handleSave}
      availableEntities={['Employee']}
      fieldMetadata={metadata}
    />
  );
};
```

### With Edit Mode

```tsx
const [rule, setRule] = useState(null);

<ValidationRuleCreator
  initialRule={rule}  // Set to enable edit mode
  onSave={(updated) => {
    setRule(updated);
    // Send to backend
  }}
/>
```

### With Tenant Scope

```tsx
<ValidationRuleCreator
  tenantId={tenant.id}
  datasourceId={datasource.id}
  fieldMetadata={getMetadataForTenant(tenant.id)}
/>
```

## UI States

### Condition Card States

```
┌─ Not Started ────────────────────────────────────┐
│ [Add Condition]                                  │
│ No conditions yet — add one to narrow the rule   │
└──────────────────────────────────────────────────┘

┌─ Field Entered ──────────────────────────────────┐
│ Field (string)  ← Type detected                  │
│ [field_name         ]                            │
│ ℹ️ Available operators for string type            │
│ Operator [Select...  ▼]                          │
│ Value [disabled]                                 │
└──────────────────────────────────────────────────┘

┌─ Operator Selected (requires value) ──────────────┐
│ Field (string)                                    │
│ [salary                ]                          │
│ Operator [greater_than ▼]                         │
│ Value [100000         ]  ← Enabled                │
│                                    [Remove]       │
└───────────────────────────────────────────────────┘

┌─ Operator Selected (no value) ────────────────────┐
│ Field (date)                                      │
│ [hire_date             ]                          │
│ Operator [is_empty     ▼]                         │
│ ✓ Operator doesn't require a value                │
│                                    [Remove]       │
└───────────────────────────────────────────────────┘
```

## Validation Flow

```
Step 1: Rule Name & Description
├─ Required: rule_name
├─ Required: description
└─ Optional: sub_entity_type

Step 2: Rule Type & Target
├─ Required: rule_type
├─ Required: target_entity
└─ Optional: sub_entity_type

Step 3: Severity & Flags
├─ Required: severity
├─ Optional: is_global
├─ Optional: is_active
└─ Optional: conditions

Step 4: Conditions
├─ Optional: conditions (leave empty for all records)
├─ Each condition has: field, operator, value
└─ Save rule
```

## Error Prevention

```
❌ Before                    ✅ After
User selects wrong          → Only valid operators shown
operator for field type       for selected type

Value field confuses        → Value hidden for empty checks
when using is_empty           with helpful message

No guidance on valid        → Field type shown with
operators                     guidance text

Too many choices (9)        → Filtered list (4-6)
```

## Real Usage Examples

### Example 1: Email Validation Rule

```typescript
const rule = {
  rule_name: 'Email Required',
  rule_type: 'field_format',
  target_entity: 'Employee',
  severity: 'error',
  conditions: [
    {
      field: 'email',
      operator: 'is_not_empty',  // Hidden value field
      value: ''
    }
  ]
};
```

### Example 2: Salary Range Validation

```typescript
const rule = {
  rule_name: 'Valid Salary Range',
  rule_type: 'business_logic',
  target_entity: 'Employee',
  severity: 'warning',
  conditions: [
    {
      field: 'salary',
      operator: 'greater_than',   // Shows value field
      value: '30000'
    },
    {
      field: 'salary',
      operator: 'less_than',       // Shows value field
      value: '500000'
    }
  ]
};
```

### Example 3: Department Assignment

```typescript
const rule = {
  rule_name: 'Valid Department',
  rule_type: 'referential_integrity',
  target_entity: 'Employee',
  severity: 'error',
  conditions: [
    {
      field: 'department',
      operator: 'in_list',         // Shows value field
      value: 'HR,Sales,Engineering'
    }
  ]
};
```

## Data Flow Diagram

```
┌─────────────┐
│ User Input  │
└──────┬──────┘
       │
       ▼
┌──────────────────────┐
│ Field Name Entered   │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────────────────┐
│ Lookup in fieldMetadata          │
│ Get type (string|number|...)     │
└──────┬───────────────────────────┘
       │
       ▼
┌──────────────────────────────────┐
│ Filter Operators by Type         │
│ getOperatorsForFieldType()       │
└──────┬───────────────────────────┘
       │
       ▼
┌──────────────────────────────────┐
│ Show Filtered Operator Dropdown  │
│ Update UI based on requiresValue │
└──────┬───────────────────────────┘
       │
       ▼
┌──────────────────────────────────┐
│ User Selects Operator            │
└──────┬───────────────────────────┘
       │
       ├─ requiresValue = true?
       │  └─ Show value input
       │
       └─ requiresValue = false?
          └─ Hide value input, show message
```

## Troubleshooting Matrix

| Issue | Cause | Solution |
|-------|-------|----------|
| All operators showing | No metadata provided | Pass fieldMetadata prop |
| Wrong operators showing | Wrong type in metadata | Check `type` field in FieldTypeInfo |
| Value field not hiding | Operator config wrong | Verify `requiresValue: false` |
| Condition not saving | Missing onSave | Add `onSave` prop |
| Edit mode not working | initialRule not set | Pass rule to `initialRule` prop |
| Type not detected | Field not in metadata | Add field to fieldMetadata object |

## Keyboard Shortcuts (Future)

```
Ctrl+Enter      Save rule
Escape          Close modal / Discard changes
Tab             Navigate to next field
Shift+Tab       Navigate to previous field
Ctrl+Shift+C    Add condition
```

## Accessibility Features

- ✓ All inputs have labels
- ✓ ARIA attributes on selects
- ✓ Keyboard navigation supported
- ✓ Clear focus indicators
- ✓ Color + text for status (not color alone)
- ✓ Help text for complex fields

## Performance Tips

1. **Cache metadata** - Don't fetch on every render
2. **Memoize operators** - Filter once, reuse often
3. **Lazy load metadata** - Only load for current entity
4. **Batch conditions** - Add multiple at once if possible

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## File Sizes

- ValidationRuleCreator.tsx: ~19KB
- ValidationRuleCreatorDemo.tsx: ~6KB
- Documentation: ~25KB
- **Total: ~50KB** (gzipped ~12KB)

---

**Last Updated**: November 7, 2025  
**Version**: 2.0 (Smart Conditions)
