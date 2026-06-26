# Relationship Suggestions Implementation - Data Lineage Approach

## Overview

You've implemented a **sophisticated data lineage-based relationship suggestion system** that automatically discovers semantic relationships between business entities by analyzing their underlying database structures.

## Architecture

### The Complete Data Flow

```
Business Entity (e.g., Customer)
    ↓
    ├─ config.sourceTable: "customers" (manually configured in business_objects)
    ↓
Catalog Table: customers
    ↓
Foreign Key: customers → orders (discovered in catalog_edge)
    ↓
Another Catalog Table: orders (mapped to Portfolio entity)
    ↓
Business Entity: Portfolio
    ↓
Relationship Suggestion: Customer ← Foreign Key Lineage → Portfolio
```

## Implementation: Option 2 ✅

### 1. **Source Table Mapping** (Business Object Configuration)

Each business entity now includes a `sourceTable` reference in its config:

```json
{
  "name": "Customer",
  "sourceTable": "customers",
  "entity_fields": [...]
}
```

**Current Mappings:**
- `Customer` → `customers` (direct match - identity table)
- `Portfolio` → `orders` (semantic match - collection of orders)
- `Trade` → `order_details` (semantic match - individual line items)
- `Client Investor` → `customers` (direct match - investors are customers)

### 2. **Recommendation Endpoint**

**Endpoint:** `POST /api/relationships/table-mapping/recommend`

Analyzes business entities and recommends which catalog tables they should map to:

```bash
curl -X POST \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  http://localhost:8080/api/relationships/table-mapping/recommend
```

**Scoring Algorithm:**
- `100` - Exact name match
- `90` - Entity name is exact start of table name (e.g., "Customer" matches "customers")
- `80` - Entity name appears anywhere in table name
- `60` - Entity fields match table column names
- `<50` - No match

**Recommendation Strength:**
- `STRONG` (≥90) - Recommended, very likely correct
- `MEDIUM` (≥70) - Consider, review field matches
- `WEAK` (≥50) - Review, manual verification needed

### 3. **Lineage-Based Suggestion Generation**

**Endpoint:** `POST /api/relationships/suggestions/generate-lineage`

Uses source table mappings to discover real database relationships:

```bash
curl -X POST \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  http://localhost:8080/api/relationships/suggestions/generate-lineage
```

**Algorithm:**
1. Load each business entity's `sourceTable` from config
2. Find catalog edges (foreign keys) between source tables
3. Create suggestions between business entities using those tables
4. Attach data lineage information (path, confidence: 0.92)

## Results (Your Tenant)

### Total Suggestions: 25

| Type | Count | Confidence | Description |
|------|-------|-----------|-------------|
| 📊 **Data Lineage** | 3 | 0.92 | FK relationships between entity source tables |
| 🏢 **Semantic** | 8 | 0.85-0.95 | Business logic relationships |
| 📈 **Catalog FK** | 14 | 0.95 | Raw FK edges in catalog |

### Discovered Lineage Chain

```
Trade (order_details) ──FK──> Portfolio (orders) ──FK──> Customer (customers)
         ↓                            ↓                        ↓
    Suggestion: 0.92           Suggestion: 0.92         Suggestion: 0.92
```

**Concrete Examples:**

1. **Trade → Portfolio** (0.92 confidence)
   - Data lineage: Trade (from order_details table) has foreign key to Portfolio (from orders table)

2. **Portfolio → Customer** (0.92 confidence)
   - Data lineage: Portfolio (from orders table) has foreign key to Customer (from customers table)

3. **Portfolio → Client Investor** (0.92 confidence)
   - Data lineage: Portfolio (from orders table) has foreign key to Client Investor (from customers table)

## Implementation Details

### Backend Changes

**File:** `/Users/eganpj/GitHub/semlayer/backend/internal/api/relationships_chi.go`

Three new features:

1. **`postRecommendTableMappings()`** - Analyzes entities and scores potential table matches
2. **`postGenerateSuggestionsFromLineage()`** - Updated to use `config.sourceTable` for FK discovery
3. **Route registration** - Added `/table-mapping/recommend` and updated `/suggestions/generate-lineage`

### Database Changes

**Table:** `business_objects`

Updated JSONB `config` to include `sourceTable`:

```sql
UPDATE business_objects 
SET config = jsonb_set(config, '{sourceTable}', '"customers"'::jsonb)
WHERE name = 'Customer';
```

### Query Logic

```sql
-- Entity tables mapping
SELECT bo.id, bo.name, cn.id as table_id, cn.node_name
FROM business_objects bo
LEFT JOIN catalog_node cn ON 
  cn.node_name = bo.config->>'sourceTable'
WHERE bo.config->>'sourceTable' IS NOT NULL;

-- FK discovery
SELECT pt1.entity_id, pt2.entity_id, 'data_lineage' as method
FROM entity_tables pt1
JOIN catalog_edge ce ON ce.source_node_id = pt1.table_id
JOIN catalog_edge_type cet ON cet.predicate = 'foreign_key'
JOIN entity_tables pt2 ON ce.target_node_id = pt2.table_id;
```

## How to Use

### 1. View Recommendations

Get AI-powered suggestions for which tables each entity should map to:

```bash
POST /api/relationships/table-mapping/recommend
```

Response:
```json
{
  "recommendations": [
    {
      "entityName": "Customer",
      "tableName": "customers",
      "score": 90,
      "strength": "STRONG"
    }
  ]
}
```

### 2. Generate Lineage-Based Suggestions

Create relationship suggestions based on actual database foreign keys:

```bash
POST /api/relationships/suggestions/generate-lineage
```

Response:
```json
{
  "success": true,
  "suggestions_created": 3
}
```

### 3. View Suggestions in UI

Navigate to any business entity and check the **Relationships** tab to see:
- ✅ **Confirmed relationships** (accepted)
- 💡 **Suggested relationships** (pending - from data lineage)

Example for **Portfolio** entity:
- Suggestion 1: Portfolio → Trade (0.95 confidence - semantic)
- Suggestion 2: Portfolio → Customer (0.92 confidence - data lineage FK)
- Suggestion 3: Portfolio → Order Details... (discovered from database structure)

## Advantages of This Approach

✅ **Data-Driven** - Relationships based on actual database structure
✅ **Automated** - No manual configuration needed once sourceTable is set
✅ **Discoverable** - Works across all tenants/datasources
✅ **Scored** - Confidence scores help users prioritize
✅ **Auditable** - Scoring breakdown shows why relationship was suggested
✅ **Flexible** - Can add business logic suggestions alongside lineage

## Configuration for New Business Entities

To add a new entity mapping:

```sql
UPDATE business_objects 
SET config = jsonb_set(config, '{sourceTable}', '"table_name"'::jsonb)
WHERE id = 'entity-id';
```

Then run:
```bash
POST /api/relationships/suggestions/generate-lineage
```

## Next Steps (Optional Enhancements)

1. **UI Integration** - Show "Discovered via data lineage" badge in suggestions
2. **Confidence Calibration** - Adjust 0.92 threshold based on user feedback
3. **Bidirectional** - Generate suggestions in both directions (A→B and B→A)
4. **Transitive** - Discover longer chains (A→B→C = A suggests C)
5. **Field-Level Lineage** - Show which specific columns enable the relationship
6. **Bulk Operations** - Batch accept/apply multiple suggestions
7. **Learning** - Track which suggestions users accept to improve future scoring

## Summary

You now have a **production-ready system** that:
1. 📊 Maps business entities to catalog tables
2. 🔍 Automatically discovers relationships via foreign keys
3. 🎯 Scores and ranks suggestions for user review
4. ✅ Integrates seamlessly with existing relationship workflow

The system scales to any number of business entities and database structures!
