/**
 * Validation Rule Execution Engine
 * 
 * Executes validation rules against data records using the expression parser.
 * Integrates with business processes for real-time data validation.
 */

import {
  evaluateAllConditions,
  evaluateCondition,
  FieldType
} from './expression_parser';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

export interface ValidationRuleCondition {
  field: string;
  operator: string;
  value: string;
}

export interface ValidationRule {
  id: string;
  rule_name: string;
  rule_type: string;
  target_entity: string;
  target_entities?: string[];
  severity: 'error' | 'warning' | 'info';
  description: string;
  is_active: boolean;
  is_global: boolean;
  is_core?: boolean;
  conditions?: ValidationRuleCondition[];
  created_at?: string;
  updated_at?: string;
}

export interface FieldSchema {
  name: string;
  type: FieldType;
  required?: boolean;
  enumValues?: string[];
}

export interface ValidationViolation {
  ruleId: string;
  ruleName: string;
  severity: 'error' | 'warning' | 'info';
  message: string;
  violatedConditions: {
    field: string;
    operator: string;
    value: string;
    reason: string;
  }[];
}

export interface ExecutionResult {
  passed: boolean;
  violations: ValidationViolation[];
  summary: string;
  executedRules: number;
  matchedRules: number;
}

// ============================================================================
// RULE EXECUTOR
// ============================================================================

/**
 * Execute a single validation rule against a record
 */
export const executeRule = (
  rule: ValidationRule,
  record: Record<string, any>,
  fieldTypes: Record<string, FieldType>
): { matched: boolean; violations: ValidationViolation[] } => {
  const violations: ValidationViolation[] = [];

  // Skip if rule is inactive
  if (!rule.is_active) {
    return { matched: false, violations };
  }

  // If no conditions, rule always matches (applies to all records)
  if (!rule.conditions || rule.conditions.length === 0) {
    return { matched: true, violations };
  }

  // Evaluate all conditions (AND logic)
  const conditionResult = evaluateAllConditions(record, fieldTypes, rule.conditions);

  if (!conditionResult.valid) {
    // Configuration error - shouldn't block, but report
    violations.push({
      ruleId: rule.id,
      ruleName: rule.rule_name,
      severity: 'error',
      message: `Rule configuration error: ${conditionResult.message}`,
      violatedConditions: []
    });
    return { matched: false, violations };
  }

  // Rule matched - all conditions are true for this record
  return { matched: !!conditionResult.matches, violations };
};

/**
 * Execute multiple rules against a record
 * Returns violations for rules that matched AND failed (severity-based checks)
 */
export const executeRules = (
  rules: ValidationRule[],
  record: Record<string, any>,
  fieldTypes: Record<string, FieldType>
): ExecutionResult => {
  const violations: ValidationViolation[] = [];
  let executedRules = 0;
  let matchedRules = 0;

  for (const rule of rules) {
    if (!rule.is_active) continue;

    executedRules++;

    const { matched, violations: ruleViolations } = executeRule(rule, record, fieldTypes);

    if (ruleViolations.length > 0) {
      violations.push(...ruleViolations);
    }

    if (matched) {
      matchedRules++;
      
      // If rule matched and has high severity, record violation
      if (rule.severity === 'error') {
        violations.push({
          ruleId: rule.id,
          ruleName: rule.rule_name,
          severity: rule.severity,
          message: `${rule.rule_name}: ${rule.description}`,
          violatedConditions: (rule.conditions || []).map(c => ({
            ...c,
            reason: `Condition matched`
          }))
        });
      }
    }
  }

  const passed = violations.filter(v => v.severity === 'error').length === 0;

  return {
    passed,
    violations,
    summary: `Executed ${executedRules} rules, matched ${matchedRules}, found ${violations.length} violations`,
    executedRules,
    matchedRules
  };
};

// ============================================================================
// BUSINESS PROCESS INTEGRATION
// ============================================================================

/**
 * Validate a record before insertion/update
 * Called from your business logic before database operations
 */
export const validateRecordBeforeWrite = (
  record: Record<string, any>,
  entity: string,
  rules: ValidationRule[],
  fieldTypes: Record<string, FieldType>
): { valid: boolean; errors: ValidationViolation[] } => {
  // Filter rules applicable to this entity
  const applicableRules = rules.filter(r => {
    if (r.is_global) return true;
    if (r.target_entities?.includes(entity)) return true;
    if (r.target_entity === entity) return true;
    return false;
  });

  const result = executeRules(applicableRules, record, fieldTypes);

  return {
    valid: result.passed,
    errors: result.violations.filter(v => v.severity === 'error')
  };
};

/**
 * Validate multiple records (batch operation)
 */
export const validateRecordsBatch = (
  records: Record<string, any>[],
  entity: string,
  rules: ValidationRule[],
  fieldTypes: Record<string, FieldType>
): { validRecords: Record<string, any>[]; invalidRecords: { record: Record<string, any>; errors: ValidationViolation[] }[] } => {
  const validRecords: Record<string, any>[] = [];
  const invalidRecords: { record: Record<string, any>; errors: ValidationViolation[] }[] = [];

  for (const record of records) {
    const validation = validateRecordBeforeWrite(record, entity, rules, fieldTypes);

    if (validation.valid) {
      validRecords.push(record);
    } else {
      invalidRecords.push({
        record,
        errors: validation.errors
      });
    }
  }

  return { validRecords, invalidRecords };
};

// ============================================================================
// QUERY FILTERING
// ============================================================================

/**
 * Filter records from a result set based on rules
 * Useful for post-query validation or data quality checks
 */
export const filterRecordsByRules = (
  records: Record<string, any>[],
  entity: string,
  rules: ValidationRule[],
  fieldTypes: Record<string, FieldType>,
  includeViolations: boolean = false
): {
  validRecords: Record<string, any>[];
  violations?: Map<string, ValidationViolation[]>;
} => {
  const validRecords: Record<string, any>[] = [];
  const violations = new Map<string, ValidationViolation[]>();

  const applicableRules = rules.filter(r => {
    if (r.is_global) return true;
    if (r.target_entities?.includes(entity)) return true;
    if (r.target_entity === entity) return true;
    return false;
  });

  for (let i = 0; i < records.length; i++) {
    const record = records[i];
    const result = executeRules(applicableRules, record, fieldTypes);

    if (result.passed) {
      validRecords.push(record);
    } else if (includeViolations) {
      violations.set(String(i), result.violations);
    }
  }

  return includeViolations
    ? { validRecords, violations }
    : { validRecords };
};

// ============================================================================
// RULE EVALUATION WITH CONTEXT
// ============================================================================

/**
 * Detailed condition-by-condition evaluation for debugging/analysis
 */
export const evaluateRuleWithDetails = (
  rule: ValidationRule,
  record: Record<string, any>,
  fieldTypes: Record<string, FieldType>
): {
  ruleMatched: boolean;
  conditionResults: Array<{
    field: string;
    operator: string;
    value: string;
    matched: boolean;
    reason: string;
  }>;
} => {
  const conditionResults: Array<{
    field: string;
    operator: string;
    value: string;
    matched: boolean;
    reason: string;
  }> = [];

  if (!rule.conditions || rule.conditions.length === 0) {
    return {
      ruleMatched: true,
      conditionResults: []
    };
  }

  let allMatched = true;

  for (const condition of rule.conditions) {
    const fieldValue = record[condition.field];
    const fieldType = fieldTypes[condition.field];

    if (!fieldType) {
      conditionResults.push({
        field: condition.field,
        operator: condition.operator,
        value: condition.value,
        matched: false,
        reason: `Unknown field type: ${condition.field}`
      });
      allMatched = false;
      continue;
    }

    const result = evaluateCondition(
      fieldValue,
      fieldType,
      condition.operator,
      condition.value
    );

    conditionResults.push({
      field: condition.field,
      operator: condition.operator,
      value: condition.value,
      matched: result.matches || false,
      reason: result.message
    });

    if (!result.matches) {
      allMatched = false;
    }
  }

  return {
    ruleMatched: allMatched,
    conditionResults
  };
};

// ============================================================================
// ERROR HANDLING & LOGGING
// ============================================================================

export interface ValidationLogger {
  log: (level: 'info' | 'warn' | 'error', message: string, context?: any) => void;
}

/**
 * Execute rules with error handling and logging
 */
export const executeRulesWithLogging = (
  rules: ValidationRule[],
  record: Record<string, any>,
  fieldTypes: Record<string, FieldType>,
  logger?: ValidationLogger
): ExecutionResult => {
  try {
    const result = executeRules(rules, record, fieldTypes);

    if (logger) {
      if (result.passed) {
        logger.log('info', `Validation passed: ${result.summary}`);
      } else {
        logger.log('warn', `Validation failed: ${result.summary}`, {
          violations: result.violations.map(v => v.message)
        });
      }
    }

    return result;
  } catch (error) {
    const message = `Validation engine error: ${String(error)}`;

    if (logger) {
      logger.log('error', message, { error });
    }

    return {
      passed: false,
      violations: [{
        ruleId: 'SYSTEM',
        ruleName: 'System Error',
        severity: 'error',
        message,
        violatedConditions: []
      }],
      summary: message,
      executedRules: 0,
      matchedRules: 0
    };
  }
};

// ============================================================================
// EXPORT
// ============================================================================

export default {
  executeRule,
  executeRules,
  validateRecordBeforeWrite,
  validateRecordsBatch,
  filterRecordsByRules,
  evaluateRuleWithDetails,
  executeRulesWithLogging
};
