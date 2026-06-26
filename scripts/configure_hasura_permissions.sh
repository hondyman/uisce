#!/bin/bash

# Hasura Permissions Configuration Script
# This script configures Row-Level Security permissions in Hasura for multi-tenant security

set -e

HASURA_URL="${HASURA_URL:-http://localhost:8081}"
HASURA_ADMIN_SECRET="${HASURA_ADMIN_SECRET:-myadminsecretkey}"

echo "🔐 Configuring Hasura Permissions for Multi-Tenant Security"
echo "Hasura URL: $HASURA_URL"
echo ""

# Function to create permission
create_permission() {
    local table=$1
    local role=$2
    local permission_type=$3
    local filter=$4
    local columns=${5:-"*"}
    local preset=${6:-"{}"}

    echo "  → Setting $permission_type permission for $role on $table"
    
    curl -s -X POST "$HASURA_URL/v1/metadata" \
        -H "Content-Type: application/json" \
        -H "x-hasura-admin-secret: $HASURA_ADMIN_SECRET" \
        -d "{
            \"type\": \"pg_create_${permission_type}_permission\",
            \"args\": {
                \"table\": \"$table\",
                \"role\": \"$role\",
                \"permission\": {
                    \"columns\": $columns,
                    \"filter\": $filter,
                    \"set\": $preset
                }
            }
        }" > /dev/null 2>&1 || echo "    (permission may already exist)"
}

# Function to configure table permissions
configure_table() {
    local table=$1
    echo ""
    echo "📋 Configuring table: $table"
    
    # User role - tenant isolation
    create_permission "$table" "user" "select" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}'
    create_permission "$table" "user" "insert" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}' "*" '{"tenant_id": "X-Hasura-Tenant-Id"}'
    create_permission "$table" "user" "update" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}'
    create_permission "$table" "user" "delete" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}'
    
    # Global admin role - full access
    create_permission "$table" "global_admin" "select" '{}'
    create_permission "$table" "global_admin" "insert" '{}'
    create_permission "$table" "global_admin" "update" '{}'
    create_permission "$table" "global_admin" "delete" '{}'
}

echo "Starting Hasura permissions configuration..."
echo ""

# Core catalog tables
configure_table "catalog_node"
configure_table "catalog_edge"
configure_table "node_types"
configure_table "edge_types"

# Business objects
configure_table "business_objects"
configure_table "business_object_instances"

# Workflows
configure_table "processes"
configure_table "process_versions"
configure_table "validation_rules"
configure_table "event_handlers"
configure_table "step_templates"
configure_table "rule_templates"
configure_table "process_execution_log"
configure_table "process_designer_permissions"

# Configuration tables (allow NULL tenant_id for system-wide configs)
echo ""
echo "📋 Configuring table: process_step_types (allows system configs)"
create_permission "process_step_types" "user" "select" '{"_or": [{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}, {"tenant_id": {"_is_null": true}}]}'
create_permission "process_step_types" "user" "insert" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}' "*" '{"tenant_id": "X-Hasura-Tenant-Id"}'
create_permission "process_step_types" "user" "update" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}'
create_permission "process_step_types" "user" "delete" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}'
create_permission "process_step_types" "global_admin" "select" '{}'
create_permission "process_step_types" "global_admin" "insert" '{}'
create_permission "process_step_types" "global_admin" "update" '{}'
create_permission "process_step_types" "global_admin" "delete" '{}'

echo ""
echo "📋 Configuring table: validation_operators (allows system configs)"
create_permission "validation_operators" "user" "select" '{"_or": [{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}, {"tenant_id": {"_is_null": true}}]}'
create_permission "validation_operators" "user" "insert" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}' "*" '{"tenant_id": "X-Hasura-Tenant-Id"}'
create_permission "validation_operators" "user" "update" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}'
create_permission "validation_operators" "user" "delete" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}'
create_permission "validation_operators" "global_admin" "select" '{}'
create_permission "validation_operators" "global_admin" "insert" '{}'
create_permission "validation_operators" "global_admin" "update" '{}'
create_permission "validation_operators" "global_admin" "delete" '{}'

echo ""
echo "📋 Configuring table: workflow_events (allows system configs)"
create_permission "workflow_events" "user" "select" '{"_or": [{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}, {"tenant_id": {"_is_null": true}}]}'
create_permission "workflow_events" "user" "insert" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}' "*" '{"tenant_id": "X-Hasura-Tenant-Id"}'
create_permission "workflow_events" "user" "update" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}'
create_permission "workflow_events" "user" "delete" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}'
create_permission "workflow_events" "global_admin" "select" '{}'
create_permission "workflow_events" "global_admin" "insert" '{}'
create_permission "workflow_events" "global_admin" "update" '{}'
create_permission "workflow_events" "global_admin" "delete" '{}'

# Pages and pipelines
configure_table "page_layouts"
configure_table "pipelines"

# Tenant management
configure_table "tenant_datasources"
configure_table "tenant_connections"

# Users table (special handling)
echo ""
echo "📋 Configuring table: users"
create_permission "users" "user" "select" '{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}'
create_permission "users" "user" "update" '{"_and": [{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}, {"id": {"_eq": "X-Hasura-User-Id"}}]}'
create_permission "users" "global_admin" "select" '{}'
create_permission "users" "global_admin" "insert" '{}'
create_permission "users" "global_admin" "update" '{}'
create_permission "users" "global_admin" "delete" '{}'

# Tenants table (read-only for users)
echo ""
echo "📋 Configuring table: tenants (read-only for users)"
create_permission "tenants" "user" "select" '{"id": {"_eq": "X-Hasura-Tenant-Id"}}'
create_permission "tenants" "global_admin" "select" '{}'
create_permission "tenants" "global_admin" "insert" '{}'
create_permission "tenants" "global_admin" "update" '{}'
create_permission "tenants" "global_admin" "delete" '{}'

echo ""
echo "✅ Hasura permissions configuration complete!"
echo ""
echo "Next steps:"
echo "1. Test with a standard user: curl -X POST http://localhost:8080/api/auth/login ..."
echo "2. Test GraphQL queries with the JWT token"
echo "3. Verify tenant isolation is working"
echo ""
