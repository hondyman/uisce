/**
 * Advanced Condition Builder
 * 
 * Supports:
 * - Simple conditions (equals, contains, etc.)
 * - Looker-style filter expressions
 * - Relative dates (last 7 days, this month, etc.)
 * - Pattern matching with wildcards
 * - Logical operators (AND, OR, NOT)
 * - Numeric intervals with algebraic notation
 */

import React, { useState, useMemo } from 'react';
import { Info, AlertCircle, CheckCircle2 } from 'lucide-react';

export type ConditionMode = 'simple' | 'advanced' | 'expressions';

export interface AdvancedFieldTypeInfo {
  type: 'string' | 'number' | 'boolean' | 'date' | 'enum' | 'unknown';
  enumValues?: string[];
  isNullable?: boolean;
  supportsExpressions?: boolean; // Enable Looker expressions for this field
  supportsRelativeDates?: boolean; // Enable relative dates for date fields
}

export interface AdvancedConditionBuilderProps {
  field: string;
  operator: string;
  value: string;
  fieldType: AdvancedFieldTypeInfo | string;
  onUpdate: (updates: { field?: string; operator?: string; value?: string }) => void;
  onRemove: () => void;
  conditionIndex?: number;
  fieldMetadata?: Record<string, { type: string }>;
}

// ============================================================================
// OPERATORS & EXPRESSIONS
// ============================================================================

const SIMPLE_OPERATORS = {
  string: [
    { value: 'equals', label: 'Equals', category: 'basic' },
    { value: 'not_equals', label: 'Not Equals', category: 'basic' },
    { value: 'contains', label: 'Contains', category: 'pattern' },
    { value: 'starts_with', label: 'Starts With', category: 'pattern' },
    { value: 'ends_with', label: 'Ends With', category: 'pattern' },
    { value: 'is_empty', label: 'Is Empty', category: 'state' },
    { value: 'is_not_empty', label: 'Is Not Empty', category: 'state' },
    { value: 'expressions', label: 'Advanced Expressions', category: 'advanced' },
  ],
  number: [
    { value: 'equals', label: 'Equals', category: 'basic' },
    { value: 'not_equals', label: 'Not Equals', category: 'basic' },
    { value: 'greater_than', label: 'Greater Than', category: 'comparison' },
    { value: 'less_than', label: 'Less Than', category: 'comparison' },
    { value: 'is_empty', label: 'Is Empty', category: 'state' },
    { value: 'is_not_empty', label: 'Is Not Empty', category: 'state' },
    { value: 'expressions', label: 'Advanced Expressions', category: 'advanced' },
  ],
  date: [
    { value: 'equals', label: 'Equals', category: 'basic' },
    { value: 'before', label: 'Before', category: 'comparison' },
    { value: 'after', label: 'After', category: 'comparison' },
    { value: 'is_empty', label: 'Is Empty', category: 'state' },
    { value: 'is_not_empty', label: 'Is Not Empty', category: 'state' },
    { value: 'relative_dates', label: 'Relative Dates', category: 'advanced' },
    { value: 'expressions', label: 'Advanced Expressions', category: 'advanced' },
  ],
  boolean: [
    { value: 'equals', label: 'Equals', category: 'basic' },
    { value: 'is_true', label: 'Is True', category: 'basic' },
    { value: 'is_false', label: 'Is False', category: 'basic' },
  ],
  enum: [
    { value: 'equals', label: 'Equals', category: 'basic' },
    { value: 'not_equals', label: 'Not Equals', category: 'basic' },
    { value: 'in_list', label: 'In List', category: 'pattern' },
    { value: 'expressions', label: 'Advanced Expressions', category: 'advanced' },
  ],
};

// ============================================================================
// EXPRESSION VALIDATORS & HELPERS
// ============================================================================

/**
 * Validates Looker-style string filter expressions
 * Examples: FOO, FOO%,  %FOO%, -FOO, FOO,BAR, EMPTY, NULL
 */
export const validateStringExpression = (expr: string): { valid: boolean; message: string } => {
  if (!expr.trim()) return { valid: false, message: 'Expression cannot be empty' };

  const trimmed = expr.trim();

  // Check for EMPTY or NULL keywords
  if (trimmed === 'EMPTY' || trimmed === 'NULL' || trimmed === '-EMPTY' || trimmed === '-NULL') {
    return { valid: true, message: 'Keyword expression' };
  }

  // Check for invalid leading characters (except -)
  if (trimmed.startsWith('^') && !trimmed.match(/^\^[-_%"]/)) {
    return { valid: false, message: 'Invalid escape sequence' };
  }

  return { valid: true, message: 'Valid string expression' };
};

/**
 * Validates Looker-style numeric filter expressions
 * Examples: 5, >10, >=5 AND <=10, 3 to 10, [5,90], (1,7), NOT 5
 */
export const validateNumericExpression = (expr: string): { valid: boolean; message: string } => {
  if (!expr.trim()) return { valid: false, message: 'Expression cannot be empty' };

  const trimmed = expr.trim().toUpperCase();

  // Check for NULL
  if (trimmed === 'NULL' || trimmed === 'NOT NULL') {
    return { valid: true, message: 'NULL expression' };
  }

  // Check for interval notation
  if (trimmed.match(/^[\(\[].*[\)\]]$/)) {
    return { valid: true, message: 'Interval notation' };
  }

  // Check for operators
  if (trimmed.match(/^[<>=!]+|^(AND|OR|NOT|TO)\b/i)) {
    return { valid: true, message: 'Operator expression' };
  }

  // Check for numbers
  if (trimmed.match(/^\d+\.?\d*$/)) {
    return { valid: true, message: 'Numeric value' };
  }

  return { valid: false, message: 'Invalid numeric expression' };
};

/**
 * Validates Looker-style date filter expressions
 * Examples: today, 7 days ago, this month, after 2018-01-01, 2018-05-10 for 3 days
 */
export const validateDateExpression = (expr: string): { valid: boolean; message: string } => {
  if (!expr.trim()) return { valid: false, message: 'Expression cannot be empty' };

  const trimmed = expr.trim().toLowerCase();

  // Relative date keywords
  const relativeDateKeywords = [
    'today', 'yesterday', 'tomorrow',
    'this week', 'this month', 'this quarter', 'this year',
    'last week', 'last month', 'last quarter', 'last year',
    'next week', 'next month', 'next quarter', 'next year',
    'weeks?', 'days?', 'months?', 'quarters?', 'years?',
    'ago', 'from now', 'before', 'after',
  ];

  const hasKeyword = relativeDateKeywords.some(kw => trimmed.includes(kw));

  // Absolute dates (YYYY-MM-DD or YYYY/MM/DD)
  const isAbsoluteDate = trimmed.match(/\d{4}[-/]\d{2}[-/]\d{2}/);

  // Day of week
  const daysOfWeek = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'];
  const isDayOfWeek = daysOfWeek.some(day => trimmed === day);

  if (hasKeyword || isAbsoluteDate || isDayOfWeek) {
    return { valid: true, message: 'Valid date expression' };
  }

  return { valid: false, message: 'Invalid date expression (use keywords like "today", "7 days ago", etc.)' };
};

// ============================================================================
// EXPRESSION PREVIEW HELPERS
// ============================================================================

/**
 * Generate human-readable preview of what a condition matches
 */
export const getExpressionPreview = (fieldType: string, operator: string, value: string): string => {
  if (!value) return 'No condition';

  switch (fieldType) {
    case 'string':
      return getStringExpressionPreview(value);
    case 'number':
      return getNumericExpressionPreview(value);
    case 'date':
      return getDateExpressionPreview(value);
    default:
      return value;
  }
};

const getStringExpressionPreview = (expr: string): string => {
  const trimmed = expr.trim();
  if (trimmed === 'EMPTY') return 'Empty or null values';
  if (trimmed === 'NULL') return 'Null values';
  if (trimmed === '-EMPTY') return 'Non-empty values';
  if (trimmed === '-NULL') return 'Non-null values';
  if (trimmed.startsWith('%') && trimmed.endsWith('%')) return `Contains "${trimmed.slice(1, -1)}"`;
  if (trimmed.startsWith('%')) return `Ends with "${trimmed.slice(1)}"`;
  if (trimmed.endsWith('%')) return `Starts with "${trimmed.slice(0, -1)}"`;
  if (trimmed.startsWith('-')) return `Anything except "${trimmed.slice(1)}"`;
  if (trimmed.includes(',')) return `One of: ${trimmed.split(',').join(', ')}`;
  return `Exactly "${trimmed}"`;
};

const getNumericExpressionPreview = (expr: string): string => {
  const trimmed = expr.trim().toUpperCase();
  if (trimmed === 'NULL') return 'Null values';
  if (trimmed === 'NOT NULL') return 'Non-null values';
  if (trimmed.includes('AND')) return `Range: ${trimmed}`;
  if (trimmed.includes('OR')) return `Multiple ranges: ${trimmed}`;
  if (trimmed.startsWith('[')) return `Interval (inclusive): ${trimmed}`;
  if (trimmed.startsWith('(')) return `Interval (exclusive): ${trimmed}`;
  if (trimmed.startsWith('>') || trimmed.startsWith('<') || trimmed.startsWith('=')) {
    return `Numeric: ${trimmed}`;
  }
  return `Exactly ${trimmed}`;
};

const getDateExpressionPreview = (expr: string): string => {
  const trimmed = expr.trim().toLowerCase();
  if (trimmed === 'today') return 'Current day';
  if (trimmed === 'yesterday') return 'Previous day';
  if (trimmed === 'tomorrow') return 'Next day';
  if (trimmed.includes('last')) return `${trimmed.charAt(0).toUpperCase() + trimmed.slice(1)}`;
  if (trimmed.includes('next')) return `${trimmed.charAt(0).toUpperCase() + trimmed.slice(1)}`;
  if (trimmed.includes('this')) return `${trimmed.charAt(0).toUpperCase() + trimmed.slice(1)}`;
  if (trimmed.includes('ago')) return `${trimmed.charAt(0).toUpperCase() + trimmed.slice(1)}`;
  if (trimmed.includes('from now')) return `${trimmed.charAt(0).toUpperCase() + trimmed.slice(1)}`;
  if (trimmed.includes('days')) return `${trimmed.charAt(0).toUpperCase() + trimmed.slice(1)}`;
  if (trimmed.includes('months')) return `${trimmed.charAt(0).toUpperCase() + trimmed.slice(1)}`;
  if (trimmed.includes('weeks')) return `${trimmed.charAt(0).toUpperCase() + trimmed.slice(1)}`;
  if (trimmed.includes('years')) return `${trimmed.charAt(0).toUpperCase() + trimmed.slice(1)}`;
  return trimmed;
};

// ============================================================================
// EXAMPLE SUGGESTIONS
// ============================================================================

const EXPRESSION_EXAMPLES = {
  string: [
    { expr: 'FOO', desc: 'Exactly "FOO"' },
    { expr: 'FOO,BAR', desc: 'Either "FOO" or "BAR"' },
    { expr: '%FOO%', desc: 'Contains "FOO"' },
    { expr: 'FOO%', desc: 'Starts with "FOO"' },
    { expr: '%FOO', desc: 'Ends with "FOO"' },
    { expr: 'EMPTY', desc: 'Empty or null' },
    { expr: '-FOO', desc: 'Anything except "FOO"' },
    { expr: '-%FOO%', desc: 'Does not contain "FOO"' },
  ],
  number: [
    { expr: '5', desc: 'Exactly 5' },
    { expr: '>10', desc: 'Greater than 10' },
    { expr: '>=5 AND <=10', desc: 'Between 5 and 10' },
    { expr: '[5,90]', desc: 'Interval 5 to 90 inclusive' },
    { expr: '(1,7)', desc: 'Interval 1 to 7 exclusive' },
    { expr: 'NOT 5', desc: 'Anything except 5' },
    { expr: '1,3,5,7', desc: 'One of: 1, 3, 5, or 7' },
  ],
  date: [
    { expr: 'today', desc: 'Current day' },
    { expr: 'yesterday', desc: 'Previous day' },
    { expr: 'this week', desc: 'Current week' },
    { expr: 'last 7 days', desc: 'Last 7 days' },
    { expr: '3 days ago', desc: '3 days ago' },
    { expr: 'this month', desc: 'Current month' },
    { expr: '2 months ago for 1 month', desc: 'Specific past month' },
    { expr: 'after 2024-01-01', desc: 'On or after specific date' },
  ],
};

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const AdvancedConditionBuilder: React.FC<AdvancedConditionBuilderProps> = ({
  field,
  operator,
  value,
  fieldType: fieldTypeProp,
  onUpdate,
  onRemove,
  conditionIndex,
  fieldMetadata: _fieldMetadata,
}) => {
  const [showExamples, setShowExamples] = useState(false);

  // Normalize fieldType - accept either string or AdvancedFieldTypeInfo
  const fieldType = useMemo(() => {
    if (typeof fieldTypeProp === 'string') {
      // If it's a string, convert to AdvancedFieldTypeInfo
      return { type: fieldTypeProp as any || 'unknown' } as AdvancedFieldTypeInfo;
    }
    return fieldTypeProp as AdvancedFieldTypeInfo;
  }, [fieldTypeProp]);

  const operators = useMemo(() => {
    const ops = SIMPLE_OPERATORS[fieldType.type as keyof typeof SIMPLE_OPERATORS] || [];
    return ops;
  }, [fieldType.type]);

  const selectedOp = useMemo(() => {
    return operators.find(op => op.value === operator);
  }, [operator, operators]);

  // Determine UI based on operator
  const isStateless = operator === 'is_empty' || operator === 'is_not_empty';
  const isAdvancedMode = operator === 'expressions' || operator === 'relative_dates';

  // Validation
  const validation = useMemo(() => {
    if (!value || isStateless) return { valid: true, message: '' };

    switch (fieldType.type) {
      case 'string':
        if (operator === 'expressions') return validateStringExpression(value);
        return { valid: true, message: '' };
      case 'number':
        if (operator === 'expressions') return validateNumericExpression(value);
        return { valid: true, message: '' };
      case 'date':
        if (operator === 'relative_dates' || operator === 'expressions') {
          return validateDateExpression(value);
        }
        return { valid: true, message: '' };
      default:
        return { valid: true, message: '' };
    }
  }, [value, fieldType.type, operator]);

  const preview = useMemo(() => {
    return getExpressionPreview(fieldType.type, operator, value);
  }, [fieldType.type, operator, value]);

  const examples = useMemo(() => {
    if (isAdvancedMode && fieldType.type in EXPRESSION_EXAMPLES) {
      return EXPRESSION_EXAMPLES[fieldType.type as keyof typeof EXPRESSION_EXAMPLES];
    }
    return [];
  }, [isAdvancedMode, fieldType.type]);

  return (
    <div className="p-4 border rounded bg-gray-50 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          {conditionIndex !== undefined && (
            <h4 className="font-semibold text-gray-900">Condition {conditionIndex + 1}</h4>
          )}
          <p className="text-xs text-gray-600">
            {fieldType.type === 'unknown' ? '(type auto-detected)' : `(${fieldType.type})`}
          </p>
        </div>
        <button
          onClick={onRemove}
          className="text-red-600 hover:bg-red-50 p-1 rounded text-sm"
          title="Remove condition"
        >
          ✕ Remove
        </button>
      </div>

      {/* Field Input */}
      <div>
        <label className="block text-xs font-semibold text-gray-700 mb-2">
          Field
        </label>
        <input
          type="text"
          value={field}
          onChange={(e) => onUpdate({ field: e.target.value })}
          placeholder="e.g., salary, hire_date"
          className="w-full px-3 py-2 border rounded text-sm"
          title="Field name"
        />
      </div>

      {/* Operator Selection */}
      <div>
        <label className="block text-xs font-semibold text-gray-700 mb-2">
          Operator
        </label>
        <select
          value={operator}
          onChange={(e) => onUpdate({ operator: e.target.value, value: '' })}
          className="w-full px-3 py-2 border rounded text-sm"
          title="Condition operator"
        >
          <option value="">Select operator...</option>
          {operators.map(op => (
            <option key={op.value} value={op.value}>
              {op.label} {op.category === 'advanced' ? '⚡' : ''}
            </option>
          ))}
        </select>
        {selectedOp && (
          <p className="text-xs text-gray-600 mt-1">
            {selectedOp.category === 'advanced' && '⚡ Advanced expression mode'}
            {selectedOp.category === 'pattern' && '🔍 Pattern matching'}
            {selectedOp.category === 'comparison' && '📊 Comparison operator'}
            {selectedOp.category === 'state' && '✓ State check (no value needed)'}
          </p>
        )}
      </div>

      {/* Value Input - Conditional */}
      {!isStateless && (
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="text-xs font-semibold text-gray-700">
              {isAdvancedMode ? 'Expression' : 'Value'}
            </label>
            {isAdvancedMode && (
              <button
                onClick={() => setShowExamples(!showExamples)}
                className="text-xs text-blue-600 hover:text-blue-700 flex items-center gap-1"
              >
                <Info size={12} /> Examples
              </button>
            )}
          </div>

          {/* Date-specific input hint for relative dates */}
          {fieldType.type === 'date' && operator === 'relative_dates' && (
            <div className="mb-2 p-2 bg-blue-50 border border-blue-200 rounded text-xs text-blue-700">
              <strong>Examples:</strong> last 7 days, this month, today, last 30 days, this quarter
            </div>
          )}

          {/* Date-specific input for equals/before/after */}
          {fieldType.type === 'date' && (operator === 'equals' || operator === 'before' || operator === 'after') ? (
            <input
              type="date"
              value={value}
              onChange={(e) => onUpdate({ value: e.target.value })}
              className={`w-full px-3 py-2 border rounded text-sm ${
                !validation.valid ? 'border-red-500 bg-red-50' : ''
              }`}
              title="Select a date"
            />
          ) : (
            <textarea
              value={value}
              onChange={(e) => onUpdate({ value: e.target.value })}
              placeholder={
                isAdvancedMode
                  ? 'Enter filter expression (e.g., "last 7 days", ">=10 AND <=100", "%pattern%")'
                  : 'Enter value'
              }
              rows={isAdvancedMode ? 3 : 1}
              className={`w-full px-3 py-2 border rounded text-sm ${
                !validation.valid ? 'border-red-500 bg-red-50' : ''
              }`}
              title={`${isAdvancedMode ? 'Filter expression' : 'Condition value'}`}
            />
          )}

          {/* Validation Feedback */}
          {value && (
            <div className={`mt-2 flex items-start gap-2 text-xs ${
              validation.valid ? 'text-green-700 bg-green-50' : 'text-red-700 bg-red-50'
            } p-2 rounded`}>
              {validation.valid ? <CheckCircle2 size={14} /> : <AlertCircle size={14} />}
              <div>
                <p className="font-medium">{validation.message}</p>
                {validation.valid && preview && (
                  <p className="mt-1 text-gray-600">Preview: {preview}</p>
                )}
              </div>
            </div>
          )}

          {/* Examples Panel */}
          {showExamples && examples.length > 0 && (
            <div className="mt-2 p-3 bg-blue-50 border border-blue-200 rounded">
              <p className="text-xs font-semibold text-blue-900 mb-2">Examples:</p>
              <div className="grid grid-cols-1 gap-2">
                {examples.map((ex, i) => (
                  <button
                    key={i}
                    onClick={() => {
                      onUpdate({ value: ex.expr });
                      setShowExamples(false);
                    }}
                    className="text-left p-2 bg-white border border-blue-200 rounded hover:bg-blue-100 text-xs"
                  >
                    <code className="font-mono text-blue-700 font-semibold">{ex.expr}</code>
                    <p className="text-gray-600">{ex.desc}</p>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Stateless Operator Message */}
      {isStateless && (
        <div className="p-2 bg-blue-50 border border-blue-200 rounded text-xs text-blue-700">
          ✓ Operator '{operator}' checks field state only — no value needed
        </div>
      )}

      {/* Mode Indicator */}
      {isAdvancedMode && (
        <div className="p-2 bg-yellow-50 border border-yellow-200 rounded text-xs text-yellow-800">
          ⚡ <strong>Advanced Mode:</strong> Enter Looker-style filter expressions for powerful filtering
        </div>
      )}
    </div>
  );
};

export default AdvancedConditionBuilder;
