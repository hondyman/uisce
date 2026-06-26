# Rules Management - Quick Reference & Examples

## Quick Start: Using the Improved Components

### Example 1: Minimal Usage

```typescript
import { useValidationRulesAPI } from '../hooks/useValidationRulesAPI';

export const MyRulesPage: React.FC = () => {
  const { tenant, datasource } = useTenant();

  const { rules, loading, createRule, deleteRule } = useValidationRulesAPI({
    tenantId: tenant?.id,
    datasourceId: datasource?.id,
  });

  return (
    <div>
      <RulesList 
        rules={rules} 
        loading={loading}
        onDelete={deleteRule}
      />
    </div>
  );
};
```

### Example 2: Full Implementation

```typescript
import { useValidationRulesAPI } from '../hooks/useValidationRulesAPI';
import { useValidationRuleForm } from '../hooks/useValidationRuleForm';
import RulesList from '../components/RulesList';

export const RulesPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const [showForm, setShowForm] = useState(false);

  // API Management
  const { rules, loading, saving, error, createRule, updateRule, deleteRule } = 
    useValidationRulesAPI({
      tenantId: tenant?.id,
      datasourceId: datasource?.id,
      onSuccess: (action) => toast.show(`✓ ${action} successful`),
      onError: (action, err) => toast.show(`✗ ${err.message}`),
    });

  // Form Management
  const form = useValidationRuleForm({
    onSubmit: async (formData) => {
      if (formData.id) {
        await updateRule(formData.id, formData);
      } else {
        await createRule(formData);
      }
      setShowForm(false);
      form.resetToBlank();
    },
  });

  return (
    <div>
      {/* Rules List */}
      <RulesList 
        rules={rules}
        loading={loading}
        onEdit={(rule) => {
          form.setFormData(rule);
          setShowForm(true);
        }}
        onDelete={deleteRule}
      />

      {/* Form Modal */}
      {showForm && (
        <form onSubmit={form.handleSubmit}>
          {/* Display errors */}
          {form.getAllErrors().length > 0 && (
            <div className="errors">
              {form.getAllErrors().map(err => <p>{err}</p>)}
            </div>
          )}

          {/* Name field */}
          <input 
            value={form.formData.name}
            onChange={(e) => form.updateField('name', e.target.value)}
            onBlur={() => form.touchField('name')}
            className={form.hasFieldError('name') ? 'error' : ''}
          />
          {form.getFieldError('name') && (
            <span className="error-text">{form.getFieldError('name')}</span>
          )}

          {/* Buttons */}
          <button type="submit" disabled={form.isSubmitting}>
            {form.isSubmitting ? 'Saving...' : 'Save'}
          </button>
        </form>
      )}
    </div>
  );
};
```

## Common Patterns

### Pattern 1: Search & Filter

```typescript
const [searchTerm, setSearchTerm] = useState('');
const [filterType, setFilterType] = useState('ALL');

<RulesList
  rules={rules}
  searchTerm={searchTerm}
  filterType={filterType}
  sortBy="name"
/>
```

### Pattern 2: Delete with Confirmation

```typescript
const handleDelete = async (ruleId: string) => {
  if (!window.confirm('Delete this rule?')) return;
  try {
    await deleteRule(ruleId);
    toast.show('✓ Deleted');
  } catch (err) {
    toast.show(`✗ ${err.message}`);
  }
};
```

### Pattern 3: Edit Mode

```typescript
const handleEdit = (rule: any) => {
  const formData = buildRuleFormDataFromRule(rule);
  form.setFormData(formData);
  setShowForm(true);
};
```

### Pattern 4: Detect Unsaved Changes

```typescript
if (form.hasChanges) {
  // Warn user before leaving
  window.onbeforeunload = () => 'You have unsaved changes';
}
```

### Pattern 5: Retry Failed Operation

```typescript
{error && (
  <div>
    <p>{error.message}</p>
    <button onClick={() => retryOperation(lastOperationId)}>
      Retry
    </button>
  </div>
)}
```

## Hook API Reference

### `useValidationRulesAPI(options)`

```typescript
interface UseValidationRulesAPIOptions {
  tenantId?: string;
  datasourceId?: string;
  onSuccess?: (action: string, rule?: any) => void;
  onError?: (action: string, error: Error) => void;
}

// Returns
{
  // State
  rules: any[];
  loading: boolean;
  saving: boolean;
  error: Error | null;

  // Methods
  loadRules(force?: boolean): Promise<any[]>;
  createRule(formData): Promise<any>;
  updateRule(ruleId, formData): Promise<any>;
  deleteRule(ruleId): Promise<boolean>;
  retryOperation(operationId): Promise<boolean>;
  clearError(): void;
  getPendingOperationsCount(): number;
}
```

### `useValidationRuleForm(options)`

```typescript
interface UseValidationRuleFormOptions {
  onSubmit?: (formData: ValidationRuleFormData) => Promise<void>;
  initialRule?: any;
}

// Returns
{
  // State
  formData: ValidationRuleFormData;
  errors: Record<string, string>;
  touched: Set<string>;
  isSubmitting: boolean;
  submitError: string | null;
  hasChanges: boolean;

  // Field Methods
  updateField(field, value): void;
  validateField(field, value?): boolean;
  touchField(field): void;
  
  // Form Methods
  validateForm(): boolean;
  handleSubmit(e?): Promise<boolean>;
  reset(): void;
  resetToBlank(): void;
  
  // Error Methods
  getFieldError(field): string | undefined;
  hasFieldError(field): boolean;
  getAllErrors(): string[];
  isValid(): boolean;
  
  // Direct State Setters
  setErrors(errors): void;
  setFormData(formData): void;
  setSubmitError(error): void;
}
```

## Utility Functions Reference

### Validation Functions

```typescript
import { validateRuleForm, isRuleComplete, hasRuleChanged } from '../lib/ruleUtils';

// Validate all fields
const errors = validateRuleForm(formData);
// Returns: ['Rule name is required', 'At least one account type must be selected']

// Check if form is complete
if (isRuleComplete(formData)) {
  // All required fields filled
}

// Detect changes
if (hasRuleChanged(originalRule, currentFormData)) {
  // User made changes - show save button
}
```

### Color Functions

```typescript
import {
  getRuleTypeBadgeColorClasses,
  getSeverityBadgeColorClasses,
  getStatusBadgeColorClasses,
} from '../lib/ruleUtils';

// Get Tailwind classes for badge
<span className={getRuleTypeBadgeColorClasses('CONCENTRATION')}>
  CONCENTRATION
</span>

<span className={getSeverityBadgeColorClasses('BLOCK')}>
  BLOCK
</span>

<span className={getStatusBadgeColorClasses(true)}>
  Active
</span>
```

### Data Builder Functions

```typescript
import {
  createDefaultRuleFormData,
  buildRuleFormDataFromRule,
  buildCreateRulePayload,
  buildUpdateRulePayload,
  formatAccountTypes,
} from '../lib/ruleUtils';

// Create blank form
const blank = createDefaultRuleFormData();

// Load existing rule for editing
const editData = buildRuleFormDataFromRule(existingRule);

// Build API payload for create
const createPayload = buildCreateRulePayload(formData, tenantId, datasourceId);

// Build API payload for update
const updatePayload = buildUpdateRulePayload(formData, tenantId, datasourceId);

// Format account types for display
const display = formatAccountTypes(['IRA_ACCOUNT', 'TRUST_ACCOUNT']);
// Returns: "IRA_ACCOUNT, TRUST_ACCOUNT"
```

## Component Props Reference

### `<RulesList />`

```typescript
interface RulesListProps {
  rules: any[];
  loading: boolean;
  onEdit: (rule: any) => void;
  onDelete: (ruleId: string) => void;
  onCreateNew: () => void;
  filterType?: string;           // Optional: filter by type
  searchTerm?: string;            // Optional: search filter
  sortBy?: 'name' | 'type' | 'severity' | 'order';
}

// Usage
<RulesList
  rules={rules}
  loading={loading}
  onEdit={handleEdit}
  onDelete={handleDelete}
  onCreateNew={handleNew}
  filterType="CONCENTRATION"
  searchTerm="position"
  sortBy="severity"
/>
```

### `<RuleCard />`

```typescript
interface RuleCardProps {
  rule: any;
  onEdit: (rule: any) => void;
  onDelete: (ruleId: string) => void;
  isDeleting?: boolean;
}

// Usage
{rules.map(rule => (
  <RuleCard
    key={rule.id}
    rule={rule}
    onEdit={handleEdit}
    onDelete={handleDelete}
    isDeleting={deletingId === rule.id}
  />
))}
```

## Testing Examples

### Test the Hook

```typescript
import { renderHook, act } from '@testing-library/react';
import { useValidationRuleForm } from '../hooks/useValidationRuleForm';

test('validate required field', () => {
  const { result } = renderHook(() => useValidationRuleForm());

  act(() => {
    result.current.validateField('name');
  });

  expect(result.current.getFieldError('name')).toBe('Rule name is required');
});

test('detect unsaved changes', () => {
  const initialRule = { id: '1', name: 'Original', ruleType: 'CONCENTRATION' };
  const { result } = renderHook(() => 
    useValidationRuleForm({ initialRule })
  );

  act(() => {
    result.current.updateField('name', 'Modified');
  });

  expect(result.current.hasChanges).toBe(true);
});
```

### Test the Utility

```typescript
import { validateRuleForm, hasRuleChanged } from '../lib/ruleUtils';

test('should detect incomplete form', () => {
  const formData = {
    name: '',
    description: '',
    ruleType: 'CONCENTRATION',
    accountTypes: [],
    severity: 'BLOCK',
    isActive: true,
    evaluationOrder: 100,
    allowOverride: false,
    parameters: {},
  };

  const errors = validateRuleForm(formData);
  expect(errors.length).toBeGreaterThan(0);
});

test('should detect changes', () => {
  const original = { id: '1', name: 'Original' };
  const current = { id: '1', name: 'Modified' };

  expect(hasRuleChanged(original, current)).toBe(true);
});
```

## Performance Tips

### 1. Memoize Expensive Computations

```typescript
const availableTypes = useMemo(() => {
  // This is expensive when rules has 1000s of items
  const types = new Set(rules.map(r => r.ruleType));
  return Array.from(types).sort();
}, [rules]); // Only recalculate when rules change
```

### 2. Memoize Callbacks

```typescript
const handleEdit = useCallback((rule) => {
  // This callback is recreated every render
  // Memoizing prevents child re-renders
  form.setFormData(rule);
  setShowForm(true);
}, [form]); // Only recreate when form changes
```

### 3. Use React.memo for Lists

```typescript
// RuleCard is already wrapped with memo
// This prevents re-renders of individual cards
const { RuleCard } = require('../components/RuleCard');
```

## Troubleshooting

### Issue: Form doesn't validate

**Solution:** Make sure to call `validateForm()` or `handleSubmit()` explicitly
```typescript
const handleSave = async () => {
  if (!form.validateForm()) return; // Validate first
  // Then proceed
};
```

### Issue: Errors not showing

**Solution:** Check if field is touched before showing error
```typescript
// WRONG: Shows error even before user touches field
{form.getFieldError('name') && <span>{form.getFieldError('name')}</span>}

// RIGHT: Only show after user interaction
{form.hasFieldError('name') && <span>{form.getFieldError('name')}</span>}
```

### Issue: Changes not detected

**Solution:** Make sure to use `updateField` method
```typescript
// WRONG: Won't trigger change detection
form.formData.name = 'New Value';

// RIGHT: Triggers change detection
form.updateField('name', 'New Value');
```

### Issue: Optimistic update showing old data

**Solution:** Check if rollback happened and retry
```typescript
{error && (
  <button onClick={() => loadRules()}>
    Reload Rules
  </button>
)}
```

---

## Migration Guide: Old → New Code

### Before (Monolithic)

```typescript
const [formData, setFormData] = useState({...});
const [errors, setErrors] = useState({});
const [saving, setSaving] = useState(false);

const handleSaveRule = async () => {
  if (!formData.name) {
    setErrors({ name: 'Required' });
    return;
  }
  setSaving(true);
  try {
    await engine.createRule(formData);
    // Hoping server responds before user navigates away
  } catch (err) {
    setErrors({ form: err.message });
  } finally {
    setSaving(false);
  }
};
```

### After (Modular with Hooks)

```typescript
const form = useValidationRuleForm();
const { createRule, saving } = useValidationRulesAPI({...});

const handleSaveRule = async () => {
  const success = await form.handleSubmit();
  if (!success) return; // Validation failed
  
  try {
    await createRule(form.formData);
    // Instant feedback (optimistic update)
    // Automatic rollback if error
  } catch (err) {
    // Already handled and displayed
  }
};
```

---

## Conclusion

You now have:
- ✅ Reusable hooks for any rule management interface
- ✅ Professional error handling and validation
- ✅ Optimistic updates for better UX
- ✅ Performance-optimized components
- ✅ Clear separation of concerns
- ✅ Production-ready code

**Happy coding! 🚀**
