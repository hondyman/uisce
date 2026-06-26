# Validation Rules Wizard - Visual Guide & Quick Reference

**Last Updated:** October 20, 2025

---

## 🎨 Visual Layout

```
┌─────────────────────────────────────────────────────────────┐
│                    Create Validation Rule              [×]  │ ← Close Button
├─────────────────────────────────────────────────────────────┤
│ Configure a new validation rule for your data              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ 📋        ⚙️        ⚠️        🔍                            │
│ Step 1    Step 2    Step 3    Step 4                       │
│ Basic     Config    Severity  Conditions                   │
│ ─────────────────────────────                             │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│ [Step Content Here]                                        │
│                                                             │
│ ...                                                         │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│ [ Cancel ]              [ Back ]  [ Next ] or [ Create ]   │
└─────────────────────────────────────────────────────────────┘
```

---

## 📋 Step 1: Basic Information

### Fields & Layout

```
ℹ️  Getting Started:
    Provide a clear name and description for your validation rule.

┌─ Rule Name *────────────────────────────────────┐
│ e.g., Employee ID Must Be Valid Format         │
└──────────────────────────────────────────────────┘

┌─ Description *──────────────────────────────────┐
│ Describe what this rule validates and why       │
│ it's important...                               │
│                                                  │
│                                                  │
└──────────────────────────────────────────────────┘
```

### Validation
- ✓ Rule name must not be empty
- ✓ Description must not be empty
- ✓ Cannot proceed without both fields

---

## ⚙️ Step 2: Configuration

### Rule Type Selection

```
┌─ Rule Type *────────────────────────────────────┐
│                                                  │
│ ┌─ 📝 Field Format ─────────────────────────┐   │
│ │ Validate data format and structure        │   │
│ └────────────────────────────────────────────┘   │
│                                                  │
│ ┌─ ⚙️ Business Logic ─────────────────────────┐ │
│ │ Enforce business rules and logic          │   │
│ └────────────────────────────────────────────┘   │
│                                                  │
│ ┌─ 📊 Cardinality ─────────────────────────────┐ │
│ │ Check required relationships              │   │
│ └────────────────────────────────────────────┘   │
│                                                  │
│ ┌─ 🔑 Uniqueness ───────────────────────────────┐ │
│ │ Ensure unique values                      │   │
│ └────────────────────────────────────────────┘   │
│                                                  │
│ ┌─ 🔗 Referential Integrity ────────────────────┐ │
│ │ Validate cross-entity references          │   │
│ └────────────────────────────────────────────┘   │
│                                                  │
└──────────────────────────────────────────────────┘

┌─ Target Entity * ─────────────────────────────┐
│ [ Select an entity... ▼ ]                      │
│   - Employee                                   │
│   - Department                                 │
│   - Position                                   │
│   - Account                                    │
└───────────────────────────────────────────────┘

┌─ Sub-Entity Type (Optional) ──────────────────┐
│ e.g., Address, Contact, etc.                  │
└───────────────────────────────────────────────┘
```

### Validation
- ✓ Rule type must be selected
- ✓ Target entity must be selected
- ✓ Sub-entity is optional

---

## ⚠️ Step 3: Severity & Scope

### Severity Levels

```
┌─ Severity Level *────────────────────────────┐
│                                              │
│ ┌─ 🔴 Error ──────────────────────────────┐ │
│ │ Blocks processing if violated        │ │
│ └────────────────────────────────────────┘ │
│                                              │
│ ┌─ 🟠 Warning ────────────────────────────┐ │
│ │ Allows processing with alert         │ │
│ └────────────────────────────────────────┘ │
│                                              │
│ ┌─ 🔵 Info ──────────────────────────────┐ │
│ │ Informational only                   │ │
│ └────────────────────────────────────────┘ │
│                                              │
└──────────────────────────────────────────────┘
```

### Scope Options

```
┌─ Rule Scope ─────────────────────────────────┐
│                                              │
│ ☑  🌍 Apply Globally                        │
│    This rule will apply to all instances    │
│    of the target entity across your         │
│    organization                             │
│                                              │
│ ☑  ✓ Active Rule                           │
│    Enable this rule immediately after      │
│    creation                                 │
│                                              │
└──────────────────────────────────────────────┘
```

### Validation
- ✓ Severity level must be selected
- ✓ Global and Active are optional checkboxes

---

## 🔍 Step 4: Conditions (Optional)

### Empty State

```
┌─ Validation Conditions ────────────────────┐
│                                            │
│      No conditions added yet              │
│                                            │
│   Click "Add Condition" to create a      │
│   specific validation rule               │
│                                            │
│                [ + Add Condition ]        │
│                                            │
└────────────────────────────────────────────┘
```

### With Conditions

```
Validation Conditions          [ + Add Condition ]

┌─ Condition 1 ────────────────────────────────┐
│ ┌─────────┐ ┌──────────┐ ┌──────────┐ [🗑]  │
│ │ field   │ │operator  │ │ value    │       │
│ │─────────│ │──────────│ │──────────│       │
│ │department│ │equals   │ │HR       │       │
│ └─────────┘ └──────────┘ └──────────┘       │
└───────────────────────────────────────────────┘

┌─ Condition 2 ────────────────────────────────┐
│ ┌─────────┐ ┌──────────┐ ┌──────────┐ [🗑]  │
│ │ field   │ │operator  │ │ value    │       │
│ │─────────│ │──────────│ │──────────│       │
│ │ salary  │ │greater   │ │50000    │       │
│ │         │ │_than     │ │          │       │
│ └─────────┘ └──────────┘ └──────────┘       │
└───────────────────────────────────────────────┘
```

### Available Operators

| Operator | Use Case |
|----------|----------|
| Equals | Exact match |
| Not Equals | Exclude value |
| Contains | Substring search |
| Starts With | Prefix match |
| Ends With | Suffix match |
| Greater Than | Numeric comparison |
| Less Than | Numeric comparison |
| Is Empty | Null/empty check |
| Is Not Empty | Required field |

---

## 🎯 Progress Indicators

### Step Indicators

```
Status          Visual          Appearance
─────────────────────────────────────────
Not Started     ①              Gray circle with number
Current         ②              Blue circle with number  
Completed       ✓              Green circle with checkmark
```

### Connectors

```
Status          Visual
─────────────────────
Not Completed   ─ (gray line)
Completed       ─ (green line)
```

### Full Progress Bar Example

```
📋  →  ⚙️  →  ⚠️  →  🔍
1   2  2   3  3   4  4
```

**Legend:**
- Number = Step number
- Arrow = Connector line
- Icon = Step type
- Color = Current state

---

## 🔘 Button States

### Navigation Buttons

```
Status              Display           Enabled?
─────────────────────────────────────────────
Step 1 Back         Hidden            N/A
Step 2+ Back        [Back]            Yes
Step 1-3 Next       [Next]            Yes (after validation)
Step 4 Submit       [✓ Create Rule]   Yes (after validation)
Cancel              [Cancel]          Always
```

### During Submission

```
Before:  [✓ Create Rule]     (clickable)
During:  [Creating...]        (disabled, loading)
After:   Modal closes, rule appears in list
```

---

## 📱 Mobile Responsive Behavior

### Small Screen (< 640px)

```
Full-width modal
Full-height modal
Stacked buttons
Single-column conditions

┌─────────────────┐
│   [×] Create    │
│   Validation... │
├─────────────────┤
│ 📋 ⚙️ ⚠️ 🔍     │
│ (Smaller icons) │
├─────────────────┤
│ [Step Content]  │
│ (Full width)    │
│                 │
├─────────────────┤
│   [Cancel]      │
│  [Next]         │
└─────────────────┘
```

### Large Screen (> 1024px)

```
┌─────────────────────────────────────────────┐
│     [×] Create Validation Rule              │
│ Configure a new validation rule for data    │
├─────────────────────────────────────────────┤
│ 📋        ⚙️        ⚠️        🔍            │
│ Basic     Config    Severity  Conditions   │
│ ─────────────────────────────────           │
├─────────────────────────────────────────────┤
│ [Step Content]                              │
│                                             │
│                                             │
├─────────────────────────────────────────────┤
│ [ Cancel ]      [ Back ]  [ Next ]          │
└─────────────────────────────────────────────┘
```

---

## 🎨 Color Reference

### UI Colors

| Element | Color | Hex Value |
|---------|-------|-----------|
| Header Background | Blue Gradient | #2563eb → #1d4ed8 |
| Primary Button | Blue | #2563eb |
| Success/Create | Green | #10b981 |
| Error/Alert | Red | #ef4444 |
| Warning | Orange | #f59e0b |
| Info | Blue | #3b82f6 |
| Selected/Active | Light Blue | #eff6ff |
| Background | Gray | #f9fafb |
| Border | Light Gray | #e5e7eb |
| Text | Dark Gray | #1f2937 |
| Muted Text | Gray | #6b7280 |

### Severity Indicators

| Severity | Color | RGB | Use |
|----------|-------|-----|-----|
| Error | Red | rgb(239, 68, 68) | Blocks processing |
| Warning | Orange | rgb(245, 158, 11) | Allows with alert |
| Info | Blue | rgb(59, 130, 246) | Informational |

---

## ✨ Animation & Transitions

### Entrance
- Modal slides up smoothly
- Duration: 300ms
- Easing: ease-out

### Step Transitions
- Progress indicators update smoothly
- Connectors animate from gray to green
- Duration: 300ms for each element

### Interactions
- Button hover: color change + slight shadow
- Input focus: blue outline ring
- Card selection: border + background change

---

## 📋 Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Tab | Move to next form field |
| Shift+Tab | Move to previous form field |
| Enter | Submit (on final step) |
| Escape | Close modal |
| Arrow Up/Down | Navigate select dropdown |
| Space | Toggle checkbox |

---

## 🔊 Screen Reader Announcements

```
Modal Opens:
"Dialog: Create Validation Rule. 
Configure a new validation rule for your data. 
List of 4 steps."

Step 1 Progress:
"Progress, step 1 of 4, Basic Information. 
Edit rule name, required. 
Edit description, required."

On Error:
"Error: Rule name is required. 
Please fill in this field."

On Completion:
"Success! Validation rule created. 
Modal closing."
```

---

## 🧪 Testing Scenarios

### Happy Path
1. Fill in rule name ✓
2. Fill in description ✓
3. Select rule type ✓
4. Select target entity ✓
5. Select severity ✓
6. Click "Create Rule" ✓
7. Rule appears in list ✓

### Error Handling
1. Try next without filling name → Error shown ✓
2. Fill in name, try next → Proceed ✓
3. Try next without entity → Error shown ✓
4. Fill all required fields → Create succeeds ✓

### Edge Cases
1. Very long rule name (100+ chars) → Truncates properly
2. Multi-line description → Wraps correctly
3. Special characters in conditions → Escaped properly
4. Rapid clicking create button → Single submission only

---

## 📊 Form Field Quick Reference

| Field | Type | Required | Max Length | Validation |
|-------|------|----------|-----------|------------|
| Rule Name | Text | Yes | 255 | Not empty |
| Description | Textarea | Yes | 2000 | Not empty |
| Rule Type | Select | Yes | N/A | One of 5 types |
| Target Entity | Select | Yes | N/A | From available list |
| Sub-Entity | Text | No | 255 | Any text |
| Severity | Select | Yes | N/A | error/warning/info |
| Global | Checkbox | No | N/A | true/false |
| Active | Checkbox | No | N/A | true/false |
| Field (Condition) | Text | If used | 255 | Not empty if condition |
| Operator (Condition) | Select | If used | N/A | Valid operator |
| Value (Condition) | Text | If used | 255 | Not empty if condition |

---

## 🚀 Quick Start for Users

### Creating Your First Rule (2 minutes)

1. **Click** "+ Add Rule" button
2. **Enter** rule name and description (Step 1)
3. **Select** rule type and target entity (Step 2)
4. **Choose** severity level (Step 3)
5. **Skip** conditions (Step 4) - optional
6. **Click** "Create Rule"
7. **Done!** Rule appears in your list

### Creating Advanced Rule (5 minutes)

1. Follow steps 1-3 above
2. **Add** conditions by clicking "+ Add Condition"
3. **Fill** in Field, Operator, and Value for each
4. **Add more** conditions by clicking again
5. **Click** "Create Rule"
6. **Done!** Complex rule is active

---

## 🆘 Troubleshooting Quick Guide

| Problem | Solution |
|---------|----------|
| Can't open modal | Check "+ Add Rule" button is visible |
| Can't submit | Verify all required fields filled in red |
| Error message | Read error text for specific field |
| Page not responding | Check network connectivity |
| CSS looks broken | Clear browser cache and refresh |
| Mobile layout wrong | Use latest browser version |

---

**Status:** 🟢 Complete and Ready for Use
