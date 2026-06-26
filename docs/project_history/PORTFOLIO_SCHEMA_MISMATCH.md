# Portfolio Service Schema Mismatch

## Issue Summary

The `portfolio-management/backend` service has Go models that do not match the actual database schema in the `alpha` database.

## Schema Comparison

### Actual Database Schema (alpha.portfolios)
```sql
CREATE TABLE portfolios (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    client_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL,
    benchmark VARCHAR(100),
    asset_allocation_targets JSONB,
    performance_metrics JSONB,
    advisor_discretion BOOLEAN DEFAULT TRUE,
    client_approval_required BOOLEAN DEFAULT TRUE,
    template_id UUID,
    custom_fields JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### Go Model (portfolio-management/backend/internal/backtest/models.go)
```go
type Portfolio struct {
    ID          uuid.UUID       `db:"id" json:"id"`
    UserID      uuid.UUID       `db:"user_id" json:"user_id"`      // ❌ DB has client_id
    Name        string          `db:"name" json:"name"`             // ❌ DB has type
    Description string          `db:"description" json:"description"` // ❌ DB has benchmark
    Currency    string          `db:"currency" json:"currency"`     // ❌ DB doesn't have this
    TotalValue  float64         `db:"total_value" json:"total_value"` // ❌ DB doesn't have this
    Holdings    []Holding       `json:"holdings"`
    Metadata    json.RawMessage `db:"metadata" json:"metadata"`     // ❌ DB has custom_fields
    CreatedAt   time.Time       `db:"created_at" json:"created_at"`
    UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}
```

### Mismatches
- ❌ Go model expects `user_id` → DB has `client_id` + `tenant_id`
- ❌ Go model expects `name` → DB has `type` (VARCHAR(50) for portfolio type)
- ❌ Go model expects `description` → DB has `benchmark` (VARCHAR(100) for benchmark symbol)
- ❌ Go model expects `currency` → DB has no currency field
- ❌ Go model expects `total_value` → DB has `performance_metrics` JSONB instead
- ❌ Go model has `Metadata` → DB has `custom_fields`
- ✅ DB requires `tenant_id` for multi-tenancy → Go model is missing this entirely

## Impact

### Current State
1. **SQL queries in service.go will FAIL** - They reference columns that don't exist:
   ```go
   // This query will fail:
   INSERT INTO portfolios (id, user_id, name, description, currency, total_value, ...)
   ```

2. **Hasura metadata is tracking the actual schema** - The metadata in `hasura/metadata/databases/alpha/tables/public_portfolios.yaml` correctly reflects the actual database.

3. **Service cannot function** - All CRUD operations will fail with "column does not exist" errors.

### Attempted Hasura Refactor
Started converting the backtest service to use Hasura GraphQL, but hit this mismatch:
- Added `HasuraClient` interface to Service
- Created `getPortfolioWithHasura()` method
- Attempted to map Hasura response to Go model → impossible due to schema mismatch

## Resolution Options

### Option 1: Update Go Models (Recommended)
Update `models.go` to match the actual database schema:

```go
type Portfolio struct {
    ID                      uuid.UUID       `db:"id" json:"id"`
    TenantID                uuid.UUID       `db:"tenant_id" json:"tenant_id"`  // NEW
    ClientID                uuid.UUID       `db:"client_id" json:"client_id"`  // Renamed from UserID
    Type                    string          `db:"type" json:"type"`             // Renamed from Name
    Benchmark               string          `db:"benchmark" json:"benchmark"`   // Renamed from Description
    AssetAllocationTargets  json.RawMessage `db:"asset_allocation_targets" json:"asset_allocation_targets"` // NEW
    PerformanceMetrics      json.RawMessage `db:"performance_metrics" json:"performance_metrics"` // NEW
    AdvisorDiscretion       bool            `db:"advisor_discretion" json:"advisor_discretion"` // NEW
    ClientApprovalRequired  bool            `db:"client_approval_required" json:"client_approval_required"` // NEW
    TemplateID              *uuid.UUID      `db:"template_id" json:"template_id"` // NEW
    CustomFields            json.RawMessage `db:"custom_fields" json:"custom_fields"` // Renamed from Metadata
    CreatedAt               time.Time       `db:"created_at" json:"created_at"`
    UpdatedAt               time.Time       `db:"updated_at" json:"updated_at"`
}
```

**Pros:**
- Aligns code with actual database
- Enables Hasura GraphQL conversion
- Supports multi-tenancy properly
- Code will actually work

**Cons:**
- Breaking change to service API
- Need to update all service methods
- May affect API consumers

### Option 2: Update Database Schema
Create migration to change database to match Go models:

```sql
ALTER TABLE portfolios RENAME COLUMN client_id TO user_id;
ALTER TABLE portfolios RENAME COLUMN type TO name;
ALTER TABLE portfolios RENAME COLUMN benchmark TO description;
ALTER TABLE portfolios ADD COLUMN currency VARCHAR(3) DEFAULT 'USD';
ALTER TABLE portfolios ADD COLUMN total_value DECIMAL(15,2);
ALTER TABLE portfolios RENAME COLUMN custom_fields TO metadata;
-- Would need to handle tenant_id somehow
```

**Pros:**
- Go code stays unchanged
- No API changes

**Cons:**
- Breaks multi-tenancy design (losing tenant_id)
- Loses semantic meaning (type → name)
- Other services may rely on current schema
- Not recommended - database schema looks intentional

### Option 3: Keep SQL for Now
Skip Hasura conversion for portfolio-management service until schema is aligned:

**Pros:**
- No immediate changes needed
- Can plan migration properly

**Cons:**
- Service still won't work due to schema mismatch
- Misses benefits of GraphQL

## Recommended Action Plan

1. **Immediate:** Update `models.go` to match actual database schema (Option 1)
2. **Then:** Complete Hasura GraphQL refactor with corrected models
3. **Test:** Verify all CRUD operations work with new models
4. **Document:** Update API docs to reflect new field names

## Files Affected

- `/Users/eganpj/GitHub/semlayer/portfolio-management/backend/internal/backtest/models.go` - Update Portfolio struct
- `/Users/eganpj/GitHub/semlayer/portfolio-management/backend/internal/backtest/service.go` - Update all SQL queries + Hasura methods
- Any API handlers that serialize Portfolio responses

## Current Status

- ✅ **RESOLVED** - Schema mismatch fixed (see PORTFOLIO_HASURA_REFACTOR_COMPLETE.md)
- ✅ **Models updated** - Portfolio struct now matches database schema
- ✅ **Hasura refactor complete** - GetPortfolio and CreatePortfolio use GraphQL
- ✅ **Tests passing** - 5/5 integration tests pass
- ✅ **Multi-tenancy support** - TenantID field added, properly scoped queries

See `PORTFOLIO_HASURA_REFACTOR_COMPLETE.md` for full details.
