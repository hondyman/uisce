# Quick Start: ParameterBuilder

> **TL;DR:** Use `ParameterBuilder` component to render dynamic parameter forms. Define schema in `parameterSchemas.ts`. Done!

---

## 🚀 5-Minute Setup

### **Step 1: Import**
```tsx
import ParameterBuilder from '../components/ParameterBuilder';
import { getParameterSchema } from '../lib/parameterSchemas';
```

### **Step 2: Use**
```tsx
<ParameterBuilder
  schema={getParameterSchema('CONCENTRATION')!}
  parameters={formData.parameters}
  onChange={(params) => setFormData({ ...formData, parameters: params })}
/>
```

### **Step 3: Done!** ✅

---

## 📋 Common Examples

### **Get Schema for Rule Type**
```typescript
const schema = getParameterSchema('CONCENTRATION');
// Returns: ParameterSchema with all field definitions
```

### **Validate Parameters**
```typescript
import { validateParameters } from '../lib/parameterSchemas';

const errors = validateParameters('CONCENTRATION', {
  maxPositionPercentage: 150, // Invalid: > 100
});
// Returns: { maxPositionPercentage: "Must be between 0 and 100" }
```

### **Get All Available Rule Types**
```typescript
import { getAvailableRuleTypes } from '../lib/parameterSchemas';

const types = getAvailableRuleTypes();
// Returns: [
//   { value: 'CONCENTRATION', label: 'Concentration', description: '...' },
//   { value: 'KYC', label: 'Know Your Customer', description: '...' },
//   ...
// ]
```

### **Add New Rule Type**
```typescript
// In frontend/src/lib/parameterSchemas.ts

export const PARAMETER_SCHEMAS = {
  // ... existing types ...
  
  MY_NEW_TYPE: {
    ruleType: 'MY_NEW_TYPE',
    name: 'My New Type',
    description: 'What it does',
    fields: [
      {
        name: 'fieldName',
        label: 'Field Label',
        type: 'number',
        required: true,
      },
    ],
  },
};
```

That's it! ParameterBuilder automatically picks it up.

---

## 🎯 Use Cases

### **Validation Rule Form**
```tsx
<ParameterBuilder
  schema={getParameterSchema(ruleType)!}
  parameters={parameters}
  onChange={setParameters}
/>
```

### **Report Configuration**
```tsx
<ParameterBuilder
  schema={getParameterSchema(reportType)!}
  parameters={reportConfig}
  onChange={setReportConfig}
/>
```

### **Business Rule Builder**
```tsx
<ParameterBuilder
  schema={getParameterSchema(ruleType)!}
  parameters={ruleParams}
  onChange={updateRule}
/>
```

### **With Error Display**
```tsx
<ParameterBuilder
  schema={schema}
  parameters={params}
  onChange={setParams}
  errors={validationErrors}
  showValidation={isSubmitting}
/>
```

---

## 🔧 Field Types

| Type | Example | Use For |
|------|---------|---------|
| `text` | "My Rule Name" | Text inputs |
| `number` | 10.5 | Numeric values |
| `checkbox` | true/false | Toggles |
| `select` | "DAILY" | Dropdowns |
| `multiselect` | ["A", "B"] | Multi-checkboxes |
| `textarea` | "Long text..." | Multi-line text |
| `slider` | 50 | Range inputs |
| `comma-list` | "a,b,c" | CSV arrays |

---

## 💾 All 11 Rule Types

1. **CONCENTRATION** - Position concentration limits
2. **KYC** - Know your customer checks
3. **ACCOUNT_STRUCTURE** - Account setup validation
4. **PORTFOLIO** - Portfolio exposure limits
5. **PRICING** - Price deviation checks
6. **TRADE** - Trade execution validation
7. **FEE** - Fee structure limits
8. **DATA_INTEGRITY** - Data accuracy checks
9. **ASSET_RESTRICTION** - Prohibited assets
10. **LIQUIDITY** - Illiquid asset limits
11. **ACCESS_CONTROL** - User access rules

---

## 📖 Full Documentation

- **Guide:** `PARAMETER_BUILDER_GUIDE.md`
- **Before/After:** `BEFORE_AFTER_COMPARISON.md`
- **Integration:** `INTEGRATION_EXAMPLES.md`
- **Summary:** `OPTION_1_COMPLETION_SUMMARY.md`

---

## ❓ FAQ

**Q: How do I add a new field to an existing rule type?**
A: Edit the `fields` array in `PARAMETER_SCHEMAS[ruleType]` in `parameterSchemas.ts`

**Q: Can I reuse this in other components?**
A: Yes! It's completely generic. Use it anywhere you need dynamic parameter input.

**Q: How do I validate?**
A: Call `validateParameters(ruleType, params)`. Use `showValidation={true}` to display errors.

**Q: Can I customize styling?**
A: Yes, pass `className` and `fieldClassName` props. It supports dark mode automatically.

**Q: How do I extend it?**
A: Add new field types to `FieldType` union in `parameterSchemas.ts` and handle in `ParameterBuilder`.

---

## 🎬 Live Example

**File:** `frontend/src/pages/ValidationRulesBuilderPage.tsx`

Already uses ParameterBuilder! Check how it's integrated:
```tsx
{getParameterSchema(formData.ruleType) && (
  <ParameterBuilder
    schema={getParameterSchema(formData.ruleType)!}
    parameters={formData.parameters}
    onChange={(params) => setFormData({ ...formData, parameters: params })}
    showValidation={false}
  />
)}
```

---

**Status:** ✅ Production Ready | **Docs:** Comprehensive | **Examples:** Ready to Use
