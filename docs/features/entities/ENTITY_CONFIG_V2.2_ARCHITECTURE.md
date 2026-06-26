# Entity Config v2.2: Architecture & Implementation Guide

**Target Audience:** Developers, architects, agents  
**Level:** Advanced  
**Focus:** Type system, data flow, component interactions  

---

## 📐 Architecture Overview

### System Layers

```
┌────────────────────────────────────────────────────────────────┐
│ PRESENTATION LAYER                                             │
├────────────────────────────────────────────────────────────────┤
│ EntityConfigPageV3.tsx (500+ lines)                            │
│ ├─ State Management (entities, searchTerm, selectedNode, etc)  │
│ ├─ Layout (Sider + Content)                                   │
│ ├─ Field Tables (Inherited + Assigned)                        │
│ ├─ Modals (Semantic term selector)                            │
│ └─ Event Handlers (Add, Delete, Reorder, Save)               │
└────────────────────────────────────────────────────────────────┘
         ↑                                      ↓
         │ Consumes                            │ Triggers
         │                                      ↓
┌────────────────────────────────────────────────────────────────┐
│ BUSINESS LOGIC LAYER                                           │
├────────────────────────────────────────────────────────────────┤
│ useEnhancedSemanticTerms.ts (150 lines)                        │
│ ├─ GraphQL Query: GET_SEMANTIC_TERMS_WITH_METADATA            │
│ ├─ Hook: useEnhancedSemanticTerms(datasourceId)              │
│ ├─ Transformations: semanticTermToField()                     │
│ ├─ Utils: searchSemanticTerms(), groupByCategory()           │
│ └─ Returns: { semanticTerms, loading, error, refetch }        │
└────────────────────────────────────────────────────────────────┘
         ↑                                      ↓
         │ Executes                            │ Calls
         │                                      ↓
┌────────────────────────────────────────────────────────────────┐
│ DATA ACCESS LAYER                                              │
├────────────────────────────────────────────────────────────────┤
│ Apollo Client (GraphQL)                                        │
│ ├─ Endpoint: /graphql                                         │
│ ├─ Query: GET_SEMANTIC_TERMS_WITH_METADATA                    │
│ ├─ Variables: { datasourceId: string }                        │
│ └─ Caching: Apollo cache (automatic)                          │
├────────────────────────────────────────────────────────────────┤
│ REST API (entitySchema.ts)                                     │
│ ├─ Endpoint: /api/entity-schema                               │
│ ├─ Methods: GET (fetch), POST (save)                          │
│ ├─ Payload: { changed: {...}, deleted: [...] }                │
│ └─ Headers: X-Tenant-ID, X-Tenant-Datasource-ID              │
└────────────────────────────────────────────────────────────────┘
         ↑                                      ↓
         │ Executes                            │ Returns
         │                                      ↓
┌────────────────────────────────────────────────────────────────┐
│ DATABASE LAYER                                                 │
├────────────────────────────────────────────────────────────────┤
│ Postgres (Local Development)                                   │
│ ├─ catalog_node (semantic terms)                              │
│ ├─ entity_schema (entity definitions + JSONB fields)         │
│ └─ Tenant isolation via tenant_id, datasource_id             │
└────────────────────────────────────────────────────────────────┘
```

---

## 🔄 Data Flow: End-to-End

### Scenario: User Adds "Tax ID" Field to Individual Investor

```
┌─────────────────────────────────────────────────────────────┐
│ 1. INITIALIZATION (Page Load)                              │
└─────────────────────────────────────────────────────────────┘

EntityConfigPageV3.tsx mounts:
  useEffect(() => {
    const savedSchema = await fetchEntitySchema()
    setEntities(savedSchema)  // { client_investor: {...}, ... }
  })

  const { semanticTerms, loading } = useEnhancedSemanticTerms(datasource.id)
    → Apollo executes: GET_SEMANTIC_TERMS_WITH_METADATA
    → GraphQL fetches: SELECT * FROM catalog_node WHERE datasource_id = ?
    → Returns: [
        { id: 'sem-1', node_name: 'Tax ID', properties: {...} },
        { id: 'sem-2', node_name: 'Status', properties: {...} },
        ...
      ]
    → Hook enhances: Adds businessName, technicalName, dataType (computed)
    → Returns: [
        { id: 'sem-1', node_name: 'Tax ID', businessName: 'Tax ID', 
          technicalName: 'tax_id', dataType: 'text', ... },
        ...
      ]

Result: UI renders with Sidebar tree + empty Assigned Fields table

┌─────────────────────────────────────────────────────────────┐
│ 2. USER INTERACTION (Select Subtype)                       │
└─────────────────────────────────────────────────────────────┘

User clicks "Individual Investor" in sidebar tree:
  → Tree onClick handler triggered
  → selectedNode = { 
      type: 'subtype', 
      entityKey: 'client_investor', 
      subtypeKey: 'individual' 
    }
  → Component computes: content = entities['client_investor'].subtypes['individual']
  → Separates fields: 
      inheritedFields = [{key: 'ssn', isCore: true}]
      assignedFields = []

Result: Right panel shows:
  - 🔒 Inherited Fields: SSN
  - ✏️ Assigned Fields (0): [+Add]

┌─────────────────────────────────────────────────────────────┐
│ 3. MODAL OPEN (Click Add Field)                            │
└─────────────────────────────────────────────────────────────┘

User clicks [+Add] button:
  → setEditingField({...})  // Modal opens
  → User types "tax" in search box
  → setSemanticSearchTerm('tax')

searchSemanticTerms() filters semanticTerms:
  function searchSemanticTerms(terms, query) {
    return terms.filter(t => 
      t.businessName.toLowerCase().includes(query.toLowerCase()) ||
      t.technicalName.toLowerCase().includes(query.toLowerCase()) ||
      t.description.toLowerCase().includes(query.toLowerCase())
    )
  }
  
Result: Modal shows filtered list:
  ✓ Tax ID (technical: tax_id, type: text)
  ✓ Tax Rate (technical: tax_rate, type: number)
  ✓ Tax Status (technical: tax_status, type: enum)

┌─────────────────────────────────────────────────────────────┐
│ 4. FIELD CREATION (User Selects Term)                      │
└─────────────────────────────────────────────────────────────┘

User clicks [Add] next to "Tax ID":
  → handleAddField(semanticTerm) executed
  
  semanticTerm = {
    id: 'sem-tax-001',
    node_name: 'Tax ID',
    businessName: 'Tax ID',
    technicalName: 'tax_id',
    dataType: 'text',
    description: 'Unique tax identifier',
    properties: { ... }
  }

  newField = semanticTermToField(semanticTerm, sequence)
  
  function semanticTermToField(term, sequence) {
    return {
      key: `field-${uuidv4()}`,
      name: term.businessName,
      businessName: term.businessName,
      technicalName: term.technicalName,
      type: term.dataType,
      semanticTermId: term.id,              // ✅ REQUIRED
      semanticTermName: term.businessName,  // ✅ REQUIRED
      isCore: false,                        // Custom field
      sequence: sequence,                   // 1 (after inherited SSN)
      lastModifiedAt: new Date().toISOString(),
      createdBy: currentUser.email,
      description: term.description
    }
  }

  Result: newField = {
    key: 'f-abc123',
    businessName: 'Tax ID',
    technicalName: 'tax_id',
    type: 'text',
    semanticTermId: 'sem-tax-001',
    semanticTermName: 'Tax ID',
    isCore: false,
    sequence: 1,
    lastModifiedAt: '2025-01-15T10:30:00Z',
    createdBy: 'user@company.com',
    description: 'Unique tax identifier'
  }

  State updated:
  entities['client_investor'].subtypes['individual'].subtype_fields = [
    { key: 'ssn', ..., isCore: true },        // Inherited
    { key: 'f-abc123', ..., isCore: false }   // Newly added
  ]

  computeChanges = {
    changed: ['client_investor'],  // This entity was modified
    deleted: []
  }

Result: Modal closes, table refreshes, "Tax ID" row appears in Assigned Fields

┌─────────────────────────────────────────────────────────────┐
│ 5. SAVE TO BACKEND (User Clicks Save)                      │
└─────────────────────────────────────────────────────────────┘

User clicks [SAVE & APPLY]:
  → saveAndApply() function executed
  
  payload = {
    changed: {
      client_investor: {
        name: 'Client Investor',
        entity_fields: [...],
        subtypes: {
          individual: {
            subtype_fields: [
              { key: 'ssn', ..., isCore: true },
              { key: 'f-abc123', businessName: 'Tax ID', 
                semanticTermId: 'sem-tax-001', sequence: 1, ... }
            ]
          }
        }
      }
    },
    deleted: []
  }

  await saveEntitySchema(payload)
    → POST /api/entity-schema
    → Headers: {
        'X-Tenant-ID': tenant.id,
        'X-Tenant-Datasource-ID': datasource.id
      }
    → Query params: ?tenant_id=...&datasource_id=...
    → Body: JSON.stringify(payload)

Backend (api.go):
  POST /api/entity-schema
    ├─ Extracts tenant_id, datasource_id from headers + query
    ├─ For each entity in changed:
    │   └─ INSERT or UPDATE entity_schema table with JSONB
    ├─ For each entity in deleted:
    │   └─ DELETE entity_schema row
    └─ Returns: { success: true, message: '...' }

Database:
  UPDATE entity_schema 
  SET schema_data = {
    "entity_fields": [...],
    "subtypes": {
      "individual": {
        "subtype_fields": [
          { "key": "ssn", ... },
          { "key": "f-abc123", "businessName": "Tax ID", ... }
        ]
      }
    }
  }
  WHERE tenant_id = ? AND datasource_id = ? AND entity_key = 'client_investor'

Response:
  ← 200 OK { success: true }
  → setInitialEntities(entities)  // Reset initial state
  → message.success('✅ Saved! 1 changed, 0 deleted')
  → computeChanges = { changed: [], deleted: [] }  // No unsaved changes
  → [SAVE & APPLY] button disabled (no pending changes)

Result: Changes persisted to backend, UI refreshed, success message shown
```

---

## 🏗️ Component Structure

### EntityConfigPageV3.tsx (500+ lines)

```typescript
export default function EntityConfigPageV3() {
  // ═══════════════════════════════════════════════════════════
  // STATE MANAGEMENT
  // ═══════════════════════════════════════════════════════════
  
  const [entities, setEntities] = useState<Entities>(CORE_ENTITIES)
  // Current state (what user sees)
  
  const [initialEntities, setInitialEntities] = useState<Entities>(CORE_ENTITIES)
  // Baseline state (last saved to backend)
  
  const [searchTerm, setSearchTerm] = useState('')
  // Entity search filter
  
  const [selectedNode, setSelectedNode] = useState<SelectedNode | null>(null)
  // { type: 'entity'|'subtype', entityKey, subtypeKey? }
  
  const [editingField, setEditingField] = useState<EditingField | null>(null)
  // { entityKey, subtypeKey?, fieldKey?, level }
  
  const [semanticSearchTerm, setSemanticSearchTerm] = useState('')
  // Semantic term search in modal
  
  const { datasource } = useTenant()
  // Current tenant scope
  
  const { semanticTerms, loading } = useEnhancedSemanticTerms(datasource?.id)
  // Fetched + enhanced semantic terms from catalog

  // ═══════════════════════════════════════════════════════════
  // DERIVED STATE (useMemo)
  // ═══════════════════════════════════════════════════════════
  
  const computeChanges = useMemo(() => {
    const changed = Object.keys(entities).filter(key => 
      JSON.stringify(entities[key]) !== JSON.stringify(initialEntities[key])
    )
    const deleted = Object.keys(initialEntities).filter(key => 
      !(key in entities)
    )
    return { changed, deleted }
  }, [entities, initialEntities])
  // Shows: "SAVE & APPLY (3)" if 3 changes pending

  const hierarchyTree = useMemo(() => {
    // Build Tree component data from entities
    // Filter by searchTerm
    // Return: [{ key, title, data, children: [...subtypes] }]
  }, [entities, searchTerm])
  
  const filteredSemanticTerms = useMemo(() => {
    return searchSemanticTerms(semanticTerms, semanticSearchTerm)
  }, [semanticTerms, semanticSearchTerm])

  // ═══════════════════════════════════════════════════════════
  // EVENT HANDLERS
  // ═══════════════════════════════════════════════════════════

  const handleAddField = (semanticTerm) => {
    // 1. Convert semantic term → field
    // 2. Update entities state
    // 3. Show success toast
  }

  const handleDeleteField = (fieldKey) => {
    // 1. Remove field from entity/subtype
    // 2. Update entities state
  }

  const handleReorderField = (fieldKey, direction) => {
    // 1. Swap with adjacent field
    // 2. Update sequence numbers
    // 3. Update entities state
  }

  const saveAndApply = async () => {
    // 1. Compute delta (changed + deleted)
    // 2. POST to backend
    // 3. Update initialEntities (mark as saved)
    // 4. Show success/error toast
  }

  // ═══════════════════════════════════════════════════════════
  // RENDER
  // ═══════════════════════════════════════════════════════════

  return (
    <div>
      {/* Header + Search */}
      {/* Layout with Sider + Content */}
      {/* Sider: Tree hierarchy */}
      {/* Content: Field tables + modals */}
      {/* Affix: Save button */}
    </div>
  )
}
```

### useEnhancedSemanticTerms Hook (150 lines)

```typescript
interface EnhancedSemanticTerm {
  id: string
  node_name: string
  description?: string
  qualified_path?: string
  properties?: any
  
  // Computed fields:
  businessName: string        // = node_name
  technicalName: string       // from properties or computed
  dataType: FieldType         // from properties
}

export function useEnhancedSemanticTerms(datasourceId?: string) {
  // Execute GraphQL query
  const { data, loading, error, refetch } = useQuery(GET_SEMANTIC_TERMS_WITH_METADATA, {
    variables: { datasourceId },
    skip: !datasourceId
  })

  // Enhance terms with computed fields
  const semanticTerms = useMemo(() => {
    if (!data?.semanticTerms) return []
    
    return data.semanticTerms.map(term => ({
      ...term,
      businessName: term.node_name,
      technicalName: term.properties?.technical_name || 
                     camelToSnake(term.node_name),
      dataType: term.properties?.data_type || 'text'
    }))
  }, [data])

  return { semanticTerms, loading, error, refetch }
}

// ───────────────────────────────────────────────────────────

export function semanticTermToField(
  term: EnhancedSemanticTerm, 
  sequence: number
): Field {
  return {
    key: `field-${uuidv4()}`,
    name: term.businessName,
    businessName: term.businessName,
    technicalName: term.technicalName,
    type: term.dataType,
    semanticTermId: term.id,            // ✅ REQUIRED
    semanticTermName: term.businessName, // ✅ REQUIRED
    isCore: false,
    sequence,
    lastModifiedAt: new Date().toISOString(),
    createdBy: getCurrentUser().email,
    description: term.description
  }
}

// ───────────────────────────────────────────────────────────

export function searchSemanticTerms(
  terms: EnhancedSemanticTerm[],
  query: string
): EnhancedSemanticTerm[] {
  if (!query) return terms
  
  const q = query.toLowerCase()
  return terms.filter(t => 
    t.businessName?.toLowerCase().includes(q) ||
    t.technicalName?.toLowerCase().includes(q) ||
    t.description?.toLowerCase().includes(q)
  )
}

// ───────────────────────────────────────────────────────────

export function groupSemanticTermsByCategory(
  terms: EnhancedSemanticTerm[]
): Map<string, EnhancedSemanticTerm[]> {
  const groups = new Map()
  
  terms.forEach(term => {
    const category = term.properties?.category || 'Uncategorized'
    if (!groups.has(category)) {
      groups.set(category, [])
    }
    groups.get(category).push(term)
  })
  
  return groups
}
```

---

## 📊 Type System (entity-schema.ts)

### Field Interface (Updated for v2.2)

```typescript
interface Field {
  // Identity
  key: string                              // Unique identifier
  name: string                             // Display name
  
  // Business & Technical Naming
  businessName: string                     // ✅ REQUIRED, User-facing
  technicalName: string                    // ✅ REQUIRED, System-facing (snake_case)
  
  // Semantic Link (✅ NEW: REQUIRED in v2.2)
  semanticTermId: string                   // ✅ REQUIRED, Link to catalog
  semanticTermName: string                 // ✅ REQUIRED, Semantic term name
  
  // Data Definition
  type: 'text' | 'number' | 'date' | 'boolean' | 'json' | 'array'
  
  // Classification
  isCore: boolean                          // true = inherited, false = assigned
  
  // Metadata (NEW in v2.2)
  description?: string                     // From semantic term
  sequence?: number                        // Display order (0, 1, 2...)
  lastModifiedAt?: string                  // ISO timestamp
  createdBy?: string                       // User email
}

// ───────────────────────────────────────────────────────────

interface Entity {
  name: string
  businessName: string
  technicalName: string
  description?: string
  isCore: boolean
  entity_fields: Field[]                   // Fields on entity
  customFields?: Field[]                   // Assigned (non-core) fields only
  subtypes: Record<string, Subtype>
}

// ───────────────────────────────────────────────────────────

interface Subtype {
  name: string
  businessName: string
  technicalName: string
  isCore: boolean
  subtype_fields: Field[]                  // Fields on subtype
  customFields?: Field[]                   // Assigned (non-core) fields only
}

// ───────────────────────────────────────────────────────────

type Entities = Record<string, Entity>
```

### Type Guards

```typescript
// Helper to identify inherited vs assigned fields
const isInheritedField = (field: Field): boolean => field.isCore

// Helper to validate semantic linkage
const hasSemanticLink = (field: Field): boolean => 
  !!field.semanticTermId && !!field.semanticTermName

// Helper to get assignable fields only
const getAssignedFields = (fields: Field[]): Field[] => 
  fields.filter(f => !f.isCore)
```

---

## 🔌 GraphQL Schema

### Query: GET_SEMANTIC_TERMS_WITH_METADATA

```graphql
query GetSemanticTermsWithMetadata($datasourceId: ID!) {
  semanticTerms(datasourceId: $datasourceId) {
    id                  # Unique term ID
    node_name           # Display name
    description         # Human-readable description
    qualified_path      # Full path in hierarchy
    properties {        # Metadata
      technical_name    # Snake_case name
      data_type         # text, number, date, boolean, json, array
      category          # e.g., entity, relationship, attribute
      tags              # e.g., ["financial", "account"]
      enum_values       # For enum types
      regex_pattern     # For validation
      min_value         # For numbers
      max_value         # For numbers
      required          # Boolean
    }
  }
}
```

### Backend Resolver (Pseudocode)

```go
// backend/internal/api/api.go

func (h *Handler) GetSemanticTerms(w http.ResponseWriter, r *http.Request) {
  datasourceID := r.URL.Query().Get("datasource_id")
  tenantID := r.Header.Get("X-Tenant-ID")
  
  // Query database
  rows := db.Query(`
    SELECT id, node_name, description, qualified_path, properties
    FROM catalog_node
    WHERE datasource_id = $1 AND tenant_id = $2
  `, datasourceID, tenantID)
  
  // Serialize to JSON
  var terms []SemanticTerm
  for rows.Next() {
    rows.Scan(&term.ID, &term.NodeName, ...)
    terms = append(terms, term)
  }
  
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(terms)
}
```

---

## 💾 Database Schema

### entity_schema Table

```sql
CREATE TABLE entity_schema (
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  entity_key VARCHAR(255) NOT NULL,
  schema_data JSONB NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  PRIMARY KEY (tenant_id, datasource_id, entity_key),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  FOREIGN KEY (datasource_id) REFERENCES datasources(id),
  
  INDEX idx_tenant_datasource (tenant_id, datasource_id)
);

-- schema_data structure (JSONB):
{
  "name": "Client Investor",
  "businessName": "Client Investor",
  "technicalName": "client_investor",
  "isCore": true,
  "entity_fields": [
    {
      "key": "investor_id",
      "businessName": "Investor ID",
      "technicalName": "investor_id",
      "type": "text",
      "semanticTermId": "sem-001",
      "semanticTermName": "Investor ID",
      "isCore": true,
      "sequence": 0
    },
    {
      "key": "f-abc123",
      "businessName": "Tax ID",
      "technicalName": "tax_id",
      "type": "text",
      "semanticTermId": "sem-tax-001",
      "semanticTermName": "Tax ID",
      "isCore": false,
      "sequence": 1,
      "lastModifiedAt": "2025-01-15T10:30:00Z",
      "createdBy": "user@company.com"
    }
  ],
  "subtypes": {
    "individual": {
      "name": "Individual Investor",
      "businessName": "Individual Investor",
      "technicalName": "individual_investor",
      "isCore": true,
      "subtype_fields": [
        {
          "key": "ssn",
          "businessName": "SSN",
          "technicalName": "ssn",
          "type": "text",
          "semanticTermId": "sem-ssn",
          "semanticTermName": "SSN",
          "isCore": true,
          "sequence": 0
        }
      ]
    }
  }
}
```

---

## ⚙️ State Management Strategy

### Immutable Updates (No Mutation)

```typescript
// ❌ WRONG: Mutating state directly
entities.client_investor.entity_fields.push(newField)
setEntities(entities)  // React won't detect change!

// ✅ CORRECT: Creating new objects
const updatedEntity = {
  ...entities.client_investor,
  entity_fields: [
    ...entities.client_investor.entity_fields,
    newField
  ]
}

setEntities({
  ...entities,
  client_investor: updatedEntity
})
```

### Delta Pattern (Track Changes)

```typescript
// Initial state (from backend)
initialEntities = { client_investor: {...}, ... }

// Current state (user modifications)
entities = { client_investor: {...with changes}, ... }

// Compute delta
changed = Object.keys(entities).filter(key => 
  JSON.stringify(entities[key]) !== JSON.stringify(initialEntities[key])
)
// Result: ['client_investor'] (only this entity changed)

// On save
POST /api/entity-schema
Body: {
  changed: { client_investor: entities.client_investor },
  deleted: []
}

// After success
setInitialEntities(entities)  // Reset baseline
// Now: initialEntities === entities (no pending changes)
```

---

## 🧪 Testing Strategy

### Unit Tests (Entity Schema Types)

```typescript
describe('semanticTermToField', () => {
  it('converts semantic term to field with all values', () => {
    const term = {
      id: 'sem-1',
      node_name: 'Tax ID',
      businessName: 'Tax ID',
      technicalName: 'tax_id',
      dataType: 'text'
    }
    
    const field = semanticTermToField(term, 0)
    
    expect(field.businessName).toBe('Tax ID')
    expect(field.technicalName).toBe('tax_id')
    expect(field.type).toBe('text')
    expect(field.semanticTermId).toBe('sem-1')  // ✅ Required
    expect(field.sequence).toBe(0)
    expect(field.isCore).toBe(false)
  })
  
  it('throws if semantic term missing required fields', () => {
    const invalidTerm = { id: 'sem-1', node_name: 'Test' }  // No technicalName
    // Should handle gracefully or throw
  })
})
```

### Integration Tests (Field Operations)

```typescript
describe('Field Operations', () => {
  it('adds field to entity via modal', async () => {
    render(<EntityConfigPageV3 />)
    
    // Select entity
    const tree = screen.getByRole('tree')
    userEvent.click(within(tree).getByText('Individual'))
    
    // Click Add
    const addButton = screen.getByText('Add Field')
    userEvent.click(addButton)
    
    // Search & select
    const search = screen.getByPlaceholderText(/Search semantic/)
    userEvent.type(search, 'tax')
    
    const addFieldButton = screen.getAllByText('Add')[0]
    userEvent.click(addFieldButton)
    
    // Verify field appears
    expect(screen.getByText('Tax ID')).toBeInTheDocument()
  })
})
```

### Performance Tests

```typescript
describe('Performance', () => {
  it('renders 100 fields without lag', () => {
    const largeFieldSet = Array.from({ length: 100 }, (_, i) => ({
      key: `f-${i}`,
      businessName: `Field ${i}`,
      technicalName: `field_${i}`,
      type: 'text',
      semanticTermId: `sem-${i}`,
      semanticTermName: `Field ${i}`,
      isCore: false
    }))
    
    const start = performance.now()
    render(<Table dataSource={largeFieldSet} />)
    const duration = performance.now() - start
    
    expect(duration).toBeLessThan(100)  // Should render in < 100ms
  })
})
```

---

## 🚀 Performance Optimizations

### 1. Memoization

```typescript
const hierarchyTree = useMemo(() => {
  // Recompute only if entities or searchTerm changes
  return buildHierarchy(entities, searchTerm)
}, [entities, searchTerm])
// Without useMemo, tree rebuilds on every render → lag

const filteredSemanticTerms = useMemo(() => {
  // Search is expensive, cache results
  return semanticTerms.filter(t => t.name.includes(searchTerm))
}, [semanticTerms, semanticTerm])
```

### 2. Lazy Loading

```typescript
// Semantic terms loaded only when needed
const { semanticTerms, loading } = useEnhancedSemanticTerms(datasource?.id)
// Query skipped if !datasource

// Modal rendered only when needed
{editingField && <Modal>...</Modal>}
// Not in DOM tree if modal closed → saves resources
```

### 3. Key Optimization (React Lists)

```typescript
{/* ✅ GOOD: Unique, stable key */}
<Table
  dataSource={fields}
  rowKey="key"  // Each field has unique key
/>

{/* ❌ BAD: Index-based key */}
<Table
  dataSource={fields}
  rowKey={(_, i) => i}  // Index changes when reordering!
/>
```

### 4. Debouncing (Future)

```typescript
// Current: Search fires on every keystroke
const handleSemanticSearch = (query) => {
  setSemanticSearchTerm(query)
}

// Future: Debounce to reduce filter calls
import { useDebouncedCallback } from 'use-debounce'

const handleSemanticSearch = useDebouncedCallback((query) => {
  setSemanticSearchTerm(query)
}, 300)  // Wait 300ms after typing stops
```

---

## 📝 Implementation Checklist

### Phase 1: Data Layer ✅ DONE
- [x] Update Field interface (semanticTermId required)
- [x] Create GraphQL query (GET_SEMANTIC_TERMS_WITH_METADATA)
- [x] Create useEnhancedSemanticTerms hook
- [x] Add semanticTermToField converter

### Phase 2: UI Components ✅ DONE
- [x] Create EntityConfigPageV3 (main component)
- [x] Implement side pane hierarchy
- [x] Implement field tables (inherited + assigned)
- [x] Implement add field modal
- [x] Implement reorder functionality
- [x] Implement delete functionality

### Phase 3: Documentation ✅ DONE
- [x] Full architecture guide (this file)
- [x] Quickstart guide
- [x] Features guide
- [x] API specs
- [x] Type documentation

### Phase 4: Testing (TODO)
- [ ] Unit tests for utility functions
- [ ] Integration tests for component
- [ ] E2E tests for workflows
- [ ] Performance benchmarks

### Phase 5: Deployment (TODO)
- [ ] Build & bundle verification
- [ ] Integration with Fabric Builder
- [ ] Production database migration
- [ ] User documentation & training

---

**Version:** v2.2 Architecture  
**Last Updated:** January 15, 2025  
**Maintained By:** GitHub Copilot  
**Next Phase:** v2.3 (Field editing, bulk operations, audit trail)
