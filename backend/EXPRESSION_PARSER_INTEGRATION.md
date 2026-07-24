# Expression Parser & Rule Executor: Integration Guide

## Overview

The expression parser and rule executor provide enterprise-grade validation for your business processes. They evaluate Looker-compatible filter expressions in real-time during data operations.

---

## Components

### 1. **expression_parser.ts** (~800 lines)
Evaluates individual conditions and expressions:
- `evaluateStringExpression()` - Looker patterns (%,-, etc.)
- `evaluateNumericExpression()` - Intervals, AND/OR logic
- `evaluateDateExpression()` - Relative and absolute dates
- `evaluateCondition()` - Route to appropriate evaluator
- `evaluateAllConditions()` - Multiple conditions with AND logic

### 2. **validation_rule_executor.ts** (~400 lines)
Orchestrates rule execution in business processes:
- `executeRule()` - Single rule against record
- `executeRules()` - Multiple rules against record
- `validateRecordBeforeWrite()` - Pre-database validation
- `validateRecordsBatch()` - Batch operation validation
- `filterRecordsByRules()` - Post-query validation
- `evaluateRuleWithDetails()` - Debugging support

---

## Architecture

```
Business Process
    ↓
validateRecordBeforeWrite() / executeRules()
    ↓
Rule Filter (by entity)
    ↓
For each rule:
  - evaluateAllConditions()
    ↓
    For each condition:
      - evaluateCondition()
        ↓
        evaluateStringExpression()
        evaluateNumericExpression()
        evaluateDateExpression()
        ↓
        Returns: { valid, matches, message }
```

---

## Usage Patterns

### Pattern 1: Pre-Insert Validation

**Scenario:** Validate employee record before INSERT

```typescript
import {
  validateRecordBeforeWrite,
  ValidationRule,
  FieldType
} from './validation_rule_executor';

// In your create() method:
async function createEmployee(data: Record<string, any>) {
  // Load applicable rules
  const rules = await loadRulesForEntity('Employee');
  
  // Define field schema
  const fieldTypes: Record<string, FieldType> = {
    email: 'string',
    salary: 'number',
    hire_date: 'date',
    is_active: 'boolean'
  };

  // Validate before INSERT
  const validation = validateRecordBeforeWrite(data, 'Employee', rules, fieldTypes);

  if (!validation.valid) {
    // Return validation errors to caller
    return {
      success: false,
      errors: validation.errors.map(v => ({
        message: v.message,
        severity: v.severity,
        conditions: v.violatedConditions
      }))
    };
  }

  // Safe to insert
  await db.employees.insert(data);
  return { success: true, id: result.id };
}
```

### Pattern 2: Batch Insert with Filtering

**Scenario:** Import 10,000 employees, validate and separate valid/invalid

```typescript
import { validateRecordsBatch } from './validation_rule_executor';

async function importEmployees(csvData: Record<string, any>[]) {
  const rules = await loadRulesForEntity('Employee');
  const fieldTypes = getFieldTypes('Employee');

  // Validate all records at once
  const { validRecords, invalidRecords } = validateRecordsBatch(
    csvData,
    'Employee',
    rules,
    fieldTypes
  );

  // Insert valid records
  if (validRecords.length > 0) {
    await db.employees.insertMany(validRecords);
    console.log(`✓ Inserted ${validRecords.length} valid records`);
  }

  // Report invalid records
  if (invalidRecords.length > 0) {
    const report = invalidRecords.map((item, idx) => ({
      row: idx + 2,  // +2 for header and 0-index
      record: item.record,
      errors: item.errors.map(e => e.message)
    }));
    
    console.warn(`✗ ${invalidRecords.length} records failed validation`);
    return { success: false, invalidRecords: report };
  }

  return { success: true, inserted: validRecords.length };
}
```

### Pattern 3: Pre-Update Validation

**Scenario:** Validate employee salary update

```typescript
async function updateEmployeeSalary(id: string, newSalary: number) {
  // Load existing record
  const employee = await db.employees.findById(id);
  
  // Merge with new data
  const updated = { ...employee, salary: newSalary };

  // Validate before UPDATE
  const rules = await loadRulesForEntity('Employee');
  const fieldTypes = getFieldTypes('Employee');
  const validation = validateRecordBeforeWrite(updated, 'Employee', rules, fieldTypes);

  if (!validation.valid) {
    throw new ValidationError(validation.errors[0].message);
  }

  // Safe to update
  await db.employees.update(id, { salary: newSalary });
}
```

### Pattern 4: Post-Query Filtering

**Scenario:** Apply data quality checks to query results

```typescript
import { filterRecordsByRules } from './validation_rule_executor';

async function getEmployeesWithQualityCheck(department: string) {
  // Fetch from database
  const records = await db.employees
    .where({ department })
    .limit(1000)
    .fetch();

  // Apply validation rules to results
  const rules = await loadRulesForEntity('Employee');
  const fieldTypes = getFieldTypes('Employee');
  
  const { validRecords, violations } = filterRecordsByRules(
    records,
    'Employee',
    rules,
    fieldTypes,
    true  // Include violation details
  );

  console.log(`✓ ${validRecords.length} records passed validation`);
  console.log(`⚠ ${violations?.size || 0} records have quality issues`);

  return {
    data: validRecords,
    qualityReport: violations
  };
}
```

### Pattern 5: Detailed Condition Debugging

**Scenario:** Understand why a rule matched (for investigation)

```typescript
import { evaluateRuleWithDetails } from './validation_rule_executor';

// User wants to know why employee was flagged
const rule = await db.rules.findById('rule_123');
const employee = await db.employees.findById('emp_456');
const fieldTypes = getFieldTypes('Employee');

const analysis = evaluateRuleWithDetails(rule, employee, fieldTypes);

console.log('Rule:', rule.rule_name);
console.log('Matched:', analysis.ruleMatched);
console.log('Conditions:');
analysis.conditionResults.forEach(c => {
  console.log(`  • ${c.field} ${c.operator} "${c.value}"`);
  console.log(`    Matched: ${c.matched}`);
  console.log(`    Reason: ${c.reason}`);
});
```

### Pattern 6: Webhook Validation

**Scenario:** Validate external webhook data before processing

```typescript
import { executeRulesWithLogging } from './validation_rule_executor';

async function handleWebhookData(
  event: string,
  data: Record<string, any>,
  logger: winston.Logger
) {
  // Extract entity type from event
  const entity = extractEntityType(event);  // e.g., 'Employee'

  // Load rules for this entity
  const rules = await loadRulesForEntity(entity);
  const fieldTypes = getFieldTypes(entity);

  // Execute with logging
  const result = executeRulesWithLogging(
    rules,
    data,
    fieldTypes,
    {
      log: (level, message, context) => {
        logger[level](message, context);
      }
    }
  );

  if (!result.passed) {
    logger.warn('Webhook validation failed', {
      event,
      violations: result.violations
    });
    return { error: 'Validation failed' };
  }

  // Safe to process
  await processWebhookData(event, data);
  return { success: true };
}
```

---

## Real-World Examples

### Example 1: Employee Salary Review Process

```typescript
// Rule: Only employees with salary >= 50k can submit for review
const salaryRule: ValidationRule = {
  id: 'rule_salary_review',
  rule_name: 'Salary Review Eligibility',
  rule_type: 'business_logic',
  target_entity: 'Employee',
  severity: 'error',
  description: 'Employee must have salary of at least 50k',
  is_active: true,
  is_global: false,
  conditions: [
    {
      field: 'salary',
      operator: 'expressions',
      value: '>=50000'  // Looker expression
    }
  ]
};

// Usage:
const employee = {
  name: 'John Doe',
  salary: 75000,
  hire_date: '2020-01-15'
};

const fieldTypes = {
  salary: 'number' as FieldType,
  hire_date: 'date'
};

const validation = validateRecordBeforeWrite(
  employee,
  'Employee',
  [salaryRule],
  fieldTypes
);

if (validation.valid) {
  // Proceed with salary review
}
```

### Example 2: Data Import with Complex Validation

```typescript
// Rule: Import products only if:
// - Price between 1 and 10000
// - SKU doesn't contain "test" or "demo"
// - Created date within last 90 days
const importRule: ValidationRule = {
  id: 'rule_product_import',
  rule_name: 'Product Import Quality Gate',
  rule_type: 'field_format',
  target_entity: 'Product',
  severity: 'error',
  description: 'Products must meet quality criteria for import',
  is_active: true,
  is_global: false,
  conditions: [
    {
      field: 'price',
      operator: 'expressions',
      value: '[1,10000]'  // Interval: 1 to 10000
    },
    {
      field: 'sku',
      operator: 'expressions',
      value: '-%test%'  // Does NOT contain "test"
    },
    {
      field: 'created_date',
      operator: 'expressions',
      value: 'last 90 days'  // Relative date
    }
  ]
};

// Import and validate
const products = await parseCSV('products.csv');
const { validRecords, invalidRecords } = validateRecordsBatch(
  products,
  'Product',
  [importRule],
  getFieldTypes('Product')
);

// Results:
// Valid: Only products matching all conditions
// Invalid: Cheap items, test SKUs, old data
```

### Example 3: Real-Time Order Validation

```typescript
// Rule: Prevent orders for high-value items from certain regions
const orderRule: ValidationRule = {
  id: 'rule_order_fraud_check',
  rule_name: 'High-Value Order Fraud Check',
  rule_type: 'business_logic',
  target_entity: 'Order',
  severity: 'warning',  // Warning only (allow but track)
  description: 'Flagging high-value orders from high-risk regions',
  is_active: true,
  is_global: true,
  conditions: [
    {
      field: 'order_amount',
      operator: 'expressions',
      value: '>=5000'  // Order >= $5000
    },
    {
      field: 'country',
      operator: 'expressions',
      value: '-US,-CA'  // NOT USA or Canada (hypothetical risk)
    }
  ]
};

// Process order
async function processOrder(order: Order) {
  const validation = validateRecordBeforeWrite(
    order,
    'Order',
    [orderRule],
    getFieldTypes('Order')
  );

  if (!validation.valid && validation.errors[0].severity === 'warning') {
    // Flag for manual review but allow processing
    await flagOrderForReview(order.id, validation.errors);
  } else if (!validation.valid) {
    // Hard error - reject order
    throw new OrderRejectedError(validation.errors[0].message);
  }

  // Process order normally
  await saveOrder(order);
}
```

---

## Performance Considerations

### Optimization Strategies

1. **Cache Rules**
```typescript
// Load rules once, reuse for batch operations
const rules = await loadRulesForEntity('Employee');
const fieldTypes = getFieldTypes('Employee');

// Validate 10,000 records with cached rules
const batch = largeDataset.map((record, idx) => ({
  index: idx,
  record,
  validation: validateRecordBeforeWrite(record, 'Employee', rules, fieldTypes)
}));
```

2. **Batch Operations**
```typescript
// Use batch validation instead of loop
const { validRecords, invalidRecords } = validateRecordsBatch(
  records,  // All 10,000 at once
  'Employee',
  rules,
  fieldTypes
);
```

3. **Index by Severity**
```typescript
// Focus on errors only
const errors = validation.errors.filter(e => e.severity === 'error');
const warnings = validation.errors.filter(e => e.severity === 'warning');

// Handle errors first, warnings optionally
if (errors.length > 0) throw new ValidationError(errors);
if (warnings.length > 0) logger.warn('Validation warnings', warnings);
```

---

## Error Handling

### Standard Pattern

```typescript
try {
  const result = executeRules(rules, record, fieldTypes);

  if (!result.passed) {
    // Handle violations
    const errorViolations = result.violations.filter(v => v.severity === 'error');
    
    if (errorViolations.length > 0) {
      throw new ValidationError(
        `Validation failed: ${errorViolations.map(v => v.message).join('; ')}`
      );
    }

    // Warnings only
    logger.warn('Validation warnings', result.violations);
  }

  // Proceed with operation
  return result;
} catch (error) {
  logger.error('Validation engine error', error);
  throw error;
}
```

---

## Logging & Monitoring

### Integration with Logger

```typescript
class ValidationLogger implements ValidationLogger {
  constructor(private logger: winston.Logger) {}

  log(level: 'info' | 'warn' | 'error', message: string, context?: any) {
    this.logger[level](message, {
      timestamp: new Date().toISOString(),
      ...context
    });
  }
}

// Usage
const logger = new ValidationLogger(winstonLogger);

const result = executeRulesWithLogging(
  rules,
  record,
  fieldTypes,
  logger
);
```

### Metrics to Track

```typescript
// Track validation metrics
const metrics = {
  totalRecordsValidated: 0,
  recordsValid: 0,
  recordsInvalid: 0,
  violationsByRule: new Map<string, number>(),
  avgValidationTime: 0
};

// Update on each validation
const start = Date.now();
const result = executeRules(rules, record, fieldTypes);
const duration = Date.now() - start;

metrics.totalRecordsValidated++;
metrics.recordsValid += result.passed ? 1 : 0;
metrics.recordsInvalid += result.passed ? 0 : 1;

result.violations.forEach(v => {
  const count = metrics.violationsByRule.get(v.ruleId) || 0;
  metrics.violationsByRule.set(v.ruleId, count + 1);
});

metrics.avgValidationTime = (metrics.avgValidationTime + duration) / 2;
```

---

## Testing

### Unit Test Examples

```typescript
describe('Expression Parser', () => {
  describe('String Expressions', () => {
    it('matches contains pattern', () => {
      const result = evaluateStringExpression('hello world', '%world%');
      expect(result.matches).toBe(true);
    });

    it('matches negation pattern', () => {
      const result = evaluateStringExpression('hello', '-test');
      expect(result.matches).toBe(true);
    });
  });

  describe('Numeric Expressions', () => {
    it('validates intervals', () => {
      const result = evaluateNumericExpression(75, '[50,100]');
      expect(result.matches).toBe(true);
    });

    it('handles AND logic', () => {
      const result = evaluateNumericExpression(75, '>=50 AND <=100');
      expect(result.matches).toBe(true);
    });
  });

  describe('Date Expressions', () => {
    it('evaluates relative dates', () => {
      const today = new Date();
      const result = evaluateDateExpression(today, 'today');
      expect(result.matches).toBe(true);
    });
  });
});
```

---

## Deployment Checklist

- [ ] Expression parser deployed to backend
- [ ] Rule executor deployed to backend
- [ ] Integration tests passing
- [ ] Performance benchmarks acceptable (< 50ms per record)
- [ ] Logging configured
- [ ] Error handling in place
- [ ] Documentation updated
- [ ] Team trained on usage patterns

---

## Next Steps

1. **Integrate into Create/Update endpoints** - Use `validateRecordBeforeWrite()`
2. **Add to batch imports** - Use `validateRecordsBatch()`
3. **Post-query validation** - Use `filterRecordsByRules()`
4. **Webhooks** - Use `executeRulesWithLogging()`
5. **Monitoring** - Track validation metrics

---

**The expression parser is production-ready and can be deployed immediately!**
