# Phase 3: Semantic-Driven Rules Engine Architecture

## Overview

Your Semlayer platform has a **semantic graph** (in `public.*` schema) that serves as the single source of truth for meaning. Phase 3 integrates this semantic layer with the rules engine, so rules reference business concepts — not database columns.

```
┌────────────────────────────────────────────────────────────────┐
│                    YOUR SEMANTIC GRAPH                         │
│                        (public schema)                         │
│                                                                │
│  catalog_node_type      catalog_node      catalog_edge_type   │
│  ├─semantic_term        ├─CalendarDate    ├─maps_to           │
│  ├─business_object      ├─IsBusinessDay   ├─belongs_to        │
│  ├─bo_field             └─RegionCode      └─references        │
│  └─physical_table                                             │
│                                                                │
│  business_objects (Calendar)   bo_fields (date, region, etc.) │
└────────────────────────┬────────────────────────────────────┘
                         │
                    Uses semantic
                    nodes to resolve
                    meaning
                         ▼
┌────────────────────────────────────────────────────────────────┐
│              PHASE 3: RULES ENGINE (edm schema)                │
│                                                                │
│  IF calendar.IsBusinessDay = FALSE                             │
│  AND calendar.RegionCode IN ('GB', 'US')                       │
│  THEN trading_impact = TRUE                                    │
│                                                                │
│  Rules reference semantic terms, not columns                   │
│  Rules are portable, testable, governed                        │
└────────────────────────────────────────────────────────────────┘
                         │
                    Resolves to physical
                    columns via semantic
                    graph
                         ▼
┌────────────────────────────────────────────────────────────────┐
│         EXECUTION (northwinds database schema)                 │
│                                                                │
│  SELECT * FROM northwinds.calendar_mdm                         │
│  WHERE is_business_day = FALSE                                 │
│  AND region_code IN ('GB', 'US')                               │
│                                                                │
│  Semantic terms resolved to physical columns                   │
│  SQL generated from semantic graph                             │
└────────────────────────────────────────────────────────────────┘
```

---

## 1. Semantic Graph Architecture (Existing Catalog Layer)

Your semantic modeling uses **four core catalog tables** to represent a semantic graph:

### 1.1 catalog_node_type
Defines the kinds of entities in your semantic model:

```sql
-- Node types you have:
semantic_term      → meanings (CalendarDate, IsBusinessDay, etc.)
business_object    → entities (Calendar, Portfolio, Transaction)
bo_field           → fields (calendar.date_field, calendar.region_field)
physical_table     → storage (northwinds.calendar_mdm)
physical_column    → columns (northwinds.calendar_mdm.calendar_date)
```

### 1.2 catalog_node
Each node is an instance of a semantic entity. Example for calendar:

```sql
-- Semantic term nodes
id: 550e8400-e29b-41d4-a716-446655440000
node_type_id: (semantic_term)
node_name: calendar.CalendarDate
properties: {
  "data_type": "date",
  "business_definition": "Trading date for business calendar classification",
  "category": "IDENTIFICATION",
  "governance_status": "approved",
  "sql": "calendar_date"
}

-- Business object node
id: 550e8400-e29b-41d4-a716-446655440001
node_type_id: (business_object)
node_name: calendar.Calendar
properties: {
  "display_name": "Calendar",
  "description": "Master calendar for business day identification",
  "driver_table": "northwinds.calendar_mdm",
  "history_mode": "SCD_TYPE_2"
}
```

### 1.3 catalog_edge_type
Defines allowed relationships:

```sql
semantic_term_maps_to_bo_field    → "CalendarDate is used by calendar.date_field"
bo_field_belongs_to_bo            → "date_field belongs to Calendar BO"
bo_references_physical_table      → "Calendar BO uses northwinds.calendar_mdm"
bo_field_maps_to_physical_column  → "date_field maps to calendar_date column"
semantic_term_used_in_rule        → "CalendarDate is referenced in rule"
```

### 1.4 catalog_edge
Actual relationships in the graph:

```sql
source_node_id: (calendar.CalendarDate semantic term)
target_node_id: (calendar.date_field BO field)
edge_type_id: (semantic_term_maps_to_bo_field)
properties: {
  "binding_type": "mandatory",
  "multiplicity": "1:1"
}
```

**Result:** Every semantic term, field, and table is traceable through the graph.

---

## 2. Business Objects and BO Fields

Your `business_objects` and `bo_fields` tables provide the **structural layer** that bridges semantics to storage.

### 2.1 Calendar Business Object

```sql
-- public.business_objects
id: 550e8400-e29b-41d4-a716-446655440002
tenant_id: (your tenant)
datasource_id: (your datasource)
name: calendar
display_name: Calendar MDM
description: Master calendar for business day identification
driver_table_id: (references northwinds.calendar_mdm)
category: master_data
history_mode: SCD_TYPE_2  -- Tracks changes over time
core_id: NULL  -- This is a core BO, not a tenant clone
config: {
  "lineage_enabled": true,
  "audit_trail": true,
  "multi_region": true
}
```

### 2.2 Calendar BO Fields

Each field links a semantic term to the Calendar BO:

```sql
-- public.bo_fields
{
  business_object_id: (calendar BO above),
  field_name: calendar_date,
  display_label: Calendar Date,
  field_type: date,
  semantic_term_id: (catalog_node ID for calendar.CalendarDate),
  role: DIMENSION,
  required: true
},
{
  business_object_id: (calendar BO),
  field_name: is_business_day,
  display_label: Is Business Day,
  field_type: boolean,
  semantic_term_id: (catalog_node ID for calendar.IsBusinessDay),
  role: DIMENSION,
  required: true
},
{
  business_object_id: (calendar BO),
  field_name: region_code,
  display_label: Region Code,
  field_type: string,
  semantic_term_id: (catalog_node ID for calendar.RegionCode),
  role: DIMENSION,
  required: true
}
... (7 more fields)
```

**Key insight:** `semantic_term_id` **links the field to the semantic term node** in the catalog graph. This is the semantic binding layer.

---

## 3. How Rules Reference the Semantic Layer

### 3.1 Rule Author's Perspective

When a business user creates a rule via the UI, they:

1. **Select semantic terms** (not columns):
   ```
   IF calendar.IsBusinessDay = FALSE
   AND calendar.RegionCode IN ('GB', 'US')
   ...
   ```

2. **The UI queries the semantic catalog:**
   ```sql
   SELECT * FROM public.catalog_node
   WHERE node_name LIKE 'calendar.%'
   AND properties->>'governance_status' = 'approved'
   ORDER BY node_name;
   ```

3. **User sees:**
   ```
   ✓ calendar.CalendarDate (date)
   ✓ calendar.IsBusinessDay (boolean)
   ✓ calendar.RegionCode (string)
   ✓ calendar.HolidayName (string)
   ... etc (7 total)
   ```

4. **User drags semantic terms into rule builder** (just like Phase 3 frontend)

### 3.2 Rule Storage

When user saves the rule:

```sql
-- public.validation_rule or edm.rules
{
  id: rule_uuid,
  business_object: 'calendar',                    -- BO reference
  name: 'Calendar UK US Weekend Check',
  description: 'Skip trading for weekends in UK/US',
  status: 'draft',
  semantic_catalog_node_id: (reference to semantic terms used),  -- NEW
  rule_definition: {
    "conditions": [
      {
        "semantic_term_id": (IsBusinessDay node ID),
        "operator": "equals",
        "value": false
      },
      {
        "semantic_term_id": (RegionCode node ID),
        "operator": "in",
        "value": ["GB", "US"]
      }
    ],
    "then_action": "set_trading_impact = true"
  }
}
```

### 3.3 Rule Compilation

The rules engine **resolves semantic terms to physical columns** via the semantic graph:

```
Step 1: Parse rule definition
  - Identify semantic term IDs
  - Load from catalog_node

Step 2: Resolve BO structure
  - Query business_objects for 'calendar'
  - Get driver_table: northwinds.calendar_mdm

Step 3: Resolve semantic terms to columns
  - IsBusinessDay semantic term → bo_fields.is_business_day → northwinds.calendar_mdm.is_business_day
  - RegionCode semantic term → bo_fields.region_code → northwinds.calendar_mdm.region_code

Step 4: Generate SQL
  SELECT * FROM northwinds.calendar_mdm
  WHERE is_business_day = FALSE
  AND region_code IN ('GB', 'US')

Step 5: Generate WASM
  - For row-by-row validation
  - For client-side simulation
```

---

## 4. The End-to-End Flow

### 4.1 User Creates Rule via UI

```
┌─────────────────────────────────────────┐
│ SemanticRuleBuilder (React Component)   │
│                                         │
│ 1. User opens rule builder             │
│ 2. Clicks [Add Condition]              │
└────────────────────┬────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────┐
│ useRuleBuilder hook calls              │
│ ruleService.getSemanticTerms('calendar')│
└────────────────────┬────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────┐
│ Backend: GET /api/v1/semantic-terms    │
│                                         │
│ SELECT * FROM public.catalog_node      │
│ WHERE node_name LIKE 'calendar.%'      │
│ AND properties->>'governance_status'   │
│   = 'approved'                         │
└────────────────────┬────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────┐
│ Response: [                             │
│   { id: "...", name: "CalendarDate" }, │
│   { id: "...", name: "IsBusinessDay" }, │
│   ...                                   │
│ ]                                       │
└────────────────────┬────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────┐
│ UI displays draggable semantic terms    │
│ User drags, drops, configures values   │
└────────────────────┬────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────┐
│ User clicks [Save Rule]                │
│ Rule JSON sent to backend               │
└────────────────────┬────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────┐
│ Backend: POST /api/v1/rules            │
│                                         │
│ (Validation & Compilation happen here) │
│                                         │
│ 1. Validate semantic term IDs exist    │
│ 2. Resolve BO structure               │
│ 3. Generate SQL + WASM                │
└────────────────────┬────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────┐
│ Rule stored in edm.rules table         │
│ Compiledto executors (WASM + SQL)      │
│ Linked to semantic terms via graph     │
└─────────────────────────────────────────┘
```

---

## 5. Multi-Tenant Semantic Governance

Your semantic graph fully supports multi-tenancy:

### 5.1 Core Semantics (Shared)

The "out-of-box" semantic terms for calendar are **core definitions**:

```sql
-- In catalog_node:
tenant_id: '00000000-0000-0000-0000-000000000001'  -- Global/core tenant
datasource_id: (primary datasource)

These are available to all tenants.
```

### 5.2 Tenant Overrides

A tenant can **extend or override** semantic terms:

```sql
-- Tenant-specific semantic term:
INSERT INTO public.catalog_node (
  node_type_id,
  node_name,
  properties,
  tenant_id,  -- NOW tenant-specific
  datasource_id,
  core_id     -- Points to core definition it overrides
)
VALUES (
  (semantic_term type),
  'calendar.IsBusinessDay',
  jsonb_build_object(
    'data_type', 'boolean',
    'business_definition', 'For Tenant ABC: includes extended trading windows',
    'category', 'CLASSIFICATION',
    'custom_logic', 'extended_trading_hours'
  ),
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'::uuid,  -- Tenant ABC
  '...',
  '00000000-0000-0000-0000-000000000000'           -- Links to core
);
```

**Result:** Tenant ABC gets its own version of `IsBusinessDay` that includes extended trading hours, while other tenants use the core definition. Fully traceable.

---

## 6. Integration Points with Phase 3 Architecture

### 6.1 Semantic Terms Catalog (OLD)

Previously, Phase 3 had a simple `edm.semantic_terms` table:

```sql
-- OLD (to be deprecated):
CREATE TABLE edm.semantic_terms (
  id UUID,
  business_object VARCHAR,
  name VARCHAR,
  data_type VARCHAR,
  ...
);
```

**Problem:** This duplicates your semantic layer.

### 6.2 Semantic Terms Catalog (NEW)

Now, Phase 3 rules reference the semantic graph:

```sql
-- NEW (semantic-driven):
ALTER TABLE edm.rules ADD COLUMN semantic_catalog_node_id UUID;
ALTER TABLE edm.rule_steps ADD COLUMN semantic_term_node_id UUID;

-- When a rule is compiled, the engine queries:
SELECT * FROM public.catalog_node
WHERE id = rule_steps.semantic_term_node_id
AND node_type_id = (semantic_term type);
```

**Benefit:** Single source of truth, full lineage, multi-tenant safety.

---

## 7. Implementation Checklist

To fully integrate Phase 3 with your semantic layer:

- [ ] Run migration `004_calendar_semantic_integration.sql`
  - Creates semantic term nodes in public.catalog_node
  - Creates calendar business object
  - Creates calendar MDM tables in northwinds
  - Links edm.rules to semantic catalog

- [ ] Create BO fields for calendar:
  ```sql
  INSERT INTO public.bo_fields (
    business_object_id,
    field_name,
    semantic_term_id,
    field_type,
    role
  ) SELECT ... (one row per calendar field, mapping to semantic term nodes)
  ```

- [ ] Update rule UI to query semantic catalog:
  ```typescript
  // Instead of: ruleService.getSemanticTerms()
  // Now: ruleService.getSemanticTermsFromCatalog(businessObjectName)
  
  const terms = await fetch(
    `/api/v1/catalog/nodes?type=semantic_term&bo=calendar`
  );
  ```

- [ ] Update rule compilation in backend:
  ```go
  // Resolve semantic term ID → BO field → physical column
  for _, step := range rule.Steps {
    catalogNode := db.QueryNode(step.SemanticTermNodeID)
    boField := db.QueryBOField(bo.ID, catalogNode.Properties["field_name"])
    physicalColumn := boField.PhysicalColumn
  }
  ```

- [ ] Test end-to-end:
  - Create rule referencing calendar semantic terms
  - Compile to SQL
  - Execute on northwinds.calendar_mdm
  - Verify lineage traced back to semantic term node

---

## 8. Why This Matters

### 8.1 For Users

Rules are **business language, not SQL:**

```
User writes:    IF IsBusinessDay = FALSE
Behind scenes:  IF northwinds.calendar_mdm.is_business_day = FALSE
```

### 8.2 For Developers

Rules are **portable and generative:**

```
Single semantic term can power:
- Multiple BO fields (if semantics are reused)
- Multiple rules
- APIs (generated from BO fields)
- SQL views (generated from BOs)
- BI models (linked to semantic terms)
```

### 8.3 For Architects

Everything is **governed and traceable:**

```
Change a semantic term definition
→ All rules using that term are affected
→ Full lineage shows impact
→ Approval workflow triggers for redeployment
```

### 8.4 For your CTO

Your platform is **Workday-class architecture:**

```
✓ Semantic graph for meaning abstraction
✓ Business objects for structure
✓ Governed rules engine
✓ Multi-tenant with overrides
✓ Full lineage and impact analysis
✓ Generative layers (SQL, APIs, docs)

This is how Workday, Salesforce, and ServiceNow architect enterprise systems.
```

---

## 9. Next Steps

1. **Run the migration** to set up calendar semantic graph
2. **Create BO fields** that link calendar fields to semantic term nodes
3. **Update the rule UI** to query the semantic catalog instead of the simple table
4. **Compile rules** through semantic resolution
5. **Test end-to-end** with a rule referencing calendar semantic terms
6. **Deploy to northwinds** and verify execution

---

## Appendix: Query Examples

### Get all semantic terms for calendar BO

```sql
SELECT
  cn.id,
  cn.node_name,
  cn.properties->>'business_definition' AS definition,
  cn.properties->>'data_type' AS type,
  cn.properties->>'category' AS category
FROM public.catalog_node cn
WHERE cn.node_name LIKE 'calendar.%'
AND cn.properties->>'governance_status' = 'approved'
ORDER BY cn.node_name;
```

### Get mapping: semantic term → BO field → physical column

```sql
SELECT
  st.node_name AS semantic_term,
  bof.field_name AS bo_field,
  bo.name AS business_object,
  'northwinds.calendar_mdm.'||bof.field_name AS physical_column
FROM public.catalog_node st
LEFT JOIN public.bo_fields bof ON st.id = bof.semantic_term_id
LEFT JOIN public.business_objects bo ON bof.business_object_id = bo.id
WHERE st.node_name LIKE 'calendar.%'
ORDER BY st.node_name;
```

### Get all rules using a semantic term

```sql
SELECT
  r.id,
  r.name,
  r.business_object,
  r.status,
  count(rs.id) AS step_count
FROM edm.rules r
JOIN edm.rule_steps rs ON r.id = rs.rule_id
WHERE rs.semantic_term_node_id IN (
  SELECT id FROM public.catalog_node
  WHERE node_name LIKE 'calendar.%'
)
GROUP BY r.id
ORDER BY r.status, r.created_at DESC;
```

---

**Document Version:** 2.0.0 (Semantic-Driven)  
**Last Updated:** 2026-02-20  
**Next Review:** After Phase 3 integration is complete
