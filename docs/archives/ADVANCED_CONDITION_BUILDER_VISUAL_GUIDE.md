# Advanced Condition Builder - Visual Reference Guide

## Component Architecture Diagram

```
┌────────────────────────────────────────────────────────────────────┐
│                   ValidationRuleEditor (Parent)                    │
│                                                                    │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │              ExpressionBuilder Wrapper                      │  │
│  │                                                             │  │
│  │  [Save Rule] [Test Rule]                                   │  │
│  │                                                             │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │      AdvancedConditionBuilder Component             │  │  │
│  │  │                                                      │  │  │
│  │  │  ┌────────────────────────────────────────────────┐ │  │  │
│  │  │  │ Condition Group (Root)                        │ │  │  │
│  │  │  │                                                │ │  │  │
│  │  │  │ [Collapse ▼] Root Conditions [AND] (2)      │ │  │  │
│  │  │  │ [+ Condition] [+ Group]                      │ │  │  │
│  │  │  │                                                │ │  │  │
│  │  │  │ ┌──────────────────────────────────────────┐ │ │  │  │
│  │  │  │ │ Condition Item 1                        │ │ │  │  │
│  │  │  │ │ ✋ [Age ▼]    [>= ▼]    [18 ___]   [✕]  │ │ │  │  │
│  │  │  │ └──────────────────────────────────────────┘ │ │  │  │
│  │  │  │                                                │ │  │  │
│  │  │  │ ┌────────────────────────────────────────┐   │ │  │  │
│  │  │  │ │         AND (Operator)                 │   │ │  │  │
│  │  │  │ └────────────────────────────────────────┘   │ │  │  │
│  │  │  │                                                │ │  │  │
│  │  │  │ ┌──────────────────────────────────────────┐ │ │  │  │
│  │  │  │ │ Nested Condition Group (OR)            │ │ │  │  │
│  │  │  │ │ [▼] Condition Group [OR] (2) [✕]     │ │ │  │  │
│  │  │  │ │ [+ Condition] [+ Group]                │ │ │  │  │
│  │  │  │ │                                          │ │ │  │  │
│  │  │  │ │ ┌──────────────────────────────────┐   │ │ │  │  │
│  │  │  │ │ │ Status = "Active"               │   │ │ │  │  │
│  │  │  │ │ └──────────────────────────────────┘   │ │ │  │  │
│  │  │  │ │          OR                             │ │ │  │  │
│  │  │  │ │ ┌──────────────────────────────────┐   │ │ │  │  │
│  │  │  │ │ │ Is VIP = true                    │   │ │ │  │  │
│  │  │  │ │ └──────────────────────────────────┘   │ │ │  │  │
│  │  │  │ └──────────────────────────────────────────┘ │ │  │  │
│  │  │  │                                                │ │  │  │
│  │  │  │ [▼ View Generated JSON]                      │ │  │  │
│  │  │  └────────────────────────────────────────────┘ │  │  │
│  │  │                                                  │  │  │
│  │  └──────────────────────────────────────────────────┘  │  │
│  │                                                         │  │
│  │  ┌──────────────────────────────────────────────────┐  │  │
│  │  │         Autosave Engine                          │  │  │
│  │  │                                                  │  │  │
│  │  │  Status: [Saving] | [Saved ✓] | [Error ✕]     │  │  │
│  │  └──────────────────────────────────────────────────┘  │  │
│  │                                                         │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                │
└────────────────────────────────────────────────────────────────────┘
                              ↓
                    Apollo GraphQL Client
                              ↓
        ┌─────────────────────────┬─────────────────────────┐
        ↓                                                   ↓
  INSERT_DRAFT_RULE                         UPDATE_RULE_BY_PK
  (First save only)                         (Subsequent saves)
        ↓                                                   ↓
        └─────────────────────────┬─────────────────────────┘
                                  ↓
                        GraphQL Server / Hasura
                                  ↓
                        PostgreSQL Database
                 (catalog_validation_rules table)
```

## UI State Transitions

### Adding a First Condition

```
Initial State:
┌──────────────────────────────────┐
│ No conditions in this group      │
│ Click "Condition" or "Group"     │
└──────────────────────────────────┘
              ↓ Click "+ Condition"
After Adding:
┌──────────────────────────────────┐
│ [✋] [Field ▼] [Operator ▼] [Value] [✕] │
│      All fields editable        │
└──────────────────────────────────┘
              ↓ Field Selected
┌──────────────────────────────────┐
│ [✋] [Age ▼] [>= ▼] [____] [✕]   │
│     Operators update to number   │
└──────────────────────────────────┘
              ↓ Value Entered
┌──────────────────────────────────┐
│ [✋] [Age ▼] [>= ▼] [18] [✕]    │
│     Ready to add more or save    │
└──────────────────────────────────┘
```

### Adding Nested Groups

```
Single Condition:
┌────────────────────────────────────┐
│ [✋] [Status ▼] [= ▼] [Active] [✕] │
└────────────────────────────────────┘
        ↓ Click "+ Group"
With Nested Group:
┌──────────────────────────────────────┐
│ [✋] [Status ▼] [= ▼] [Active] [✕]  │
│              AND                     │
│ [▼] Condition Group [OR] (0) [✕]   │
│     [+ Condition] [+ Group]          │
└──────────────────────────────────────┘
        ↓ Add Condition to Nested Group
┌──────────────────────────────────────┐
│ [✋] [Status ▼] [= ▼] [Active] [✕]  │
│              AND                     │
│ [▼] Condition Group [OR] (1) [✕]   │
│     ┌──────────────────────────────┐│
│     │ [✋] [is_vip ▼] [true ▼]  [✕]││
│     └──────────────────────────────┘│
│     [+ Condition] [+ Group]          │
└──────────────────────────────────────┘
```

### Toggling AND/OR Operator

```
Current: AND
┌────────────────────────────────────┐
│ Root Conditions [AND] (2)          │
│ [+ Condition] [+ Group]            │
│                                    │
│ Condition 1: Age >= 18            │
│          AND                       │
│ Condition 2: Status = Active      │
└────────────────────────────────────┘
        ↓ Click [AND] button
After Toggle: OR
┌────────────────────────────────────┐
│ Root Conditions [OR] (2)           │
│ [+ Condition] [+ Group]            │
│                                    │
│ Condition 1: Age >= 18            │
│          OR                        │
│ Condition 2: Status = Active      │
└────────────────────────────────────┘
```

## Type-Specific Input Controls

### String Field
```
Field: Email
Operators: [Equals | Not Equals | Contains | Starts With | Ends With | Is Empty | Is Not Empty]

Selected Operator: "Contains"
Input Type: Text
┌─────────────────┐
│ @company.com    │
└─────────────────┘
```

### Number Field
```
Field: Age
Operators: [Equals | Not Equals | > | < | >= | <=]

Selected Operator: ">="
Input Type: Number
┌─────────────────┐
│ 18              │
└─────────────────┘
```

### Date Field
```
Field: Hire Date
Operators: [On Date | Before | After | Between]

Selected Operator: "Between"
Input Type: Date
┌──────────────────────────────┐
│ 2020-01-01 to 2025-12-31    │
└──────────────────────────────┘
```

### Boolean Field
```
Field: Is VIP
Operators: [Is True | Is False]

Selected Operator: "Is True"
Input Type: Dropdown (no value needed)
┌─────────────────┐
│ True ▼          │
└─────────────────┘
```

## Autosave Timeline

```
0ms:  User types in condition
      └─> schedulePersist() called
          └─> Timer set for 1000ms

500ms: User types in another condition
       └─> Previous timer cleared
           └─> New timer set for 1000ms

1500ms: No changes for 1000ms
        └─> persistNow() executed
            ├─> Check tenant scope ✓
            ├─> Check ruleId exists?
            │   └─> NO → INSERT_DRAFT_RULE mutation
            │           └─> Response: { id: "draft-123" }
            │           └─> onDraftCreated("draft-123")
            │           └─> Toast: "Draft created"
            └─> Keep watching for changes

2000ms: User edits condition again
        └─> schedulePersist() called
            └─> Timer set for 1000ms

3000ms: No changes for 1000ms
        └─> persistNow() executed
            ├─> ruleId = "draft-123" (from previous)
            ├─> UPDATE_RULE_BY_PK mutation
            ├─> Response: { id: "draft-123" }
            └─> Toast: "Rule autosaved"

[User navigates away]
3500ms: Component unmount triggered
        └─> useEffect cleanup called
            └─> Check if save pending
                └─> YES → Flush persistNow()
                └─> Final save executed
```

## Condition Tree JSON Structure

### Simple Condition
```json
{
  "id": "root",
  "type": "group",
  "operator": "AND",
  "conditions": [
    {
      "id": "cond_1",
      "field": "age",
      "operator": "greater_equal",
      "value": "18",
      "fieldType": "number"
    }
  ]
}
```

### Nested Groups
```json
{
  "id": "root",
  "type": "group",
  "operator": "AND",
  "conditions": [
    {
      "id": "cond_1",
      "field": "age",
      "operator": "greater_equal",
      "value": "18",
      "fieldType": "number"
    },
    {
      "id": "group_1",
      "type": "group",
      "operator": "OR",
      "conditions": [
        {
          "id": "cond_2",
          "field": "status",
          "operator": "equals",
          "value": "Active",
          "fieldType": "string"
        },
        {
          "id": "cond_3",
          "field": "is_vip",
          "operator": "is_true",
          "value": "true",
          "fieldType": "boolean"
        }
      ]
    }
  ]
}
```

## Evaluation Logic Flow

### Single Condition
```
Data: { age: 25 }
Condition: { field: "age", operator: "greater_equal", value: "18" }

Evaluate:
  Get fieldValue = data["age"] = 25
  Get compareValue = "18"
  Check operator: "greater_equal"
  Return: 25 >= 18 = TRUE ✓
```

### AND Group (Both must be true)
```
Data: { age: 25, status: "Active" }
Conditions:
  1. { field: "age", operator: ">=", value: "18" } → TRUE
  2. { field: "status", operator: "=", value: "Active" } → TRUE

Evaluate AND:
  results = [TRUE, TRUE]
  Return: all TRUE = TRUE ✓
```

### OR Group (Any can be true)
```
Data: { age: 25, status: "Inactive" }
Conditions:
  1. { field: "age", operator: ">=", value: "18" } → TRUE
  2. { field: "status", operator: "=", value: "Active" } → FALSE

Evaluate OR:
  results = [TRUE, FALSE]
  Return: any TRUE = TRUE ✓
```

### Complex Nested (AND with OR)
```
Data: { age: 25, status: "Active", is_vip: false }
Structure:
  Root AND [
    Condition: age >= 18 → TRUE
    Group OR [
      Condition: status = Active → TRUE
      Condition: is_vip = true → FALSE
    ] → TRUE (because OR)
  ]

Evaluate:
  age >= 18 = TRUE
  (status = Active OR is_vip = true) = (TRUE OR FALSE) = TRUE
  ROOT AND = (TRUE AND TRUE) = TRUE ✓
```

## Error States & Handling

### Missing Tenant Scope
```
User clicks "Add Condition" without selecting tenant
↓
Autosave triggers
↓
persistNow() checks: tenant ? datasource ?
↓
Both null
↓
Toast: ⚠️ "Select a tenant & datasource to persist visual rule"
↓
Save skipped (graceful fallback)
```

### Network Error with Retry
```
Save attempt 1 → Network error ✕
  └─ Wait 200ms
Save attempt 2 → Network error ✕
  └─ Wait 400ms
Save attempt 3 → Network error ✕
  └─ Wait 800ms
Save attempt 4 → Exceeded max retries
  └─ Toast: ❌ "Failed to persist rule. Please check your network."
```

### Successful Retry
```
Save attempt 1 → Network error ✕
  └─ Wait 200ms
Save attempt 2 → Success ✓
  └─ Toast: ✅ "Rule autosaved"
  └─ Stop retrying
```

## Keyboard Navigation

```
Tab:          Move to next interactive element
Shift+Tab:    Move to previous interactive element
Enter:        Activate button (Add Condition, Add Group, etc.)
Space:        Toggle AND/OR operator button
Delete:       Delete condition (if focused on delete button)
Escape:       Close any expanded details
```

## Mobile Responsive Behavior

### Desktop (3-column grid)
```
┌─────────────────────────────────────┐
│ [Field ▼]  [Operator ▼]  [Value ___] │
│ [Age ▼]    [>= ▼]        [18     ]   │
└─────────────────────────────────────┘
```

### Tablet (1-column stack)
```
┌──────────────────────┐
│ [Field ▼]            │
│ [Age ▼]              │
├──────────────────────┤
│ [Operator ▼]         │
│ [>= ▼]               │
├──────────────────────┤
│ [Value ___]          │
│ [18        ]         │
└──────────────────────┘
```

## Color Scheme

| Element | Color | Hex |
|---------|-------|-----|
| **Background** | White | #ffffff |
| **Border** | Gray | #d1d5db |
| **Border Hover** | Light Blue | #60a5fa |
| **AND Button** | Blue | #1e40af |
| **OR Button** | Orange | #ea580c |
| **Add Button** | Blue | #3b82f6 |
| **Add Group** | Purple | #a855f7 |
| **Delete** | Red | #dc2626 |
| **Success** | Green | #10b981 |
| **Warning** | Yellow | #f59e0b |
| **Error** | Red | #ef4444 |
| **Text Primary** | Dark Gray | #111827 |
| **Text Secondary** | Medium Gray | #6b7280 |

## Icon Reference

- ✋ Drag Handle
- ▼ Collapse
- ▶ Expand
- + Add
- ✕ Delete
- ✓ Success
- ✕ Error/Fail
- ? Help
- ⚠️ Warning
- ℹ️ Info

---

This visual reference guide helps developers and users understand the component's behavior and layout.
