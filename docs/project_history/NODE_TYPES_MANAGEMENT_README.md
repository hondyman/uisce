# Node Types Management System

## Overview

This document describes the comprehensive Node Types management system for configuring catalog node types (Business Term, Semantic Term, Database Column, etc.) with full CRUD operations and flexible property configuration.

## Architecture

### Database Schema

The system leverages the existing `catalog_node_type` table:

```sql
CREATE TABLE catalog_node_type (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    catalog_type_name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    parent_type_id VARCHAR(255),
    config JSONB,  -- Stores property definitions
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, catalog_type_name)
);
```

### Property Configuration Format

Properties are stored in the `config.properties` JSONB field with the following structure:

```json
{
  "properties": [
    {
      "name": "business_owner",
      "label": "Business Owner",
      "data_type": "string",
      "nullable": false,
      "default_value": "",
      "input_type": "text",
      "format": "email",
      "validation": {
        "maxLength": 255,
        "pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
      },
      "order": 0
    },
    {
      "name": "status",
      "label": "Status",
      "data_type": "string",
      "nullable": false,
      "input_type": "select",
      "options": ["Draft", "Published", "Archived"],
      "order": 1
    }
  ]
}
```

## Backend Implementation

### Files Created

- **`backend/internal/api/node_types_routes.go`** - Complete REST API handlers

### API Endpoints

All endpoints require tenant scope via `tenant_id` query parameter and `X-Tenant-ID` header.

#### Node Type Management

```
GET    /api/node-types                    - List all node types
POST   /api/node-types                    - Create node type
GET    /api/node-types/:id                - Get single node type
PATCH  /api/node-types/:id                - Update node type
DELETE /api/node-types/:id                - Delete node type
```

#### Property Management

```
GET    /api/node-types/:id/properties            - List properties
POST   /api/node-types/:id/properties            - Add property
PATCH  /api/node-types/:id/properties/:propName  - Update property
DELETE /api/node-types/:id/properties/:propName  - Delete property
```

### Request/Response Examples

#### Create Node Type

**Request:**
```json
POST /api/node-types?tenant_id=default

{
  "tenant_id": "default",
  "catalog_type_name": "business_term",
  "description": "Business Term",
  "is_active": true,
  "parent_type_id": null,
  "config": {
    "properties": [
      {
        "name": "definition",
        "label": "Definition",
        "data_type": "text",
        "nullable": false,
        "input_type": "textarea",
        "order": 0
      }
    ]
  }
}
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "default",
  "catalog_type_name": "business_term",
  "description": "Business Term",
  "is_active": true,
  "parent_type_id": null,
  "config": { ... },
  "created_at": "2025-10-15T12:00:00Z",
  "updated_at": "2025-10-15T12:00:00Z"
}
```

#### Add Property

**Request:**
```json
POST /api/node-types/:id/properties?tenant_id=default

{
  "name": "owner_email",
  "label": "Owner Email",
  "data_type": "string",
  "nullable": false,
  "input_type": "text",
  "format": "email",
  "validation": {
    "maxLength": 255,
    "pattern": "^[\\w._%+-]+@[\\w.-]+\\.[a-zA-Z]{2,}$"
  },
  "order": 1
}
```

## Frontend Implementation

### Files Created

```
frontend/src/
├── types/
│   └── nodeTypes.ts              - TypeScript interfaces
├── api/
│   └── nodeTypes.ts              - React Query hooks
└── pages/nodes/
    ├── NodeTypeSetupPage.tsx     - Main management page
    ├── NodeTypeTable.tsx         - List view component
    ├── NodeTypeFormModal.tsx     - Create/edit modal
    ├── PropertyListEditor.tsx    - Property list management
    ├── PropertyFormModal.tsx     - Property editor
    └── NodeTypePreviewPanel.tsx  - Property preview renderer
```

### Component Hierarchy

```
NodeTypeSetupPage
├── NodeTypeTable
│   └── [table rows with Edit/Delete buttons]
└── NodeTypeFormModal
    ├── [basic form fields]
    └── PropertyListEditor
        └── PropertyFormModal
```

### React Query Hooks

```typescript
// List node types
const { data: nodeTypes } = useNodeTypes(tenantId);

// Create node type
const createMutation = useCreateNodeType();
await createMutation.mutateAsync(data);

// Update node type
const updateMutation = useUpdateNodeType();
await updateMutation.mutateAsync({ id, tenantId, data });

// Delete node type
const deleteMutation = useDeleteNodeType();
await deleteMutation.mutateAsync({ id, tenantId });

// Property operations
const { data: properties } = useNodeTypeProperties(id, tenantId);
const addProperty = useAddNodeTypeProperty();
const updateProperty = useUpdateNodeTypeProperty();
const deleteProperty = useDeleteNodeTypeProperty();
```

### Property Data Types

| Data Type | Input Types | Description |
|-----------|-------------|-------------|
| `string` | text, select, textarea | Short text values |
| `text` | textarea | Long text content |
| `integer` | number | Whole numbers |
| `float` | number | Decimal numbers |
| `boolean` | checkbox | True/false values |
| `date` | date-picker | Date values |
| `json` | json-editor | JSON objects |

### Validation Rules

Property validation is configurable per data type:

**String/Text:**
- `minLength` - Minimum character count
- `maxLength` - Maximum character count
- `pattern` - Regex validation

**Integer/Float:**
- `min` - Minimum value
- `max` - Maximum value

**All Types:**
- `nullable` - Whether field can be empty
- `default_value` - Default value when creating new nodes

## Navigation

The Node Types page is accessible via:

1. **URL:** `/core/node-types`
2. **Navigation:** Core menu → Node Types
3. **Description:** "Configure node types and their properties for the business glossary"

## Tenant Scope Requirements

⚠️ **All operations require an active tenant selection.**

The page checks `localStorage.selected_tenant` and displays a warning if no tenant is selected. This ensures:

- Data isolation between tenants
- Proper security boundaries
- Consistent scoping with other Fabric features

## Usage Flow

### Creating a Node Type

1. Click "Create Node Type" button
2. Fill in basic details:
   - Type Name (e.g., `business_term`)
   - Description
   - Parent Type (optional)
   - Active status
3. Add properties:
   - Click "Add Property"
   - Configure name, label, data type, input type
   - Set validation rules
   - Reorder with up/down buttons
4. Preview how properties will render
5. Save the node type

### Editing a Node Type

1. Click "Edit" on any node type row
2. Modify basic details or properties
3. Add, edit, or delete properties
4. Save changes

### Deleting a Node Type

1. Click "Delete" on any node type row
2. Confirm deletion (cascades to child types)

## Dynamic Form Rendering

When node types are configured with properties, glossary detail pages can dynamically render forms:

```typescript
// Fetch node type configuration
const { data: nodeType } = useNodeType(typeId, tenantId);

// Render form based on properties
nodeType.config.properties.map(prop => {
  switch (prop.input_type) {
    case 'text':
      return <input type="text" ... />;
    case 'select':
      return <select><option ... /></select>;
    case 'textarea':
      return <textarea ... />;
    // etc.
  }
});
```

## Extensibility

### Adding New Data Types

1. Add to `DATA_TYPE_OPTIONS` in `types/nodeTypes.ts`
2. Update validation logic in `PropertyFormModal`
3. Add rendering case in `NodeTypePreviewPanel`

### Adding New Input Types

1. Add to `INPUT_TYPE_OPTIONS` in `types/nodeTypes.ts`
2. Implement rendering in `NodeTypePreviewPanel`
3. Add any specific configuration UI in `PropertyFormModal`

### Custom Validation

Extend the `validation` object in property configuration:

```json
{
  "validation": {
    "custom_rule": "value",
    "business_logic": { ... }
  }
}
```

## Security Considerations

- ✅ Tenant isolation enforced at API and UI level
- ✅ Authentication required via session middleware
- ✅ Input validation on all endpoints
- ✅ SQL injection protection via parameterized queries
- ✅ JSONB structure validation
- ✅ Cascade delete protection with confirmation

## Future Enhancements

1. **Import/Export** - JSON/YAML format for property configs
2. **Versioning** - Track changes to property definitions
3. **Templates** - Pre-built property sets for common node types
4. **Bulk Operations** - Multi-node updates
5. **Audit Trail** - Log all configuration changes
6. **Property Groups** - Organize properties into collapsible sections
7. **Conditional Properties** - Show/hide based on other field values
8. **Formula Fields** - Computed properties based on other values

## Testing

### Backend Tests

```bash
cd backend
go test ./internal/api/node_types_routes_test.go
```

### Frontend Tests

```bash
cd frontend
npm test -- NodeTypeSetupPage
npm test -- PropertyFormModal
```

### Manual Testing Checklist

- [ ] Create node type without properties
- [ ] Create node type with multiple properties
- [ ] Edit node type and modify properties
- [ ] Delete property from node type
- [ ] Reorder properties
- [ ] Test all data types
- [ ] Test all input types
- [ ] Test validation rules
- [ ] Test nullable vs required fields
- [ ] Test select with options
- [ ] Test parent-child relationships
- [ ] Test tenant isolation
- [ ] Test cascade delete

## Troubleshooting

### "Tenant Required" Warning

**Cause:** No tenant selected in localStorage.  
**Solution:** Use tenant picker to select a tenant.

### Properties Not Saving

**Cause:** JSONB structure mismatch.  
**Solution:** Check browser console for validation errors.

### API Errors

**Cause:** Missing tenant scope parameters.  
**Solution:** Ensure `tenant_id` query param and `X-Tenant-ID` header are present.

## Support

For issues or questions:
- Check the agent runbook in `agents.md`
- Review API logs: `docker logs semlayer-backend`
- Check frontend console for client-side errors
- Verify database schema: `psql postgres://postgres:postgres@localhost:5432/alpha`

---

**Implementation Date:** October 15, 2025  
**Version:** 1.0.0  
**Status:** ✅ Complete
