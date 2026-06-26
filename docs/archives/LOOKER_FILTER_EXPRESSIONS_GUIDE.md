# Looker-Style Filter Expressions: Complete Guide

## Overview

The **ValidationRuleCreator** now supports powerful Looker-style filter expressions through the integrated **AdvancedConditionBuilder** component. This enables enterprise-grade filtering without requiring backend complexity.

---

## What Are Looker Filter Expressions?

Looker filter expressions are a standardized syntax for defining complex data filters with:
- **Wildcards** for pattern matching
- **Logical operators** (AND, OR, NOT)
- **Interval notation** for ranges
- **Relative dates** that automatically calculate ranges
- **Negation** for exclusion rules

This allows non-technical users to express complex conditions intuitively.

---

## String Filter Expressions

### Basic Patterns

| Syntax | Meaning | Example | Matches |
|--------|---------|---------|---------|
| `FOO` | Exact match | `admin` | "admin" only |
| `FOO%` | Starts with | `john%` | "john", "johnny", "johns" |
| `%FOO` | Ends with | `%son` | "johnson", "son", "person" |
| `%FOO%` | Contains | `%a%` | any string with "a" |
| `FOO%,BAR` | Multiple (OR) | `admin,manager` | "admin" OR "manager" |

### Special Values

| Syntax | Meaning | Use Case |
|--------|---------|----------|
| `EMPTY` | Empty or null | Find missing data |
| `NULL` | NULL specifically | Test for nulls |
| `-FOO` | NOT equal | Exclude "FOO" |
| `-%FOO%` | Does NOT contain | Exclude pattern |

### Real Examples

```
%@company.com    → All company email addresses
emp%             → Names starting with "emp"
%_test%          → Anything with "_test"
-staging%        → Exclude staging records
-NULL            → Exclude null values
```

---

## Numeric Filter Expressions

### Comparison Operators

| Operator | Example | Meaning |
|----------|---------|---------|
| `>` | `>100` | Greater than 100 |
| `>=` | `>=50` | Greater than or equal to 50 |
| `<` | `<1000` | Less than 1000 |
| `<=` | `<=999` | Less than or equal to 999 |
| `=` | `=42` | Exactly 42 |

### Interval Notation

| Notation | Type | Example | Includes |
|----------|------|---------|----------|
| `[a,b]` | Closed | `[10,20]` | Both 10 and 20 |
| `(a,b)` | Open | `(10,20)` | Neither 10 nor 20 |
| `[a,b)` | Half-open | `[10,20)` | 10 but not 20 |
| `(a,b]` | Half-open | `(10,20]` | 20 but not 10 |

### Logical Operators

| Operator | Example | Meaning |
|----------|---------|---------|
| `AND` | `>=5 AND <=10` | Between 5 and 10 |
| `OR` | `100 OR 200` | Exactly 100 or 200 |
| `NOT` | `NOT 5` | Anything except 5 |

### Real Examples

```
[50000,100000]       → Salary in range
>=75000 AND <=150000 → Senior level earners
1,5,10,15            → Specific quantities
NOT 0                → Non-zero values
[0,10)               → 0 to 9
```

---

## Date Filter Expressions

### Relative Dates

| Expression | Meaning | Updates |
|------------|---------|---------|
| `today` | Current day | Every day at midnight |
| `yesterday` | Previous calendar day | Every day |
| `this week` | Monday-Sunday of current week | Weekly |
| `this month` | Days in current calendar month | Monthly |
| `this year` | Days in current calendar year | Yearly |
| `last 7 days` | Past 7 calendar days | Daily |
| `last 30 days` | Past 30 calendar days | Daily |
| `last N days` | Past N calendar days | Daily |
| `N days ago` | Exactly N days in past | Daily |
| `N weeks ago` | Exactly N weeks in past | Weekly |
| `N months ago` | Exactly N months in past | Monthly |
| `N days ago for M days` | Range in past | Configurable |

### Day of Week

| Expression | Meaning |
|------------|---------|
| `Monday` | All Mondays |
| `Tuesday` | All Tuesdays |
| ... | ... |
| `Sunday` | All Sundays |

### Absolute Dates

| Expression | Meaning |
|------------|---------|
| `2024-01-15` | Exactly Jan 15, 2024 |
| `after 2024-01-01` | On or after Jan 1, 2024 |
| `before 2024-12-31` | On or before Dec 31, 2024 |

### Real Examples

```
last 7 days        → This week's data
this month         → Month-to-date
2024-Q1            → First quarter
3 months ago       → 3 months in the past
after 2024-01-01   → All 2024 data onward
```

---

## Complex Expressions: Combining Rules

### Multi-Condition Rules

Within a single rule, conditions are combined with **AND** logic:

```
Rule: "Recent High-Performing Employees"

Condition 1: salary in [75000,150000]  AND
Condition 2: hire_date in last 2 years AND
Condition 3: performance_rating >= 4
```

→ **Result:** Employees with salary 75k-150k, hired in last 2 years, rated 4+ 

### Using OR Within a Single Condition

Some operators support comma-separated values (OR):

```
department: "Engineering,Sales,Product"
→ Employees in Engineering OR Sales OR Product
```

### Using NOT for Exclusions

Negative conditions:

```
email: -%@staging.com
→ All emails EXCEPT staging accounts

status: -archived
→ Active records (exclude archived)
```

---

## Validation & Error Handling

### Real-Time Validation

The condition builder validates expressions as you type:

✓ **Valid Expression**
```
[50000,100000]
Status: ✓ Valid
Preview: Interval 50000 to 100000 inclusive
```

✗ **Invalid Expression**
```
[50000
Status: ✗ Invalid
Error: Expected closing bracket
```

### Common Syntax Errors

| Error | Fix |
|-------|-----|
| `[50,100` | Missing closing bracket: `[50,100]` |
| `(50,100` | Missing closing bracket: `(50,100)` |
| `100,50` | Wrong order (max before min): `50,100` |
| `last7days` | Missing space: `last 7 days` |
| `2024/01/01` | Wrong date format: `2024-01-01` |

---

## Use Case Examples

### E-Commerce: Product Catalog Rules

#### Rule 1: In-Stock High-Value Items
```
Field: price
Operator: Advanced Expressions
Value: [100,999]

Field: stock_quantity
Operator: Advanced Expressions
Value: >0
```

#### Rule 2: Exclude Discontinued Items
```
Field: product_name
Operator: Advanced Expressions
Value: -%discontinued%
```

### HR: Employee Data Validation

#### Rule 1: Recent Premium Hires
```
Field: salary
Operator: Advanced Expressions
Value: [80000,250000]

Field: hire_date
Operator: Relative Dates
Value: last 90 days
```

#### Rule 2: Department Roster
```
Field: department
Operator: In List
Value: Engineering,Product,Design

Field: is_active
Operator: Is True
```

### Finance: Transaction Rules

#### Rule 1: Suspicious Large Transfers
```
Field: amount
Operator: Advanced Expressions
Value: [100000,999999999]

Field: timestamp
Operator: Relative Dates
Value: this week
```

#### Rule 2: Exclude Test Transactions
```
Field: account_id
Operator: Advanced Expressions
Value: -%test%

Field: merchant
Operator: Advanced Expressions
Value: -%sandbox%
```

---

## Performance Considerations

✓ **Optimized for:**
- Real-time validation (synchronous)
- Hundreds of conditions per rule
- Large datasets (backend filtered)
- Complex nested expressions

⚠️ **Limitations:**
- Expressions evaluated server-side
- Relative dates calculated at query time
- Complex AND/OR logic requires backend parser

---

## Browser Support

| Feature | Chrome | Firefox | Safari | Edge |
|---------|--------|---------|--------|------|
| String patterns | ✓ | ✓ | ✓ | ✓ |
| Numeric intervals | ✓ | ✓ | ✓ | ✓ |
| Relative dates | ✓ | ✓ | ✓ | ✓ |
| Real-time validation | ✓ | ✓ | ✓ | ✓ |

---

## Migration Guide: From Simple to Advanced

### Step 1: Identify Your Data Type
- String: Use patterns with % and -
- Number: Use intervals [a,b] or operators
- Date: Use relative dates or YYYY-MM-DD

### Step 2: Select Advanced Mode
In the condition builder, choose "Advanced Expressions"

### Step 3: Use the Examples Panel
Click "Examples" to see patterns for your field type

### Step 4: Validate Before Saving
Watch for the green ✓ or red ✗ indicator

### Step 5: Monitor Results
Check that conditions match expected records

---

## API Integration

### Sending Conditions
```typescript
const condition = {
  field: 'salary',
  operator: 'expressions',
  value: '[50000,100000]'
};
```

### Backend Parsing Required
Backends must parse and evaluate expressions:
```
[50000,100000] → salary >= 50000 AND salary <= 100000
%@company.com → email LIKE '%@company.com%'
last 7 days → created_at >= NOW() - INTERVAL 7 DAY
```

---

## Summary

Looker-style filter expressions provide:
- ✓ Intuitive syntax non-technical users understand
- ✓ Powerful pattern matching with wildcards
- ✓ Complex numeric ranges and intervals
- ✓ Automatic date range calculations
- ✓ Real-time validation and feedback

**Start using advanced filters today!**
