# Quick Start: Antd Removal Migration

**TL;DR** - Standard patterns have been established. Use these to migrate remaining 41 files quickly.

---

## 🚀 Quick Reference Patterns

### Pattern 1: Simple Select Component
```typescript
// BEFORE
import { Select } from 'antd';
<Select value={value} onChange={onChange}>
  <Select.Option value="a">Option A</Select.Option>
  <Select.Option value="b">Option B</Select.Option>
</Select>

// AFTER
import { Select, MenuItem } from '@mui/material';
<Select value={value} onChange={(e) => onChange(e.target.value)}>
  <MenuItem value="a">Option A</MenuItem>
  <MenuItem value="b">Option B</MenuItem>
</Select>
```

### Pattern 2: Form with Fields
```typescript
// BEFORE
const [form] = Form.useForm();
<Form form={form} onFinish={handleSave}>
  <Form.Item name="email" label="Email" rules={[{ required: true }]}>
    <Input />
  </Form.Item>
  <Button type="primary" htmlType="submit">Save</Button>
</Form>

// AFTER
import { useForm, Controller } from 'react-hook-form';
const { control, handleSubmit } = useForm();
<Box component="form" onSubmit={handleSubmit(handleSave)}>
  <Controller
    name="email"
    control={control}
    rules={{ required: 'Required' }}
    render={({ field, fieldState: { error } }) => (
      <TextField {...field} label="Email" error={!!error} helperText={error?.message} />
    )}
  />
  <Button type="submit" variant="contained">Save</Button>
</Box>
```

### Pattern 3: Icon Replacement
```typescript
// BEFORE
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
<Button icon={<PlusOutlined />}>Add</Button>
<DeleteOutlined />

// AFTER
import { Plus, Trash2 } from 'lucide-react';
<Button startIcon={<Plus className="w-5 h-5" />}>Add</Button>
<Trash2 className="w-5 h-5" />

// OR use MUI icons
import { AddIcon, DeleteIcon } from '@mui/icons-material';
<Button startIcon={<AddIcon />}>Add</Button>
<DeleteIcon />
```

### Pattern 4: Message/Notification
```typescript
// BEFORE
import { message } from 'antd';
message.success('Saved!');
message.error('Failed!');

// AFTER
import { useNotification } from '../../hooks/useNotification';
const notification = useNotification();
notification.success('Saved!');
notification.error('Failed!');
```

### Pattern 5: Card Component
```typescript
// BEFORE
<Card title="My Card">
  Content here
</Card>

// AFTER
import { Card, CardHeader, CardContent } from '@mui/material';
<Card>
  <CardHeader title="My Card" />
  <CardContent>
    Content here
  </CardContent>
</Card>
```

### Pattern 6: Table (Simple)
```typescript
// BEFORE
<Table columns={columns} dataSource={data} />

// AFTER
import { DataGrid } from '@mui/x-data-grid';
<DataGrid rows={data} columns={columns} />
```

### Pattern 7: Modal/Dialog
```typescript
// BEFORE
<Modal visible={open} onOk={handleOk} onCancel={handleCancel}>
  Content
</Modal>

// AFTER
import { Dialog, DialogContent } from '@mui/material';
<Dialog open={open} onClose={handleCancel}>
  <DialogContent>
    Content
  </DialogContent>
</Dialog>
```

---

## 🗺️ File Priority & Difficulty

### QUICK WINS (15 minutes each)
- ✅ BPTriggerBuilder.tsx _(DONE)_
- ExpressionBuilder components _(DONE - 4 files)_
- CalendarModeToggle.tsx
- CohortFilterSelector.tsx
- LineageVisualizer.tsx
- RelationshipPathVisualizer.tsx
- UnifiedCRUDPage.tsx

### MEDIUM (30-45 minutes each)
- StewardUnionReview.tsx
- StewardGranularityReview.tsx
- AuditLogViewer.tsx
- DelegationManager.tsx
- RelationshipDiscoveryModal.tsx
- AIRoutingDashboard.tsx

### COMPLEX (1-2 hours each)
- PolicyBuilder.tsx
- TriggerBuilder.tsx
- ReportBuilder.tsx
- EntityEditDetailModal.tsx
- EntityDrawerTreeView.tsx

### PAGES (30 minutes each)
- EntityConfigPageV2.tsx
- EntityConfigPageV3.tsx
- WorkflowTimeoutTriggersPage.tsx
- etc.

---

## 📋 Migration Checklist

For each file, follow this checklist:

```
☐ 1. Identify all antd imports
☐ 2. Replace with MUI/lucide equivalents
☐ 3. Update form handling to react-hook-form (if applicable)
☐ 4. Replace message() calls with useNotification()
☐ 5. Replace icons with lucide-react
☐ 6. Test component renders
☐ 7. Verify TypeScript errors resolved
☐ 8. Check for inline styles linting (OK to ignore)
☐ 9. Commit with message: "chore: migrate {component} from antd to MUI"
☐ 10. Ready for review
```

---

## 🎯 Daily Goals

**Day 1:** Quick wins (7 files) - 2-3 hours  
**Day 2:** Medium complexity (6 files) - 3-4 hours  
**Day 3:** Complex components (5 files) - 4-5 hours  
**Day 4:** Pages (6-7 files) - 3-4 hours  
**Day 5:** Final polish, testing, PRs - 4-5 hours  

---

## 🔧 Tools & References

### Local Reference Files
- Icon mapping: `frontend/src/utils/iconMapping.ts`
- Notification hook: `frontend/src/hooks/useNotification.ts`
- Full guide: `ANTD_TO_MUI_MIGRATION_GUIDE.md`
- Plan: `MIGRATION_PLAN_UI_STANDARDIZATION.md`

### Online Documentation
- MUI Components: https://mui.com/material-ui/api/
- React Hook Form: https://react-hook-form.com/
- Lucide Icons: https://lucide.dev/
- Tailwind: https://tailwindcss.com/

---

## ⚡ Speed Tips

1. **Use VSCode Search & Replace**
   - Find: `from 'antd'`
   - Replace: `from '@mui/material'`

2. **Mass Icon Replacement**
   - Find: `from '@ant-design/icons'`
   - Replace: `from 'lucide-react'`

3. **Template Components**
   - Copy patterns from BPTriggerBuilder (forms)
   - Copy patterns from ExpressionBuilder components (simple selects)
   - Copy patterns from this guide

4. **Batch Testing**
   - Test a few components at once
   - Use `npm run dev` to see live changes
   - Watch for TypeScript errors

---

## ⚠️ Common Pitfalls to Avoid

| Pitfall | Solution |
|---------|----------|
| Forgetting `useForm` import | Add to component: `const { control, handleSubmit } = useForm()` |
| Message calls not imported | Use `useNotification` hook at top of component |
| Icon name mismatches | Check `iconMapping.ts` for correct lucide name |
| Select onChange wrong signature | MUI: `onChange={(e) => fn(e.target.value)}` not `onChange={fn}` |
| Form validation lost | Use Controller with `rules` prop |
| Styling gone | Check if styles were inline; move to CSS module |

---

## ✅ Success Criteria

Each migrated component should:
- ✅ No imports from `'antd'` remain
- ✅ No imports from `'@ant-design/icons'` remain
- ✅ All TypeScript errors resolved
- ✅ Component renders in dev
- ✅ Functionality intact (forms submit, tables sort, etc.)
- ✅ Looks similar to original
- ✅ Responsive on mobile
- ✅ No console errors

---

## 🎬 Getting Started

### First Migration
1. Open `frontend/src/components/ExpressionBuilder/OperatorSelector.tsx` - **ALREADY DONE** ✅
2. Look at the changes made (it's your template)
3. Pick the next file: `CalendarModeToggle.tsx`
4. Follow the pattern
5. Commit and move to next

### Expected Timeline
- **5 files/day at 30 min each** = Full project in 2 weeks
- **10 files/day at 15 min each** = Full project in 1 week
- **With two developers** = 3-5 days

---

## 📞 Support

**Blocked on a component?**
1. Check `ANTD_TO_MUI_MIGRATION_GUIDE.md` for that file
2. Look for similar patterns in already-migrated files
3. Reference icon mapping: `frontend/src/utils/iconMapping.ts`
4. Check MUI docs for equivalent component

**TypeScript errors?**
- Run `npm run lint` to see all errors
- Most are simple type mismatches
- MUI components are well-typed

**Design doesn't match?**
- MUI default styling may differ
- Use `sx` prop for MUI or Tailwind for layout
- Check original antd theme settings if needed

---

**Ready to start? Pick a file and follow the patterns above!**

