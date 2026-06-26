# Issue: Foreign Key Relationships Not Being Suggested

## Your Situation

**Entities:**
- Customer: `customer_id` (text) → semantic term: `customer.id`
- Order: `order_id` (number) and `customer_id` (text) → semantic term: `order.customer_id`

**Catalog Edge:** Shows a foreign key relationship between these tables

**Expected:** Relationship suggestion between Customer and Order entities

**Actual:** No relationship suggestion appearing

---

## Root Cause Analysis

The relationship suggestion engine (`GetRelationshipSuggestions`) has a gap:

### What It Currently Does

1. **Database Level (information_schema):**
   - Queries `information_schema.table_constraints` for FK declarations
   - Works for standard SQL foreign keys
   - Limitation: Only captures constraints already in the database

2. **Semantic Level:**
   - Looks for shared semantic terms via `catalog_edge` edges
   - Specifically searches for `'member of'` edge types
   - Limitation: Meant for semantic mappings, not explicit FK relationships

### What It's Missing

3. **Catalog Edge Level:** ❌ NOT IMPLEMENTED
   - Does NOT query explicit `catalog_edge` entries that represent FK relationships
   - Does NOT use `relationship_type = 'FOREIGN_KEY'` edges
   - Does NOT check for qualified_path matching patterns (e.g., `/public/order/customer_id`)

---

## The Gap

### Scenario: You Have a Catalog Edge Relationship

```
catalog_edge row:
{
  source_node_id: "order-customer_id-node",
  target_node_id: "customer-id-node",
  relationship_type: "FOREIGN_KEY",
  properties: { fk_column: "customer_id", pk_table: "customer", ... }
}
```

**Current Behavior:** Relationship suggestion logic does NOT query this!

**Expected:** Should suggest: "Order → Customer" relationship

---

## Why This Matters

When a user (or system) explicitly creates an edge in `catalog_edge` showing FK relationship:
- ✓ It means they've validated the relationship
- ✓ It's already been curated
- ✓ It should be automatically suggested as a valid relationship

But currently, the suggestion engine ignores these curated edges!

---

## Solution: Add Catalog Edge FK Query

The `getFKHints()` function needs an additional query to check for explicit FK relationships in `catalog_edge`:

```go
// NEW: Query explicit FK relationships from catalog_edge
catalogFKQuery := `
  SELECT DISTINCT
    cn_target.node_name as target_table,
    ce.properties->>'fk_column' as fk_column
  FROM catalog_edge ce
  JOIN catalog_node cn_source ON ce.source_node_id = cn_source.id
  JOIN catalog_node cn_target ON ce.target_node_id = cn_target.id
  WHERE ce.relationship_type = 'FOREIGN_KEY'
    AND ce.tenant_datasource_id = $1
    AND (
      -- Match if source node path contains the entity name
      cn_source.qualified_path LIKE '/public/' || $2 || '/%'
      OR cn_source.node_name = $2
    )
  UNION ALL
  SELECT DISTINCT
    cn_target.node_name as target_table,
    ce.properties->>'fk_column' as fk_column
  FROM catalog_edge ce
  JOIN catalog_node cn_source ON ce.source_node_id = cn_source.id
  JOIN catalog_node cn_target ON ce.target_node_id = cn_target.id
  WHERE ce.relationship_type = 'REFERENCE'
    AND ce.properties->>'edge_kind' = 'FOREIGN_KEY'
    AND ce.tenant_datasource_id = $1
    AND (
      cn_source.qualified_path LIKE '/public/' || $2 || '/%'
      OR cn_source.node_name = $2
    )
`
```

---

## Implementation Steps

### Step 1: Update `getFKHints()` Function

Add a third query loop that processes `catalog_edge` entries:

```go
func (s *RelationshipService) getFKHints(ctx context.Context, tenantID, datasourceID, entity string) ([]struct{ Target, FKColumn string }, error) {
  // ... existing information_schema query ...
  
  // ... existing semantic query ...
  
  // NEW: Query catalog_edge for explicit FK relationships
  catalogFKQuery := `
    SELECT DISTINCT
      cn_target.node_name as target_table,
      ce.properties->>'fk_column' as fk_column
    FROM catalog_edge ce
    JOIN catalog_node cn_source ON ce.source_node_id = cn_source.id
    JOIN catalog_node cn_target ON ce.target_node_id = cn_target.id
    WHERE (ce.relationship_type = 'FOREIGN_KEY' OR ce.relationship_type = 'reference')
      AND ce.tenant_datasource_id = $1
      AND (
        cn_source.qualified_path LIKE '/public/' || $2 || '/%'
        OR cn_source.node_name = $2
        OR cn_source.properties->>'entity_name' = $2
      )
  `
  
  catalogFKRows, err := s.db.QueryContext(ctx, catalogFKQuery, datasourceID, entity)
  if err != nil {
    // Log but don't fail - this is optional enhancement
    log.Printf("Warning: failed to query catalog_edge for FKs: %v", err)
  } else {
    defer catalogFKRows.Close()
    for catalogFKRows.Next() {
      var hint struct{ Target, FKColumn string }
      if err := catalogFKRows.Scan(&hint.Target, &hint.FKColumn); err != nil {
        continue
      }
      if hint.Target != "" && hint.Target != entity {
        hints = append(hints, hint)
      }
    }
  }
  
  return hints, nil
}
```

### Step 2: Test with Your Scenario

After implementing:
1. Create FK relationship edge in catalog_edge
2. Call `GetRelationshipSuggestions` for Order entity
3. Should return Customer as a suggested relationship with high confidence

---

## Affected Files

**File to modify:**
```
/backend/internal/api/relationship_suggestions.go
- Function: getFKHints (around line 280)
- Add: Catalog edge FK query section
```

---

## Expected Result

After fix:

```
GET /api/relationships/suggestions?entity=order

Response:
{
  "suggestions": [
    {
      "id": "...",
      "title": "Order to Customer",
      "sourceEntity": "order",
      "targetEntity": "customer",
      "edgeType": "FOREIGN_KEY",
      "confidence": 0.95,
      "reasoning": "Foreign key relationship found in catalog_edge (order.customer_id → customer.id)",
      "dismissible": true
    }
  ]
}
```

---

## About Subtypes (Additional Note)

You also mentioned: *"if I click on a sub type then the inherited fields are collapsed and the subtype fields are expanded"*

This is a separate UI behavior that affects how inherited fields appear when viewing subtypes. This would typically be handled in:
- `frontend/src/pages/TabbedModal/Catalog/EntityDetailsPanel.tsx` (or similar)
- Component state management for field visibility
- UI logic to distinguish inherited vs. subtype-specific fields

Would you like me to also investigate this subtype display behavior?

---

## Summary

**Issue:** Foreign key relationships explicitly recorded in `catalog_edge` are not being suggested by the relationship suggestion engine.

**Cause:** `getFKHints()` only checks database-level FKs and semantic term relationships, not catalog edges.

**Fix:** Add a third query to `getFKHints()` that explicitly looks for `FOREIGN_KEY` relationship types in `catalog_edge`.

**Impact:** Relationships already curated in the system will now be properly suggested to users.
