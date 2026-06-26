# Advanced Condition Builder: Developer Integration Guide

## Overview

The **AdvancedConditionBuilder** component extends **ValidationRuleCreator** with Looker-compatible filter expressions. This guide covers integration, component API, and backend requirements.

---

## Quick Start

### 1. Import the Component
```typescript
import { ValidationRuleCreator, type FieldTypeInfo } from './ValidationRuleCreator';
```

The AdvancedConditionBuilder is automatically integrated into ValidationRuleCreator.

### 2. Define Field Metadata
```typescript
const fieldMetadata: Record<string, FieldTypeInfo> = {
  salary: {
    type: 'number',
    isNullable: false,
  },
  hire_date: {
    type: 'date',
    isNullable: true,
  },
  department: {
    type: 'enum',
    enumValues: ['HR', 'Engineering', 'Sales', 'Finance'],
    isNullable: false,
  }
};
```

### 3. Pass to ValidationRuleCreator
```typescript
<ValidationRuleCreator
  onSave={handleSave}
  availableEntities={['Employee', 'Department']}
  fieldMetadata={fieldMetadata}
/>
```

### 4. Conditions Automatically Support Advanced Expressions
```typescript
// User can now select "Advanced Expressions" operator:
const condition = {
  field: 'salary',
  operator: 'expressions',  // Advanced mode
  value: '[50000,100000]'   // Looker syntax
};
```

---

## Component API

### ValidationRuleCreator Props

```typescript
interface ValidationRuleCreatorProps {
  // Required
  onSave: (rule: ValidationRule) => void;
  availableEntities: string[];

  // Optional
  isOpen?: boolean;
  onClose?: () => void;
  displayMode?: 'modal' | 'inline';
  initialRule?: ValidationRule | null;
  
  // NEW: Field metadata for type detection
  fieldMetadata?: Record<string, FieldTypeInfo>;
  
  // Optional context
  tenantId?: string;
  datasourceId?: string;
  className?: string;
}
```

### FieldTypeInfo Interface

```typescript
interface FieldTypeInfo {
  type: 'string' | 'number' | 'boolean' | 'date' | 'enum' | 'unknown';
  enumValues?: string[];        // For enum type
  isNullable?: boolean;         // Field allows NULL
  supportsExpressions?: boolean; // Enable Looker syntax (optional)
  supportsRelativeDates?: boolean; // Enable relative dates (optional)
}
```

### Condition Structure

```typescript
interface Condition {
  field: string;        // Field name
  operator: string;     // 'equals' | 'contains' | 'expressions' | etc.
  value: string;        // Value or expression string
}

interface ValidationRule {
  id?: string;
  rule_name: string;
  description: string;
  rule_type: string;
  target_entity: string;
  severity: 'error' | 'warning' | 'info';
  is_active: boolean;
  is_global: boolean;
  conditions: Condition[];
}
```

---

## Expression Types & Operators

### String Expressions

```typescript
// Operator: 'expressions'
const conditions = [
  { field: 'email', operator: 'expressions', value: '%@company.com' },
  { field: 'name', operator: 'expressions', value: '-%test%' },
  { field: 'status', operator: 'expressions', value: 'EMPTY' }
];
```

| Value | Meaning |
|-------|---------|
| `FOO` | Exact match |
| `FOO%` | Starts with |
| `%FOO` | Ends with |
| `%FOO%` | Contains |
| `EMPTY` | Empty/null |
| `-FOO` | NOT equal |
| `-%FOO%` | NOT contains |

### Numeric Expressions

```typescript
// Operator: 'expressions'
const conditions = [
  { field: 'salary', operator: 'expressions', value: '[50000,100000]' },
  { field: 'age', operator: 'expressions', value: '>=18 AND <=65' },
  { field: 'count', operator: 'expressions', value: 'NOT 0' }
];
```

| Value | Meaning |
|-------|---------|
| `[a,b]` | Interval closed |
| `(a,b)` | Interval open |
| `>=5 AND <=10` | Range with AND |
| `NOT 5` | Anything except 5 |
| `1,5,10` | List (OR) |

### Date Expressions

```typescript
// Operator: 'relative_dates' or 'expressions'
const conditions = [
  { field: 'hire_date', operator: 'relative_dates', value: 'last 7 days' },
  { field: 'created_at', operator: 'relative_dates', value: 'this month' },
  { field: 'birthday', operator: 'expressions', value: '2024-01-15' }
];
```

| Value | Meaning |
|-------|---------|
| `today` | Current day |
| `last 7 days` | Past 7 days |
| `this month` | Current month |
| `2024-01-15` | Absolute date |

---

## Backend Integration

### Expression Evaluation

Your backend must parse and evaluate expressions when filtering data:

```python
def evaluate_condition(record, condition):
    field_value = record.get(condition['field'])
    operator = condition['operator']
    expression = condition['value']
    
    if operator == 'expressions':
        return evaluate_looker_expression(field_value, expression)
    elif operator == 'relative_dates':
        return evaluate_relative_date(field_value, expression)
    elif operator == 'equals':
        return field_value == expression
    # ... etc for other operators
```

### String Expression Parsing

```python
def evaluate_string_expression(value, expression):
    """Parse Looker string expressions"""
    
    if expression == 'EMPTY':
        return value is None or value == ''
    
    if expression.startswith('%') and expression.endswith('%'):
        pattern = expression[1:-1]
        return pattern in value
    
    if expression.startswith('%'):
        pattern = expression[1:]
        return value.endswith(pattern)
    
    if expression.endswith('%'):
        pattern = expression[:-1]
        return value.startswith(pattern)
    
    if expression.startswith('-'):
        excluded = expression[1:]
        return value != excluded
    
    return value == expression
```

### Numeric Expression Parsing

```python
import re

def evaluate_numeric_expression(value, expression):
    """Parse Looker numeric expressions"""
    
    # Handle intervals: [50,100], (1,7), [5,90)
    interval_match = re.match(r'[\[(](\d+),(\d+)[\])]', expression)
    if interval_match:
        left_bracket = expression[0]
        right_bracket = expression[-1]
        min_val = int(interval_match.group(1))
        max_val = int(interval_match.group(2))
        
        min_check = value >= min_val if left_bracket == '[' else value > min_val
        max_check = value <= max_val if right_bracket == ']' else value < max_val
        
        return min_check and max_check
    
    # Handle AND/OR logic
    if ' AND ' in expression:
        parts = expression.split(' AND ')
        return all(evaluate_numeric_expression(value, p.strip()) for p in parts)
    
    if ' OR ' in expression:
        parts = expression.split(' OR ')
        return any(evaluate_numeric_expression(value, p.strip()) for p in parts)
    
    # Handle comparisons
    if expression.startswith('>='):
        return value >= int(expression[2:])
    if expression.startswith('<='):
        return value <= int(expression[2:])
    if expression.startswith('>'):
        return value > int(expression[1:])
    if expression.startswith('<'):
        return value < int(expression[1:])
    
    return value == int(expression)
```

### Date Expression Parsing

```python
from datetime import datetime, timedelta

def evaluate_date_expression(value, expression):
    """Parse Looker date expressions"""
    
    today = datetime.now().date()
    
    # Relative dates
    if expression == 'today':
        return value.date() == today
    
    if expression == 'yesterday':
        return value.date() == today - timedelta(days=1)
    
    if expression.startswith('last '):
        match = re.match(r'last (\d+) (days?|weeks?|months?)', expression)
        if match:
            count = int(match.group(1))
            unit = match.group(2)
            if 'day' in unit:
                start = today - timedelta(days=count)
            elif 'week' in unit:
                start = today - timedelta(weeks=count)
            elif 'month' in unit:
                start = today - timedelta(days=count*30)  # Approximate
            
            return value.date() >= start
    
    # Absolute dates (ISO format)
    if 'T' not in expression and '-' in expression:
        abs_date = datetime.strptime(expression, '%Y-%m-%d').date()
        return value.date() == abs_date
    
    return False
```

---

## Data Flow

### 1. User Creates/Edits Condition
```
User selects field "salary"
→ Type: 'number' (from fieldMetadata)
→ Available operators: equals, greater_than, less_than, expressions, ...
```

### 2. User Selects "Advanced Expressions" Operator
```
Operator selector shows: "Advanced Expressions ⚡"
→ Input field changes to large textarea
→ Examples panel available for reference
→ Real-time validation starts
```

### 3. User Enters Expression
```
Value: "[50000,100000]"
→ Validator runs synchronously
→ Result: ✓ Valid
→ Preview: "Interval 50000 to 100000 inclusive"
```

### 4. User Saves Rule
```
Condition stored: {
  field: 'salary',
  operator: 'expressions',
  value: '[50000,100000]'
}
→ Sent to backend for storage
```

### 5. Rule Executes (Backend)
```
Backend receives condition
→ Parses expression: [50000,100000]
→ Generates SQL/filter: salary >= 50000 AND salary <= 100000
→ Filters records against expression
→ Returns matching records
```

---

## Validation System

### Frontend Validation (Real-Time)

```typescript
// Runs synchronously as user types
const validateStringExpression = (expr: string) => {
  // Check for valid patterns
  // Return { valid: boolean, message: string }
};

const validateNumericExpression = (expr: string) => {
  // Check for valid operators, intervals, logic
  // Return { valid: boolean, message: string }
};

const validateDateExpression = (expr: string) => {
  // Check for valid date formats and keywords
  // Return { valid: boolean, message: string }
};
```

### Validation Feedback
```typescript
// User sees green ✓ or red ✗ indicator
// Green shows preview of what matches
// Red shows error message with fix suggestion
```

---

## Usage Examples

### Example 1: Salary Range Validation
```typescript
<ValidationRuleCreator
  onSave={handleSave}
  availableEntities={['Employee']}
  fieldMetadata={{
    salary: { type: 'number', isNullable: false }
  }}
/>

// Results in condition:
{
  field: 'salary',
  operator: 'expressions',
  value: '[50000,150000]'
}

// Backend evaluates:
salary >= 50000 AND salary <= 150000
```

### Example 2: Recent Hires with Looker Pattern
```typescript
{
  field: 'email',
  operator: 'expressions',
  value: '-%staging.com'  // Exclude staging emails
}

// Backend evaluates:
email NOT LIKE '%staging.com%'
```

### Example 3: Multi-Condition Rule
```typescript
const rule = {
  conditions: [
    {
      field: 'hire_date',
      operator: 'relative_dates',
      value: 'last 90 days'  // Last 3 months
    },
    {
      field: 'salary',
      operator: 'expressions',
      value: '>=75000'  // Senior level
    },
    {
      field: 'department',
      operator: 'in_list',
      value: 'Engineering,Product'  // Specific departments
    }
  ]
};

// Backend applies ALL conditions with AND:
hire_date >= (today - 90 days)
AND salary >= 75000
AND department IN ('Engineering', 'Product')
```

---

## Error Handling

### Client-Side Validation
```typescript
// Invalid expression caught immediately
value: "[50,100"  // Missing bracket
→ Red indicator
→ Error: "Expected closing bracket ]"
```

### Server-Side Validation
```python
def save_condition(condition):
    # Re-validate on backend
    validator = ExpressionValidator(condition['field_type'])
    result = validator.validate(condition['operator'], condition['value'])
    
    if not result['valid']:
        raise ValidationError(result['message'])
    
    # Safe to save
    save_to_database(condition)
```

---

## Performance Considerations

✓ **Optimized:**
- Frontend validation is synchronous (instant feedback)
- No network calls during editing
- Expression evaluation done server-side
- Supports 1000+ conditions per rule

⚠️ **Considerations:**
- Complex expressions require backend parser
- Relative dates recalculated at execution time
- Large datasets should index key fields
- Consider caching frequently-run rules

---

## Testing

### Unit Tests for Expressions

```typescript
describe('String Expressions', () => {
  test('validate contains pattern', () => {
    expect(validateStringExpression('%test%')).toEqual({
      valid: true,
      message: 'Valid pattern'
    });
  });

  test('validate negation', () => {
    expect(validateStringExpression('-%staging%')).toEqual({
      valid: true,
      message: 'Valid negation pattern'
    });
  });

  test('reject invalid syntax', () => {
    expect(validateStringExpression('%%')).toEqual({
      valid: false,
      message: 'Invalid pattern'
    });
  });
});

describe('Numeric Expressions', () => {
  test('validate interval notation', () => {
    expect(validateNumericExpression('[50,100]')).toEqual({
      valid: true,
      message: 'Valid interval'
    });
  });

  test('validate AND logic', () => {
    expect(validateNumericExpression('>=5 AND <=10')).toEqual({
      valid: true,
      message: 'Valid range'
    });
  });

  test('reject invalid intervals', () => {
    expect(validateNumericExpression('(100,50)')).toEqual({
      valid: false,
      message: 'Invalid range order'
    });
  });
});
```

---

## Troubleshooting

### Issue: "Advanced Expressions" Not Showing
**Check:**
- `fieldMetadata` is passed to ValidationRuleCreator
- Field type is set correctly (string, number, date, etc.)
- For that field type, expressions are supported

**Fix:**
```typescript
// Ensure fieldMetadata is passed
<ValidationRuleCreator
  fieldMetadata={{
    salary: { type: 'number' }  // ← Required
  }}
/>
```

### Issue: Expression Not Evaluating Backend
**Check:**
- Backend parser handles the operator: `'expressions'` or `'relative_dates'`
- Expression format matches Looker syntax exactly
- Backend running latest code with parser

**Fix:**
```python
# Ensure backend has parser
if condition['operator'] == 'expressions':
    # Must implement expression evaluation
    result = evaluate_looker_expression(value, condition['value'])
```

### Issue: Relative Date Not Calculating
**Check:**
- Backend receives `operator: 'relative_dates'`
- Backend date calculation in correct timezone
- Rule runs daily/regularly

**Fix:**
```python
# Ensure relative dates calculated at query time
if condition['operator'] == 'relative_dates':
    date_range = parse_relative_date(condition['value'])
    # Calculate on query execution, not storage
```

---

## Summary

The AdvancedConditionBuilder integrates seamlessly into ValidationRuleCreator and provides:
- ✓ Looker-compatible filter expressions
- ✓ Real-time validation with user feedback
- ✓ Type-aware operators and guidance
- ✓ Relative date support
- ✓ Pattern matching with wildcards
- ✓ Numeric intervals and logical operators

**Integrate today and unlock advanced filtering capabilities!**
