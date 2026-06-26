# Relationship Discovery Modal - Complete API Specification

## Quick Reference

### Three Core Endpoints

```
POST /api/relationships/existing
POST /api/relationships/discover
POST /api/relationships/apply
```

All require tenant context headers:
```
X-Tenant-ID: <tenant_uuid>
X-Tenant-Datasource-ID: <datasource_uuid>
```

---

## Endpoint 1: Fetch Existing Relationships

### POST /api/relationships/existing

**Purpose**: Retrieve already-established relationships for an entity.

**Request**:
```json
{
  "entity_attribute_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response** (200 OK):
```json
{
  "existing_relationships": [
    {
      "entity_id": "550e8400-e29b-41d4-a716-446655440001",
      "entity_name": "Customer",
      "table_name": "customers",
      "link_type": "DIRECT_FK",
      "cardinality": "1:N",
      "confidence": 1.0,
      "confidence_reason": "Established relationship",
      "foreign_key_path": "orders.customer_id -> customers.id",
      "semantic_term_name": null,
      "discovered_at": "2025-11-12T10:30:00Z"
    }
  ]
}
```

**Error Responses**:
- 400 Bad Request: Missing entity_attribute_id
- 400 Bad Request: Missing tenant context headers
- 500 Internal Server Error: Database query failed

**Implementation**:
- File: `backend/internal/api/relationship_api_handlers.go`
- Function: `postGetExistingRelationships()`
- Queries: `business_object_relationships` table
- Filters: `is_user_applied = true` AND `tenant_id` matches

---

## Endpoint 2: Discover Relationships

### POST /api/relationships/discover

**Purpose**: Find direct and multi-hop relationships based on database schema.

**Request**:
```json
{
  "entity_attribute_id": "550e8400-e29b-41d4-a716-446655440000",
  "include_multi_hop": true,
  "max_hop_depth": 3
}
```

**Request Fields**:
| Field | Type | Required | Default | Notes |
|-------|------|----------|---------|-------|
| entity_attribute_id | string (UUID) | Yes | - | Entity to discover relationships for |
| include_multi_hop | boolean | No | true | Include multi-hop paths |
| max_hop_depth | integer | No | 3 | Max hops (1-5) |

**Response** (200 OK):
```json
{
  "entity_attribute_id": "550e8400-e29b-41d4-a716-446655440000",
  "direct_relationships": [
    {
      "entity_id": "550e8400-e29b-41d4-a716-446655440001",
      "entity_name": "Customer",
      "table_name": "customers",
      "link_type": "DIRECT_FK",
      "cardinality": "1:N",
      "confidence": 0.95,
      "confidence_reason": "Foreign key constraint detected",
      "foreign_key_path": "orders.customer_id -> customers.id",
      "semantic_term_name": null,
      "entity_key": "customer",
      "source_column": "customer_id",
      "source_table": "orders",
      "target_column": "id",
      "target_table": "customers",
      "hierarchy_depth": 1,
      "relationship_path": null,
      "discovery_method": "FK_SCAN",
      "discovered_at": "2025-11-12T10:30:00Z"
    }
  ],
  "multi_hop_paths": [
    {
      "path_id": "path-uuid",
      "source_entity_id": "550e8400-e29b-41d4-a716-446655440000",
      "target_entity_id": "550e8400-e29b-41d4-a716-446655440002",
      "hierarchy_depth": 2,
      "hops": [
        {
          "order": 1,
          "entity_id": "550e8400-e29b-41d4-a716-446655440001",
          "entity_name": "Customer",
          "semantic_term_name": null,
          "link_type": "DIRECT_FK",
          "source_column": "customer_id",
          "target_column": "id",
          "foreign_key_path": "orders.customer_id -> customers.id",
          "cardinality": "1:N"
        },
        {
          "order": 2,
          "entity_id": "550e8400-e29b-41d4-a716-446655440002",
          "entity_name": "Region",
          "semantic_term_name": null,
          "link_type": "DIRECT_FK",
          "source_column": "region_id",
          "target_column": "id",
          "foreign_key_path": "customers.region_id -> regions.id",
          "cardinality": "N:1"
        }
      ],
      "total_confidence": 0.90,
      "total_cardinality": "N:M",
      "entities": [
        {
          "order": 0,
          "entity_id": "550e8400-e29b-41d4-a716-446655440000",
          "entity_name": "Orders",
          "semantic_term_name": null,
          "is_primary_key": true,
          "column_name": "id"
        }
      ]
    }
  ]
}
```

**Response Fields**:

**EnhancedRelatedEntity** (in direct_relationships):
| Field | Type | Notes |
|-------|------|-------|
| entity_id | string | UUID of related entity |
| entity_name | string | Display name |
| table_name | string | Database table name |
| link_type | enum | "DIRECT_FK", "SEMANTIC", or "MULTI_HOP" |
| cardinality | enum | "1:1", "1:N", "N:1", or "N:M" |
| confidence | float | 0.0 to 1.0, higher = more confident |
| confidence_reason | string | Human-readable explanation |
| foreign_key_path | string | "table1.col -> table2.col" |
| semantic_term_name | string or null | If linked to semantic term |
| discovery_method | string | "FK_SCAN", "SEMANTIC_MATCH", etc. |

**RelationshipPath** (in multi_hop_paths):
| Field | Type | Notes |
|-------|------|-------|
| path_id | string | Unique path identifier |
| source_entity_id | string | Starting entity UUID |
| target_entity_id | string | Ending entity UUID |
| hierarchy_depth | int | Number of hops |
| hops | array | PathHop objects |
| total_confidence | float | Combined confidence of all hops |
| total_cardinality | string | "1:1", "1:N", "N:1", or "N:M" |

**Error Responses**:
- 400 Bad Request: Missing entity_attribute_id
- 400 Bad Request: Missing tenant context
- 400 Bad Request: max_hop_depth > 5
- 500 Internal Server Error: Discovery failed

**Implementation**:
- File: `backend/internal/api/relationship_api_handlers.go`
- Function: `postDiscoverRelationships()`
- Services: `EnhancedRelationshipDiscoveryService`
- Fallback: Simple business object discovery if catalog unavailable

---

## Endpoint 3: Apply Relationship

### POST /api/relationships/apply

**Purpose**: Save a discovered relationship as an established link.

**Request**:
```json
{
  "sourceEntity": "550e8400-e29b-41d4-a716-446655440000",
  "targetEntity": "550e8400-e29b-41d4-a716-446655440001",
  "edgeType": "DIRECT_FK",
  "cardinality": "1:N",
  "confidence": 0.95,
  "foreignKeyPath": "orders.customer_id -> customers.id"
}
```

**Request Fields**:
| Field | Type | Required | Notes |
|-------|------|----------|-------|
| sourceEntity | string (UUID) | Yes | Source entity UUID |
| targetEntity | string (UUID) | Yes | Target entity UUID |
| edgeType | string | Yes | "DIRECT_FK", "SEMANTIC", "MULTI_HOP" |
| cardinality | string | Yes | "1:1", "1:N", "N:1", "N:M" |
| confidence | float | No | 0.0-1.0, defaults to 1.0 |
| foreignKeyPath | string | No | FK constraint details |

**Response** (200 OK):
```json
{
  "success": true,
  "message": "Relationship applied"
}
```

**Behavior**:
1. Updates `relationship_suggestions` table if suggestion exists (marks as accepted)
2. Creates or updates edge in `business_object_relationships` table
3. Sets `is_user_applied = true` and `user_applied_at = NOW()`
4. Uses ON CONFLICT to handle duplicates

**Error Responses**:
- 400 Bad Request: sourceEntity or targetEntity missing
- 400 Bad Request: Invalid UUID format
- 400 Bad Request: Missing tenant context
- 500 Internal Server Error: Database operation failed

**Implementation**:
- File: `backend/internal/api/relationships_chi.go`
- Function: `postApplyRelationship()`
- Creates edge in `business_object_relationships` table
- Updates suggestions if applicable

---

## Enum Values

### LinkType
```
"DIRECT_FK"     - Direct foreign key relationship from schema
"SEMANTIC"      - Semantic term based relationship
"MULTI_HOP"     - Multi-hop relationship through intermediate tables
"ASSOCIATION"   - Generic association (fallback)
```

### Cardinality
```
"1:1"  - One-to-one
"1:N"  - One-to-many
"N:1"  - Many-to-one
"N:M"  - Many-to-many
```

### DiscoveryMethod
```
"FK_SCAN"           - Found via foreign key scan
"SEMANTIC_MATCH"    - Matched via semantic terms
"PATTERN"           - Pattern-based matching
"MANUAL"            - Manually created by user
```

---

## HTTP Headers (Required)

All requests must include tenant context:

```
X-Tenant-ID: <tenant-uuid>
X-Tenant-Datasource-ID: <datasource-uuid>
Content-Type: application/json
```

### Optional: Query Parameters
Alternatively, can be passed as query params:
```
?tenant_id=<tenant-uuid>&datasource_id=<datasource-uuid>
```

The frontend fetch shim (setupTenantFetch.ts) automatically adds these.

---

## Error Response Format

All endpoints return errors in this format:

```json
{
  "message": "Human-readable error message",
  "error": "error_code",
  "details": "Additional details if available"
}
```

**Common Error Codes**:
- `invalid_request`: Malformed request body
- `missing_tenant`: Missing tenant context headers
- `discovery_error`: Relationship discovery failed
- `db_error`: Database operation failed
- `not_found`: Entity or relationship not found

---

## Pagination & Limits

### Result Limits:
- Direct relationships: Max 100 per query
- Multi-hop paths: Max 50 per query
- Existing relationships: Max 100 per query

### Query Limits:
- Timeout: 30 seconds per request
- max_hop_depth: 1-5 (enforced)
- max_results: Configurable per query

---

## Database Schema

### business_object_relationships Table
```sql
CREATE TABLE business_object_relationships (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  tenant_datasource_id UUID NOT NULL,
  source_object_id UUID NOT NULL,
  target_object_id UUID NOT NULL,
  relationship_type VARCHAR(50),
  cardinality VARCHAR(10),
  confidence FLOAT8 DEFAULT 1.0,
  is_user_applied BOOLEAN DEFAULT false,
  user_applied_at TIMESTAMP,
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  UNIQUE(tenant_id, source_object_id, target_object_id, relationship_type)
);
```

### relationship_suggestions Table
```sql
CREATE TABLE relationship_suggestions (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  source_entity_id UUID NOT NULL,
  target_entity_id UUID NOT NULL,
  confidence FLOAT8,
  rationale TEXT,
  accepted BOOLEAN DEFAULT false,
  accepted_at TIMESTAMP,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

---

## cURL Examples

### Fetch Existing Relationships
```bash
curl -X POST http://localhost:8080/api/relationships/existing \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_attribute_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

### Discover Relationships
```bash
curl -X POST http://localhost:8080/api/relationships/discover \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_attribute_id": "550e8400-e29b-41d4-a716-446655440000",
    "include_multi_hop": true,
    "max_hop_depth": 3
  }'
```

### Apply Relationship
```bash
curl -X POST http://localhost:8080/api/relationships/apply \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -H "Content-Type: application/json" \
  -d '{
    "sourceEntity": "550e8400-e29b-41d4-a716-446655440000",
    "targetEntity": "550e8400-e29b-41d4-a716-446655440001",
    "edgeType": "DIRECT_FK",
    "cardinality": "1:N",
    "confidence": 0.95,
    "foreignKeyPath": "orders.customer_id -> customers.id"
  }'
```

---

## TypeScript Interfaces

Modal component uses these interfaces:

```typescript
interface EnhancedRelatedEntity {
  entity_id: string;
  entity_name: string;
  table_name: string;
  link_type: 'DIRECT_FK' | 'SEMANTIC' | 'MULTI_HOP';
  cardinality: '1:1' | '1:N' | 'N:1' | 'N:M';
  confidence: number;
  confidence_reason: string;
  foreign_key_path: string;
  semantic_term_name?: string;
}

interface RelationshipPath {
  path_id: string;
  source_entity_id: string;
  target_entity_id: string;
  hierarchy_depth: number;
  hops: Array<{
    order: number;
    entity_id: string;
    entity_name: string;
    link_type: string;
    cardinality: string;
  }>;
  total_confidence: number;
  total_cardinality: string;
}
```

---

## Performance Notes

### Query Optimization Tips:
1. Ensure indexes on:
   - `business_object_relationships(tenant_id, source_object_id, is_user_applied)`
   - `business_objects(id, name)`
   - `catalog_edge(tenant_datasource_id, source_node_id, target_node_id)`

2. Consider caching for:
   - Multi-hop discovery results (5-min TTL)
   - Existing relationships (1-min TTL)

3. Pagination recommended for:
   - Entities with 100+ relationships
   - Complex schemas with deep FK chains

---

## Changelog

### Version 1.0 (Current)
- ✅ Discover direct relationships
- ✅ Discover multi-hop paths
- ✅ Apply relationships
- ✅ Fetch existing relationships (NEW)
- ✅ Tenant scoping
- ✅ Confidence scoring
- ✅ Cardinality detection

### Future Enhancements
- Relationship validation
- Batch operations
- Relationship history
- Cross-datasource linking
- Export functionality

---

**Last Updated**: 2025-11-12  
**Status**: ✅ Production Ready
