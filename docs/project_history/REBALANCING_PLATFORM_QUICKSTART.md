# 🚀 Metadata-First Rebalancing Platform: Quick Start

## What We Built

This document summarizes the implementation of a **Workday-style metadata-driven rebalancing platform** that will enable you to beat competitors like SS&C, BlackRock Aladdin, and Envestnet through "Config over Code" architecture.

## Files Created

| File | Purpose |
|------|---------|
| `REBALANCING_PLATFORM_BLUEPRINT.md` | Complete architectural blueprint mapping vision to your existing infrastructure |
| `schemas/rdl/tax_loss_harvesting_rule.schema.json` | JSON Schema for Tax-Loss Harvesting rules with full wash sale & substitute asset config |
| `backend/migrations/20241126_001_metadata_first_rebalancing.sql` | Database migration for all new tables |
| `backend/internal/rdl/service.go` | Rule Definition Language service with CEL expression evaluation |
| `backend/internal/rdl/handler.go` | HTTP API handlers for CRUD and rule evaluation |

## Key Features Implemented

### 1. Rule Definition Language (RDL)
- **CEL Expression Engine**: Rules defined as expressions, not code
- **Multi-tenant Isolation**: Every rule scoped to `tenant_id`
- **Version Control**: Rules are versioned (`1.0.0`, `1.1.0`, etc.)
- **Jurisdiction Support**: Rules tagged by country (`US`, `GB`, `DE`, etc.)

### 2. Database Schema
```
┌─────────────────────────┐
│   rule_definitions      │  ← The "Secret Sauce"
├─────────────────────────┤
│   global_calendars      │  ← Business day metadata
├─────────────────────────┤
│   trigger_definitions   │  ← Event-driven configs
├─────────────────────────┤
│   cppi_floors           │  ← Personalized protection
├─────────────────────────┤
│   substitute_asset_mappings │  ← TLH replacements
├─────────────────────────┤
│   drift_snapshots       │  ← Historical tracking
└─────────────────────────┘
```

### 3. API Endpoints
```
GET    /api/rules                 → List all rules for tenant
GET    /api/rules/{ruleID}        → Get specific rule
POST   /api/rules                 → Create new rule
PUT    /api/rules/{ruleID}        → Update rule
DELETE /api/rules/{ruleID}        → Deactivate rule
POST   /api/rules/evaluate        → Evaluate single rule
POST   /api/rules/evaluate-batch  → Evaluate all rules of type
```

## Quick Start

### 1. Apply Database Migration
```bash
cd /Users/eganpj/GitHub/semlayer
psql postgres://postgres:postgres@localhost:5432/alpha -f backend/migrations/20241126_001_metadata_first_rebalancing.sql
```

### 2. Verify Tables Created
```sql
SELECT table_name FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name IN ('rule_definitions', 'global_calendars', 'trigger_definitions', 'cppi_floors');
```

### 3. Test Rule Evaluation
```bash
# Create a TLH rule (UK Bed & Breakfast)
curl -X POST http://localhost:8080/api/rules \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'${TENANT_ID}'",
    "rule_id": "UK_BED_BREAKFAST_V1",
    "type": "tax_loss_harvesting",
    "version": "1.0.0",
    "name": "UK Bed & Breakfast Rule",
    "jurisdiction": "GB",
    "parameters": {
      "min_loss_percentage": 10,
      "min_loss_amount_usd": 1000,
      "window_days_after": 30
    },
    "expression": "input.unrealized_loss_pct >= params.min_loss_percentage && input.days_held >= 0",
    "active": true
  }'

# Evaluate against a position
curl -X POST http://localhost:8080/api/rules/evaluate \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'${TENANT_ID}'",
    "rule_id": "UK_BED_BREAKFAST_V1",
    "input": {
      "portfolio_id": "PORT001",
      "ticker": "AAPL",
      "unrealized_loss_pct": 15.5,
      "unrealized_loss_usd": 2500,
      "days_held": 45,
      "account_type": "TAXABLE"
    }
  }'
```

## Architecture Comparison

| Feature | SS&C/Aladdin | Your Platform |
|---------|--------------|---------------|
| Rule Changes | Code deploy (weeks) | JSON update (minutes) |
| Multi-Jurisdiction | Separate instances | Single codebase, tenant-scoped |
| Wash Sale Logic | Hard-coded C++ | CEL expression, configurable window |
| New Tax Rule | 6-month project | 1 JSON file |

## What You Already Had (Leveraged)

- ✅ CEL Rules Engine (`backend/internal/rules/engine.go`)
- ✅ Wash Sale Registry (`backend/internal/rebalancer/tlh/wash_sale.go`)
- ✅ QP Optimizer (`backend/internal/rebalancer/engine/optimizer.go`)
- ✅ Temporal Workflows (`backend/internal/temporal/workflows/rebalance_workflow.go`)
- ✅ Redpanda (Kafka) Event Bus
- ✅ Multi-tenant Scope Enforcement

## Next Steps

1. **Phase 1 (This Week)**: Apply migration, integrate RDL handler into main router
2. **Phase 2 (Next Week)**: Real-time drift detection via RabbitMQ price feed consumer
3. **Phase 3 (Week 3)**: Global calendar API with holiday lookups
4. **Phase 4 (Week 4)**: Rule Builder UI component
5. **Phase 5 (Week 5)**: AI Drift Prediction integration

## Example: How UK → EU Rule Change Works

**Before (Competitors)**: 
- UK changes Bed & Breakfast rule from 30 to 31 days
- Aladdin deploys code patch to C++ engine
- 6-week release cycle, regression testing, rollout

**After (Your Platform)**:
```sql
UPDATE rule_definitions 
SET parameters = jsonb_set(parameters, '{window_days_after}', '31')
WHERE rule_id = 'UK_BED_BREAKFAST_V1' AND jurisdiction = 'GB';
```
- Done in seconds
- Only affects UK tenants
- Full audit trail in `audit` column

---

*This is the "Config over Code" advantage that will differentiate you from legacy wealth management platforms.*
