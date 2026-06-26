/**
 * Parameter Schema Definitions
 * Centralized schema for all parameter types across validation rules, reports, and builders
 * This eliminates duplication and ensures consistency across the platform
 */

export type FieldType = 'text' | 'number' | 'checkbox' | 'select' | 'multiselect' | 'textarea' | 'slider' | 'comma-list';

export interface ParameterField {
  name: string;
  label: string;
  type: FieldType;
  placeholder?: string;
  description?: string;
  required?: boolean;
  validation?: (value: any) => string | null; // Error message or null if valid
  options?: Array<{ value: string; label: string }>;
  min?: number;
  max?: number;
  step?: number;
  rows?: number; // For textarea
  defaultValue?: any;
}

export interface ParameterSchema {
  ruleType: string;
  name: string;
  description: string;
  fields: ParameterField[];
}

/**
 * All parameter schemas for validation rules
 * Schema drives the parameter builder UI
 */
export const PARAMETER_SCHEMAS: Record<string, ParameterSchema> = {
  CONCENTRATION: {
    ruleType: 'CONCENTRATION',
    name: 'Concentration',
    description: 'Control maximum position concentration in portfolios',
    fields: [
      {
        name: 'maxPositionPercentage',
        label: 'Max Position Percentage',
        type: 'number',
        placeholder: 'e.g., 10',
        required: true,
        min: 0,
        max: 100,
        step: 0.1,
        validation: (value) => {
          if (value < 0 || value > 100) return 'Must be between 0 and 100';
          return null;
        },
      },
      {
        name: 'warningThreshold',
        label: 'Warning Threshold',
        type: 'number',
        placeholder: 'e.g., 7.5',
        min: 0,
        max: 100,
        step: 0.1,
      },
      {
        name: 'blockThreshold',
        label: 'Block Threshold',
        type: 'number',
        placeholder: 'e.g., 10',
        min: 0,
        max: 100,
        step: 0.1,
      },
      {
        name: 'minimumPositionSize',
        label: 'Minimum Position Size',
        type: 'number',
        placeholder: 'e.g., 1000',
        min: 0,
      },
    ],
  },
  KYC: {
    ruleType: 'KYC',
    name: 'Know Your Customer',
    description: 'KYC compliance checks and customer verification',
    fields: [
      {
        name: 'requiredFields',
        label: 'Required Fields',
        type: 'comma-list',
        placeholder: 'e.g., fullName,dateOfBirth',
        description: 'Comma-separated list of required KYC fields',
      },
      {
        name: 'pepCheckRequired',
        label: 'PEP Check Required',
        type: 'checkbox',
        description: 'Check if customer is a Politically Exposed Person',
      },
      {
        name: 'sanctionsCheckRequired',
        label: 'Sanctions Check Required',
        type: 'checkbox',
        description: 'Check against sanctions lists',
      },
    ],
  },
  ACCOUNT_STRUCTURE: {
    ruleType: 'ACCOUNT_STRUCTURE',
    name: 'Account Structure',
    description: 'Account setup and structure validation',
    fields: [
      {
        name: 'requireAgeVerification',
        label: 'Require Age Verification',
        type: 'checkbox',
      },
      {
        name: 'erisaComplianceCheck',
        label: 'ERISA Compliance Check',
        type: 'checkbox',
      },
    ],
  },
  PORTFOLIO: {
    ruleType: 'PORTFOLIO',
    name: 'Portfolio Restrictions',
    description: 'Portfolio composition and exposure limits',
    fields: [
      {
        name: 'maxForeignExposurePercentage',
        label: 'Max Foreign Exposure Percentage',
        type: 'number',
        placeholder: 'e.g., 30',
        min: 0,
        max: 100,
        step: 0.1,
      },
    ],
  },
  PRICING: {
    ruleType: 'PRICING',
    name: 'Pricing Validation',
    description: 'Price deviation and reasonableness checks',
    fields: [
      {
        name: 'maxDeviationPercentage',
        label: 'Max Deviation Percentage',
        type: 'number',
        placeholder: 'e.g., 5',
        min: 0,
        step: 0.1,
      },
    ],
  },
  TRADE: {
    ruleType: 'TRADE',
    name: 'Trade Validation',
    description: 'Trade execution and method validation',
    fields: [
      {
        name: 'maxDeviationPercentage',
        label: 'Max Deviation Percentage',
        type: 'number',
        placeholder: 'e.g., 2.5',
        min: 0,
        step: 0.1,
      },
      {
        name: 'allowedMethods',
        label: 'Allowed Methods',
        type: 'comma-list',
        placeholder: 'e.g., FIFO,LIFO,SPECIFIC_ID',
        description: 'Comma-separated list of allowed trade methods',
      },
    ],
  },
  FEE: {
    ruleType: 'FEE',
    name: 'Fee Validation',
    description: 'Fee structure and revenue share limits',
    fields: [
      {
        name: 'maxRevenueSharePercentage',
        label: 'Max Revenue Share Percentage',
        type: 'number',
        placeholder: 'e.g., 15',
        min: 0,
        max: 100,
        step: 0.1,
      },
    ],
  },
  DATA_INTEGRITY: {
    ruleType: 'DATA_INTEGRITY',
    name: 'Data Integrity',
    description: 'Data validation and mathematical accuracy checks',
    fields: [
      {
        name: 'allowZeroCostBasis',
        label: 'Allow Zero Cost Basis',
        type: 'checkbox',
      },
      {
        name: 'requireMathematicalValidation',
        label: 'Require Mathematical Validation',
        type: 'checkbox',
      },
      {
        name: 'maxReasonableCostBasisFactor',
        label: 'Max Reasonable Cost Basis Factor',
        type: 'number',
        placeholder: 'e.g., 2.0',
        min: 0,
        step: 0.1,
      },
    ],
  },
  ASSET_RESTRICTION: {
    ruleType: 'ASSET_RESTRICTION',
    name: 'Asset Restrictions',
    description: 'Restricted and prohibited asset types',
    fields: [
      {
        name: 'prohibitedAssets',
        label: 'Prohibited Assets',
        type: 'comma-list',
        placeholder: 'e.g., ALTERNATIVE,CRYPTOCURRENCY',
        description: 'Comma-separated list of prohibited asset types',
      },
    ],
  },
  LIQUIDITY: {
    ruleType: 'LIQUIDITY',
    name: 'Liquidity Management',
    description: 'Illiquid asset limits and restrictions',
    fields: [
      {
        name: 'maxIlliquidPercentage',
        label: 'Max Illiquid Percentage',
        type: 'number',
        placeholder: 'e.g., 25',
        min: 0,
        max: 100,
        step: 0.1,
      },
      {
        name: 'illiquidAssetTypes',
        label: 'Illiquid Asset Types',
        type: 'comma-list',
        placeholder: 'e.g., PRIVATE_EQUITY,HEDGE_FUND',
        description: 'Comma-separated list of illiquid asset types',
      },
    ],
  },
  ACCESS_CONTROL: {
    ruleType: 'ACCESS_CONTROL',
    name: 'Access Control',
    description: 'User access and fiduciary requirements',
    fields: [
      {
        name: 'requireAdvisorAssignment',
        label: 'Require Advisor Assignment',
        type: 'checkbox',
      },
      {
        name: 'requireFiduciaryCertification',
        label: 'Require Fiduciary Certification',
        type: 'checkbox',
      },
    ],
  },
};

/**
 * Get schema for a specific rule type
 */
export function getParameterSchema(ruleType: string): ParameterSchema | null {
  return PARAMETER_SCHEMAS[ruleType] || null;
}

/**
 * Get all available rule types with their names
 */
export function getAvailableRuleTypes(): Array<{ value: string; label: string; description: string }> {
  return Object.values(PARAMETER_SCHEMAS).map((schema) => ({
    value: schema.ruleType,
    label: schema.name,
    description: schema.description,
  }));
}

/**
 * Validate parameters against schema
 */
export function validateParameters(ruleType: string, parameters: Record<string, any>): Record<string, string> {
  const schema = getParameterSchema(ruleType);
  if (!schema) return {};

  const errors: Record<string, string> = {};

  for (const field of schema.fields) {
    const value = parameters[field.name];

    // Check required fields
    if (field.required && (value === null || value === undefined || value === '')) {
      errors[field.name] = `${field.label} is required`;
      continue;
    }

    // Run field-specific validation
    if (field.validation && value !== null && value !== undefined && value !== '') {
      const error = field.validation(value);
      if (error) {
        errors[field.name] = error;
      }
    }
  }

  return errors;
}

/**
 * Transform comma-list strings to arrays and vice versa for display
 */
export function normalizeParameterValue(field: ParameterField, value: any): any {
  if (field.type === 'comma-list') {
    if (Array.isArray(value)) {
      return value.join(',');
    }
    return value || '';
  }
  return value || '';
}

/**
 * Transform display values back to storage format
 */
export function denormalizeParameterValue(field: ParameterField, value: any): any {
  if (field.type === 'comma-list') {
    if (typeof value === 'string') {
      return value
        .split(',')
        .map((item) => item.trim())
        .filter((item) => item.length > 0);
    }
    return value || [];
  }
  if (field.type === 'number') {
    return value === '' ? null : parseFloat(value);
  }
  return value;
}
