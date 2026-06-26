/**
 * Validation Rule Utilities
 * Reusable functions and constants for validation rule management
 */

export interface ValidationRuleFormData {
  id?: string;
  name: string;
  description: string;
  ruleType: string;
  accountTypes: string[];
  severity: 'BLOCK' | 'WARNING' | 'INFO';
  isActive: boolean;
  evaluationOrder: number;
  allowOverride: boolean;
  requiredAuthority?: string;
  parameters: Record<string, any>;
}

export interface ValidationRule extends ValidationRuleFormData {
  createdAt?: Date;
  updatedAt?: Date;
}

/**
 * Get badge color classes for rule type
 */
export const getRuleTypeBadgeColorClasses = (ruleType: string): string => {
  const colors: Record<string, string> = {
    CONCENTRATION: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-200',
    KYC: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-200',
    ASSET_RESTRICTION: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-200',
    LIQUIDITY: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-200',
    DATA_INTEGRITY: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-200',
    TRADE: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-200',
    FEE: 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/30 dark:text-indigo-200',
    ACCESS_CONTROL: 'bg-rose-100 text-rose-800 dark:bg-rose-900/30 dark:text-rose-200',
  };
  return colors[ruleType] || 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-200';
};

/**
 * Get badge color classes for rule severity
 */
export const getSeverityBadgeColorClasses = (severity: string): string => {
  const colors: Record<string, string> = {
    BLOCK: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-200',
    WARNING: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-200',
    INFO: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-200',
  };
  return colors[severity] || 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-200';
};

/**
 * Get badge color classes for active status
 */
export const getStatusBadgeColorClasses = (isActive: boolean): string => {
  return isActive
    ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-200'
    : 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-200';
};

/**
 * Validate rule form data
 * @returns array of error messages, empty if valid
 */
export const validateRuleForm = (formData: ValidationRuleFormData): string[] => {
  const errors: string[] = [];

  if (!formData.name || !formData.name.trim()) {
    errors.push('Rule name is required');
  } else if (formData.name.length > 100) {
    errors.push('Rule name must be 100 characters or less');
  }

  if (!formData.ruleType) {
    errors.push('Rule type is required');
  }

  if (formData.accountTypes.length === 0) {
    errors.push('At least one account type must be selected');
  }

  if (formData.evaluationOrder < 0) {
    errors.push('Evaluation order must be non-negative');
  }

  if (formData.description && formData.description.length > 500) {
    errors.push('Description must be 500 characters or less');
  }

  return errors;
};

/**
 * Create default form data
 */
export const createDefaultRuleFormData = (): ValidationRuleFormData => ({
  name: '',
  description: '',
  ruleType: 'CONCENTRATION',
  accountTypes: ['ALL_ACCOUNTS'],
  severity: 'BLOCK',
  isActive: true,
  evaluationOrder: 100,
  allowOverride: false,
  parameters: {},
});

/**
 * Build rule form data from existing rule (for editing)
 */
export const buildRuleFormDataFromRule = (rule: any): ValidationRuleFormData => ({
  id: rule.id,
  name: rule.name,
  description: rule.description || '',
  ruleType: rule.ruleType,
  accountTypes: rule.accountTypes || rule.scope || ['ALL_ACCOUNTS'],
  severity: rule.severity || 'BLOCK',
  isActive: rule.isActive !== false,
  evaluationOrder: rule.evaluationOrder || 100,
  allowOverride: rule.allowOverride || false,
  requiredAuthority: rule.requiredAuthority,
  parameters: rule.parameters || {},
});

/**
 * Build create payload from form data
 */
export const buildCreateRulePayload = (
  formData: ValidationRuleFormData,
  tenantId: string | undefined,
  datasourceId: string | undefined
): Record<string, any> => {
  return {
    ...formData,
    tenantId,
    datasourceId,
    scope: formData.accountTypes,
    frequency: 'CONTINUOUS',
    effectiveFrom: new Date(),
  };
};

/**
 * Build update payload from form data
 */
export const buildUpdateRulePayload = (
  formData: ValidationRuleFormData,
  tenantId: string | undefined,
  datasourceId: string | undefined
): Record<string, any> => {
  return {
    ...formData,
    tenantId,
    datasourceId,
  };
};

/**
 * Format account type display string
 */
export const formatAccountTypes = (accountTypes: string[] = []): string => {
  if (accountTypes.length === 0) return 'ALL_ACCOUNTS';
  if (accountTypes.includes('ALL_ACCOUNTS')) return 'ALL_ACCOUNTS';
  return accountTypes.join(', ');
};

/**
 * Check if rule has required fields filled
 */
export const isRuleComplete = (rule: ValidationRuleFormData): boolean => {
  return validateRuleForm(rule).length === 0;
};

/**
 * Deep comparison of two rules (for detecting unsaved changes)
 */
export const hasRuleChanged = (original: ValidationRule, current: ValidationRuleFormData): boolean => {
  if (!original) return true;

  return (
    original.id !== current.id ||
    original.name !== current.name ||
    original.description !== current.description ||
    original.ruleType !== current.ruleType ||
    original.severity !== current.severity ||
    original.isActive !== current.isActive ||
    original.evaluationOrder !== current.evaluationOrder ||
    original.allowOverride !== current.allowOverride ||
    original.requiredAuthority !== current.requiredAuthority ||
    JSON.stringify(original.accountTypes?.sort()) !== JSON.stringify(current.accountTypes?.sort()) ||
    JSON.stringify(original.parameters) !== JSON.stringify(current.parameters)
  );
};
