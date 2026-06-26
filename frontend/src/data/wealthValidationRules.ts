// Wealth management validation rules seed data
// These are sample rules adapted from the WealthManagementValidationEngine definitions
export const WEALTH_VALIDATION_RULES = [
  {
    id: 'concentration-limit-v1',
    name: 'Concentration Limit',
    description: 'No single position exceeds maximum percentage of portfolio',
    rule_type: 'business_logic', // Changed from 'CONCENTRATION'
    scope: ['INDIVIDUAL_ACCOUNT', 'JOINT_ACCOUNT'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'CONTINUOUS',
    evaluationOrder: 1,
    overrideConditions: ['INHERITED_POSITION', 'FOUNDER_STOCK'],
    requiredAuthority: 'SUPERVISOR',
    parameters: {
      maxPositionPercentage: 0.3,
      warningThreshold: 0.28,
      blockThreshold: 0.35,
      minimumPositionSize: 100000
    }
  },
  {
    id: 'kyc-completeness-v1',
    name: 'KYC Completeness',
    description: 'Client profile must have required KYC information',
    rule_type: 'field_format', // Changed from 'KYC', as it involves field validation
    scope: ['ALL_ACCOUNTS'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ON_TRADE',
    evaluationOrder: 2,
    requiredAuthority: 'COMPLIANCE',
    parameters: {
      requiredFields: ['fullName','dateOfBirth','riskTolerance','investmentObjective','netWorth','accreditedInvestorStatus'],
      pepCheckRequired: true,
      sanctionsCheckRequired: true,
      revalidationFrequencyDays: 365
    }
  },
  {
    id: 'account-type-restriction-v1',
    name: 'Account Type Restriction',
    description: 'Validate asset types allowed in account structure',
    rule_type: 'business_logic', // Changed from 'ASSET_RESTRICTION'
    scope: ['IRA_ACCOUNT','TRUST_ACCOUNT'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ON_TRADE',
    evaluationOrder: 3,
    parameters: {
      IRA_ACCOUNT: { prohibitedAssets: ['ALTERNATIVE','CRYPTOCURRENCY','PRIVATE_EQUITY'], maxDerivativePercentage: 0.1 },
      TRUST_ACCOUNT: { prohibitedTransactions: ['SHORT_SELLING','OPTIONS_WRITING'] }
    }
  },
  {
    id: 'liquidity-constraint-v1',
    name: 'Liquidity Constraint',
    description: 'Illiquid assets cannot exceed stated percentage',
    rule_type: 'business_logic', // Changed from 'LIQUIDITY'
    scope: ['IRA_ACCOUNT','ALL_ACCOUNTS'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'DAILY',
    evaluationOrder: 4,
    parameters: { maxIlliquidPercentage: 0.2, illiquidAssetTypes: ['PRIVATE_EQUITY','HEDGE_FUND','REAL_ESTATE'], flagThreshold: 0.18 }
  },
  {
    id: 'position-existence-v1',
    name: 'Position Existence',
    description: 'Security must exist in master and be actively traded',
    rule_type: 'business_logic', // Changed from 'DATA_INTEGRITY'
    scope: ['ALL_ACCOUNTS'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ON_TRADE',
    evaluationOrder: 5,
    parameters: { requireSecurityMasterLookup: true, requireActiveTradingStatus: true, maxCostBasisDeviation: 0.5 }
  },
  {
    id: 'trade-execution-v1',
    name: 'Trade Execution',
    description: 'Sufficient cash/securities available for trade',
    rule_type: 'business_logic', // Changed from 'TRADE'
    scope: ['ALL_ACCOUNTS'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ON_TRADE',
    evaluationOrder: 6,
    parameters: { cashBuffer: 0.01, requireT2Settlement: true }
  },
  {
    id: 'fee-validation-v1',
    name: 'Fee Validation',
    description: 'Fees must comply with regulatory limits and agreements',
    rule_type: 'business_logic', // Changed from 'FEE'
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ON_TRADE',
    evaluationOrder: 7,
    requiredAuthority: 'SUPERVISOR',
    parameters: { maxAdvisoryFeePercentage: 0.02, maxPerformanceFeePercentage: 0.25, reasonableFeeThreshold: 0.015 }
  },
  {
    id: 'advisor-permission-v1',
    name: 'Advisor Permission',
    description: 'Advisor can only access assigned accounts',
    rule_type: 'business_logic', // Changed from 'ACCESS_CONTROL'
    scope: ['ALL_ACCOUNTS'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'CONTINUOUS',
    evaluationOrder: 8,
    parameters: { requireAdvisorAssignment: true, requireFiduciaryCertification: true }
  },
  {
    id: 'temporal-consistency-v1',
    name: 'Temporal Consistency',
    description: 'Position cannot exist before account opened',
    rule_type: 'business_logic', // Changed from 'DATA_INTEGRITY'
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'DAILY',
    evaluationOrder: 9,
    parameters: { validatePositionDates: true, validateTransferDates: true }
  },
  {
    id: 'beneficiary-validation-v1',
    name: 'Beneficiary Validation',
    description: 'Ensures beneficiary information is valid and complete',
    rule_type: 'business_logic', // Changed from 'ACCOUNT_STRUCTURE'
    scope: ['IRA_ACCOUNT', 'TRUST_ACCOUNT'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ON_CHANGE',
    evaluationOrder: 10,
    requiredAuthority: 'COMPLIANCE',
    parameters: {
      requireAgeVerification: true,
      disallowMinorsForCertainAccounts: ['IRA_ACCOUNT'],
      erisaComplianceCheck: true
    }
  },
  {
    id: 'investment-profile-alignment-v1',
    name: 'Investment Profile Alignment',
    description: 'Risk tolerance must align with stated investment objectives',
    rule_type: 'business_logic', // Changed from 'KYC'
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ON_CHANGE',
    evaluationOrder: 11,
    parameters: {
      alignmentRules: [
        { riskTolerance: 'CONSERVATIVE', allowedObjectives: ['CAPITAL_PRESERVATION', 'INCOME'] },
        { riskTolerance: 'MODERATE', allowedObjectives: ['INCOME', 'GROWTH', 'TOTAL_RETURN'] },
        { riskTolerance: 'AGGRESSIVE', allowedObjectives: ['GROWTH', 'AGGRESSIVE_GROWTH', 'TOTAL_RETURN'] },
        { riskTolerance: 'VERY_AGGRESSIVE', allowedObjectives: ['AGGRESSIVE_GROWTH'] }
      ]
    }
  },
  {
    id: 'net-worth-verification-v1',
    name: 'Net Worth Verification',
    description: 'Stated net worth should align with total account holdings',
    rule_type: 'business_logic', // Changed from 'KYC'
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'QUARTERLY',
    evaluationOrder: 12,
    parameters: {
      mismatchThresholdPercentage: 0.5,
      minimumNetWorth: 100000
    }
  },
  {
    id: 'accredited-investor-revalidation-v1',
    name: 'Accredited Investor Re-validation',
    description: 'Accredited investor status must be re-validated annually',
    rule_type: 'business_logic', // Changed from 'KYC'
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ANNUALLY',
    evaluationOrder: 13,
    parameters: {
      revalidationPeriodDays: 365
    }
  },
  {
    id: 'currency-exposure-v1',
    name: 'Currency Exposure Limit',
    description: 'Foreign currency exposure must align with stated risk tolerance',
    rule_type: 'business_logic', // Changed from 'PORTFOLIO'
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'DAILY',
    evaluationOrder: 14,
    parameters: {
      exposureLimits: [
        { riskTolerance: 'CONSERVATIVE', maxForeignExposurePercentage: 0.05 },
        { riskTolerance: 'MODERATE', maxForeignExposurePercentage: 0.15 },
        { riskTolerance: 'AGGRESSIVE', maxForeignExposurePercentage: 0.30 }
      ]
    }
  },
  {
    id: 'fair-value-validation-v1',
    name: 'Fair Value Validation',
    description: 'Price deviation from benchmark is within acceptable range',
    rule_type: 'business_logic', // Changed from 'PRICING'
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'DAILY',
    evaluationOrder: 15,
    parameters: {
      maxDeviationPercentage: 0.1,
      benchmarkSource: 'default'
    }
  },
  {
    id: 'cost-basis-validation-v1',
    name: 'Cost Basis Validation',
    description: 'Cost basis cannot be negative or in the future, and must be reasonable',
    rule_type: 'business_logic', // Changed from 'DATA_INTEGRITY'
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ON_CHANGE',
    evaluationOrder: 16,
    parameters: {
      allowZeroCostBasis: false,
      maxReasonableCostBasisFactor: 1.5
    }
  },
  {
    id: 'corporate-action-validation-v1',
    name: 'Corporate Action Validation',
    description: 'Validate position adjustments after corporate actions',
    rule_type: 'business_logic', // Changed from 'DATA_INTEGRITY'
    scope: ['ALL_ACCOUNTS'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ON_CHANGE',
    evaluationOrder: 17,
    parameters: {
      requireMathematicalValidation: true
    }
  },
  {
    id: 'revenue-sharing-validation-v1',
    name: 'Revenue Sharing Validation',
    description: 'Revenue sharing agreements must comply with regulatory limits',
    rule_type: 'business_logic', // Changed from 'FEE'
    scope: ['ALL_ACCOUNTS'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ON_CHANGE',
    evaluationOrder: 18,
    parameters: {
      maxRevenueSharePercentage: 0.4
    }
  },
  {
    id: 'rebalancing-rules-v1',
    name: 'Rebalancing Rules',
    description: 'Proposed allocation must hit stated targets within tolerance',
    rule_type: 'business_logic', // Changed from 'TRADE'
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ON_REBALANCE',
    evaluationOrder: 19,
    parameters: {
      maxDeviationPercentage: 0.05
    }
  },
  {
    id: 'tax-lot-selection-v1',
    name: 'Tax Lot Selection',
    description: 'Validate tax lot selection method for tax-loss harvesting',
    rule_type: 'business_logic', // Changed from 'TRADE'
    scope: ['ALL_ACCOUNTS'],
    severity: 'INFO',
    isActive: true,
    isCore: true, // Core wealth management rule
    effectiveFrom: '2024-01-01T00:00:00.000Z',
    frequency: 'ON_TRADE',
    evaluationOrder: 20,
    parameters: {
      allowedMethods: ['FIFO', 'LIFO', 'SPECIFIC_ID']
    }
  },
  // ===== ADVANCED WEALTH MANAGEMENT RULES (Evaluation Order 21-25) =====
  {
    id: 'tax-optimization-v1',
    name: 'Tax Optimization',
    description: 'Ensure trades minimize taxable gains and comply with wash-sale rules',
    rule_type: 'business_logic',
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: true,
    effectiveFrom: '2025-01-01T00:00:00.000Z',
    frequency: 'ON_TRADE',
    evaluationOrder: 21,
    parameters: {
      maxTaxableGainPercentage: 0.15,
      washSaleWindowDays: 30,
      taxBracketThresholds: [
        { bracket: 'LOW', maxGain: 50000 },
        { bracket: 'MEDIUM', maxGain: 35000 },
        { bracket: 'HIGH', maxGain: 20000 }
      ]
    }
  },
  {
    id: 'esg-compliance-v1',
    name: 'ESG Compliance',
    description: 'Ensure portfolio aligns with client ESG preferences and regulatory requirements',
    rule_type: 'business_logic',
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: true,
    effectiveFrom: '2025-01-01T00:00:00.000Z',
    frequency: 'DAILY',
    evaluationOrder: 22,
    requiredAuthority: 'COMPLIANCE',
    parameters: {
      minEsgScore: 7.0,
      maxEsgScoreDeviation: 2.0,
      restrictedSectors: ['OIL_GAS', 'TOBACCO', 'WEAPONS'],
      esgDataSource: 'msci_api',
      integrationEndpoint: 'https://api.msci.com/esg-ratings'
    }
  },
  {
    id: 'margin-compliance-v1',
    name: 'Regulatory Margin Compliance',
    description: 'Ensure margin loans comply with FINRA Rule 4210 and maintain maintenance margin',
    rule_type: 'business_logic',
    scope: ['MARGIN_ACCOUNT'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true,
    effectiveFrom: '2025-01-01T00:00:00.000Z',
    frequency: 'ON_TRADE',
    evaluationOrder: 23,
    requiredAuthority: 'COMPLIANCE',
    parameters: {
      initialMarginLimit: 0.5,
      maintenanceMarginLimit: 0.25,
      maxLoanValue: 1000000,
      marginCallThreshold: 0.3,
      regulatoryFramework: 'FINRA_4210'
    }
  },
  {
    id: 'portfolio-drift-v1',
    name: 'Portfolio Drift Detection',
    description: 'Detect and flag deviations from target asset allocations due to market movements',
    rule_type: 'business_logic',
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: true,
    effectiveFrom: '2025-01-01T00:00:00.000Z',
    frequency: 'DAILY',
    evaluationOrder: 24,
    parameters: {
      maxDriftPercentage: 0.05,
      rebalancingThreshold: 0.08,
      targetAllocations: {
        EQUITY: 0.6,
        FIXED_INCOME: 0.35,
        CASH: 0.05
      }
    }
  },
  {
    id: 'communication-compliance-v1',
    name: 'Communication Compliance',
    description: 'Ensure advisor communications comply with SEC advertising rules (Rule 206(4)-1)',
    rule_type: 'field_format',
    scope: ['ALL_ACCOUNTS'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true,
    effectiveFrom: '2025-01-01T00:00:00.000Z',
    frequency: 'ON_CHANGE',
    evaluationOrder: 25,
    requiredAuthority: 'COMPLIANCE',
    parameters: {
      prohibitedPhrases: ['guaranteed return', 'risk-free', 'assured profit', 'no risk'],
      requiredDisclosures: ['past performance disclaimer', 'fee disclosure', 'conflict of interest'],
      regulatoryFramework: 'SEC_206_4_1'
    }
  },
  // ===== COMPETITIVE MANAGEMENT RULES (Evaluation Order 26-30) =====
  {
    id: 'ai-risk-assessment-v1',
    name: 'AI-Driven Risk Assessment',
    description: 'Assess portfolio risk using machine learning models for Value-at-Risk and stress testing',
    rule_type: 'business_logic',
    scope: ['ALL_ACCOUNTS'],
    severity: 'WARNING',
    isActive: true,
    isCore: false, // Advanced feature, not core compliance
    effectiveFrom: '2025-01-01T00:00:00.000Z',
    frequency: 'DAILY',
    evaluationOrder: 26,
    parameters: {
      maxVaR: 0.05,
      varConfidenceLevel: 0.95,
      stressTestScenarios: ['market_crash_10', 'interest_rate_spike', 'currency_volatility'],
      aiModelEndpoint: 'https://api.sagemaker.example.com/risk-model',
      modelType: 'tensorflow_var',
      integrationTimeout: 30000
    }
  },
  {
    id: 'client-engagement-v1',
    name: 'Client Engagement Tracking',
    description: 'Ensure timely client interactions based on portfolio events or milestones',
    rule_type: 'business_logic',
    scope: ['ALL_ACCOUNTS'],
    severity: 'INFO',
    isActive: true,
    isCore: false,
    effectiveFrom: '2025-01-01T00:00:00.000Z',
    frequency: 'DAILY',
    evaluationOrder: 27,
    parameters: {
      minInteractionFrequencyDays: 90,
      triggerEvents: [
        'portfolio_drop_10_percent',
        'portfolio_gain_15_percent',
        'rebalancing_required',
        'significant_market_event'
      ],
      notificationChannel: 'email',
      escalationThreshold: 180
    }
  },
  {
    id: 'performance-benchmarking-v1',
    name: 'Performance Benchmarking',
    description: 'Compare portfolio performance against industry benchmarks and track alpha generation',
    rule_type: 'business_logic',
    scope: ['ALL_ACCOUNTS'],
    severity: 'INFO',
    isActive: true,
    isCore: false,
    effectiveFrom: '2025-01-01T00:00:00.000Z',
    frequency: 'MONTHLY',
    evaluationOrder: 28,
    parameters: {
      benchmarkIndex: 'SP500',
      secondaryBenchmarks: ['NASDAQ', 'RUSSELL2000', 'MSCI_WORLD'],
      minPerformanceDelta: -0.02,
      evaluationPeriodMonths: 12,
      dataSource: 'bloomberg_api',
      integrationEndpoint: 'https://api.bloomberg.com/benchmark-data'
    }
  },
  {
    id: 'aml-compliance-v1',
    name: 'Anti-Money Laundering (AML) Compliance',
    description: 'Detect suspicious transactions to comply with Bank Secrecy Act and AML regulations',
    rule_type: 'business_logic',
    scope: ['ALL_ACCOUNTS'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true,
    effectiveFrom: '2025-01-01T00:00:00.000Z',
    frequency: 'ON_TRADE',
    evaluationOrder: 29,
    requiredAuthority: 'COMPLIANCE',
    parameters: {
      transactionThreshold: 10000,
      cumulativeThreshold: 50000,
      cumulativeWindowDays: 30,
      suspiciousPatterns: ['rapid_transfers', 'high_frequency_small_amounts', 'round_number_trades'],
      amlScreeningService: 'world_check_api',
      integrationEndpoint: 'https://api.world-check.com/screen',
      reportingRequirement: 'SAR'
    }
  },
  {
    id: 'alternative-investments-v1',
    name: 'Alternative Investments Eligibility',
    description: 'Validate eligibility for alternative investments based on accreditation and portfolio size',
    rule_type: 'business_logic',
    scope: ['ALL_ACCOUNTS'],
    severity: 'BLOCK',
    isActive: true,
    isCore: true,
    effectiveFrom: '2025-01-01T00:00:00.000Z',
    frequency: 'ON_TRADE',
    evaluationOrder: 30,
    requiredAuthority: 'SUPERVISOR',
    parameters: {
      minNetWorth: 2000000,
      minAnnualIncome: 200000,
      maxAlternativeAllocation: 0.2,
      alternativeAssetTypes: ['PRIVATE_EQUITY', 'HEDGE_FUND', 'REAL_ESTATE', 'COMMODITIES'],
      requiredAccreditation: true,
      accreditationRevalidationDays: 365
    }
  }
];

export default WEALTH_VALIDATION_RULES;
