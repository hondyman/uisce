/**
 * Expression Parser & Evaluator
 * 
 * Evaluates Looker-style filter expressions for data validation.
 * Supports strings, numbers, dates, booleans, and enums.
 */

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

export type FieldType = 'string' | 'number' | 'date' | 'boolean' | 'enum';

export interface ExpressionResult {
  valid: boolean;
  message: string;
  matches?: boolean;  // Whether the value matches the expression
}

export interface EvaluationContext {
  fieldValue: any;
  fieldType: FieldType;
  expression: string;
  operator: string;
}

// ============================================================================
// STRING EXPRESSION EVALUATOR
// ============================================================================

export const evaluateStringExpression = (
  value: any,
  expression: string
): ExpressionResult => {
  if (value === null || value === undefined) {
    value = '';
  }
  
  const str = String(value).trim();

  try {
    // EMPTY or NULL check
    if (expression === 'EMPTY' || expression === 'NULL') {
      return {
        valid: true,
        matches: str === '' || value === null || value === undefined,
        message: 'String check: EMPTY/NULL'
      };
    }

    // Negation: -pattern
    if (expression.startsWith('-')) {
      const pattern = expression.slice(1).trim();
      
      // -% prefix means NOT contains
      if (pattern.startsWith('%') && pattern.endsWith('%')) {
        const substring = pattern.slice(1, -1);
        return {
          valid: true,
          matches: !str.includes(substring),
          message: `String check: does NOT contain "${substring}"`
        };
      }
      
      // -%FOO means NOT contains FOO anywhere
      if (pattern.startsWith('%')) {
        const substring = pattern.slice(1);
        return {
          valid: true,
          matches: !str.includes(substring),
          message: `String check: does NOT contain "${substring}"`
        };
      }
      
      // -FOO% means NOT starts with FOO
      if (pattern.endsWith('%')) {
        const prefix = pattern.slice(0, -1);
        return {
          valid: true,
          matches: !str.startsWith(prefix),
          message: `String check: does NOT start with "${prefix}"`
        };
      }
      
      // -FOO means NOT equal
      return {
        valid: true,
        matches: str !== pattern,
        message: `String check: is NOT equal to "${pattern}"`
      };
    }

    // Wildcard patterns
    if (expression.startsWith('%') && expression.endsWith('%')) {
      // %FOO% = contains
      const substring = expression.slice(1, -1);
      return {
        valid: true,
        matches: str.includes(substring),
        message: `String check: contains "${substring}"`
      };
    }

    if (expression.startsWith('%')) {
      // %FOO = ends with
      const suffix = expression.slice(1);
      return {
        valid: true,
        matches: str.endsWith(suffix),
        message: `String check: ends with "${suffix}"`
      };
    }

    if (expression.endsWith('%')) {
      // FOO% = starts with
      const prefix = expression.slice(0, -1);
      return {
        valid: true,
        matches: str.startsWith(prefix),
        message: `String check: starts with "${prefix}"`
      };
    }

    // Exact match
    return {
      valid: true,
      matches: str === expression,
      message: `String check: exact match "${expression}"`
    };
  } catch (error) {
    return {
      valid: false,
      message: `Error evaluating string expression: ${String(error)}`
    };
  }
};

// ============================================================================
// NUMERIC EXPRESSION EVALUATOR
// ============================================================================

export const evaluateNumericExpression = (
  value: any,
  expression: string
): ExpressionResult => {
  try {
    const num = Number(value);

    if (isNaN(num)) {
      return {
        valid: false,
        message: `Value "${value}" is not a valid number`
      };
    }

    const expr = expression.trim();

    // Handle AND logic: ">=5 AND <=10"
    if (expr.includes(' AND ')) {
      const parts = expr.split(' AND ').map(p => p.trim());
      for (const part of parts) {
        const result = evaluateNumericExpression(num, part);
        if (!result.valid || !result.matches) {
          return {
            valid: result.valid,
            matches: false,
            message: result.valid ? `Failed: ${result.message}` : result.message
          };
        }
      }
      return {
        valid: true,
        matches: true,
        message: `Numeric check: all AND conditions met`
      };
    }

    // Handle OR logic: "100 OR 200"
    if (expr.includes(' OR ')) {
      const parts = expr.split(' OR ').map(p => p.trim());
      for (const part of parts) {
        const result = evaluateNumericExpression(num, part);
        if (result.valid && result.matches) {
          return {
            valid: true,
            matches: true,
            message: `Numeric check: matches one of the OR conditions`
          };
        }
      }
      return {
        valid: true,
        matches: false,
        message: `Numeric check: does not match any OR condition`
      };
    }

    // Handle NOT: "NOT 5"
    if (expr.startsWith('NOT ')) {
      const target = Number(expr.slice(4).trim());
      if (isNaN(target)) {
        return {
          valid: false,
          message: `Invalid NOT expression: "${expr}"`
        };
      }
      return {
        valid: true,
        matches: num !== target,
        message: `Numeric check: is NOT equal to ${target}`
      };
    }

    // Handle intervals: [5,90], (1,7), [5,90)
    if (expr.match(/^[\[(][\d.]+,[\d.]+[\])]$/)) {
      const isLeftInclusive = expr[0] === '[';
      const isRightInclusive = expr[expr.length - 1] === ']';
      
      const innerPart = expr.slice(1, -1);
      const [minStr, maxStr] = innerPart.split(',').map(s => s.trim());
      
      const min = Number(minStr);
      const max = Number(maxStr);
      
      if (isNaN(min) || isNaN(max)) {
        return {
          valid: false,
          message: `Invalid interval: "${expr}"`
        };
      }
      
      if (min > max) {
        return {
          valid: false,
          message: `Invalid interval: min (${min}) > max (${max})`
        };
      }
      
      const minCheck = isLeftInclusive ? num >= min : num > min;
      const maxCheck = isRightInclusive ? num <= max : num < max;
      
      const intervalType = `${isLeftInclusive ? '[' : '('}${min},${max}${isRightInclusive ? ']' : ')'}`;
      
      return {
        valid: true,
        matches: minCheck && maxCheck,
        message: `Numeric check: in interval ${intervalType}`
      };
    }

    // Handle comparisons: >, <, >=, <=, =, !=
    if (expr.startsWith('>=')) {
      const target = Number(expr.slice(2).trim());
      return {
        valid: true,
        matches: num >= target,
        message: `Numeric check: >= ${target}`
      };
    }

    if (expr.startsWith('<=')) {
      const target = Number(expr.slice(2).trim());
      return {
        valid: true,
        matches: num <= target,
        message: `Numeric check: <= ${target}`
      };
    }

    if (expr.startsWith('>')) {
      const target = Number(expr.slice(1).trim());
      return {
        valid: true,
        matches: num > target,
        message: `Numeric check: > ${target}`
      };
    }

    if (expr.startsWith('<')) {
      const target = Number(expr.slice(1).trim());
      return {
        valid: true,
        matches: num < target,
        message: `Numeric check: < ${target}`
      };
    }

    if (expr.startsWith('=')) {
      const target = Number(expr.slice(1).trim());
      return {
        valid: true,
        matches: num === target,
        message: `Numeric check: = ${target}`
      };
    }

    if (expr.startsWith('!=')) {
      const target = Number(expr.slice(2).trim());
      return {
        valid: true,
        matches: num !== target,
        message: `Numeric check: != ${target}`
      };
    }

    // Handle list: "1,5,10"
    if (expr.includes(',') && !expr.match(/[\[\(]/)) {
      const values = expr.split(',').map(v => Number(v.trim()));
      if (values.some(isNaN)) {
        return {
          valid: false,
          message: `Invalid list expression: "${expr}"`
        };
      }
      return {
        valid: true,
        matches: values.includes(num),
        message: `Numeric check: in list (${expr})`
      };
    }

    // Simple exact match
    const target = Number(expr);
    if (isNaN(target)) {
      return {
        valid: false,
        message: `Invalid numeric expression: "${expr}"`
      };
    }

    return {
      valid: true,
      matches: num === target,
      message: `Numeric check: exactly ${target}`
    };
  } catch (error) {
    return {
      valid: false,
      message: `Error evaluating numeric expression: ${String(error)}`
    };
  }
};

// ============================================================================
// DATE EXPRESSION EVALUATOR
// ============================================================================

export const evaluateDateExpression = (
  value: any,
  expression: string
): ExpressionResult => {
  try {
    let date: Date;

    // Parse input date
    if (value instanceof Date) {
      date = value;
    } else if (typeof value === 'string') {
      date = new Date(value);
    } else if (typeof value === 'number') {
      date = new Date(value);
    } else {
      return {
        valid: false,
        message: `Invalid date value: ${value}`
      };
    }

    if (isNaN(date.getTime())) {
      return {
        valid: false,
        message: `Could not parse date: ${value}`
      };
    }

    const expr = expression.trim().toLowerCase();
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    // === RELATIVE DATES ===

    // today
    if (expr === 'today') {
      const dateOnly = new Date(date);
      dateOnly.setHours(0, 0, 0, 0);
      return {
        valid: true,
        matches: dateOnly.getTime() === today.getTime(),
        message: `Date check: is today`
      };
    }

    // yesterday
    if (expr === 'yesterday') {
      const yesterday = new Date(today);
      yesterday.setDate(yesterday.getDate() - 1);
      const dateOnly = new Date(date);
      dateOnly.setHours(0, 0, 0, 0);
      return {
        valid: true,
        matches: dateOnly.getTime() === yesterday.getTime(),
        message: `Date check: is yesterday`
      };
    }

    // this week (Mon-Sun)
    if (expr === 'this week') {
      const weekStart = new Date(today);
      const day = weekStart.getDay();
      weekStart.setDate(weekStart.getDate() - (day === 0 ? 6 : day - 1));
      
      const weekEnd = new Date(weekStart);
      weekEnd.setDate(weekEnd.getDate() + 6);
      
      const dateOnly = new Date(date);
      dateOnly.setHours(0, 0, 0, 0);
      
      const matches = dateOnly >= weekStart && dateOnly <= weekEnd;
      return {
        valid: true,
        matches,
        message: `Date check: in this week`
      };
    }

    // this month
    if (expr === 'this month') {
      const year = today.getFullYear();
      const month = today.getMonth();
      
      const monthStart = new Date(year, month, 1);
      const monthEnd = new Date(year, month + 1, 0);
      
      const dateOnly = new Date(date);
      dateOnly.setHours(0, 0, 0, 0);
      
      const matches = dateOnly >= monthStart && dateOnly <= monthEnd;
      return {
        valid: true,
        matches,
        message: `Date check: in this month`
      };
    }

    // this year
    if (expr === 'this year') {
      const year = today.getFullYear();
      const yearStart = new Date(year, 0, 1);
      const yearEnd = new Date(year, 11, 31);
      
      const dateOnly = new Date(date);
      dateOnly.setHours(0, 0, 0, 0);
      
      const matches = dateOnly >= yearStart && dateOnly <= yearEnd;
      return {
        valid: true,
        matches,
        message: `Date check: in this year`
      };
    }

    // last N days
    const lastDaysMatch = expr.match(/^last\s+(\d+)\s+(day|days)$/);
    if (lastDaysMatch) {
      const days = parseInt(lastDaysMatch[1], 10);
      const startDate = new Date(today);
      startDate.setDate(startDate.getDate() - days);
      
      const dateOnly = new Date(date);
      dateOnly.setHours(0, 0, 0, 0);
      
      const matches = dateOnly >= startDate && dateOnly <= today;
      return {
        valid: true,
        matches,
        message: `Date check: in last ${days} days`
      };
    }

    // N days ago
    const daysAgoMatch = expr.match(/^(\d+)\s+(day|days)\s+ago$/);
    if (daysAgoMatch) {
      const days = parseInt(daysAgoMatch[1], 10);
      const targetDate = new Date(today);
      targetDate.setDate(targetDate.getDate() - days);
      
      const dateOnly = new Date(date);
      dateOnly.setHours(0, 0, 0, 0);
      
      return {
        valid: true,
        matches: dateOnly.getTime() === targetDate.getTime(),
        message: `Date check: is ${days} days ago`
      };
    }

    // N weeks ago
    const weeksAgoMatch = expr.match(/^(\d+)\s+(week|weeks)\s+ago$/);
    if (weeksAgoMatch) {
      const weeks = parseInt(weeksAgoMatch[1], 10);
      const targetDate = new Date(today);
      targetDate.setDate(targetDate.getDate() - weeks * 7);
      
      const dateOnly = new Date(date);
      dateOnly.setHours(0, 0, 0, 0);
      
      return {
        valid: true,
        matches: dateOnly.getTime() === targetDate.getTime(),
        message: `Date check: is ${weeks} weeks ago`
      };
    }

    // N months ago
    const monthsAgoMatch = expr.match(/^(\d+)\s+(month|months)\s+ago$/);
    if (monthsAgoMatch) {
      const months = parseInt(monthsAgoMatch[1], 10);
      const targetDate = new Date(today);
      targetDate.setMonth(targetDate.getMonth() - months);
      
      const dateOnly = new Date(date);
      dateOnly.setHours(0, 0, 0, 0);
      
      return {
        valid: true,
        matches: dateOnly.getTime() === targetDate.getTime(),
        message: `Date check: is ${months} months ago`
      };
    }

    // Day of week (Monday, Tuesday, etc.)
    const daysOfWeek = ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'];
    if (daysOfWeek.includes(expr)) {
      const targetDay = daysOfWeek.indexOf(expr);
      const dateDay = date.getDay();
      return {
        valid: true,
        matches: dateDay === targetDay,
        message: `Date check: is ${expr}`
      };
    }

    // === ABSOLUTE DATES ===

    // ISO format: YYYY-MM-DD
    if (expr.match(/^\d{4}-\d{2}-\d{2}$/)) {
      const targetDate = new Date(expr + 'T00:00:00Z');
      const dateOnly = new Date(date);
      dateOnly.setHours(0, 0, 0, 0);
      
      return {
        valid: true,
        matches: dateOnly.getTime() === targetDate.getTime(),
        message: `Date check: is ${expr}`
      };
    }

    // after YYYY-MM-DD
    if (expr.startsWith('after ')) {
      const dateStr = expr.slice(6).trim();
      if (!dateStr.match(/^\d{4}-\d{2}-\d{2}$/)) {
        return {
          valid: false,
          message: `Invalid date format: "${dateStr}". Use YYYY-MM-DD`
        };
      }
      const targetDate = new Date(dateStr + 'T00:00:00Z');
      const dateOnly = new Date(date);
      dateOnly.setHours(0, 0, 0, 0);
      
      return {
        valid: true,
        matches: dateOnly >= targetDate,
        message: `Date check: on or after ${dateStr}`
      };
    }

    // before YYYY-MM-DD
    if (expr.startsWith('before ')) {
      const dateStr = expr.slice(7).trim();
      if (!dateStr.match(/^\d{4}-\d{2}-\d{2}$/)) {
        return {
          valid: false,
          message: `Invalid date format: "${dateStr}". Use YYYY-MM-DD`
        };
      }
      const targetDate = new Date(dateStr + 'T00:00:00Z');
      const dateOnly = new Date(date);
      dateOnly.setHours(0, 0, 0, 0);
      
      return {
        valid: true,
        matches: dateOnly <= targetDate,
        message: `Date check: on or before ${dateStr}`
      };
    }

    return {
      valid: false,
      message: `Unrecognized date expression: "${expression}"`
    };
  } catch (error) {
    return {
      valid: false,
      message: `Error evaluating date expression: ${String(error)}`
    };
  }
};

// ============================================================================
// MAIN EVALUATOR
// ============================================================================

/**
 * Evaluate a condition against a value
 * 
 * @param value - The field value to evaluate
 * @param fieldType - The data type of the field
 * @param operator - The operator (equals, expressions, relative_dates, etc.)
 * @param expression - The value or expression to match against
 * @returns ExpressionResult with validation and match status
 */
export const evaluateCondition = (
  value: any,
  fieldType: FieldType,
  operator: string,
  expression: string
): ExpressionResult => {
  // Special case: stateless operators
  if (operator === 'is_empty') {
    const isEmpty = value === null || value === undefined || value === '' || value === 0 || value === false;
    return {
      valid: true,
      matches: isEmpty,
      message: `Check: value is empty`
    };
  }

  if (operator === 'is_not_empty') {
    const isEmpty = value === null || value === undefined || value === '' || value === 0 || value === false;
    return {
      valid: true,
      matches: !isEmpty,
      message: `Check: value is not empty`
    };
  }

  if (operator === 'is_true') {
    return {
      valid: true,
      matches: value === true || value === 1 || value === 'true' || value === 'True',
      message: `Check: value is true`
    };
  }

  if (operator === 'is_false') {
    return {
      valid: true,
      matches: value === false || value === 0 || value === 'false' || value === 'False',
      message: `Check: value is false`
    };
  }

  // Simple operators
  if (operator === 'equals') {
    return {
      valid: true,
      matches: String(value).trim() === String(expression).trim(),
      message: `Check: equals "${expression}"`
    };
  }

  if (operator === 'not_equals') {
    return {
      valid: true,
      matches: String(value).trim() !== String(expression).trim(),
      message: `Check: not equals "${expression}"`
    };
  }

  // Type-specific evaluators
  if (fieldType === 'string') {
    if (operator === 'contains') {
      return {
        valid: true,
        matches: String(value).includes(expression),
        message: `Check: contains "${expression}"`
      };
    }

    if (operator === 'starts_with') {
      return {
        valid: true,
        matches: String(value).startsWith(expression),
        message: `Check: starts with "${expression}"`
      };
    }

    if (operator === 'ends_with') {
      return {
        valid: true,
        matches: String(value).endsWith(expression),
        message: `Check: ends with "${expression}"`
      };
    }

    if (operator === 'expressions') {
      return evaluateStringExpression(value, expression);
    }
  }

  if (fieldType === 'number') {
    if (operator === 'greater_than') {
      const num = Number(value);
      const target = Number(expression);
      return {
        valid: true,
        matches: num > target,
        message: `Check: > ${target}`
      };
    }

    if (operator === 'less_than') {
      const num = Number(value);
      const target = Number(expression);
      return {
        valid: true,
        matches: num < target,
        message: `Check: < ${target}`
      };
    }

    if (operator === 'expressions') {
      return evaluateNumericExpression(value, expression);
    }
  }

  if (fieldType === 'date') {
    if (operator === 'before') {
      const date = new Date(value);
      const target = new Date(expression);
      date.setHours(0, 0, 0, 0);
      target.setHours(0, 0, 0, 0);
      return {
        valid: true,
        matches: date <= target,
        message: `Check: before ${expression}`
      };
    }

    if (operator === 'after') {
      const date = new Date(value);
      const target = new Date(expression);
      date.setHours(0, 0, 0, 0);
      target.setHours(0, 0, 0, 0);
      return {
        valid: true,
        matches: date >= target,
        message: `Check: after ${expression}`
      };
    }

    if (operator === 'relative_dates' || operator === 'expressions') {
      return evaluateDateExpression(value, expression);
    }
  }

  return {
    valid: false,
    message: `Unsupported operator "${operator}" for type "${fieldType}"`
  };
};

// ============================================================================
// BATCH EVALUATION (FOR MULTIPLE CONDITIONS)
// ============================================================================

/**
 * Evaluate multiple conditions (all must match = AND logic)
 */
export const evaluateAllConditions = (
  record: Record<string, any>,
  fieldTypes: Record<string, FieldType>,
  conditions: Array<{
    field: string;
    operator: string;
    value: string;
  }>
): ExpressionResult => {
  for (const condition of conditions) {
    const fieldValue = record[condition.field];
    const fieldType = fieldTypes[condition.field];

    if (!fieldType) {
      return {
        valid: false,
        message: `Unknown field type for "${condition.field}"`
      };
    }

    const result = evaluateCondition(
      fieldValue,
      fieldType,
      condition.operator,
      condition.value
    );

    if (!result.valid) {
      return result;
    }

    if (!result.matches) {
      // This condition didn't match - AND logic requires all to match
      return {
        valid: true,
        matches: false,
        message: `Condition failed: ${result.message}`
      };
    }
  }

  // All conditions matched
  return {
    valid: true,
    matches: true,
    message: `All conditions matched`
  };
};

export default {
  evaluateCondition,
  evaluateAllConditions,
  evaluateStringExpression,
  evaluateNumericExpression,
  evaluateDateExpression
};
