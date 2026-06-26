# Normalizing business_objects.fields JSONB

## TL;DR
- ✅ **YES**, drop the `fields` JSONB column
- ✅ Use `bo_fields` table instead (one row per attribute)
- ✅ Run Migration 000031 to extract and normalize existing data
- ✅ Simpler queries, better performance, type-safe

---

## What Needs to Change

### 1. Database Schema (Migration 000031)
**File:** `/backend/migrations/000031_normalize_bo_fields.sql`

- Extracts all fields from `business_objects.fields` JSONB
- Inserts them as rows in `bo_fields` table
- Drops the `fields` column
- **Status:** ✅ Created

---

### 2. Backend Code Changes

#### In `businessobject_service.go`:

**Before:**
```go
bo := &models.BusinessObjectDefinition{
    ID:           id,
    Name:         req.Name,
    DisplayName:  req.DisplayName,
    Description:  req.Description,
    CoreFields:   []models.FieldDefinition{},    // ❌ Will be loaded separately
    CustomFields: []models.FieldDefinition{},    // ❌ Will be loaded separately
}
```

**After:**
```go
bo := &models.BusinessObjectDefinition{
    ID:           id,
    Name:         req.Name,
    DisplayName:  req.DisplayName,
    Description:  req.Description,
    // Fields are no longer in the BO struct itself
    // They're loaded via a separate query or JOIN
}

// Load fields separately
fields, err := s.LoadFieldsForBO(ctx, boID)
bo.CoreFields = filterCoreFields(fields)
bo.CustomFields = filterCustomFields(fields)
```

#### In models:

**No change needed** to `BusinessObjectDefinition` struct — it already has `CoreFields` and `CustomFields` as separate slices. The service just needs to load them differently.

```go
// In businessobjects.go (already correct):
type BusinessObjectDefinition struct {
    ID           string
    Name         string
    DisplayName  string
    Description  string
    CoreFields   []FieldDefinition   // ✅ Loaded from bo_fields table
    CustomFields []FieldDefinition   // ✅ Loaded from bo_fields table
    // ... (no fields JSONB here)
}
```

---

### 3. Service Layer Methods to Add/Update

#### New method: `LoadFieldsForBO`
```go
// LoadFieldsForBO fetches all fields for a business object
func (s *BusinessObjectService) LoadFieldsForBO(ctx context.Context, tenantID, boID string) ([]models.FieldDefinition, error) {
    query := `
        SELECT id, key, name, display_name, technical_name, type, is_core, 
               is_required, is_system, description, reference_entity, sequence,
               created_at, created_by, last_modified_at, last_modified_by
        FROM public.bo_fields
        WHERE business_object_id = $1 
          AND tenant_id = $2
          AND subtype_id IS NULL
        ORDER BY sequence
    `
    
    var fields []models.FieldDefinition
    err := s.db.SelectContext(ctx, &fields, query, boID, tenantID)
    return fields, err
}
```

#### Update: `GetBusinessObject`
```go
func (s *BusinessObjectService) GetBusinessObject(ctx context.Context, tenantID, boID string) (*models.BusinessObjectDefinition, error) {
    // 1. Query business_objects table
    var bo models.BusinessObjectDefinition
    err := s.db.GetContext(ctx, &bo, `
        SELECT id, name, display_name, description, icon, is_system, created_at, updated_at, tenant_id
        FROM public.business_objects
        WHERE id = $1 AND tenant_id = $2
    `, boID, tenantID)
    if err != nil {
        return nil, err
    }
    
    // 2. Load fields separately
    fields, err := s.LoadFieldsForBO(ctx, tenantID, boID)
    if err != nil {
        return nil, err
    }
    
    // 3. Split into core and custom
    bo.CoreFields = filterCoreFields(fields)
    bo.CustomFields = filterCustomFields(fields)
    
    return &bo, nil
}
```

#### Update: `CreateBusinessObject`
```go
func (s *BusinessObjectService) CreateBusinessObject(
    ctx context.Context,
    tenantID string,
    req models.CreateBusinessObjectRequest,
    userID string,
) (*models.BusinessObjectDefinition, error) {
    boID := uuid.New().String()
    now := time.Now()
    
    // 1. Insert BO definition (without fields)
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO public.business_objects 
            (id, tenant_id, name, display_name, description, icon, is_system, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `, boID, tenantID, req.Name, req.DisplayName, req.Description, req.Icon, false, now, now)
    if err != nil {
        return nil, err
    }
    
    // 2. Insert fields separately (if any provided in req)
    if req.Fields != nil && len(req.Fields) > 0 {
        for i, field := range req.Fields {
            _, err := s.db.ExecContext(ctx, `
                INSERT INTO public.bo_fields 
                    (tenant_id, business_object_id, key, name, display_name, technical_name, type,
                     is_core, is_required, is_system, description, reference_entity, sequence,
                     created_at, created_by, last_modified_at, last_modified_by)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
            `, tenantID, boID, field.Key, field.Name, field.DisplayName, field.TechnicalName, 
               field.Type, field.IsCore, field.IsRequired, field.IsSystem, field.Description,
               field.ReferenceEntity, i, now, userID, now, userID)
            if err != nil {
                return nil, err
            }
        }
    }
    
    // 3. Fetch and return complete BO
    return s.GetBusinessObject(ctx, tenantID, boID)
}
```

---

### 4. Frontend Changes

#### In TypeScript services:
```ts
// Before (parsing JSONB):
const fields = JSON.parse(boData.fields);
const coreFields = fields.filter(f => f.is_core);

// After (fields already normalized):
const coreFields = boData.coreFields;
const customFields = boData.customFields;
```

#### In React components:
```tsx
// No changes needed — components already expect 
// bo.coreFields and bo.customFields as arrays
```

---

### 5. Queries That Change

#### Get all BOs with their fields:

**Before:**
```sql
SELECT bo.id, bo.name, bo.fields
FROM business_objects bo
WHERE bo.tenant_id = $1;
-- Then parse JSON in app
```

**After:**
```sql
SELECT 
    bo.id, 
    bo.name, 
    bo.display_name,
    bf.id as field_id,
    bf.key,
    bf.name as field_name,
    bf.type,
    bf.is_core
FROM business_objects bo
LEFT JOIN bo_fields bf ON bf.business_object_id = bo.id
WHERE bo.tenant_id = $1
ORDER BY bo.id, bf.sequence;
```

#### Search fields by name:

**Before:**
```sql
-- Not easily queryable
-- Requires JSONB operators
SELECT bo.id, bo.name
FROM business_objects bo
WHERE bo.fields @> '[{"name": "Customer ID"}]'
  AND bo.tenant_id = $1;
```

**After:**
```sql
-- Simple, fast query
SELECT DISTINCT bo.id, bo.name
FROM business_objects bo
JOIN bo_fields bf ON bf.business_object_id = bo.id
WHERE bf.name ILIKE '%Customer ID%'
  AND bo.tenant_id = $1;
```

---

### 6. Testing

#### Update unit tests:

```go
func TestGetBusinessObjectWithFields(t *testing.T) {
    service := setupTestService()
    
    // Create BO
    bo, err := service.CreateBusinessObject(ctx, tenantID, 
        CreateBusinessObjectRequest{
            Name: "Customer",
            Fields: []FieldDefinition{
                {Key: "id", Name: "ID", Type: "text"},
                {Key: "email", Name: "Email", Type: "email"},
            },
        }, userID)
    assert.NoError(t, err)
    
    // Fetch and verify fields are loaded
    retrieved, err := service.GetBusinessObject(ctx, tenantID, bo.ID)
    assert.NoError(t, err)
    assert.Equal(t, 2, len(retrieved.CoreFields) + len(retrieved.CustomFields))
}
```

---

## Rollout Plan

### Phase 1: Deploy Migration
1. Run Migration 000031 on dev/staging
2. Verify data integrity (row counts match)
3. Deploy to production

### Phase 2: Deploy Code
1. Update service layer to use `bo_fields` table
2. Update API handlers
3. Update frontend
4. Deploy simultaneously to avoid API contract changes

### Phase 3: Cleanup (Optional)
- Remove any legacy code that handled JSONB fields
- Remove old helper functions for JSON parsing
- Update API documentation

---

## Verification Checklist

After migration:

```sql
-- ✅ Check migration success
SELECT 'BO count' as check, COUNT(*) as count FROM business_objects
UNION ALL
SELECT 'Field count' as check, COUNT(*) as count FROM bo_fields;

-- ✅ Verify no BOs missing fields (if they had them)
SELECT bo.id, bo.name, COUNT(bf.id) as field_count
FROM business_objects bo
LEFT JOIN bo_fields bf ON bf.business_object_id = bo.id
GROUP BY bo.id, bo.name
ORDER BY field_count DESC;

-- ✅ Check for orphaned fields (shouldn't exist)
SELECT COUNT(*) FROM bo_fields bf
WHERE NOT EXISTS (SELECT 1 FROM business_objects bo WHERE bo.id = bf.business_object_id);

-- ✅ Verify fields JSONB column is gone
SELECT column_name 
FROM information_schema.columns 
WHERE table_name = 'business_objects' 
  AND column_name = 'fields';
-- Result: should be empty/no rows
```

---

## Benefits Summary

| Metric | Before | After |
|--------|--------|-------|
| Query complexity | High (JSONB parsing) | Low (simple JOINs) |
| Index efficiency | GIN only | B-tree + partial |
| Data integrity | Manual validation | FK constraints |
| Query performance | 🐢 Slow on large datasets | 🚀 Fast, indexed |
| Update atomicity | Replace all fields | Update individual fields |
| Maintainability | Complex application logic | Standard SQL |

