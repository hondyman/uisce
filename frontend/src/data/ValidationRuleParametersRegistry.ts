/**
 * ValidationRuleParametersRegistry.ts
 * 
 * Maps validation rule names to their parameter configurations for dynamic UI rendering.
 * This enables the ValidationRuleCreator and ValidationRuleEditor to dynamically display
 * parameter input fields based on the selected rule.
 * 
 * Structure: Each rule name maps to an array of ParameterConfig objects defining:
 * - fieldName: The parameter key in the rule.parameters object
 * - label: Display label in the UI
 * - type: HTML input type (text, number, checkbox, select, array, object)
 * - required: Whether the field is mandatory
 * - defaultValue: Initial value if not provided
 * - placeholder: Placeholder text for inputs
 * - options: For select types, array of {label, value} options
 * - description: Tooltip/help text
 */

export interface ParameterConfig {
  fieldName: string;
  label: string;
  type: 'text' | 'number' | 'checkbox' | 'select' | 'array' | 'object' | 'textarea';
  required: boolean;
  defaultValue?: any;
  placeholder?: string;
  options?: Array<{ label: string; value: any }>;
  description?: string;
  min?: number;
  max?: number;
  step?: number;
  nestedFields?: ParameterConfig[]; // For object types
}

export const VALIDATION_RULE_PARAMETERS_REGISTRY: Record<string, ParameterConfig[]> = {
  // ===== CORE WEALTH MANAGEMENT RULES =====
  'Concentration Limit': [
    {
      fieldName: 'maxPositionPercentage',
      label: 'Max Position Percentage',
      type: 'number',
      required: true,
      defaultValue: 0.3,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Maximum percentage of portfolio any single position can occupy'
    },
    {
      fieldName: 'warningThreshold',
      label: 'Warning Threshold',
      type: 'number',
      required: true,
      defaultValue: 0.28,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Percentage at which a warning is triggered'
    },
    {
      fieldName: 'blockThreshold',
      label: 'Block Threshold',
      type: 'number',
      required: true,
      defaultValue: 0.35,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Percentage at which trades are blocked'
    },
    {
      fieldName: 'minimumPositionSize',
      label: 'Minimum Position Size (USD)',
      type: 'number',
      required: true,
      defaultValue: 100000,
      min: 0,
      description: 'Minimum position value to apply concentration limit'
    }
  ],

  'KYC Completeness': [
    {
      fieldName: 'requiredFields',
      label: 'Required Fields',
      type: 'array',
      required: true,
      defaultValue: ['fullName', 'dateOfBirth', 'riskTolerance', 'investmentObjective', 'netWorth', 'accreditedInvestorStatus'],
      description: 'Comma-separated list of required KYC fields'
    },
    {
      fieldName: 'pepCheckRequired',
      label: 'PEP Check Required',
      type: 'checkbox',
      required: true,
      defaultValue: true,
      description: 'Whether Politically Exposed Person (PEP) check is required'
    },
    {
      fieldName: 'sanctionsCheckRequired',
      label: 'Sanctions Check Required',
      type: 'checkbox',
      required: true,
      defaultValue: true,
      description: 'Whether international sanctions screening is required'
    },
    {
      fieldName: 'revalidationFrequencyDays',
      label: 'Revalidation Frequency (Days)',
      type: 'number',
      required: true,
      defaultValue: 365,
      min: 30,
      description: 'How often KYC information must be revalidated'
    }
  ],

  'Account Type Restriction': [
    {
      fieldName: 'IRA_ACCOUNT',
      label: 'IRA Account Restrictions',
      type: 'object',
      required: false,
      nestedFields: [
        {
          fieldName: 'prohibitedAssets',
          label: 'Prohibited Assets',
          type: 'array',
          required: false,
          defaultValue: ['ALTERNATIVE', 'CRYPTOCURRENCY', 'PRIVATE_EQUITY'],
          description: 'Asset types not allowed in IRA accounts'
        },
        {
          fieldName: 'maxDerivativePercentage',
          label: 'Max Derivative Percentage',
          type: 'number',
          required: false,
          defaultValue: 0.1,
          min: 0,
          max: 1,
          step: 0.01
        }
      ]
    },
    {
      fieldName: 'TRUST_ACCOUNT',
      label: 'Trust Account Restrictions',
      type: 'object',
      required: false,
      nestedFields: [
        {
          fieldName: 'prohibitedTransactions',
          label: 'Prohibited Transactions',
          type: 'array',
          required: false,
          defaultValue: ['SHORT_SELLING', 'OPTIONS_WRITING'],
          description: 'Transaction types not allowed in trust accounts'
        }
      ]
    }
  ],

  'Liquidity Constraint': [
    {
      fieldName: 'maxIlliquidPercentage',
      label: 'Max Illiquid Percentage',
      type: 'number',
      required: true,
      defaultValue: 0.2,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Maximum percentage of portfolio that can be illiquid'
    },
    {
      fieldName: 'illiquidAssetTypes',
      label: 'Illiquid Asset Types',
      type: 'array',
      required: true,
      defaultValue: ['PRIVATE_EQUITY', 'HEDGE_FUND', 'REAL_ESTATE'],
      description: 'Asset types considered illiquid'
    },
    {
      fieldName: 'flagThreshold',
      label: 'Flag Threshold',
      type: 'number',
      required: true,
      defaultValue: 0.18,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Percentage at which illiquidity flag is raised'
    }
  ],

  'Trade Execution': [
    {
      fieldName: 'cashBuffer',
      label: 'Cash Buffer Percentage',
      type: 'number',
      required: true,
      defaultValue: 0.01,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Minimum cash percentage required for trade execution'
    },
    {
      fieldName: 'requireT2Settlement',
      label: 'Require T+2 Settlement',
      type: 'checkbox',
      required: true,
      defaultValue: true,
      description: 'Whether trades must settle in T+2 format'
    }
  ],

  'Fee Validation': [
    {
      fieldName: 'maxAdvisoryFeePercentage',
      label: 'Max Advisory Fee (%)',
      type: 'number',
      required: true,
      defaultValue: 0.02,
      min: 0,
      max: 1,
      step: 0.0001,
      description: 'Maximum annual advisory fee as percentage of AUM'
    },
    {
      fieldName: 'maxPerformanceFeePercentage',
      label: 'Max Performance Fee (%)',
      type: 'number',
      required: true,
      defaultValue: 0.25,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Maximum performance/incentive fee percentage'
    },
    {
      fieldName: 'reasonableFeeThreshold',
      label: 'Reasonable Fee Threshold (%)',
      type: 'number',
      required: true,
      defaultValue: 0.015,
      min: 0,
      max: 1,
      step: 0.0001,
      description: 'Threshold below which fees are considered reasonable'
    }
  ],

  'Advisor Permission': [
    {
      fieldName: 'requireAdvisorAssignment',
      label: 'Require Advisor Assignment',
      type: 'checkbox',
      required: true,
      defaultValue: true,
      description: 'Whether advisor must be explicitly assigned to account'
    },
    {
      fieldName: 'requireFiduciaryCertification',
      label: 'Require Fiduciary Certification',
      type: 'checkbox',
      required: true,
      defaultValue: true,
      description: 'Whether advisor must hold fiduciary certification'
    }
  ],

  'Beneficiary Validation': [
    {
      fieldName: 'requireAgeVerification',
      label: 'Require Age Verification',
      type: 'checkbox',
      required: true,
      defaultValue: true,
      description: 'Whether beneficiary age must be verified'
    },
    {
      fieldName: 'disallowMinorsForCertainAccounts',
      label: 'Disallow Minors For',
      type: 'array',
      required: false,
      defaultValue: ['IRA_ACCOUNT'],
      description: 'Account types that cannot have minors as beneficiaries'
    },
    {
      fieldName: 'erisaComplianceCheck',
      label: 'ERISA Compliance Check',
      type: 'checkbox',
      required: false,
      defaultValue: true,
      description: 'Whether ERISA compliance rules should be enforced'
    }
  ],

  'Currency Exposure Limit': [
    {
      fieldName: 'exposureLimits',
      label: 'Currency Exposure Limits',
      type: 'array',
      required: true,
      description: 'Array of risk tolerance levels with corresponding max foreign exposure percentages'
    }
  ],

  'Fair Value Validation': [
    {
      fieldName: 'maxDeviationPercentage',
      label: 'Max Price Deviation (%)',
      type: 'number',
      required: true,
      defaultValue: 0.1,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Maximum acceptable deviation from benchmark price'
    },
    {
      fieldName: 'benchmarkSource',
      label: 'Benchmark Source',
      type: 'text',
      required: true,
      defaultValue: 'default',
      description: 'Primary source for benchmark pricing data'
    }
  ],

  'Cost Basis Validation': [
    {
      fieldName: 'allowZeroCostBasis',
      label: 'Allow Zero Cost Basis',
      type: 'checkbox',
      required: true,
      defaultValue: false,
      description: 'Whether positions can have zero cost basis'
    },
    {
      fieldName: 'maxReasonableCostBasisFactor',
      label: 'Max Reasonable Cost Basis Factor',
      type: 'number',
      required: true,
      defaultValue: 1.5,
      min: 1,
      step: 0.1,
      description: 'Maximum factor of current price for historical cost basis'
    }
  ],

  // ===== ADVANCED WEALTH MANAGEMENT RULES =====
  'Tax Optimization': [
    {
      fieldName: 'maxTaxableGainPercentage',
      label: 'Max Taxable Gain Percentage',
      type: 'number',
      required: true,
      defaultValue: 0.15,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Maximum percentage of portfolio that should realize taxable gains in a period'
    },
    {
      fieldName: 'washSaleWindowDays',
      label: 'Wash Sale Window (Days)',
      type: 'number',
      required: true,
      defaultValue: 30,
      min: 1,
      description: 'Number of days to avoid repurchasing substantially identical securities'
    },
    {
      fieldName: 'taxBracketThresholds',
      label: 'Tax Bracket Thresholds',
      type: 'array',
      required: true,
      description: 'Array of tax brackets with maximum gain limits'
    }
  ],

  'ESG Compliance': [
    {
      fieldName: 'minEsgScore',
      label: 'Minimum ESG Score',
      type: 'number',
      required: true,
      defaultValue: 7.0,
      min: 0,
      max: 10,
      step: 0.5,
      description: 'Minimum MSCI ESG rating for holdings'
    },
    {
      fieldName: 'maxEsgScoreDeviation',
      label: 'Max ESG Score Deviation',
      type: 'number',
      required: true,
      defaultValue: 2.0,
      min: 0,
      step: 0.1,
      description: 'Maximum deviation from target ESG score'
    },
    {
      fieldName: 'restrictedSectors',
      label: 'Restricted Sectors',
      type: 'array',
      required: false,
      defaultValue: ['OIL_GAS', 'TOBACCO', 'WEAPONS'],
      description: 'Sectors to exclude from portfolio'
    },
    {
      fieldName: 'esgDataSource',
      label: 'ESG Data Source',
      type: 'select',
      required: true,
      defaultValue: 'msci_api',
      options: [
        { label: 'MSCI API', value: 'msci_api' },
        { label: 'Sustainalytics', value: 'sustainalytics' },
        { label: 'Bloomberg ESG', value: 'bloomberg_esg' },
        { label: 'Refinitiv', value: 'refinitiv' }
      ],
      description: 'Data provider for ESG ratings'
    },
    {
      fieldName: 'integrationEndpoint',
      label: 'Integration Endpoint',
      type: 'text',
      required: true,
      defaultValue: 'https://api.msci.com/esg-ratings',
      description: 'API endpoint for ESG data provider'
    }
  ],

  'Regulatory Margin Compliance': [
    {
      fieldName: 'initialMarginLimit',
      label: 'Initial Margin Limit',
      type: 'number',
      required: true,
      defaultValue: 0.5,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Maximum initial margin requirement (FINRA 4210)'
    },
    {
      fieldName: 'maintenanceMarginLimit',
      label: 'Maintenance Margin Limit',
      type: 'number',
      required: true,
      defaultValue: 0.25,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Maintenance margin minimum requirement'
    },
    {
      fieldName: 'maxLoanValue',
      label: 'Max Loan Value (USD)',
      type: 'number',
      required: true,
      defaultValue: 1000000,
      min: 0,
      description: 'Maximum margin loan value allowed'
    },
    {
      fieldName: 'marginCallThreshold',
      label: 'Margin Call Threshold',
      type: 'number',
      required: true,
      defaultValue: 0.3,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Margin level at which call is triggered'
    },
    {
      fieldName: 'regulatoryFramework',
      label: 'Regulatory Framework',
      type: 'select',
      required: true,
      defaultValue: 'FINRA_4210',
      options: [
        { label: 'FINRA Rule 4210', value: 'FINRA_4210' },
        { label: 'SEC Regulation T', value: 'SEC_REG_T' },
        { label: 'MiFID II', value: 'MIFID_II' }
      ],
      description: 'Regulatory framework for margin requirements'
    }
  ],

  'Portfolio Drift Detection': [
    {
      fieldName: 'maxDriftPercentage',
      label: 'Max Drift Percentage',
      type: 'number',
      required: true,
      defaultValue: 0.05,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Maximum acceptable deviation from target allocation'
    },
    {
      fieldName: 'rebalancingThreshold',
      label: 'Rebalancing Threshold',
      type: 'number',
      required: true,
      defaultValue: 0.08,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Drift percentage at which rebalancing is recommended'
    },
    {
      fieldName: 'targetAllocations',
      label: 'Target Allocations',
      type: 'object',
      required: true,
      description: 'Target allocation percentages by asset class'
    }
  ],

  'Communication Compliance': [
    {
      fieldName: 'prohibitedPhrases',
      label: 'Prohibited Phrases',
      type: 'array',
      required: true,
      defaultValue: ['guaranteed return', 'risk-free', 'assured profit', 'no risk'],
      description: 'Phrases prohibited in advisor communications'
    },
    {
      fieldName: 'requiredDisclosures',
      label: 'Required Disclosures',
      type: 'array',
      required: true,
      defaultValue: ['past performance disclaimer', 'fee disclosure', 'conflict of interest'],
      description: 'Required disclosures in all communications'
    },
    {
      fieldName: 'regulatoryFramework',
      label: 'Regulatory Framework',
      type: 'select',
      required: true,
      defaultValue: 'SEC_206_4_1',
      options: [
        { label: 'SEC Rule 206(4)-1', value: 'SEC_206_4_1' },
        { label: 'FINRA Rule 2210', value: 'FINRA_2210' },
        { label: 'MiFID II Article 24', value: 'MIFID_II_24' }
      ],
      description: 'Regulatory framework for communications'
    }
  ],

  // ===== COMPETITIVE MANAGEMENT RULES =====
  'AI-Driven Risk Assessment': [
    {
      fieldName: 'maxVaR',
      label: 'Maximum Value-at-Risk (VaR)',
      type: 'number',
      required: true,
      defaultValue: 0.05,
      min: 0,
      max: 1,
      step: 0.01,
      description: '5% VaR - maximum acceptable portfolio loss at 95% confidence level'
    },
    {
      fieldName: 'varConfidenceLevel',
      label: 'VaR Confidence Level',
      type: 'number',
      required: true,
      defaultValue: 0.95,
      min: 0.5,
      max: 0.99,
      step: 0.01,
      description: 'Statistical confidence level for VaR calculation'
    },
    {
      fieldName: 'stressTestScenarios',
      label: 'Stress Test Scenarios',
      type: 'array',
      required: true,
      defaultValue: ['market_crash_10', 'interest_rate_spike', 'currency_volatility'],
      description: 'Market scenarios for stress testing'
    },
    {
      fieldName: 'aiModelEndpoint',
      label: 'AI Model Endpoint',
      type: 'text',
      required: true,
      defaultValue: 'https://api.sagemaker.example.com/risk-model',
      description: 'URL of the AI risk model service (AWS SageMaker, TensorFlow, etc.)'
    },
    {
      fieldName: 'modelType',
      label: 'Model Type',
      type: 'select',
      required: true,
      defaultValue: 'tensorflow_var',
      options: [
        { label: 'TensorFlow VaR', value: 'tensorflow_var' },
        { label: 'AWS SageMaker', value: 'sagemaker_xgboost' },
        { label: 'PyTorch LSTM', value: 'pytorch_lstm' },
        { label: 'Custom Model', value: 'custom_model' }
      ],
      description: 'Type of AI model for risk assessment'
    },
    {
      fieldName: 'integrationTimeout',
      label: 'Integration Timeout (ms)',
      type: 'number',
      required: true,
      defaultValue: 30000,
      min: 5000,
      step: 1000,
      description: 'Maximum time to wait for AI model response'
    }
  ],

  'Client Engagement Tracking': [
    {
      fieldName: 'minInteractionFrequencyDays',
      label: 'Min Interaction Frequency (Days)',
      type: 'number',
      required: true,
      defaultValue: 90,
      min: 1,
      description: 'Minimum days between required client interactions'
    },
    {
      fieldName: 'triggerEvents',
      label: 'Trigger Events',
      type: 'array',
      required: true,
      defaultValue: ['portfolio_drop_10_percent', 'portfolio_gain_15_percent', 'rebalancing_required', 'significant_market_event'],
      description: 'Events that trigger client engagement notifications'
    },
    {
      fieldName: 'notificationChannel',
      label: 'Notification Channel',
      type: 'select',
      required: true,
      defaultValue: 'email',
      options: [
        { label: 'Email', value: 'email' },
        { label: 'SMS', value: 'sms' },
        { label: 'Portal', value: 'portal' },
        { label: 'All', value: 'all' }
      ],
      description: 'How advisors are notified of engagement triggers'
    },
    {
      fieldName: 'escalationThreshold',
      label: 'Escalation Threshold (Days)',
      type: 'number',
      required: true,
      defaultValue: 180,
      min: 1,
      description: 'Days without contact before escalation to supervisor'
    }
  ],

  'Performance Benchmarking': [
    {
      fieldName: 'benchmarkIndex',
      label: 'Primary Benchmark Index',
      type: 'select',
      required: true,
      defaultValue: 'SP500',
      options: [
        { label: 'S&P 500', value: 'SP500' },
        { label: 'NASDAQ', value: 'NASDAQ' },
        { label: 'Russell 2000', value: 'RUSSELL2000' },
        { label: 'MSCI World', value: 'MSCI_WORLD' },
        { label: 'FTSE 100', value: 'FTSE_100' }
      ],
      description: 'Primary benchmark for performance comparison'
    },
    {
      fieldName: 'secondaryBenchmarks',
      label: 'Secondary Benchmarks',
      type: 'array',
      required: false,
      defaultValue: ['NASDAQ', 'RUSSELL2000', 'MSCI_WORLD'],
      description: 'Additional benchmarks for comparison'
    },
    {
      fieldName: 'minPerformanceDelta',
      label: 'Min Performance Delta',
      type: 'number',
      required: true,
      defaultValue: -0.02,
      min: -1,
      max: 1,
      step: 0.01,
      description: 'Minimum acceptable performance vs. benchmark (e.g., -0.02 = 2% underperformance allowed)'
    },
    {
      fieldName: 'evaluationPeriodMonths',
      label: 'Evaluation Period (Months)',
      type: 'number',
      required: true,
      defaultValue: 12,
      min: 1,
      step: 1,
      description: 'Number of months to evaluate performance'
    },
    {
      fieldName: 'dataSource',
      label: 'Data Source',
      type: 'select',
      required: true,
      defaultValue: 'bloomberg_api',
      options: [
        { label: 'Bloomberg', value: 'bloomberg_api' },
        { label: 'Refinitiv', value: 'refinitiv' },
        { label: 'Yahoo Finance', value: 'yahoo_finance' },
        { label: 'Custom API', value: 'custom_api' }
      ],
      description: 'Data provider for benchmark performance'
    },
    {
      fieldName: 'integrationEndpoint',
      label: 'Integration Endpoint',
      type: 'text',
      required: true,
      defaultValue: 'https://api.bloomberg.com/benchmark-data',
      description: 'API endpoint for benchmark data provider'
    }
  ],

  'Anti-Money Laundering (AML) Compliance': [
    {
      fieldName: 'transactionThreshold',
      label: 'Transaction Threshold (USD)',
      type: 'number',
      required: true,
      defaultValue: 10000,
      min: 1000,
      description: 'Individual transaction amount requiring reporting'
    },
    {
      fieldName: 'cumulativeThreshold',
      label: 'Cumulative Threshold (USD)',
      type: 'number',
      required: true,
      defaultValue: 50000,
      min: 1000,
      description: 'Cumulative transaction amount within time window requiring reporting'
    },
    {
      fieldName: 'cumulativeWindowDays',
      label: 'Cumulative Window (Days)',
      type: 'number',
      required: true,
      defaultValue: 30,
      min: 1,
      description: 'Time window for cumulative transaction monitoring'
    },
    {
      fieldName: 'suspiciousPatterns',
      label: 'Suspicious Patterns',
      type: 'array',
      required: true,
      defaultValue: ['rapid_transfers', 'high_frequency_small_amounts', 'round_number_trades'],
      description: 'Transaction patterns to flag for investigation'
    },
    {
      fieldName: 'amlScreeningService',
      label: 'AML Screening Service',
      type: 'select',
      required: true,
      defaultValue: 'world_check_api',
      options: [
        { label: 'World-Check', value: 'world_check_api' },
        { label: 'OFAC List', value: 'ofac_list' },
        { label: 'Dow Jones Watchlist', value: 'dow_jones_watchlist' }
      ],
      description: 'External AML screening service'
    },
    {
      fieldName: 'integrationEndpoint',
      label: 'Integration Endpoint',
      type: 'text',
      required: true,
      defaultValue: 'https://api.world-check.com/screen',
      description: 'API endpoint for AML screening service'
    },
    {
      fieldName: 'reportingRequirement',
      label: 'Reporting Requirement',
      type: 'select',
      required: true,
      defaultValue: 'SAR',
      options: [
        { label: 'Suspicious Activity Report (SAR)', value: 'SAR' },
        { label: 'Currency Transaction Report (CTR)', value: 'CTR' },
        { label: 'Both', value: 'BOTH' }
      ],
      description: 'Type of regulatory report required'
    }
  ],

  'Alternative Investments Eligibility': [
    {
      fieldName: 'minNetWorth',
      label: 'Minimum Net Worth (USD)',
      type: 'number',
      required: true,
      defaultValue: 2000000,
      min: 0,
      description: 'Minimum net worth for alternative investment eligibility'
    },
    {
      fieldName: 'minAnnualIncome',
      label: 'Minimum Annual Income (USD)',
      type: 'number',
      required: true,
      defaultValue: 200000,
      min: 0,
      description: 'Minimum annual income for alternative investment eligibility'
    },
    {
      fieldName: 'maxAlternativeAllocation',
      label: 'Max Alternative Allocation',
      type: 'number',
      required: true,
      defaultValue: 0.2,
      min: 0,
      max: 1,
      step: 0.01,
      description: 'Maximum percentage of portfolio in alternative investments'
    },
    {
      fieldName: 'alternativeAssetTypes',
      label: 'Alternative Asset Types',
      type: 'array',
      required: true,
      defaultValue: ['PRIVATE_EQUITY', 'HEDGE_FUND', 'REAL_ESTATE', 'COMMODITIES'],
      description: 'Types of alternative investments allowed'
    },
    {
      fieldName: 'requiredAccreditation',
      label: 'Require Accreditation',
      type: 'checkbox',
      required: true,
      defaultValue: true,
      description: 'Whether client must be accredited investor'
    },
    {
      fieldName: 'accreditationRevalidationDays',
      label: 'Accreditation Revalidation (Days)',
      type: 'number',
      required: true,
      defaultValue: 365,
      min: 30,
      description: 'How often accreditation status must be revalidated'
    }
  ]
};

/**
 * Get parameter configurations for a specific rule
 */
export function getParametersForRule(ruleName: string): ParameterConfig[] {
  return VALIDATION_RULE_PARAMETERS_REGISTRY[ruleName] || [];
}

/**
 * Get a specific parameter configuration
 */
export function getParameterConfig(
  ruleName: string,
  fieldName: string
): ParameterConfig | undefined {
  const params = getParametersForRule(ruleName);
  return params.find(p => p.fieldName === fieldName);
}

/**
 * Validate parameter values against their configurations
 */
export function validateParameters(
  ruleName: string,
  parameters: Record<string, any>
): { valid: boolean; errors: string[] } {
  const errors: string[] = [];
  const configs = getParametersForRule(ruleName);

  for (const config of configs) {
    if (config.required && !(config.fieldName in parameters)) {
      errors.push(`Missing required parameter: ${config.label}`);
      continue;
    }

    const value = parameters[config.fieldName];
    if (value === undefined || value === null) continue;

    // Type validation
    if (config.type === 'number') {
      if (typeof value !== 'number') {
        errors.push(`${config.label} must be a number`);
      }
      if (config.min !== undefined && value < config.min) {
        errors.push(`${config.label} must be at least ${config.min}`);
      }
      if (config.max !== undefined && value > config.max) {
        errors.push(`${config.label} must be at most ${config.max}`);
      }
    } else if (config.type === 'array') {
      if (!Array.isArray(value)) {
        errors.push(`${config.label} must be an array`);
      }
    }
  }

  return { valid: errors.length === 0, errors };
}

export default VALIDATION_RULE_PARAMETERS_REGISTRY;
