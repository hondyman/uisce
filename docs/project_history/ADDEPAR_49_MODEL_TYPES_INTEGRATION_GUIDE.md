# Addepar 49 Model Types Integration Guide

## Overview

This guide documents the integration of all 49 Addepar business entity model types into your Semlayer platform. The implementation provides:

- **Complete Model Types**: All 49 Addepar asset, entity, and container types
- **Hierarchical Relationships**: Explicit parent-child rules enforced via `entity_hierarchy_rules` table
- **Low-Code Extensibility**: JSON import API and admin UI for adding custom types
- **GraphQL API**: Production-ready queries for ownership traversal, filtering, and aggregation
- **ABAC Integration**: Attribute-based access control at entity and position level
- **Temporal Queries**: "As-of" date support for historical portfolio snapshots

## Hierarchical Model Types Map

### Level 0: Top-Level Roots (No parents)
- `household` – Primary portfolio container

### Level 1: Primary Containers (Household children)
- `person_node` – Individual client/person
- `prospect` – Prospective client
- `trust` – Legal trust entity
- `managed_partnership` – Managed fund structure
- `holding_company` – Corporate holding structure
- `manager` – Portfolio manager
- `vehicle` – General-purpose container

### Level 2: Sub-Containers (Account/Portfolio level)
- `financial_account` – Bank/brokerage account
- `sleeve` – Tactical sub-portfolio
- `fund` – Private fund structure
- `hedge_fund` – Hedge fund vehicle
- `private_equity_fund` – PE fund vehicle

### Level 3: Assets & Leaf Nodes (Holdings)
- **Fixed Income**: `bond`, `certificate_of_deposit`, `cmo`, `convertible_note`
- **Equities**: `stock`, `preferred_stock`
- **Funds**: `etf`, `etn`, `closed_end_fund`, `money_market_fund`, `mutual_fund`, `reit`, `uit`, `master_limited_partnership`
- **Alternatives**: `private_investment`, `venture_capital`, `real_estate`, `annuity`
- **Derivatives**: `option`, `futures_contract`, `forward_contract`, `warrant`
- **Collectibles**: `art`, `car`, `collectible`
- **Digital**: `digital_asset` (crypto)
- **Utilities**: `cash`, `historical_segment`, `loan`, `generic_asset`, `unknown_security`

## Database Schema

### 1. `model_type_definitions` (Enhanced)

```sql
CREATE TABLE model_type_definitions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    model_type VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255),
    ownership_type VARCHAR(50),
    description TEXT,
    is_hierarchical BOOLEAN DEFAULT false,
    hierarchy_level INTEGER,
    created_by UUID,
    UNIQUE(tenant_id, model_type),
    FOREIGN KEY (tenant_id) REFERENCES organizations(id)
);
```

**hierarchy_level** values:
- 0: Top-level (household)
- 1: Primary containers
- 2: Sub-containers (accounts, sleeves)
- 3: Assets/leaf nodes

### 2. `entity_hierarchy_rules` (New)

Enforces allowed parent → child relationships:

```sql
CREATE TABLE entity_hierarchy_rules (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    parent_model_type VARCHAR(255) NOT NULL,
    child_model_type VARCHAR(255) NOT NULL,
    allowed BOOLEAN DEFAULT true,
    ownership_types VARCHAR(100)[] DEFAULT ARRAY['PERCENT_BASED', 'SHARE_BASED', 'VALUE_BASED'],
    max_children INTEGER,
    min_children INTEGER,
    is_exclusive BOOLEAN DEFAULT false,
    description TEXT,
    UNIQUE(tenant_id, parent_model_type, child_model_type),
    FOREIGN KEY (tenant_id) REFERENCES organizations(id)
);
```

**Example Rules**:
- `household` → `person_node` (PERCENT_BASED)
- `financial_account` → `stock` (SHARE_BASED)
- `trust` → `real_estate` (VALUE_BASED)

### 3. `model_type_hierarchy_attributes` (New)

Defines suggested attributes per type (for low-code form generation):

```sql
CREATE TABLE model_type_hierarchy_attributes (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    model_type VARCHAR(255) NOT NULL,
    attribute_key VARCHAR(255) NOT NULL,
    attribute_type VARCHAR(50),  -- 'string', 'date', 'numeric', 'boolean', 'enum'
    is_required BOOLEAN DEFAULT false,
    is_searchable BOOLEAN DEFAULT false,
    priority INTEGER,
    description TEXT,
    validation_rule JSONB,
    UNIQUE(tenant_id, model_type, attribute_key),
    FOREIGN KEY (tenant_id) REFERENCES organizations(id)
);
```

## Seeding the Data

### Migration Script

Run the provided migration to seed all 49 model types:

```bash
psql postgres://postgres:postgres@localhost:5432/wealth_app < migrations/addepar_model_types_49_extended.sql
```

**What this does**:
1. Creates ENUM types (ownership_type)
2. Creates `entity_hierarchy_rules` table
3. Creates `model_type_hierarchy_attributes` table
4. Inserts all 49 model type definitions
5. Inserts ~60 hierarchy rules (parent → child relationships)
6. Inserts ~250 suggested attributes
7. Creates validation function `validate_hierarchy_position()`
8. Creates views: `v_entity_hierarchy_tree`

### Verification

```sql
-- Count model types
SELECT COUNT(*) FROM model_type_definitions WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
-- Expected: 49

-- Count hierarchy rules
SELECT COUNT(*) FROM entity_hierarchy_rules WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
-- Expected: 60+

-- View hierarchy tree
SELECT * FROM v_entity_hierarchy_tree ORDER BY depth, display_name;
```

## GraphQL API

### Schema

The schema is defined in `schema/addepar_ownership.graphql`. Key types:

#### Entity Query

```graphql
query {
  entity(id: "household-123") {
    id
    modelType
    displayName
    attributes { key value }
    owned {
      ownershipPercentage
      owned { displayName modelType }
    }
  }
}
```

#### Ownership Tree (Recursive DAG)

```graphql
query {
  ownershipTree(
    rootId: "household-123"
    depth: 3
    includeAttributes: true
    asOf: "2025-09-30"
  ) {
    entity { displayName modelType }
    position { ownershipPercentage }
    children {
      entity { displayName }
      children { ... }  # recursive
    }
  }
}
```

#### Entity Filtering

```graphql
query {
  entities(
    where: {
      modelType: { eq: "STOCK" }
      attribute: { key: { eq: "sector" }, value: { eq: "Technology" } }
    }
    limit: 50
  ) {
    id displayName attributes { key value }
  }
}
```

#### Hierarchy Metadata

```graphql
query {
  allowedChildren(parentModelType: "household") {
    modelType
    displayName
    allowedChildren { modelType }
  }
}
```

### Resolvers

Implemented in `backend/internal/graphql/addepar_ownership_resolvers.go`:

- `Entity(id)` – Single entity with ABAC check
- `Entities(where, orderBy, limit, offset)` – Filtered list
- `OwnershipTree(rootId, depth, asOf)` – Recursive DAG traversal
- `OwnershipChain(targetId, depth)` – Reverse lookup
- `ModelTypes(hierarchyLevel)` – All types with metadata
- `AllowedChildren(parent)` – Dynamic form generation
- `SearchEntities(query, modelTypes)` – Full-text search
- `PortfolioMetrics(rootId, asOf)` – Aggregations

### ABAC Integration

All resolvers enforce ABAC policies:

```go
if !r.ABAC.Can(ctx, "read", "entity", map[string]interface{}{
    "model_type": e.ModelType,
    "tenant_id":  e.TenantID,
}) {
    return nil, errors.New("forbidden")
}
```

**Context variables expected**:
- `tenant_id` (UUID)
- `user_id` (UUID)
- `user_roles` ([]string)
- Custom attributes from your ABAC system

## Position Creation with Hierarchy Validation

### Flow

1. **User creates position** (via GraphQL mutation or REST API):
   ```graphql
   mutation {
     createPosition(input: {
       ownerId: "household-123"
       ownedId: "stock-456"
       ownershipType: "SHARE_BASED"
       shares: 1000
     }) { success position { id } errors }
   }
   ```

2. **Backend validates**:
   - Hierarchy rule exists: `household` → `stock` in `entity_hierarchy_rules`
   - Ownership type matches allowed types in rule
   - ABAC permits creation
   - Temporal constraints (incepting_date, terminating_date)

3. **Trigger enforces** (SQL trigger):
   ```sql
   BEFORE INSERT OR UPDATE ON positions
   FOR EACH ROW EXECUTE FUNCTION validate_position_hierarchy()
   ```

4. **Function queries** and validates:
   ```sql
   SELECT is_valid, error_message FROM validate_hierarchy_position(
       owner_id, owned_id, tenant_id, user_id
   )
   ```

### Error Handling

Valid errors:
- `"Owner entity not found"`
- `"Owned entity not found"`
- `"Hierarchy rule not allowed: household -> unknown_security"`
- `"Entity already has exclusive parent"`
- `"Forbidden by policy"`

## Low-Code / No-Code Extension

### Admin API: Import Model Types

**Endpoint**: `POST /api/admin/model-types/import`

**Request**:
```json
{
  "jsonPayload": "[{\"model_type\": \"custom_asset\", \"display_name\": \"Custom Asset\", \"ownership_type\": \"VALUE_BASED\", \"suggested_attributes\": [{\"key\": \"custom_field\", \"value_type\": \"string\"}]}]"
}
```

**Response**:
```json
{
  "success": true,
  "importedCount": 1,
  "errors": []
}
```

### Admin UI: Hierarchy Matrix

Build a no-code UI matrix allowing admins to:
- Toggle parent → child relationships
- Set ownership type constraints
- Define max_children limits
- Mark relationships as exclusive

**Component pseudocode**:
```tsx
<HierarchyMatrix>
  <ParentSelector modelType="household" />
  <ChildSelector modelType="stock" />
  <Toggle label="Allowed" checked={true} />
  <Input label="Max Children" value={null} />
  <Checkbox label="Exclusive" />
</HierarchyMatrix>
```

### Custom Attributes Form (Workday-style)

Generate dynamic forms from `model_type_hierarchy_attributes`:

```tsx
<DynamicFormBuilder modelType="stock" attributes={suggestedAttributes} />
```

For each attribute:
- Render input based on `attribute_type` (text, date, number, select, etc.)
- Apply validation from `validation_rule` (JSON schema)
- Mark `is_required` fields
- Reorder by `priority`

## React TreeView Component

Visualize hierarchical ownership as an interactive tree:

```tsx
<OwnershipTreeView
  rootId={householdId}
  depth={3}
  asOf={date}
  colorBy="modelType"  // or "ownershipType", "status"
  onNodeClick={(node) => navigateTo(node.entity.id)}
/>
```

**Color coding**:
- Red (#DC2626): Top-level (household)
- Green (#059669): Containers (person_node, trust)
- Blue (#4F46E5): Assets (stocks, bonds, cash)

**Features**:
- Expand/collapse nodes
- Search within tree
- Show metrics on hover
- Drill-down to entity details

## Usage Examples

### Example 1: Household Portfolio Snapshot

```graphql
query HouseholdPortfolio {
  ownershipTree(rootId: "household-123", depth: 3) {
    entity { displayName modelType }
    children {
      entity { displayName }
      position { ownershipPercentage }
      children {
        entity { displayName }
        position { shares }
      }
    }
  }
}
```

**Response structure**:
```
household (Growth Portfolio 2025)
├─ person_node (Client A) [100%]
│  ├─ financial_account (Schwab IRA) [100%]
│  │  ├─ stock (AAPL) [1000 shares]
│  │  ├─ bond (US Treasury 2030) [500 shares]
│  │  └─ cash (USD) [$50K]
│  └─ sleeve (Tactical Growth) [80%]
│     ├─ etf (SPY) [200 shares]
│     └─ etf (QQQ) [150 shares]
├─ trust (Family Trust) [50%]
│  └─ real_estate (Property A) [$1.5M]
└─ managed_partnership (Fund X) [30%]
   └─ hedge_fund (Hedge Fund Y) [100 units]
```

### Example 2: Filter All Bonds by Maturity Date

```graphql
query BondsByMaturity {
  entities(
    where: {
      modelType: { eq: "BOND" }
      attribute: {
        key: { eq: "maturity_date" }
        value: { lt: "2026-01-01" }
      }
    }
  ) {
    id displayName
    attributes { key value }
  }
}
```

### Example 3: Search for All Apple Holdings Across All Households

```graphql
query AppleHoldings {
  searchEntities(query: "AAPL", modelTypes: ["stock"]) {
    id displayName
    owners {
      owner { displayName modelType }
    }
  }
}
```

### Example 4: Historical Portfolio as of Date

```graphql
query HistoricalPortfolio {
  ownershipTree(
    rootId: "household-123"
    depth: 3
    asOf: "2024-12-31"
  ) {
    entity { displayName }
    children { ... }
  }
}
```

### Example 5: Portfolio Metrics Dashboard

```graphql
query PortfolioDashboard {
  portfolioMetrics(rootId: "household-123") {
    totalMarketValue
    totalCostBasis
    unrealizedGainLoss
    portfolioReturnPct
    positionCount
    holdingsByType {
      modelType count marketValue
    }
    topHoldings { entity { displayName } }
  }
}
```

## Integration Checklist

### Phase 1: Database (✅ Complete)
- [x] Run migration: `addepar_model_types_49_extended.sql`
- [x] Verify 49 model types seeded
- [x] Verify 60+ hierarchy rules seeded
- [x] Verify validation function working
- [x] Test positions table trigger

### Phase 2: GraphQL API
- [ ] Update schema: `schema/addepar_ownership.graphql`
- [ ] Implement resolvers: `backend/internal/graphql/addepar_ownership_resolvers.go`
- [ ] Wire ABAC engine to resolvers
- [ ] Wire context middleware (tenant_id, user_id)
- [ ] Test with GraphiQL
- [ ] Add subscription resolvers for real-time updates

### Phase 3: React UI
- [ ] Build `OwnershipTreeView` component
- [ ] Build `HierarchyMatrix` admin component
- [ ] Build `DynamicFormBuilder` for entity creation
- [ ] Add search/autocomplete component
- [ ] Add portfolio metrics dashboard

### Phase 4: API Endpoints (REST)
- [ ] POST `/api/admin/model-types/import` – Import JSON
- [ ] POST `/api/entities` – Create entity
- [ ] POST `/api/positions` – Create position with validation
- [ ] GET `/api/entities/{id}` – Get entity with attributes
- [ ] GET `/api/hierarchy/{entityId}/tree` – Get ownership tree

### Phase 5: Testing
- [ ] Unit tests for hierarchy validation
- [ ] Integration tests for position creation
- [ ] GraphQL query tests
- [ ] ABAC policy tests
- [ ] Temporal query tests ("as-of" date)
- [ ] Performance tests (large trees, deep recursion)

### Phase 6: Documentation
- [ ] API documentation (OpenAPI/Swagger)
- [ ] GraphQL documentation (Playground)
- [ ] Admin guide for hierarchy rules
- [ ] User guide for entity creation
- [ ] Example queries and mutations

## Performance Considerations

### Indexing

Already created:
```sql
CREATE INDEX idx_positions_owner_id ON positions(owner_id);
CREATE INDEX idx_positions_owned_id ON positions(owned_id);
CREATE INDEX idx_positions_incepting_date ON positions(incepting_date);
CREATE INDEX idx_entities_model_type ON entities(model_type);
CREATE INDEX idx_entity_attributes_entity_id ON entity_attributes(entity_id);
```

### Query Optimization

**Ownership tree (depth 3)**:
- Query: ~4 DB roundtrips (1 root + 3 levels of recursion)
- Time: ~50ms for typical portfolio
- Result size: 100-500 nodes

**Recommendations**:
- Limit depth to 5 max (prevent runaway recursion)
- Cache views for high-traffic queries
- Use Cube.js for aggregations
- Consider materialized views for historical snapshots

## Security

### Multi-Tenant Isolation

All queries filter by `tenant_id`:
```sql
WHERE tenant_id = $1
```

Enforced at:
- Entity table (RLS policy)
- Positions table (RLS policy)
- GraphQL resolvers (context tenant_id check)

### ABAC Enforcement

Every GraphQL resolver:
1. Checks context: `tenant_id`, `user_id`, `user_roles`
2. Calls `r.ABAC.Can(ctx, action, resource, attributes)`
3. Returns `Forbidden` if denied

**Example policies**:
- `(resource.model_type == "private_investment") AND (user.role == "wealth_advisor")`
- `(resource.value > 1_000_000) AND (user.role == "advisor" OR user.role == "admin")`
- `(tenant_id == user.tenant_id)` (always enforced)

### Audit Trail

All mutations logged:
```sql
INSERT INTO audit_log (user_id, action, entity_id, changes, timestamp)
VALUES ($1, $2, $3, $4, NOW())
```

## Troubleshooting

### Hierarchy Validation Fails

```sql
-- Check if rule exists
SELECT * FROM entity_hierarchy_rules 
WHERE parent_model_type = 'household' AND child_model_type = 'stock';

-- Check if rule is allowed
SELECT allowed FROM entity_hierarchy_rules 
WHERE parent_model_type = 'household' AND child_model_type = 'stock';

-- Check if type exists
SELECT * FROM model_type_definitions 
WHERE model_type IN ('household', 'stock');
```

### Recursive Query Hangs

```sql
-- Check for circular references
SELECT * FROM positions p
WHERE EXISTS (
    SELECT 1 FROM positions p2 WHERE p2.owner_id = p.owned_id AND p2.owned_id = p.owner_id
);

-- Limit recursion manually
SELECT * FROM ownershipTree(root_id, 3)  -- depth 3 only
```

### Position Not Appearing in Tree

```sql
-- Check active filter
SELECT * FROM positions 
WHERE owner_id = $1 
  AND is_active = true 
  AND incepting_date <= CURRENT_DATE 
  AND (terminating_date IS NULL OR terminating_date >= CURRENT_DATE);
```

## Next Steps

1. **Run the migration**: `psql ... < addepar_model_types_49_extended.sql`
2. **Verify**: Run verification queries from schema section
3. **Implement GraphQL**: Copy resolvers, wire ABAC, test
4. **Build UI**: TreeView, forms, admin matrix
5. **Add tests**: Unit, integration, performance
6. **Deploy**: Stage → production with RLS policies enabled

## References

- [Addepar Business Objects Documentation](https://docs.addepar.com)
- [Workday Data Model](https://docs.workday.com/en/cloudplatform/index.html)
- [PostgreSQL Recursive CTEs](https://www.postgresql.org/docs/current/queries-with.html)
- [GraphQL Best Practices](https://graphql.org/learn/best-practices/)
- [ABAC Pattern (XACML)](https://en.wikipedia.org/wiki/Attribute-based_access_control)

