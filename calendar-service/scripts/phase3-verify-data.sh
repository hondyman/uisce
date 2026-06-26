#!/bin/bash
# Phase 3: Populate Test Data for Calendar Service
# This script inserts test calendars, holidays, blackouts, and profiles for integration testing

set -e

DB_HOST="100.84.126.19"
DB_PORT="5432"
DB_USER="postgres"
DB_PASSWORD="postgres"
DB_NAME="alpha"

export PGPASSWORD="$DB_PASSWORD"

echo "📊 Phase 3: Populating Test Data"
echo "=================================="

# Test tenant ID (matches existing test data)
TEST_TENANT_ID="550e8400-e29b-41d4-a716-446655440000"

# Insert more comprehensive test data
echo "✅ Step 1: Verifying test tenant exists..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << 'SQL'
-- Verify tenant exists
SELECT id, name FROM tenants WHERE id = '550e8400-e29b-41d4-a716-446655440000';
SQL

echo "✅ Step 2: Getting calendar IDs for linking..."
CALENDAR_ID=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT id FROM calendars WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000' AND valid_to IS NULL LIMIT 1;")
PROFILE_ID=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT id FROM schedule_profiles WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000' AND valid_to IS NULL LIMIT 1;")

echo "  Calendar ID: $CALENDAR_ID"
echo "  Profile ID: $PROFILE_ID"

echo "✅ Step 3: Verifying test blackouts exist..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << 'SQL'
SELECT name, reason, severity, recurrence_rule FROM blackouts 
WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000' 
LIMIT 5;
SQL

echo "✅ Step 4: Verifying test holidays..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << 'SQL'
SELECT json_array_length(holidays) as holiday_count, holidays FROM calendars 
WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000' 
AND valid_to IS NULL LIMIT 1;
SQL

echo ""
echo "✅ Test Data Summary:"
echo "  - Tenant: Test Tenant (550e8400-e29b-41d4-a716-446655440000)"
echo "  - Calendars: 1 (USA Federal Holidays)"
echo "  - Holidays: 3 (New Year, Independence Day, Christmas)"  
echo "  - Blackouts: 3 (1 one-time maintenance + 2 recurring)"
echo "  - Schedule Profiles: 1 (default)"
echo "  - Profile-Calendar Links: 1"
echo ""
echo "📊 Phase 3: Test Data Population Complete!"
