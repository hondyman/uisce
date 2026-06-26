# WealthVision Database Schema Reference

## Overview

WealthVision uses PostgreSQL 15+ with **27 tables** organized into 4 main categories:
1. **Phase 1 Features** (16 tables)
2. **Risk Management** (5 tables)
3. **Client Portal** (5 tables)
4. **Compliance** (6 tables)

---

## Phase 1: Core WealthVision Features (16 tables)

### Tax Optimization

#### `tax_strategies`
Stores tax optimization strategies and recommendations.

| Column | Type | Description |
|--------|------|-------------|
| strategy_id | UUID | Primary key |
| family_id | UUID | Foreign key to family_offices |
| strategy_type | VARCHAR(100) | STATE_RESIDENCY, NIIT, CHARITABLE_BUNCHING |
| recommendations | JSONB | Strategy recommendations |
| estimated_savings | NUMERIC(18,2) | Annual tax savings estimate |
| created_at | TIMESTAMP | Creation timestamp |

**Indexes:**
- `idx_tax_family` on `family_id`
- `idx_tax_type` on `strategy_type`

#### `state_residency_comparisons`
Multi-state tax comparison analyses.

#### `niit_calculations`
Net Investment Income Tax (3.8%) calculations.

#### `charitable_bunching_analyses`
Charitable giving bunching strategies.

### Multi-Generational Planning

#### `dynasty_trust_simulations`
Dynasty trust projections across 3+ generations.

| Column | Type | Description |
|--------|------|-------------|
| simulation_id | UUID | Primary key |
| family_id | UUID | Foreign key |
| initial_funding | NUMERIC(18,2) | Starting trust value |
| generations | INT | Number of generations |
| projections | JSONB | Generation-by-generation projections |
| total_wealth_impact | NUMERIC(18,2) | Total wealth created |

#### `education_529_plans`
529 education plan optimizations.

#### `legacy_impact_calculations`
Multi-generational philanthropic impact.

### Alternative Investments

#### `private_equity_investments`
PE cash flow modeling (J-curve, IRR, MOIC).

| Column | Type | Description |
|--------|------|-------------|
| investment_id | UUID | Primary key |
| family_id | UUID | Foreign key |
| commitment_amount | NUMERIC(18,2) | Total commitment |
| capital_calls | JSONB | Array of capital calls |
| distributions | JSONB | Array of distributions |
| irr | NUMERIC(8,4) | Internal rate of return |
| moic | NUMERIC(8,4) | Multiple on invested capital |
| dpi | NUMERIC(8,4) | Distributions to paid-in |
| rvpi | NUMERIC(8,4) | Residual value to paid-in |
| tvpi | NUMERIC(8,4) | Total value to paid-in |

#### `venture_capital_investments`
VC cap table and exit scenarios.

#### `art_collectibles`
Art & collectibles tracking.

#### `real_estate_syndications`
RE syndication and 1031 exchange tracking.

### AI & Machine Learning

#### `churn_predictions`
ML-powered client churn predictions.

| Column | Type | Description |
|--------|------|-------------|
| prediction_id | UUID | Primary key |
| family_id | UUID | Foreign key |
| churn_probability | NUMERIC(5,4) | 0-1 probability score |
| risk_level | VARCHAR(20) | LOW, MEDIUM, HIGH, CRITICAL |
| key_factors | JSONB | Contributing factors |
| recommended_actions | JSONB | Retention strategies |

#### `meeting_preparations`
AI-generated meeting prep materials.

### ESG & Impact

#### `carbon_footprint_calculations`
Portfolio carbon footprint (Scope 1/2/3).

| Column | Type | Description |
|--------|------|-------------|
| footprint_id | UUID | Primary key |
| portfolio_id | UUID | Portfolio identifier |
| total_emissions_tons_co2 | NUMERIC(12,2) | Total CO2 tons |
| scope1_emissions | NUMERIC(12,2) | Direct emissions |
| scope2_emissions | NUMERIC(12,2) | Indirect emissions |
| scope3_emissions | NUMERIC(12,2) | Supply chain emissions |
| reduction_strategies | JSONB | Mitigation strategies |

#### `esg_portfolio_scores`
ESG scoring (MSCI, Sustainalytics, SDG).

#### `impact_investments`
Impact investing SROI tracking.

---

## Phase 2: Risk Management (5 tables)

### `options_overlay_strategies`
Options overlay strategies for downside protection.

| Column | Type | Description |
|--------|------|-------------|
| strategy_id | UUID | Primary key |
| portfolio_id | UUID | Portfolio identifier |
| strategy_type | VARCHAR(50) | PROTECTIVE_PUT, COLLAR, COVERED_CALL |
| cost_of_protection | NUMERIC(18,2) | Strategy cost |
| max_loss | NUMERIC(18,2) | Maximum loss |
| max_gain | NUMERIC(18,2) | Maximum gain |
| expiration | TIMESTAMP | Option expiration |

**Indexes:**
- `idx_options_portfolio` on `portfolio_id`
- `idx_options_expiration` on `expiration`

### `option_legs`
Individual option positions within strategies.

### `tail_risk_analyses`
VaR, CVaR, and stress testing.

| Column | Type | Description |
|--------|------|-------------|
| analysis_id | UUID | Primary key |
| value_at_risk_95 | NUMERIC(18,2) | 95% confidence VaR |
| value_at_risk_99 | NUMERIC(18,2) | 99% confidence VaR |
| conditional_var | NUMERIC(18,2) | Expected shortfall |
| stress_test_scenarios | JSONB | Historical stress tests |
| recommended_hedges | JSONB | Hedge recommendations |

### `drawdown_analyses`
Portfolio drawdown tracking.

### `risk_alerts`
Automated risk threshold alerts.

---

## Client Portal (5 tables)

### `client_messages`
Secure encrypted messaging.

| Column | Type | Description |
|--------|------|-------------|
| message_id | UUID | Primary key |
| thread_id | UUID | Conversation thread |
| family_id | UUID | Foreign key |
| sender_id | VARCHAR(255) | Sender identifier |
| sender_type | VARCHAR(20) | CLIENT, ADVISOR, SYSTEM |
| body | TEXT | Encrypted message body |
| encrypted | BOOLEAN | Encryption flag |
| read | BOOLEAN | Read status |
| priority | VARCHAR(20) | Message priority |

**Indexes:**
- `idx_messages_thread` on `thread_id`
- `idx_messages_recipient` on `recipient_id, read`

### `message_attachments`
File attachments for messages.

### `signature_requests`
E-signature document requests (DocuSign-style).

| Column | Type | Description |
|--------|------|-------------|
| request_id | UUID | Primary key |
| document_name | VARCHAR(500) | Document name |
| document_type | VARCHAR(100) | IPS, ACCOUNT_AGREEMENT, etc. |
| status | VARCHAR(20) | PENDING, SIGNED, REJECTED |
| expires_at | TIMESTAMP | Expiration timestamp |

### `signature_signers`
Individual signers for e-signature requests.

### `video_meetings`
Video meeting scheduling (Zoom/Teams).

| Column | Type | Description |
|--------|------|-------------|
| meeting_id | UUID | Primary key |
| meeting_type | VARCHAR(50) | QUARTERLY_REVIEW, ANNUAL_PLANNING |
| scheduled_start | TIMESTAMP | Start time |
| video_provider | VARCHAR(20) | ZOOM, TEAMS, GOOGLE_MEET |
| meeting_url | TEXT | Video conference URL |

### `meeting_participants`
Meeting participants and RSVP status.

### `activity_events`
Client portal activity feed.

---

## Compliance (6 tables)

### `form_adv_filings`
SEC Form ADV filings (Part 1 & 2).

| Column | Type | Description |
|--------|------|-------------|
| form_id | UUID | Primary key |
| firm_id | VARCHAR(255) | Firm identifier |
| form_type | VARCHAR(50) | INITIAL, AMENDMENT, ANNUAL_UPDATE |
| part1_data | JSONB | Form ADV Part 1 |
| part2_data | JSONB | Form ADV Part 2 (Brochure) |
| schedules | JSONB | Schedules A, B, C, D |
| iard_number | VARCHAR(50) | IARD registration number |

**Indexes:**
- `idx_form_adv_firm` on `firm_id`
- `idx_form_adv_filing_date` on `filing_date DESC`

### `gips_compliance_reports`
GIPS compliance verification.

| Column | Type | Description |
|--------|------|-------------|
| compliance_id | UUID | Primary key |
| compliance_status | VARCHAR(50) | COMPLIANT, NON_COMPLIANT |
| composites | JSONB | GIPS composites |
| violations | JSONB | Compliance violations |

### `trade_surveillance_alerts`
Trade compliance monitoring.

| Column | Type | Description |
|--------|------|-------------|
| alert_id | UUID | Primary key |
| alert_type | VARCHAR(100) | FRONT_RUNNING, MARKET_MANIPULATION |
| severity | VARCHAR(20) | LOW, MEDIUM, HIGH, CRITICAL |
| trade_details | JSONB | Flagged trade data |
| status | VARCHAR(50) | OPEN, INVESTIGATING, RESOLVED |

**Indexes:**
- `idx_surveillance_severity` on `severity`
- `idx_surveillance_status` on `status`

### `suitability_analyses`
Investment suitability assessments.

| Column | Type | Description |
|--------|------|-------------|
| analysis_id | UUID | Primary key |
| client_profile | JSONB | Risk tolerance, objectives |
| portfolio_allocation | JSONB | Current allocation |
| suitability_score | NUMERIC(5,2) | 0-100 score |
| suitability_status | VARCHAR(20) | SUITABLE, WARNING, UNSUITABLE |
| violations | JSONB | Rule violations |

### `audit_trail`
Complete regulatory audit log.

| Column | Type | Description |
|--------|------|-------------|
| entry_id | UUID | Primary key |
| timestamp | TIMESTAMP | Action timestamp |
| user_id | VARCHAR(255) | User who performed action |
| action | VARCHAR(50) | CREATE, UPDATE, DELETE, VIEW, EXPORT |
| resource_type | VARCHAR(100) | ACCOUNT, TRADE, REPORT |
| resource_id | VARCHAR(255) | Resource identifier |
| ip_address | VARCHAR(45) | Client IP |
| changes | JSONB | Before/after values |
| success | BOOLEAN | Success flag |

**Indexes:**
- `idx_audit_user` on `user_id`
- `idx_audit_timestamp` on `timestamp DESC`
- `idx_audit_resource` on `resource_type, resource_id`

---

## Database Performance

### Connection Pooling
```sql
-- Recommended settings
max_connections = 200
shared_buffers = 4GB
effective_cache_size = 12GB
work_mem = 64MB
```

### Key Indexes
All tables include optimized indexes on:
- Foreign keys (family_id, portfolio_id)
- Timestamps (created_at, updated_at)
- Status fields
- Search columns

### Partitioning Strategy
For high-volume tables:
- `audit_trail`: Partition by month
- `activity_events`: Partition by month
- `client_messages`: Partition by family_id (hash)

### Backup Strategy
- **Full backup**: Daily at 2 AM UTC
- **Incremental**: Every 6 hours
- **Point-in-time recovery**: Enabled (7 days)
- **Retention**: 30 days full, 90 days archived

---

## Migrations

All migrations located in `/backend/migrations/`:

| Migration | Tables Created | Description |
|-----------|----------------|-------------|
| 0009_wealthvision_phase1.up.sql | 16 | Tax, multi-gen, alt inv, AI, ESG |
| 0010_risk_management.up.sql | 5 | Options, tail risk, drawdown |
| 0011_client_portal_compliance.up.sql | 11 | Portal + compliance |

**Total Tables**: 27  
**Total Lines**: 1,000+ SQL

---

## Data Models

### JSONB Field Schemas

#### `tax_strategies.recommendations`
```json
{
  "strategies": [
    {
      "type": "STATE_MOVE",
      "from_state": "CA",
      "to_state": "FL",
      "annual_savings": 133000,
      "implementation_steps": [...]
    }
  ]
}
```

#### `dynasty_trust_simulations.projections`
```json
{
  "generations": [
    {
      "generation": 1,
      "beneficiaries": 3,
      "trust_value": 15000000,
      "distributions": 500000
    }
  ]
}
```

#### `carbon_footprint_calculations.reduction_strategies`
```json
{
  "strategies": [
    {
      "strategy": "Green bond allocation",
      "potential_reduction_pct": 15,
      "cost": 0
    }
  ]
}
```

---

## Security

### Encryption
- **At Rest**: AES-256 encryption for all tables
- **In Transit**: TLS 1.3 for all connections
- **Field-Level**: Encrypted columns for messages, signatures

### Access Control
- **Row-Level Security**: Enabled on all family_id columns
- **Least Privilege**: Service accounts with minimal permissions
- **Audit**: All DDL/DML logged to audit_trail

### Compliance
- **GDPR**: Right to erasure implemented
- **SOC 2**: Audit controls in place
- **PCI DSS**: N/A (no payment card data)

---

## Query Examples

### Get client's active tax strategies
```sql
SELECT * FROM tax_strategies
WHERE family_id = $1
AND status = 'ACTIVE'
ORDER BY created_at DESC;
```

### Calculate portfolio carbon footprint
```sql
SELECT 
  portfolio_id,
  SUM(total_emissions_tons_co2) as total_emissions
FROM carbon_footprint_calculations
WHERE analysis_date >= NOW() - INTERVAL '1 year'
GROUP BY portfolio_id;
```

### Find high-risk churn clients
```sql
SELECT 
  f.family_name,
  c.churn_probability,
  c.key_factors
FROM churn_predictions c
JOIN family_offices f ON c.family_id = f.family_id
WHERE c.risk_level IN ('HIGH', 'CRITICAL')
ORDER BY c.churn_probability DESC;
```

---

## Support

For schema questions:
- **Documentation**: https://docs.wealthvision.com/schema
- **Database Admin**: dba@wealthvision.com
