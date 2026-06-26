# bo_fields and Semantic Catalog Integration Reference

## Overview

The `bo_fields` table is a normalized storage mechanism for Business Object field definitions, separate from the semantic catalog structure. While they represent different layers of the data architecture, they work together to provide complete data governance.

---

## 1. Table Schema Definitions

### bo_fields Table Schema

**Location:** `/backend/migrations/000032_redesign_bo_fields_table.sql`

```sql
CREATE TABLE bo_fields (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    business_object_id uuid NOT NULL,      -- Always set: parent BO reference
    subtype_id uuid,                       -- NULL for parent fields, UUID for subtype fields
    
    key varchar(255) NOT NULL,             -- Unique field identifier (e.g., "first_name")
    name varchar(255) NOT NULL,            -- Display name
    display_name varchar(255),             -- UI label
    technical_name varchar(255),           -- Backend name
    type varchar(50) NOT NULL,             -- Field type (text, number, date, reference, etc.)
    
    is_core boolean DEFAULT false,         -- True if core system field
    is_subtype_only boolean DEFAULT false, -- True if custom to subtype
    is_required boolean DEFAULT false,
    is_system boolean DEFAULT false,
    description text,
    reference_entity varchar(255),         -- For reference type fields
    sequence integer DEFAULT 0,            -- Field order
    
    created_at timestamptz DEFAULT now(),
    created_by uuid,
    last_modified_at timestamptz DEFAULT now(),
    last_modified_by uuid,
    
    CONSTRAINT bo_fields_pk PRIMARY KEY (id),
    CONSTRAINT bo_fields_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
    CONSTRAINT bo_fields_bo_fk FOREIGN KEY (business_object_id) REFERENCES public.business_objects(id) ON DELETE CASCADE,
    CONSTRAINT bo_fields_subtype_fk FOREIGN KEY (subtype_id) REFERENCES public.bo_subtypes(id) ON DELETE CASCADE
);

CREATE INDEX bo_fields_bo_idx ON public.bo_fields (business_object_id);
CREATE INDEX bo_fields_subtype_idx ON public.bo_fields (subtype_id);
CREATE INDEX bo_fields_tenant_idx ON public.bo_fields (tenant_id);
CREATE INDEX bo_fields_key_idx ON public.bo_fields (key);
CREATE INDEX bo_fields_bo_subtype_idx ON public.bo_fields (business_object_id, subtype_id);
```

**Key Characteristics:**
- One row per field (normalized)
- Foreign key to `business_objects.id`
- Optional foreign key to `bo_subtypes.id` (NULL for parent BO fields)
- Sequence ordering for field display order
- Audit trail columns (created_by, last_modified_by)
- Type information for validation and display

---

## 2. SQL Queries Loading Fields with Context

### Query 1: Load All Fields for a Business Object

```sql
-- Load entity-level fields (parent BO fields, no subtype)
SELECT 
    id, 
    key, 
    name, 
    display_name, 
    technical_name, 
    type,
    is_core, 
    is_required, 
    is_system, 
    description,
    reference_entity, 
    sequence,
    created_at, 
    created_by, 
    last_modified_at, 
    last_modified_by
FROM public.bo_fields
WHERE business_object_id = $1 
  AND tenant_id = $2
  AND subtype_id IS NULL          -- Only parent BO fields
ORDER BY sequence;
```

### Query 2: Load Subtype-Specific Fields

```sql
-- Load fields for a business object subtype
SELECT 
    id, 
    key, 
    name, 
    display_name, 
    technical_name, 
    type,
    is_core, 
    is_required, 
    is_system, 
    description,
    reference_entity, 
    sequence,
    created_at, 
    created_by, 
    last_modified_at, 
    last_modified_by
FROM public.bo_fields
WHERE subtype_id = $1
ORDER BY sequence;
```

### Query 3: Get All Fields for BO with Business Object Context

```sql
SELECT 
    bo.id as bo_id,
    bo.name as bo_name,
    bo.display_name as bo_display_name,
    bo.key as bo_key,
    bf.id as field_id,
    bf.key as field_key,
    bf.name as field_name,
    bf.display_name as field_display_name,
    bf.type as field_type,
    bf.is_core,
    bf.is_required,
    bf.sequence as field_sequence
FROM business_objects bo
LEFT JOIN bo_fields bf ON bf.business_object_id = bo.id AND bf.subtype_id IS NULL
WHERE bo.tenant_id = $1 
  AND bo.id = $2
ORDER BY bo.id, bf.sequence;
```

### Query 4: Search Fields by Name

```sql
SELECT DISTINCT 
    bo.id, 
    bo.name,
    bo.display_name,
    bo.key,
    COUNT(bf.id) as field_count
FROM business_objects bo
JOIN bo_fields bf ON bf.business_object_id = bo.id
WHERE bf.name ILIKE '%' || $1 || '%'     -- Case-insensitive search
  AND bo.tenant_id = $2
GROUP BY bo.id, bo.name, bo.display_name, bo.key
ORDER BY bo.name;
```

### Query 5: Find Required Fields for a BO

```sql
SELECT 
    name, 
    type, 
    display_name,
    description
FROM bo_fields 
WHERE business_object_id = $1
  AND is_required = true
  AND subtype_id IS NULL
ORDER BY sequence;
```

---

## 3. API Endpoints and Code

### Backend Service: BusinessObjectService

**File:** `/backend/internal/services/businessobject_service.go`

#### Method: loadBOSubtypesAndFields

```go
func (s *BusinessObjectService) loadBOSubtypesAndFields(
    ctx context.Context,
    bo *models.BusinessObjectDefinition,
) error {
    if bo.Subtypes == nil {
        bo.Subtypes = make(map[string]models.SubtypeDefinition)
    }

    // Load subtypes
    subtypeQuery := `
        SELECT id, key, name, display_name, technical_name, description,
               is_core, based_on_entity, clone_parent_key, sequence,
               created_at, created_by, last_modified_at, last_modified_by
        FROM bo_subtypes
        WHERE business_object_id = $1
        ORDER BY sequence
    `

    var subtypes []models.SubtypeDefinition
    if err := s.db.SelectContext(ctx, &subtypes, subtypeQuery, bo.ID); err != nil {
        return fmt.Errorf("failed to load subtypes: %w", err)
    }

    for i := range subtypes {
        // Load fields for each subtype
        fieldQuery := `
            SELECT id, key, name, display_name, technical_name, type,
                   is_core, is_required, is_system, description,
                   reference_entity, sequence,
                   created_at, created_by, last_modified_at, last_modified_by
            FROM bo_fields
            WHERE subtype_id = $1
            ORDER BY sequence
        `

        var fields []models.FieldDefinition
        if err := s.db.SelectContext(ctx, &fields, fieldQuery, subtypes[i].ID); err != nil {
            return fmt.Errorf("failed to load subtype fields: %w", err)
        }

        subtypes[i].SubtypeFields = fields
        bo.Subtypes[subtypes[i].Key] = subtypes[i]
    }

    // Load entity-level fields (parent BO fields)
    fieldQuery := `
        SELECT id, key, name, display_name, technical_name, type,
               is_core, is_required, is_system, description,
               reference_entity, sequence,
               created_at, created_by, last_modified_at, last_modified_by
        FROM bo_fields
        WHERE business_object_id = $1 AND subtype_id IS NULL
        ORDER BY sequence
    `

    var entityFields []models.FieldDefinition
    if err := s.db.SelectContext(ctx, &entityFields, fieldQuery, bo.ID); err != nil {
        return fmt.Errorf("failed to load entity fields: %w", err)
    }

    bo.CoreFields = []models.FieldDefinition{}
    bo.CustomFields = []models.FieldDefinition{}

    for _, field := range entityFields {
        if field.IsCore {
            bo.CoreFields = append(bo.CoreFields, field)
        } else {
            bo.CustomFields = append(bo.CustomFields, field)
        }
    }

    return nil
}
```

#### Method: GetBusinessObject

```go
func (s *BusinessObjectService) GetBusinessObject(
    ctx context.Context,
    tenantID, boKey string,
) (*models.BusinessObjectDefinition, error) {
    query := `
        SELECT id, tenant_id, key, name, display_name, technical_name,
               description, icon, is_core, clones_from, clone_parent_key,
               clone_parent_display_name, category, instance_count,
               created_at, created_by, last_modified_at, last_modified_by
        FROM business_objects
        WHERE tenant_id = $1 AND key = $2
    `

    bo := &models.BusinessObjectDefinition{}
    err := s.db.GetContext(ctx, bo, query, tenantID, boKey)
    if err != nil {
        return nil, fmt.Errorf("failed to get business object: %w", err)
    }

    // Load subtypes and fields via bo_fields table
    if err := s.loadBOSubtypesAndFields(ctx, bo); err != nil {
        log.Printf("Warning: failed to load subtypes and fields: %v", err)
    }

    return bo, nil
}
```

---

## 4. Field-to-Semantic-Term Relationship Mapping

### Current Architecture

**Two Separate Systems:**

1. **bo_fields** - Business Object field structure
   - Stores field definitions for BOs
   - Manages validation rules, types, required flags
   - Used for form generation and business process designer

2. **catalog_node** - Semantic catalog structure
   - Stores semantic terms, semantic columns, database columns
   - Manages lineage, relationships, and metadata
   - Tracks business context and data governance

### Potential Integration Points

While currently separate, these could be linked through:

```typescript
// Frontend example of potential integration
interface FieldWithSemanticContext {
    // bo_fields data
    field_id: string;
    field_name: string;
    field_type: string;
    business_object_id: string;
    
    // Potential semantic mapping
    semantic_term_id?: string;      // Links to catalog_node (semantic_term type)
    semantic_column_id?: string;    // Links to catalog_node (semantic_column type)
    is_mapped_to_semantic?: boolean;
    semantic_metadata?: {
        node_name: string;
        description: string;
        qualified_path: string;
        properties: Record<string, any>;
    };
}
```

### GraphQL Type Definition (Proposed)

```graphql
type BusinessObject {
    id: ID!
    name: String!
    kind: String!
    description: String
    coreFields: [Field!]!
    customFields: [Field!]!
    createdAt: String!
    updatedAt: String!
}

type Field {
    id: ID!
    key: String!
    name: String!
    displayName: String!
    type: String!
    isCore: Boolean!
    isRequired: Boolean!
    description: String
    sequence: Int!
    # Potential future links to semantic layer
    semanticTermId: ID
    semanticTerm: SemanticTerm
}

type SemanticTerm {
    id: ID!
    node_name: String!
    description: String
    qualified_path: String!
    properties: JSON
}
```

---

## 5. Data Migration Example

### Migration: Normalize bo_fields JSONB to Table

**File:** `/backend/migrations/000031_normalize_bo_fields.sql`

```sql
-- Step 1: Create temporary table to hold extracted fields
CREATE TEMP TABLE temp_extracted_fields AS
SELECT 
    bo.id as business_object_id,
    bo.tenant_id,
    (field->>'key')::text as key,
    (field->>'name')::text as name,
    (field->>'display_name')::text as display_name,
    (field->>'technical_name')::text as technical_name,
    (field->>'type')::text as type,
    (field->>'is_core')::boolean as is_core,
    (field->>'is_required')::boolean as is_required,
    (field->>'is_system')::boolean as is_system,
    (field->>'description')::text as description,
    (field->>'reference_entity')::text as reference_entity,
    (field->>'sequence')::integer as sequence,
    (field->>'created_at')::timestamptz as created_at,
    (field->>'created_by')::uuid as created_by,
    (field->>'last_modified_at')::timestamptz as last_modified_at,
    (field->>'last_modified_by')::uuid as last_modified_by,
    row_number() OVER (PARTITION BY bo.id ORDER BY (field->>'sequence')::integer, (field->>'key')) as row_num
FROM public.business_objects bo,
     jsonb_array_elements(COALESCE(bo.fields, '[]'::jsonb)) as field
WHERE bo.fields IS NOT NULL AND bo.fields != '[]'::jsonb;

-- Step 2: Insert extracted fields into bo_fields table
INSERT INTO public.bo_fields (
    tenant_id,
    business_object_id,
    subtype_id,
    key,
    name,
    display_name,
    technical_name,
    type,
    is_core,
    is_required,
    is_system,
    description,
    reference_entity,
    sequence,
    created_at,
    created_by,
    last_modified_at,
    last_modified_by
)
SELECT 
    business_object_id,
    business_object_id,
    NULL,  -- Parent BO fields have no subtype
    COALESCE(key, 'field_' || row_num::text),
    COALESCE(name, 'Field ' || row_num::text),
    COALESCE(display_name, 'Field ' || row_num::text),
    COALESCE(technical_name, key),
    COALESCE(type, 'text'),
    COALESCE(is_core, false),
    COALESCE(is_required, false),
    COALESCE(is_system, false),
    description,
    reference_entity,
    COALESCE(sequence, row_num),
    COALESCE(created_at, now()),
    created_by,
    COALESCE(last_modified_at, now()),
    last_modified_by
FROM temp_extracted_fields
ON CONFLICT DO NOTHING;

-- Step 3: Drop the old JSONB column
ALTER TABLE public.business_objects DROP COLUMN IF EXISTS fields CASCADE;
```

---

## 6. Frontend Integration (React/TypeScript)

### Type Definitions

**File:** `/frontend/src/types/SemanticTypes.ts`

```typescript
export interface SemanticNode {
    id: string;
    node_name: string;
    node_type: 'business_term' | 'semantic_term' | 'semantic_column' | 'database_column' | 'semantic_model';
    description: string;
    qualified_path: string;
    properties: Record<string, any>;
}

export interface SemanticEdge {
    id: string;
    source_node_id: string;
    target_node_id: string;
    edge_type_id: string;
    relationship_type: string;
    properties: Record<string, unknown>;
}

export interface RawSemanticChart {
    businessTerms: SemanticNode[];
    semanticTerms: SemanticNode[];
    semanticColumns: SemanticNode[];
    databaseColumns: SemanticNode[];
    edges: SemanticEdge[];
    viewport: Record<string, unknown>;
    metadata: Record<string, any>;
}
```

### Component Integration Example

```typescript
// Load fields for a Business Object
async function loadFieldsForBO(boId: string): Promise<FieldDefinition[]> {
    const response = await api.get(`/business-objects/${boId}`);
    return [
        ...response.coreFields,
        ...response.customFields
    ];
}

// Map fields to semantic terms (future integration)
interface FieldSemanticMapping {
    field_name: string;
    semantic_term_id?: string;
    semantic_term_name?: string;
    mapping_confidence?: number;
}

function mapFieldsToSemanticTerms(
    fields: FieldDefinition[],
    semanticTerms: SemanticNode[]
): FieldSemanticMapping[] {
    return fields.map(field => {
        // Simple name matching example
        const matchedTerm = semanticTerms.find(st => 
            st.node_name.toLowerCase().includes(field.name.toLowerCase())
        );
        
        return {
            field_name: field.name,
            semantic_term_id: matchedTerm?.id,
            semantic_term_name: matchedTerm?.node_name,
            mapping_confidence: matchedTerm ? 0.8 : 0
        };
    });
}
```

---

## 7. Key Relationships and Flow

```
┌─────────────────────────────────────────┐
│       business_objects Table            │
│  (BO definition - name, display, etc)   │
└────────────┬────────────────────────────┘
             │ (1:N relationship)
             │ foreign_key: business_object_id
             ▼
┌─────────────────────────────────────────┐
│      bo_fields Table (NORMALIZED)       │
│  - field_name, type, is_required, etc   │
│  - One row per field                    │
│  - indexed by business_object_id        │
│  - indexed by key                       │
└─────────────────────────────────────────┘

SEPARATE SYSTEM:

┌─────────────────────────────────────────┐
│     catalog_node Table (SEMANTIC)       │
│  - business_term, semantic_term         │
│  - semantic_column, database_column     │
│  - node_name, qualified_path, props     │
└────────────┬────────────────────────────┘
             │ (1:N relationship)
             │ foreign_key: source_node_id, target_node_id
             ▼
┌─────────────────────────────────────────┐
│      catalog_edge Table (SEMANTIC)      │
│  - relationship_type (BusinessTerm...)  │
│  - properties (semantic metadata)       │
└─────────────────────────────────────────┘

POTENTIAL INTEGRATION:
bo_fields.technical_name → catalog_node.properties['field_mapping']
```

---

## 8. Performance Considerations

### Index Strategy for bo_fields

```sql
-- Primary access patterns
CREATE INDEX bo_fields_bo_idx ON public.bo_fields (business_object_id);
CREATE INDEX bo_fields_subtype_idx ON public.bo_fields (subtype_id);
CREATE INDEX bo_fields_key_idx ON public.bo_fields (key);

-- Composite index for common query pattern
CREATE INDEX bo_fields_bo_subtype_idx ON public.bo_fields (business_object_id, subtype_id);

-- Tenant isolation
CREATE INDEX bo_fields_tenant_idx ON public.bo_fields (tenant_id);

-- Full-text search (future enhancement)
-- CREATE INDEX bo_fields_name_fts ON bo_fields USING GIN (to_tsvector('english', name));
```

### Query Performance

**Before (JSONB parsing):**
- Sequential scan required
- JSONB operators slow on large datasets
- No index available

**After (normalized table):**
- Index seeks on bo_fields_bo_idx
- B-tree indexes for fast lookups
- Partial indexes possible for is_required = true, etc.

---

## 9. Summary

| Aspect | bo_fields | catalog_node | Relationship |
|--------|-----------|--------------|--------------|
| Purpose | BO field definitions | Semantic catalog data | Complementary |
| Normalization | One row per field | Hierarchical metadata | Different concerns |
| Primary Key | id (uuid) | id (uuid) | Not directly linked |
| Foreign Keys | business_object_id, subtype_id | source_node_id, target_node_id | Via properties JSON |
| Query Pattern | SELECT WHERE business_object_id = X | SELECT WHERE tenant_datasource_id = Y | Separate queries |
| Use Case | Form generation, validation | Lineage, governance, mapping | Data dictionary |
| Indexing | B-tree on bo_idx, key_idx | B-tree on source/target | Independent |

The two systems remain separate by design but can be integrated through metadata properties and application-level mapping logic.
