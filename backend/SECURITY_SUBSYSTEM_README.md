# Business Object Security Subsystem - Complete Implementation

## Overview

Production-ready security subsystem for Business Objects with row-level and column-level access controls. Implements Kyvos-style governance with LDAP group binding, DSL-based row filtering, semantic term masking, and multi-scope enforcement (APIs, BI, AI).

## Architecture

### Components

1. **Backend (Go)**
   - REST API handlers (`internal/api/security_rules_handler.go`)
   - Service layer (`internal/services/access_rule_service.go`)
   - Repository layer (`internal/repository/access_rule_repository.go`)
   - DSL validator (`internal/services/dsl_validator.go`)
   - Impact analyzer (`internal/services/impact_analyzer.go`)
   - DSL parser (`internal/dsl/parser.go`)
   - Security enforcement (`internal/services/business_object_security.go`)

2. **Frontend (React/TypeScript)**
   - Access rules list page
   - Access rule editor with tabs (Edit, Preview, Test)
   - Rule preview component
   - Rule test component
   - API client (`api/accessRules.ts`)

3. **Temporal Workflows**
   - Rule promotion workflow (`workflows/access_rule_promotion.go`)
   - Activities for validation, testing, and deployment (`internal/activities/access_rule_activities.go`)

4. **Testing**
   - DSL validation tests (`internal/services/dsl_validator_test.go`)
   - Rule composition tests (`internal/services/rule_composition_test.go`)

## API Endpoints

All endpoints are documented in [api/openapi-security-rules.yaml](api/openapi-security-rules.yaml).

### Core Endpoints

- `GET /security/rules` - List access rules with filters
- `POST /security/rules` - Create new rule
- `GET /security/rules/{ruleId}` - Get single rule
- `PUT /security/rules/{ruleId}` - Update rule
- `POST /security/rules/validate` - Validate DSL expression
- `GET /security/rules/{ruleId}/impact` - Analyze rule impact

### Request Examples

**Create Rule:**
```bash
curl -X POST http://localhost:8080/security/rules \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-123",
    "businessObjectId": "bo:portfolio",
    "groupDn": "cn=emea-analysts,ou=groups,dc=example,dc=com",
    "accessLevel": "READ",
    "status": "DRAFT",
    "rowFilterDsl": "region = '\''EMEA'\'' AND client_type != '\''internal'\''",
    "columnMasks": [
      {"semanticTermId": "term:client_ssn", "maskType": "HIDE"},
      {"semanticTermId": "term:client_email", "maskType": "MASK"}
    ],
    "scope": {
      "appliesToApis": true,
      "appliesToBi": true,
      "appliesToAi": true
    }
  }'
```

**Validate DSL:**
```bash
curl -X POST http://localhost:8080/security/rules/validate \
  -H "Content-Type: application/json" \
  -d '{
    "businessObjectId": "bo:portfolio",
    "rowFilterDsl": "region = '\''EMEA'\'' AND status = '\''active'\''"
  }'
```

**Get Impact:**
```bash
curl http://localhost:8080/security/rules/{ruleId}/impact
```

## DSL Language

### Supported Operators

- **Comparison:** `=`, `!=`, `>`, `<`, `>=`, `<=`
- **Logical:** `AND`, `OR`, `NOT`
- **Special:** `IN (...)`, `LIKE`, `IS NULL`, `IS NOT NULL`
- **Grouping:** Parentheses `(...)`

### Examples

```sql
-- Simple equality
region = 'EMEA'

-- Complex conditions
region = 'EMEA' AND client_type != 'internal' AND status = 'active'

-- OR conditions
region = 'EMEA' OR region = 'APAC'

-- IN operator
region IN ('EMEA', 'APAC', 'LATAM')

-- NULL checks
deleted_at IS NULL

-- LIKE pattern matching
name LIKE '%Smith%'

-- Complex nested
(region = 'EMEA' OR region = 'APAC') AND client_type != 'internal'
```

### Field Validation

The DSL validator checks that all referenced fields exist in the business object's semantic terms. This prevents typos and ensures rules only reference valid data.

## Rule Composition

When a user belongs to multiple groups with rules for the same business object:

1. **Row Predicates:** Combined with `OR` (union of permitted rows)
2. **Access Level:** Highest level wins (`WRITE > READ > NONE`)
3. **Column Masks:** Most restrictive wins (`HIDE > MASK > NONE`)

### Example

User in groups: `developers`, `emea-analysts`

**Rule 1 (developers):**
- Row filter: `region = 'EMEA'`
- Access level: `READ`
- Masks: `client_email = MASK`

**Rule 2 (emea-analysts):**
- Row filter: `region = 'APAC'`
- Access level: `WRITE`
- Masks: `client_email = HIDE`

**Effective Decision:**
- Row filter: `(region = 'EMEA') OR (region = 'APAC')`
- Access level: `WRITE`
- Masks: `client_email = HIDE`

## Scope Controls

Rules can be scoped to specific consumption patterns:

- **APIs:** REST/GraphQL queries
- **BI:** Tableau, PowerBI, Looker
- **AI:** RAG queries, model training

This enables granular control over where security applies.

## Temporal Workflow: Rule Promotion

For production deployments, rules go through a promotion workflow:

1. **Load Rule** - Fetch from source environment
2. **Validate DSL** - Parse and check syntax
3. **Impact Analysis** - Identify affected artifacts
4. **Security Tests** - Run integration tests
5. **Approval** - Wait for manual approval (signal)
6. **Promote** - Deploy to target environment
7. **Audit & Cache** - Log change, invalidate caches

### Start Workflow

```go
params := workflows.PromoteRuleParams{
    RuleID:      "rule-123",
    TargetEnv:   "prod",
    RequestedBy: "admin@example.com",
}

workflowOptions := client.StartWorkflowOptions{
    ID:        "promote-rule-123",
    TaskQueue: "security-workflows",
}

we, err := temporalClient.ExecuteWorkflow(context.Background(), workflowOptions, workflows.PromoteAccessRuleWorkflow, params)
```

### Approve Promotion

```go
err := temporalClient.SignalWorkflow(context.Background(), "promote-rule-123", "", "approval-decision", true)
```

## Frontend UI

### Access Rules List Page

- Displays all rules in a table
- Filters by BO, status, group
- Create, edit, view impact actions

### Access Rule Editor

**Edit Tab:**
- Form for rule properties
- DSL input with validation
- Column mask table
- Scope toggles

**Preview Tab:**
- Effective configuration
- Row predicate display
- Column mask summary
- Impact analysis (terms, APIs, BI, AI)

**Test Tab:**
- Test against specific group/user
- Shows effective SQL predicate
- Lists masked/unmasked fields
- Sample result preview

## Database Schema

```sql
CREATE TABLE access_rule (
    id                   TEXT PRIMARY KEY,
    tenant_id            TEXT NOT NULL,
    business_object_id   TEXT NOT NULL,
    group_dn             TEXT NOT NULL,
    access_level         TEXT NOT NULL, -- NONE, READ, WRITE
    status               TEXT NOT NULL, -- DRAFT, REVIEW, APPROVED, DEPRECATED
    row_filter_dsl       TEXT,
    column_masks         JSONB,
    applies_to_apis      BOOLEAN DEFAULT TRUE,
    applies_to_bi        BOOLEAN DEFAULT TRUE,
    applies_to_ai        BOOLEAN DEFAULT TRUE,
    created_by           TEXT,
    created_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_by           TEXT,
    updated_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    version              INTEGER NOT NULL DEFAULT 1,
    description          TEXT
);

CREATE INDEX idx_access_rule_tenant ON access_rule(tenant_id);
CREATE INDEX idx_access_rule_bo ON access_rule(business_object_id);
CREATE INDEX idx_access_rule_group ON access_rule(group_dn);
CREATE INDEX idx_access_rule_status ON access_rule(status);
```

## Performance Optimization

### Caching

**Cache Key:** `(tenantId, businessObjectId, groupSetHash)`

**Cache Value:**
```json
{
  "rowPredicateSql": "(region = 'EMEA') OR (region = 'APAC')",
  "columnMasks": {
    "term:client_email": "HIDE",
    "term:client_ssn": "HIDE"
  },
  "accessLevel": "WRITE"
}
```

**TTL:** 5-15 minutes

**Invalidation:** On rule create/update/promote

### Predicate Pushdown

1. **SQL Injection:** Inject row predicates into `WHERE` clauses
2. **RLS:** Use Postgres Row-Level Security for hot tables
3. **Materialized Views:** Pre-compute filtered datasets per tenant/group

### Metrics

Monitor these:
- `security_rule_resolution_ms` - Time to resolve rules
- `security_sql_rewrite_ms` - Time to inject predicates
- `security_cache_hit_ratio` - Cache effectiveness
- `security_filtered_rows` - Rows filtered by predicates

## Testing

### Run DSL Validator Tests

```bash
cd backend
go test ./internal/services -run TestDslValidation -v
```

### Run Rule Composition Tests

```bash
go test ./internal/services -run TestRuleComposition -v
```

### Run All Security Tests

```bash
go test ./internal/services ./internal/dsl ./internal/repository -v
```

## Integration with Business Object Service

The `business_object_service.go` enforces access decisions on instance CRUD:

1. Extract principal from context
2. Resolve rules for principal + BO
3. Compose access decision
4. Inject row predicates into queries
5. Apply column masks to results

### Example (Pseudocode)

```go
func (s *BusinessObjectService) GetInstance(ctx context.Context, boID, instanceID string) (*Instance, error) {
    // 1. Get principal
    principal := services.PrincipalFromContext(ctx)
    
    // 2. Resolve access decision
    decision, err := s.securityRepo.ResolveAccessDecision(ctx, principal, boID)
    if err != nil {
        return nil, err
    }
    
    if decision.AccessLevel == services.AccessLevelNone {
        return nil, services.ErrForbidden
    }
    
    // 3. Build query with row predicate
    query := buildQuery(boID, instanceID, decision.RowPredicate)
    
    // 4. Execute query
    instance, err := s.repo.Query(ctx, query)
    if err != nil {
        return nil, err
    }
    
    // 5. Apply column masks
    maskedInstance := applyMasks(instance, decision.ColumnMasks)
    
    return maskedInstance, nil
}
```

## Deployment Checklist

- [ ] Deploy database schema and indexes
- [ ] Seed default rules (see `migrations/misc/seed_access_rules.sql`)
- [ ] Configure cache (Redis recommended)
- [ ] Wire security handlers into API router
- [ ] Deploy Temporal workers for promotion workflow
- [ ] Configure LDAP/AD integration for group resolution
- [ ] Set up metrics and dashboards
- [ ] Run integration tests
- [ ] Train security admins on UI
- [ ] Document group naming conventions

## Example: Complete Flow

1. **Security admin creates rule:**
   - Opens UI at `/security/access-rules/new`
   - Fills form: Tenant, BO, Group, DSL, masks
   - Clicks "Validate" to check DSL syntax
   - Switches to "Preview" tab to see effective config
   - Switches to "Test" tab, enters test group
   - Clicks "Save" → Rule created with status `DRAFT`

2. **Rule review:**
   - Reviewer opens rule, sees impact analysis
   - Checks which APIs, BI, AI artifacts affected
   - Updates status to `REVIEW`

3. **Approval & promotion:**
   - Admin starts Temporal workflow to promote to `prod`
   - Workflow runs validation, impact, tests
   - Waits for approval signal
   - Admin sends approval signal
   - Workflow promotes rule, updates status to `APPROVED`
   - Cache invalidated

4. **Runtime enforcement:**
   - User authenticates, groups resolved from LDAP
   - User queries business object via API
   - Backend resolves rules for user's groups + BO
   - Composes decision (row predicate + masks + level)
   - Injects predicate into SQL
   - Applies masks to result
   - Returns secured data to user

## Future Enhancements

- **Time-based rules:** Enable/disable rules on schedule
- **Attribute-based rules:** Reference user attributes in DSL (e.g., `$user.department`)
- **Data tagging:** Auto-apply rules based on data sensitivity tags
- **Rule templates:** Pre-defined rule patterns for common scenarios
- **Audit dashboard:** Visualize rule changes and access patterns
- **Fine-grained BI integration:** Per-dashboard rule overrides
- **AI governance:** Token-level masking for LLM queries

## Support

For questions or issues:
- Slack: #security-platform
- Email: security-team@example.com
- Docs: https://docs.semlayer.com/security

## License

Copyright © 2026 SemLayer. All rights reserved.
