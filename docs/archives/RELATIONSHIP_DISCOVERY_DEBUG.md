# Relationship Discovery Analysis & Fix

## Problem Identified

The current discovery algorithm requires this chain to find relationships:

```
Entity (Customer) 
  → has Semantic Term
    → maps to Column
      → in Table
        → connected via Foreign Key
          → to Target Table
            → has Column
              → maps to Semantic Term
                → has Entity (Order)
```

**Issue:** If the foreign key relationship exists in `catalog_edge` but there are no semantic term mappings on both sides, the relationship won't be discovered.

## Your Scenario

```
Customer Entity:
  - customer_id column
  
Order Entity:
  - customer_id column (Foreign Key)
  
catalog_edge table:
  - Has edge: Customer.customer_id → Order.customer_id (FK relationship)
```

**The missing piece:** The algorithm doesn't have a direct path to discover this FK without semantic terms.

## Solution

Update the discovery algorithm to add a **direct foreign key discovery path** that doesn't require semantic term mappings. This should run FIRST and be combined with the existing semantic-term-based discovery.

### New Algorithm Flow

**Path 1: Direct Foreign Key Discovery (NEW)**
```
Entity (Customer)
  → has mapped columns
    → in tables
      → connected via Foreign Key in catalog_edge
        → to target tables
          → collect ALL entities that represent those tables
```

**Path 2: Semantic Term Discovery (EXISTING)**
```
(Keep existing logic for semantic-based discovery)
```

**Then combine both paths.**

## Implementation Changes Needed

In `relationships_discovery.go`:

1. Add a new CTE that finds foreign keys directly from source tables
2. For each foreign key target table, find ANY entity associated with that table
3. Union with existing semantic-term-based discovery

The key is to add this logic:
```sql
-- Direct FK discovery without requiring semantic terms
direct_fk_targets AS (
  -- Find tables that have FK relationships to/from source tables
  SELECT DISTINCT target_table_id as table_id, target_table_name as table_name
  FROM all_foreign_keys
  WHERE source_table_id IN (SELECT table_id FROM source_tables)
  
  UNION
  
  SELECT DISTINCT source_table_id as table_id, source_table_name as table_name  
  FROM all_foreign_keys
  WHERE target_table_id IN (SELECT table_id FROM source_tables)
),

-- Get entities for these tables (may have no semantic terms)
entities_for_fk_targets AS (
  SELECT DISTINCT
    cn.id as entity_id,
    cn.node_name as entity_name,
    dft.table_name,
    'foreign_key_direct' as link_type
  FROM direct_fk_targets dft
  JOIN catalog_node cn ON cn.node_name = dft.table_name
  WHERE cn.catalog_type_name = 'entity'
    AND cn.tenant_datasource_id = $2
)
```

Then combine with existing results.

## Questions for Debugging

To help you further, I need to understand your data:

1. **In catalog_node:** Do you have entity nodes for both "Customer" and "Order"?
2. **In catalog_edge:** Is there an edge with:
   - `predicate = 'foreign_key'`
   - `relationship_type = 'foreign_key'`
   - Connecting Customer table to Order table?
3. **For columns:** Are customer_id columns in catalog_node?
4. **Semantic mapping:** Are there "customer_id" semantic terms mapped to the columns?

## Recommended Debugging Steps

Run these queries to diagnose:

```sql
-- 1. Check entities exist
SELECT id, node_name, catalog_type_name 
FROM catalog_node 
WHERE node_name IN ('Customer', 'Order', 'customer', 'order')
AND catalog_type_name IN ('entity', 'business_term');

-- 2. Check FK relationships exist
SELECT ce.id, cs.node_name as source, ct.node_name as target, cet.predicate
FROM catalog_edge ce
JOIN catalog_edge_type cet ON ce.edge_type_id = cet.id
JOIN catalog_node cs ON ce.source_node_id = cs.id
JOIN catalog_node ct ON ce.target_node_id = ct.id
WHERE cet.predicate = 'foreign_key'
ORDER BY cs.node_name, ct.node_name;

-- 3. Check semantic terms
SELECT node_name, catalog_type_name
FROM catalog_node
WHERE catalog_type_name = 'semantic_term'
AND node_name LIKE '%customer%';

-- 4. Check semantic mappings
SELECT ns.node_name as semantic, no.node_name as column, ct.node_name as table
FROM catalog_edge ce
JOIN catalog_edge_type cet ON ce.edge_type_id = cet.id
JOIN catalog_node ns ON ce.source_node_id = ns.id
JOIN catalog_node no ON ce.target_node_id = no.id
JOIN catalog_node ct ON ct.id = no.parent_id
WHERE cet.predicate = 'member of'
AND ce.relationship_type = 'MAPS_TO'
ORDER BY ct.node_name, no.node_name;
```

## Next Steps

Once I understand your data structure, I can:

1. Update the discovery algorithm to add the direct FK path
2. Test it with your Customer/Order scenario
3. Deploy the fix

Run the diagnostic queries above and share the results, then I can create the exact fix for your situation.
