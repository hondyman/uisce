/**
 * Rule Templates - Pre-built validation rule templates
 * Speeds up rule creation by providing common patterns
 */

export interface RuleTemplate {
  id: string;
  name: string;
  description: string;
  category: 'data-quality' | 'business-logic' | 'referential-integrity' | 'format-validation';
  icon: string;
  baseRule: Partial<ValidationRule>;
  helpText: string;
  commonUse: string[];
}

export interface ValidationRule {
  id?: string;
  name: string;
  description: string;
  target_entity: string;
  field_name: string;
  rule_type: 'null-check' | 'range' | 'pattern' | 'lookup' | 'comparison' | 'custom';
  rule_condition: string;
  severity: 'error' | 'warning' | 'info';
  is_enabled: boolean;
  created_at?: string;
  updated_at?: string;
}

/**
 * Pre-built rule templates for common validation scenarios
 */
export const RULE_TEMPLATES: RuleTemplate[] = [
  {
    id: 'not-null',
    name: 'Not Null Check',
    description: 'Ensure a field is never empty',
    category: 'data-quality',
    icon: '🚫',
    baseRule: {
      rule_type: 'null-check',
      rule_condition: 'field IS NOT NULL',
      severity: 'error',
    },
    helpText: 'Creates a rule that flags records where the selected field is empty or null. Perfect for required fields like customer ID or transaction amount.',
    commonUse: [
      'Customer IDs',
      'Transaction dates',
      'Employee names',
      'Account numbers',
    ],
  },

  {
    id: 'unique-values',
    name: 'Uniqueness Check',
    description: 'Ensure field values are unique within the entity',
    category: 'referential-integrity',
    icon: '🔑',
    baseRule: {
      rule_type: 'lookup',
      rule_condition: 'COUNT(*) = 1 FOR EACH field_value',
      severity: 'error',
    },
    helpText: 'Validates that each value appears only once. Useful for IDs, email addresses, and other fields that should be unique.',
    commonUse: [
      'Email addresses',
      'Employee IDs',
      'Account numbers',
      'Social security numbers',
    ],
  },

  {
    id: 'range-validation',
    name: 'Range/Bounds Check',
    description: 'Verify numeric field is within acceptable range',
    category: 'business-logic',
    icon: '📊',
    baseRule: {
      rule_type: 'range',
      rule_condition: 'field BETWEEN 0 AND 100',
      severity: 'warning',
    },
    helpText: 'Ensures numeric fields stay within defined bounds. Great for percentages, ages, scores, and other bounded values.',
    commonUse: [
      'Percentage values',
      'Age verification',
      'Quality scores',
      'Discount rates',
    ],
  },

  {
    id: 'pattern-match',
    name: 'Pattern/Format Check',
    description: 'Validate field matches expected format',
    category: 'format-validation',
    icon: '🔤',
    baseRule: {
      rule_type: 'pattern',
      rule_condition: "field MATCHES '[A-Z]{2}[0-9]{4}'",
      severity: 'error',
    },
    helpText: 'Uses regex patterns to validate formatting. Perfect for phone numbers, postal codes, product codes, and other formatted data.',
    commonUse: [
      'Phone numbers',
      'Email formats',
      'Postal codes',
      'Product codes',
    ],
  },

  {
    id: 'referential-integrity',
    name: 'Referential Integrity',
    description: 'Check foreign key exists in related entity',
    category: 'referential-integrity',
    icon: '🔗',
    baseRule: {
      rule_type: 'lookup',
      rule_condition: 'field IN (SELECT id FROM related_table)',
      severity: 'error',
    },
    helpText: 'Ensures foreign key values exist in the referenced table. Prevents orphaned records.',
    commonUse: [
      'Department IDs',
      'Manager IDs',
      'Category IDs',
      'Region IDs',
    ],
  },

  {
    id: 'comparison-check',
    name: 'Cross-Field Comparison',
    description: 'Compare one field against another',
    category: 'business-logic',
    icon: '⚖️',
    baseRule: {
      rule_type: 'comparison',
      rule_condition: 'start_date < end_date',
      severity: 'warning',
    },
    helpText: 'Validates relationships between two or more fields. Useful for date ranges, price validations, and logical consistency.',
    commonUse: [
      'Start/end dates',
      'Min/max prices',
      'From/to amounts',
      'Begin/end dates',
    ],
  },

  {
    id: 'business-rule',
    name: 'Custom Business Rule',
    description: 'Define custom business logic validation',
    category: 'business-logic',
    icon: '⚙️',
    baseRule: {
      rule_type: 'custom',
      rule_condition: '',
      severity: 'info',
    },
    helpText: 'Create a custom rule for specific business logic. Write your own SQL or logic condition.',
    commonUse: [
      'Complex calculations',
      'Multi-field validations',
      'Seasonal rules',
      'Role-based logic',
    ],
  },

  {
    id: 'duplicate-check',
    name: 'Duplicate Detection',
    description: 'Find duplicate records based on field values',
    category: 'data-quality',
    icon: '👥',
    baseRule: {
      rule_type: 'lookup',
      rule_condition: 'COUNT(*) > 1 FOR EACH field_value',
      severity: 'warning',
    },
    helpText: 'Identifies records with duplicate values in specified fields. Great for spotting data entry errors or test records.',
    commonUse: [
      'Email addresses',
      'Phone numbers',
      'Customer names',
      'Product names',
    ],
  },

  {
    id: 'enum-validation',
    name: 'Enum/Allowed Values',
    description: 'Ensure field contains one of allowed values',
    category: 'format-validation',
    icon: '✓',
    baseRule: {
      rule_type: 'lookup',
      rule_condition: "field IN ('value1', 'value2', 'value3')",
      severity: 'error',
    },
    helpText: 'Validates that a field only contains predefined allowed values. Perfect for status fields and categories.',
    commonUse: [
      'Status fields',
      'Priority levels',
      'Departments',
      'Order states',
    ],
  },
];

/**
 * Get templates by category
 */
export function getTemplatesByCategory(
  category: RuleTemplate['category']
): RuleTemplate[] {
  return RULE_TEMPLATES.filter((t) => t.category === category);
}

/**
 * Get template by ID
 */
export function getTemplateById(id: string): RuleTemplate | undefined {
  return RULE_TEMPLATES.find((t) => t.id === id);
}

/**
 * Get all unique categories
 */
export function getTemplateCategories(): RuleTemplate['category'][] {
  const categories = new Set<RuleTemplate['category']>();
  RULE_TEMPLATES.forEach((t) => categories.add(t.category));
  return Array.from(categories);
}

/**
 * Search templates by name or description
 */
export function searchTemplates(query: string): RuleTemplate[] {
  const lowerQuery = query.toLowerCase();
  return RULE_TEMPLATES.filter(
    (t) =>
      t.name.toLowerCase().includes(lowerQuery) ||
      t.description.toLowerCase().includes(lowerQuery) ||
      t.commonUse.some((use) => use.toLowerCase().includes(lowerQuery))
  );
}
