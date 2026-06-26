#!/bin/bash
# Script to manually drop Apache AGE extension from local PostgreSQL database
# Run this after running the migration: 20260123_drop_age_extension.up.sql

set -e

echo "Connecting to PostgreSQL to drop AGE extension..."

# Database connection details
DB_NAME="${DB_NAME:-alpha}"
DB_HOST="${DB_HOST:-host.docker.internal}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASS="${DB_PASS:-postgres}"

# Set password for psql
export PGPASSWORD="$DB_PASS"

# Execute the drop commands
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<'EOF'
-- Drop the AGE graph if it exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM ag_catalog.ag_graph WHERE name = 'semantic_lineage') THEN
        PERFORM ag_catalog.drop_graph('semantic_lineage', true);
        RAISE NOTICE 'Dropped AGE graph: semantic_lineage';
    END IF;
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'AGE functions not available, skipping graph drop';
    WHEN undefined_table THEN
        RAISE NOTICE 'ag_catalog schema does not exist, skipping graph drop';
END $$;

-- Drop the AGE extension
DROP EXTENSION IF EXISTS age CASCADE;

-- Confirm removal
SELECT 'AGE extension successfully removed' AS status;

-- Show remaining extensions
\dx
EOF

echo ""
echo "✅ AGE extension removal complete!"
echo ""
echo "The database now uses only relational tables:"
echo "  - catalog_node (for nodes/entities)"
echo "  - catalog_edge (for relationships)"
echo "  - semantic.lineage_nodes (for lineage nodes)"
echo "  - semantic.lineage_edges (for lineage relationships)"
