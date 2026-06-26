#!/bin/bash
# ============================================================================
# Phase 3.21 Schema Initialization Script
# ============================================================================
# Run this script to initialize the feature engineering schema in your
# development or staging PostgreSQL database.
#
# Usage:
#   ./init_schema.sh
#
# Prerequisites:
#   - PostgreSQL 13+ running
#   - PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE env vars set
#   - Or modify the psql connection string below
# ============================================================================

set -e
set -o pipefail

# Configuration
DB_HOST="${PGHOST:-localhost}"
DB_PORT="${PGPORT:-5432}"
DB_USER="${PGUSER:-postgres}"
DB_NAME="${PGDATABASE:-semlayer}"
SCHEMA_FILE="./phase_3_21_schema.sql"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Phase 3.21 Feature Engineering Schema Initialization${NC}"
echo "================================================================"
echo "Database: $DB_NAME @ $DB_HOST:$DB_PORT"
echo "Schema file: $SCHEMA_FILE"
echo ""

# Check if schema file exists
if [ ! -f "$SCHEMA_FILE" ]; then
    echo -e "${RED}ERROR: Schema file not found: $SCHEMA_FILE${NC}"
    exit 1
fi

# Connect and run schema
echo -e "${YELLOW}Connecting to PostgreSQL...${NC}"
PGPASSWORD="$PGPASSWORD" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 \
    -f "$SCHEMA_FILE"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Schema initialized successfully${NC}"
    
    # Print summary
    echo ""
    echo -e "${YELLOW}Summary${NC}"
    echo "================================================================"
    PGPASSWORD="$PGPASSWORD" psql \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -t -c "
    SELECT 'Tables Created:' as section, count(*) as count 
    FROM information_schema.tables 
    WHERE table_schema = 'public' AND table_type = 'BASE TABLE' 
        AND table_name LIKE 'feature%'
    UNION ALL
    SELECT 'Views Created:', count(*) 
    FROM information_schema.views 
    WHERE table_schema = 'public' 
        AND table_name LIKE 'feature%' OR table_name LIKE '%_active%' OR table_name LIKE '%_failures%'
    UNION ALL
    SELECT 'Indexes Created:', count(*) 
    FROM pg_indexes 
    WHERE schemaname = 'public' AND indexname LIKE 'idx_%'
    "
    
    echo ""
    echo -e "${GREEN}Ready for Phase 3.21 feature engineering operations${NC}"
else
    echo -e "${RED}ERROR: Schema initialization failed${NC}"
    exit 1
fi
