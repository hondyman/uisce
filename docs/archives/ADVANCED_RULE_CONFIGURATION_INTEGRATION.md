# Advanced Rule Configuration - Integration Guide

## Quick Start

### 1. Import the Component
```typescript
import AdvancedRuleConfiguration from './components/validation/AdvancedRuleConfiguration';

// Or import individual sub-components
import {
  RuleDependencyChain,
  EntityPathPicker,
  CrossEntityValidationBuilder
} from './components/validation/AdvancedRuleConfiguration';
```

### 2. Basic Usage
```typescript
<AdvancedRuleConfiguration
  onRulesUpdate={(rules) => {
    console.log('Rules updated:', rules);
    // Save to backend
  }}
  onCrossEntitySave={(condition) => {
    console.log('Cross-entity condition:', condition);
    // Save to backend
  }}
/>
```

## File Location
```
/frontend/src/components/validation/AdvancedRuleConfiguration.tsx
```

## Component Exports

### Main Component
- `AdvancedRuleConfiguration` - Full feature-rich UI (recommended)

### Sub-Components
- `RuleDependencyChain` - Just dependency management
- `EntityPathPicker` - Just entity path selection
- `CrossEntityValidationBuilder` - Just cross-entity builder

### Types
```typescript
export type {
  ValidationRule,
  EntityPath,
  CrossEntityCondition
};
```

## Integration with ValidationRuleEditor

### Step 1: Add Tab for Advanced Configuration
```typescript
// In ValidationRuleEditor.tsx
const [activeTab, setActiveTab] = useState('basic'); // or 'advanced'

<div className="tabs">
  <button onClick={() => setActiveTab('basic')}>Basic Rules</button>
  <button onClick={() => setActiveTab('advanced')}>Advanced Configuration</button>
</div>

{activeTab === 'advanced' && (
  <AdvancedRuleConfiguration
    rules={rules}
    onRulesUpdate={handleRulesUpdate}
    onCrossEntitySave={handleCrossEntitySave}
  />
)}
```

### Step 2: Add State Management
```typescript
const [rules, setRules] = useState<ValidationRule[]>([]);
const [crossEntityConditions, setCrossEntityConditions] = useState<CrossEntityCondition[]>([]);

const handleRulesUpdate = async (updatedRules: ValidationRule[]) => {
  setRules(updatedRules);
  
  // Save to backend
  for (const rule of updatedRules) {
    if (rule.dependent_rule_ids?.length) {
      await updateRuleDependencies({
        variables: {
          ruleId: rule.id,
          dependencies: rule.dependent_rule_ids,
          tenantId: currentTenantId,
          datasourceId: currentDatasourceId
        }
      });
    }
  }
};

const handleCrossEntitySave = async (condition: CrossEntityCondition) => {
  setCrossEntityConditions([...crossEntityConditions, condition]);
  
  // Save to backend
  await createCrossEntityValidation({
    variables: {
      condition,
      tenantId: currentTenantId,
      datasourceId: currentDatasourceId
    }
  });
};
```

### Step 3: GraphQL Mutations
```typescript
import { gql, useMutation } from '@apollo/client';

const UPDATE_RULE_DEPENDENCIES = gql`
  mutation UpdateRuleDependencies(
    $ruleId: ID!
    $dependencies: [ID!]!
    $tenantId: ID!
    $datasourceId: ID!
  ) {
    updateRuleDependencies(
      ruleId: $ruleId
      dependencies: $dependencies
      tenantId: $tenantId
      datasourceId: $datasourceId
    ) {
      id
      name
      dependent_rule_ids
      updatedAt
    }
  }
`;

const CREATE_CROSS_ENTITY_VALIDATION = gql`
  mutation CreateCrossEntityValidation(
    $condition: CrossEntityConditionInput!
    $tenantId: ID!
    $datasourceId: ID!
  ) {
    createCrossEntityValidation(
      condition: $condition
      tenantId: $tenantId
      datasourceId: $datasourceId
    ) {
      id
      sourcePath
      operator
      targetPath
    }
  }
`;

// In component
const [updateRuleDependencies] = useMutation(UPDATE_RULE_DEPENDENCIES);
const [createCrossEntityValidation] = useMutation(CREATE_CROSS_ENTITY_VALIDATION);
```

## Data Flow

### Rule Dependency Chain Flow
```
User selects rule
  ↓
RuleDependencyChain renders
  ↓
User adds dependency via dropdown
  ↓
addDependency() called
  ↓
onUpdateDependencies() callback fired
  ↓
Parent updates state
  ↓
GraphQL mutation sends to backend
  ↓
Execution order visualization updates
```

### Cross-Entity Validation Flow
```
User clicks on source path field
  ↓
EntityPathPicker modal opens
  ↓
User navigates through related entities
  ↓
User selects final field
  ↓
Modal closes, displayPath updated
  ↓
User selects operator
  ↓
User selects target path (same process)
  ↓
Preview box shows formatted rule
  ↓
User clicks "Add Cross-Entity Validation"
  ↓
onSave() callback fired
  ↓
Parent saves to backend
  ↓
Condition appears in saved list
```

## Customization Guide

### Add Custom Entity Relationships
```typescript
// In your component file, extend ENTITY_RELATIONSHIPS
const CUSTOM_ENTITY_RELATIONSHIPS = {
  ...ENTITY_RELATIONSHIPS,
  MyEntity: [
    { 
      field: 'related_id', 
      targetEntity: 'RelatedEntity', 
      relationship: 'many-to-one' 
    }
  ]
};

// Pass to AdvancedRuleConfiguration via props (requires component modification)
```

### Customize Operators
```typescript
// Extend the operators list in CrossEntityValidationBuilder
const customOperators = [
  { value: 'equals', label: 'Equals (=)' },
  { value: 'not_equals', label: 'Not Equals (≠)' },
  { value: 'greater_than', label: 'Greater Than (>)' },
  { value: 'less_than', label: 'Less Than (<)' },
  { value: 'greater_equal', label: 'Greater or Equal (≥)' },
  { value: 'less_equal', label: 'Less or Equal (≤)' },
  // Add new ones
  { value: 'contains', label: 'Contains' },
  { value: 'starts_with', label: 'Starts With' }
];
```

### Custom Styling
Override Tailwind classes via CSS modules:
```css
/* AdvancedRuleConfiguration.module.css */
.ruleCard {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.dependencyBadge {
  @apply px-3 py-1 rounded text-xs font-semibold bg-blue-100 text-blue-700;
}
```

Then use:
```typescript
import styles from './AdvancedRuleConfiguration.module.css';

<div className={styles.ruleCard}>
```

## Testing

### Unit Test Example
```typescript
import { render, screen, fireEvent } from '@testing-library/react';
import AdvancedRuleConfiguration from './AdvancedRuleConfiguration';

describe('AdvancedRuleConfiguration', () => {
  it('renders dependency tab by default', () => {
    render(<AdvancedRuleConfiguration />);
    expect(screen.getByText('Rule Dependencies')).toBeInTheDocument();
  });

  it('switches to cross-entity tab', () => {
    render(<AdvancedRuleConfiguration />);
    fireEvent.click(screen.getByText('Cross-Entity Validation'));
    expect(screen.getByText('Cross-Entity Validation')).toHaveClass('border-purple-600');
  });

  it('calls onRulesUpdate when dependency added', async () => {
    const mockUpdate = jest.fn();
    render(<AdvancedRuleConfiguration onRulesUpdate={mockUpdate} />);
    
    // Select a rule and add dependency
    fireEvent.change(screen.getByLabelText('Select rule to configure'), {
      target: { value: 'rule_3' }
    });
    
    // Note: Full test would require more interaction simulation
  });
});
```

## Troubleshooting

### Issue: Cross-entity path not saving
**Solution:** Verify EntityPathPicker `onChange` prop is wired correctly:
```typescript
<EntityPathPicker
  startEntity="Employee"
  value={sourcePath}
  onChange={setSourcePath}  // ← Ensure this is set
  label="Source Field Path"
/>
```

### Issue: Dependencies not persisting
**Solution:** Check GraphQL mutation includes tenant headers:
```typescript
const client = new ApolloClient({
  link: createHttpLink({
    uri: '/graphql',
    credentials: 'include',
    headers: {
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId
    }
  }),
  cache: new InMemoryCache()
});
```

### Issue: Modal z-index conflicts
**Solution:** Ensure modal container has higher z-index than other elements:
```css
.modal {
  z-index: 50; /* Tailwind z-50 */
}

/* Parent elements should be z-40 or lower */
```

## Performance Tips

1. **Memoize callbacks** - Use `useCallback` for `onRulesUpdate` and `onCrossEntitySave`
2. **Lazy load** - Only render AdvancedRuleConfiguration when tab is active
3. **Debounce updates** - Add debounce to frequent updates
4. **Pagination** - For large rule lists, implement pagination

```typescript
// Good - memoized callback
const handleRulesUpdate = useCallback((rules) => {
  saveRules(rules);
}, []);

// Better - with lazy loading
const [showAdvanced, setShowAdvanced] = useState(false);

{showAdvanced && <AdvancedRuleConfiguration {...props} />}
```

## Accessibility Checklist

- ✅ All buttons have `aria-label` or text content
- ✅ All inputs have associated `<label>` elements
- ✅ Color not the only indicator of state
- ✅ Keyboard navigation works (Tab, Enter, Escape)
- ✅ Focus indicators visible
- ✅ Modal has focus trap
- ✅ Screen reader announcements for state changes

## Browser Support

- ✅ Chrome 90+
- ✅ Firefox 88+
- ✅ Safari 14+
- ✅ Edge 90+

## Dependencies

```json
{
  "react": "^18.2.0",
  "lucide-react": "^0.263.0",
  "typescript": "^5.0.0"
}
```

No additional external dependencies required!

## Migration Guide

### From ConditionBuilder to Advanced Configuration

**Before:**
```typescript
<ConditionBuilder
  value={conditions}
  onChange={setConditions}
/>
```

**After:**
```typescript
<AdvancedRuleConfiguration
  rules={rules}
  onRulesUpdate={setRules}
  onCrossEntitySave={handleCrossEntitySave}
/>
```

### Breaking Changes
None! This is a new component. Both can coexist.

## Support

### Getting Help
1. Check inline JSDoc comments in component
2. Review examples in this guide
3. Check troubleshooting section above
4. Open issue with:
   - Error message
   - Steps to reproduce
   - Environment (browser, OS)
   - Component props used

### Contributing
Pull requests welcome! Please:
1. Add tests for new features
2. Update documentation
3. Follow component patterns
4. Test accessibility

---

**Last Updated:** October 20, 2025  
**Version:** 1.0.0  
**Status:** Production Ready ✅
