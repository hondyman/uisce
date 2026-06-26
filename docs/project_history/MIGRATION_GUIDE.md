# Business Objects Consolidation Migration Guide

## Overview
This migration consolidates all business objects (Client Investor, Customer, Portfolio, Trade) into the centralized `business_objects` table in PostgreSQL.

## What Gets Consolidated

### Business Objects (4 total)
1. **Client Investor** - with 2 subtypes (Individual, Institutional)
2. **Customer** - with 3 subtypes (Retail, Industry, Government)
3. **Portfolio** - with 1 subtype (Discretionary)
4. **Trade** - with 2 subtypes (Regular, Block Trade)

### Total Schema Objects
- **Business Objects**: 4
- **Subtypes**: 8
- **Fields**: 20+ (5 base fields per BO, plus subtype-specific fields)

## Prerequisites

1. **PostgreSQL Connection**: Ensure you have access to the `alpha` database
2. **Tenant Setup**: Verify that at least one tenant exists in the `tenants` table

## Steps to Execute Migration

### Step 1: Connect to PostgreSQL
```bash
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable
```

### Step 2: Run the Migration
```bash
\i migrations/000_consolidate_business_objects.sql
```

Or from the command line:
```bash
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable < migrations/000_consolidate_business_objects.sql
```

### Step 3: Verify the Migration

#### Check Business Objects
```sql
SELECT key, name, display_name, is_core, category FROM public.business_objects ORDER BY key;
```

**Expected Output:**
```
        key         |     name      |  display_name   | is_core |       category       
--------------------+---------------+-----------------+---------+----------------------
 client_investor    | Client Investor   | Client Investor | t       | Customer & Relationships
 customer           | Customer          | Customer        | t       | Customer & Relationships
 portfolio          | Portfolio         | Portfolio       | t       | Financial Assets
 trade              | Trade             | Trade           | t       | Financial Transactions
```

#### Check Subtypes
```sql
SELECT 
    bo.key as bo_key, 
    st.key as subtype_key, 
    st.display_name 
FROM public.bo_subtypes st
JOIN public.business_objects bo ON st.business_object_id = bo.id
ORDER BY bo.key, st.sequence;
```

**Expected Output:**
```
        bo_key         |    subtype_key     |     display_name      
--------------------+--------------------+-----------------------
 client_investor    | individual         | Individual Investor
 client_investor    | institutional      | Institutional Investor
 customer           | retail_customer    | Retail Customer
 customer           | industry_customer  | Industry Customer
 customer           | government_customer| Government Customer
 portfolio          | discretionary      | Discretionary Portfolio
 trade              | regular            | Regular Trade
 trade              | block_trade        | Block Trade
```

#### Check Fields Count
```sql
SELECT 
    bo.key as bo_key,
    COUNT(*) as field_count
FROM public.bo_fields f
JOIN public.business_objects bo ON f.business_object_id = bo.id
GROUP BY bo.key
ORDER BY bo.key;
```

**Expected Output:**
```
     bo_key      | field_count
-----------------+-------------
 client_investor |           5
 customer        |           5
 portfolio       |           4
 trade           |           5
```

## Rollback Instructions

If you need to rollback, run:

```sql
BEGIN;

DELETE FROM public.bo_fields WHERE business_object_id IN (
  SELECT id FROM public.business_objects 
  WHERE key IN ('client_investor', 'customer', 'portfolio', 'trade')
);

DELETE FROM public.bo_subtypes WHERE business_object_id IN (
  SELECT id FROM public.business_objects 
  WHERE key IN ('client_investor', 'customer', 'portfolio', 'trade')
);

DELETE FROM public.business_objects 
WHERE key IN ('client_investor', 'customer', 'portfolio', 'trade');

COMMIT;
```

## Frontend Sync

After running this migration, the frontend application will automatically load these consolidated business objects when you:

1. Navigate to the Entity Configuration page
2. The `fetchEntitySchema()` API call will retrieve all business objects from the database
3. The EntityDrawerTreeView will display all 4 business objects in a unified view

### API Endpoints Affected
- `GET /api/entity-schema` - Returns all consolidated business objects
- `POST /api/entity-schema` - Saves updates to business objects

## Troubleshooting

### Error: "Tenant not found"
**Solution**: Insert a test tenant first:
```sql
INSERT INTO public.tenants (id, name, display_name, created_at)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  'Default Tenant',
  'Default Tenant',
  now()
) ON CONFLICT DO NOTHING;
```

### Error: "Foreign key constraint"
**Solution**: Ensure all tenant IDs are valid UUID format and exist in the `tenants` table.

### Migration hangs
**Solution**: Check for long-running transactions:
```sql
SELECT * FROM pg_stat_activity WHERE state = 'active';
```

## Performance Impact
- Migration time: < 1 second
- No downtime required
- Database size increase: Negligible (< 1 MB)

## Next Steps

1. ✅ Run the migration script
2. ✅ Verify consolidation in PostgreSQL
3. ✅ Test the frontend Entity Config page
4. ✅ Validate that all business objects appear in the UI
5. ✅ Update validation rules to reference consolidated BOs (if needed)

