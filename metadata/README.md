# WealthStream Metadata Package

Production-ready metadata definitions for WealthStream platform.

## Structure

```
metadata/
├── bp/                 # Business Processes
│   └── workflow_advisor_review.yml
├── bo/                 # Business Objects
│   └── advisor_queue_item.yml
├── persona/            # AI Personas
│   ├── persona_market_strategist.yml
│   └── persona_planning_specialist.yml
├── view/               # Feed Cards
│   ├── card_dividend_income.yml
│   ├── card_tax_loss_harvest.yml
│   └── card_rebalance_alert.yml
├── policy/             # Governance Policies
│   ├── policy_throttling.yml
│   ├── policy_disclosures.yml
│   └── policy_approval_thresholds.yml
├── experiment/         # Experimentation
│   └── card_outcome_bindings.yml
└── ci/                 # CI/CD Tests
    └── acceptance_tests.yml
```

## Validation

```bash
# Validate YAML syntax
find metadata -name "*.yml" -exec yamllint {} \;

# Validate with registry CLI (when available)
registry-cli validate metadata/

# Publish to staging
registry-cli publish metadata/ --env=staging
```

## Import to Registry

```bash
# Draft status
registry-cli publish metadata/ --status=draft

# Validate expressions
registry-cli publish metadata/ --validate-only

# Publish to production
registry-cli publish metadata/ --env=production
```

## CI/CD

See `metadata/ci/acceptance_tests.yml` for the complete test suite.

Run locally:
```bash
cd backend
go test ./pkg/policy/... -v
go test ./internal/feed/... -v
```
