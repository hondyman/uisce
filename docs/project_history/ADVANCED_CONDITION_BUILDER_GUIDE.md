# Advanced Condition Builder - Implementation Guide

## Overview

The **Advanced Condition Builder** is a Workday-inspired visual rule engine that allows non-technical users to build complex validation rules with nested groups, AND/OR logic, and automatic evaluation. This component replaces the previous simpler DnD-based builder with a production-grade condition tree system.

## ✨ Key Features

### 1. **Nested Condition Groups**
- Users can create hierarchical condition groups using the "Group" button
- Each group can contain multiple conditions or other groups
- Supports unlimited nesting levels
- Visual indentation shows nesting depth

### 2. **AND/OR Logic Operators**
- Toggle between AND/OR operators with a single click
- Visual indicators show which operator is active for each group
- AND logic: all conditions must pass
- OR logic: any condition can pass
- Automatic evaluation respects operator precedence

### 3. **Dynamic Field Selection**
- Autocomplete field selector with type hints
- Fields are categorized by data type (string, number, date, boolean)
- Operators change automatically based on field type
- Field type badges for quick reference

### 4. **Type-Specific Operators**
- **String**: Equals, Contains, Starts With, Ends With, Is Empty, Not Equals
- **Number**: Equals, Greater Than, Less Than, Greater/Less or Equal, Not Equals
- **Date**: On Date, Before, After, Between
- **Boolean**: Is True, Is False

### 5. **Recursive Condition Evaluation**
```typescript
const result = evaluateCondition(conditionTree, testData);
// true/false based on all nested conditions and operators
```

### 6. **Autosave with Draft Management**
- Optional debounced autosave (configurable debounce interval, default 1000ms)
- Draft creation for new rules (insert_catalog_validation_rules_one)
- Update-by-pk for subsequent saves to avoid conflicts
- Tenant-scoped persistence with proper headers
- Retry logic with exponential backoff (up to 3 attempts)
- Best-effort flush on unmount

### 7. **Workday-Inspired UI/UX**
- Clean, card-based interface
- Blue accent colors matching Workday design
- Smooth transitions and hover states
- Collapsible groups for complex expressions
- Clear visual hierarchy with badges and icons

## 📁 File Structure

```
frontend/src/components/ExpressionBuilder/
├── AdvancedConditionBuilder.tsx          # Core builder component
├── AdvancedConditionBuilder.module.css   # Styling (CSS Modules)
├── ExpressionBuilder.tsx                 # Integration wrapper with autosave
├── ExpressionBuilder.module.css          # Builder wrapper styles
├── __tests__/
│   └── AdvancedConditionBuilder.test.tsx # Unit tests (to be updated)
└── [legacy components]
```

## 🔧 Component API

### AdvancedConditionBuilder

```typescript
interface AdvancedConditionBuilderProps {
  value: ConditionGroup;                    // Current condition tree
  onChange: (value: ConditionGroup) => void; // Called when tree changes
  availableFields: Array<{                  // Fields for dropdown
    name: string;
    type: string;
    label: string;
  }>;
  entityName: string;                       // Display name for context
}

export const AdvancedConditionBuilder: React.FC<AdvancedConditionBuilderProps>
```

### Condition Types

```typescript
interface Condition {
  id: string;
  field: string;
  operator: string;
  value: string;
  fieldType?: string;
}

interface ConditionGroup {
  id: string;
  type: 'group';
  operator: 'AND' | 'OR';
  conditions: (Condition | ConditionGroup)[];
}

type ConditionNode = Condition | ConditionGroup;
```

### Evaluation Function

```typescript
export const evaluateCondition = (
  node: ConditionNode, 
  data: Record<string, any>
): boolean
```

### ExpressionBuilder (Integration Wrapper)

```typescript
interface ExpressionBuilderProps {
  onSave?: (conditionJson: any) => void;           // Manual save callback
  onChange?: (conditionJson: any) => void;         // Changes callback
  autosave?: boolean;                               // Enable autosave (default: false)
  debounceMs?: number;                              // Debounce interval (default: 1000ms)
  ruleName?: string;                                // Rule name for persistence
  targetEntity?: string;                            // Target entity for validation
  ruleId?: string;                                  // Existing rule ID for updates
  onDraftCreated?: (id: string, ruleName?: string) => void; // Draft creation callback
}
```

## 🚀 Usage Examples

### Basic Integration

```tsx
import AdvancedConditionBuilder, { 
  ConditionGroup,
  evaluateCondition 
} from './AdvancedConditionBuilder';

function MyComponent() {
  const [conditionTree, setConditionTree] = useState<ConditionGroup>({
    id: 'root',
    type: 'group',
    operator: 'AND',
    conditions: []
  });

  const availableFields = [
    { name: 'age', type: 'number', label: 'Age' },
    { name: 'status', type: 'string', label: 'Status' }
  ];

  return (
    <AdvancedConditionBuilder
      value={conditionTree}
      onChange={setConditionTree}
      availableFields={availableFields}
      entityName="Employee"
    />
  );
}
```

### With Autosave

```tsx
import ExpressionBuilder from './ExpressionBuilder';

function RuleEditor() {
  const handleDraftCreated = (id: string, ruleName?: string) => {
    // Update editing state with new draft ID
    setEditingRuleId(id);
    setEditingRuleName(ruleName);
  };

  return (
    <ExpressionBuilder
      autosave={true}           // Enable autosave
      debounceMs={1000}         // Save after 1 second of inactivity
      ruleName="Income Validation Rule"
      targetEntity="Employee"
      onDraftCreated={handleDraftCreated}
    />
  );
}
```

### Condition Tree JSON Output

```json
{
  "id": "root",
  "type": "group",
  "operator": "AND",
  "conditions": [
    {
      "id": "cond_123",
      "field": "age",
      "operator": "greater_equal",
      "value": "18",
      "fieldType": "number"
    },
    {
      "id": "group_456",
      "type": "group",
      "operator": "OR",
      "conditions": [
        {
          "id": "cond_789",
          "field": "status",
          "operator": "equals",
          "value": "Active",
          "fieldType": "string"
        },
        {
          "id": "cond_101",
          "field": "is_vip",
          "operator": "is_true",
          "value": "true",
          "fieldType": "boolean"
        }
      ]
    }
  ]
}
```

### Evaluating Conditions

```tsx
const testData = {
  age: 25,
  status: 'Active',
  is_vip: true,
  email: 'user@example.com'
};

const conditionTree: ConditionGroup = {
  id: 'root',
  type: 'group',
  operator: 'AND',
  conditions: [
    {
      id: 'cond_1',
      field: 'age',
      operator: 'greater_equal',
      value: '18'
    },
    {
      id: 'cond_2',
      field: 'status',
      operator: 'equals',
      value: 'Active'
    }
  ]
};

const result = evaluateCondition(conditionTree, testData);
// result = true (age >= 18 AND status = Active)
```

## 🔐 Tenant Scoping & Autosave

The autosave system respects the mandatory tenant scope:

1. **Tenant Selection**: User selects tenant via localStorage
   ```javascript
   localStorage.setItem('selected_tenant', JSON.stringify({ id: '...', display_name: '...' }));
   ```

2. **Autosave Headers**: All mutations include tenant headers
   ```
   X-Tenant-ID: <TENANT_ID>
   X-Tenant-Datasource-ID: <DATASOURCE_ID>
   ```

3. **Draft Creation**: New rules are created with `is_active: false`
   ```graphql
   mutation {
     insert_catalog_validation_rules_one(object: {
       tenant_id: "..."
       rule_name: "Draft Rule"
       condition_json: {...}
       is_active: false
     }) { id }
   }
   ```

4. **Update by PK**: Subsequent saves use update-by-pk
   ```graphql
   mutation {
     update_catalog_validation_rules_by_pk(
       pk_columns: { id: "..." }
       _set: { condition_json: {...} }
     ) { id }
   }
   ```

## 🎨 Styling & Customization

The component uses CSS Modules for encapsulation. Key classes:

- `.builderContainer` - Main wrapper
- `.conditionItem` - Individual condition row
- `.group` - Nested group container
- `.groupHeader` - Group title and actions
- `.operatorButton` - AND/OR toggle
- `.fieldsGrid` - Field/Operator/Value inputs grid

Override styles by modifying `AdvancedConditionBuilder.module.css`:

```css
.conditionItem {
  /* Customize condition box styling */
  background-color: #ffffff;
  border: 2px solid #d1d5db;
}

.operatorAnd {
  /* Customize AND button */
  background-color: #1e40af;
  color: #ffffff;
}
```

## 🧪 Testing

### Unit Tests
- Draft creation on first autosave
- Update-by-pk for subsequent saves
- Flush-on-unmount triggering final save
- Nested group handling
- AND/OR operator toggling
- Condition evaluation with test data

### Manual Testing Checklist
- [ ] Add condition to empty builder
- [ ] Add nested group
- [ ] Toggle AND/OR operators
- [ ] Drag field selectors and set values
- [ ] Delete conditions and groups
- [ ] Collapse/expand groups
- [ ] Test evaluation with sample data
- [ ] Verify JSON output is valid
- [ ] Test autosave with draft creation
- [ ] Test update-by-pk after draft exists

## 🐛 Debugging

### Console Logs
```typescript
// In ExpressionBuilder.tsx
console.log('Condition tree:', conditionTree);
console.log('Evaluation result:', evaluateCondition(conditionTree, testData));
```

### Apollo DevTools
```typescript
// View mutations in Apollo DevTools
// - INSERT_DRAFT_RULE
// - UPDATE_RULE_BY_PK
```

### Tenant Scope Verification
```javascript
// Verify tenant is set
JSON.parse(localStorage.getItem('selected_tenant'))
JSON.parse(localStorage.getItem('selected_datasource'))
```

## 🚧 Future Enhancements

1. **Smart Field Autocomplete**
   - Searchable dropdown with recent fields
   - Field type icons and descriptions
   - Related entity traversal (employee.department.name)

2. **Rule Templates**
   - Pre-built templates for common patterns
   - Clone existing rules
   - Industry-specific rule libraries

3. **Live Preview & Testing**
   - Sample data generator
   - Before/After visualization
   - Pass/fail statistics

4. **Rule Impact Analysis**
   - Estimated affected records
   - Conflict detection with existing rules
   - Performance impact estimate

5. **Collaboration Features**
   - Comments on rules
   - Approval workflows
   - Change audit trail
   - Rule versioning and history

## 📚 References

- [Workday Condition Builder Documentation](https://www.workday.com)
- [GraphQL Mutations in Apollo Client](https://www.apollographql.com/docs/react/data/mutations/)
- [Tenant-Scoped Architecture](./agents.md)
- [Validation Rules Database Schema](./BACKEND_VALIDATION_INTEGRATION.md)
