# Multi-Entity Validation System Guide

## Overview

The **Multi-Entity Validation System** allows you to apply a single validation rule across multiple entities. This eliminates rule duplication and ensures consistency across your data model.

### Real-World Example: Phone Validation

Instead of creating three separate rules:
- ✅ Validate phone format for **Customer** entity
- ✅ Validate phone format for **Employee** entity  
- ✅ Validate phone format for **Supplier** entity

You can now create **ONE rule** that applies to all three entities:

```
Rule Name: Phone Number Format Validation
Rule Type: Field Format
Target Entity: (select all relevant entities)
Target Entities: [Customer, Employee, Supplier]  ← NEW FEATURE
Field: phone_number
Pattern: ^\+?[1-9]\d{1,14}$
Severity: error
```

## Key Features

### 1. **Searchable Entity Multi-Select**

In the "Apply to Entities" field:
- Click to open the entity selector dropdown
- Search for entities by typing (e.g., type "cust" → shows "Customer")
- Select multiple entities at once
- Leave empty to apply only to the single Target Entity (backward compatible)

**Entities Available:**
- Customer
- Employee
- Supplier
- Product
- Order
- OrderDetail
- Department
- (more can be added to the entity list in the database)

### 2. **Enhanced FK (Foreign Key) Picker**

When creating **Referential Integrity** rules:

#### Source Entity Dropdown
- Pre-populated list of available entities (Order, OrderDetail, Product, etc.)
- Validates that the source entity exists

#### Source Field Auto-Complete
- Searchable input for field names
- Suggestions include: `id`, `customer_id`, `product_id`, `order_id`, etc.
- Accepts custom field names via free-form input

#### Target Entity Dropdown
- Pre-populated list of available entities
- The entity containing the reference data

#### Target Field Auto-Complete
- Searchable input for the reference field
- Typically the primary key field (e.g., `id`)
- Also accepts custom field names

**Example FK Rule:**
```
Rule Name: Valid Customer Reference
Rule Type: Referential Integrity
Source Entity: Order
Source Field: customer_id
Target Entity: Customer
Target Field: id
```

This ensures that every `customer_id` in the Order table points to a valid `id` in the Customer table.

## Implementation Architecture

### Frontend Components

**Multi-Select Autocomplete:**
```tsx
<Autocomplete
  multiple
  options={['Customer', 'Employee', 'Supplier', 'Product', 'Order', 'OrderDetail', 'Department', 'global']}
  value={formData.target_entities || []}
  onChange={(event, newValue) => handleFormChange('target_entities', newValue)}
  // ... additional props for filtering and rendering
/>
```

**FK Source Field Auto-Complete:**
```tsx
<Autocomplete
  freeSolo  // Allow custom values
  options={['id', 'customer_id', 'employee_id', 'supplier_id', 'order_id', 'product_id', 'department_id', 'email', 'phone']}
  value={formData.ref_source_field}
  onChange={(event, newValue) => handleFormChange('ref_source_field', newValue || '')}
  // ... with filtering and input handling
/>
```

### Data Model

**Form Data Structure:**
```typescript
interface ValidationRuleFormData {
  rule_name: string;
  rule_type: 'field_format' | 'cardinality' | 'uniqueness' | 'referential_integrity' | 'business_logic';
  description: string;
  target_entity: string;              // Single entity (legacy)
  target_entities: string[];          // Multiple entities (NEW)
  severity: 'error' | 'warning' | 'info';
  is_active: boolean;
  // ... type-specific fields
  ref_source_entity: string;
  ref_source_field: string;
  ref_target_entity: string;
  ref_target_field: string;
}
```

### Backend Database

**New Column Required:**
```sql
ALTER TABLE catalog_validation_rules
ADD COLUMN IF NOT EXISTS target_entities TEXT[] DEFAULT ARRAY['global'];
```

**Query Logic (with multi-entity support):**
```sql
SELECT * FROM catalog_validation_rules
WHERE tenant_id = $1
  AND datasource_id = $2
  AND ('global' = ANY(target_entities) OR $3 = ANY(target_entities))
  AND is_active = true;
```

This query retrieves rules where:
- `'global'` is in the `target_entities` array (applies to all), **OR**
- The specific entity type (e.g., `'Customer'`) is in the array

## Usage Workflows

### Creating a Multi-Entity Validation Rule

1. **Navigate** to Catalog → Validation Rules
2. **Click** "Create New Validation Rule" button
3. **Fill Form:**
   - Rule Name: "Phone Number Format Validation"
   - Rule Type: "Field Format"
   - Target Entity: "Customer" (or primary entity)
   - **NEW:** Apply to Entities: Select "Customer", "Employee", "Supplier"
   - Field: "phone_number"
   - Pattern: `^\+?[1-9]\d{1,14}$`
   - Severity: "error"
4. **Click** "Save Validation Rule"
5. **Verify** in the rules table - rule now applies to all selected entities

### Creating an FK Rule with Dropdowns

1. **Click** "Create New Validation Rule"
2. **Select Rule Type:** "Referential Integrity"
3. **Fill FK Picker:**
   - Source Entity: (dropdown) → Select "Order"
   - Source Field: (autocomplete) → Type "customer_" or select "customer_id"
   - Target Entity: (dropdown) → Select "Customer"
   - Target Field: (autocomplete) → Select "id"
4. **Click** "Save Validation Rule"

### Editing an Existing Multi-Entity Rule

1. **Click** the ✏️ Edit icon on a rule row
2. **Update** the "Apply to Entities" field to add/remove entities
3. **Click** "Save Validation Rule"
4. **Changes are persisted** to the database

## API Integration

### Fetching Multi-Entity Rules

**Endpoint:** `GET /api/validation-rules?tenant_id=<ID>&datasource_id=<ID>&entity=Customer`

**Response:**
```json
[
  {
    "id": "rule-123",
    "rule_name": "Phone Number Format",
    "rule_type": "field_format",
    "target_entity": "Customer",
    "target_entities": ["Customer", "Employee", "Supplier"],
    "condition_json": { "pattern": "^\\+?[1-9]\\d{1,14}$", "field": "phone_number" },
    "severity": "error",
    "is_active": true
  }
]
```

### Creating a Multi-Entity Rule

**Endpoint:** `POST /api/validation-rules?tenant_id=<ID>&datasource_id=<ID>`

**Request Body:**
```json
{
  "rule_name": "Phone Validation Everywhere",
  "rule_type": "field_format",
  "target_entity": "Customer",
  "target_entities": ["Customer", "Employee", "Supplier"],
  "condition_json": {
    "field": "phone_number",
    "pattern": "^\\+?[1-9]\\d{1,14}$"
  },
  "severity": "error",
  "is_active": true
}
```

### Updating a Multi-Entity Rule

**Endpoint:** `PATCH /api/validation-rules/:id?tenant_id=<ID>&datasource_id=<ID>`

**Request Body:**
```json
{
  "target_entities": ["Customer", "Employee", "Supplier", "Department"]
}
```

## Performance Considerations

### Query Optimization

The multi-entity validation query uses PostgreSQL's `ANY()` operator:

```sql
WHERE 'global' = ANY(target_entities) OR 'Customer' = ANY(target_entities)
```

This is **efficient** because:
- PostgreSQL optimizes array comparisons
- Indexed scans work well with `ANY()` operator
- Consider adding a GIN index for large datasets:
  ```sql
  CREATE INDEX idx_rules_target_entities 
  ON catalog_validation_rules USING GIN (target_entities);
  ```

### Recommendation

For production deployments with many rules:
```sql
CREATE INDEX idx_rules_entity_lookup 
ON catalog_validation_rules (tenant_id, is_active, (target_entities))
WHERE is_active = true;
```

## Backward Compatibility

### Legacy Systems

The system maintains backward compatibility:

1. **Existing rules** with `target_entities = NULL` or `[]` fall back to using `target_entity` field
2. **New rules** can use either approach:
   - `target_entity` only (single entity, legacy mode)
   - `target_entities` (multi-entity, new mode)
3. **Migration is optional** - no need to update existing rules

### Migration Path

If you have many existing single-entity rules and want to consolidate:

1. Identify rules that should apply to multiple entities
2. Use the UI to create new multi-entity rules
3. Gradually deactivate old single-entity rules
4. Verify all entities are covered before deleting old rules

## Troubleshooting

### Issue: "Apply to Entities" field not showing

**Solution:** Ensure the component imports include `Autocomplete` and `OutlinedInput` from Material-UI

### Issue: Multi-entity rules not being applied

**Possible Causes:**
1. Backend database migration not run (missing `target_entities` column)
2. Backend engine not updated to use multi-entity query logic
3. Entity name doesn't match exactly (case-sensitive)

**Solution:**
1. Run database migration: `ALTER TABLE catalog_validation_rules ADD COLUMN IF NOT EXISTS target_entities TEXT[] DEFAULT ARRAY['global'];`
2. Verify backend validation logic includes multi-entity query
3. Check entity names in UI vs. database tables

### Issue: Can't find entity in dropdown

**Solution:**
1. Entity may need to be added to the available entity list in the UI
2. Update the Autocomplete `options` array to include the entity
3. Consider making the field `freeSolo` to allow custom entity names

## Advanced Features

### Global Rules

Set `target_entities: ['global']` to create rules that apply to ALL entities:

```json
{
  "rule_name": "Data Quality - Non-Null Check",
  "target_entities": ["global"],
  "condition_json": { "field": "created_at", "required": true }
}
```

### Entity-Specific Rules

Create rules for specific entity combinations:

```json
{
  "rule_name": "Financial Validation",
  "target_entities": ["Order", "Invoice", "Payment"],
  "condition_json": { ... }
}
```

### Dynamic Entity Fetching

To fetch available entities from the backend instead of hardcoding:

```tsx
const [entities, setEntities] = useState<string[]>([]);

useEffect(() => {
  fetch(`/api/entities?tenant_id=${tenant.id}`)
    .then(r => r.json())
    .then(data => setEntities(data.entities));
}, [tenant]);

// Then use: <Autocomplete options={entities} ... />
```

## Next Steps

1. ✅ **UI Implementation** - Multi-select entity picker added
2. ✅ **FK Picker Enhancement** - Dropdown and autocomplete components added
3. **Database Migration** - Run the `ALTER TABLE` command
4. **Backend Engine** - Update validation query logic for multi-entity support
5. **Testing** - Verify multi-entity rules across all entities
6. **Documentation** - Add to system documentation

## References

- **MUI Autocomplete:** https://mui.com/api/autocomplete/
- **PostgreSQL Array Functions:** https://www.postgresql.org/docs/current/functions-array.html
- **Validation Rules API:** `/backend/internal/api/validation_rules_routes.go`
- **Database Schema:** `/backend/migrations/create_validation_rules.sql`
