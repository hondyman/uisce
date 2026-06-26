/**
 * Validation Rules Constants and Metadata
 * Investment Management Platform - Wealth Management Validation Rules
 */

export const RULE_TYPES = {
  CONCENTRATION: {
    id: 'CONCENTRATION',
    label: 'Concentration Limit',
    description: 'No single position exceeds maximum percentage of portfolio',
    category: 'Portfolio Management',
    color: 'bg-blue-50 dark:bg-blue-950/20',
  },
  KYC: {
    id: 'KYC',
    label: 'KYC Completeness',
    description: 'Client profile must have required KYC information',
    category: 'Compliance',
    color: 'bg-purple-50 dark:bg-purple-950/20',
  },
  ASSET_RESTRICTION: {
    id: 'ASSET_RESTRICTION',
    label: 'Asset Restriction',
    description: 'Validate asset types allowed in account structure',
    category: 'Account Management',
    color: 'bg-orange-50 dark:bg-orange-950/20',
  },
  LIQUIDITY: {
    id: 'LIQUIDITY',
    label: 'Liquidity Constraint',
    description: 'Illiquid assets cannot exceed stated percentage',
    category: 'Risk Management',
    color: 'bg-green-50 dark:bg-green-950/20',
  },
  DATA_INTEGRITY: {
    id: 'DATA_INTEGRITY',
    label: 'Data Integrity',
    description: 'Security must exist in master and be actively traded',
    category: 'Data Quality',
    color: 'bg-indigo-50 dark:bg-indigo-950/20',
  },
  TRADE: {
    id: 'TRADE',
    label: 'Trade Execution',
    description: 'Sufficient cash/securities available for trade',
    category: 'Trading',
    color: 'bg-pink-50 dark:bg-pink-950/20',
  },
  FEE: {
    id: 'FEE',
    label: 'Fee Validation',
    description: 'Fees must comply with regulatory limits and agreements',
    category: 'Compliance',
    color: 'bg-yellow-50 dark:bg-yellow-950/20',
  },
  ACCESS_CONTROL: {
    id: 'ACCESS_CONTROL',
    label: 'Advisor Permission',
    description: 'Advisor can only access assigned accounts',
    category: 'Security',
    color: 'bg-red-50 dark:bg-red-950/20',
  },
  ACCOUNT_STRUCTURE: {
    id: 'ACCOUNT_STRUCTURE',
    label: 'Account Structure',
    description: 'Rules related to account structure and regulatory constraints',
    category: 'Account Management',
    color: 'bg-teal-50 dark:bg-teal-950/20',
  },
  PORTFOLIO: {
    id: 'PORTFOLIO',
    label: 'Portfolio Constraint',
    description: 'Rules related to portfolio construction and constraints',
    category: 'Portfolio Management',
    color: 'bg-sky-50 dark:bg-sky-950/20',
  },
  PRICING: {
    id: 'PRICING',
    label: 'Pricing and Valuation',
    description: 'Rules related to pricing and valuation of assets',
    category: 'Data Quality',
    color: 'bg-lime-50 dark:bg-lime-950/20',
  },
} as const;

export const ACCOUNT_TYPES = {
  INDIVIDUAL_ACCOUNT: {
    id: 'INDIVIDUAL_ACCOUNT',
    label: 'Individual Account',
    description: 'Single-owner account',
  },
  JOINT_ACCOUNT: {
    id: 'JOINT_ACCOUNT',
    label: 'Joint Account',
    description: 'Multi-owner joint account',
  },
  TRUST_ACCOUNT: {
    id: 'TRUST_ACCOUNT',
    label: 'Trust Account',
    description: 'Trust account with fiduciary responsibilities',
  },
  IRA_ACCOUNT: {
    id: 'IRA_ACCOUNT',
    label: 'IRA Account',
    description: 'Individual Retirement Account',
  },
  CORPORATE_ACCOUNT: {
    id: 'CORPORATE_ACCOUNT',
    label: 'Corporate Account',
    description: 'Corporate/business account',
  },
  FOUNDATION_ACCOUNT: {
    id: 'FOUNDATION_ACCOUNT',
    label: 'Foundation Account',
    description: 'Charitable foundation account',
  },
} as const;

export const RULE_FREQUENCIES = {
  CONTINUOUS: {
    id: 'CONTINUOUS',
    label: 'Continuous',
    description: 'Evaluated continuously',
  },
  DAILY: {
    id: 'DAILY',
    label: 'Daily',
    description: 'Evaluated once per day',
  },
  WEEKLY: {
    id: 'WEEKLY',
    label: 'Weekly',
    description: 'Evaluated once per week',
  },
  ON_TRADE: {
    id: 'ON_TRADE',
    label: 'On Trade',
    description: 'Evaluated when a trade is executed',
  },
  ON_REBALANCE: {
    id: 'ON_REBALANCE',
    label: 'On Rebalance',
    description: 'Evaluated when portfolio is rebalanced',
  },
  QUARTERLY: {
    id: 'QUARTERLY',
    label: 'Quarterly',
    description: 'Evaluated once per quarter',
  },
  ANNUALLY: {
    id: 'ANNUALLY',
    label: 'Annually',
    description: 'Evaluated once per year',
  },
} as const;

export const SEVERITY_LEVELS = {
  BLOCK: {
    id: 'BLOCK',
    label: 'Block',
    description: 'Blocks the action or transaction',
    color: 'text-red-600 dark:text-red-400',
    bgColor: 'bg-red-100 dark:bg-red-900/30',
    badgeColor: 'bg-red-500 dark:bg-red-600',
    icon: '🚫',
    priority: 3,
  },
  WARNING: {
    id: 'WARNING',
    label: 'Warning',
    description: 'Shows a warning but allows proceeding',
    color: 'text-yellow-600 dark:text-yellow-400',
    bgColor: 'bg-yellow-100 dark:bg-yellow-900/30',
    badgeColor: 'bg-yellow-500 dark:bg-yellow-600',
    icon: '⚠️',
    priority: 2,
  },
  INFO: {
    id: 'INFO',
    label: 'Info',
    description: 'Informational message only',
    color: 'text-blue-600 dark:text-blue-400',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    badgeColor: 'bg-blue-500 dark:bg-blue-600',
    icon: 'ℹ️',
    priority: 1,
  },
} as const;

export const ASSET_TYPES = {
  EQUITY: {
    id: 'EQUITY',
    label: 'Equities',
    description: 'Stocks and equity positions',
    liquid: true,
  },
  FIXED_INCOME: {
    id: 'FIXED_INCOME',
    label: 'Fixed Income',
    description: 'Bonds and fixed income securities',
    liquid: true,
  },
  MUTUAL_FUND: {
    id: 'MUTUAL_FUND',
    label: 'Mutual Funds',
    description: 'Mutual fund investments',
    liquid: true,
  },
  ETF: {
    id: 'ETF',
    label: 'ETFs',
    description: 'Exchange-traded funds',
    liquid: true,
  },
  CASH: {
    id: 'CASH',
    label: 'Cash',
    description: 'Cash and cash equivalents',
    liquid: true,
  },
  ALTERNATIVE: {
    id: 'ALTERNATIVE',
    label: 'Alternatives',
    description: 'Alternative investments',
    liquid: false,
  },
  PRIVATE_EQUITY: {
    id: 'PRIVATE_EQUITY',
    label: 'Private Equity',
    description: 'Private equity investments',
    liquid: false,
  },
  HEDGE_FUND: {
    id: 'HEDGE_FUND',
    label: 'Hedge Funds',
    description: 'Hedge fund investments',
    liquid: false,
  },
  REAL_ESTATE: {
    id: 'REAL_ESTATE',
    label: 'Real Estate',
    description: 'Real estate investments',
    liquid: false,
  },
  CRYPTOCURRENCY: {
    id: 'CRYPTOCURRENCY',
    label: 'Cryptocurrency',
    description: 'Digital assets and cryptocurrencies',
    liquid: true,
  },
} as const;

export const OVERRIDE_CONDITIONS = [
  {
    id: 'INHERITED_POSITION',
    label: 'Inherited Position',
    description: 'Position inherited from beneficiary',
  },
  {
    id: 'FOUNDER_STOCK',
    label: 'Founder Stock',
    description: 'Founder/pre-IPO company stock',
  },
  {
    id: 'RESTRICTED_STOCK',
    label: 'Restricted Stock',
    description: 'Restricted stock with lock-up period',
  },
  {
    id: 'EMPLOYER_STOCK',
    label: 'Employer Stock',
    description: 'Employer/compensation-related stock',
  },
  {
    id: 'DIVIDEND_REINVESTMENT',
    label: 'Dividend Reinvestment',
    description: 'Position from dividend reinvestment',
  },
  {
    id: 'LEGACY_POSITION',
    label: 'Legacy Position',
    description: 'Long-term legacy position',
  },
] as const;

export const REQUIRED_AUTHORITIES = [
  {
    id: 'ADVISOR',
    label: 'Advisor',
    description: 'Advisor authorization required',
  },
  {
    id: 'SUPERVISOR',
    label: 'Supervisor',
    description: 'Supervisor/manager authorization required',
  },
  {
    id: 'COMPLIANCE',
    label: 'Compliance Officer',
    description: 'Compliance officer authorization required',
  },
  {
    id: 'EXECUTIVE',
    label: 'Executive',
    description: 'Executive-level authorization required',
  },
] as const;

export const KYC_REQUIRED_FIELDS = [
  'fullName',
  'dateOfBirth',
  'riskTolerance',
  'investmentObjective',
  'netWorth',
  'accreditedInvestorStatus',
];

export const RISK_TOLERANCE_LEVELS = [
  { id: 'CONSERVATIVE', label: 'Conservative' },
  { id: 'MODERATE', label: 'Moderate' },
  { id: 'AGGRESSIVE', label: 'Aggressive' },
  { id: 'VERY_AGGRESSIVE', label: 'Very Aggressive' },
];

export const INVESTMENT_OBJECTIVES = [
  { id: 'CAPITAL_PRESERVATION', label: 'Capital Preservation' },
  { id: 'INCOME', label: 'Income' },
  { id: 'GROWTH', label: 'Growth' },
  { id: 'AGGRESSIVE_GROWTH', label: 'Aggressive Growth' },
  { id: 'TOTAL_RETURN', label: 'Total Return' },
];

/**
 * Get rule type metadata
 */
export function getRuleTypeMetadata(ruleType: string) {
  const key = ruleType as keyof typeof RULE_TYPES;
  return RULE_TYPES[key] || null;
}

/**
 * Get all rule type options for select/dropdown
 */
export function getRuleTypeOptions() {
  return Object.values(RULE_TYPES).map((rt) => ({
    value: rt.id,
    label: rt.label,
    description: rt.description,
  }));
}

/**
 * Get account type options
 */
export function getAccountTypeOptions() {
  return Object.values(ACCOUNT_TYPES).map((at) => ({
    value: at.id,
    label: at.label,
  }));
}

/**
 * Get frequency options
 */
export function getFrequencyOptions() {
  return Object.values(RULE_FREQUENCIES).map((f) => ({
    value: f.id,
    label: f.label,
  }));
}

/**
 * Get severity options
 */
export function getSeverityOptions() {
  return Object.values(SEVERITY_LEVELS)
    .sort((a, b) => b.priority - a.priority)
    .map((s) => ({
      value: s.id,
      label: s.label,
      color: s.color,
    }));
}

/**
 * Get asset type options
 */
export function getAssetTypeOptions() {
  return Object.values(ASSET_TYPES).map((at) => ({
    value: at.id,
    label: at.label,
    liquid: at.liquid,
  }));
}

/**
 * Get override condition options
 */
export function getOverrideConditionOptions() {
  return OVERRIDE_CONDITIONS.map((oc) => ({
    value: oc.id,
    label: oc.label,
  }));
}

/**
 * Get authority options
 */
export function getAuthorityOptions() {
  return REQUIRED_AUTHORITIES.map((ra) => ({
    value: ra.id,
    label: ra.label,
  }));
}

/**
 * Check if asset type is liquid
 */
export function isLiquidAsset(assetType: string): boolean {
  const key = assetType as keyof typeof ASSET_TYPES;
  const asset = ASSET_TYPES[key];
  return asset ? asset.liquid : true;
}

/**
 * Get illiquid asset types
 */
export function getIlliquidAssetTypes(): string[] {
  return Object.values(ASSET_TYPES)
    .filter((at) => !at.liquid)
    .map((at) => at.id);
}

/**
 * Format percentage for display
 */
export function formatPercentage(value: number, decimals: number = 2): string {
  return `${(value * 100).toFixed(decimals)}%`;
}

/**
 * Format currency for display
 */
export function formatCurrency(value: number, decimals: number = 2): string {
  return `$${value.toLocaleString('en-US', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  })}`;
}

/**
 * Validation rule categories for grouping
 */
export const VALIDATION_CATEGORIES = [
  {
    id: 'compliance',
    label: 'Compliance',
    description: 'Regulatory and compliance rules',
    rules: ['KYC', 'FEE'],
  },
  {
    id: 'portfolio',
    label: 'Portfolio Management',
    description: 'Portfolio and asset management rules',
    rules: ['CONCENTRATION', 'LIQUIDITY', 'ASSET_RESTRICTION'],
  },
  {
    id: 'trading',
    label: 'Trading',
    description: 'Trade execution rules',
    rules: ['TRADE'],
  },
  {
    id: 'quality',
    label: 'Data Quality',
    description: 'Data integrity and quality rules',
    rules: ['DATA_INTEGRITY'],
  },
  {
    id: 'security',
    label: 'Security',
    description: 'Access control and security rules',
    rules: ['ACCESS_CONTROL'],
  },
] as const;

export default {
  RULE_TYPES,
  ACCOUNT_TYPES,
  RULE_FREQUENCIES,
  SEVERITY_LEVELS,
  ASSET_TYPES,
  OVERRIDE_CONDITIONS,
  REQUIRED_AUTHORITIES,
};
