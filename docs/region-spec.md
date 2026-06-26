# REGION MODEL — AUTHORITATIVE SPEC FOR CODE GENERATION

This file is a ready-to-paste block for your code generator/system prompt. It defines the *region* concept (first-class) and the non-negotiable rules the generator must enforce when generating code.

---

## 1. What a Region Is
A **region** is a first-class semantic and physical boundary that controls:
- Data residency
- Execution locality
- Pre-aggregation placement
- Query routing
- Snapshot scoping
- Tenant isolation across geographies

Examples: `"us-east"`, `"eu-west"`, `"ap-south"`.

A region is **never optional**.

---

## 2. Global Invariants (ENFORCE)
1. Every runtime operation must be scoped by `(tenant_id, region)`.
2. Snapshots are always `(tenant_id, region)` pairs.
3. Pre-aggregations are region-bound. A pre-agg is eligible only if its `region == request.region`.
4. No cross-region joins or reads. Planner must reject cross-region plans.
5. No silent defaults for region — missing region is an error.

---

## 3. Where `region` Appears in the Schema
- `business_objects.region` (optional override; otherwise derive from tenant/datasource)
- `preaggregations.region` (required)
- `semantic_snapshots.tenant_id`, `semantic_snapshots.region` (region required)
- `semantic_policies.region` (optional)

---

## 4. Runtime Contracts (MUST be enforced by generated code)
- Snapshot lookup must always be: `snapshot := snapshots.Get(ctx, tenantID, region)` (never `Get(ctx, tenantID)`)

- Planner request must include `Region`:
```go
type PlanRequest struct {
    Snapshot       *SemanticSnapshot
    TenantID       string
    Region         string
    BusinessObject string
    Dimensions     []string
    Measures       []string
    Filters        []string
}
```

- MCP tool schemas must include `tenant_id` and `region`:
```json
{
  "name": "plan_query",
  "parameters": {
    "type": "object",
    "properties": {
      "tenant_id": { "type": "string" },
      "region": { "type": "string" },
      "business_object": { "type": "string" },
      "dimensions": { "type": "array", "items": { "type": "string" } },
      "measures": { "type": "array", "items": { "type": "string" } }
    },
    "required": ["tenant_id", "region", "business_object"]
  }
}
```

- Pre-agg eligibility check must filter on region:
```go
if pa.Region != req.Region {
    continue
}
```

- Execution engine selection is region-bound:
```go
engine := engineRegistry.ForRegion(req.Region)
```

---

## 5. Snapshot Structure
Snapshots are immutable and region-scoped:
```json
{
  "snapshot_id": "v2026_02_06_001",
  "tenant_id": "acme",
  "region": "eu-west",
  "business_objects": { ... },
  "policies": { ... }
}
```

---

## 6. Resolution & Error Rules
- If `region` is missing: return `Error: region is required for all semantic operations.`
- If `region` unknown for tenant: `Error: region '<value>' is not configured for tenant '<tenant>'.`
  - Source of truth: Tenant-level configuration stored in `tenants.allowed_regions` (JSONB array). The system also falls back to `tenants.metadata.allowed_regions` when `allowed_regions` is not present.
- If pre-agg wrong region: `Error: pre-aggregation '<name>' is not available in region '<region>'.`
- If BO not available in region: `Error: business object '<bo>' is not available in region '<region>'.`

---

**Paste this block into your generator's system prompt or design context** so generated code adheres strictly to region invariants.
