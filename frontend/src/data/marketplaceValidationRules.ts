// marketplaceValidationRules.ts
export const MARKETPLACE_VALIDATION_RULES = [
  // Retail Mutual Funds
  {
    "id": "mutual-fund-fee-disclosure-v1",
    "name": "Mutual Fund Fee Disclosure",
    "description": "Ensures mutual fund fees are disclosed in compliance with SEC Rule 498A and 2025 updates on expense ratios",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 40,
    "parameters": {
      "requiredDisclosures": ["expense_ratio", "load_fees", "12b1_fees"],
      "maxExpenseRatio": 0.015
    }
  },
  {
    "id": "mutual-fund-liquidity-v1",
    "name": "Mutual Fund Liquidity Risk",
    "description": "Validates fund liquidity under SEC Rule 22e-4, ensuring no more than 15% illiquid assets",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 41,
    "parameters": {
      "maxIlliquidAssetsPercentage": 0.15,
      "remedialActionRequired": true
    }
  },
  {
    "id": "mutual-fund-investor-protection-v1",
    "name": "Mutual Fund Investor Protection",
    "description": "Verifies protections under federal securities laws, including suitability for retail investors",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 42,
    "parameters": {
      "minInvestorQualification": "retail_investor",
      "riskAlignmentRequired": true
    }
  },
  // Wealth Management (Expanded)
  {
    "id": "wealth-onboarding-compliance-v1",
    "name": "Wealth Onboarding Compliance",
    "description": "Ensures client onboarding meets 2025 FINRA requirements for suitability and KYC",
    "rule_type": "field_format",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_CHANGE",
    "evaluationOrder": 43,
    "parameters": {
      "requiredFields": ["identity_verification", "risk_profile", "investment_objectives"]
    }
  },
  {
    "id": "wealth-performance-reporting-v1",
    "name": "Wealth Performance Reporting",
    "description": "Validates accuracy of performance reports per SEC guidelines",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 44,
    "parameters": {
      "benchmarkComparisonRequired": true,
      "errorTolerancePercentage": 0.01
    }
  },
  // Alternatives
  {
    "id": "alternatives-hedge-due-diligence-v1",
    "name": "Hedge Fund Due Diligence",
    "description": "Validates hedge fund investments under 2025 SEC guidelines for due diligence",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 45,
    "parameters": {
      "requiredChecks": ["manager_track_record", "strategy_alignment", "fee_structure"]
    }
  },
  {
    "id": "alternatives-private-equity-lockup-v1",
    "name": "Private Equity Lockup Validation",
    "description": "Ensures compliance with lockup periods for private equity investments",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 46,
    "parameters": {
      "minLockupPeriodYears": 5,
      "earlyExitPenalty": 0.1
    }
  },
  {
    "id": "alternatives-vc-risk-v1",
    "name": "Venture Capital Risk Assessment",
    "description": "Assesses risk in venture capital investments per 2025 regulatory standards",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 47,
    "parameters": {
      "maxExposurePercentage": 0.1,
      "startupStageThreshold": "series_a"
    }
  },
  // Insurance General Account
  {
    "id": "insurance-solvency-ratio-v1",
    "name": "Insurance Solvency Ratio",
    "description": "Validates solvency ratios under 2025 NAIC guidelines",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 48,
    "parameters": {
      "minSolvencyRatio": 1.5,
      "reportingRequired": true
    }
  },
  {
    "id": "insurance-annuity-payout-v1",
    "name": "Annuity Payout Compliance",
    "description": "Ensures annuity payouts comply with state insurance regulations",
    "rule_type": "business_logic",
    "scope": ["IRA_ACCOUNT"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_CHANGE",
    "evaluationOrder": 49,
    "parameters": {
      "maxPayoutPercentage": 0.08,
      "minGuaranteedPeriodYears": 10
    }
  },
  {
    "id": "insurance-beneficiary-v1",
    "name": "Life Insurance Beneficiary Validation",
    "description": "Validates beneficiary designations for life insurance funds",
    "rule_type": "field_format",
    "scope": ["TRUST_ACCOUNT"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_CHANGE",
    "evaluationOrder": 50,
    "parameters": {
      "requiredBeneficiaryFields": ["name", "relationship", "allocation_percentage"]
    }
  },
  // ETFs
  {
    "id": "etf-liquidity-v1",
    "name": "ETF Liquidity Validation",
    "description": "Ensures ETF liquidity complies with 2025 SEC Rule 22e-4",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 51,
    "parameters": {
      "minAverageDailyVolume": 100000,
      "maxIlliquidAssetsPercentage": 0.15
    }
  },
  {
    "id": "etf-expense-ratio-v1",
    "name": "ETF Expense Ratio Check",
    "description": "Validates ETF expense ratios against regulatory caps",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "INFO",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 52,
    "parameters": {
      "maxExpenseRatio": 0.01,
      "disclosureRequired": true
    }
  },
  {
    "id": "etf-diversification-v1",
    "name": "ETF Diversification",
    "description": "Ensures ETF holdings meet diversification standards per SEC Rule 12d1-4",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 53,
    "parameters": {
      "maxSingleHoldingPercentage": 0.25,
      "minHoldingsCount": 20
    }
  },
  // Bonds/Fixed Income
  {
    "id": "bond-credit-rating-v1",
    "name": "Bond Credit Rating Validation",
    "description": "Validates bond credit ratings against client risk tolerance per 2025 fixed income outlook",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 54,
    "parameters": {
      "minCreditRating": "BBB",
      "ratingAgencies": ["MOODYS", "SP", "FITCH"]
    }
  },
  {
    "id": "bond-yield-curve-v1",
    "name": "Bond Yield Curve Check",
    "description": "Validates bond investments in a steepening yield curve environment",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 55,
    "parameters": {
      "maxDurationExposure": 10,
      "yieldThreshold": 0.04
    }
  },
  {
    "id": "fixed-income-diversification-v1",
    "name": "Fixed Income Diversification",
    "description": "Ensures diversification in fixed income investments",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 56,
    "parameters": {
      "maxIssuerExposurePercentage": 0.2,
      "minRatingDiversity": 3
    }
  },
  // Stocks/Equities
  {
    "id": "equity-volatility-v1",
    "name": "Equity Volatility Limit",
    "description": "Limits exposure to high-volatility equities per 2025 market validations",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 57,
    "parameters": {
      "maxBeta": 1.5,
      "volatilityThreshold": 0.2
    }
  },
  {
    "id": "equity-diversification-v1",
    "name": "Equity Diversification",
    "description": "Ensures diversification in equity holdings",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 58,
    "parameters": {
      "maxSectorExposurePercentage": 0.3,
      "minHoldingsCount": 15
    }
  },
  {
    "id": "equity-insider-trading-v1",
    "name": "Equity Insider Trading Compliance",
    "description": "Validates trades for insider trading compliance",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 59,
    "parameters": {
      "blackoutPeriodDays": 30,
      "reportingRequired": true
    }
  },
  // Real Estate
  {
    "id": "real-estate-due-diligence-v1",
    "name": "Real Estate Due Diligence",
    "description": "Validates due diligence for real estate investments per 2025 FinCEN rules",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 60,
    "parameters": {
      "requiredChecks": ["property_valuation", "legal_review", "environmental_assessment"]
    }
  },
  {
    "id": "real-estate-reit-liquidity-v1",
    "name": "REIT Liquidity Validation",
    "description": "Ensures REIT investments meet liquidity requirements",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 61,
    "parameters": {
      "minLiquidityRatio": 0.2,
      "maxIlliquidREPercentage": 0.3
    }
  },
  {
    "id": "real-estate-property-tax-v1",
    "name": "Real Estate Property Tax Compliance",
    "description": "Validates property tax compliance for real estate investments",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "INFO",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ANNUALLY",
    "evaluationOrder": 62,
    "parameters": {
      "taxReportingRequired": true,
      "deductionThreshold": 10000
    }
  },
  // Cryptocurrencies
  {
    "id": "crypto-custody-v1",
    "name": "Crypto Custody Validation",
    "description": "Ensures crypto assets meet custody requirements per 2025 SEC guidelines",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 63,
    "parameters": {
      "approvedCustodians": ["fidelity_digital", "coinbase_custody"],
      "coldStorageRequired": true
    }
  },
  {
    "id": "crypto-volatility-v1",
    "name": "Crypto Volatility Check",
    "description": "Limits exposure to high-volatility cryptocurrencies",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 64,
    "parameters": {
      "maxVolatilityPercentage": 0.5,
      "maxPortfolioAllocation": 0.05
    }
  },
  {
    "id": "crypto-aml-kyc-v1",
    "name": "Crypto AML/KYC Compliance",
    "description": "Validates AML/KYC for crypto transactions per 2025 FinCEN rules",
    "rule_type": "field_format",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 65,
    "parameters": {
      "requiredChecks": ["wallet_address_validation", "transaction_monitoring"],
      "suspiciousPatternThreshold": 10000
    }
  }, // Added missing comma
  // CLOs (Collateralized Loan Obligations)
  {
    "id": "clo-credit-tranching-validation-v1",
    "name": "CLO Credit Tranching Validation",
    "description": "Validates CLO credit tranching structures for compliance with NAIC SSG 2025 modeling methodology, ensuring appropriate subordination levels",
    "rule_type": "business_logic",
    "scope": ["INSURANCE_ACCOUNT", "PRIVATE_MARKETS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 113,
    "parameters": {
      "minSubordinationPercentage": 0.3,
      "trancheRatingAlignment": ["AA", "BBB"],
      "recoveryRateAssumption": 0.7
    }
  },
  {
    "id": "clo-diversification-requirements-v1",
    "name": "CLO Diversification Requirements",
    "description": "Enforces diversification limits on underlying leveraged loans per NAIC Academy CLO Model 2025 guidelines to mitigate concentration risk",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 114,
    "parameters": {
      "maxIndustryExposurePercentage": 0.15,
      "maxObligorConcentration": 0.02,
      "minLoanCount": 200
    }
  },
  {
    "id": "clo-liquidity-stress-testing-v1",
    "name": "CLO Liquidity Stress Testing",
    "description": "Mandates liquidity stress tests for CLO portfolios under FINRA 2025 oversight report on structured product resilience",
    "rule_type": "business_logic",
    "scope": ["WEALTH_ACCOUNT", "PRIVATE_MARKETS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 115,
    "parameters": {
      "stressScenarioThreshold": ["market_downturn_20pct", "redemption_spike_50pct"],
      "liquidityCoverageRatioMin": 1.0,
      "ramp_up_period_months": 3
    }
  },
  {
    "id": "clo-electronic-disclosure-v1",
    "name": "CLO Electronic Disclosure Compliance",
    "description": "Ensures electronic disclosures for CLO private placements per NAIC SAPWG Proposal 2025-19, including valuation and RBC impacts",
    "rule_type": "field_format",
    "scope": ["INSURANCE_ACCOUNT"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_CHANGE",
    "evaluationOrder": 116,
    "parameters": {
      "requiredDisclosureFields": ["clo_tranche_details", "underlying_loan_schedule", "rbc_treatment"],
      "filingFormat": "XBRL",
      "submissionDeadlineDays": 30
    }
  },
  {
    "id": "clo-valuation-model-adjustment-v1",
    "name": "CLO Valuation Model Adjustment",
    "description": "Applies adjustments to CLO valuations based on NAIC Academy 2025 model refinements, focusing on non-finalized parameter calibrations",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "INFO",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ANNUALLY",
    "evaluationOrder": 117,
    "parameters": {
      "defaultProbabilityAdjustment": 0.05,
      "correlationFactorMax": 0.25,
      "modelVersionCheck": "academy_2025_v1.2"
    }
  },
  // ABS (Asset-Backed Securities)
  {
    "id": "abs-credit-enhancement-validation-v1",
    "name": "ABS Credit Enhancement Validation",
    "description": "Validates credit enhancements (e.g., overcollateralization, subordination) in ABS structures per NAIC PBBP 2025 guidance for risk-based capital treatments",
    "rule_type": "business_logic",
    "scope": ["INSURANCE_ACCOUNT", "STRUCTURED_PRODUCTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 118,
    "parameters": {
      "minOvercollateralizationPercentage": 0.08,
      "subordinationLevelMin": 0.2,
      "enhancementTypes": ["excess_spread", "reserve_account"]
    }
  },
  {
    "id": "abs-underlying-asset-diversification-v1",
    "name": "ABS Underlying Asset Diversification",
    "description": "Enforces diversification in ABS collateral pools (e.g., auto loans, receivables) under SEC Regulation AB 2025 updates to mitigate concentration risks",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 119,
    "parameters": {
      "maxSingleAssetClassExposure": 0.4,
      "minObligorCount": 1000,
      "geographicDiversificationMin": 0.5
    }
  },
  {
    "id": "abs-naic-designation-compliance-v1",
    "name": "ABS NAIC Designation Compliance",
    "description": "Ensures ABS receive appropriate NAIC designations (1-6) for Schedule D filing per August 6, 2025, Blanks Editorial Changes and PBBP requirements",
    "rule_type": "field_format",
    "scope": ["INSURANCE_ACCOUNT"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_CHANGE",
    "evaluationOrder": 120,
    "parameters": {
      "validDesignations": [1, 2, 3, 4, 5, 6],
      "filingSection": "Schedule_D_Part_1B",
      "rbcFactorAlignment": true
    }
  },
  {
    "id": "abs-disclosure-reporting-v1",
    "name": "ABS Disclosure and Reporting",
    "description": "Mandates enhanced disclosures for ABS private placements under SEC 2025 comment solicitation and NAIC SAPWG Proposal 2024-005.01 exclusions",
    "rule_type": "business_logic",
    "scope": ["PRIVATE_MARKETS", "STRUCTURED_PRODUCTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 121,
    "parameters": {
      "requiredFields": ["pool_composition", "delinquency_rates", "servicer_reports"],
      "format": "XBRL",
      "submissionDeadlineDays": 45
    }
  },
  {
    "id": "abs-liquidity-stress-valuation-v1",
    "name": "ABS Liquidity Stress and Valuation",
    "description": "Applies liquidity stress testing and valuation adjustments for ABS per FINRA 2025 oversight on structured products and NAIC Valuation Task Force amendments",
    "rule_type": "business_logic",
    "scope": ["WEALTH_ACCOUNT", "INSURANCE_ACCOUNT"],
    "severity": "INFO",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 122,
    "parameters": {
      "stressScenarios": ["interest_rate_shock_200bps", "default_rate_spike_15pct"],
      "liquidityBufferMin": 0.1,
      "valuationAdjustmentMax": 0.05
    }
  },
  // MBS (Mortgage-Backed Securities)
  {
    "id": "mbs-asset-level-disclosure-privacy-v1",
    "name": "MBS Asset-Level Disclosure Privacy",
    "description": "Validates privacy protections in MBS asset-level disclosures per SEC 2025 Concept Release on Regulation AB Item 1125, limiting sensitive data like 5-digit ZIP codes or credit scores to prevent borrower deanonymization",
    "rule_type": "field_format",
    "scope": ["STRUCTURED_PRODUCTS", "INSURANCE_ACCOUNT"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 123,
    "parameters": {
      "restrictedDataPoints": ["exact_zip_code", "credit_score_raw", "borrower_name"],
      "roundingRequirements": { "upb": "nearest_thousand", "loan_amount": "nearest_thousand" },
      "hostingPlatform": "issuer_sponsored_website"
    }
  },
  {
    "id": "mbs-pool-diversification-v1",
    "name": "MBS Pool Diversification",
    "description": "Enforces geographic and obligor diversification in MBS loan pools under NAIC PBBP 2025 guidance and SEC Regulation AB updates to mitigate concentration risks",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 124,
    "parameters": {
      "maxGeographicExposure": 0.2,
      "minObligorCount": 500,
      "maxSingleOriginatorPercentage": 0.15
    }
  },
  {
    "id": "mbs-naic-designation-filing-v1",
    "name": "MBS NAIC Designation Filing",
    "description": "Ensures MBS receive NAIC designations (1-6) for Schedule D compliance per NAIC Valuation of Securities Task Force 2025 amendments and SSG surveillance requirements",
    "rule_type": "business_logic",
    "scope": ["INSURANCE_ACCOUNT"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_CHANGE",
    "evaluationOrder": 125,
    "parameters": {
      "validDesignations": [1, 2, 3, 4, 5, 6],
      "filingSection": "Schedule_D_Part_1A",
      "rbcFactorVerification": true,
      "submissionDeadlineDays": 30
    }
  },
  {
    "id": "mbs-macroeconomic-stress-v1",
    "name": "MBS Macroeconomic Stress Testing",
    "description": "Mandates through-the-cycle stress testing for MBS per NAIC SSG November 2025 scenarios for RMBS/CMBS surveillance, including interest rate shocks and default spikes",
    "rule_type": "business_logic",
    "scope": ["WEALTH_ACCOUNT", "PRIVATE_MARKETS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ANNUALLY",
    "evaluationOrder": 126,
    "parameters": {
      "stressScenarios": ["rate_shock_200bps", "default_increase_10pct", "unemployment_rise_5pct"],
      "probabilityWeightings": { "base": 0.4, "adverse": 0.4, "severe": 0.2 },
      "liquidityCoverageMin": 0.85
    }
  },
  {
    "id": "mbs-reg-bi-suitability-v1",
    "name": "MBS Reg BI Suitability",
    "description": "Validates MBS recommendations for retail investors under FINRA 2025 Oversight Report priorities, ensuring care obligations for complex structured products",
    "rule_type": "business_logic",
    "scope": ["RETAIL_ACCOUNT"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 127,
    "parameters": {
      "suitabilityFactors": ["risk_tolerance", "liquidity_needs", "prepayment_risk_awareness"],
      "disclosureRequired": ["tranching_risks", "prepayment_speeds"],
      "conflictMitigation": true
    }
  },
  // CMBS (Commercial Mortgage-Backed Securities)
  {
    "id": "cmbs-property-diversification-v1",
    "name": "CMBS Property Diversification",
    "description": "Mandates diversification across property types and geographies in CMBS pools per NAIC SSG 2025 surveillance guidelines to address CRE concentration risks",
    "rule_type": "business_logic",
    "scope": ["INSURANCE_ACCOUNT", "STRUCTURED_PRODUCTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 128,
    "parameters": {
      "maxPropertyTypeExposure": 0.25,
      "minGeographicRegions": 5,
      "creSectorLimits": { "office": 0.2, "retail": 0.15, "multifamily": 0.3 }
    }
  },
  {
    "id": "cmbs-ltv-dscr-validation-v1",
    "name": "CMBS LTV and DSCR Validation",
    "description": "Validates loan-to-value (LTV) and debt service coverage ratios (DSCR) for underlying commercial loans under SEC Regulation AB 2025 asset-level disclosure requirements",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 129,
    "parameters": {
      "maxLTVPercentage": 0.65,
      "minDSCRRatio": 1.25,
      "amortizationScheduleCheck": true
    }
  },
  {
    "id": "cmbs-naic-rbc-filing-v1",
    "name": "CMBS NAIC RBC Filing Compliance",
    "description": "Ensures CMBS receive NAIC designations (1-6) and RBC adjustments per the June 2025 NAIC investment framework overhaul and Schedule D requirements",
    "rule_type": "field_format",
    "scope": ["INSURANCE_ACCOUNT"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_CHANGE",
    "evaluationOrder": 130,
    "parameters": {
      "validDesignations": [1, 2, 3, 4, 5, 6],
      "rbcAdjustmentFactors": { "cre_stress": 1.15 },
      "filingSection": "Schedule_D_Part_2"
    }
  },
  {
    "id": "cmbs-cre-stress-testing-v1",
    "name": "CMBS CRE Stress Testing",
    "description": "Requires macroeconomic stress testing for CMBS CRE exposures, aligned with NAIC SSG through-the-cycle scenarios and FINRA 2025 oversight on structured products",
    "rule_type": "business_logic",
    "scope": ["WEALTH_ACCOUNT", "PRIVATE_MARKETS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ANNUALLY",
    "evaluationOrder": 131,
    "parameters": {
      "stressScenarios": ["office_vacancy_rise_20pct", "cap_rate_increase_150bps", "rental_decline_15pct"],
      "passFailThreshold": 0.9,
      "recoveryRateAssumption": 0.6
    }
  },
  {
    "id": "cmbs-sales-practice-suitability-v1",
    "name": "CMBS Sales Practice Suitability",
    "description": "Assesses CMBS recommendations for investor suitability under FINRA 2025 Report priorities, including Reg BI care obligations for CRE-linked complexities",
    "rule_type": "business_logic",
    "scope": ["RETAIL_ACCOUNT"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 132,
    "parameters": {
      "suitabilityCriteria": ["cre_risk_awareness", "liquidity_profile", "tranching_understanding"],
      "disclosureElements": ["servicer_advances", "special_servicing_risks"],
      "thirdPartyRiskReview": true
    }
  },
  // MMFs (Money Market Funds)
  {
    "id": "mmf-liquidity-fee-compliance-v1",
    "name": "MMF Liquidity Fee Compliance",
    "description": "Validates imposition of liquidity fees under SEC Rule 2a-7 2025 updates when weekly liquid assets fall below 25%",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 133,
    "parameters": {
      "minWeeklyLiquidAssets": 0.25,
      "feeRangePercentage": [0.01, 0.02],
      "noticePeriodHours": 24
    }
  },
  {
    "id": "mmf-stress-testing-v1",
    "name": "MMF Stress Testing",
    "description": "Mandates stress testing for money market funds per SEC 2025 Rule 2a-7 amendments, including interest rate shocks and redemption surges",
    "rule_type": "business_logic",
    "scope": ["INSURANCE_ACCOUNT", "WEALTH_ACCOUNT"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 134,
    "parameters": {
      "stressScenarios": ["rate_shock_150bps", "redemption_spike_20pct"],
      "passThreshold": 0.95,
      "reportFrequency": "monthly"
    }
  },
  // ETNs (Exchange-Traded Notes)
  {
    "id": "etn-counterparty-risk-v1",
    "name": "ETN Counterparty Risk Assessment",
    "description": "Evaluates counterparty credit risk for ETNs per FINRA 2025 oversight, ensuring issuer solvency",
    "rule_type": "business_logic",
    "scope": ["RETAIL_ACCOUNT", "WEALTH_ACCOUNT"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 135,
    "parameters": {
      "minCreditRating": "A-",
      "counterpartyExposureLimit": 0.1,
      "diversificationCheck": true
    }
  },
  {
    "id": "etn-suitability-check-v1",
    "name": "ETN Suitability Check",
    "description": "Validates ETN suitability for retail investors under FINRA Reg BI 2025 updates, focusing on complexity and liquidity risks",
    "rule_type": "business_logic",
    "scope": ["RETAIL_ACCOUNT"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 136,
    "parameters": {
      "riskToleranceThreshold": 0.6,
      "liquidityHorizonMinDays": 30,
      "disclosureRequired": true
    }
  },
  // Hedge Funds
  {
    "id": "hedge-fund-leverage-limit-v1",
    "name": "Hedge Fund Leverage Limit",
    "description": "Imposes leverage limits on hedge funds per SEC Form PF 2025 revisions and ESMA AIFMD Article 15 updates",
    "rule_type": "business_logic",
    "scope": ["PRIVATE_MARKETS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 137,
    "parameters": {
      "maxGrossLeverage": 2.0,
      "netLeverageThreshold": 1.5,
      "reportingRequired": true
    }
  },
  {
    "id": "hedge-fund-liquidity-terms-v1",
    "name": "Hedge Fund Liquidity Terms",
    "description": "Validates liquidity terms (e.g., lock-up periods, gates) per ESMA 2025 AIFMD liquidity risk management rules",
    "rule_type": "business_logic",
    "scope": ["PRIVATE_MARKETS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_CHANGE",
    "evaluationOrder": 138,
    "parameters": {
      "maxLockupPeriodMonths": 12,
      "gateTriggerPercentage": 0.2,
      "noticePeriodDays": 30
    }
  },
  // Venture Debt
  {
    "id": "venture-debt-covenant-compliance-v1",
    "name": "Venture Debt Covenant Compliance",
    "description": "Ensures compliance with financial covenants in venture debt agreements per SEC 2025 private credit exam priorities",
    "rule_type": "business_logic",
    "scope": ["VENTURE_CAPITAL", "PRIVATE_MARKETS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 139,
    "parameters": {
      "covenantTypes": ["debt_service_coverage", "leverage_ratio"],
      "breachThreshold": 1.1,
      "remediationPeriodDays": 45
    }
  },
  {
    "id": "venture-debt-valuation-v1",
    "name": "Venture Debt Valuation",
    "description": "Validates fair value estimates for venture debt under FINRA 2025 oversight on private market valuations",
    "rule_type": "business_logic",
    "scope": ["PRIVATE_MARKETS"],
    "severity": "INFO",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ANNUALLY",
    "evaluationOrder": 140,
    "parameters": {
      "valuationMethod": ["discounted_cash_flow", "market_comparable"],
      "discountRateMin": 0.08,
      "independentReviewRequired": true
    }
  },
  {
    "id": "fixed-annuity-suitability-v1",
    "name": "Fixed Annuity Suitability",
    "description": "Ensures fixed annuity recommendations align with investor needs per NAIC Suitability in Annuity Transactions Model Regulation 2025 revisions",
    "rule_type": "business_logic",
    "scope": ["INSURANCE_ACCOUNT", "WEALTH_ACCOUNT"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 141,
    "parameters": {
      "suitabilityCriteria": ["income_needs", "risk_tolerance", "liquidity_profile"],
      "disclosureRequired": true
    }
  },
  {
    "id": "fixed-annuity-reserve-check-v1",
    "name": "Fixed Annuity Reserve Check",
    "description": "Validates reserve adequacy for fixed annuities under NAIC 2025 reserve modernization",
    "rule_type": "business_logic",
    "scope": ["INSURANCE_ACCOUNT"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 142,
    "parameters": {
      "minReserveRatio": 1.0,
      "stressTestRequired": true,
      "reportingFrequency": "annual"
    }
  },
  {
    "id": "portfolio-rebalancing-v1",
    "name": "Portfolio Rebalancing Validation",
    "description": "Ensures timely rebalancing of portfolios per GIPS 2025 standards and SEC Form ADV best practices",
    "rule_type": "business_logic",
    "scope": ["WEALTH_ACCOUNT"],
    "severity": "INFO",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "MONTHLY",
    "evaluationOrder": 143,
    "parameters": {
      "driftTolerancePercentage": 0.05,
      "rebalanceFrequencyMaxDays": 90,
      "clientNotification": true
    }
  },
  {
    "id": "portfolio-risk-attribution-v1",
    "name": "Portfolio Risk Attribution",
    "description": "Validates risk attribution reporting for portfolios under SEC 2025 Form ADV requirements",
    "rule_type": "business_logic",
    "scope": ["WEALTH_ACCOUNT"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "QUARTERLY",
    "evaluationOrder": 144,
    "parameters": {
      "riskMetrics": ["VaR", "tracking_error"],
      "benchmarkComparison": true,
      "thresholdBreach": 0.03
    }
  },
  {
    "id": "cybersecurity-incident-response-v1",
    "name": "Cybersecurity Incident Response",
    "description": "Mandates incident response plans for investment data breaches per SEC Regulation S-P 2025 updates",
    "rule_type": "field_format",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_CHANGE",
    "evaluationOrder": 145,
    "parameters": {
      "requiredElements": ["incident_log", "notification_process", "remediation_plan"],
      "notificationDeadlineHours": 48
    }
  },
  {
    "id": "data-privacy-compliance-v1",
    "name": "Data Privacy Compliance",
    "description": "Ensures data privacy compliance with NAIC 2025 privacy model law for investor data",
    "rule_type": "business_logic",
    "scope": ["ALL_ACCOUNTS"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ANNUALLY",
    "evaluationOrder": 146,
    "parameters": {
      "consentGranularity": ["opt_in", "purpose_specific"],
      "dataRetentionMaxYears": 7,
      "auditRequired": true
    }
  },
  {
    "id": "emerging-market-currency-risk-v1",
    "name": "Emerging Market Currency Risk",
    "description": "Manages currency risk in emerging market investments per ESMA MiFID II 2025 enhancements",
    "rule_type": "business_logic",
    "scope": ["INTERNATIONAL_ACCOUNT"],
    "severity": "WARNING",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "DAILY",
    "evaluationOrder": 147,
    "parameters": {
      "hedgingRatioMin": 0.7,
      "volatilityThreshold": 0.15,
      "reportingRequired": true
    }
  },
  {
    "id": "emerging-market-sanctions-v1",
    "name": "Emerging Market Sanctions",
    "description": "Checks sanctions compliance for emerging market investments per SEC 2025 OFAC alignment",
    "rule_type": "business_logic",
    "scope": ["INTERNATIONAL_ACCOUNT"],
    "severity": "BLOCK",
    "isActive": true,
    "effectiveFrom": "2026-01-01T00:00:00.000Z",
    "frequency": "ON_TRADE",
    "evaluationOrder": 148,
    "parameters": {
      "sanctionedEntitiesList": ["OFAC_SDN"],
      "screeningFrequency": "daily",
      "blockThreshold": 0.9
    }
  }
];
export default MARKETPLACE_VALIDATION_RULES;