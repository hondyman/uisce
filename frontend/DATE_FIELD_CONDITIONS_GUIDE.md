# Date Field Conditions Guide

## Your Employee Rule Example

Your rule shows:
```json
{
  "field": "birth_date",
  "fieldType": "date",
  "operator": "is_not_empty",
  "value": ""
}
```

### What's Fixed

✅ **Value field now correctly hidden** - When operator is `is_empty` or `is_not_empty`, no value needed

✅ **Relative dates now available** - Date fields show `Relative Dates` as an operator option

✅ **Proper date picker** - For `equals/before/after` operators, you get a native calendar widget

## Date Field Operators

### 1. State Checks (No Value Needed)

```
Operator: Is Empty
Result: Field has no date value
Value: (field hidden - not needed)

Operator: Is Not Empty  
Result: Field has any date value
Value: (field hidden - not needed)
```

**Why the value field is hidden:**
- These operators only check if a date EXISTS or NOT
- They don't compare to any specific date
- Showing a value field is confusing

**Your rule example:**
```
Operator: is_not_empty
Meaning: Birth date is populated/not null
Value: (automatically hidden - correct!)
```

---

### 2. Specific Date Comparison

```
Operator: Equals
Pick Date: [calendar picker]
Example: Equals 2000-01-15
Result: Birth date matches exactly

Operator: Before
Pick Date: [calendar picker]
Example: Before 2005-12-31
Result: Birth date is earlier than this date

Operator: After
Pick Date: [calendar picker]
Example: After 1990-01-01
Result: Birth date is later than this date
```

**Value field:** Shows native date picker

---

### 3. Relative Dates (Advanced)

```
Operator: Relative Dates
Enter Expression: [text field]
Examples:
  - "last 7 days"
  - "this month"
  - "last 30 days"
  - "this quarter"
  - "today"
Result: Dynamically calculates date range based on when rule runs
```

**Why this is useful:**
- Rule adapts over time
- Example: "last 30 days" always means the last 30 calendar days
- No need to update rule when dates change

---

### 4. Advanced Expressions

```
Operator: Advanced Expressions
Enter Expression: [text field]
Examples:
  - ">=2000-01-01 AND <=2010-12-31"
  - "%2020%"  (contains 2020)
  - "NOT %2021%"
Result: Looker-style filter expressions
```

---

## Complete Workflow for Date Rules

### Scenario 1: Check if Birth Date is Populated

```
Step 1: Select Field
  Field: birth_date
  📊 Data Type: date ✓

Step 2: Select Operator
  Operator: [Is Not Empty ▼]
  Help: State check (no value needed)

Step 3: Value Field
  ✓ Automatically hidden
  Message: "Operator 'is_not_empty' checks field state only — no value needed"

Result: Rule validates that birth_date is not null/empty
```

### Scenario 2: Check Birth Date is Before Specific Date

```
Step 1: Select Field
  Field: birth_date
  📊 Data Type: date ✓

Step 2: Select Operator
  Operator: [Before ▼]

Step 3: Value Field
  📅 Calendar picker appears
  Click: Select a date (e.g., 1970-01-01)

Result: Rule validates birth_date is before 1970-01-01
```

### Scenario 3: Check Birth Date in Last N Days

```
Step 1: Select Field
  Field: birth_date
  📊 Data Type: date ✓

Step 2: Select Operator
  Operator: [Relative Dates ▼]

Step 3: Value Field
  Text area shows hints:
    "Examples: last 7 days, this month, today, last 30 days, this quarter"
  
  Enter: "last 30 days"

Result: Rule validates birth_date is within last 30 calendar days
```

---

## Why Value Field Shows/Hides

### Hidden ✓ (No Input Needed)
- `is_empty` → Checking if field is null
- `is_not_empty` → Checking if field has any value
- `is_true` (boolean) → Field is true
- `is_false` (boolean) → Field is false

**Reason:** These check state, not compare to a value

### Shown 📝 (Input Required)
- `equals` → Must specify comparison date
- `before` → Must specify cutoff date
- `after` → Must specify cutoff date
- `contains` → Must specify pattern
- `relative_dates` → Must enter relative expression
- `expressions` → Must enter Looker expression

**Reason:** These need a value to compare against

---

## Your Employee DOB Rule

### Current Setup
```json
{
  "rule_name": "Employee date of birth cannot be empty",
  "target_entity": "employee",
  "condition": {
    "field": "birth_date",
    "fieldType": "date",
    "operator": "is_not_empty",
    "value": ""
  }
}
```

### What It Does
✅ Validates that every employee record has a `birth_date` value  
✅ Rejects records where birth_date is NULL or empty  
✅ Runs automatically on any employee data update

### Why No Value Field Needed
- Operator `is_not_empty` only checks: "Does this field exist?"
- No comparison needed → No value field shown
- The empty `""` value is automatically handled

---

## Common Questions

### Q: I select a date operator but don't see a calendar picker
**A:** Make sure:
1. ✓ Field type shows as `date` in the blue debug panel
2. ✓ Operator is `equals`, `before`, or `after` (not relative_dates)
3. ✓ Refresh the page and try again

### Q: Why does my `is_empty` rule still show a value field?
**A:** 
- In the new version, it shouldn't!
- If you see it: refresh your browser
- The field shows hint: "no value needed"

### Q: How do I check "birthdate is in the past"?
**A:** Use `before` operator with today's date, or use Relative Dates:
- `Operator: Relative Dates`
- `Value: last 50 years` (approx for adult employees)

### Q: Can I check "birthdate is between two dates"?
**A:** Use Advanced Expressions:
- `Operator: Advanced Expressions`
- `Value: >=1970-01-01 AND <=2005-12-31`

---

## Date Operator Reference

| Operator | Input Type | Example | Use Case |
|----------|-----------|---------|----------|
| Is Empty | None | - | Field is NULL/empty |
| Is Not Empty | None | - | Field has a value (your rule) |
| Equals | Date picker | 2000-01-15 | Exact date match |
| Before | Date picker | 2005-12-31 | Earlier than date |
| After | Date picker | 1990-01-01 | Later than date |
| Relative Dates | Text | last 7 days | Dynamic date range |
| Advanced Expressions | Text | >=1990 AND <=2000 | Complex comparisons |

---

## What Changed

### Before
- Date fields showed generic text input
- Relative date option wasn't clearly marked
- Value field didn't hide for `is_empty`/`is_not_empty`
- Calendar widget couldn't be used

### After ✅
- `is_empty`/`is_not_empty` → Value field completely hidden
- `equals`/`before`/`after` → Native browser calendar picker
- `relative_dates` → Text field with helpful examples
- `expressions` → Text field for Looker-style filters
- Better visual indicators for each operator type

---

## Quick Reference: Your Rule

**Employee DOB Cannot Be Empty**

```
What it checks: Is the birth_date field populated?
Operator: is_not_empty
Why no value: Stateless operator - just checking if field exists
Expected: Rejects employee records with NULL birth_date
```

✅ **This is working correctly** - no value field needed for state checks!
