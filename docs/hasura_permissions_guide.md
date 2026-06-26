# Hasura Permissions Configuration Guide

This guide explains how to configure Hasura permissions for multi-tenant security.

## Overview

Hasura permissions are configured per table and per role. The JWT token includes:
- `x-hasura-user-id`: User's UUID
- `x-hasura-tenant-id`: User's tenant UUID  
- `x-hasura-allowed-roles`: List of roles (e.g., `["user", "global_admin"]`)
- `x-hasura-default-role`: Default role to use

## Roles

### 1. `user` Role (Standard Tenant Users)

**Purpose:** Standard users who can only access their own tenant's data.

**Configuration for each table:**

**Select Permission:**
```json
{
  "tenant_id": {
    "_eq": "X-Hasura-Tenant-Id"
  }
}
```

**Insert Permission:**
```json
{
  "tenant_id": {
    "_eq": "X-Hasura-Tenant-Id"
  }
}
```
**Column Presets:**
```json
{
  "tenant_id": "X-Hasura-Tenant-Id"
}
```

**Update Permission:**
```json
{
  "tenant_id": {
    "_eq": "X-Hasura-Tenant-Id"
  }
}
```

**Delete Permission:**
```json
{
  "tenant_id": {
    "_eq": "X-Hasura-Tenant-Id"
  }
}
```

### 2. `global_admin` Role (Uisce Organization Admins)

**Purpose:** Support staff who can access all tenants' data.

**Configuration for each table:**

**All Permissions (Select, Insert, Update, Delete):**
```json
{}
```
*(Empty object = no filter, full access)*

**Column Presets for Insert:**
```json
{}
```
*(No presets - admins can set any tenant_id)*

## Tables Requiring Configuration

Apply the above permissions to these tables:

### Core Catalog
- `catalog_node`
- `catalog_edge`
- `node_types`
- `edge_types`

### Business Objects
- `business_objects`
- `business_object_instances`

### Workflows
- `processes`
- `process_versions`
- `validation_rules`
- `event_handlers`
- `step_templates`
- `rule_templates`
- `process_execution_log`
- `process_designer_permissions`

### Configuration Tables
- `process_step_types`
- `validation_operators`
- `workflow_events`

### Pages & Pipelines
- `page_layouts`
- `pipelines`

### Tenant Management
- `tenant_datasources`
- `tenant_connections`
- `tenants` (read-only for users, full access for global_admin)
- `users` (users can only see users in their tenant)

## Step-by-Step Configuration

### Via Hasura Console

1. **Open Hasura Console:** Navigate to `http://localhost:8081/console`

2. **For each table:**
   - Go to "Data" → Select table → "Permissions" tab
   
3. **Configure `user` role:**
   - Click "Enter new role" → Type "user" → Enter
   - For each operation (select, insert, update, delete):
     - Click the ✏️ icon
     - Set "Row select permissions" to custom check
     - Paste the JSON filter: `{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}`
     - For insert, add column preset: `tenant_id` = `X-Hasura-Tenant-Id`
     - Select allowed columns
     - Save

4. **Configure `global_admin` role:**
   - Click "Enter new role" → Type "global_admin" → Enter
   - For each operation:
     - Click the ✏️ icon
     - Select "Without any checks" (empty `{}` filter)
     - Select all columns
     - Save

### Via Hasura Metadata API

You can also configure permissions programmatically:

```bash
# Example: Set permissions for catalog_node table
curl -X POST http://localhost:8081/v1/metadata \
  -H "Content-Type: application/json" \
  -H "x-hasura-admin-secret: myadminsecretkey" \
  -d '{
    "type": "pg_create_select_permission",
    "args": {
      "table": "catalog_node",
      "role": "user",
      "permission": {
        "columns": "*",
        "filter": {
          "tenant_id": {
            "_eq": "X-Hasura-Tenant-Id"
          }
        }
      }
    }
  }'
```

## Testing Permissions

### Test as Standard User

```bash
# Login to get JWT token
TOKEN=$(curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@tenant-a.com", "password": "password"}' \
  | jq -r '.access_token')

# Query Hasura with user token
curl -X POST http://localhost:8080/v1/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "query": "{ catalog_node { id node_name tenant_id } }"
  }'

# Should only return nodes for tenant-a
```

### Test as Global Admin

```bash
# Login as global admin
ADMIN_TOKEN=$(curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@uisce.com", "password": "password"}' \
  | jq -r '.access_token')

# Query Hasura with admin token
curl -X POST http://localhost:8080/v1/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "query": "{ catalog_node { id node_name tenant_id } }"
  }'

# Should return nodes for ALL tenants
```

## Verification Checklist

- [ ] All tables have `user` role configured with tenant_id filter
- [ ] All tables have `global_admin` role with no filters
- [ ] Insert operations have column presets for tenant_id
- [ ] Test queries return correct data for each role
- [ ] Verify users cannot access other tenants' data
- [ ] Verify global admins can access all data

## Common Issues

### Issue: "field not found in type"
**Solution:** Ensure the table has a `tenant_id` column in the database.

### Issue: "permission denied"
**Solution:** Check that the role is in `x-hasura-allowed-roles` in the JWT token.

### Issue: "no rows returned"
**Solution:** Verify `x-hasura-tenant-id` is set in JWT and matches data in the table.

## Next Steps

After configuring Hasura permissions:
1. Test with real user accounts
2. Configure Superset security (Phase 11.5)
3. Implement comprehensive audit logging (Phase 11.6)
4. Run security penetration tests (Phase 11.7)
