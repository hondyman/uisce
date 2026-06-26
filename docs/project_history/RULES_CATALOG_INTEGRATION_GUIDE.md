/**
 * RULES_CATALOG_INTEGRATION_GUIDE.md
 * 
 * Complete integration guide for the Rules Catalog feature
 * Shows how to integrate RulesCatalog into the existing bundle workflow
 */

# Rules Catalog Integration Guide

## Overview

The Rules Catalog is a new feature that allows users to:
- **Browse** all available validation rules in an organized catalog
- **Search & Filter** by category (ESG, Private Capital, Mutual Funds, Funds Accounting, Risk, Compliance, etc.)
- **Discover Rules** by severity, frequency, evaluation order
- **Add Rules** directly to the rules builder
- **Compare** multiple rules side-by-side
- **Save Favorites** for quick access

## Architecture

### Components

1. **RulesCatalog.tsx** - Main component
   - Location: `frontend/src/pages/bundles/RulesCatalog.tsx`
   - Lines: 674 (fully implemented)
   - Features: Grid/List/Compare views, search, filtering, multi-select

2. **RulesCatalog.module.css** - Styling
   - Location: `frontend/src/pages/bundles/RulesCatalog.module.css`
   - Fully responsive design (desktop, tablet, mobile)
   - CSS Grid, Flexbox layout
   - 900+ lines of production-ready styles

3. **Data Source** - wealthValidationRules.ts
   - Location: `frontend/src/data/wealthValidationRules.ts`
   - Contains all 30 validation rules (core + advanced)
   - Each rule includes: id, name, description, severity, frequency, evaluationOrder, rule_type, isCore

4. **Parameter Registry** - ValidationRuleParametersRegistry.ts
   - Location: `frontend/src/data/ValidationRuleParametersRegistry.ts`
   - Maps rule names to parameter configurations
   - Used for dynamic parameter form generation

## Integration Points

### Option 1: Add as Tab in BundleListPage (Recommended)

```tsx
// frontend/src/pages/bundles/BundleListPage.tsx

import RulesCatalog from './RulesCatalog';

const BundleListPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'bundles' | 'rules'>('bundles');

  return (
    <Container maxWidth="lg">
      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Stack direction="row" spacing={2}>
          <Button
            variant={activeTab === 'bundles' ? 'contained' : 'outlined'}
            onClick={() => setActiveTab('bundles')}
          >
            Bundles
          </Button>
          <Button
            variant={activeTab === 'rules' ? 'contained' : 'outlined'}
            onClick={() => setActiveTab('rules')}
          >
            Rules Catalog
          </Button>
        </Stack>
      </Box>

      {activeTab === 'bundles' && (
        // Existing bundles list content
      )}

      {activeTab === 'rules' && (
        <RulesCatalog />
      )}
    </Container>
  );
};
```

### Option 2: Add as Separate Route

```tsx
// frontend/src/routes/index.tsx

import RulesCatalog from '../pages/bundles/RulesCatalog';

const routes = [
  {
    path: '/bundles',
    element: <BundleListPage />,
    label: 'Bundles'
  },
  {
    path: '/rules-catalog',
    element: <RulesCatalog />,
    label: 'Rules Catalog'
  }
];
```

### Option 3: Add as Drawer/Modal from Bundle Editor

```tsx
// frontend/src/pages/bundles/BundleEditor.tsx

import RulesCatalog from './RulesCatalog';

const BundleEditor: React.FC = () => {
  const [showRulesCatalog, setShowRulesCatalog] = useState(false);

  return (
    <Box>
      <Button onClick={() => setShowRulesCatalog(true)}>
        Browse Rules Catalog
      </Button>

      <Modal
        open={showRulesCatalog}
        onClose={() => setShowRulesCatalog(false)}
      >
        <Box sx={{ width: '95vw', height: '95vh' }}>
          <RulesCatalog onAddRules={handleAddSelectedRules} />
        </Box>
      </Modal>
    </Box>
  );
};
```

## Data Model

### Rule Categories (10 Total)

```typescript
interface RuleCategory {
  id: string;           // Unique identifier
  name: string;         // Display name
  description: string;  // User-friendly description
  icon: string;         // Emoji icon
  color: string;        // Hex color for styling
  ruleIds: string[];    // Array of rule IDs in category
}
```

**Categories:**
1. **esg** - ESG & Sustainability
2. **private-capital** - Private Capital
3. **mutual-funds** - Mutual Funds
4. **funds-accounting** - Funds Accounting
5. **risk-management** - Risk Management
6. **compliance** - Compliance & Regulatory
7. **access-control** - Access & Permissions
8. **client-experience** - Client Experience
9. **trade-execution** - Trade & Settlement
10. **data-integrity** - Data Integrity

### Filtering Options

```typescript
interface FilterOptions {
  search: string;                    // Free-text search
  categories: string[];              // Category IDs
  severities: string[];              // BLOCK, WARNING, INFO
  frequencies: string[];             // ON_TRADE, DAILY, MONTHLY, etc.
  ruleTypes: string[];               // CONDITION, ACTION, etc.
  isCore?: boolean;                  // Filter by core/advanced
  sortBy: 'evaluationOrder' | 'name' | 'severity';
}
```

### View Modes

- **Grid View** (Default)
  - Card-based layout
  - Shows rule name, description, severity, categories, metadata
  - Select checkbox overlay
  - Save to favorites button

- **List View**
  - Row-based layout
  - Compact display
  - Easier to scan many rules
  - Checkbox selection

- **Compare View**
  - Side-by-side comparison
  - Only available with 2+ rules selected
  - Shows all rule properties

## Feature Implementation

### Search & Filter

The component uses `useMemo` to efficiently filter and sort rules:

```tsx
// Supports filtering by:
// - Rule name and description
// - Category names
// - Severity level (BLOCK, WARNING, INFO)
// - Evaluation frequency
// - Rule type
// - Core vs. advanced rules

// Supports sorting by:
// - Evaluation order (default, respects execution sequence)
// - Alphabetical (A-Z)
// - Severity (BLOCK → WARNING → INFO)
```

### Multi-Select & Actions

```tsx
// Select rules
const [selectedRules, setSelectedRules] = useState<string[]>([]);

// User can:
// - Click cards to select/deselect
// - Use checkboxes in list view
// - Compare 2+ selected rules

// Actions:
// - Add selected rules to builder
// - Clear all filters
// - Save favorite rules
```

### Saved Rules/Favorites

```tsx
// Persist using localStorage or backend
const [savedRules, setSavedRules] = useState<string[]>([]);

// User can star/unstar rules
// Later: Add "Saved Rules" filter for quick access
```

## Integration with ValidationRuleCreator

When user adds rules to the builder from the catalog:

```tsx
// RulesCatalog.tsx - callback function
const handleAddSelectedToBuilder = useCallback(() => {
  // Get selected rule objects
  const rulesToAdd = filteredRules.filter(item =>
    selectedRules.includes(item.rule.id)
  );

  // Pass to parent component or context
  onAddRules?.(rulesToAdd);

  // Alternative: Use context
  // const { addRulesToBuilder } = useRulesBuilder();
  // addRulesToBuilder(rulesToAdd);
}, [selectedRules, filteredRules]);
```

Update ValidationRuleCreator:

```tsx
// frontend/src/pages/bundles/ValidationRuleCreator.tsx

const handleAddFromCatalog = (rules: typeof WEALTH_VALIDATION_RULES) => {
  rules.forEach(rule => {
    const newRuleConfig = {
      id: generateId(),
      rule_name: rule.name,
      parameters: [],
      // ... other default values
    };

    setRules(prev => [...prev, newRuleConfig]);
  });
};
```

## Styling & Theming

### CSS Modules Approach

The component uses `RulesCatalog.module.css` for:
- Complete separation of concerns
- No style conflicts with existing code
- Easy to override via design system variables
- Responsive design (mobile-first)

### Color System

```css
/* Severity Colors */
BLOCK:   #EF4444 (Red)
WARNING: #F59E0B (Amber)
INFO:    #3B82F6 (Blue)

/* Category Colors */
ESG:              #10B981 (Emerald)
Private Capital:  #8B5CF6 (Purple)
Mutual Funds:     #3B82F6 (Blue)
Funds Accounting: #F59E0B (Amber)
Risk Management:  #EF4444 (Red)
Compliance:       #059669 (Teal)
Access Control:   #DC2626 (Rose)
Client Experience: #06B6D4 (Cyan)
Trade Execution:  #7C3AED (Violet)
Data Integrity:   #16A34A (Green)

/* Interaction Colors */
Primary:   #3B82F6
Hover:     #2563eb
Success:   #10B981
```

### Responsive Breakpoints

```css
/* Large Desktop: 1024px+ */
- Sidebar always visible
- Multi-column grid
- Full feature set

/* Tablet: 768px - 1023px */
- Sidebar collapses to horizontal filter bar
- 2-column grid
- Adaptive layout

/* Mobile: < 768px */
- Full-width single column
- Collapsed filters in accordion
- Touch-friendly buttons (44px minimum)
```

## Performance Considerations

### Optimization Strategies

1. **useMemo for filtering**
   - Prevents unnecessary re-renders
   - Only recalculates when filters change

2. **useCallback for handlers**
   - Stable function references
   - Prevents child re-renders

3. **Lazy loading (Future)**
   - Virtual scrolling for large rule sets
   - Pagination support

4. **Caching (Future)**
   - Cache filtered results
   - Store favorite rules in localStorage

## Accessibility (WCAG 2.1 AA)

✅ **Implemented:**
- Semantic HTML (buttons, inputs, labels)
- ARIA labels on interactive elements
- Keyboard navigation (Tab, Enter, Space)
- Color not the only means of communication
- Sufficient color contrast (4.5:1 for text)
- Focus indicators visible
- Form labels associated with inputs

✅ **What's included:**
- `aria-label` on view mode buttons
- `aria-pressed` on selected cards
- `title` attributes for tooltips
- Semantic form inputs
- Proper heading hierarchy

## Error Handling

```tsx
// Handle missing data
if (!WEALTH_VALIDATION_RULES || WEALTH_VALIDATION_RULES.length === 0) {
  return <EmptyState />;
}

// Handle filter errors
try {
  const filtered = filteredRules;
  if (filtered.length === 0) {
    return <NoResultsFound />;
  }
} catch (error) {
  console.error('Filter error:', error);
  return <ErrorState />;
}
```

## Testing Checklist

### Unit Tests

- [ ] Filter functions work correctly
- [ ] Sort functions maintain rule order
- [ ] Multi-select state management
- [ ] Favorite toggle logic
- [ ] Category mapping

### Integration Tests

- [ ] Rules load from data source
- [ ] Search/filter combinations work
- [ ] Add to builder callback fires
- [ ] Selected rules persist during sorting/filtering
- [ ] View mode switching preserves filters

### E2E Tests

- [ ] User can search for rules
- [ ] User can filter by multiple categories
- [ ] User can select and add rules to builder
- [ ] User can compare rules
- [ ] User can save/unsave favorites
- [ ] Responsive behavior on mobile

### Accessibility Tests

- [ ] Keyboard navigation works
- [ ] Screen reader announces filters
- [ ] Color contrast meets standards
- [ ] Focus indicators visible
- [ ] Form elements have labels

## Deployment Checklist

- [ ] RulesCatalog.tsx created and exported
- [ ] RulesCatalog.module.css created
- [ ] Imported in BundleListPage.tsx (or chosen route)
- [ ] Tested with all 30 validation rules
- [ ] Filter combinations verified
- [ ] Responsive design tested on mobile/tablet
- [ ] ESLint/TypeScript errors resolved
- [ ] Accessibility audit passed
- [ ] Performance acceptable (< 100ms render)
- [ ] Documented for team

## Future Enhancements

### Phase 2

- [ ] **Saved Rules Feature**
  - Persist favorites to backend
  - Access from separate "My Rules" view
  - Quick-add from favorites

- [ ] **Rule Templates**
  - Pre-configured rule groups
  - One-click add entire templates
  - Custom templates by user

- [ ] **Rule Import/Export**
  - Export selected rules as JSON
  - Import rules from other bundles
  - Duplicate existing rules

### Phase 3

- [ ] **Advanced Search**
  - Full-text search in descriptions and parameters
  - Suggested searches based on history

- [ ] **Rules Analytics**
  - Most used rules
  - Recently added rules
  - Rule usage statistics

- [ ] **Integration with Backend**
  - Rules sync from backend catalog
  - Rule version history
  - Rule versioning support

- [ ] **Custom Rules**
  - User-defined custom rules
  - Save and reuse custom rules
  - Share rules with team

## Troubleshooting

### Issue: Rules not showing

**Solution:**
- Verify wealthValidationRules.ts is imported correctly
- Check that rule IDs in RULE_CATEGORIES match actual rules
- Verify WEALTH_VALIDATION_RULES array is not empty

### Issue: Filters not working

**Solution:**
- Check FilterOptions interface matches implementation
- Verify filter comparison logic (strict equality)
- Check that rule properties exist on all rules

### Issue: Styles not applying

**Solution:**
- Verify CSS module is imported correctly
- Check that className values match CSS module keys
- Run TypeScript compiler to catch module errors

### Issue: Performance degradation

**Solution:**
- Check that useMemo dependencies are correct
- Verify no unnecessary re-renders with React DevTools
- Consider virtual scrolling for large rule sets

## Support & Documentation

- **Component Code**: `frontend/src/pages/bundles/RulesCatalog.tsx`
- **Styles**: `frontend/src/pages/bundles/RulesCatalog.module.css`
- **Data Source**: `frontend/src/data/wealthValidationRules.ts`
- **Parameter Registry**: `frontend/src/data/ValidationRuleParametersRegistry.ts`

For questions or issues, refer to:
1. ADVANCED_WEALTH_RULES_DOCUMENTATION_INDEX.md
2. ADVANCED_WEALTH_RULES_IMPLEMENTATION_GUIDE.md
3. ValidationRuleCreator.tsx (example of rule builder integration)
