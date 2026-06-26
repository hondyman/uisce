# 🏗️ Metadata-First, Event-Driven Rebalancing Platform Blueprint

## Executive Summary

This blueprint maps the **"Config over Code"** architectural vision to your existing semlayer infrastructure. Your codebase already has significant foundation pieces in place—this document identifies gaps and provides implementation paths for a Workday-style metadata-driven rebalancing platform.

---

## 📊 Current Architecture Assessment

### ✅ What You Already Have

| Component | Location | Status |
|-----------|----------|--------|
| **Rules Engine (CEL)** | `backend/internal/rules/engine.go` | ✅ Working |
| **Wash Sale Registry** | `backend/internal/rebalancer/tlh/wash_sale.go` | ✅ Working |
| **QP Optimizer** | `backend/internal/rebalancer/engine/optimizer.go` | ✅ Working (gonum) |
| **Tax-Aware Rebalance Workflow** | `backend/internal/temporal/workflows/rebalance_workflow.go` | ✅ Working |
| **Multi-Tenant Scope** | `agents.md`, `TenantContext` | ✅ Enforced |
| **RabbitMQ Event Bus** | `docker-compose.yml`, services | ✅ Running |
| **Temporal Workflows** | `backend/internal/temporal/` | ✅ Running |
| **Semantic Layer** | `backend/internal/metadata/` | ✅ Working |
| **Policy YAML Definitions** | `metadata/policy/` | ✅ Sample exists |
| **Business Object YAML** | `metadata/bo/` | ✅ Sample exists |

### 🔧 Gaps to Fill

| Component | Priority | Effort |
|-----------|----------|--------|
| **Rule Definition Language (RDL) Schema** | 🔴 Critical | 2-3 days |
| **Real-Time Drift Detection (Streaming)** | 🟡 High | 3-5 days |
| **Global Calendar Metadata** | 🟡 High | 2 days |
| **FIX Protocol Gateway** | 🟢 Medium | 5-7 days |
| **AI Drift Prediction Agent** | 🟢 Medium | 3-5 days |
| **Rule Builder UI** | 🟢 Medium | 5-7 days |

---

## 🎯 The "Config Over Code" Architecture

### 1. Rule Definition Language (RDL) Schema

Your existing `ComplianceRule` model in `backend/internal/rules/model.go` is a good start. Here's the enhanced metadata schema:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "RuleDefinition",
  "type": "object",
  "required": ["tenant_id", "rule_id", "type", "version", "active"],
  "properties": {
    "tenant_id": {
      "type": "string",
      "format": "uuid",
      "description": "Tenant isolation key"
    },
    "rule_id": {
      "type": "string",
      "pattern": "^[A-Z0-9_]+$",
      "description": "Unique rule identifier (e.g., UK_BED_AND_BREAKFAST_RULE)"
    },
    "type": {
      "type": "string",
      "enum": ["tax_constraint", "wash_sale", "esg_restriction", "drift_trigger", "cash_flow", "cppi_floor", "tlh_opportunity"]
    },
    "version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+\\.\\d+$"
    },
    "jurisdiction": {
      "type": "string",
      "description": "ISO 3166-1 alpha-2 country code (e.g., US, GB, DE)"
    },
    "parameters": {
      "type": "object",
      "additionalProperties": true,
      "description": "Rule-specific configuration"
    },
    "expression": {
      "type": "string",
      "description": "CEL expression for evaluation"
    },
    "active": {
      "type": "boolean"
    },
    "effective_from": {
      "type": "string",
      "format": "date"
    },
    "effective_to": {
      "type": "string",
      "format": "date"
    },
    "audit": {
      "type": "object",
      "properties": {
        "created_by": { "type": "string" },
        "created_at": { "type": "string", "format": "date-time" },
        "approved_by": { "type": "string" },
        "approved_at": { "type": "string", "format": "date-time" }
      }
    }
  }
}
```

### Example Rule Definitions

#### UK Bed & Breakfast Rule (Tax Constraint)
```json
{
  "tenant_id": "GLOBAL_WEALTH_CORP",
  "rule_id": "UK_BED_AND_BREAKFAST_RULE",
  "type": "tax_constraint",
  "version": "1.0.0",
  "jurisdiction": "GB",
  "parameters": {
    "days_restricted": 30,
    "asset_class": "equity",
    "applies_to": ["SELL_LOSS", "BUY_SAME_SECURITY"]
  },
  "expression": "daysSince(input.sale_date) < parameters.days_restricted && input.action == 'BUY' && input.security_id == input.previously_sold_security",
  "active": true,
  "effective_from": "1998-04-06"
}
```

#### US Wash Sale Rule
```json
{
  "tenant_id": "US_WEALTH_ADVISORS",
  "rule_id": "US_WASH_SALE_30_DAY",
  "type": "wash_sale",
  "version": "1.0.0",
  "jurisdiction": "US",
  "parameters": {
    "window_days_before": 30,
    "window_days_after": 30,
    "substantially_identical_threshold": 0.85
  },
  "expression": "input.is_loss_sale && (hasRecentPurchase(input.household_id, input.ticker, parameters.window_days_before) || hasFuturePurchase(input.household_id, input.ticker, parameters.window_days_after))",
  "active": true,
  "effective_from": "1921-01-01"
}
```

#### Tax-Loss Harvesting Opportunity
```json
{
  "tenant_id": "GLOBAL_WEALTH_CORP",
  "rule_id": "TLH_OPPORTUNITY_TRIGGER",
  "type": "tlh_opportunity",
  "version": "1.0.0",
  "parameters": {
    "min_loss_percentage": 10.0,
    "min_loss_amount_usd": 1000,
    "holding_period_days": 31,
    "substitute_correlation_min": 0.90
  },
  "expression": "input.unrealized_loss_pct >= parameters.min_loss_percentage && input.unrealized_loss_usd >= parameters.min_loss_amount_usd && input.days_held >= parameters.holding_period_days",
  "active": true
}
```

---

## 📅 Rebalancing Types Implementation Matrix

### Current vs. Target State

| Rebalancing Type | Current State | Target State | Implementation Path |
|------------------|---------------|--------------|---------------------|
| **Calendar-Based** | ❌ Manual cron | ✅ Metadata-driven global calendars | Add `global_calendars` table + Temporal scheduled workflows |
| **Drift/Tolerance** | ✅ `CheckDriftActivity` | ✅ Real-time streaming | Enhance with RabbitMQ consumer for price feeds |
| **Cash-Flow Driven** | ⚠️ Partial in workflow | ✅ Tax-aware cash allocation | Extend `TaxAwareOptimizeActivity` |
| **Tax-Loss Harvesting** | ✅ `wash_sale.go` | ✅ AI-Predictive Harvester | Add prediction model integration |
| **CPPI (Insurance)** | ❌ Not implemented | ✅ Personalized floor protection | New workflow + activity |

---

## 🔄 Event-Driven Trigger System

### Current Event Flow (RabbitMQ)
```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Price Feed     │────▶│   RabbitMQ      │────▶│  Drift Monitor  │
│  (External)     │     │  Exchange       │     │  (Consumer)     │
└─────────────────┘     └─────────────────┘     └────────┬────────┘
                                                         │
                                                         ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │  Temporal       │◀────│  Trigger        │
                        │  Workflow       │     │  Decision       │
                        └─────────────────┘     └─────────────────┘
```

### Enhanced Trigger Types

```go
// backend/internal/triggers/types.go

type TriggerType string

const (
    TriggerTime        TriggerType = "TIME"        // Calendar-based
    TriggerDrift       TriggerType = "DRIFT"       // Tolerance breach
    TriggerCashFlow    TriggerType = "CASH_FLOW"   // Deposit/Withdrawal
    TriggerMarket      TriggerType = "MARKET"      // Price movement
    TriggerRisk        TriggerType = "RISK"        // CPPI floor breach
    TriggerTLH         TriggerType = "TLH"         // Tax-loss opportunity
)

type TriggerDefinition struct {
    TenantID      string                 `json:"tenant_id"`
    TriggerID     string                 `json:"trigger_id"`
    Type          TriggerType            `json:"type"`
    Condition     string                 `json:"condition"`      // CEL expression
    Parameters    map[string]interface{} `json:"parameters"`
    WorkflowRef   string                 `json:"workflow_ref"`   // Temporal workflow to invoke
    Priority      int                    `json:"priority"`
    Active        bool                   `json:"active"`
}
```

---

## 🗃️ Database Schema Enhancements

### New Tables Required

```sql
-- Global Calendars (metadata-driven business days)
CREATE TABLE global_calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    calendar_code VARCHAR(50) NOT NULL,  -- e.g., 'US_NYSE', 'UK_LSE', 'IN_BSE'
    region VARCHAR(10) NOT NULL,         -- ISO 3166-1 alpha-2
    year INT NOT NULL,
    holidays JSONB NOT NULL,             -- Array of dates
    trading_hours JSONB,                 -- Start/end times per day
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, calendar_code, year)
);

-- Rule Definitions (the RDL store)
CREATE TABLE rule_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    rule_id VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    version VARCHAR(20) NOT NULL,
    jurisdiction VARCHAR(10),
    parameters JSONB NOT NULL DEFAULT '{}',
    expression TEXT NOT NULL,
    active BOOLEAN DEFAULT true,
    effective_from DATE,
    effective_to DATE,
    audit JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, rule_id, version)
);

-- Trigger Definitions
CREATE TABLE trigger_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    trigger_id VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    condition TEXT NOT NULL,
    parameters JSONB NOT NULL DEFAULT '{}',
    workflow_ref VARCHAR(200),
    priority INT DEFAULT 0,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, trigger_id)
);

-- CPPI Floor Configurations
CREATE TABLE cppi_floors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    portfolio_id UUID NOT NULL,
    floor_value_usd DECIMAL(18,2) NOT NULL,
    floor_type VARCHAR(20) DEFAULT 'ABSOLUTE',  -- ABSOLUTE, PERCENTAGE
    multiplier DECIMAL(5,2) DEFAULT 1.0,
    cushion_calculation VARCHAR(50) DEFAULT 'NAV_MINUS_FLOOR',
    last_rebalance_date DATE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for multi-tenant performance
CREATE INDEX idx_rule_definitions_tenant ON rule_definitions(tenant_id, active);
CREATE INDEX idx_trigger_definitions_tenant ON trigger_definitions(tenant_id, active);
CREATE INDEX idx_cppi_floors_tenant ON cppi_floors(tenant_id, portfolio_id);
```

---

## 🧠 The Semantic Layer ("Brain") Integration

Your existing `backend/internal/metadata/` provides the foundation. Enhance with:

### Business Object Model for Rebalancing

```yaml
# metadata/bo/rebalance_proposal.yml
id: bo_rebalance_proposal
type: BusinessObject
version: 1.0.0
name: RebalanceProposal
status: active
description: AI-generated rebalancing proposal awaiting advisor review

attributes:
  - name: proposal_id
    type: string
    required: true
    semantic_term: "Unique identifier for the proposal"
    
  - name: tenant_id
    type: string
    required: true
    
  - name: portfolio_id
    type: string
    required: true
    semantic_term: "Target portfolio for rebalancing"
    
  - name: trigger_type
    type: string
    enum: ["DRIFT", "CALENDAR", "CASH_FLOW", "TLH", "CPPI"]
    semantic_term: "What initiated this rebalance"
    
  - name: tracking_error_before
    type: number
    semantic_term: "Portfolio tracking error before rebalance"
    unit: "percentage"
    
  - name: tracking_error_after
    type: number
    semantic_term: "Projected tracking error after rebalance"
    unit: "percentage"
    
  - name: tax_impact_usd
    type: number
    semantic_term: "Net tax impact (negative = benefit)"
    unit: "USD"
    
  - name: trades
    type: array
    items: Trade
    semantic_term: "Proposed buy/sell orders"
    
  - name: monte_carlo
    type: object
    semantic_term: "Tax impact simulation results"
    
  - name: confidence
    type: number
    semantic_term: "AI confidence in proposal"
    range: [0, 1]
    
  - name: citations
    type: array
    items: Citation
    semantic_term: "Data sources for explainability"
    
  - name: status
    type: string
    enum: ["pending", "approved", "rejected", "executed", "expired"]

relationships:
  - target: Portfolio
    type: belongs_to
  - target: Advisor
    type: assigned_to
  - target: Trade
    type: has_many

policies:
  - "audit_required"
  - "approval_required_above_threshold"
```

### Semantic Query Examples

With this model, an advisor can query:

```sql
-- Natural Language: "Show me high-confidence tax-saving opportunities"
SELECT * FROM bo_rebalance_proposal 
WHERE confidence > 0.8 
  AND tax_impact_usd < -1000 
  AND status = 'pending'
ORDER BY tax_impact_usd ASC;

-- Natural Language: "Drift alerts for portfolios over $1M"
SELECT p.* FROM bo_rebalance_proposal p
JOIN portfolios pf ON p.portfolio_id = pf.id
WHERE p.trigger_type = 'DRIFT'
  AND pf.market_value > 1000000
  AND p.status = 'pending';
```

---

## 🔧 Implementation Roadmap

### Phase 1: Rule Definition Language (Week 1-2)
```
[ ] Create rule_definitions table migration
[ ] Implement RuleDefinitionService in backend/internal/rules/
[ ] Add CRUD API endpoints for rule management
[ ] Migrate existing wash sale logic to RDL format
[ ] Unit tests for CEL expression evaluation
```

### Phase 2: Global Calendars (Week 2)
```
[ ] Create global_calendars table
[ ] Implement CalendarService with holiday lookups
[ ] Add regional calendar data (US, UK, EU, APAC)
[ ] Integrate with Temporal scheduled workflows
[ ] Add calendar-based trigger support
```

### Phase 3: Real-Time Drift Detection (Week 3-4)
```
[ ] Create RabbitMQ consumer for price feed events
[ ] Implement streaming drift calculator
[ ] Add drift trigger definitions to database
[ ] Connect drift alerts to Temporal workflow initiation
[ ] Add drift prediction ML model (optional)
```

### Phase 4: CPPI Floor Protection (Week 4-5)
```
[ ] Create cppi_floors table
[ ] Implement CPPI calculation activities
[ ] Add CPPI workflow to Temporal
[ ] UI for personalized floor configuration
[ ] Backtesting for floor breach scenarios
```

### Phase 5: Rule Builder UI (Week 5-7)
```
[ ] Design rule builder React component
[ ] Implement drag-drop condition builder
[ ] Add expression validation (CEL syntax check)
[ ] Version control for rule changes
[ ] Approval workflow for rule activation
```

---

## 🤖 AI Integration Points

### 1. Drift Prediction Agent

```go
// backend/internal/ai/drift_predictor.go

type DriftPrediction struct {
    PortfolioID     string    `json:"portfolio_id"`
    PredictedDrift  float64   `json:"predicted_drift"`
    DaysUntilBreach int       `json:"days_until_breach"`
    Confidence      float64   `json:"confidence"`
    Factors         []Factor  `json:"contributing_factors"`
    Recommendation  string    `json:"recommendation"`
}

type Factor struct {
    Name       string  `json:"name"`
    Impact     float64 `json:"impact"`
    Direction  string  `json:"direction"` // "INCREASE" or "DECREASE"
}

// PredictDrift uses ML model to forecast when portfolio will breach threshold
func (a *DriftPredictorAgent) PredictDrift(ctx context.Context, portfolioID string, horizon int) (*DriftPrediction, error) {
    // 1. Fetch historical positions and returns
    // 2. Call ML model (could be Python service via gRPC)
    // 3. Return structured prediction
}
```

### 2. Natural Language Rule Generation

```go
// backend/internal/ai/rule_generator.go

type RuleGenerationRequest struct {
    TenantID    string `json:"tenant_id"`
    UserPrompt  string `json:"user_prompt"`  // "Never own more than 10% in Oil & Gas for ESG clients"
    RuleType    string `json:"rule_type"`    // Suggested or auto-detected
}

type RuleGenerationResponse struct {
    GeneratedRule   RuleDefinition `json:"generated_rule"`
    Explanation     string         `json:"explanation"`
    Confidence      float64        `json:"confidence"`
    RequiresReview  bool           `json:"requires_review"`
}

// GenerateRuleFromNL uses LLM to convert natural language to RDL
func (a *RuleGeneratorAgent) GenerateRuleFromNL(ctx context.Context, req RuleGenerationRequest) (*RuleGenerationResponse, error) {
    // 1. Send prompt to LLM with RDL schema context
    // 2. Parse LLM response into RuleDefinition struct
    // 3. Validate CEL expression syntax
    // 4. Return for human review
}
```

### 3. Compliance Sentinel Agent

```go
// backend/internal/ai/compliance_sentinel.go

type ComplianceSentinel struct {
    ragClient   RAGClient
    ruleEngine  *rules.GenericRuleEngine
    auditLogger AuditLogger
}

// ReviewTrades checks all proposed trades against latest regulations
func (s *ComplianceSentinel) ReviewTrades(ctx context.Context, trades []Trade, tenantID string) (*ComplianceReport, error) {
    // 1. Load tenant-specific rules from RDL store
    // 2. Query RAG for recent regulatory updates
    // 3. Evaluate each trade against all active rules
    // 4. Generate compliance report with citations
}
```

---

## 📈 Competitive Advantage Summary

| Capability | SS&C/Aladdin | Your Platform |
|------------|--------------|---------------|
| **Rule Changes** | Code deploy (weeks) | Metadata update (minutes) |
| **Multi-Jurisdiction** | Separate instances | Single codebase, tenant-scoped rules |
| **Drift Detection** | Nightly batch | Real-time streaming |
| **Tax Optimization** | Basic wash sale | AI-predictive harvesting |
| **Personalized Floors** | Not available | Per-client CPPI configuration |
| **Rule Creation** | Developer required | Natural language + UI |
| **Audit Trail** | Limited | Complete event sourcing |

---

## 🚀 Quick Start Commands

```bash
# Apply database migrations
cd backend
go run cmd/migrate/main.go up

# Start rule definition service
go run cmd/server/main.go --enable-rules-api

# Test wash sale rule evaluation
curl -X POST http://localhost:8080/api/rules/evaluate \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "rule_id": "US_WASH_SALE_30_DAY",
    "input": {
      "household_id": "HH001",
      "ticker": "AAPL",
      "is_loss_sale": true,
      "sale_date": "2024-01-15"
    }
  }'

# Trigger rebalance workflow
curl -X POST http://localhost:8080/api/rebalance/trigger \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "portfolio_id": "PORT001",
    "trigger_type": "DRIFT",
    "advisor_id": "ADV001"
  }'
```

---

## 📚 Related Documentation

- `REBALANCING_GUIDE.md` - Existing rebalancing implementation details
- `SEMANTIC_PLATFORM_BLUEPRINT.md` - Semantic layer architecture
- `TEMPORAL_QUICK_START.md` - Temporal workflow setup
- `VALIDATION_RULES_COMPLETE_GUIDE.md` - Validation engine reference
- `agents.md` - Tenant scoping requirements

---

## ✅ Success Checklist

- [ ] RDL Schema defined and validated
- [ ] Rule definitions table created with multi-tenant partitioning
- [ ] CEL expressions evaluated against input data
- [ ] Global calendars loaded for major markets
- [ ] Real-time drift detection via RabbitMQ
- [ ] CPPI floor protection workflow operational
- [ ] Rule Builder UI deployed
- [ ] AI Drift Predictor integrated
- [ ] Compliance Sentinel reviewing all trades
- [ ] Natural language rule generation available

---

*Generated: November 2024 | Platform: semlayer*
