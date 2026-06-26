# Workday-Inspired Business Object Linking - Architecture Overview

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │         BusinessObjectDesigner                               │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │ Metadata │ Fields │ Related Objects │ (NEW PANEL)  │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  │                                                               │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │ RelatedObjectsPanel                                  │  │  │
│  │  │ ┌──────────────────────────────────────────────────┐ │  │  │
│  │  │ │ Links To        │ Links From                     │ │  │  │
│  │  │ ├──────────────────────────────────────────────────┤ │  │  │
│  │  │ │ • Employee→Dept │ • Company→Employee            │ │  │  │
│  │  │ │ • Emp→Manager   │ • Department→Employee         │ │  │  │
│  │  │ └──────────────────────────────────────────────────┘ │  │  │
│  │  │                                                        │  │  │
│  │  │ ┌─────────────────┐  ┌──────────────────────────────┐ │  │  │
│  │  │ │ AI Suggest BTN  │  │ Suggestion List (Top 5)     │ │  │  │
│  │  │ │ [Loading...]    │  │ ├─ Link A→B (85%)           │ │  │  │
│  │  │ └─────────────────┘  │ ├─ Link C→D (72%)           │ │  │  │
│  │  │                      │ └─ [Preview] [Dismiss]       │ │  │  │
│  │  │                      └──────────────────────────────┘ │  │  │
│  │  └──────────────────────────────────────────────────────┘  │  │
│  │                                                               │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │ SuggestionPreviewModal (Lazy-Loaded)               │  │  │
│  │  │ ┌──────────────────────────────────────────────────┐ │  │  │
│  │  │ │ Link Employee → Department                       │ │  │  │
│  │  │ ├──────────────────────────────────────────────────┤ │  │  │
│  │  │ │ Type: FOREIGN_KEY | Cardinality: N:1            │ │  │  │
│  │  │ │ FK Column: dept_id | Confidence: 85%            │ │  │  │
│  │  │ ├──────────────────────────────────────────────────┤ │  │  │
│  │  │ │ Reasoning: FK evidence + high join freq         │ │  │  │
│  │  │ │ ███████░░░░ 85% [Apply] [Cancel]               │ │  │  │
│  │  │ └──────────────────────────────────────────────────┘ │  │  │
│  │  └──────────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
                                 │
                    Apollo Client │ GraphQL
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    GraphQL API Layer                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Queries:                                                            │
│  • getRelatedObjects(tenantId, datasourceId, entity)               │
│  • getRelationshipSuggestions(tenantId, datasourceId, entity)      │
│  • getCatalogNode(tenantId, datasourceId, name)                    │
│  • getCatalogEdges(tenantId, datasourceId, nodeId, direction)      │
│                                                                      │
│  Mutations:                                                          │
│  • applyRelationship(...)  → CatalogEdge                           │
│  • dismissRelationshipSuggestion(...) → audit                      │
│  • createRelationshipEdge(...)                                     │
│  • updateRelationshipEdge(...)                                     │
│  • deleteRelationshipEdge(...)                                     │
│                                                                      │
│  Subscriptions:                                                      │
│  • relationshipEdgeUpdated(tenantId, datasourceId)                │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
                                 │
                             HTTP/REST
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                   Backend Service (Go)                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  RelationshipService                                                │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                                                               │  │
│  │ GetRelationshipSuggestions(ctx, tenantID, datasourceID)     │  │
│  │   ├─ Check cache (5 min TTL)                                │  │
│  │   ├─ getExistingEdges() → Deduplication                     │  │
│  │   ├─ getFKHints() → Extract from information_schema         │  │
│  │   ├─ For each hint:                                         │  │
│  │   │   ├─ computeConfidence()                                │  │
│  │   │   │   ├─ fkFeature (1.0 or 0.5)        [40% weight]    │  │
│  │   │   │   ├─ estimatedJoinFrequency()      [20% weight]    │  │
│  │   │   │   ├─ stringSimilarity()            [15% weight]    │  │
│  │   │   │   ├─ semanticSimilarity()          [15% weight]    │  │
│  │   │   │   ├─ edgePrior (0.8)               [10% weight]    │  │
│  │   │   │   └─ Score: [0.0, 1.0]                            │  │
│  │   │   ├─ inferEdgeTypeAndCardinality()                      │  │
│  │   │   └─ Create RelationshipSuggestion                      │  │
│  │   ├─ Sort by confidence DESC                                │  │
│  │   ├─ Limit to N (default 5)                                │  │
│  │   ├─ Cache result                                           │  │
│  │   └─ Return []RelationshipSuggestion                        │  │
│  │                                                               │  │
│  │ Scoring Weights:                                            │  │
│  │ ┌─────────────────────────────────────┐                    │  │
│  │ │ FK Evidence        40% ███████░░░░░░ │                    │  │
│  │ │ Join Frequency     20% ████░░░░░░░░░ │                    │  │
│  │ │ Name Similarity    15% ███░░░░░░░░░░ │                    │  │
│  │ │ Text Similarity    15% ███░░░░░░░░░░ │                    │  │
│  │ │ Edge Prior         10% ██░░░░░░░░░░░ │                    │  │
│  │ └─────────────────────────────────────┘                    │  │
│  │                                                               │  │
│  │ Performance:                                                │  │
│  │ • Cached: <100ms     • Uncached: 100-200ms                 │  │
│  │ • Memory: 10-50MB    • Cache Hit Rate: 80%+                │  │
│  │                                                               │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Multi-Tenant Context (Required):                                   │
│  • Header: X-Tenant-ID, X-Tenant-Datasource-ID                    │
│  • Query: ?tenant_id=...&datasource_id=...                        │
│  • Enforced on every request                                       │
│                                                                      │
│  In-Memory Cache:                                                   │
│  • Type: sync.Map                                                  │
│  • Key: "tenant_id:datasource_id:entity"                          │
│  • TTL: 5 minutes                                                  │
│  • Eviction: Automatic on expiry                                   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
                                 │
                            SQL Queries
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                  PostgreSQL Database                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  catalog_node (~ 1K-10K rows per tenant)                           │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │ id (UUID) │ tenant_id │ datasource_id │ name │ kind │ ...  │  │
│  ├─────────────────────────────────────────────────────────────┤  │
│  │ UUID-1    │ tenant-A  │ datasource-1  │ Emp  │ bo   │      │  │
│  │ UUID-2    │ tenant-A  │ datasource-1  │ Dept │ view │      │  │
│  │ UUID-3    │ tenant-B  │ datasource-2  │ Ord  │ table│      │  │
│  └─────────────────────────────────────────────────────────────┘  │
│  Index: (tenant_id, datasource_id, name)                           │
│                                                                      │
│  catalog_edge_types (4 rows - controlled vocabulary)                │
│  ┌────────────────┬──────────────────┐                            │
│  │ code           │ label            │                            │
│  ├────────────────┼──────────────────┤                            │
│  │ REFERENCE      │ Reference        │                            │
│  │ COMPOSITION    │ Composition      │                            │
│  │ ASSOCIATION    │ Association      │                            │
│  │ FOREIGN_KEY    │ Foreign Key      │                            │
│  └────────────────┴──────────────────┘                            │
│                                                                      │
│  catalog_edge (~ 5K-50K rows per tenant)                           │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ id │ tenant │ datasource │ source_id │ target_id │ type   │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │ E1 │ t-A    │ ds-1       │ UUID-1    │ UUID-2    │ FK     │  │
│  │ E2 │ t-A    │ ds-1       │ UUID-2    │ UUID-3    │ COMP   │  │
│  │ E3 │ t-B    │ ds-2       │ ...       │ ...       │ ...    │  │
│  └────────────────────────────────────────────────────────────┘  │
│  │ confidence │ cardinality │ fk_column │ suggested │ created_by │
│  ├────────────┼─────────────┼───────────┼───────────┼────────────┤  │
│  │ 0.85       │ N:1         │ dept_id   │ true      │ user-123   │  │
│  │ 1.0        │ 1:N         │ NULL      │ false     │ system     │  │
│  │ 0.72       │ N:N         │ semantic  │ true      │ user-456   │  │
│  └────────────┴─────────────┴───────────┴───────────┴────────────┘  │
│  Indexes:                                                            │
│  • (tenant_id, datasource_id, source_id)                            │
│  • (tenant_id, datasource_id, target_id)                            │
│  • (tenant_id, datasource_id, edge_type)                            │
│  • UNIQUE (tenant_id, datasource_id, source_id, target_id, type)   │
│                                                                      │
│  relationship_suggestion_audit (Append-only audit log)             │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │ id │ tenant │ entity │ target_entity │ edge_type │ action   │ │
│  ├──────────────────────────────────────────────────────────────┤ │
│  │ A1 │ t-A    │ Emp    │ Dept          │ FK        │ accepted │ │
│  │ A2 │ t-A    │ Ord    │ Item          │ COMP      │ dismissed│ │
│  │ A3 │ t-B    │ ...    │ ...           │ ...       │ ...      │ │
│  └──────────────────────────────────────────────────────────────┘ │
│  │ confidence │ reason │ acted_by │ acted_at                     │  │
│  ├────────────┼────────┼──────────┼──────────────────────────────┤  │
│  │ 0.85       │ NULL   │ user-1   │ 2025-10-24 14:30:00+00      │  │
│  │ 0.72       │ "dup"  │ user-2   │ 2025-10-24 14:31:00+00      │  │
│  │ 0.68       │ NULL   │ system   │ 2025-10-24 14:32:00+00      │  │
│  └────────────┴────────┴──────────┴──────────────────────────────┘ │
│                                                                      │
│  Triggers:                                                           │
│  • audit_suggested_edge: AUTO-INSERT to audit on catalog_edge      │
│    (when suggested = true)                                          │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## Data Flow

### 1. User Views Related Objects Panel
```
User Opens BusinessObjectDesigner
  ↓
RelatedObjectsPanel mounts
  ↓
useQuery(GET_RELATED_OBJECTS) fires
  ↓
GraphQL Query → Backend
  ↓
Database: SELECT * FROM catalog_edge WHERE source_id = ?
  ↓
Response: Array of related objects
  ↓
Display in two columns (Outbound/Inbound)
```

### 2. User Clicks "AI Suggest"
```
Click "AI Suggest" button
  ↓
Lazy-load SuggestionPreviewModal component
  ↓
Fetch: GET /api/relationship-suggestions?entity=...
  ↓
Backend RelationshipService.GetRelationshipSuggestions()
  ↓
Check Cache:
  - HIT: Return cached results (80%+ of time)
  - MISS: Continue below
  ↓
Query FK hints from information_schema
  ↓
Filter existing edges (deduplication)
  ↓
Score each candidate (weighted model)
  ↓
Sort by confidence
  ↓
Limit to top 5
  ↓
Cache result (5 min TTL)
  ↓
Return to frontend
  ↓
Display suggestion list with confidence badges
```

### 3. User Previews & Applies Suggestion
```
Click "Preview" on suggestion
  ↓
SuggestionPreviewModal opens (animated slide-in)
  ↓
Display:
  - Source/Target entities
  - Edge type, cardinality, FK column
  - Confidence bar (color-coded)
  - Reasoning explanation
  ↓
User clicks "Apply Relationship"
  ↓
Mutation: applyRelationship(...)
  ↓
Backend: INSERT INTO catalog_edge WITH:
  - source_id, target_id
  - edge_type, cardinality
  - confidence, suggested=true
  - created_by, created_at
  ↓
Trigger fires: audit_suggested_edge()
  ↓
INSERT INTO relationship_suggestion_audit (accepted)
  ↓
Invalidate cache
  ↓
refetch() queries
  ↓
Modal closes
  ↓
Display refreshed relationships in panel
```

## Tenant Scoping Security Model

```
Request arrives at API
  ↓
Check for tenant context (headers or query params)
  ├─ Missing? → Error 401 "Tenant context required"
  └─ Present? → Continue
  ↓
Validate tenant_id in context
  ├─ Invalid? → Error 403 "Unauthorized"
  └─ Valid? → Continue
  ↓
Execute query with WHERE tenant_id = ?
  ↓
All results automatically scoped to tenant
  ↓
User can only see their own relationships
```

## Performance Characteristics

### Cache Performance
```
Scenario: 100 users, 50 requests/second

Without Cache:
- Each request: 100-200ms
- Database: 50 queries/sec × 100ms = 5000 QPS load
- Expensive!

With Cache (5 min TTL):
- First request: 100-200ms (MISS)
- Subsequent 299 requests: <10ms (HIT)
- Cache hit rate: 80%+ in typical usage
- Database: 50 / 60 = ~1 request/sec
- 5000x improvement!
```

### Database Query Performance
```
catalog_edge lookup:
- With indexes: 10-50ms (1K-50K rows)
- Without indexes: 500ms+ (scan entire table)

FK detection (information_schema):
- Typical: 20-50ms (cached by Postgres)

Scoring model:
- String similarity: <1ms
- FK evidence: <1ms
- Join frequency: 20-50ms (DB query)
- Total per candidate: ~30ms for top 5

End-to-end (uncached): 100-200ms ✓
End-to-end (cached): <10ms ✓
```

## Scaling Considerations

### Horizontal Scaling
- Backend is stateless (cache is per-instance)
- Share PostgreSQL across instances
- Use distributed cache (Redis) if needed

### Vertical Scaling
- Increase instance memory for larger caches
- Optimize indexes for DB queries
- Consider partitioning edges by tenant

### Monitoring
- Cache hit/miss ratio → Model scoring quality
- Average response time → Performance health
- Suggestion acceptance rate → Model accuracy
- DB query time → Index effectiveness

---

This architecture provides a scalable, secure, multi-tenant system for discovering and managing business object relationships in Semlayer.
