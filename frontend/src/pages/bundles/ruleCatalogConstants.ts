/**
 * Rule Categories and Types
 *
 * Constants and type definitions for the Rules Catalog
 */

export interface RuleCategory {
  id: string;
  name: string;
  description: string;
  icon: string;
  color: string;
  ruleIds: string[];
}

// Define rule categories based on business domain
export const RULE_CATEGORIES: RuleCategory[] = [
  {
    id: 'esg',
    name: 'ESG & Sustainability',
    description: 'Environmental, Social & Governance compliance rules',
    icon: '🌱',
    color: '#10B981',
    ruleIds: ['esg-compliance-v1']
  },
  {
    id: 'private-capital',
    name: 'Private Capital',
    description: 'Private equity, hedge funds, alternative investments',
    icon: '💼',
    color: '#8B5CF6',
    ruleIds: ['alternative-investments-v1', 'accredited-investor-revalidation-v1']
  },
  {
    id: 'mutual-funds',
    name: 'Mutual Funds',
    description: 'Fund operations, allocations, performance',
    icon: '📊',
    color: '#3B82F6',
    ruleIds: ['performance-benchmarking-v1', 'rebalancing-rules-v1', 'portfolio-drift-v1']
  },
  {
    id: 'funds-accounting',
    name: 'Funds Accounting',
    description: 'Cost basis, fees, revenue sharing, positions',
    icon: '📝',
    color: '#F59E0B',
    ruleIds: [
      'cost-basis-validation-v1',
      'fee-validation-v1',
      'revenue-sharing-validation-v1',
      'position-existence-v1',
      'corporate-action-validation-v1'
    ]
  },
  {
    id: 'risk-management',
    name: 'Risk Management',
    description: 'Portfolio risk, margin, concentration limits',
    icon: '⚠️',
    color: '#EF4444',
    ruleIds: [
      'concentration-limit-v1',
      'margin-compliance-v1',
      'liquidity-constraint-v1',
      'ai-risk-assessment-v1'
    ]
  },
  {
    id: 'compliance',
    name: 'Compliance & Regulatory',
    description: 'KYC, AML, communications, tax, regulatory',
    icon: '⚖️',
    color: '#059669',
    ruleIds: [
      'kyc-completeness-v1',
      'aml-compliance-v1',
      'communication-compliance-v1',
      'tax-optimization-v1',
      'beneficiary-validation-v1'
    ]
  },
  {
    id: 'access-control',
    name: 'Access & Permissions',
    description: 'Advisor permissions, account restrictions',
    icon: '🔐',
    color: '#DC2626',
    ruleIds: ['advisor-permission-v1', 'account-type-restriction-v1']
  },
  {
    id: 'client-experience',
    name: 'Client Experience',
    description: 'Client engagement, communications, reporting',
    icon: '👥',
    color: '#06B6D4',
    ruleIds: ['client-engagement-v1', 'investment-profile-alignment-v1']
  },
  {
    id: 'trade-execution',
    name: 'Trade & Settlement',
    description: 'Trade execution, settlement, tax lots',
    icon: '💱',
    color: '#7C3AED',
    ruleIds: ['trade-execution-v1', 'tax-lot-selection-v1']
  },
  {
    id: 'data-integrity',
    name: 'Data Integrity',
    description: 'Data validation, temporal consistency, fair value',
    icon: '✓',
    color: '#16A34A',
    ruleIds: [
      'temporal-consistency-v1',
      'fair-value-validation-v1',
      'currency-exposure-v1',
      'net-worth-verification-v1'
    ]
  }
];

export interface FilterOptions {
  search: string;
  categories: string[];
  severities: string[];
  frequencies: string[];
  ruleTypes: string[];
  isCore?: boolean;
  sortBy: 'evaluationOrder' | 'name' | 'severity';
}

export interface RuleCatalogItem {
  rule: any; // We'll type this properly when we import the rule type
  categories: RuleCategory[];
  parameters: any[];
}

export type ViewMode = 'grid' | 'list' | 'compare';
export type SortOption = 'evaluationOrder' | 'name' | 'severity';