# Advanced Wealth Validation Rules - Quick Reference

## New Rules at a Glance

### Tier 1: Advanced Wealth Management (21-25)

| Rule ID | Name | Severity | Frequency | Key Parameters |
|---------|------|----------|-----------|-----------------|
| `tax-optimization-v1` | Tax Optimization | WARNING | ON_TRADE | maxTaxableGainPercentage (15%), washSaleWindowDays (30) |
| `esg-compliance-v1` | ESG Compliance | WARNING | DAILY | minEsgScore (7.0), restrictedSectors, **API: MSCI** |
| `margin-compliance-v1` | Margin Compliance | BLOCK | ON_TRADE | initialMarginLimit (50%), maintenanceMarginLimit (25%) |
| `portfolio-drift-v1` | Portfolio Drift | WARNING | DAILY | maxDriftPercentage (5%), targetAllocations |
| `communication-compliance-v1` | Communication Compliance | BLOCK | ON_CHANGE | prohibitedPhrases, requiredDisclosures, **SEC 206(4)-1** |

### Tier 2: Competitive Management (26-30)

| Rule ID | Name | Severity | Frequency | Key Parameters |
|---------|------|----------|-----------|-----------------|
| `ai-risk-assessment-v1` | AI Risk Assessment | WARNING | DAILY | maxVaR (5%), **API: AWS SageMaker**, varConfidenceLevel (95%) |
| `client-engagement-v1` | Client Engagement | INFO | DAILY | minInteractionFrequencyDays (90), triggerEvents |
| `performance-benchmarking-v1` | Performance Benchmarking | INFO | MONTHLY | benchmarkIndex (SP500), minPerformanceDelta (-2%), **API: Bloomberg** |
| `aml-compliance-v1` | AML Compliance | BLOCK | ON_TRADE | transactionThreshold ($10K), **API: World-Check** |
| `alternative-investments-v1` | Alternative Investments | BLOCK | ON_TRADE | minNetWorth ($2M), maxAlternativeAllocation (20%) |

---

## Files to Know

### Frontend Files

```
frontend/src/
├── data/
│   ├── wealthValidationRules.ts              ✨ 30 rules (updated with 10 new)
│   └── ValidationRuleParametersRegistry.ts   ✨ NEW - Dynamic parameter configs
├── services/
│   └── ExternalApiIntegrationService.ts      ✨ NEW - MSCI, World-Check, Bloomberg, SageMaker
├── pages/bundles/
│   ├── BundleListPage.tsx
│   ├── BundleEditor.tsx
│   └── ValidationRulesWithFacets.tsx         (update facets for new rules)
└── components/
    ├── ValidationRuleCreator.tsx              (update for dynamic parameters)
    ├── ValidationRuleEditor.tsx               (update for dynamic parameters)
    └── ValidationRuleList.tsx
```

### Backend Files

```
backend/
├── internal/
│   ├── api/
│   │   └── validation_rules_routes.go         (add handlers for new rules)
│   └── services/
│       ├── external_api_client.go             ✨ NEW - External API calls
│       └── validation_engine.go               (extend for new rule types)
└── db/
    └── migrations/
        └── validate_rules_schema.sql          (ensure schema supports all rules)
```

---

## Environment Variables (Frontend)

Add to `.env.local` before running tests or integration with real APIs:

```bash
# MSCI ESG API
VITE_MSCI_API_KEY=your_api_key_here
VITE_MSCI_ENDPOINT=https://api.msci.com/esg-ratings

# World-Check AML API
VITE_WORLD_CHECK_USERNAME=your_username
VITE_WORLD_CHECK_PASSWORD=your_password
VITE_WORLD_CHECK_ENDPOINT=https://api.world-check.com/screen

# Bloomberg API
VITE_BLOOMBERG_TOKEN=your_token_here
VITE_BLOOMBERG_ENDPOINT=https://api.bloomberg.com/benchmark-data

# AWS SageMaker
VITE_SAGEMAKER_ENDPOINT=https://your-endpoint.sagemaker.amazonaws.com/invocations
```

---

## Quick Start: Import & Execute Rules

### 1. Import All Rules

```bash
curl -X POST http://localhost:8080/api/validation-rules/import \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -d @WEALTH_VALIDATION_RULES.json
```

### 2. List All Rules (with new ones)

```bash
curl -X GET "http://localhost:8080/api/validation-rules?tenant_id=00000000-0000-0000-0000-000000000000&datasource_id=11111111-1111-1111-1111-111111111111" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111"
```

Expected: 30 rules total (20 core + 10 new)

### 3. Execute a Rule

```bash
curl -X POST http://localhost:8080/api/validation-rules/execute \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{
    "ruleId": "esg-compliance-v1",
    "context": {
      "accountId": "account-123",
      "accountType": "INDIVIDUAL_ACCOUNT",
      "portfolio": {
        "holdings": [
          { "ticker": "AAPL", "weight": 0.3, "price": 150, "quantity": 100 }
        ]
      }
    }
  }'
```

---

## UI Integration: Key Components

### ValidationRuleParametersRegistry Usage

```typescript
// Get parameter configs for a rule
import { getParametersForRule } from '@/data/ValidationRuleParametersRegistry';

const params = getParametersForRule('ESG Compliance');
// Returns: Array of ParameterConfig objects for dynamic form rendering
```

### ValidationRuleCreator with Dynamic Fields

```typescript
// Automatically render parameter fields based on selected rule
const selectedRule = 'ESG Compliance';
const configs = getParametersForRule(selectedRule);

configs.forEach(config => {
  // Render input based on config.type (text, number, checkbox, select, array, etc.)
  renderFormField(config);
});
```

### External API Service Usage

```typescript
import { externalApiService } from '@/services/ExternalApiIntegrationService';

// Get ESG rating
const esgRating = await externalApiService.getESGRating('AAPL', 'ticker');

// Screen for AML
const amlScreen = await externalApiService.screenAML('John Smith', 'INDIVIDUAL');

// Get benchmark performance
const benchmark = await externalApiService.getBenchmarkPerformance(
  'SP500',
  '2024-01-01',
  '2025-10-27'
);

// Assess portfolio risk using AI
const riskAssessment = await externalApiService.assessPortfolioRisk({
  holdings: [...],
  correlationMatrix: [...],
  historicalReturns: [...]
});
```

---

## Testing Checklist

### Manual Testing

- [ ] **Import**: Run import flow, verify all 30 rules created (HTTP 201)
- [ ] **Facets**: Filter by "Advanced Wealth Mgmt" and "Competitive Features"
- [ ] **Create**: Add new rule instance using ValidationRuleCreator
- [ ] **Edit**: Edit "ESG Compliance" rule, change minEsgScore
- [ ] **Parameters**: Verify all parameter fields display correctly for each rule type
- [ ] **APIs**: Test external API calls (if credentials available)

### Automated Testing

```bash
# Run unit tests
npm run test -- validationRules.test.ts

# Run integration tests with live APIs
npm run test:integration -- validation_rules.integration.ts

# Check TypeScript compilation
npm run build

# Run linter
npm run lint
```

---

## API Endpoints

### Create Rule
```
POST /api/validation-rules
Headers: X-Tenant-ID, X-Tenant-Datasource-ID
Body: ValidationRule object
Response: 201 { id, ...rule }
```

### List Rules
```
GET /api/validation-rules?tenant_id=...&datasource_id=...
Headers: X-Tenant-ID, X-Tenant-Datasource-ID
Response: 200 [ ...rules ]
```

### Get Rule
```
GET /api/validation-rules/{ruleId}
Headers: X-Tenant-ID, X-Tenant-Datasource-ID
Response: 200 { ...rule }
```

### Update Rule
```
PUT /api/validation-rules/{ruleId}
Headers: X-Tenant-ID, X-Tenant-Datasource-ID
Body: Updated rule fields
Response: 200 { ...updatedRule }
```

### Execute Rule
```
POST /api/validation-rules/execute
Headers: X-Tenant-ID, X-Tenant-Datasource-ID
Body: { ruleId, context }
Response: 200 { passed, severity, message, externalApiCalls }
```

### Import Bulk Rules
```
POST /api/validation-rules/import
Headers: X-Tenant-ID, X-Tenant-Datasource-ID
Body: { rules: [...] }
Response: 201 { imported: N, errors: [...] }
```

---

## Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| Rule List Load | < 500ms | With pagination |
| Single Rule Execute | < 2s | Without external APIs |
| ESG API Call | < 10s | Cached after first call |
| AML Screening | < 15s | Cached for 7 days |
| AI Risk Assessment | < 30s | Cached for 1 hour |
| Portfolio Drift Check | < 5s | Daily batch processing |
| Cache Hit Ratio | > 70% | For frequently used rules |

---

## Troubleshooting Guide

### Rule Not Appearing in UI
1. Clear browser cache: Ctrl+Shift+Delete (or Cmd+Shift+Delete on Mac)
2. Refresh page: F5
3. Check browser DevTools > Console for errors
4. Verify tenant/datasource is selected

### External API Timeouts
1. Check API credentials in environment variables
2. Verify API endpoint is accessible (curl test)
3. Check rate limits haven't been exceeded
4. Review API service health page

### Parameter Validation Errors
1. Verify parameters match expected types (number, string, array)
2. Check min/max constraints
3. Review required fields list

### Poor Performance
1. Check cache effectiveness (browser DevTools > Network)
2. Review database query performance
3. Monitor external API response times
4. Check for N+1 query problems in backend

---

## Competitive Advantages

### vs. SS&C Black Diamond

| Feature | Black Diamond | Fabric Builder |
|---------|---------------|-----------------|
| ESG Compliance | Basic | ✅ **MSCI Integration** |
| AI Risk Assessment | Static | ✅ **ML-Driven (SageMaker)** |
| Client Engagement | Manual | ✅ **Automated Triggers** |
| AML Screening | Limited | ✅ **World-Check Integration** |
| Tax Optimization | Basic | ✅ **Wash-Sale Aware** |
| Regulatory Compliance | Good | ✅ **Comprehensive** |
| Extensibility | Limited | ✅ **Metadata-Driven** |

---

## Next Steps

1. ✅ **Complete**: Added 10 new advanced rules to wealthValidationRules.ts
2. ✅ **Complete**: Created ValidationRuleParametersRegistry for dynamic UI
3. ✅ **Complete**: Created ExternalApiIntegrationService
4. 🔲 **Next**: Update backend handlers for new rule types
5. 🔲 **Next**: Configure external API credentials
6. 🔲 **Next**: Run import & execution tests
7. 🔲 **Next**: Deploy to production
8. 🔲 **Next**: Monitor rule execution metrics

---

## Resources

- **Implementation Guide**: See ADVANCED_WEALTH_RULES_IMPLEMENTATION_GUIDE.md
- **Rule Definitions**: frontend/src/data/wealthValidationRules.ts
- **Parameter Registry**: frontend/src/data/ValidationRuleParametersRegistry.ts
- **External APIs**: frontend/src/services/ExternalApiIntegrationService.ts
- **Agent Runbook**: agents.md (tenant scoping)

---

**Last Updated**: October 27, 2025
**Status**: Core implementation complete, external API integration pending
**Maintainer**: Development Team

