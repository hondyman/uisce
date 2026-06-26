/**
 * Name validation utilities for cube and view names
 * 
 * Common rules apply to names of entities within the data model. All names must:
 * - Start with a letter
 * - Consist of letters, numbers, and underscore (_) symbols only
 * - Not be a reserved keyword in Python
 * - When using the DAX API, not clash with the names of columns in date hierarchies
 * - Use snake_case (recommended)
 */

// Python reserved keywords that should not be used as names
const PYTHON_RESERVED_KEYWORDS = new Set([
  'False', 'None', 'True', '__peg_parser__', 'and', 'as', 'assert', 'async', 'await',
  'break', 'class', 'continue', 'def', 'del', 'elif', 'else', 'except', 'finally',
  'for', 'from', 'global', 'if', 'import', 'in', 'is', 'lambda', 'nonlocal',
  'not', 'or', 'pass', 'raise', 'return', 'try', 'while', 'with', 'yield'
]);

// DAX date hierarchy column names that should be avoided
const DAX_DATE_HIERARCHY_NAMES = new Set([
  'year', 'quarter', 'month', 'day', 'hour', 'minute', 'second',
  'date', 'time', 'datetime', 'timestamp'
]);

export interface NameValidationResult {
  isValid: boolean;
  errors: string[];
  warnings: string[];
  suggestions?: string;
}

/**
 * Validates a cube or view name according to the naming rules
 */
export function validateEntityName(name: string, type: 'cube' | 'view' | 'measure' | 'dimension' | 'pre-aggregation' = 'cube'): NameValidationResult {
  const result: NameValidationResult = {
    isValid: true,
    errors: [],
    warnings: []
  };

  if (!name || name.trim() === '') {
    result.isValid = false;
    result.errors.push('Name cannot be empty');
    return result;
  }

  const trimmedName = name.trim();

  // Rule 1: Must start with a letter
  if (!/^[a-zA-Z]/.test(trimmedName)) {
    result.isValid = false;
    result.errors.push('Name must start with a letter');
  }

  // Rule 2: Must consist only of letters, numbers, and underscores
  if (!/^[a-zA-Z][a-zA-Z0-9_]*$/.test(trimmedName)) {
    result.isValid = false;
    result.errors.push('Name can only contain letters, numbers, and underscores');
  }

  // Rule 3: Must not be a Python reserved keyword
  if (PYTHON_RESERVED_KEYWORDS.has(trimmedName.toLowerCase())) {
    result.isValid = false;
    result.errors.push(`"${trimmedName}" is a reserved Python keyword and cannot be used`);
  }

  // Rule 4: DAX API - should not clash with date hierarchy names
  if (DAX_DATE_HIERARCHY_NAMES.has(trimmedName.toLowerCase())) {
    result.warnings.push(`"${trimmedName}" may clash with DAX date hierarchy columns`);
  }

  // Recommendation: Use snake_case
  if (trimmedName !== toSnakeCase(trimmedName)) {
    result.warnings.push('Consider using snake_case for better consistency');
    result.suggestions = toSnakeCase(trimmedName);
  }

  // Additional validation for specific types
  if (type === 'measure' || type === 'dimension') {
    // These should typically be shorter and more descriptive
    if (trimmedName.length > 50) {
      result.warnings.push(`${type} name is quite long (${trimmedName.length} characters). Consider a shorter name for better readability`);
    }
  }

  return result;
}

/**
 * Converts a string to snake_case
 */
export function toSnakeCase(str: string): string {
  return str
    // Handle camelCase and PascalCase
    .replace(/([a-z])([A-Z])/g, '$1_$2')
    // Handle spaces and hyphens
    .replace(/[\s-]+/g, '_')
    // Convert to lowercase
    .toLowerCase()
    // Remove any invalid characters
    .replace(/[^a-z0-9_]/g, '')
    // Ensure it starts with a letter
    .replace(/^[^a-z]+/, '')
    // Remove multiple consecutive underscores
    .replace(/_+/g, '_')
    // Remove trailing underscores
    .replace(/_+$/, '');
}

/**
 * Validates multiple names in a batch
 */
export function validateEntityNames(names: Array<{ name: string; type?: 'cube' | 'view' | 'measure' | 'dimension' | 'pre-aggregation' }>): Array<{ name: string; validation: NameValidationResult }> {
  const results = names.map(({ name, type = 'cube' }) => ({
    name,
    validation: validateEntityName(name, type)
  }));

  // Check for duplicates
  const nameCount = new Map<string, number>();
  names.forEach(({ name }) => {
    const lowerName = name.toLowerCase();
    nameCount.set(lowerName, (nameCount.get(lowerName) || 0) + 1);
  });

  // Add duplicate warnings
  results.forEach(result => {
    const count = nameCount.get(result.name.toLowerCase());
    if (count && count > 1) {
      result.validation.warnings.push(`Duplicate name detected: "${result.name}" appears ${count} times`);
    }
  });

  return results;
}

/**
 * Generates good example names for different entity types
 */
export function getExampleNames(type: 'cube' | 'view' | 'measure' | 'dimension' | 'pre-aggregation'): string[] {
  switch (type) {
    case 'cube':
      return ['orders', 'stripe_invoices', 'base_payments', 'customer_accounts', 'product_catalog'];
    case 'view':
      return ['opportunities', 'cloud_accounts', 'arr_monthly', 'sales_dashboard', 'user_analytics'];
    case 'measure':
      return ['count', 'avg_price', 'total_amount_shipped', 'revenue_ytd', 'active_users'];
    case 'dimension':
      return ['name', 'is_shipped', 'created_at', 'customer_tier', 'region_code'];
    case 'pre-aggregation':
      return ['main', 'orders_by_status', 'lambda_invoices', 'daily_rollup', 'user_metrics'];
    default:
      return ['example_name', 'another_example'];
  }
}
