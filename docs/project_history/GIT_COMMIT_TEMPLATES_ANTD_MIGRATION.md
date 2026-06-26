# Git Commit Messages for Antd Migration

Use these commit message templates when submitting migrations:

---

## Template 1: Single File Migration

```
chore: migrate {ComponentName} from antd to MUI

- Replace antd Form with react-hook-form
- Convert antd Select to MUI Select + MenuItem
- Update icons: {AntdIcon} → {LucideIcon}
- Replace message calls with useNotification hook

Related: Antd Removal Project
```

**Example:**
```
chore: migrate BPTriggerBuilder from antd to MUI

- Replace antd Form with react-hook-form + Controller
- Convert Select components to MUI Select + MenuItem
- Update icons: ThunderboltOutlined → Zap
- Replace message.success() with notification.success()

Related: Antd Removal Project
```

---

## Template 2: Batch Component Migration

```
chore: migrate ExpressionBuilder components from antd to MUI

- Migrate OperatorSelector.tsx: Select → Select + MenuItem
- Migrate ValueInput.tsx: Input/InputNumber → TextField
- Migrate DroppableCondition.tsx: Select → Select + MenuItem
- Migrate ExpressionBuilder.tsx: Form + message → react-hook-form + useNotification

Related: Antd Removal Project - Phase 2
```

---

## Template 3: Utility/Helper Updates

```
chore: add antd→MUI migration utilities

- Add iconMapping.ts: Complete mapping of 100+ antd icons
- Add useNotification.ts: Hook to replace antd message API
- Update package.json: Remove antd and @ant-design/icons

This enables rapid migration of remaining components.

Related: Antd Removal Project - Phase 1
```

---

## Template 4: Fix/Refinement

```
chore: fix antd migration issues in {ComponentName}

- Fix TypeScript errors in Select onChange handler
- Correct MUI TextField prop usage
- Update icon sizes to match original

Related: Antd Removal Project
```

---

## Best Practices

### DO:
✅ Be specific about what components changed  
✅ Mention the MUI equivalent  
✅ Reference icon replacements  
✅ Use "chore:" prefix (non-feature change)  
✅ Add "Related: Antd Removal Project" tag  
✅ Keep commits focused (one component or related group)  

### DON'T:
❌ Mix migration with feature changes  
❌ Vague messages like "update components"  
❌ Include unrelated refactoring  
❌ Forget to mention breaking changes (none expected)  

---

## Commit Message Examples

### ✅ GOOD Examples

```
chore: migrate PolicyBuilder from antd to MUI

- Form: antd Form → react-hook-form
- Icons: PlusOutlined, DeleteOutlined → Plus, Trash2
- Table: antd Table → MUI DataGrid
- Modal: antd Modal → MUI Dialog
- Notifications: message → useNotification hook

Removes 200+ lines of antd-specific code.
Related: Antd Removal Project
```

```
chore: migrate POP components from antd to MUI

Batch migration of 4 simple components:
- CalendarModeToggle: Select, Card, Typography
- CohortFilterSelector: Select, Card, Tag
- LineageVisualizer: Card, Spin, message
- RelationshipPathVisualizer: Card, Tooltip, Badge

All use standard MUI patterns defined in guide.
Related: Antd Removal Project - Phase 3
```

```
chore: complete antd icon replacement in RelationshipBuilder

Maps 15+ @ant-design/icons to lucide-react:
- DatabaseOutlined → Database
- RobotOutlined → Bot
- CheckCircleOutlined → CheckCircle
- etc.

Related: Antd Removal Project - Phase 4
```

### ❌ BAD Examples

```
✗ Update components
✗ Migrate UI
✗ Fix stuff
✗ chore: massive refactor (too vague, too large)
```

---

## PR Description Template

Use this when opening a pull request:

```markdown
## Description
This PR migrates [NUMBER] components from Ant Design to Material-UI as part of the standardization project.

## Components Migrated
- [ ] Component 1
- [ ] Component 2
- [ ] Component 3

## Changes Made
- Removed: `antd`, `@ant-design/icons` imports
- Added: `@mui/material` imports, `react-hook-form`, `useNotification` hook
- Updated: Form handling, icon references, styling

## Testing
- [x] Components render without errors
- [x] Forms submit correctly
- [x] Icons display properly
- [x] Responsive design verified
- [x] No console errors

## Related
- Closes: Antd Removal Project (Phase X)
- References: ANTD_TO_MUI_MIGRATION_GUIDE.md
```

---

## Quick Commit Commands

### Single File
```bash
git add frontend/src/components/BPTriggerBuilder.tsx
git commit -m "chore: migrate BPTriggerBuilder from antd to MUI

- Replace antd Form with react-hook-form + Controller
- Update icons to lucide-react
- Replace message with useNotification hook"
```

### Batch
```bash
git add frontend/src/components/ExpressionBuilder/*.tsx
git commit -m "chore: migrate ExpressionBuilder components from antd to MUI

Files:
- OperatorSelector.tsx
- ValueInput.tsx  
- DroppableCondition.tsx
- ExpressionBuilder.tsx"
```

### Utilities
```bash
git add frontend/src/utils/iconMapping.ts frontend/src/hooks/useNotification.ts
git commit -m "chore: add antd→MUI migration utilities

- Icon mapping for 100+ antd icons
- Notification hook to replace message API"
```

---

## Commit Frequency

**Recommended schedule:**
- 🟢 Quick wins: 1 commit per component group (3-5 components)
- 🟡 Medium: 1 commit per complex component
- 🔴 Complex: 1 commit per file (if very large)
- 📦 Pages: 1 commit per page or related group

**Daily average:** 3-5 commits across ~8 files

---

## Reference Links in Commits

Include these for traceability:

```
Related: Antd Removal Project
Ref: ANTD_TO_MUI_MIGRATION_GUIDE.md
Icons: frontend/src/utils/iconMapping.ts
Hook: frontend/src/hooks/useNotification.ts
Docs: MIGRATION_PLAN_UI_STANDARDIZATION.md
```

---

## Post-Migration Checklist (in commit message)

When closing a migration task, include:

```
Migration checklist:
✅ All antd imports removed
✅ MUI equivalents imported
✅ Form handling updated (if applicable)
✅ Icons replaced
✅ Message calls replaced
✅ TypeScript errors resolved
✅ Component tested
✅ No console warnings
```

---

**Ready to commit? Use these templates to keep history clean and traceable!**

