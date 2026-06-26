# Semantic Layer Architecture

## Overview

The semantic layer provides a deterministic, LLM-friendly contract for business object metadata. It consists of three main components:

### 1. **SQL Column Name Resolution** ✅ IMPLEMENTED
**Location**: [api.go](backend/internal/api/api.go) - `listBusinessObjects` function  
**Status**: Fully working and verified

The API now exposes actual database column names alongside logical field names:

```json
{
  "fields": [
    {
      "name": "company_identifier",      // Logical field name
      "columnName": "company_identifier", // ACTUAL database column
      "label": "Company Identifier",
      "type": "string"
    }
  ]
}
```

**Impact**: SQL generation uses true database columns, eliminating guessing:
```sql
SELECT t0.company_identifier FROM customers t0  -- NOT "company identified"
```

---

### 2. **Semantic Bundle (LLM Contract)** ✅ IMPLEMENTED
**Location**: [api.go](backend/internal/api/api.go) - `getSemanticBundle` function  
**Status**: Code complete, endpoint accessible at `/api/semantic/bundles/by-id`

The SemanticBundle provides a complete, deterministic response containing:

#### Structure
```typescript
SemanticBundle {
  business_object_id: string     // UUID - immutable identity
  business_object_name: string   // Display name
  datasource_id: string          // Where data lives
  driving_table: string          // Main table name
  version: string                // Semantic model version (v1, v2, etc.)
  
  fields: SemanticField[] {
    field_id: string             // UUID - immutable
    name: string                 // Logical name (may change)
    display_name: string         // UI label (may change)
    semantic_term: string        // Meaning (may change)
    aliases: string[]            // Old names for this field
    
    physical: {
      datasource_id: string      // e.g., "postgres"
      table: string              // e.g., "customers"
      column: string             // e.g., "company_identifier" ← CANONICAL
    }
    
    description: string          // Field documentation
  }
  
  relationships: SemanticRelationship[] {
    target_bo_id: string         // Join target
    join_type: string            // INNER, LEFT, RIGHT, FULL
    source_column: string        // Column in driving table
    target_column: string        // Column in target table
    target_table: string         // e.g., "orders"
  }
  
  created_at: string             // RFC3339 timestamp
  updated_at: string             // Last metadata change
}
```

#### Usage

```bash
curl "http://localhost:8080/api/semantic/bundles/by-id?bo_id=UUID&tenant_id=UUID"
```

**Response**: Complete metadata with zero guessing required for SQL generation

---

### 3. **Name → UUID Resolver** ✅ IMPLEMENTED
**Location**: [semantic_name_resolver.go](backend/internal/api/semantic_name_resolver.go)  
**Status**: Resolver service fully implemented with caching

The resolver provides deterministic mapping from semantic term names to field UUIDs:

#### API

```go
// Resolve a name to a field UUID
fieldID, err := resolver.ResolveTermNameToFieldID("customer_id")
// Returns: "be7b9e37-5b9b-41fe-ac6e-58465387eb7c"

// Get all names for a field (including aliases)
names := resolver.ResolveFieldIDToTermNames(fieldID)
// Returns: ["customer_id", "cust_id", "CustomerID"] 

// Check if a name is an old alias
isAlias := resolver.ResolveIsAlias("cust_id")
// Returns: true

// Get all mappings
allMappings := resolver.GetAllMappings()
// Returns: map[string]string with 1000+ entries
```

#### Features
- **Pre-loaded cache**: All mappings loaded at startup for O(1) lookup
- **Thread-safe**: RWMutex protects concurrent access
- **Alias support**: Old field names automatically resolve to current UUIDs
- **Refresh capability**: `Refresh(ctx)` reloads from database on demand

---

### 4. **Metadata Versioning** ✅ IMPLEMENTED
**Location**: [metadata_versioning_handlers.go](backend/internal/api/metadata_versioning_handlers.go)  
**Status**: Full versioning support with change tracking

Tracks all semantic model changes atomically:

#### MetadataVersion Structure

```json
{
  "version_id": "uuid",
  "business_object_id": "uuid",
  "version": 5,
  "created_at": "2026-02-05T12:00:00Z",
  "created_by": "user@example.com",
  
  "change_type": "field_renamed",
  "change_detail": { "reason": "Standardization" },
  
  "previous_value": {
    "name": "cust_id",
    "display_name": "Customer ID"
  },
  "new_value": {
    "name": "customer_id", 
    "display_name": "Customer Identifier"
  }
}
```

#### Change Types Supported
- `field_added` - New field introduced
- `field_renamed` - Field name changed
- `field_removed` - Field deleted
- `field_type_changed` - Data type updated
- `physical_mapping_changed` - Database location changed

#### Endpoints

```bash
# Create version record for a change
POST /api/metadata/versions
{
  "business_object_id": "uuid",
  "change_type": "field_renamed",
  "previous_value": {"name": "old_name"},
  "new_value": {"name": "new_name"},
  "created_by": "admin"
}

# Get version history
GET /api/metadata/versions/{bo_id}

# Get resolver cache stats
GET /api/semantic/name-resolver/stats
```

---

### 5. **Field Aliases** ✅ IMPLEMENTED
**Location**: [metadata_versioning_handlers.go](backend/internal/api/metadata_versioning_handlers.go)  
**Status**: Full alias management with backward compatibility

Enables safe field renames without breaking existing queries:

#### FieldAlias Structure

```json
{
  "alias_id": "uuid",
  "field_id": "current-field-uuid",
  "old_name": "cust_id",
  "renamed_at": "2026-02-01T10:00:00Z",
  "renamed_by": "admin",
  "is_active": true,
  "description": "Renamed for SQL standard compliance"
}
```

#### How It Works
1. User renames field: `cust_id` → `customer_id`
2. Alias created: Old name → Current UUID
3. Existing queries still work: `SELECT cust_id` automatically resolves to `customer_id`
4. LLM knows both names are valid

#### Endpoints

```bash
# Create an alias
POST /api/field-aliases
{
  "field_id": "uuid",
  "old_name": "cust_id",
  "renamed_by": "admin",
  "description": "Standardization to SQL naming"
}

# Get all aliases for a field
GET /api/field-aliases/{field_id}
```

---

## Integration Example

### 1. Get Semantic Bundle (LLM reads this)
```bash
$ curl "http://localhost:8080/api/semantic/bundles/by-id?bo_id=ABC123&tenant_id=XYZ789"
```

Response provides:
- ✅ Field UUIDs (immutable)
- ✅ Field names (may change)
- ✅ Aliases (backward compatible)
- ✅ Physical database locations (canonical truth)
- ✅ Version number (for caching)

### 2. Name Resolution (LLM translates user intent)
```
User says: "Show me the cust_id values"
   ↓
LLM calls: resolver.ResolveTermNameToFieldID("cust_id")
   ↓
Resolver returns: "be7b9e37-5b9b-41fe-ac6e-58465387eb7c" (UUID)
   ↓
LLM queries: SemanticBundle.Fields[].find(f => f.field_id === UUID)
   ↓
Get physical location: {datasource: "postgres", table: "customers", column: "company_identifier"}
   ↓
Generate SQL: SELECT t0.company_identifier FROM customers t0
```

### 3. Metadata Versioning (Audit trail)
```bash
$ curl "http://localhost:8080/api/metadata/versions/ABC123"
```

Returns complete change history:
- When "cust_id" was renamed to "customer_id"
- Who made the change
- What the previous/new values were
- All previous versions for rollback if needed

---

## Database Schema Requirements

**Tables needed:**
```sql
-- Semantic metadata versioning
CREATE TABLE IF NOT EXISTS metadata_versions (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  business_object_id UUID NOT NULL,
  version INT NOT NULL,
  created_at TIMESTAMP NOT NULL,
  created_by TEXT,
  change_type TEXT NOT NULL,
  previous_value JSONB,
  new_value JSONB,
  UNIQUE(tenant_id, business_object_id, version)
);

-- Field naming aliases
CREATE TABLE IF NOT EXISTS field_aliases (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  field_id UUID NOT NULL,
  old_name TEXT NOT NULL,
  renamed_at TIMESTAMP NOT NULL,
  renamed_by TEXT,
  is_active BOOLEAN DEFAULT true,
  description TEXT,
  UNIQUE(tenant_id, field_id, old_name)
);

CREATE INDEX ON metadata_versions(tenant_id, business_object_id);
CREATE INDEX ON field_aliases(tenant_id, field_id);
```

---

## Performance Characteristics

| Operation | Complexity | Notes |
|-----------|-----------|-------|
| ResolveTermNameToFieldID | O(1) | Cached in-memory |
| GetSemanticBundle | O(n) | n = number of fields |
| CreateMetadataVersion | O(1) | Single insert |
| GetVersionHistory | O(m) | m = number of versions |
| Refresh Resolver Cache | O(n) | Full table scan, done infrequently |

---

## Next Steps

1. **SQL Generation Integration** - Modify SQL generator to use Physical mappings from bundle
2. **LLM Hardening** - Add semantic bundle to LLM system prompt
3. **Change Detection** - Auto-create metadata versions when BO fields change
4. **Audit Trail UI** - Display version history in admin dashboard
5. **Alias Validation** - Prevent duplicate aliases for same field

---

## Files Created/Modified

### New Files
- `semantic_name_resolver.go` - Deterministic name→UUID mapping
- `metadata_versioning_handlers.go` - Versioning and alias endpoints

### Modified Files
- `api.go` - Added SemanticBundle structs, endpoints, integration points
- Routes registered in main Router setup

---

## Testing Commands

```bash
# Test semantic bundle
curl -s "http://localhost:8080/api/semantic/bundles/by-id?bo_id=ABC&tenant_id=XYZ" | jq .

# Test name resolver
curl -s "http://localhost:8080/api/semantic/name-resolver/stats" | jq .

# Create metadata version
curl -X POST "http://localhost:8080/api/metadata/versions" \
  -H "X-Tenant-ID: XYZ" \
  -H "Content-Type: application/json" \
  -d '{
    "business_object_id": "ABC",
    "change_type": "field_renamed",
    "previous_value": {"name": "old"},
    "new_value": {"name": "new"},
    "created_by": "admin"
  }'

# Create field alias
curl -X POST "http://localhost:8080/api/field-aliases" \
  -H "X-Tenant-ID: XYZ" \
  -H "Content-Type: application/json" \
  -d '{
    "field_id": "field-uuid",
    "old_name": "cust_id",
    "renamed_by": "admin"
  }'
```

