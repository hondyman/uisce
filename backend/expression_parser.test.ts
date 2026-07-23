import { describe, it, expect } from '@jest/globals';
import {
  evaluateStringExpression,
  evaluateNumericExpression,
  evaluateDateExpression,
  evaluateCondition,
  evaluateAllConditions
} from './expression_parser';

import {
  executeRule,
  executeRules,
  validateRecordBeforeWrite,
  validateRecordsBatch,
  filterRecordsByRules,
  evaluateRuleWithDetails
} from './validation_rule_executor';

import { ValidationRule, FieldType } from './validation_rule_executor';
import { ValidationRule, FieldType, ValidationViolation } from './validation_rule_executor';

describe('Expression Parser Tests', () => {
describe('Expression Parser Tests', () => { // eslint-disable-line
  describe('String Expression Evaluator', () => {
    it('should match contains pattern (%pattern%)', () => {
      const result = evaluateStringExpression('hello world', '%world%');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
      expect(result.message).toBe('String contains "world"');
    });

    it('should match starts with pattern (pattern%)', () => {
      const result = evaluateStringExpression('hello world', 'hello%');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should match ends with pattern (%pattern)', () => {
      const result = evaluateStringExpression('hello world', '%world');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should handle negation (-pattern)', () => {
      const result = evaluateStringExpression('hello', '-world');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should fail negation when pattern matches', () => {
      const result = evaluateStringExpression('hello', '-hello');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(false);
    });

    it('should match EMPTY value', () => {
      const result = evaluateStringExpression('', 'EMPTY');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should not match EMPTY for non-empty string', () => {
      const result = evaluateStringExpression('value', 'EMPTY');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(false);
    });

    it('should match NULL value', () => {
      const result = evaluateStringExpression(null, 'NULL');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should handle case-insensitive matching', () => {
      const result = evaluateStringExpression('Hello World', '%world%');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should handle complex negation (-%pattern%)', () => {
      const result = evaluateStringExpression('hello test', '-%test%');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(false);
    });
  });

  describe('Numeric Expression Evaluator', () => {
    it('should match closed interval [50,100]', () => {
      expect(evaluateNumericExpression(50, '[50,100]').matches).toBe(true);
      expect(evaluateNumericExpression(75, '[50,100]').matches).toBe(true);
      expect(evaluateNumericExpression(100, '[50,100]').matches).toBe(true);
    });

    it('should not match outside closed interval', () => {
      expect(evaluateNumericExpression(49, '[50,100]').matches).toBe(false);
      expect(evaluateNumericExpression(101, '[50,100]').matches).toBe(false);
    });

    it('should match open interval (50,100)', () => {
      expect(evaluateNumericExpression(50, '(50,100)').matches).toBe(false);
      expect(evaluateNumericExpression(75, '(50,100)').matches).toBe(true);
      expect(evaluateNumericExpression(100, '(50,100)').matches).toBe(false);
    });

    it('should match half-open intervals [50,100)', () => {
      expect(evaluateNumericExpression(50, '[50,100)').matches).toBe(true);
      expect(evaluateNumericExpression(100, '[50,100)').matches).toBe(false);
    });

    it('should match half-open intervals (50,100]', () => {
      expect(evaluateNumericExpression(50, '(50,100]').matches).toBe(false);
      expect(evaluateNumericExpression(100, '(50,100]').matches).toBe(true);
    });

    it('should handle comparison operators', () => {
      expect(evaluateNumericExpression(75, '>50').matches).toBe(true);
      expect(evaluateNumericExpression(75, '>=75').matches).toBe(true);
      expect(evaluateNumericExpression(75, '<100').matches).toBe(true);
      expect(evaluateNumericExpression(75, '<=75').matches).toBe(true);
    });

    it('should handle comma-separated list', () => {
      expect(evaluateNumericExpression(5, '1,5,10').matches).toBe(true);
      expect(evaluateNumericExpression(7, '1,5,10').matches).toBe(false);
    });

    it('should handle AND logic', () => {
      expect(evaluateNumericExpression(75, '>=50 AND <=100').matches).toBe(true);
      expect(evaluateNumericExpression(125, '>=50 AND <=100').matches).toBe(false);
    });

    it('should handle OR logic', () => {
      expect(evaluateNumericExpression(25, '<50 OR >100').matches).toBe(true);
      expect(evaluateNumericExpression(75, '<50 OR >100').matches).toBe(false);
    });

    it('should handle NOT logic', () => {
      expect(evaluateNumericExpression(25, 'NOT [50,100]').matches).toBe(true);
      expect(evaluateNumericExpression(75, 'NOT [50,100]').matches).toBe(false);
    });

    it('should handle complex expressions', () => {
      // (x > 50 AND x < 100) OR x > 500
      const result = evaluateNumericExpression(750, '>50 AND <100 OR >500');
      expect(result.valid).toBe(true);
    });
  });

  describe('Date Expression Evaluator', () => {
    it('should match today', () => {
      const today = new Date();
      const result = evaluateDateExpression(today, 'today');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should match yesterday', () => {
      const yesterday = new Date();
      yesterday.setDate(yesterday.getDate() - 1);
      const result = evaluateDateExpression(yesterday, 'yesterday');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should match this week', () => {
      const result = evaluateDateExpression(new Date(), 'this week');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should match this month', () => {
      const result = evaluateDateExpression(new Date(), 'this month');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should match relative days (7 days ago)', () => {
      const sevenDaysAgo = new Date();
      sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 7);
      const result = evaluateDateExpression(sevenDaysAgo, 'last 7 days');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should match day of week', () => {
      // Find a Monday
      const monday = new Date();
      const day = monday.getDay();
      const diff = monday.getDate() - day + (day === 0 ? -6 : 1);
      monday.setDate(diff);

      const result = evaluateDateExpression(monday, 'Monday');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should match absolute date (YYYY-MM-DD)', () => {
      const date = new Date('2024-01-15');
      const result = evaluateDateExpression(date, '2024-01-15');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should match after comparison', () => {
      const date = new Date('2024-02-01');
      const result = evaluateDateExpression(date, 'after 2024-01-15');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should match before comparison', () => {
      const date = new Date('2024-01-15');
      const result = evaluateDateExpression(date, 'before 2024-02-01');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should handle N days ago', () => {
      const threeDaysAgo = new Date();
      threeDaysAgo.setDate(threeDaysAgo.getDate() - 3);
      const result = evaluateDateExpression(threeDaysAgo, '3 days ago');
      expect(result.valid).toBe(true);
    });
  });

  describe('Condition Evaluator (Type Router)', () => {
    it('should route string conditions to string evaluator', () => {
      const result = evaluateCondition(
        'hello world',
        'string',
        'expressions',
        '%world%'
      );
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should route numeric conditions to numeric evaluator', () => {
      const result = evaluateCondition(75, 'number', 'expressions', '[50,100]');
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should route date conditions to date evaluator', () => {
      const result = evaluateCondition(
        new Date(),
        'date',
        'relative_dates',
        'today'
      );
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should handle type mismatch gracefully', () => {
      const result = evaluateCondition(
        'not a number',
        'number',
        'expressions',
        '[50,100]'
      );
      expect(result.valid).toBe(false);
    });
  });

  describe('All Conditions Evaluator (Batch AND)', () => {
    it('should pass when all conditions match', () => {
      const record = {
        salary: 75000,
        status: 'active',
        hire_date: new Date()
      };

      const fieldTypes: Record<string, FieldType> = {
        salary: 'number',
        status: 'string',
        hire_date: 'date'
      };

      const conditions = [
        { field: 'salary', operator: 'expressions', value: '[50000,150000]' },
        { field: 'status', operator: 'expressions', value: 'active' },
        { field: 'hire_date', operator: 'relative_dates', value: 'last 10 years' }
      ];

      const result = evaluateAllConditions(record, fieldTypes, conditions);
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(true);
    });

    it('should fail when any condition does not match', () => {
      const record = {
        salary: 30000,  // Below minimum
        status: 'active',
        hire_date: new Date()
      };

      const fieldTypes: Record<string, FieldType> = {
        salary: 'number',
        status: 'string',
        hire_date: 'date'
      };

      const conditions = [
        { field: 'salary', operator: 'expressions', value: '[50000,150000]' }
      ];

      const result = evaluateAllConditions(record, fieldTypes, conditions);
      expect(result.valid).toBe(true);
      expect(result.matches).toBe(false);
    });
  });
});

describe('Validation Rule Executor Tests', () => {
  const testRule: ValidationRule = {
    id: 'rule_1',
    rule_name: 'Test Rule',
    rule_type: 'field_format',
    target_entity: 'Employee',
    severity: 'error',
    description: 'Test rule',
    is_active: true,
    is_global: false,
    conditions: [
      {
        field: 'salary',
        operator: 'expressions',
        value: '[50000,150000]'
      }
    ]
  };

  const testRecord = {
    id: 'emp_1',
    name: 'John Doe',
    salary: 75000,
    email: 'john@example.com'
  };

  const fieldTypes: Record<string, FieldType> = {
    salary: 'number',
    email: 'string'
  };

  describe('Execute Rule', () => {
    it('should return true when rule conditions match', () => {
      const result = executeRule(testRule, testRecord, fieldTypes);
      expect(result.passed).toBe(true);
      expect(result.matched).toBe(true);
      expect(result.violations.length).toBe(0);
    });

    it('should return false when rule conditions do not match', () => {
      const record = { ...testRecord, salary: 30000 };
      const result = executeRule(testRule, record, fieldTypes);
      expect(result.passed).toBe(false);
      expect(result.matched).toBe(false);
      expect(result.violations.length).toBeGreaterThan(0);
    });

    it('should include rule details in result', () => {
      const result = executeRule(testRule, testRecord, fieldTypes);
      expect(result.summary).toContain('Test Rule');
    });
  });
  }); // eslint-disable-line

  describe('Execute Multiple Rules', () => {
    it('should pass all rules', () => {
      const rule2: ValidationRule = {
        ...testRule,
        id: 'rule_2',
        conditions: [
          {
            field: 'email',
            operator: 'expressions',
            value: '%@example.com'
          }
        ]
      };

      const result = executeRules([testRule, rule2], testRecord, fieldTypes);
      expect(result.passed).toBe(true);
      expect(result.violations.length).toBe(0);
      expect(result.executedRules).toBe(2);
    });

    it('should fail if any rule fails', () => {
      const rule2: ValidationRule = {
        ...testRule,
        id: 'rule_2',
        conditions: [
          {
            field: 'email',
            operator: 'expressions',
            value: '%@company.com'  // Wrong domain
          }
        ]
      };

      const result = executeRules([testRule, rule2], testRecord, fieldTypes);
      expect(result.passed).toBe(false);
      expect(result.violations.length).toBeGreaterThan(0);
    });
  });

  describe('Validate Record Before Write', () => {
    it('should return valid when rules pass', () => {
      const validation = validateRecordBeforeWrite(
        testRecord,
        'Employee',
        [testRule],
        fieldTypes
      );
      expect(validation.valid).toBe(true);
      expect(validation.errors.length).toBe(0);
    });

    it('should return invalid when rules fail', () => {
      const record = { ...testRecord, salary: 30000 };
      const validation = validateRecordBeforeWrite(
        record,
        'Employee',
        [testRule],
        fieldTypes
      );
      expect(validation.valid).toBe(false);
      expect(validation.errors.length).toBeGreaterThan(0);
    });

    it('should filter by target entity', () => {
      const rule = { ...testRule, target_entity: 'Manager' };
      const validation = validateRecordBeforeWrite(
        testRecord,
        'Employee',
        [rule],
        fieldTypes
      );
      // Should pass because rule doesn't apply to Employee
      expect(validation.valid).toBe(true);
    });
  });

  describe('Validate Records Batch', () => {
    it('should separate valid and invalid records', () => {
      const records = [
        { id: 'emp_1', salary: 75000, email: 'john@example.com' },
        { id: 'emp_2', salary: 30000, email: 'jane@example.com' },  // Invalid
        { id: 'emp_3', salary: 85000, email: 'bob@example.com' }
      ];

      const result = validateRecordsBatch(
        records,
        'Employee',
        [testRule],
        fieldTypes
      );

      expect(result.validRecords.length).toBe(2);
      expect(result.invalidRecords.length).toBe(1);
      expect(result.invalidRecords[0].record.id).toBe('emp_2');
    });

    it('should provide error details for invalid records', () => {
      const records = [
        { id: 'emp_1', salary: 30000, email: 'test@invalid.com' }
      ];

      const result = validateRecordsBatch(
        records,
        'Employee',
        [testRule],
        fieldTypes
      );

      expect(result.invalidRecords.length).toBe(1);
      expect(result.invalidRecords[0].errors.length).toBeGreaterThan(0);
      expect(result.invalidRecords[0].errors[0].message).toBeDefined();
    });
  });

  describe('Filter Records By Rules', () => {
    it('should return only valid records', () => {
      const records = [
        { id: 'emp_1', salary: 75000, email: 'john@example.com' },
        { id: 'emp_2', salary: 30000, email: 'jane@example.com' },  // Invalid
        { id: 'emp_3', salary: 85000, email: 'bob@example.com' }
      ];

      const result = filterRecordsByRules(
        records,
        'Employee',
        [testRule],
        fieldTypes
      );

      expect(result.validRecords.length).toBe(2);
      expect(result.validRecords.every(r => r.salary >= 50000)).toBe(true);
    });

    it('should include violation details when requested', () => {
      const records = [
        { id: 'emp_1', salary: 30000, email: 'test@test.com' }
      ];

      const result = filterRecordsByRules(
        records,
        'Employee',
        [testRule],
        fieldTypes,
        true  // includeViolations
      );

      expect(result.violations).toBeDefined();
      expect(result.violations?.size).toBeGreaterThan(0);
    });
  });

  describe('Evaluate Rule With Details', () => {
    it('should provide condition-by-condition analysis', () => {
      const analysis = evaluateRuleWithDetails(testRule, testRecord, fieldTypes);

      expect(analysis.ruleMatched).toBe(true);
      expect(analysis.conditionResults.length).toBeGreaterThan(0);
      expect(analysis.conditionResults[0]).toHaveProperty('field');
      expect(analysis.conditionResults[0]).toHaveProperty('operator');
      expect(analysis.conditionResults[0]).toHaveProperty('matched');
      expect(analysis.conditionResults[0]).toHaveProperty('reason');
    });

    it('should track reason for each condition', () => {
      const record = { ...testRecord, salary: 30000 };
      const analysis = evaluateRuleWithDetails(testRule, record, fieldTypes);

      expect(analysis.ruleMatched).toBe(false);
      expect(analysis.conditionResults[0].matched).toBe(false);
      expect(analysis.conditionResults[0].reason).toBeDefined();
    });
  });
});

describe('Integration Tests', () => {
  const rules: ValidationRule[] = [
    {
      id: 'salary_range',
      rule_name: 'Salary Range Check',
      rule_type: 'field_format',
      target_entity: 'Employee',
      severity: 'error',
      description: 'Salary must be between 30k and 200k',
      is_active: true,
      is_global: true,
      conditions: [
        {
          field: 'salary',
          operator: 'expressions',
          value: '[30000,200000]'
        }
      ]
    },
    {
      id: 'email_format',
      rule_name: 'Email Format',
      rule_type: 'field_format',
      target_entity: 'Employee',
      severity: 'warning',
      description: 'Email must contain company domain',
      is_active: true,
      is_global: false,
      conditions: [
        {
          field: 'email',
          operator: 'expressions',
          value: '%@company.com'
        }
      ]
    }
  ];

  const fieldTypes: Record<string, FieldType> = {
    salary: 'number',
    email: 'string',
    hire_date: 'date'
  };

  it('should validate complex employee records', () => {
    const employee = {
      id: 'emp_1',
      name: 'John Doe',
      salary: 85000,
      email: 'john@company.com',
      hire_date: new Date()
    };

    const result = executeRules(rules, employee, fieldTypes);
    expect(result.passed).toBe(true);
    expect(result.matchedRules).toBe(2);
  });

  it('should handle batch processing', () => {
    const employees = [
      { id: 'emp_1', salary: 85000, email: 'john@company.com' },
      { id: 'emp_2', salary: 250000, email: 'jane@gmail.com' },  // Invalid salary
      { id: 'emp_3', salary: 65000, email: 'bob@company.com' }
    ];

    const { validRecords, invalidRecords } = validateRecordsBatch(
      employees,
      'Employee',
      rules,
      fieldTypes
    );

    expect(validRecords.length).toBe(2);
    expect(invalidRecords.length).toBe(1);
  });

  it('should provide detailed debugging info', () => {
    const employee = {
      id: 'emp_1',
      salary: 200000,  // At boundary
      email: 'test@other.com'  // Invalid domain
    };

    const analysis = evaluateRuleWithDetails(rules[0], employee, fieldTypes);
    expect(analysis.conditionResults).toBeDefined();
    expect(analysis.conditionResults.length).toBeGreaterThan(0);
  });
});
