# GraphQL, API, and Services Update Status

## 🔴 CURRENT STATUS: PARTIALLY UPDATED

The codebase **is NOT fully updated** to reflect the `bo_fields` normalization. Here's the detailed breakdown:

---

## ✅ What HAS Been Updated

### 1. Backend Service Layer (GOOD)
**File:** `backend/internal/services/businessobject_service.go`

The service **ALREADY** loads fields from the `bo_fields` table:

```go
// ✅ Correctly loads from bo_fields table
func (s *BusinessObjectService) loadBOSubtypesAndFields(ctx context.Context, bo *models.BusinessObjectDefinition) error {
    // Load entity-level fields
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
        return err
    }
    
    // Split into core/custom
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

**Status:** ✅ **CORRECT** - Already queries `bo_fields` table

---

### 2. Backend Models (GOOD)
**File:** `backend/internal/models/businessobjects.go`

The model struct is already correct:

```go
type BusinessObjectDefinition struct {
    ID             string
    Name           string
    DisplayName    string
    CoreFields     []FieldDefinition      // ✅ Loaded from bo_fields
    CustomFields   []FieldDefinition      // ✅ Loaded from bo_fields
    Subtypes       map[string]SubtypeDefinition
    // No 'Fields' JSONB here
}
```

**Status:** ✅ **CORRECT** - No JSONB fields in model

---

### 3. HTTP Handlers (UNCERTAIN)
**File:** `backend/internal/handlers/businessobject_handler.go`

- Handlers appear to publish commands via RabbitMQ
- Eventually delegates to `BusinessObjectService`
- **Status:** ⚠️ **LIKELY OK** - Delegates to service which is correct

---

## ❌ What NEEDS Updating

### 1. Frontend Components (BROKEN)
**File:** `frontend/src/pages/DynamicUIGeneratorPage.tsx`

Multiple components still reference old field structure:

```tsx
// ❌ PROBLEM: Accessing bo.fields directly
const getFieldById = (fieldId: string) => primaryBO.fields.find((f: BOField) => f.id === fieldId);

// ❌ PROBLEM: Accessing bo.fields array
fields={primaryBO.fields}

// ❌ PROBLEM: Accessing bo.fields.find()
const f = bo.fields.find(x => x.id === fid);

// ❌ PROBLEM: Accessing bo.fields.length
{primaryBO ? `${primaryBO.fields.length} fields ...` : 'No BO selected'}
```

**Fix Needed:**
```tsx
// ✅ CORRECT: Use coreFields and customFields
const allFields = [...(primaryBO?.coreFields || []), ...(primaryBO?.customFields || [])];
const getFieldById = (fieldId: string) => allFields.find((f: BOField) => f.id === fieldId);

// ✅ CORRECT: Use combined fields
fields={allFields}

// ✅ CORRECT: Use combined fields
const allFields = [...(bo.coreFields || []), ...(bo.customFields || [])];
const f = allFields.find(x => x.id === fid);

// ✅ CORRECT: Count from both arrays
const fieldCount = (primaryBO?.coreFields?.length || 0) + (primaryBO?.customFields?.length || 0);
{primaryBO ? `${fieldCount} fields ...` : 'No BO selected'}
```

**Status:** ❌ **BROKEN** - Still expects `bo.fields` array

---

### 2. Frontend Components - RelatedListConfigurator
**File:** `frontend/src/components/ui/RelatedListConfigurator.tsx`

```tsx
// ❌ PROBLEM: Accessing primaryBO.fields
{(primaryBO.fields.slice(0, 8)).map((f: any) => (...))}
```

**Fix Needed:**
```tsx
// ✅ CORRECT: Use combined fields
const allFields = [...(primaryBO?.coreFields || []), ...(primaryBO?.customFields || [])];
{(allFields.slice(0, 8)).map((f: any) => (...))}
```

**Status:** ❌ **BROKEN**

---

### 3. GraphQL Schema (OUTDATED)
**File:** `backend/graphql/relationship_suggestions.graphql`

The BusinessObject type doesn't expose fields:

```graphql
type BusinessObject {
  id: ID!
  name: String!
  kind: String!
  description: String
  createdAt: String!
  updatedAt: String!
  # ❌ Missing fields/coreFields/customFields
}
```

**Fix Needed:**
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
}
```

**Status:** ❌ **NEEDS UPDATE**

---

### 4. GraphQL Resolvers (LIKELY BROKEN)
**Status:** ❌ **UNKNOWN** - No resolver files found in search, but if they exist they likely don't populate fields

Need to verify:
- `backend/graphql/resolvers/*.go` or similar
- Need to check if they call `loadBOSubtypesAndFields()`

---

### 5. API Response Types (CHECK NEEDED)
If API returns `businessObject` DTOs, they might have an outdated structure.

**Potential Issues:**
- API might serialize `Fields` JSONB instead of `CoreFields`/`CustomFields`
- Frontend might not match backend response shape

---

### 6. Migration Seeds (OUTDATED)
**File:** `backend/internal/migrations/005_business_process_designer_seed.sql`

```sql
-- ❌ Still inserting into fields JSONB
INSERT INTO business_objects (name, display_name, description, fields, is_system)
VALUES (
    'Client',
    'Client',
    'Core client entity',
    '[{"key": "client_id", "name": "Client ID", ...}]'::jsonb,
    true
);
```

**Fix Needed:** After Migration 000031, need to:
1. Insert BO into `business_objects` table (without fields)
2. Insert fields into `bo_fields` table separately

**Status:** ❌ **BROKEN** - Still uses JSONB fields

---

### 7. Old Documentation Queries (OUTDATED)
Multiple documentation files still reference the old pattern:

- `BP_DESIGNER_COMPLETE_GUIDE.md` — references `fields` JSONB
- `BP_DESIGNER_IMPLEMENTATION_SUMMARY.md` — references `fields` JSONB
- Other `.md` files in documentation

**Status:** ⚠️ **Minor** - Doesn't affect code execution, but confusing for developers

---

## 🛠️ What Needs to Be Done

### Priority 1: CRITICAL (Breaking Changes)
- [ ] **Update Migration Seeds** — Fix `005_business_process_designer_seed.sql`
- [ ] **Update Frontend Components** — Fix `DynamicUIGeneratorPage.tsx` and `RelatedListConfigurator.tsx`
- [ ] **Update GraphQL Schema** — Add `coreFields` and `customFields` to BusinessObject type

### Priority 2: HIGH (Data Integrity)
- [ ] **Run Migration 000031** on all environments
- [ ] **Verify no code still reads from `business_objects.fields`** JSONB
- [ ] **Test API responses** match frontend expectations

### Priority 3: MEDIUM (Code Quality)
- [ ] **Find and update GraphQL resolvers** (if they exist)
- [ ] **Update API handler response types** if needed
- [ ] **Add unit tests** for bo_fields loading

### Priority 4: LOW (Documentation)
- [ ] Update `.md` documentation files
- [ ] Add comments explaining the normalization

---

## 🔍 Search Results to Check

Run these searches to find more places that need updating:

```bash
# Find all references to .fields
grep -r "\.fields" frontend/src --include="*.tsx" --include="*.ts"

# Find JSONB parsing in Go
grep -r "json.Unmarshal" backend/internal --include="*.go" | grep -i field

# Find old INSERT patterns
grep -r "INSERT INTO business_objects" backend --include="*.sql" | grep fields

# Find GraphQL resolvers
find backend -name "*resolver*" -o -name "*gql*"
```

---

## Recommended Rollout

### Step 1: Prepare
- [ ] Run Migration 000031 on staging
- [ ] Verify all old fields are migrated to bo_fields

### Step 2: Update Code
- [ ] Fix seed migration (005_business_process_designer_seed.sql)
- [ ] Update frontend components
- [ ] Update GraphQL schema + resolvers
- [ ] Add integration tests

### Step 3: Deploy
- [ ] Deploy backend + API changes
- [ ] Deploy frontend changes
- [ ] Verify bo_fields are loaded correctly in API responses

### Step 4: Cleanup
- [ ] Remove old code that handled JSONB fields
- [ ] Update documentation
- [ ] Add migration notes

---

## Example: How Frontend Should Query Now

### OLD (JSONB parsing)
```tsx
const bo = {
  fields: '[{"key": "id", "name": "ID"}, {"key": "name", "name": "Name"}]'
};
const fields = JSON.parse(bo.fields); // ❌ Manual parsing
```

### NEW (Already separated)
```tsx
const bo = {
  coreFields: [
    { key: "id", name: "ID", isCore: true },
    { key: "name", name: "Name", isCore: true }
  ],
  customFields: []
};
const fields = [...bo.coreFields, ...bo.customFields]; // ✅ Already parsed
```

---

## Quick Audit Checklist

```sql
-- Run these after Migration 000031

-- 1. Verify no BOs still using fields JSONB
SELECT COUNT(*) as bos_with_fields 
FROM business_objects 
WHERE fields IS NOT NULL;
-- Expected: 0

-- 2. Count migrated fields
SELECT COUNT(*) as total_fields FROM bo_fields;

-- 3. Verify all BOs have their fields
SELECT bo.id, bo.name, COUNT(bf.id) as field_count
FROM business_objects bo
LEFT JOIN bo_fields bf ON bf.business_object_id = bo.id
GROUP BY bo.id, bo.name
HAVING COUNT(bf.id) = 0;
-- Expected: 0 rows

-- 4. Check for orphaned fields
SELECT COUNT(*) as orphaned 
FROM bo_fields 
WHERE business_object_id NOT IN (SELECT id FROM business_objects);
-- Expected: 0
```

