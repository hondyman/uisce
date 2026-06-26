# Validation Rules: Lazy Loading & Faceted Search Architecture

## System Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                         USER BROWSER                                 │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │        ValidationRulesWithFacets Component (React)             │ │
│  │                                                                │ │
│  │  ┌─────────────────┐  ┌──────────────────────────────────┐   │ │
│  │  │  SIDEBAR        │  │  MAIN CONTENT                    │   │ │
│  │  │  (240px)        │  │                                  │   │ │
│  │  │                 │  │  Search Bar                      │   │ │
│  │  │ ☑ Entities     │  │  [🔍 Search...]                 │   │ │
│  │  │   • Order (45)  │  │                                  │   │ │
│  │  │   • Customer    │  │  ┌──────────────────────────────┐│   │ │
│  │  │                 │  │  │ Rule 1                       ││   │ │
│  │  │ ☑ Rule Types   │  │  ├──────────────────────────────┤│   │ │
│  │  │                 │  │  │ Rule 2                       ││   │ │
│  │  │ ☐ Sub-Entity   │  │  │ ...                          ││   │ │
│  │  │                 │  │  │ Rule 20                      ││   │ │
│  │  │ ☐ Severity     │  │  └──────────────────────────────┘│   │ │
│  │  │                 │  │  [⏬ Load More]                  │   │ │
│  │  │ [Clear All]    │  │                                  │   │ │
│  │  └─────────────────┘  └──────────────────────────────────┘   │ │
│  │                                                                │ │
│  │  State Management:                                             │ │
│  │  • FilterState (entities, sub_entities, types, severities)    │ │
│  │  • Rules (current page)                                       │ │
│  │  • Facets (live counts)                                       │ │
│  │  • Pagination (page, hasMore, total)                          │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                              ↓                                       │
│                   HTTP GET Request (params)                          │
│                              ↓                                       │
└──────────────────────────────────────────────────────────────────────┘
                               │
                               │ JSON Response (6-10 KB)
                               │
┌──────────────────────────────────────────────────────────────────────┐
│                         BACKEND (Go)                                 │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │         ListValidationRulesHandler()                          │ │
│  │                                                               │ │
│  │  Parse Query Params:                                         │ │
│  │  • page, limit                                               │ │
│  │  • entities, sub_entities, rule_types, severities            │ │
│  │  • search, tenant_id, datasource_id                          │ │
│  │                                                               │ │
│  │  Build WHERE Clause:                                         │ │
│  │  • Dynamic WHERE construction                                │ │
│  │  • Parameterized queries (SQL injection safe)                │ │
│  │  • Full-text search condition                                │ │
│  │                                                               │ │
│  │  Execute Queries:                                            │ │
│  │  ├─ Count total rules (with filters)                         │ │
│  │  ├─ Fetch paginated rules                                    │ │
│  │  └─ Calculate facet counts                                   │ │
│  │                                                               │ │
│  │  Build Response:                                             │ │
│  │  • Rules array (20 items)                                    │ │
│  │  • Pagination metadata (total, page, has_more)               │ │
│  │  • Facet options with counts                                 │ │
│  │                                                               │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                              ↓                                       │
│                      SQL Queries (3x)                                │
│                              ↓                                       │
└──────────────────────────────────────────────────────────────────────┘
                               │
                               │ PostgreSQL
                               │
┌──────────────────────────────────────────────────────────────────────┐
│                     DATABASE (PostgreSQL)                            │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Table: validation_rules (1,608 rows)                               │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ Columns:                                                     │  │
│  │ • id (PK)                                                   │  │
│  │ • rule_name (indexed: GIN full-text)                        │  │
│  │ • target_entity (indexed)  ← Facet 1                        │  │
│  │ • sub_entity_type (indexed) ← Facet 2                       │  │
│  │ • rule_type (indexed) ← Facet 3                             │  │
│  │ • severity (indexed) ← Facet 4                              │  │
│  │ • description (indexed: GIN full-text)                      │  │
│  │ • condition (JSON)                                          │  │
│  │ • is_active (indexed in composite)                          │  │
│  │ • tenant_id, datasource_id (indexed composite)              │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Indexes:                                                            │
│  ✓ idx_rules_entity                                                 │
│  ✓ idx_rules_sub_entity                                             │
│  ✓ idx_rules_type                                                   │
│  ✓ idx_rules_severity                                               │
│  ✓ idx_rules_entity_type_active                                     │
│  ✓ idx_rules_name_search (GIN)                                      │
│  ✓ idx_rules_tenant_scope                                           │
│                                                                      │
│  Query Types:                                                        │
│  1. Count Query: SELECT COUNT(*) FROM validation_rules WHERE ...    │
│  2. Rules Query: SELECT ... FROM validation_rules WHERE ... LIMIT  │
│  3. Facet Queries: SELECT target_entity, COUNT(*) FROM validation  │
│                    rules WHERE ... GROUP BY ...                     │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

## Data Flow Sequence

```
User Action                Component                Backend              Database
─────────────              ─────────────            ───────              ────────

Select "Order"  
Entity ──────────►  Update FilterState
                    (add "Order" to entities)
                                ──────────────────►  Build WHERE clause
                                                     Execute 4 queries:
                                                     ├─ Count total
                                                     ├─ Fetch rules
                                                     ├─ Entity facets
                                                     └─ Other facets
                                                                    ◄────── Read indexes
                                                     Return response
                                ◄──────────────────  (rules + facets)
                    
                    Update UI:
                    • Show rules
                    • Update facet counts
                    • Show "Load More"


Click "Load More"  ───────►  Increment page
                            Fetch next batch
                                ──────────────────►  Execute queries
                                                     (same as above)
                                                                    ◄────── Read indexes
                                                     Return next 20 rules
                                ◄──────────────────
                    
                    Append rules to list
                    Update "Load More" button


Type "discount"   ───────►  Update searchQuery
                           Debounce 300ms
                           Fetch with search param
                                ──────────────────►  Add full-text WHERE
                                                     Execute queries with
                                                     plainto_tsquery()
                                                                    ◄────── Search index
                                                     Return filtered results
                                ◄──────────────────
                    
                    Show search results
                    Update facet counts
```

## Component State Flow

```
┌─────────────────────────────────────────────────┐
│         FilterState (User Input)                │
│                                                 │
│ selectedEntities: []                            │
│ selectedSubEntities: []                         │
│ selectedRuleTypes: []                           │
│ selectedSeverities: []                          │
│ searchQuery: ""                                 │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │ buildFilterQuery()   │
        │ (construct params)   │
        └──────────────────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │ fetchRules()         │
        │ (API call)           │
        └──────────────────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │ API Response         │
        │ • rules[]            │
        │ • total              │
        │ • has_more           │
        │ • *_facets[]         │
        └──────────────────────┘
                   │
                   ▼
    ┌──────────────────────────────┐
    │ Update Component State:      │
    │ • setRules()                 │
    │ • setFacets()                │
    │ • setHasMore()               │
    │ • setTotalCount()            │
    │ • setPage()                  │
    └──────────────────────────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │ Re-render UI         │
        │ with new data        │
        └──────────────────────┘
```

## Facet Calculation Flow

```
Query Parameters:
  entities: ["Order"]
  rule_types: ["business_logic"]
  (other facets: none selected)

                    ↓

Build Base WHERE:
  WHERE tenant_id = ? AND datasource_id = ?
  AND target_entity IN ('Order')
  AND rule_type IN ('business_logic')

                    ↓
        
Calculate Each Facet:

┌─────────────────────────────────┐
│ Entity Facets:                  │
│ SELECT target_entity,           │
│        COUNT(*)                 │
│ FROM validation_rules           │
│ WHERE [base WHERE]              │ ◄─ excludes entities filter
│       AND rule_type = 'business'│
│ GROUP BY target_entity          │
│                                 │
│ Result:                         │
│ Order: 25 (matching rules)      │
│ Customer: 10                    │
│ Product: 8                      │
└─────────────────────────────────┘

┌─────────────────────────────────┐
│ Rule Type Facets:               │
│ SELECT rule_type,               │
│        COUNT(*)                 │
│ FROM validation_rules           │
│ WHERE [base WHERE]              │ ◄─ excludes rule_types filter
│       AND target_entity = 'Order'
│ GROUP BY rule_type              │
│                                 │
│ Result:                         │
│ business_logic: 25              │
│ field_format: 15                │
│ cardinality: 10                 │
└─────────────────────────────────┘

(Similar for other facets)

                    ↓

Response: Facets with current counts reflecting
remaining filters (helps user understand impact)
```

## API Request/Response Example

### Request
```http
GET /api/validation-rules?
  page=1&
  limit=20&
  entities=Order,Customer&
  rule_types=business_logic&
  severities=error&
  search=total&
  tenant_id=00000000-0000-0000-0000-000000000000&
  datasource_id=11111111-1111-1111-1111-111111111111
```

### Response
```json
{
  "rules": [
    {
      "id": "rule-123",
      "rule_name": "Order Total Positive",
      "rule_type": "business_logic",
      "target_entity": "Order",
      "sub_entity_type": "Order.Items",
      "severity": "error",
      "description": "Order total must be greater than 0",
      "condition": {"field": "total", "operator": ">", "value": 0},
      "is_active": true,
      "created_at": "2025-10-19T10:30:00Z"
    },
    ...19 more rules
  ],
  "total": 42,
  "page": 1,
  "limit": 20,
  "has_more": true,
  "entity_facets": [
    {"value": "Order", "label": "Order", "count": 25},
    {"value": "Customer", "label": "Customer", "count": 12},
    {"value": "Product", "label": "Product", "count": 5}
  ],
  "sub_entity_facets": [
    {"value": "Order.Items", "label": "Order Items", "count": 18},
    {"value": "Order.LineItems", "label": "Line Items", "count": 7}
  ],
  "rule_type_facets": [
    {"value": "business_logic", "label": "Business Logic", "count": 25},
    {"value": "field_format", "label": "Field Format", "count": 12},
    {"value": "cardinality", "label": "Cardinality", "count": 5}
  ],
  "severity_facets": [
    {"value": "error", "label": "Error", "count": 35},
    {"value": "warning", "label": "Warning", "count": 7}
  ]
}
```

## Performance Timeline

```
User selects "Order" entity:

T0:   Checkbox clicked ──────┐
T0:   Filter updated         │
T0:   API request sent       │
      ↓                      │
T50:  Database query executed
      ├─ Parse WHERE clause  │ Query: 2-3ms
      ├─ Use indexes         │ Facet: 0.5ms
      └─ Execute 4 queries   │ Total: ~5ms
      ↓                      │
T60:  API response received  │ Network: ~60ms latency
      ├─ 20 rules            │ Response: 6 KB
      ├─ Facets              │
      └─ Metadata            │
      ↓                      │
T70:  React re-renders       │ Render: ~10ms
      ├─ Update state        │
      ├─ DOM updates         │
      └─ Visual feedback     │
      ↓                      │
T70:  UI shows results ──────┘ Total: ~70ms from click

(Debounced search adds 300ms before API call)
```

---

**Architecture Design**: October 20, 2025  
**Phase**: 6.4 Post-Deployment Monitoring  
**Status**: ✅ Complete
