# Advanced Wealth Validation Rules - Implementation Guide

## Overview

This document provides comprehensive guidance for implementing, testing, and integrating the 10 advanced wealth validation rules added to the Fabric Builder wealth management platform. These rules build upon the existing 20 core validation rules to create a world-class, Workday-inspired wealth management solution that surpasses competitors like SS&C Black Diamond.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [New Rules Summary](#new-rules-summary)
3. [Frontend Integration](#frontend-integration)
4. [Backend Integration](#backend-integration)
5. [External API Integrations](#external-api-integrations)
6. [Testing & Validation](#testing--validation)
7. [Performance Considerations](#performance-considerations)
8. [Security & Compliance](#security--compliance)

---

## Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                        Fabric Builder UI                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Rule Creator │  │ Rule Editor  │  │ Rule List (Filtered) │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└──────────────────────────────┬──────────────────────────────────┘
                               │
              ┌────────────────┴────────────────┐
              │                                 │
              ▼                                 ▼
    ┌──────────────────────┐        ┌────────────────────────┐
    │ wealthValidationRules│        │ ValidationRuleParameters│
    │ (Rule Definitions)   │        │ Registry (UI Metadata) │
    └──────────────────────┘        └────────────────────────┘
              │                                 │
              └────────────────┬────────────────┘
                               ▼
                   ┌────────────────────────┐
                   │  Backend API Layer     │
                   │  /api/validation-rules│
                   └────────────────────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
              ▼                ▼                ▼
    ┌──────────────────┐ ┌─────────────┐ ┌─────────────────┐
    │ PostgreSQL DB    │ │ Rule Engine │ │External API Svcs│
    │ (validation_rules)│ │ Executor    │ │(MSCI, WC, etc)  │
    └──────────────────┘ └─────────────┘ └─────────────────┘
```

### Data Flow for Rule Execution

```
1. Portfolio/Trade Event → Validation Trigger
2. Load Rule Definition → Get Parameters from Registry
3. Create Execution Context → Gather Portfolio/Transaction Data
4. Execute Business Logic → Validate Against Parameters
5. Call External APIs (if needed) → ESG, AML, Risk Assessment
6. Aggregate Results → Pass/Fail/Warn
7. Update Audit Trail → Log to PostgreSQL
```

---

## New Rules Summary

### Tier 1: Advanced Wealth Management Rules (Evaluation Order 21-25)

#### 1. **Tax Optimization Rule** (ID: `tax-optimization-v1`)
- **Purpose**: Minimize tax liability through intelligent trade selection
- **Scope**: All accounts
- **Severity**: WARNING
- **Key Parameters**:
  - `maxTaxableGainPercentage`: Max % of portfolio realizing gains per period (default: 15%)
  - `washSaleWindowDays`: Days to avoid security repurchase (default: 30)
  - `taxBracketThresholds`: Gain limits by tax bracket
- **Execution**:
  - Analyzes trade for tax-loss harvesting opportunities
  - Checks wash-sale rule compliance
  - Suggests alternative securities if wash-sale violation detected
- **Example Trigger**: Trade execution triggers tax optimization check

#### 2. **ESG Compliance Rule** (ID: `esg-compliance-v1`)
- **Purpose**: Align portfolio with client ESG preferences
- **Scope**: All accounts
- **Severity**: WARNING
- **Key Parameters**:
  - `minEsgScore`: Minimum ESG score (default: 7.0/10)
  - `restrictedSectors`: Excluded sectors (e.g., OIL_GAS, TOBACCO)
  - `esgDataSource`: API provider (MSCI, Sustainalytics, Bloomberg)
  - `integrationEndpoint`: API URL for ESG ratings
- **Execution**:
  - Queries MSCI/Sustainalytics API for security ESG rating
  - Compares against client preferences
  - Flags investments in restricted sectors
- **External Integration**: MSCI ESG Ratings API

#### 3. **Margin Compliance Rule** (ID: `margin-compliance-v1`)
- **Purpose**: Ensure FINRA Rule 4210 compliance for leveraged accounts
- **Scope**: Margin accounts only
- **Severity**: BLOCK
- **Key Parameters**:
  - `initialMarginLimit`: Max initial margin (default: 50%)
  - `maintenanceMarginLimit`: Min maintenance margin (default: 25%)
  - `maxLoanValue`: Maximum margin loan (default: $1M)
  - `marginCallThreshold`: Margin level triggering call (default: 30%)
- **Execution**:
  - Calculates loan-to-value ratio
  - Checks against regulatory limits
  - Predicts margin call scenarios
- **Regulatory Framework**: FINRA 4210, SEC Regulation T

#### 4. **Portfolio Drift Detection Rule** (ID: `portfolio-drift-v1`)
- **Purpose**: Proactive detection of asset allocation deviations
- **Scope**: All accounts
- **Severity**: WARNING
- **Key Parameters**:
  - `maxDriftPercentage`: Max deviation before warning (default: 5%)
  - `rebalancingThreshold`: Deviation triggering rebalancing recommendation (default: 8%)
  - `targetAllocations`: Target by asset class (e.g., 60% equity, 35% bonds, 5% cash)
- **Execution**:
  - Daily calculation of current vs. target allocation
  - Flags excessive drift
  - Recommends rebalancing trades
- **Frequency**: Daily

#### 5. **Communication Compliance Rule** (ID: `communication-compliance-v1`)
- **Purpose**: Ensure SEC Rule 206(4)-1 compliance in advisor communications
- **Scope**: All accounts
- **Severity**: BLOCK
- **Key Parameters**:
  - `prohibitedPhrases`: Banned language (e.g., "guaranteed return", "risk-free")
  - `requiredDisclosures`: Mandatory disclosures (past performance disclaimer, fees)
  - `regulatoryFramework`: SEC/FINRA/MiFID II
- **Execution**:
  - Text analysis of client communications
  - Scans for prohibited phrases
  - Enforces required disclosures
- **Integration**: Could use NLP for advanced phrase detection

### Tier 2: Competitive Management Rules (Evaluation Order 26-30)

#### 6. **AI-Driven Risk Assessment Rule** (ID: `ai-risk-assessment-v1`)
- **Purpose**: Dynamic portfolio risk assessment using ML models
- **Scope**: All accounts
- **Severity**: WARNING
- **Key Parameters**:
  - `maxVaR`: Maximum Value-at-Risk tolerance (default: 5%)
  - `varConfidenceLevel`: VaR confidence level (default: 95%)
  - `stressTestScenarios`: Market scenarios to test (market crash, rate spike, etc.)
  - `aiModelEndpoint`: AWS SageMaker or TensorFlow endpoint URL
  - `integrationTimeout`: Max response time (default: 30s)
- **Execution**:
  - Sends portfolio data to ML model endpoint
  - Receives VaR, stress test results, risk recommendations
  - Flags portfolios exceeding VaR threshold
- **External Integration**: AWS SageMaker, TensorFlow.js

#### 7. **Client Engagement Tracking Rule** (ID: `client-engagement-v1`)
- **Purpose**: Ensure timely advisor-client interactions
- **Scope**: All accounts
- **Severity**: INFO (escalates to WARNING after 180 days)
- **Key Parameters**:
  - `minInteractionFrequencyDays`: Min days between touches (default: 90)
  - `triggerEvents`: Events requiring immediate contact
  - `notificationChannel`: Email, SMS, or portal notification
  - `escalationThreshold`: Days until supervisor escalation (default: 180)
- **Execution**:
  - Monitors last client contact date
  - Triggers advisor notifications for portfolio events
  - Escalates if no contact in threshold period
- **Trigger Examples**: Portfolio drops 10%, rebalancing needed, market events

#### 8. **Performance Benchmarking Rule** (ID: `performance-benchmarking-v1`)
- **Purpose**: Compare portfolio performance against industry benchmarks
- **Scope**: All accounts
- **Severity**: INFO
- **Key Parameters**:
  - `benchmarkIndex`: Primary benchmark (S&P 500, NASDAQ, Russell 2000, etc.)
  - `secondaryBenchmarks`: Additional comparison indices
  - `minPerformanceDelta`: Min acceptable underperformance (default: -2%)
  - `evaluationPeriodMonths`: Lookback period (default: 12 months)
  - `dataSource`: Bloomberg, Refinitiv, Yahoo Finance
  - `integrationEndpoint`: Benchmark data API URL
- **Execution**:
  - Fetches benchmark performance data
  - Compares portfolio returns and volatility
  - Calculates alpha generation
  - Flags underperformance
- **Monthly Evaluation**

#### 9. **AML Compliance Rule** (ID: `aml-compliance-v1`)
- **Purpose**: Detect suspicious transactions to comply with Bank Secrecy Act
- **Scope**: All accounts
- **Severity**: BLOCK
- **Key Parameters**:
  - `transactionThreshold`: Individual tx reporting threshold (default: $10K)
  - `cumulativeThreshold`: Cumulative threshold (default: $50K in 30 days)
  - `suspiciousPatterns`: Red flags (rapid transfers, round numbers, frequency)
  - `amlScreeningService`: World-Check, OFAC, Dow Jones Watchlist
  - `integrationEndpoint`: AML screening service API
  - `reportingRequirement`: SAR or CTR filing
- **Execution**:
  - Real-time transaction monitoring
  - Pattern analysis for suspicious activity
  - World-Check API screening of parties
  - Generates Suspicious Activity Reports (SAR) as needed
- **External Integration**: World-Check API, OFAC List

#### 10. **Alternative Investments Eligibility Rule** (ID: `alternative-investments-v1`)
- **Purpose**: Validate client eligibility for alternative investments
- **Scope**: All accounts
- **Severity**: BLOCK
- **Key Parameters**:
  - `minNetWorth`: Minimum net worth (default: $2M)
  - `minAnnualIncome`: Minimum annual income (default: $200K)
  - `maxAlternativeAllocation`: Max portfolio % (default: 20%)
  - `alternativeAssetTypes`: Allowed alternatives (PE, hedge funds, RE, commodities)
  - `requiredAccreditation`: Must be accredited investor
  - `accreditationRevalidationDays`: How often to revalidate (default: 365)
- **Execution**:
  - Validates accreditation status
  - Checks net worth and income thresholds
  - Monitors allocation caps
  - Flags revalidation dates
- **On Trade Execution**

---

## Frontend Integration

### 1. ValidationRuleParametersRegistry.ts

This file maps rule names to dynamic parameter configurations for UI rendering.

**Key Features**:
- Maps 30 rules to their parameter configurations
- Supports multiple input types: text, number, checkbox, select, array, object, textarea
- Provides validation helpers
- Includes descriptions and defaults for each parameter

**Usage**:
```typescript
import { getParametersForRule, validateParameters } from '@/data/ValidationRuleParametersRegistry';

// Get parameter configs for a rule
const params = getParametersForRule('ESG Compliance');

// Validate rule parameters
const validation = validateParameters('ESG Compliance', {
  minEsgScore: 7.0,
  restrictedSectors: ['OIL_GAS', 'TOBACCO']
});

if (!validation.valid) {
  console.error('Parameter errors:', validation.errors);
}
```

### 2. ValidationRuleCreator.tsx Enhancement

**Updates Required**:

```typescript
import { getParametersForRule } from '@/data/ValidationRuleParametersRegistry';

export function ValidationRuleCreator() {
  const [selectedRule, setSelectedRule] = useState<string>('');
  const [parameters, setParameters] = useState<Record<string, any>>({});

  const renderParameterFields = () => {
    const configs = getParametersForRule(selectedRule);
    
    return configs.map(config => {
      switch (config.type) {
        case 'number':
          return (
            <div key={config.fieldName}>
              <label>{config.label}</label>
              <input
                type="number"
                min={config.min}
                max={config.max}
                step={config.step}
                value={parameters[config.fieldName] || ''}
                onChange={(e) => setParameters({
                  ...parameters,
                  [config.fieldName]: parseFloat(e.target.value)
                })}
              />
              {config.description && <p className="hint">{config.description}</p>}
            </div>
          );
        case 'checkbox':
          return (
            <div key={config.fieldName}>
              <label>
                <input
                  type="checkbox"
                  checked={parameters[config.fieldName] || false}
                  onChange={(e) => setParameters({
                    ...parameters,
                    [config.fieldName]: e.target.checked
                  })}
                />
                {config.label}
              </label>
              {config.description && <p className="hint">{config.description}</p>}
            </div>
          );
        case 'select':
          return (
            <div key={config.fieldName}>
              <label>{config.label}</label>
              <select
                value={parameters[config.fieldName] || ''}
                onChange={(e) => setParameters({
                  ...parameters,
                  [config.fieldName]: e.target.value
                })}
              >
                <option value="">Select...</option>
                {config.options?.map(opt => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>
          );
        case 'array':
          return (
            <div key={config.fieldName}>
              <label>{config.label}</label>
              <textarea
                placeholder="Enter comma-separated values"
                value={Array.isArray(parameters[config.fieldName]) ? 
                  parameters[config.fieldName].join(', ') : ''}
                onChange={(e) => setParameters({
                  ...parameters,
                  [config.fieldName]: e.target.value.split(',').map(s => s.trim())
                })}
              />
              {config.description && <p className="hint">{config.description}</p>}
            </div>
          );
        default:
          return null;
      }
    });
  };

  return (
    <form onSubmit={handleSubmit}>
      <select value={selectedRule} onChange={(e) => setSelectedRule(e.target.value)}>
        <option value="">Select a rule...</option>
        {/* Options populated from WEALTH_VALIDATION_RULES */}
      </select>
      
      {selectedRule && renderParameterFields()}
      
      <button type="submit">Create Rule</button>
    </form>
  );
}
```

### 3. ValidationRuleEditor.tsx Enhancement

**Similar enhancements for editing existing rules, with parameter loading from rule.parameters object**

### 4. ValidationRulesWithFacets.tsx Enhancement

**Add new facet categories**:
```typescript
const advancedFacets = [
  {
    id: 'advanced-wealth-mgmt',
    label: 'Advanced Wealth Mgmt',
    values: [
      { id: 'tax-optimization', label: 'Tax Optimization', count: 1 },
      { id: 'esg-compliance', label: 'ESG Compliance', count: 1 },
      { id: 'margin-compliance', label: 'Margin Compliance', count: 1 },
      // ... more
    ]
  },
  {
    id: 'competitive-features',
    label: 'Competitive Features',
    values: [
      { id: 'ai-risk', label: 'AI Risk Assessment', count: 1 },
      { id: 'client-engagement', label: 'Client Engagement', count: 1 },
      // ... more
    ]
  }
];
```

---

## Backend Integration

### 1. PostgreSQL Schema

Ensure `validation_rules` table supports new fields:

```sql
CREATE TABLE validation_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  name VARCHAR(255) NOT NULL,
  description TEXT,
  rule_type VARCHAR(50) NOT NULL, -- 'business_logic', 'field_format', etc.
  scope TEXT[] NOT NULL,
  severity VARCHAR(50) NOT NULL, -- 'BLOCK', 'WARNING', 'INFO'
  is_active BOOLEAN DEFAULT true,
  is_core BOOLEAN DEFAULT false,
  effective_from TIMESTAMP,
  frequency VARCHAR(50),
  evaluation_order INTEGER,
  required_authority VARCHAR(100),
  parameters JSONB NOT NULL, -- For flexible parameter storage
  condition_json JSONB,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by UUID,
  updated_by UUID
);

CREATE INDEX idx_validation_rules_tenant_id ON validation_rules(tenant_id);
CREATE INDEX idx_validation_rules_is_active ON validation_rules(is_active);
CREATE INDEX idx_validation_rules_evaluation_order ON validation_rules(evaluation_order);
```

### 2. Backend API Enhancements (Go)

**File**: `backend/internal/api/validation_rules_routes.go`

```go
// Extend existing handlers to support new rule types

func handleExecuteValidationRule(w http.ResponseWriter, r *http.Request) {
    var req ExecuteRuleRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    rule, err := getRule(req.RuleID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    var result interface{}
    
    switch rule.RuleType {
    case "business_logic":
        result, err = executeBusinessLogicRule(req.Context, rule)
    case "field_format":
        result, err = executeFieldFormatRule(req.Context, rule)
    default:
        http.Error(w, "Unknown rule type", http.StatusBadRequest)
        return
    }

    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}

func executeBusinessLogicRule(ctx ValidationContext, rule ValidationRule) (interface{}, error) {
    // Route to specific rule handlers
    switch rule.Name {
    case "Tax Optimization":
        return executeTaxOptimization(ctx, rule)
    case "ESG Compliance":
        return executeESGCompliance(ctx, rule)
    case "Margin Compliance":
        return executeMarginCompliance(ctx, rule)
    case "AI-Driven Risk Assessment":
        return executeAIRiskAssessment(ctx, rule)
    case "AML Compliance":
        return executeAMLCompliance(ctx, rule)
    // ... more handlers
    }
    return nil, fmt.Errorf("Unknown business logic rule: %s", rule.Name)
}

// Example handler for ESG Compliance
func executeESGCompliance(ctx ValidationContext, rule ValidationRule) (interface{}, error) {
    params := rule.Parameters
    minEsgScore := params["minEsgScore"].(float64)
    
    // For each holding in portfolio
    for _, holding := range ctx.Portfolio.Holdings {
        // Call external API to get ESG rating
        esgRating, err := externalAPI.GetESGRating(holding.SecurityID)
        if err != nil {
            return nil, fmt.Errorf("ESG API error: %w", err)
        }
        
        if esgRating.Score < minEsgScore {
            return ValidationResult{
                Passed: false,
                Severity: "WARNING",
                Message: fmt.Sprintf("Security %s has ESG score %.1f below minimum %.1f",
                    holding.SecurityID, esgRating.Score, minEsgScore),
            }, nil
        }
    }
    
    return ValidationResult{Passed: true}, nil
}

// Example handler for AI Risk Assessment
func executeAIRiskAssessment(ctx ValidationContext, rule ValidationRule) (interface{}, error) {
    params := rule.Parameters
    endpoint := params["aiModelEndpoint"].(string)
    maxVar := params["maxVaR"].(float64)
    
    // Prepare portfolio data for AI model
    portfolioData := preparePortfolioForAI(ctx.Portfolio)
    
    // Call SageMaker/TensorFlow endpoint
    riskAssessment, err := callAIRiskModel(endpoint, portfolioData)
    if err != nil {
        return nil, fmt.Errorf("AI model error: %w", err)
    }
    
    if riskAssessment.Var95 > maxVar {
        return ValidationResult{
            Passed: false,
            Severity: "WARNING",
            Message: fmt.Sprintf("VaR 95%% = %.2f%% exceeds max %.2f%%",
                riskAssessment.Var95*100, maxVar*100),
        }, nil
    }
    
    return ValidationResult{Passed: true}, nil
}
```

### 3. External API Client (Go)

**File**: `backend/internal/services/external_api_client.go`

```go
package services

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type ExternalAPIClient struct {
    httpClient *http.Client
    msciKey    string
    bloombergToken string
    // ... other credentials
}

// Get ESG rating for a security
func (c *ExternalAPIClient) GetESGRating(securityID string) (*ESGRating, error) {
    url := fmt.Sprintf("https://api.msci.com/esg-ratings?ticker=%s", securityID)
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.msciKey))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result ESGRating
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result, nil
}

// Screen entity for AML
func (c *ExternalAPIClient) ScreenAML(name string) (*AMLScreeningResult, error) {
    // Similar implementation for World-Check API
    // ...
}

// Call AI risk model
func (c *ExternalAPIClient) CallAIRiskModel(endpoint string, portfolioData interface{}) (*RiskAssessment, error) {
    body, err := json.Marshal(portfolioData)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    // Set timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    req = req.WithContext(ctx)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result RiskAssessment
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result, nil
}
```

---

## External API Integrations

### 1. MSCI ESG Ratings API

**Configuration**:
- Endpoint: `https://api.msci.com/esg-ratings`
- Authentication: Bearer token in header
- Rate Limit: Check MSCI documentation

**Environment Variables**:
```bash
VITE_MSCI_API_KEY=your_api_key_here
VITE_MSCI_ENDPOINT=https://api.msci.com/esg-ratings
```

**Request Example**:
```bash
curl -H "Authorization: Bearer YOUR_API_KEY" \
  "https://api.msci.com/esg-ratings?ticker=AAPL&format=json"
```

**Response Structure**:
```json
{
  "ticker": "AAPL",
  "esgScore": 8.2,
  "esgRating": "AAA",
  "environmentScore": 7.9,
  "socialScore": 8.5,
  "governanceScore": 8.1,
  "controversies": [],
  "dataAsOfDate": "2025-10-27"
}
```

### 2. World-Check AML API

**Configuration**:
- Endpoint: `https://api.world-check.com/screen`
- Authentication: Basic auth (username:password)
- Rate Limit: 100 requests/minute

**Environment Variables**:
```bash
VITE_WORLD_CHECK_USERNAME=your_username
VITE_WORLD_CHECK_PASSWORD=your_password
VITE_WORLD_CHECK_ENDPOINT=https://api.world-check.com/screen
```

**Request Example**:
```json
POST /screen
{
  "entityName": "John Smith",
  "entityType": "INDIVIDUAL",
  "screeningDate": "2025-10-27"
}
```

**Response Structure**:
```json
{
  "screeningId": "screening_12345",
  "entityName": "John Smith",
  "riskLevel": "LOW",
  "matches": [],
  "screeningDate": "2025-10-27"
}
```

### 3. Bloomberg Benchmark API

**Configuration**:
- Endpoint: `https://api.bloomberg.com/benchmark-data`
- Authentication: Bearer token
- Rate Limit: 1000 requests/day

**Environment Variables**:
```bash
VITE_BLOOMBERG_TOKEN=your_token
VITE_BLOOMBERG_ENDPOINT=https://api.bloomberg.com/benchmark-data
```

**Request Example**:
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "https://api.bloomberg.com/benchmark-data?index=SP500&startDate=2024-01-01&endDate=2025-10-27"
```

### 4. AWS SageMaker Risk Model

**Configuration**:
- Type: Custom trained ML model for portfolio VaR/stress testing
- Endpoint: Your deployed SageMaker endpoint
- Input: JSON with holdings and portfolio composition
- Output: VaR, stress test results, risk recommendations

**Environment Variables**:
```bash
VITE_SAGEMAKER_ENDPOINT=https://your-sagemaker-endpoint.amazonaws.com/invocations
```

**Model Input Structure**:
```json
{
  "holdings": [
    {
      "ticker": "AAPL",
      "weight": 0.25,
      "price": 150.00,
      "volatility": 0.28
    },
    {
      "ticker": "BND",
      "weight": 0.40,
      "price": 82.00,
      "volatility": 0.06
    }
  ],
  "correlationMatrix": [[1.0, -0.15], [-0.15, 1.0]],
  "historicalReturns": [0.12, 0.08, -0.05, 0.18]
}
```

**Model Output Structure**:
```json
{
  "var95": 0.045,
  "var99": 0.065,
  "cvar": 0.085,
  "stressTests": {
    "market_crash_10": -0.095,
    "interest_rate_spike": -0.032,
    "currency_volatility": -0.018
  },
  "recommendations": [
    "Consider reducing equity allocation to 55%",
    "Diversify international exposure"
  ]
}
```

---

## Testing & Validation

### 1. Unit Tests

**File**: `frontend/src/__tests__/validationRules.test.ts`

```typescript
import { validateParameters } from '@/data/ValidationRuleParametersRegistry';
import WEALTH_VALIDATION_RULES from '@/data/wealthValidationRules';

describe('Validation Rules', () => {
  test('ESG Compliance rule has required parameters', () => {
    const esgRule = WEALTH_VALIDATION_RULES.find(r => r.id === 'esg-compliance-v1');
    expect(esgRule).toBeDefined();
    expect(esgRule!.parameters.minEsgScore).toBeDefined();
    expect(esgRule!.parameters.integrationEndpoint).toBeDefined();
  });

  test('AI Risk Assessment parameters validate correctly', () => {
    const validation = validateParameters('AI-Driven Risk Assessment', {
      maxVaR: 0.05,
      varConfidenceLevel: 0.95,
      stressTestScenarios: ['market_crash_10'],
      aiModelEndpoint: 'https://api.example.com/model',
      modelType: 'tensorflow_var',
      integrationTimeout: 30000
    });
    
    expect(validation.valid).toBe(true);
  });

  test('Parameter validation catches missing required fields', () => {
    const validation = validateParameters('Margin Compliance', {
      initialMarginLimit: 0.5
      // Missing maintenanceMarginLimit and others
    });
    
    expect(validation.valid).toBe(false);
    expect(validation.errors.length).toBeGreaterThan(0);
  });
});
```

### 2. Integration Tests

**Test Rule Import Flow**:

```typescript
describe('Rule Import and Execution', () => {
  test('Import all advanced wealth rules successfully', async () => {
    const tenantId = 'test-tenant-123';
    const datasourceId = 'test-datasource-456';
    
    for (const rule of WEALTH_VALIDATION_RULES.slice(20, 30)) {
      const response = await fetch('/api/validation-rules', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId
        },
        body: JSON.stringify(rule)
      });
      
      expect(response.status).toBe(201);
      const data = await response.json();
      expect(data.id).toBeDefined();
    }
  });

  test('Execute ESG Compliance rule with sample context', async () => {
    const context = createSampleValidationContext(
      'account-123',
      'INDIVIDUAL_ACCOUNT',
      tenantId,
      datasourceId
    );
    
    const response = await fetch('/api/validation-rules/execute', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': tenantId,
        'X-Tenant-Datasource-ID': datasourceId
      },
      body: JSON.stringify({
        ruleId: 'esg-compliance-v1',
        context
      })
    });
    
    expect(response.status).toBe(200);
    const result = await response.json();
    expect(result.passed).toBeDefined();
    expect(result.message).toBeDefined();
  });
});
```

### 3. Manual Testing Checklist

- [ ] Navigate to Fabric Builder and select a tenant/datasource
- [ ] Click "Import Wealth Rules" and verify HTTP 201 responses for all 30 rules
- [ ] Filter rules by "Advanced Wealth Mgmt" and "Competitive Features" facets
- [ ] Create a new rule instance using ValidationRuleCreator
  - [ ] Select "ESG Compliance" rule
  - [ ] Adjust minEsgScore and restrictedSectors
  - [ ] Click Create and verify success
- [ ] Edit an existing rule
  - [ ] Click Edit on "Margin Compliance" rule
  - [ ] Change initialMarginLimit to 0.45
  - [ ] Save and verify update
- [ ] Test external API integrations (if credentials available)
  - [ ] Execute ESG rule and verify MSCI API call succeeds
  - [ ] Execute AML rule and verify World-Check screening succeeds
  - [ ] Execute AI Risk rule and verify SageMaker endpoint response

---

## Performance Considerations

### 1. Caching Strategy

All external API responses are cached with TTLs:
- ESG Ratings: 24 hours
- AML Screenings: 7 days
- Benchmark Data: 1 day
- AI Risk Assessments: 1 hour

**Cache Implementation** (ExternalApiIntegrationService):
```typescript
private cache: Map<string, CacheEntry<any>> = new Map();

private getFromCache<T>(key: string): T | null {
  const entry = this.cache.get(key);
  if (!entry) return null;
  
  if (Date.now() - entry.timestamp > entry.ttl) {
    this.cache.delete(key);
    return null;
  }
  
  return entry.data as T;
}
```

### 2. Concurrency & Timeouts

- ESG/AML requests: 10-15 second timeout
- SageMaker AI requests: 30 second timeout
- Bloomberg requests: 12 second timeout
- Retry logic: 3 attempts with exponential backoff

### 3. Database Optimization

Index on validation rules table for fast filtering:
```sql
CREATE INDEX idx_validation_rules_evaluation_order 
  ON validation_rules(evaluation_order);
CREATE INDEX idx_validation_rules_active 
  ON validation_rules(is_active, evaluation_order);
```

### 4. Batch Processing

For daily portfolio drift and AML monitoring:
- Process portfolios in batches of 100
- Parallelize external API calls with Promise.all()
- Implement rate limiting to respect API quotas

---

## Security & Compliance

### 1. API Credentials Management

- Store API keys in environment variables or AWS Secrets Manager
- Never commit credentials to repository
- Use short-lived tokens where possible
- Implement credential rotation policies

**Example .env.local**:
```bash
# Not to be committed to git!
VITE_MSCI_API_KEY=sk-msci-xxxxxxxxxxxx
VITE_WORLD_CHECK_USERNAME=username
VITE_WORLD_CHECK_PASSWORD=password
VITE_BLOOMBERG_TOKEN=tok-bbg-xxxxxxxxxxxx
VITE_SAGEMAKER_ENDPOINT=https://your-endpoint.sagemaker.amazonaws.com
```

> Tip: Front-end code should use Vite env names (import.meta.env.VITE_*) or the `getEnv()` helper
> which supports both `REACT_APP_*` (legacy) and `VITE_*` (Vite) variants.

### 2. Data Privacy

- Comply with GDPR when processing client ESG preferences
- Maintain audit trail for all rule executions
- Log rule violations and compliance events
- Anonymize personal data in logs

### 3. Regulatory Compliance

- **ESG Rule**: Aligns with SEC ESG disclosure requirements (SAB 121 considered)
- **Margin Rule**: FINRA Rule 4210 compliant
- **AML Rule**: Bank Secrecy Act, FinCEN SAR requirements
- **Communication Rule**: SEC Rule 206(4)-1, FINRA 2210
- **Alternative Investments**: SEC Regulation D accreditation requirements

### 4. Audit Trail

All rule executions logged with:
```json
{
  "ruleId": "esg-compliance-v1",
  "ruleName": "ESG Compliance",
  "portfolioId": "port-123",
  "executedAt": "2025-10-27T14:30:00Z",
  "executedBy": "system",
  "result": "PASS",
  "duration": 1250,
  "externalApiCalls": [
    {
      "service": "msci_api",
      "endpoint": "/esg-ratings",
      "duration": 800,
      "cached": false
    }
  ],
  "tenantId": "tenant-001",
  "datasourceId": "datasource-001"
}
```

---

## Deployment Checklist

- [ ] Deploy ValidationRuleParametersRegistry to frontend
- [ ] Deploy ExternalApiIntegrationService to frontend
- [ ] Update backend handlers for new rule types
- [ ] Configure environment variables for external APIs
- [ ] Update PostgreSQL schema if needed
- [ ] Deploy database migrations
- [ ] Run unit and integration tests
- [ ] Verify all rules import successfully
- [ ] Test external API integrations
- [ ] Monitor API rate limits and performance
- [ ] Document custom configurations in runbooks
- [ ] Train team on new rule usage

---

## Troubleshooting

### Common Issues

**Issue**: ESG rule fails with "MSCI API key not configured"
- **Solution**: Set `VITE_MSCI_API_KEY` environment variable before build (frontend). For legacy or server-side use, `REACT_APP_MSCI_API_KEY` may be recognized by `getEnv()`; however, prefer `VITE_*` naming for all new frontend configuration.

**Issue**: AML screening takes >15 seconds
- **Solution**: Check World-Check API rate limits; implement batch processing

**Issue**: SageMaker endpoint returns 503
- **Solution**: Verify endpoint is deployed and active; check IAM permissions

**Issue**: Rules don't appear in UI after import
- **Solution**: Clear browser cache; verify tenant/datasource scope is set

---

## Next Steps

1. **Implement AI Model**: Deploy TensorFlow/SageMaker model for risk assessment
2. **API Integrations**: Set up production API credentials and endpoints
3. **Advanced UI**: Add rule conflict detection and optimization suggestions
4. **Reporting**: Create compliance reports from rule execution logs
5. **Webhooks**: Add webhook notifications for rule violations
6. **GraphQL**: Migrate to GraphQL for flexible rule querying

---

## References

- **Workday Financial Management**: https://www.workday.com/en-us/products/financial-management/overview.html
- **FINRA Rule 4210**: https://www.finra.org/rules-guidance/rulebooks/finra-rules/4210
- **SEC Rule 206(4)-1**: https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&SEC_ACCOUNT_ID=&owner=exclude&match=&count=100
- **MSCI ESG API**: https://www.msci.com/
- **World-Check OneWorld**: https://www.refinitiv.com/
- **AWS SageMaker**: https://aws.amazon.com/sagemaker/

