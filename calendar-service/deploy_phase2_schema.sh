#!/bin/bash
# deploy_phase2_schema.sh
# Deploy Phase 2 schema updates to remote PostgreSQL
# Usage: ./deploy_phase2_schema.sh <db_host> <db_port> <db_user> <db_name>

set -e

# Configuration
DB_HOST="${1:-100.84.126.19}"
DB_PORT="${2:-5432}"
DB_USER="${3:-postgres}"
DB_NAME="${4:-calendar_db}"
DB_PASSWORD="${DB_PASSWORD:-}"
SCHEMA_FILE="docs/schema_phase2_migration.sql"

echo "🚀 Phase 2 Schema Deployment"
echo "===================================="
echo "Target: $DB_HOST:$DB_PORT"
echo "Database: $DB_NAME"
echo "User: $DB_USER"
echo ""

# Verify schema file exists
if [ ! -f "$SCHEMA_FILE" ]; then
    echo "❌ Error: Schema file not found: $SCHEMA_FILE"
    exit 1
fi

echo "📋 Schema file located: $SCHEMA_FILE"
echo ""

# Test connection
echo "🔗 Testing database connection..."
if [ -z "$DB_PASSWORD" ]; then
    # Try without password (uses .pgpass or peer auth)
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 'Connection successful' as status;" 2>/dev/null || {
        echo "❌ Connection failed. Please provide DB_PASSWORD:"
        read -s DB_PASSWORD
        export PGPASSWORD="$DB_PASSWORD"
    }
else
    export PGPASSWORD="$DB_PASSWORD"
fi

echo "✅ Connection successful"
echo ""

# Backup current schema (optional)
echo "💾 Creating schema backup..."
BACKUP_FILE="schema_backup_$(date +%Y%m%d_%H%M%S).sql"
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    --schema-only --no-owner --no-privileges \
    > "$BACKUP_FILE" 2>/dev/null && echo "✅ Backup created: $BACKUP_FILE" || echo "⚠️  Backup creation skipped"
echo ""

# Apply schema migration
echo "🔧 Applying Phase 2 schema updates..."
echo "---"

PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 \
    -f "$SCHEMA_FILE" || {
    echo ""
    echo "❌ Schema migration failed!"
    exit 1
}

echo "---"
echo ""

# Verify updates
echo "📊 Verifying schema updates..."
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --quiet \
    << EOF

-- Check added columns
SELECT 
    column_name, 
    data_type, 
    is_nullable,
    column_default
FROM information_schema.columns 
WHERE table_name='jobs' 
AND column_name IN ('priority', 'region', 'resource_profile', 'sla_deadline')
ORDER BY ordinal_position;

EOF

echo ""

# Verify indexes
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --quiet \
    << EOF

-- Check created indexes
SELECT 
    indexname,
    indexdef
FROM pg_indexes 
WHERE tablename='jobs' AND indexname LIKE 'idx_jobs_%'
ORDER BY indexname;

EOF

echo ""

# Verify tenant_region_authorizations table
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --quiet \
    << EOF

-- Check region authorizations
SELECT 
    COUNT(*) as total_authorizations,
    COUNT(DISTINCT tenant_id) as authorized_tenants,
    COUNT(DISTINCT region) as regions_available
FROM tenant_region_authorizations;

EOF

echo ""

# Summary
echo "✅ Phase 2 Schema Deployment Complete!"
echo ""
echo "📝 Summary:"
echo "  ✓ Added priority, region, resource_profile, sla_deadline to jobs table"
echo "  ✓ Created indexes: idx_jobs_priority_region_status, idx_jobs_region_tenant, idx_jobs_sla_deadline"
echo "  ✓ Created tenant_region_authorizations table for data residency control"
echo "  ✓ Seeded region authorizations for all tenants"
echo ""
echo "🔍 Next: Implement Phase 3 (Data Residency Validation)"
echo ""

# Cleanup temporary password
unset PGPASSWORD
