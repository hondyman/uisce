# UI Library Standardization & Antd Removal Plan

**Status:** In Progress  
**Target:** Consolidate on Material-UI (MUI) + Tailwind CSS, remove antd and other redundant libraries

---

## Executive Summary

This document outlines the complete strategy for removing **antd** (Ant Design) and **@ant-design/icons** from the Semlayer project, while standardizing on:

- **Material-UI (MUI)**: Primary component library for complex UI patterns
- **Tailwind CSS**: Utility-first styling layer (already present)
- **Lucide React**: Icon library (already widely used)
- **Radix UI**: Maintained only for primitives where MUI doesn't provide equivalents

### Current State Audit

**Current Dependencies:**
- ✅ `@mui/material` ^5.18.0 (core)
- ✅ `@mui/icons-material` ^5.18.0 (icons)
- ✅ `@mui/x-data-grid` ^7.8.0 (tables)
- ✅ `@mui/x-date-pickers` ^8.11.1 (date input)
- ✅ `@mui/x-tree-view` ^7.8.0 (trees)
- ✅ `tailwindcss` ^4.1.11 (styling)
- ✅ `lucide-react` ^0.540.0 (icons)
- ✅ `@radix-ui/*` (primitives for custom components)
- ✅ `@mantine/core` ^8.2.4 (used in tests only)
- ❌ `antd` ^5.27.5 (TO REMOVE)
- ❌ `@ant-design/icons` ^5.3.7 (TO REMOVE)
- ⚠️ `@fortawesome/react-fontawesome` (optional, can use lucide instead)

---

## Component Mapping: Antd → MUI

| Antd Component | MUI Alternative | Status |
|---|---|---|
| **Layout** | | |
| Layout, Layout.Header, Layout.Content | Drawer + Box (custom) | Phase 2 |
| Space | Stack, Box (gap prop) | Phase 2 |
| Row, Col | Grid, Stack | Phase 2 |
| Divider | Divider | Phase 2 |
| **Forms** | | |
| Form | FormProvider (react-hook-form) | Phase 2 |
| Form.Item | FormControlLabel, FormHelperText | Phase 2 |
| Input | TextField | Phase 2 |
| InputNumber | TextField (type="number") | Phase 2 |
| Select | Select, MenuItem | Phase 2 |
| DatePicker | DatePicker (MUI X Date Pickers) | Phase 2 |
| Switch | Switch | Phase 2 |
| Checkbox | Checkbox, FormControlLabel | Phase 2 |
| Radio | Radio, FormControlLabel | Phase 2 |
| **Data Display** | | |
| Table | DataGrid (MUI X) | Phase 3 |
| List | List, ListItem | Phase 3 |
| Card | Card, CardContent | Phase 2 |
| Tag | Chip | Phase 2 |
| Badge | Badge | Phase 2 |
| Tree | TreeView (MUI X Tree View) | Phase 3 |
| **Overlay** | | |
| Modal | Dialog | Phase 3 |
| Drawer | Drawer | Phase 3 |
| Popover | Popover | Phase 3 |
| Tooltip | Tooltip | Phase 3 |
| Popconfirm | Dialog (confirm pattern) | Phase 3 |
| Message (global) | Snackbar + useSnackbar | Phase 3 |
| Notification | Snackbar + useSnackbar | Phase 3 |
| **Navigation** | | |
| Menu | Menu, MenuItem | Phase 2 |
| Tabs | Tabs | Phase 2 |
| Breadcrumb | Breadcrumbs (MUI or custom) | Phase 2 |
| **Buttons & Actions** | | |
| Button | Button | Phase 1 |
| Dropdown | Menu | Phase 2 |
| **Icons** | | |
| @ant-design/icons | @mui/icons-material + lucide-react | Phase 1 |

---

## Files Currently Using Antd (46 files)

### High Priority (Core UI)
1. `frontend/src/components/BPTriggerBuilder.tsx` - Card, Select, Form, InputNumber, Switch, Button, Timeline, Input
2. `frontend/src/components/EntityEditDetailModal.tsx` - Modal, Form, Tree
3. `frontend/src/components/EntityDrawerTreeView.tsx` - Drawer, Tree, Button, Select
4. `frontend/src/components/bp-designer/TriggerBuilder.tsx` - Modal, Form, Select, Input, Button, Table, Space, Tag, Tooltip, Popconfirm, Card
5. `frontend/src/components/abac/PolicyBuilder.tsx` - Form, Input, Select, Button, Table, Space, Card, Tag

### Medium Priority (Features)
6. `frontend/src/components/pop/CalendarModeToggle.tsx` - Select, Card, Space, Typography, Tooltip
7. `frontend/src/components/pop/StewardUnionReview.tsx` - Card, List, Button, Space, Tag, Typography, Modal, Descriptions, Tooltip
8. `frontend/src/components/pop/CohortFilterSelector.tsx` - Select, Card, Tag, Space, message
9. `frontend/src/components/pop/LineageVisualizer.tsx` - Card, Spin, Space, Tooltip, message
10. `frontend/src/components/pop/StewardGranularityReview.tsx` - Card, List, Button, Space, Tag, Typography, Modal, Descriptions, Tooltip
11. `frontend/src/components/abac/AuditLogViewer.tsx` - Table, Tag, Space, Select
12. `frontend/src/components/abac/DelegationManager.tsx` - Table, Space, Modal, DatePicker, Select, message
13. `frontend/src/components/relationship/RelationshipDiscoveryModal.tsx` - Modal, Spin, Empty, message, Tabs, Badge, Tooltip
14. `frontend/src/components/relationship/ReportBuilder.tsx` - Form, Input, Select, Button, Card, message, Table, Space
15. `frontend/src/components/relationship/RelationshipPathVisualizer.tsx` - Card, Tooltip, Badge, Space
16. `frontend/src/components/AIRouting/AIRoutingDashboard.tsx` - Form, Select, Button, Card, Space, message

### Lower Priority (Pages/Legacy)
17. `frontend/src/pages/WorkflowTimeoutTriggersPage.tsx`
18. `frontend/src/pages/EntityConfigPageV2.tsx`
19. `frontend/src/pages/admin/RelatedObjectsPage.tsx`
20. `frontend/src/pages/EntityConfigPageV3.tsx`
21. `frontend/src/pages/timeouts/WorkflowTimeoutTriggersPage.tsx`
22. `frontend/src/pages/EntityConfigPage.tsx` (type import only)
23. `frontend/src/components/ExpressionBuilder/OperatorSelector.tsx` (Select only)
24. `frontend/src/components/ExpressionBuilder/ValueInput.tsx` (Input, InputNumber)
25. `frontend/src/components/ExpressionBuilder/DroppableCondition.tsx` (Select)
26. `frontend/src/components/ExpressionBuilder/ExpressionBuilder.tsx` (Card, Typography, message)
27. `frontend/local/src/pages/UnifiedCRUDPage.tsx` (Button only)
28. `frontend/scripts/AISuggestedRelationships.tsx`

---

## Migration Phases

### Phase 1: Preparation (Week 1)
- [ ] Update `package.json`: Remove antd & @ant-design/icons
- [ ] Install missing MUI dependencies (if any)
- [ ] Create MUI theme override (if needed)
- [ ] Set up global Snackbar provider for notifications
- [ ] Create migration utilities (icon mapping, etc.)

### Phase 2: Core Components & Forms (Week 2-3)
- [ ] Migrate `BPTriggerBuilder.tsx`
- [ ] Migrate `EntityEditDetailModal.tsx`
- [ ] Migrate `EntityDrawerTreeView.tsx`
- [ ] Migrate all ExpressionBuilder components
- [ ] Migrate form-related components in ABAC & relationship modules

### Phase 3: Data Display & Overlays (Week 4-5)
- [ ] Migrate Tables → MUI DataGrid
- [ ] Migrate Modals → MUI Dialog
- [ ] Migrate Drawers
- [ ] Migrate Popovers & Tooltips
- [ ] Migrate Trees → MUI TreeView

### Phase 4: Feature Modules (Week 5-6)
- [ ] Migrate POP (Power of a Portfolio) components
- [ ] Migrate ABAC components
- [ ] Migrate Relationship components
- [ ] Migrate AIRouting components

### Phase 5: Pages & Legacy (Week 6-7)
- [ ] Migrate entity config pages
- [ ] Migrate workflow pages
- [ ] Remove all antd imports from tests
- [ ] Update Mantine usage (only for test fixtures)

### Phase 6: Cleanup & Testing (Week 7-8)
- [ ] Remove antd from package.json
- [ ] Remove @ant-design/icons from package.json
- [ ] Run full test suite
- [ ] Visual regression testing
- [ ] Performance benchmarking

---

## Key Implementation Decisions

### 1. Message/Notification System
**Current:** `message` from antd  
**New:** Use MUI `Snackbar` with a custom hook wrapper

```typescript
// hooks/useNotification.ts
import { useSnackbar } from 'notistack';

export const useNotification = () => {
  const { enqueueSnackbar } = useSnackbar();
  
  return {
    success: (msg: string) => enqueueSnackbar(msg, { variant: 'success' }),
    error: (msg: string) => enqueueSnackbar(msg, { variant: 'error' }),
    info: (msg: string) => enqueueSnackbar(msg, { variant: 'info' }),
    warning: (msg: string) => enqueueSnackbar(msg, { variant: 'warning' }),
  };
};
```

### 2. Icon Mapping Strategy
Replace @ant-design/icons with lucide-react + @mui/icons-material

| Antd Icon | Replacement |
|---|---|
| PlusOutlined | Plus (lucide) or AddIcon (@mui/icons-material) |
| DeleteOutlined | Trash2 (lucide) or DeleteIcon (@mui/icons-material) |
| EditOutlined | Edit (lucide) or EditIcon (@mui/icons-material) |
| SearchOutlined | Search (lucide) or SearchIcon (@mui/icons-material) |
| etc. | See icon mapping document |

### 3. Global Message Provider
Wrap app in MUI's SnackbarProvider (already present: notistack) or use MUI's built-in notification system.

### 4. Tailwind + MUI Coexistence
- Use MUI components for interactive elements (forms, modals, tables, etc.)
- Use Tailwind for layout, spacing, and simple styling
- Override MUI theme colors with Tailwind palette if needed

---

## Testing Strategy

1. **Unit Tests:** Update component tests to mock MUI instead of antd
2. **Integration Tests:** Test page workflows with new components
3. **E2E Tests:** Verify forms, modals, and tables work correctly
4. **Visual Regression:** Screenshot tests before/after
5. **Performance:** Measure bundle size reduction

---

## Rollback Plan

If issues arise:
1. Commit on a feature branch before major changes
2. Keep original antd files in a `deprecated/` folder
3. Tag version before removal: `v*.*.* (antd-present)`
4. Tag version after removal: `v*.*.* (antd-removed)`

---

## Bundle Size Expectations

**Before:** ~500KB (antd bundle)  
**After:** Potentially -200KB to -300KB (depends on tree-shaking)

MUI is already included, so the net savings come primarily from removing antd's substantial size.

---

## Documentation Updates

After migration, update:
- [ ] Component library documentation
- [ ] Developer setup guide
- [ ] Design system docs
- [ ] Icon usage guide

---

## Contingencies

- **If MUI DataGrid too heavyweight:** Use simpler MUI Table + custom pagination
- **If Tree component missing features:** Use MUI TreeView or Radix Tree
- **If Form patterns complex:** Keep react-hook-form integration, but with MUI fields
- **If date picker features insufficient:** Keep @mui/x-date-pickers or use smaller alternative

---

## Sign-off Checklist

- [ ] All 46 antd files migrated
- [ ] No antd imports remain in codebase
- [ ] No @ant-design/icons imports remain
- [ ] All tests passing
- [ ] Zero console warnings about antd
- [ ] Bundle size verified
- [ ] Performance acceptable
- [ ] Cross-browser testing complete
- [ ] Accessibility (a11y) verified
- [ ] Design review approved
