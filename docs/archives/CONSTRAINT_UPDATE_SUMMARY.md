# Tenant Product Datasource Constraint Update

**Date**: December 10, 2025  
**Status**: ✅ Complete

## Summary

Updated the `tenant_product_datasource` table constraint to allow multiple datasources of the same type (e.g., multiple PostgreSQL databases) per tenant-product combination.

## Changes Made

### 1. Database Constraint Change

**Old Constraint:**
```sql
CONSTRAINT tenant_product_datasource_uniq UNIQUE (tenant_product_id, alpha_datasource_id)
```
- ❌ Prevented multiple datasources of the same type
- ❌ Could only have one PostgreSQL datasource per tenant-product

**New Constraint:**
```sql
CONSTRAINT tenant_product_datasource_source_uniq UNIQUE (tenant_product_id, source_name)
```
- ✅ Allows multiple datasources of the same type
- ✅ Requires unique `source_name` per tenant-product
- ✅ Can have "northwinds", "datamart", "analytics" PostgreSQL databases

### 2. Files Updated

#### Migrations
- ✅ `/hasura/migrations/alpha/20251210_fix_tenant_product_datasource_constraint/up.sql` - New migration
- ✅ `/hasura/migrations/alpha/20251210_fix_tenant_product_datasource_constraint/down.sql` - Rollback migration
- ✅ `/hasura/migrations/default/20251102_add_tenant_product_tables/up.sql` - Updated constraint definition
- ✅ `/hasura/metadata/databases/alpha/tables/tables.yaml` - Already includes connections table

#### Schema Files
- ✅ `/backend/totalddl.sql` - Updated constraint definition

### 3. Database Applied

The constraint change was successfully applied to the alpha database:

```bash
# Verified with:
SELECT conname, pg_get_constraintdef(oid) 
FROM pg_constraint 
WHERE conrelid = 'public.tenant_product_datasource'::regclass 
AND conname LIKE '%uniq%';

# Result:
tenant_product_datasource_source_uniq | UNIQUE (tenant_product_id, source_name)
```

### 4. Verification

Successfully created multiple PostgreSQL datasources for the same tenant-product:

```json
{
  "tenant_product_datasources": [
    {
      "id": "982aef38-418f-46dc-acd0-35fe8f3b97b0",
      "source_name": "northwinds",
      "alpha_datasource_id": "3f5066fb-a8d9-4087-8bbf-c8fe1cbcd747"
    },
    {
      "id": "097793ae-ceeb-48f4-a7bb-4bcbba1796ae",
      "source_name": "datamart",
      "alpha_datasource_id": "3f5066fb-a8d9-4087-8bbf-c8fe1cbcd747"
    }
  ]
}
```

## Impact Analysis

### ✅ What Works Now
- Multiple PostgreSQL datasources per tenant-product
- Multiple Snowflake datasources per tenant-product
- Multiple SQL Server datasources per tenant-product
- Each datasource must have a unique `source_name`

### ⚠️ Validation Required
- Frontend forms should validate that `source_name` is unique before submission
- Backend should handle constraint violation errors gracefully
- Consider adding a frontend check before insert to show friendly error messages

### 🔄 No Changes Required
- Hasura GraphQL CRUD operations work automatically
- Backend queries using `tenant_product_datasource` table are unaffected
- Frontend components querying datasources continue to work
- All foreign key relationships remain intact

## Future Considerations

1. **Frontend Validation**: Add client-side validation to prevent duplicate `source_name` entries
2. **Error Handling**: Improve error messages for constraint violations
3. **Documentation**: Update API documentation to reflect the new constraint
4. **Migration Path**: For new environments, ensure migrations are run in order

## Rollback Procedure

If needed, rollback using the down migration:

```sql
ALTER TABLE public.tenant_product_datasource 
DROP CONSTRAINT IF EXISTS tenant_product_datasource_source_uniq;

ALTER TABLE public.tenant_product_datasource 
ADD CONSTRAINT tenant_product_datasource_uniq 
UNIQUE (tenant_product_id, alpha_datasource_id);
```

## Related Issues

- Resolved: Uniqueness violation when adding multiple datasources of same type
- Resolved: Constraint name inconsistency across environments
- Verified: Hasura metadata reload successful
- Verified: GraphQL mutations working correctly
