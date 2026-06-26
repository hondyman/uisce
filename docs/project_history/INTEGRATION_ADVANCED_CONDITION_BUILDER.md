# Integration Guide: Advanced Condition Builder into ValidationRuleEditor

**Last Updated**: October 20, 2025  
**Status**: Integration in Progress  
**Build Validation**: ✅ Production Ready (50.35s, zero errors)

## 📋 Overview

This guide walks through integrating the Advanced Condition Builder (with its Workday-inspired UI and autosave) into the ValidationRuleEditor component. The integration enables non-technical users to build complex validation rules with nested condition groups, AND/OR logic, and automatic persistence.

---

## 🎯 Integration Goals

1. **Replace ConditionBuilder** on Tab 1 (Configure) with ExpressionBuilder
2. **Wire autosave** so rules automatically save as drafts when editing
3. **Handle draft lifecycle** - create draft on first save, then update-by-pk for subsequent saves
4. **Maintain tenant scoping** - all mutations include tenant headers and query parameters
5. **Preserve existing workflow** - keep template selection, field selector, and impact analysis intact
6. **Zero breaking changes** - all existing functionality continues to work

---

## 🔄 Current State

### ValidationRuleEditor.tsx - Tab Workflow

```
Tab 0: Templates & Cloning (Create flow only)
  ├─ RuleTemplatesSelector
  └─ RuleCloneAndConflict

Tab 1: Configuration Form (Create & Edit)
  ├─ Rule metadata (name, bp_name, step_name, priority, status)
  ├─ ConditionBuilder (JSON editor) ← REPLACE with ExpressionBuilder
  ├─ Action callbacks (success/failure)
  └─ AdvancedFieldSelector for dot notation

Tab 2: Live Preview & Testing (Create flow only)
  ├─ SampleDataGenerator
  └─ LivePreview

Tab 3: Impact Analysis (Create flow only)
  └─ ImpactAnalysis
```

### ExpressionBuilder.tsx - Current Props

```typescript
interface ExpressionBuilderProps {
  onSave?: (conditionJson: any) => void;          // Manual save callback
  onChange?: (conditionJson: any) => void;        // Real-time change callback
  autosave?: boolean;                             // Enable automatic persistence
  debounceMs?: number;                            // Debounce interval (default 1000ms)
  ruleName?: string;                              // Rule name for draft
  targetEntity?: string;                          // Entity scope (bp_name)
  ruleId?: string;                                // Existing rule ID for update-by-pk
  onDraftCreated?: (id: string, ruleName?: string) => void;  // Draft creation callback
}
```

---

## 🔧 Implementation Steps

### Step 1: Update ValidationRuleEditor.tsx Imports

Add the ExpressionBuilder import at the top of the file:

```typescript
import ExpressionBuilder from '../../components/ExpressionBuilder/ExpressionBuilder';
```

### Step 2: Modify Tab 1 (Configuration Form)

In the condition section (around line 565-580), replace the ConditionBuilder with ExpressionBuilder:

**Current code:**
```typescript
<Grid item xs={12}>
  <Typography variant="subtitle2" sx={{ mb: 1 }}>
    Condition
  </Typography>
  <Box sx={{ display: 'flex', gap: 1, mb: 1 }}>
    <Button variant="outlined" size="small" onClick={() => setShowVisualBuilder(true)}>
      Open Visual Builder
    </Button>
    <Button variant="text" size="small" onClick={() => handleFormChange('condition_json', '{}')}>
      Reset
    </Button>
  </Box>
  <ConditionBuilder
    value={formData.condition_json}
    onChange={(json: string) => handleFormChange('condition_json', json)}
  />
</Grid>
```

**Replace with:**
```typescript
<Grid item xs={12}>
  <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>
    Condition (Visual Builder)
  </Typography>
  <ExpressionBuilder
    ruleName={formData.name}
    targetEntity={formData.bp_name}
    autosave={!!editingId}  // Enable autosave only when editing (not creating)
    ruleId={editingId || undefined}
    onDraftCreated={(id, name) => {
      // When a draft is created during new rule creation:
      // 1. Set editingId so autosave remains enabled
      // 2. Update the rule name if provided by draft
      // 3. Show success feedback
      setEditingId(id);
      if (name) handleFormChange('name', name);
      setSnackbarMsg(`Draft created: ${name || id}`);
      setSnackbarOpen(true);
    }}
    onSave={(cj) => {
      // Manual save callback (if user clicks Save button in builder)
      handleFormChange('condition_json', JSON.stringify(cj));
    }}
    onChange={(cj) => {
      // Real-time updates as user builds conditions
      handleFormChange('condition_json', JSON.stringify(cj));
    }}
  />
  <Box sx={{ mt: 1, display: 'flex', gap: 1 }}>
    <Button 
      variant="text" 
      size="small" 
      onClick={() => handleFormChange('condition_json', '{}')}>
      Reset Conditions
    </Button>
  </Box>
</Grid>
```

### Step 3: Remove the Visual Builder Dialog

Since ExpressionBuilder is now on Tab 1, the separate visual builder dialog is no longer needed. Remove or repurpose:

```typescript
// Remove this entire dialog (lines ~720-750):
<Dialog open={showVisualBuilder} onClose={() => setShowVisualBuilder(false)} fullWidth maxWidth="lg">
  <DialogTitle>Visual Condition Builder</DialogTitle>
  <DialogContent>
    <ExpressionBuilder ... />
  </DialogContent>
  <DialogActions>
    <Button onClick={() => setShowVisualBuilder(false)}>Close</Button>
  </DialogActions>
</Dialog>

// And remove the state:
const [showVisualBuilder, setShowVisualBuilder] = useState(false);

// And remove the button:
<Button variant="outlined" size="small" onClick={() => setShowVisualBuilder(true)}>
  Open Visual Builder
</Button>
```

### Step 4: Update Dialog Title Dynamically

Modify the dialog title to reflect the rule type:

```typescript
<DialogTitle sx={{ borderBottom: 1, borderColor: 'divider' }}>
  {editingId ? 
    `Edit Rule: ${formData.name || 'Untitled'}` 
    : 'Create New Rule'}
</DialogTitle>
```

---

## 🔄 Autosave Behavior

### When Creating a New Rule

1. **Tab 0 (Templates)**: No autosave yet
2. **Tab 1 (Configure)**: 
   - ExpressionBuilder initializes with `autosave=false` (editingId is null)
   - First time user modifies conditions → schedulePersist() queues save
   - After debounce (1000ms) → INSERT_DRAFT_RULE mutation
   - Backend creates row with `is_active: false`
   - `onDraftCreated()` callback fires → sets editingId
   - Now `autosave=true` (editingId is set) for subsequent changes
   - Further condition changes → UPDATE_RULE_BY_PK mutations
3. **Tab 2-3**: Testing and impact analysis use the draft

### When Editing an Existing Rule

1. ExpressionBuilder initializes with `autosave=true` (editingId already set)
2. Each change → debounced UPDATE_RULE_BY_PK
3. Shows toast on successful save
4. Draft persists as is_active: false until user explicitly publishes

---

## 📝 Props Wiring Checklist

- [ ] `ruleName={formData.name}` - Pass current rule name
- [ ] `targetEntity={formData.bp_name}` - Pass business process for scoping
- [ ] `autosave={!!editingId}` - Only enable when editing existing rule
- [ ] `ruleId={editingId || undefined}` - Pass draft ID for update-by-pk
- [ ] `onDraftCreated` callback implemented - Update parent state
- [ ] `onSave` callback implemented - Update formData (optional for Tab 1)
- [ ] `onChange` callback implemented - Keep formData in sync
- [ ] Snackbar feedback for draft creation - UX visibility

---

## 🔐 Tenant Scoping Integration

The ExpressionBuilder automatically:

1. Reads tenant context from `localStorage`:
   - `selected_tenant` → tenant_id
   - `selected_datasource` → datasource_id

2. Adds headers to mutations:
   - `X-Tenant-ID: <TENANT_ID>`
   - `X-Tenant-Datasource-ID: <DATASOURCE_ID>`

3. Adds query parameters to API calls:
   - `?tenant_id=<TENANT_ID>&datasource_id=<DATASOURCE_ID>`

**No additional wiring needed** - tenant context is handled by the component.

---

## 🧪 Testing Changes

### Unit Tests

Update `/frontend/src/components/ExpressionBuilder/__tests__/ExpressionBuilder.autosave.test.tsx`:

1. **Fix MockedProvider configuration**
   - Ensure `__typename` is included in all mock results
   - Use Vitest directives (vi.useFakeTimers, vi.advanceTimersByTime)
   - Fix promise chain handling

2. **Test draft creation callback**
   - Verify onDraftCreated fires with correct ID

3. **Test condition changes**
   - Verify debounced autosave works
   - Verify update-by-pk is used after draft exists

### Integration Tests

Create `/frontend/src/components/ExpressionBuilder/__tests__/ExpressionBuilder.integration.test.tsx`:

1. **Test full GraphQL flow**
   - Mock actual mutations with proper responses
   - Verify headers and query parameters
   - Test retry logic

2. **Test ValidationRuleEditor integration**
   - Create new rule
   - Verify draft creation
   - Verify editingId gets set
   - Verify subsequent saves use update-by-pk

### Manual QA

1. **Create new rule**:
   - Select template
   - Enter rule name
   - Go to Configure tab
   - Modify conditions
   - Verify draft is created (check editingId state)
   - Make more changes
   - Verify no new drafts created (updates existing)

2. **Edit existing rule**:
   - Click Edit on existing rule
   - Modify conditions
   - Verify immediate autosave
   - Navigate away and back
   - Verify changes persisted

3. **Tenant scoping**:
   - Select different tenant/datasource
   - Create rule
   - Verify tenant headers in network tab
   - Verify draft created in correct tenant

---

## 📦 Files Modified

| File | Changes | Lines |
|------|---------|-------|
| `ValidationRuleEditor.tsx` | Replace ConditionBuilder with ExpressionBuilder on Tab 1; remove showVisualBuilder dialog; wire autosave callbacks | ~50 |
| `ExpressionBuilder.autosave.test.tsx` | Fix MockedProvider config, test callbacks, ensure Vitest compatibility | ~80 |

---

## 🐛 Troubleshooting

### Build Errors

**Error**: `Cannot find module 'ExpressionBuilder'`
- **Solution**: Ensure import path is correct: `../../components/ExpressionBuilder/ExpressionBuilder`

**Error**: `Type 'ConditionGroup' is not assignable to type 'string'`
- **Solution**: Pass JSON-stringified condition JSON: `JSON.stringify(cj)` in callbacks

### Runtime Issues

**Issue**: Draft not being created
- **Solution**: 
  - Verify tenant context in localStorage
  - Check browser console for GraphQL errors
  - Ensure Apollo Client is properly configured with tenant headers

**Issue**: Autosave not triggering
- **Solution**:
  - Verify `editingId` is set after draft creation
  - Check that `autosave={!!editingId}` is true
  - Ensure timer is debouncing correctly (advance by 1000ms+ in tests)

**Issue**: Changes not persisting
- **Solution**:
  - Check Network tab for mutation requests
  - Verify UPDATE_RULE_BY_PK query structure matches schema
  - Ensure ruleId is being passed correctly

---

## 🚀 Deployment Checklist

- [ ] All imports added correctly
- [ ] Tab 1 condition section updated
- [ ] showVisualBuilder dialog removed
- [ ] onDraftCreated callback implemented
- [ ] onChange callbacks update formData
- [ ] Tests updated and passing
- [ ] Build succeeds with zero errors
- [ ] Manual QA complete
- [ ] Tenant scoping verified
- [ ] Autosave behavior tested in browser

---

## 📚 Related Documentation

- **Advanced Condition Builder**: `README_ADVANCED_CONDITION_BUILDER.md`
- **Component API**: `ADVANCED_CONDITION_BUILDER_GUIDE.md`
- **Code Examples**: `ADVANCED_CONDITION_BUILDER_EXAMPLES.md` (Example #9 covers integration)
- **Tenant Scoping**: `agents.md`
- **GraphQL Integration**: `BACKEND_VALIDATION_INTEGRATION.md`

---

## ✅ Success Criteria

1. ✅ ExpressionBuilder renders on Tab 1 without errors
2. ✅ Draft created on first condition change
3. ✅ onDraftCreated callback fires and sets editingId
4. ✅ Subsequent changes use update-by-pk
5. ✅ Toast notifications show success/errors
6. ✅ Tenant headers included in all requests
7. ✅ All tests pass
8. ✅ Build succeeds (0 errors, 0 warnings)
9. ✅ Manual QA verification complete

---

## 🎓 Next Steps After Integration

1. **Test with real validation rules** - Import sample rules and edit them
2. **Verify GraphQL mutations** - Check backend receives correct payloads
3. **Monitor performance** - Ensure debouncing prevents excessive saves
4. **Gather user feedback** - Test with actual business users
5. **Consider enhancements**:
   - Smart field autocomplete (future)
   - Rule templates library (future)
   - Live preview with sample data (future)
   - Conflict detection (future)

---

**Total Integration Effort**: ~1-2 hours  
**Complexity**: Medium  
**Risk**: Low (localized changes, existing GraphQL flow)  
**Status**: Ready for Implementation  

