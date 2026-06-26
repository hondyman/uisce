# Advanced Wealth Management Rules - UI Integration Guide

This guide provides detailed instructions for integrating the new advanced wealth management rules into the Fabric Builder's validation rule UI components.

## Overview

10 new validation rules have been added to `wealthValidationRules.ts` (evaluationOrder 21-30):
- **Advanced Wealth Management** (21-25): Tax Optimization, ESG Compliance, Regulatory Margin, Portfolio Drift, Communication Compliance
- **Competitive Management** (26-30): AI Risk Assessment, Client Engagement, Performance Benchmarking, AML Compliance, Alternative Investments

## Component Updates Required

### 1. ValidationRuleCreator.tsx

**Purpose**: Add form fields for new rule-specific parameters.

**Changes**:

```typescript
// Add to the parameters rendering section
const renderParameterFields = (ruleType: string, ruleName: string) => {
  const baseFields = (
    <>
      {/* Common business_logic fields */}
    </>
  );

  // Tax Optimization (Rule 21)
  if (ruleName === 'Tax Optimization') {
    return (
      <>
        {baseFields}
        <div className="parameter-group">
          <label>Max Taxable Gain Percentage</label>
          <input
            type="number"
            step="0.01"
            min="0"
            max="1"
            defaultValue={0.15}
            onChange={(e) => updateParameter('maxTaxableGainPercentage', parseFloat(e.target.value))}
          />
          <small>Percentage of trades that can result in taxable gains (0.15 = 15%)</small>
        </div>
        <div className="parameter-group">
          <label>Wash Sale Window (Days)</label>
          <input
            type="number"
            min="1"
            max="60"
            defaultValue={30}
            onChange={(e) => updateParameter('washSaleWindowDays', parseInt(e.target.value))}
          />
          <small>Days before/after sale to prohibit repurchase of substantially identical securities</small>
        </div>
        <div className="parameter-group">
          <label>Tax Bracket Thresholds</label>
          <JsonEditor
            value={formData.parameters.taxBracketThresholds}
            onChange={(value) => updateParameter('taxBracketThresholds', value)}
            schema={{
              type: 'array',
              items: {
                type: 'object',
                properties: {
                  bracket: { type: 'string', enum: ['LOW', 'MEDIUM', 'HIGH'] },
                  maxGain: { type: 'number' }
                }
              }
            }}
          />
        </div>
      </>
    );
  }

  // ESG Compliance (Rule 22)
  if (ruleName === 'ESG Compliance') {
    return (
      <>
        {baseFields}
        <div className="parameter-group">
          <label>Minimum ESG Score</label>
          <input
            type="number"
            step="0.1"
            min="0"
            max="10"
            defaultValue={7.0}
            onChange={(e) => updateParameter('minEsgScore', parseFloat(e.target.value))}
          />
          <small>MSCI ESG rating minimum (0-10 scale)</small>
        </div>
        <div className="parameter-group">
          <label>Max ESG Score Deviation</label>
          <input
            type="number"
            step="0.1"
            min="0"
            max="5"
            defaultValue={2.0}
            onChange={(e) => updateParameter('maxEsgScoreDeviation', parseFloat(e.target.value))}
          />
          <small>Maximum allowed deviation from portfolio target ESG score</small>
        </div>
        <div className="parameter-group">
          <label>Restricted Sectors (comma-separated)</label>
          <input
            type="text"
            placeholder="OIL_GAS,TOBACCO,WEAPONS"
            defaultValue={(formData.parameters.restrictedSectors || []).join(',')}
            onChange={(e) => updateParameter('restrictedSectors', e.target.value.split(',').map(s => s.trim()))}
          />
        </div>
        <div className="parameter-group">
          <label>ESG Data Source</label>
          <select
            value={formData.parameters.esgDataSource || 'msci_api'}
            onChange={(e) => updateParameter('esgDataSource', e.target.value)}
          >
            <option value="msci_api">MSCI API</option>
            <option value="refinitiv">Refinitiv</option>
            <option value="sustainalytics">Sustainalytics</option>
            <option value="internal">Internal Database</option>
          </select>
        </div>
      </>
    );
  }

  // Regulatory Margin Compliance (Rule 23)
  if (ruleName === 'Regulatory Margin Compliance') {
    return (
      <>
        {baseFields}
        <div className="parameter-group">
          <label>Initial Margin Limit</label>
          <input
            type="number"
            step="0.01"
            min="0"
            max="1"
            defaultValue={0.5}
            onChange={(e) => updateParameter('initialMarginLimit', parseFloat(e.target.value))}
          />
          <small>Fraction of portfolio that can be borrowed (0.5 = 50%)</small>
        </div>
        <div className="parameter-group">
          <label>Maintenance Margin Limit</label>
          <input
            type="number"
            step="0.01"
            min="0"
            max="1"
            defaultValue={0.25}
            onChange={(e) => updateParameter('maintenanceMarginLimit', parseFloat(e.target.value))}
          />
          <small>Minimum equity percentage to maintain (0.25 = 25%)</small>
        </div>
        <div className="parameter-group">
          <label>Max Loan Value ($)</label>
          <input
            type="number"
            min="0"
            step="100000"
            defaultValue={1000000}
            onChange={(e) => updateParameter('maxLoanValue', parseInt(e.target.value))}
          />
        </div>
        <div className="parameter-group">
          <label>Margin Call Threshold</label>
          <input
            type="number"
            step="0.01"
            min="0"
            max="1"
            defaultValue={0.3}
            onChange={(e) => updateParameter('marginCallThreshold', parseFloat(e.target.value))}
          />
          <small>Equity percentage at which margin call is triggered</small>
        </div>
      </>
    );
  }

  // Portfolio Drift Detection (Rule 24)
  if (ruleName === 'Portfolio Drift Detection') {
    return (
      <>
        {baseFields}
        <div className="parameter-group">
          <label>Max Drift Percentage</label>
          <input
            type="number"
            step="0.01"
            min="0"
            max="1"
            defaultValue={0.05}
            onChange={(e) => updateParameter('maxDriftPercentage', parseFloat(e.target.value))}
          />
          <small>Maximum allowed deviation from target allocation (0.05 = 5%)</small>
        </div>
        <div className="parameter-group">
          <label>Rebalancing Threshold</label>
          <input
            type="number"
            step="0.01"
            min="0"
            max="1"
            defaultValue={0.08}
            onChange={(e) => updateParameter('rebalancingThreshold', parseFloat(e.target.value))}
          />
          <small>Drift percentage at which rebalancing is recommended (0.08 = 8%)</small>
        </div>
        <div className="parameter-group">
          <label>Target Allocations (%)</label>
          <div className="allocation-inputs">
            <div>
              <label>Equity %</label>
              <input
                type="number"
                step="1"
                min="0"
                max="100"
                defaultValue={60}
                onChange={(e) => {
                  const alloc = { ...formData.parameters.targetAllocations };
                  alloc.EQUITY = parseInt(e.target.value) / 100;
                  updateParameter('targetAllocations', alloc);
                }}
              />
            </div>
            <div>
              <label>Fixed Income %</label>
              <input
                type="number"
                step="1"
                min="0"
                max="100"
                defaultValue={35}
                onChange={(e) => {
                  const alloc = { ...formData.parameters.targetAllocations };
                  alloc.FIXED_INCOME = parseInt(e.target.value) / 100;
                  updateParameter('targetAllocations', alloc);
                }}
              />
            </div>
            <div>
              <label>Cash %</label>
              <input
                type="number"
                step="1"
                min="0"
                max="100"
                defaultValue={5}
                onChange={(e) => {
                  const alloc = { ...formData.parameters.targetAllocations };
                  alloc.CASH = parseInt(e.target.value) / 100;
                  updateParameter('targetAllocations', alloc);
                }}
              />
            </div>
          </div>
        </div>
      </>
    );
  }

  // Communication Compliance (Rule 25)
  if (ruleName === 'Communication Compliance') {
    return (
      <>
        {baseFields}
        <div className="parameter-group">
          <label>Prohibited Phrases (one per line)</label>
          <textarea
            placeholder="guaranteed return&#10;risk-free&#10;assured profit"
            defaultValue={(formData.parameters.prohibitedPhrases || []).join('\n')}
            onChange={(e) => updateParameter('prohibitedPhrases', e.target.value.split('\n').map(p => p.trim()).filter(p => p))}
          />
          <small>Phrases that violate SEC Rule 206(4)-1</small>
        </div>
        <div className="parameter-group">
          <label>Required Disclosures (one per line)</label>
          <textarea
            placeholder="past performance disclaimer&#10;fee disclosure&#10;conflict of interest"
            defaultValue={(formData.parameters.requiredDisclosures || []).join('\n')}
            onChange={(e) => updateParameter('requiredDisclosures', e.target.value.split('\n').map(d => d.trim()).filter(d => d))}
          />
        </div>
      </>
    );
  }

  // AI-Driven Risk Assessment (Rule 26)
  if (ruleName === 'AI-Driven Risk Assessment') {
    return (
      <>
        {baseFields}
        <div className="parameter-group">
          <label>Max Value-at-Risk (VaR)</label>
          <input
            type="number"
            step="0.01"
            min="0"
            max="1"
            defaultValue={0.05}
            onChange={(e) => updateParameter('maxVaR', parseFloat(e.target.value))}
          />
          <small>Maximum allowed VaR (0.05 = 5% daily loss threshold)</small>
        </div>
        <div className="parameter-group">
          <label>VaR Confidence Level</label>
          <input
            type="number"
            step="0.01"
            min="0.90"
            max="0.99"
            defaultValue={0.95}
            onChange={(e) => updateParameter('varConfidenceLevel', parseFloat(e.target.value))}
          />
          <small>Confidence interval for VaR calculation (0.95 = 95%)</small>
        </div>
        <div className="parameter-group">
          <label>Stress Test Scenarios (comma-separated)</label>
          <input
            type="text"
            placeholder="market_crash_10,interest_rate_spike,currency_volatility"
            defaultValue={(formData.parameters.stressTestScenarios || []).join(',')}
            onChange={(e) => updateParameter('stressTestScenarios', e.target.value.split(',').map(s => s.trim()))}
          />
        </div>
        <div className="parameter-group">
          <label>AI Model Endpoint</label>
          <input
            type="url"
            placeholder="https://api.sagemaker.example.com/risk-model"
            defaultValue={formData.parameters.aiModelEndpoint || ''}
            onChange={(e) => updateParameter('aiModelEndpoint', e.target.value)}
          />
          <small>AWS SageMaker or custom ML endpoint URL</small>
        </div>
        <div className="parameter-group">
          <label>Model Type</label>
          <select
            value={formData.parameters.modelType || 'tensorflow_var'}
            onChange={(e) => updateParameter('modelType', e.target.value)}
          >
            <option value="tensorflow_var">TensorFlow VAR</option>
            <option value="pytorch_risk">PyTorch Risk Model</option>
            <option value="sagemaker_autopilot">SageMaker Autopilot</option>
          </select>
        </div>
      </>
    );
  }

  // Client Engagement Tracking (Rule 27)
  if (ruleName === 'Client Engagement Tracking') {
    return (
      <>
        {baseFields}
        <div className="parameter-group">
          <label>Min Interaction Frequency (Days)</label>
          <input
            type="number"
            min="1"
            defaultValue={90}
            onChange={(e) => updateParameter('minInteractionFrequencyDays', parseInt(e.target.value))}
          />
          <small>Expected days between client interactions</small>
        </div>
        <div className="parameter-group">
          <label>Trigger Events (comma-separated)</label>
          <input
            type="text"
            placeholder="portfolio_drop_10_percent,portfolio_gain_15_percent,rebalancing_required"
            defaultValue={(formData.parameters.triggerEvents || []).join(',')}
            onChange={(e) => updateParameter('triggerEvents', e.target.value.split(',').map(t => t.trim()))}
          />
        </div>
        <div className="parameter-group">
          <label>Notification Channel</label>
          <select
            value={formData.parameters.notificationChannel || 'email'}
            onChange={(e) => updateParameter('notificationChannel', e.target.value)}
          >
            <option value="email">Email</option>
            <option value="sms">SMS</option>
            <option value="in_app">In-App</option>
            <option value="all">All Channels</option>
          </select>
        </div>
        <div className="parameter-group">
          <label>Escalation Threshold (Days)</label>
          <input
            type="number"
            min="1"
            defaultValue={180}
            onChange={(e) => updateParameter('escalationThreshold', parseInt(e.target.value))}
          />
          <small>Days without interaction before escalation</small>
        </div>
      </>
    );
  }

  // Performance Benchmarking (Rule 28)
  if (ruleName === 'Performance Benchmarking') {
    return (
      <>
        {baseFields}
        <div className="parameter-group">
          <label>Primary Benchmark Index</label>
          <select
            value={formData.parameters.benchmarkIndex || 'SP500'}
            onChange={(e) => updateParameter('benchmarkIndex', e.target.value)}
          >
            <option value="SP500">S&P 500</option>
            <option value="NASDAQ">NASDAQ-100</option>
            <option value="RUSSELL2000">Russell 2000</option>
            <option value="MSCI_WORLD">MSCI World</option>
            <option value="CUSTOM">Custom Benchmark</option>
          </select>
        </div>
        <div className="parameter-group">
          <label>Min Performance Delta (%)</label>
          <input
            type="number"
            step="0.01"
            defaultValue={-2}
            onChange={(e) => updateParameter('minPerformanceDelta', parseFloat(e.target.value) / 100)}
          />
          <small>Allowed underperformance (-0.02 = -2%)</small>
        </div>
        <div className="parameter-group">
          <label>Evaluation Period (Months)</label>
          <input
            type="number"
            min="1"
            step="1"
            defaultValue={12}
            onChange={(e) => updateParameter('evaluationPeriodMonths', parseInt(e.target.value))}
          />
        </div>
        <div className="parameter-group">
          <label>Data Source</label>
          <select
            value={formData.parameters.dataSource || 'bloomberg_api'}
            onChange={(e) => updateParameter('dataSource', e.target.value)}
          >
            <option value="bloomberg_api">Bloomberg API</option>
            <option value="refinitiv">Refinitiv</option>
            <option value="yahoo_finance">Yahoo Finance</option>
            <option value="internal">Internal Database</option>
          </select>
        </div>
      </>
    );
  }

  // AML Compliance (Rule 29)
  if (ruleName === 'Anti-Money Laundering (AML) Compliance') {
    return (
      <>
        {baseFields}
        <div className="parameter-group">
          <label>Transaction Threshold ($)</label>
          <input
            type="number"
            min="0"
            step="1000"
            defaultValue={10000}
            onChange={(e) => updateParameter('transactionThreshold', parseInt(e.target.value))}
          />
          <small>Amount triggering AML review</small>
        </div>
        <div className="parameter-group">
          <label>Cumulative Threshold ($)</label>
          <input
            type="number"
            min="0"
            step="1000"
            defaultValue={50000}
            onChange={(e) => updateParameter('cumulativeThreshold', parseInt(e.target.value))}
          />
        </div>
        <div className="parameter-group">
          <label>Cumulative Window (Days)</label>
          <input
            type="number"
            min="1"
            defaultValue={30}
            onChange={(e) => updateParameter('cumulativeWindowDays', parseInt(e.target.value))}
          />
        </div>
        <div className="parameter-group">
          <label>Suspicious Patterns (comma-separated)</label>
          <input
            type="text"
            placeholder="rapid_transfers,high_frequency_small_amounts,round_number_trades"
            defaultValue={(formData.parameters.suspiciousPatterns || []).join(',')}
            onChange={(e) => updateParameter('suspiciousPatterns', e.target.value.split(',').map(p => p.trim()))}
          />
        </div>
        <div className="parameter-group">
          <label>AML Screening Service</label>
          <select
            value={formData.parameters.amlScreeningService || 'world_check_api'}
            onChange={(e) => updateParameter('amlScreeningService', e.target.value)}
          >
            <option value="world_check_api">World-Check (Refinitiv)</option>
            <option value="sanctions_screening">Sanctions Screening Service</option>
            <option value="internal">Internal Screening</option>
          </select>
        </div>
      </>
    );
  }

  // Alternative Investments Eligibility (Rule 30)
  if (ruleName === 'Alternative Investments Eligibility') {
    return (
      <>
        {baseFields}
        <div className="parameter-group">
          <label>Minimum Net Worth ($)</label>
          <input
            type="number"
            min="0"
            step="100000"
            defaultValue={2000000}
            onChange={(e) => updateParameter('minNetWorth', parseInt(e.target.value))}
          />
        </div>
        <div className="parameter-group">
          <label>Minimum Annual Income ($)</label>
          <input
            type="number"
            min="0"
            step="10000"
            defaultValue={200000}
            onChange={(e) => updateParameter('minAnnualIncome', parseInt(e.target.value))}
          />
        </div>
        <div className="parameter-group">
          <label>Max Alternative Allocation</label>
          <input
            type="number"
            step="0.01"
            min="0"
            max="1"
            defaultValue={0.2}
            onChange={(e) => updateParameter('maxAlternativeAllocation', parseFloat(e.target.value))}
          />
          <small>Maximum portfolio percentage in alternatives (0.2 = 20%)</small>
        </div>
        <div className="parameter-group">
          <label>Alternative Asset Types (comma-separated)</label>
          <input
            type="text"
            placeholder="PRIVATE_EQUITY,HEDGE_FUND,REAL_ESTATE,COMMODITIES"
            defaultValue={(formData.parameters.alternativeAssetTypes || []).join(',')}
            onChange={(e) => updateParameter('alternativeAssetTypes', e.target.value.split(',').map(a => a.trim()))}
          />
        </div>
        <div className="parameter-group">
          <label>
            <input
              type="checkbox"
              checked={formData.parameters.requiredAccreditation !== false}
              onChange={(e) => updateParameter('requiredAccreditation', e.target.checked)}
            />
            Require Accredited Investor Status
          </label>
        </div>
        <div className="parameter-group">
          <label>Accreditation Revalidation (Days)</label>
          <input
            type="number"
            min="1"
            defaultValue={365}
            onChange={(e) => updateParameter('accreditationRevalidationDays', parseInt(e.target.value))}
          />
        </div>
      </>
    );
  }

  return null;
};
```

### 2. ValidationRuleEditor.tsx

**Purpose**: Update existing rules with new parameters.

**Key Updates**:
- Add similar form fields for editing existing rules
- Include validation for parameter ranges (e.g., percentages 0-1)
- Add preview of current settings

### 3. ValidationRulesWithFacets.tsx

**Purpose**: Update facet filters to include new rules.

**Changes**:

```typescript
// Add new rule names to filters
const ruleNameFacets = [
  // ... existing rules ...
  'Tax Optimization',
  'ESG Compliance',
  'Regulatory Margin Compliance',
  'Portfolio Drift Detection',
  'Communication Compliance',
  'AI-Driven Risk Assessment',
  'Client Engagement Tracking',
  'Performance Benchmarking',
  'Anti-Money Laundering (AML) Compliance',
  'Alternative Investments Eligibility'
];

// Update severity and frequency filters
const severityFacets = ['BLOCK', 'WARNING', 'INFO'];
const frequencyFacets = ['CONTINUOUS', 'ON_TRADE', 'DAILY', 'MONTHLY', 'ANNUALLY', 'ON_CHANGE', 'ON_REBALANCE'];
```

## Testing the Integration

### 1. Import Test

```bash
# Build frontend
cd /Users/eganpj/GitHub/semlayer/frontend
npm run build

# Start dev server
npm run dev
```

### 2. Manual Verification

1. Navigate to http://localhost:5173
2. Select a tenant and datasource
3. Go to "Import Wealth Rules" section
4. Trigger the import
5. Check Network tab in DevTools for HTTP 201 responses
6. Verify all 30 rules appear in the validation rules list

### 3. Rule Creation Test

1. Click "Create New Rule"
2. Select "Tax Optimization" from the dropdown
3. Verify all tax-specific parameters appear
4. Submit and check backend confirmation

## Console Output Example

Expected logs on successful import:

```javascript
✅ Importing 30 wealth validation rules...
Rule 1: concentration-limit-v1
Rule 2: kyc-completeness-v1
...
Rule 21: tax-optimization-v1
Rule 22: esg-compliance-v1
Rule 23: margin-compliance-v1
Rule 24: portfolio-drift-v1
Rule 25: communication-compliance-v1
Rule 26: ai-risk-assessment-v1
Rule 27: client-engagement-v1
Rule 28: performance-benchmarking-v1
Rule 29: aml-compliance-v1
Rule 30: alternative-investments-v1
✅ Import completed: 30 rules created (30 successful, 0 failed)
```

## Next Steps

1. **Update UI Components**: Implement form fields in `ValidationRuleCreator.tsx`
2. **Add Validation**: Add parameter validation and type checking
3. **Backend Integration**: Update rule execution engine (see ADVANCED_WEALTH_RULES_BACKEND_INTEGRATION.md)
4. **Test API Integrations**: Configure and test external service endpoints
