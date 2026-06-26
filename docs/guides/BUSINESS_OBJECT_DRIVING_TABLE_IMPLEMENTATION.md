# Business Object Layer Implementation - Driving Table Pattern

## Overview

This implementation provides a complete **Business Object definition layer** using the **driving table pattern** you outlined. It enables flexible, extensible modeling of complex business entities without requiring schema migrations.

## What Was Implemented

### 1. **Database Schema** (`backend/migrations/20251218_create_business_object_layer.sql`)

Core tables:

- **`business_object_def`** - The driving table
  - One row per BO type (Customer, Portfolio, IPS, etc.)
  - References a **driver table** from `catalog_node` (optional but recommended)
  - Stores metadata: name, display_name, description, status, config
  - Includes created_by, updated_by, timestamps for audit

- **`bo_subtype_def`** - Subtypes/variants (FIRST, before fields)
  - Models variant BO types (e.g., Account → {Taxable, IRA, Trust})
  - Each subtype has its own set of applicable fields
  - Parent-child hierarchy optional (some inherit from others)

- **`bo_field_def`** - Field definitions (SCOPED TO SUBTYPE)
  - Defines fields **per subtype** (not globally)
  - Each field includes: subtype_def_id FK
  - Includes: field_key, display_name, technical_name, field_type
  - JSON schema constraints (min/max, patterns, enums)
  - `is_required` per subtype (not inherited)

- **`bo_instance`** - Actual records
  - Stores `core_field_values` and `custom_field_values` as JSONB
  - Supports soft deletes (is_deleted flag)
  - Allows JSON evolution—add fields without migrations
  - Includes subtype_def_id for variant-specific logic

- **`bo_relationship`** - Related objects graph
  - Links instances to other instances (BO → BO)
  - Optionally links to hard tables (BO → account, household, etc.)
  - Typed edges: 'owns', 'depends_on', 'member_of', 'generated_by', etc.
  - Extensible via properties JSONB column

- **`bo_audit_log`** - Compliance & governance
  - Tracks all CREATE/UPDATE/DELETE operations
  - Stores changes as JSONB for full history

### 2. **Frontend Edit Modal** (`frontend/src/components/BusinessObjectManager/EditBusinessObjectModal.tsx`)

**Features:**

- **Inline Edit/Create** - No navigation, modal-based workflow
- **Driver Table Selection** - Autocomplete search from catalog_node
  - Shows qualified_path for clarity
  - Auto-loads available tables on modal open
  - Pre-populates if editing existing BO
- **Basic Info Section**
  - Name (technical key for code)
  - Display Name (UI label)
  - Description
- **Configuration Section**
  - Status dropdown (draft/active/deprecated)
  - Enable/disable toggle
  - Extensible config JSONB
- **Validation** - Ensures required fields before save
- **Loading States** - Handles catalog load spinner gracefully

### 3. **Updated BusinessObjectsPage**

- Integrated `EditBusinessObjectModal`
- New `handleSaveBusinessObject()` - POST/PATCH API call
- Edit button opens modal instead of navigating
- Supports both create and edit modes
- Real-time feedback via notifications
- Optimistic UI updates

### 4. **TypeScript Types** (`frontend/src/types/businessObject.ts`)

Comprehensive interfaces for:
- BusinessObjectDefinition
- FieldDefinition
- SubtypeDefinition
- BusinessObjectInstance
- RelatedObject
- AuditLogEntry
- All request/response types

## How It Works: The Driving Table Pattern

### Example: Customer Business Object

```sql
-- 1. Define the BO (driving table)
INSERT INTO business_object_def (bo_key, name, display_name, driver_table_id, driver_table_name)
VALUES ('customer', 'Customer', 'Customer Profile', 'node-123', 'public.customers');

-- 2. Define fields
INSERT INTO bo_field_def (bo_def_id, field_key, display_name, field_type, is_core)
VALUES 
  (bo_1, 'name', 'Full Name', 'string', true),
  (bo_1, 'email', 'Email Address', 'string', true),
  (bo_1, 'risk_score', 'Risk Score', 'number', false);

-- 3. Create instances
INSERT INTO bo_instance (bo_def_id, core_field_values)
VALUES (bo_1, '{"name": "John Doe", "email": "john@example.com"}');

-- 4. Link to related objects
INSERT INTO bo_relationship (from_instance_id, to_hard_table_id, relationship_type)
VALUES (inst_1, 'household_123', 'member_of');
```

### Key Benefits

1. **No Schema Migrations** - Add fields via JSON instead
2. **Flexible Relationships** - BO instances can link to:
   - Other BO instances (BO → BO)
   - Hard tables (BO → account, household)
3. **Variants/Subtypes** - Model variations naturally (Account → Taxable/IRA/Trust)
4. **Audit Trail** - Every change logged for compliance
5. **Tenant-Safe** - All tables scoped to tenant_id
6. **Extensible** - Config JSONB allows custom metadata

## Next Steps to Complete

### 1. Backend API Endpoints (Go)

Create handlers in `backend/internal/api/`:

```go
// POST /api/business-objects - Create BO
// PATCH /api/business-objects/{id} - Update BO
// GET /api/business-objects - List BOs
// GET /api/business-objects/{id} - Get BO details
// DELETE /api/business-objects/{id} - Delete BO

// POST /api/bo/{boKey}/instances - Create instance
// PATCH /api/bo/{boKey}/instances/{id} - Update instance
// GET /api/bo/{boKey}/instances - List instances
// DELETE /api/bo/{boKey}/instances/{id} - Delete instance

// POST /api/bo/{boKey}/relationships - Create relationship
// GET /api/bo/{boKey}/relationships - List relationships
```

### 2. Service Layer (Go)

Implement in `backend/internal/metadata/businessobject_service.go`:

```go
CreateBusinessObject(ctx, tenantID, req) → *BusinessObjectDefinition
UpdateBusinessObject(ctx, tenantID, boKey, req) → *BusinessObjectDefinition
ListBusinessObjects(ctx, tenantID) → []*BusinessObjectDefinition

CreateInstance(ctx, tenantID, boKey, req) → *BusinessObjectInstance
UpdateInstance(ctx, tenantID, boKey, instanceID, req) → *BusinessObjectInstance
ListInstances(ctx, tenantID, boKey, pagination) → ([]*BusinessObjectInstance, total, error)

CreateRelationship(ctx, tenantID, req) → *RelatedObject
ListRelationships(ctx, tenantID, fromInstanceID) → []*RelatedObject
```

### 3. Frontend Service

Create `frontend/src/services/businessObjectService.ts`:

```ts
class BusinessObjectService {
  async createBusinessObject(tenantId, datasourceId, object)
  async updateBusinessObject(tenantId, datasourceId, id, object)
  async listBusinessObjects(tenantId, datasourceId)
  
  async createInstance(tenantId, datasourceId, boKey, instance)
  async updateInstance(tenantId, datasourceId, boKey, instanceId, instance)
  async listInstances(tenantId, datasourceId, boKey, pagination)
  
  async createRelationship(tenantId, datasourceId, relationship)
  async listRelationships(tenantId, datasourceId, fromInstanceId)
}
```

### 4. Instance Manager UI

Create `frontend/src/components/BusinessObjectManager/InstanceManager.tsx`:
- List instances table
- Create/edit instance modal
- Populate core/custom fields based on BO definition
- Validate required fields per subtype
- Relationships tab

### 5. Run Migrations

```bash
cd backend
psql postgres://postgres:postgres@localhost:5432/alpha < migrations/20251218_create_business_object_layer.sql
```

## Testing Checklist

- [ ] Create a new BO via modal
- [ ] Select driver table from catalog_node
- [ ] Edit BO name/description
- [ ] Delete BO
- [ ] Create instances for a BO
- [ ] Edit instance core/custom fields
- [ ] Create relationships between instances
- [ ] Verify audit log entries
- [ ] Test with different subtypes
- [ ] Validate tenant scoping on all endpoints
- [ ] Test pagination on instance lists

## Architectural Decisions

### Why JSONB for Field Values?

- **Flexibility** - Add fields without ALTER TABLE
- **Soft Schema** - Field definitions act as schema, not DB constraints
- **GIN Indexing** - Can index and query nested fields efficiently
- **Evolution** - Instances can have different field sets based on timestamp

### Why driver_table_id?

- **Discovery** - Auto-load column definitions from source table
- **Validation** - Cross-reference actual data against BO definition
- **Lineage** - Track origin of BO in data catalog
- **Governance** - Apply catalog-level policies to BO data

### Why Soft Deletes?

- **Audit Trail** - Keep deleted records for compliance
- **Recovery** - Restore accidentally deleted objects
- **Relationships** - Don't cascade delete to related objects
- **Performance** - No index rebuilds on delete

## References

- [Business Object Pattern](https://en.wikipedia.org/wiki/Business_object)
- [Temporal Modeling](https://www.datagenie.ai/blog/temporal-tables)
- [JSON in PostgreSQL](https://www.postgresql.org/docs/current/datatype-json.html)
- Your Northwind BO documentation

## Questions?

Refer to the agents.md runbook for tenant scope requirements when implementing API endpoints.
