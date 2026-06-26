# AntD Removal - Work Summary

## Current Status

**Files with AntD imports: 23 active files** (excluding docs, backups, tests)

## ✅ Completed
- **EntityDetailsPage.tsx** - Fully removed AntD, using Tailwind + lucide-react

## 🎯 Priority Queue (Recommended Order)

### Tier 1: Quick Wins (1-2 hours total)
Minimal complexity, high impact on codebase cleanliness

- [ ] **pop/CalendarModeToggle.tsx** - Select, Card, Space, Typography, Tooltip (4 components)
- [ ] **pop/CohortFilterSelector.tsx** - Select, Card, Tag, Space, message (5 components)
- [ ] **ExpressionBuilder/OperatorSelector.tsx** - Just Select (1 component)
- [ ] **ExpressionBuilder/ValueInput.tsx** - Input, InputNumber (2 components)
- [ ] **ExpressionBuilder/DroppableCondition.tsx** - Button, Select (2 components)

### Tier 2: Medium Complexity (3-5 hours total)
Standard components, manageable scope

- [ ] **ExpressionBuilder/ExpressionBuilder.tsx** - Button, Card, Typography, message (4 components)
- [ ] **AIRouting/AIRoutingDashboard.tsx** - Card, Button, Tabs, Input, Select, Form, message, Popconfirm, Modal (9 components)
- [ ] **pop/LineageVisualizer.tsx** - Card, Spin, Button, Space, Tooltip, message (6 components)
- [ ] **BPTriggerBuilder.tsx** - Card, Select, Form, InputNumber, Switch, Button, Input (7 components)

### Tier 3: High Complexity (5-10 hours total)
Complex components, significant refactoring

- [ ] **EntityDrawerTreeView.tsx** - Drawer, Button, Tree, Input, Collapse, Space, Badge, Popconfirm, message (9 components, Tree is complex)
- [ ] **EntityEditDetailModal.tsx** - Modal, Form, Input, Button, message, Select (6 components, modal-heavy)
- [ ] **pop/StewardApprovalPanel.tsx** - Card, List, Button, Modal, Form, Input, Select, Tag, Space, message, Avatar, Tooltip, Checkbox (13 components!)
- [ ] **pop/StewardGranularityReview.tsx** - Card, List, Button, Space, Tag, Typography, message, Modal, Descriptions, Tooltip (10 components)
- [ ] **pop/StewardUnionReview.tsx** - Card, List, Button, Space, Tag, Typography, message, Modal, Descriptions, Tooltip (10 components)
- [ ] **WorkflowTimeoutTriggersPage.tsx** - Card, Form, Select, InputNumber, Button, Table, Space, message, Modal, Input (10 components)
- [ ] **admin/RelatedObjectsPage.tsx** - Card, Row, Col, Typography, Space, Button, Select, message, Spin, Empty (10 components)

### Tier 4: Very High Complexity (10-15+ hours total)
Largest refactoring efforts, multiple interconnected components

- [ ] **EntityConfigPage.tsx** - Card, Table, Button, Form, Input, Select, message, Modal, Tree, Col, Row, Space, Tooltip, Popconfirm, Tag, Tabs, Empty, Spin (18 components, includes large Table with TreeProps)
- [ ] **EntityConfigPageV2.tsx** - Large form-heavy page with many AntD components
- [ ] **EntityConfigPageV3.tsx** - Complex multi-tab interface with forms and tables
- [ ] **WorkflowTimeoutTriggersPageEnhanced.tsx** - Unknown scope (need to examine)

---

## 📊 Statistics

```
Total files: 23
Completed: 1 ✅
Quick wins: 5
Medium complexity: 4
High complexity: 5
Very high complexity: 4
Unknown: 4
```

---

## 🚀 Recommended Approach

### Option A: Aggressive (Remove AntD completely)
1. Install replacement libraries:
   ```bash
   npm install sonner @headlessui/react
   ```

2. Follow Tier 1 → Tier 2 → Tier 3 → Tier 4 order
3. Remove `antd` from `package.json` when complete
4. Time estimate: **20-30 hours** for entire project

### Option B: Selective (Keep AntD for complex components)
1. Remove AntD from Tier 1 & 2 (quick wins)
2. Keep AntD for:
   - EntityConfigPage*.tsx (Table, Form complexity)
   - Tree-based components (Tree component is very complex)
   - Steward* components (highly interconnected)

3. Time estimate: **5-10 hours** to modernize most of codebase

### Option C: Hybrid (Recommended)
1. **Immediate**: Complete Tier 1 (1-2 hours)
2. **Phase 1**: Complete Tier 2 (3-5 hours)
3. **Phase 2**: Evaluate Tier 3-4 individually based on ROI
4. Keep complex components (Tree, large Tables) with AntD longer-term

---

## 🔧 Implementation Steps for Any File

1. **Audit current AntD usage** in the file
2. **Replace imports** with Tailwind + lucide-react
3. **Replace components** using the replacement table from ANTD_REMOVAL_GUIDE.md
4. **Add toast library** if file uses `message.*` calls
5. **Test the component** thoroughly
6. **Run linter** to ensure no errors
7. **Commit** with descriptive message

---

## 📋 Template for File Migration

```tsx
// BEFORE
import { Card, Button, message } from 'antd';

// AFTER  
import { Send } from 'lucide-react';
import { toast } from 'sonner';

// Component usage changes
// AntD: message.success('Saved!')
// Sonner: toast.success('Saved!')

// AntD: <Card> ... </Card>
// Tailwind: <div className="bg-white dark:bg-slate-900 rounded-lg border border-slate-200 dark:border-slate-800 shadow-sm p-6"> ... </div>

// AntD: <Button> Save </Button>
// Tailwind: <button className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"> Save </button>
```

---

## ⚠️ Known Challenges

### 1. Tree Component (EntityDrawerTreeView, EntityConfigPage)
- AntD Tree is highly optimized for large hierarchies
- Replacement: Keep AntD Tree for now, or use alternative like `@tanstack/react-table` with custom tree logic
- **Recommendation**: Isolate AntD Tree in wrapper component, leave for Phase 2

### 2. Table Component (EntityConfigPage, WorkflowTimeoutTriggersPage)
- AntD Table has excellent sorting, filtering, pagination built-in
- Replacement: Use `@tanstack/react-table` (React Table) + Tailwind styling
- **Recommendation**: Migrate table-heavy pages last, use React Table if needed

### 3. Form Validation (EntityConfigPage*, WorkflowTimeoutTriggersPageEnhanced)
- AntD Form has integrated validation
- Replacement: Use `react-hook-form` + standard `<input>` elements
- **Recommendation**: Check if `react-hook-form` is already available in project

### 4. Message/Toast (All pages)
- AntD message.* functions
- Replacement: Use `sonner` or `react-hot-toast`
- **Recommendation**: Install one toast library, use consistently

### 5. Modal/Drawer (Multiple pages)
- AntD Modal and Drawer are convenient
- Replacement: Use `@headlessui/react` Dialog + custom styling
- **Recommendation**: Create reusable Modal wrapper component

---

## 💾 Backup Strategy

Before starting large migrations:
```bash
# Create backup branch
git checkout -b antd-removal-backup

# Commit current state
git commit -m "backup: snapshot before AntD removal"

# Switch to working branch
git checkout -b antd-removal-work
```

---

## 📞 Progress Tracking

Mark progress in this file as you complete tiers:
- Update checkboxes above
- Update git commit history
- Document any findings or blockers

