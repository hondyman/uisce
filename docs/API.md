# WealthVision API Documentation

## Overview

WealthVision is a comprehensive wealth management platform API providing advanced tax optimization, multi-generational planning, alternative investments, AI intelligence, ESG tracking, risk management, client portal, and compliance features.

**Base URL**: `https://api.wealthvision.com/v1`  
**Authentication**: Bearer Token (JWT)

---

## API Endpoints (40 Total)

### 1. Tax Optimization (3 endpoints)

#### POST /wealthvision/tax/state-residency
Compare tax savings across multiple states.

**Request:**
```json
{
  "family_id": "uuid",
  "current_state": "CA",
  "gross_income": 1000000,
  "investment_income": 500000,
  "capital_gains": 250000,
  "estate_value": 50000000,
  "states_to_compare": ["FL", "TX", "NV", "WA"]
}
```

**Response:**
```json
{
  "analysis_id": "uuid",
  "current_state_details": {...},
  "state_comparisons": [...],
  "top_recommendations": [...]
}
```

#### POST /wealthvision/tax/niit
Calculate Net Investment Income Tax (3.8%).

**Request:**
```json
{
  "family_id": "uuid",
  "member_id": "uuid",
  "tax_year": 2024,
  "filing_status": "MARRIED_JOINT",
  "modified_agi": 600000,
  "investment_income": {
    "interest": 50000,
    "dividends": 100000,
    "capital_gains": 150000,
    "rental_income": 50000
  }
}
```

#### POST /wealthvision/tax/charitable-bunching
Analyze charitable giving strategies.

---

### 2. Multi-Generational Planning (3 endpoints)

#### POST /wealthvision/multi-gen/dynasty-trust
Simulate dynasty trust across 3+ generations.

**Request:**
```json
{
  "family_id": "uuid",
  "initial_funding": 10000000,
  "generations": 3,
  "annual_growth_rate": 7.5,
  "distribution_rules": {...}
}
```

#### POST /wealthvision/multi-gen/529-plan
Optimize 529 education planning.

#### POST /wealthvision/multi-gen/legacy-impact
Calculate multi-generational philanthropic impact.

---

### 3. Alternative Investments (5 endpoints)

#### POST /wealthvision/alt-investments/pe-metrics
Calculate private equity metrics (IRR, MOIC, DPI, RVPI, TVPI).

**Request:**
```json
{
  "investment_id": "uuid",
  "family_id": "uuid",
  "commitment_amount": 5000000,
  "capital_calls": [...],
  "distributions": [...],
  "current_nav": 6500000
}
```

#### POST /wealthvision/alt-investments/vc-scenarios
Model venture capital exit scenarios.

#### POST /wealthvision/alt-investments/1031-exchange
Analyze 1031 exchange opportunities.

#### POST /wealthvision/alt-investments/art-appreciation
Track art & collectibles appreciation.

#### POST /wealthvision/alt-investments/hedge-fund-drift
Detect hedge fund style drift.

---

### 4. AI Intelligence (3 endpoints)

#### POST /wealthvision/ai/churn-prediction
Predict client churn risk using ML.

**Response:**
```json
{
  "prediction_id": "uuid",
  "churn_probability": 0.35,
  "risk_level": "MEDIUM",
  "key_factors": [...],
  "recommended_actions": [...]
}
```

#### POST /wealthvision/ai/meeting-prep
Generate AI-powered meeting preparation.

#### POST /wealthvision/ai/portfolio-optimization
Get AI portfolio optimization recommendations.

---

### 5. ESG Intelligence (3 endpoints)

#### POST /wealthvision/esg/carbon-footprint
Calculate portfolio carbon footprint.

**Response:**
```json
{
  "footprint_id": "uuid",
  "total_emissions_tons_co2": 4250,
  "scope1_emissions": 500,
  "scope2_emissions": 1500,
  "scope3_emissions": 2250,
  "reduction_strategies": [...]
}
```

#### POST /wealthvision/esg/score
Calculate ESG score with multiple providers.

#### POST /wealthvision/esg/impact-investing
Track impact investing metrics (SROI).

---

### 6. Risk Management (4 endpoints)

#### POST /risk/options/protective-put
Build protective put options strategy.

**Request:**
```json
{
  "portfolio_id": "uuid",
  "family_id": "uuid",
  "underlying_symbol": "SPY",
  "position_value": 10000000,
  "desired_protection_pct": 10,
  "expiration_months": 3
}
```

**Response:**
```json
{
  "strategy_id": "uuid",
  "cost_of_protection": 25000,
  "option_legs": [...],
  "max_loss": 1000000,
  "greeks": {...}
}
```

#### POST /risk/options/collar
Build collar strategy (put + covered call).

#### POST /risk/tail-risk/analyze
Perform tail risk analysis (VaR, CVaR).

**Response:**
```json
{
  "value_at_risk_95": 15000000,
  "conditional_var": 30000000,
  "stress_test_scenarios": [...],
  "recommended_hedges": [...]
}
```

#### POST /risk/drawdown/analyze
Analyze portfolio drawdowns.

---

### 7. Client Portal (8 endpoints)

#### POST /portal/messages/send
Send secure encrypted message.

**Request:**
```json
{
  "family_id": "uuid",
  "sender_id": "uuid",
  "recipient_id": "uuid",
  "subject": "Portfolio Review",
  "body": "encrypted_content",
  "priority": "HIGH"
}
```

#### GET /portal/messages/thread/{threadID}
Get message thread.

#### POST /portal/signatures/request
Create e-signature request.

**Request:**
```json
{
  "family_id": "uuid",
  "document_name": "IPS Amendment 2024",
  "document_type": "IPS",
  "document_url": "s3://...",
  "signers": [
    {
      "name": "John Doe",
      "email": "john@example.com",
      "signing_order": 1
    }
  ],
  "expiration_days": 30
}
```

#### POST /portal/signatures/{requestID}/sign
Sign document.

#### POST /portal/meetings/schedule
Schedule video meeting.

**Request:**
```json
{
  "family_id": "uuid",
  "advisor_id": "uuid",
  "meeting_type": "QUARTERLY_REVIEW",
  "title": "Q4 2024 Review",
  "scheduled_start": "2024-12-15T14:00:00Z",
  "duration_minutes": 60,
  "participants": [...]
}
```

#### DELETE /portal/meetings/{meetingID}
Cancel meeting.

#### GET /portal/activity/{familyID}
Get activity feed.

#### PUT /portal/notifications/preferences
Update notification preferences.

---

### 8. Compliance (10 endpoints)

#### POST /compliance/form-adv/generate
Generate Form ADV filing.

**Request:**
```json
{
  "firm_id": "uuid",
  "form_type": "ANNUAL_UPDATE",
  "effective_date": "2024-12-31"
}
```

**Response:**
```json
{
  "form_id": "uuid",
  "part1": {...},
  "part2": {...},
  "schedules": [...],
  "status": "DRAFT"
}
```

#### GET /compliance/form-adv/{formID}
Retrieve Form ADV.

#### POST /compliance/gips/check
Check GIPS compliance.

**Response:**
```json
{
  "compliance_status": "COMPLIANT",
  "composites": [...],
  "violations": []
}
```

#### GET /compliance/gips/{complianceID}
Get GIPS compliance report.

#### POST /compliance/surveillance/run
Run trade surveillance.

**Response:**
```json
{
  "alerts": [
    {
      "alert_type": "EXCESSIVE_TRADING",
      "severity": "MEDIUM",
      "description": "High turnover ratio"
    }
  ]
}
```

#### GET /compliance/surveillance/alerts
Get surveillance alerts.

#### POST /compliance/suitability/analyze
Analyze investment suitability.

**Response:**
```json
{
  "suitability_score": 85,
  "suitability_status": "SUITABLE",
  "violations": [],
  "recommendations": [...]
}
```

#### GET /compliance/suitability/{analysisID}
Get suitability analysis.

#### POST /compliance/audit/log
Log audit entry.

#### GET /compliance/audit/query
Query audit trail.

---

## Authentication

All API requests require a Bearer token:

```http
Authorization: Bearer {your_jwt_token}
```

To obtain a token:

```bash
POST /auth/login
{
  "email": "user@example.com",
  "password": "secure_password"
}
```

---

## Rate Limiting

- **Standard**: 1,000 requests/hour
- **Premium**: 10,000 requests/hour
- **Enterprise**: Unlimited

Rate limit headers:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 950
X-RateLimit-Reset: 1640995200
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "Missing required field: family_id",
    "details": {...}
  }
}
```

**Common Error Codes:**
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `429` - Rate Limit Exceeded
- `500` - Internal Server Error

---

## Webhooks

Subscribe to events:

```json
POST /webhooks/subscribe
{
  "url": "https://your-app.com/webhook",
  "events": [
    "message.sent",
    "signature.completed",
    "meeting.scheduled",
    "compliance.alert"
  ]
}
```

---

## Testing

**Sandbox Environment**: `https://sandbox-api.wealthvision.com/v1`

Use test credentials:
- Email: `test@wealthvision.com`
- API Key: `test_sk_1234567890`

---

## SDKs

Available SDKs:
- **Node.js**: `npm install @wealthvision/sdk`
- **Python**: `pip install wealthvision`
- **Go**: `go get github.com/wealthvision/go-sdk`
- **Ruby**: `gem install wealthvision`

Example (Node.js):
```javascript
const WealthVision = require('@wealthvision/sdk');
const client = new WealthVision('your_api_key');

const analysis = await client.tax.compareStates({
  familyId: 'uuid',
  currentState: 'CA',
  grossIncome: 1000000
});
```

---

## Support

- **Documentation**: https://docs.wealthvision.com
- **Status**: https://status.wealthvision.com
- **Email**: support@wealthvision.com
- **Slack**: https://wealthvision-community.slack.com
