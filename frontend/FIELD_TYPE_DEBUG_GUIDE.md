# Field Type Debug Display Guide

## What Was Added

When you add a validation condition and select a field, you'll now see a **blue debug panel** that displays:

### Panel Contents

```
📊 Field Type Info:
  Field Name: employee_id
  Data Type: string
  ✓ Type detected - Advanced operators should be available
```

## Why This Helps

### Case 1: Type is Detected ✓
```
Data Type: string
✓ Type detected - Advanced operators should be available
```
**What this means:**
- Field metadata loaded successfully from your entity
- Advanced Condition Builder will show Looker expressions
- You can use pattern matching (contains, starts_with, etc.)

**Next step:** Click in the Advanced Condition Builder section below

### Case 2: Type is Unknown ⚠️
```
Data Type: unknown
⚠️ Type is unknown - check if entity was selected and fields are loaded
```
**What this means:**
- Field metadata is NOT loading from your semantic objects
- Advanced operators won't be available
- Likely causes:
  1. Entity not selected in Step 2
  2. Semantic objects API not returning field metadata
  3. Field metadata not being transformed properly

**Troubleshooting steps:**
1. Go back to Step 2 (Config) and verify you selected an **entity**
2. Check browser DevTools Console for API errors
3. Verify your semantic objects endpoint returns data in the right format
4. Check that field names match exactly

## What Data Types Look Like

| Data Type | Advanced Options | Example Use |
|-----------|-----------------|-------------|
| `string` | Contains, Starts With, Ends With, Looker Expressions | Employee name, department code |
| `number` | Greater Than, Less Than, Intervals, Expressions | Salary, age, employee count |
| `date` | Relative dates (Last 7 days), Date ranges | Hire date, review date |
| `boolean` | True/False, Empty/Not Empty | Is active, is manager |
| `enum` | Dropdown of values | Status, region, department |
| `unknown` | Basic operators only | ⚠️ Indicates metadata loading issue |

## Quick Diagnostic Flow

```
✅ Add condition
  ↓
✅ Select field
  ↓
📊 Debug panel shows data type
  ↓
Is it "unknown"?
  ├─ YES → Check if entity selected, verify API returns fields
  └─ NO → Proceed! Advanced operators available
  ↓
✅ Advanced Condition Builder should show type-specific operators
```

## Example Workflow

### Working State (Type Loaded)
```
Condition 1
  Field: [Select a field...]
    ↓ (select "employee_id" from dropdown)
  
  📊 Field Type Info:
    Field Name: employee_id
    Data Type: string
    ✓ Type detected - Advanced operators should be available

  Advanced Condition Builder
    Operator: [Equals ▼]  (shows string operators)
    Value: [________________]
    Help text: Use % for wildcards, -% to exclude
```

### Problem State (Type Not Loading)
```
Condition 1
  Field: [Select a field...]
    ↓ (select "employee_id" but field list was empty)
    ERROR: No field metadata available

  📊 Field Type Info:
    Field Name: employee_id
    Data Type: unknown
    ⚠️ Type is unknown - check if entity was selected and fields are loaded

  Advanced Condition Builder
    Operator: [Equals ▼]  (only basic operators)
    Value: [________________]
```

## Checking in DevTools

### 1. Verify Entity Selection
```javascript
// In browser console:
JSON.parse(localStorage.getItem('selected_entity'))
// Should return: { id: "...", name: "Employee" }
```

### 2. Check Loaded Field Metadata
```javascript
// Component state (requires React DevTools extension)
// Look for dynamicFieldMetadata in component state
// Should show: { "employee_id": { type: "string" }, "salary": { type: "number" }, ... }
```

### 3. Test API Endpoint
```bash
# Terminal:
curl -H "X-Tenant-ID: <tenant_id>" \
     -H "X-Tenant-Datasource-ID: <datasource_id>" \
     "http://localhost:8080/api/semantic-objects/Employee/fields"
```

Expected response:
```json
[
  { "name": "employee_id", "data_type": "string", "required": true },
  { "name": "salary", "data_type": "integer", "required": false },
  { "name": "hire_date", "data_type": "date", "required": true }
]
```

## What Changed in the Code

**ValidationRuleCreator.tsx**

Before:
```tsx
// No indication of field type loaded or not
{c.field && <AdvancedConditionBuilder ... />}
```

After:
```tsx
// Shows debug info with:
// - Field name
// - Detected data type (or "unknown")
// - Status indicator (✓ or ⚠️)
{c.field && (
  <div className="bg-blue-50 border border-blue-200 rounded">
    <div>📊 Field Type Info:</div>
    <div>Field Name: {c.field}</div>
    <div>Data Type: {fieldType}</div>
    <div>{fieldType === 'unknown' ? '⚠️ ...' : '✓ ...'}</div>
  </div>
)}
{c.field && <AdvancedConditionBuilder ... />}
```

## Key Takeaway

**If you see `unknown` as the data type:**
- Field metadata is not loading from your semantic objects
- Advanced Condition Builder won't have type-specific operators
- Check that: entity is selected → API endpoint is configured → data types are being transformed correctly

**If you see the actual type (string, number, date, etc.):**
- ✅ Field metadata loaded successfully
- ✅ Advanced operators should work
- ✅ Looker expressions available for that type
