# AntD Removal Guide

## Overview
This guide provides a comprehensive strategy for removing Ant Design (AntD) from the Semlayer project and replacing it with Tailwind CSS-based components.

## ✅ Completed Removals

### EntityDetailsPage (/Users/eganpj/GitHub/semlayer/frontend/src/pages/EntityDetailsPage.tsx)
- **Removed imports**: Card, Button, Tabs, Spin, Empty, Typography, Space, Alert, ArrowLeftOutlined
- **Replaced with**:
  - `Tabs` → Custom Tailwind tabs with useState
  - `Card` → `<div className="bg-white dark:bg-slate-900 rounded-lg border border-slate-200 dark:border-slate-800">`
  - `Button` → `<button className="...">`
  - `Spin` → Custom animated spinner div
  - `Empty` → Custom empty state with emoji and Tailwind styling
  - `Alert` → Custom blue-tinted alert box with AlertCircle icon
  - `Typography` → Standard `<h3>`, `<p>` tags with Tailwind classes
  - `Space` → CSS flexbox with `flex`, `gap-*` classes
- **Status**: ✅ Complete - No AntD dependencies, uses lucide-react icons only

---

## 📋 AntD Usage Inventory

### Pages & Components Using AntD

#### High Priority (Used by active features)

1. **EntityConfigPage.tsx** (Line 3)
   ```tsx
   import { Card, Table, Button, Form, Input, Select, message, Modal, Tree as _Tree, Col, Row, Space, Tooltip, Popconfirm, Tag as _Tag, Tabs, Empty, Spin as _Spin }
   ```
   - Uses: Card, Table (large data grid), Button, Form, Input, Select, message, Modal, Tree, Col/Row (layout), Space, Tooltip, Popconfirm, Tag, Tabs, Empty, Spin
   - Complexity: **HIGH** - Table with large dataset, Form validation, Tree component
   - Recommendation: Gradual migration, start with simpler parts

2. **EntityDrawerTreeView.tsx** (Line 15)
   ```tsx
   import { Drawer, Button, Tree, Input, Button, Collapse, Space, Badge, Popconfirm, message }
   ```
   - Uses: Drawer (modal-like side panel), Tree, Collapse, Badge, Popconfirm
   - Complexity: **HIGH** - Tree navigation, Drawer modal
   - Recommendation: Keep Tree for now, replace Drawer with custom modal

3. **AIRouting/AIRoutingDashboard.tsx** (Line 16)
   ```tsx
   import { Card, Button, Tabs, Input, Select, Form, message, Popconfirm, Modal }
   ```
   - Uses: Card, Tabs, Form, Modal
   - Complexity: **MEDIUM**

4. **ExpressionBuilder/* components** (Lines 2-3)
   ```tsx
   // OperatorSelector.tsx: Select from 'antd'
   // ValueInput.tsx: Input, InputNumber from 'antd'
   // ExpressionBuilder.tsx: Button, Card, Typography, message
   ```
   - Complexity: **MEDIUM**

5. **WorkflowTimeoutTriggersPage.tsx** (Line 3)
   ```tsx
   import { Card, Form, Select, InputNumber, Button, Table, Space, message, Modal, Input }
   ```
   - Uses: Card, Form, Table, Modal
   - Complexity: **MEDIUM**

#### Medium Priority

6. **RelatedObjectsPage.tsx** (admin)
   ```tsx
   import { Card, Row, Col, Typography, Space, Button, Select, message, Spin, Empty }
   ```
   - Complexity: **MEDIUM**

7. **EntityConfigPageV2.tsx**
   - Full layout with Form, Card, Table, etc.
   - Complexity: **HIGH**

8. **EntityConfigPageV3.tsx**
   - Multiple complex components
   - Complexity: **HIGH**

#### Lower Priority (Support components)

9. **EntityEditDetailModal.tsx**
   - Uses: Modal, Form, Input, Button, message, Select
   - Status: Modal wrapper

---

## 🔄 Replacement Strategy

### Common AntD → Tailwind Replacements

#### Layout Components
| AntD | Tailwind | Notes |
|------|----------|-------|
| `<Card>` | `<div className="bg-white dark:bg-slate-900 rounded-lg border border-slate-200 dark:border-slate-800 shadow-sm p-6">` | Use CSS for shadow/border |
| `<Row>` | `<div className="flex flex-wrap gap-4">` | Or use grid: `grid grid-cols-*` |
| `<Col>` | `<div className="flex-1">` | Or use `grid col-span-*` |
| `<Space>` | `<div className="flex gap-2">` or `<div className="flex flex-col gap-4">` | Use flex with gap |

#### Form Components
| AntD | Tailwind | Notes |
|------|----------|-------|
| `<Form>` | `<form className="space-y-4">` | Use space-y for vertical spacing |
| `<Input>` | `<input className="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500">` | Add focus states |
| `<Select>` | Use headless UI Select or standard `<select>` | Tailwind has limited Select support |
| `<Button>` | `<button className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors">` | Various variants available |

#### Data Display
| AntD | Tailwind | Notes |
|------|----------|-------|
| `<Table>` | Custom `<table>` with Tailwind classes | More control, more code |
| `<Tree>` | No direct equivalent - keep AntD Tree OR build custom | Complex component, recommend keeping for now |
| `<Tag>` | `<span className="inline-block px-3 py-1 bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200 rounded-full text-sm">` | Badge-like styling |

#### Modals & Overlays
| AntD | Tailwind | Notes |
|------|----------|-------|
| `<Modal>` | Custom modal with Portal + `fixed` positioning | Requires more setup |
| `<Drawer>` | Custom drawer with `fixed` + transform | Can use Headless UI |
| `message.info()` | Toast library (Sonner, react-hot-toast, react-toastify) | Recommend adding toast library |

#### Others
| AntD | Tailwind | Notes |
|------|----------|-------|
| `<Tabs>` | Custom with button toggle + conditional rendering | Like EntityDetailsPage example |
| `<Spin>` | `<div className="w-8 h-8 border-4 border-slate-200 border-t-blue-500 rounded-full animate-spin">` | Built-in animation |
| `<Empty>` | Custom div with emoji + text | Simple HTML |
| `<Alert>` | `<div className="p-4 bg-blue-50 dark:bg-blue-950/30 border border-blue-200 rounded-lg">` | Various colors |
| `<Tooltip>` | Headless UI Popover or custom div on hover | Can be done with CSS |
| `<Popconfirm>` | Custom modal overlay | Build with state management |
| `<Badge>` | See Tag above | Similar styling |

---

## 🛠️ Migration Checklist

### Phase 1: Simple Components (Easy wins)
- [ ] Replace Typography with `<h1>`, `<p>`, `<span>` + Tailwind classes
- [ ] Replace Card with div + Tailwind classes
- [ ] Replace Button with `<button>` + Tailwind classes
- [ ] Replace Space with flex/gap utilities
- [ ] Replace Alert with custom div
- [ ] Replace Empty with custom div
- [ ] Replace Spin with custom animated div
- [ ] Replace Tag/Badge with span + classes

### Phase 2: Medium Components
- [ ] Replace Tabs with custom implementation (see EntityDetailsPage)
- [ ] Replace Input with standard `<input>` + Tailwind
- [ ] Replace InputNumber with `<input type="number">` + Tailwind
- [ ] Replace Select with Headless UI `Listbox` or keep as standard `<select>`

### Phase 3: Complex Components
- [ ] Replace Form with standard `<form>` + custom validation
- [ ] Replace Table with custom `<table>` + Tailwind classes
- [ ] Replace Modal with custom modal (Portal + fixed positioning)
- [ ] Replace Drawer with custom drawer (Portal + transform animations)
- [ ] Replace Tree with custom Tree OR keep AntD Tree (isolated)

### Phase 4: Utilities & Migrations
- [ ] Remove `message` - replace with toast library (e.g., sonner, react-hot-toast)
- [ ] Remove `Modal.confirm` - replace with custom confirmation modal
- [ ] Remove `Popconfirm` - replace with custom confirmation popover
- [ ] Remove `Tooltip` - replace with custom CSS tooltip or Headless UI Popover

---

## 📦 Recommended Libraries for Replacements

### Form Handling
- **react-hook-form** - Lightweight form validation (already might be available)

### UI Components
- **@headlessui/react** - Unstyled, accessible components for Modals, Popovers, etc.
- **lucide-react** - Icon library (already in use)

### Toast/Notifications
- **sonner** - Modern toast library with great DX
- **react-hot-toast** - Simple alternative
- **react-toastify** - Full-featured alternative

### Tables
- **TanStack Table (React Table)** - Headless table library for complex tables

---

## 🔍 File-by-File Migration Guide

### EntityDetailsPage.tsx ✅
**Status**: COMPLETE
**AntD imports removed**: All
**Replacement approach**: Direct Tailwind classes, custom tabs, lucide-react icons

---

## 📝 Next Steps

1. **Decide on scope**: Are you removing AntD from entire project or just specific pages?
   - Option A: Remove globally (breaking change, affects all pages)
   - Option B: Remove selectively (keep AntD for complex pages, remove from simple ones)

2. **Pick toast library**: For message/notification replacements
   - Recommendation: **Sonner** (modern, great UX)

3. **Identify must-have components**: Which AntD components are genuinely complex?
   - Tree navigation → Consider keeping
   - Table grids → Can use TanStack Table
   - Forms → Use react-hook-form + standard inputs

4. **Start migration**: Follow Phase 1 → Phase 2 → Phase 3 order

---

## 💡 Tips & Best Practices

### Dark Mode Support
All replacement components should support dark mode using `dark:` Tailwind prefix:
```tsx
<div className="bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-50">
```

### Accessibility
Use proper semantic HTML and ARIA attributes:
```tsx
<button aria-label="Delete item" className="...">
  <Trash2 size={20} />
</button>
```

### Consistency
Create reusable component wrappers to maintain consistency:
```tsx
// Button wrapper
export function Button({ variant = 'primary', ...props }) {
  const baseClass = 'px-4 py-2 rounded-lg font-medium transition-colors';
  const variants = {
    primary: 'bg-blue-600 hover:bg-blue-700 text-white',
    secondary: 'bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600',
  };
  return <button className={`${baseClass} ${variants[variant]}`} {...props} />;
}
```

### Performance
Monitor re-renders when replacing AntD controlled components with custom implementations.

---

## ❓ Questions?

- For component-specific migration help, check the component's current implementation
- For styling issues, reference Tailwind documentation at https://tailwindcss.com
- For accessibility, see WCAG 2.1 guidelines and Headless UI docs

