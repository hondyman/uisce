#!/bin/bash

# Phase 2 Test Data Setup & Integration Testing
# Sets up test calendars, holidays, blackouts, and profiles
# Then runs comprehensive integration tests

set -e

echo "🧪 Phase 2: Holiday/Blackout Resolution Testing"
echo "=================================================="
echo ""

# Configuration
TENANT_ID="550e8400-e29b-41d4-a716-446655440000"
CALENDAR_1_ID="660e8400-e29b-41d4-a716-446655440001"
CALENDAR_2_ID="660e8400-e29b-41d4-a716-446655440002"
PROFILE_ID="770e8400-e29b-41d4-a716-446655440000"
API_BASE="${API_BASE:-http://localhost:8081}"

# Get database connection details
: ${PGHOST:=100.84.126.19}
: ${PGPORT:=5432}
: ${PGUSER:=postgres}
: ${PGPASSWORD:=postgres}
: ${PGDATABASE:=alpha}

export PGPASSWORD

echo "📋 Creating Test Data"
echo "---"

# Create Calendar 1: US Federal Holidays
echo "Creating Calendar 1 (US Federal Holidays)..."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" << EOF
INSERT INTO calendars (id, tenant_id, name, description, region, holidays, created_at, updated_at, valid_from, valid_to, active)
VALUES (
    '${CALENDAR_1_ID}',
    '${TENANT_ID}',
    'US Federal Holidays',
    'US Federal holidays calendar for 2026',
    'US',
    '[
        {"date": "2026-01-01", "name": "New Year'\''s Day", "type": "public", "severity": "HIGH", "all_day": true},
        {"date": "2026-02-16", "name": "Presidents Day", "type": "public", "severity": "MEDIUM", "all_day": true},
        {"date": "2026-07-04", "name": "Independence Day", "type": "public", "severity": "HIGH", "all_day": true},
        {"date": "2026-12-25", "name": "Christmas Day", "type": "public", "severity": "HIGH", "all_day": true}
    ]'::jsonb,
    NOW(),
    NOW(),
    NOW(),
    NULL,
    true
)
ON CONFLICT (id) DO UPDATE SET 
    holidays = EXCLUDED.holidays,
    updated_at = NOW();
EOF

# Create Calendar 2: Company Maintenance Windows
echo "Creating Calendar 2 (Company Maintenance Windows)..."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" << EOF
INSERT INTO calendars (id, tenant_id, name, description, region, holidays, created_at, updated_at, valid_from, valid_to, active)
VALUES (
    '${CALENDAR_2_ID}',
    '${TENANT_ID}',
    'Company Maintenance',
    'Company maintenance and blackout windows',
    'US',
    '[
        {"date": "2026-02-20", "name": "Quarterly Review Day", "type": "observance", "severity": "MEDIUM", "all_day": true}
    ]'::jsonb,
    NOW(),
    NOW(),
    NOW(),
    NULL,
    true
)
ON CONFLICT (id) DO UPDATE SET 
    holidays = EXCLUDED.holidays,
    updated_at = NOW();
EOF

# Create recurring blackouts (Monthly on first Friday, and Weekly on Monday evening)
echo "Creating Blackout Records..."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" << EOF

-- Monthly First Friday blackout (2-hour maintenance window)
INSERT INTO blackouts (id, tenant_id, calendar_id, name, description, start_time, end_time, is_recurring, recurrence_rule, recurrence_end, reason, severity, valid_from, valid_to, created_at, updated_at, active)
VALUES (
    gen_random_uuid(),
    '${TENANT_ID}',
    '${CALENDAR_2_ID}',
    'Monthly Maintenance - First Friday',
    'Monthly maintenance window on first Friday 2AM-4AM UTC',
    make_timestamp(2026, 2, 6, 2, 0, 0),  -- First Friday of Feb 2026
    make_timestamp(2026, 2, 6, 4, 0, 0),
    true,
    'FREQ=MONTHLY;BYMONTHDAY=1;BYDAY=FR;UNTIL=20261231T235959Z',
    make_timestamp(2026, 12, 31, 23, 59, 59),
    'Monthly system maintenance',
    'HIGH',
    NOW(),
    NULL,
    NOW(),
    NOW(),
    true
)
ON CONFLICT DO NOTHING;

-- Weekly Monday evening blackout (6PM-10PM UTC)
INSERT INTO blackouts (id, tenant_id, calendar_id, name, description, start_time, end_time, is_recurring, recurrence_rule, recurrence_end, reason, severity, valid_from, valid_to, created_at, updated_at, active)
VALUES (
    gen_random_uuid(),
    '${TENANT_ID}',
    '${CALENDAR_2_ID}',
    'Weekly Backup - Monday Evening',
    'Weekly backup window every Monday 6PM-10PM UTC',
    make_timestamp(2026, 2, 2, 18, 0, 0),  -- First Monday of Feb
    make_timestamp(2026, 2, 2, 22, 0, 0),
    true,
    'FREQ=WEEKLY;BYDAY=MO;UNTIL=20261231T235959Z',
    make_timestamp(2026, 12, 31, 23, 59, 59),
    'Weekly database backup',
    'MEDIUM',
    NOW(),
    NULL,
    NOW(),
    NOW(),
    true
)
ON CONFLICT DO NOTHING;

-- One-time emergency blackout
INSERT INTO blackouts (id, tenant_id, calendar_id, name, description, start_time, end_time, is_recurring, recurrence_rule, reason, severity, valid_from, valid_to, created_at, updated_at, active)
VALUES (
    gen_random_uuid(),
    '${TENANT_ID}',
    '${CALENDAR_2_ID}',
    'Emergency Maintenance',
    'Unplanned emergency maintenance',
    make_timestamp(2026, 3, 15, 8, 0, 0),
    make_timestamp(2026, 3, 15, 12, 0, 0),
    false,
    '',
    'Emergency system patch',
    'CRITICAL',
    NOW(),
    NULL,
    NOW(),
    NOW(),
    true
)
ON CONFLICT DO NOTHING;

EOF

# Create Schedule Profile linking both calendars
echo "Creating Schedule Profile..."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" << EOF
INSERT INTO schedule_profiles (id, tenant_id, profile_name, description, timezone, region, conflict_resolution, valid_from, valid_to, active, created_at, updated_at)
VALUES (
    '${PROFILE_ID}',
    '${TENANT_ID}',
    'default',
    'Default schedule profile combining US holidays and maintenance windows',
    'UTC',
    'US',
    'UNION',  -- Include holidays/blackouts from ALL calendars
    NOW(),
    NULL,
    true,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE SET
    updated_at = NOW();
EOF

# Link calendars to profile
echo "Linking calendars to profile..."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" << EOF
DELETE FROM profile_calendars WHERE profile_id = '${PROFILE_ID}';

INSERT INTO profile_calendars (id, profile_id, calendar_id, weight, priority, valid_from, valid_to, created_at, updated_at)
VALUES 
    (gen_random_uuid(), '${PROFILE_ID}', '${CALENDAR_1_ID}', 1, 1, NOW(), NULL, NOW(), NOW()),
    (gen_random_uuid(), '${PROFILE_ID}', '${CALENDAR_2_ID}', 2, 2, NOW(), NULL, NOW(), NOW());
EOF

echo ""
echo "✅ Test data created successfully"
echo ""
echo "📊 Test Data Summary:"
echo "  Tenant ID:           ${TENANT_ID}"
echo "  Profile ID:          ${PROFILE_ID}"
echo "  Calendar 1 (Federal Holidays): ${CALENDAR_1_ID}"
echo "  Calendar 2 (Maintenance):      ${CALENDAR_2_ID}"
echo "  Holidays created:    4 (New Year, Presidents Day, Independence Day, Christmas)"
echo "  Blackouts created:   3 (2 recurring + 1 one-time)"
echo ""
echo "📝 Created Blackout Patterns:"
echo "  1. Monthly First Friday 2AM-4AM UTC (high severity)"
echo "  2. Weekly Monday 6PM-10PM UTC (medium severity)"
echo "  3. One-time March 15 8AM-12PM UTC (critical - emergency)"
echo ""
echo "🔄 Expected Recurring Expansions (Feb-Dec 2026):"
echo "  First Friday: ~11 occurrences per year"
echo "  Every Monday: ~52 occurrences per year"
echo ""
