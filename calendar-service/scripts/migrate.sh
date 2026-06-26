#!/bin/bash
set -e

echo "🚀 Starting Epic 31 Database Migration..."

# Load environment
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Database connection defaults
DB_HOST=${POSTGRES_HOST:-localhost}
DB_PORT=${POSTGRES_PORT:-5432}
DB_USER=${POSTGRES_USER:-postgres}
DB_NAME=${POSTGRES_DB:-calendar_db}
DB_PASS=${POSTGRES_PASSWORD:-postgres}

export PGPASSWORD=$DB_PASS

echo "📦 Connecting to PostgreSQL at ${DB_HOST}:${DB_PORT}/${DB_NAME}..."

# Create database if it doesn't exist
echo "🗄️  Creating database if not exists..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -tc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1 || \
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -c "CREATE DATABASE $DB_NAME"

# Run schema migration
echo "📝 Applying schema migrations..."
if [ -f ./docs/schema.sql ]; then
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f ./docs/schema.sql
else
    echo "⚠️  Schema file not found at ./docs/schema.sql"
    exit 1
fi

# Create partitions for audit_log (if not exists)
echo "📊 Creating audit_log partitions..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<'EOF'
-- Q1 2026
CREATE TABLE IF NOT EXISTS audit_log_2026_q1 PARTITION OF audit_log
    FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');

-- Q2 2026
CREATE TABLE IF NOT EXISTS audit_log_2026_q2 PARTITION OF audit_log
    FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');

-- Q3 2026
CREATE TABLE IF NOT EXISTS audit_log_2026_q3 PARTITION OF audit_log
    FOR VALUES FROM ('2026-07-01') TO ('2026-10-01');

-- Q4 2026
CREATE TABLE IF NOT EXISTS audit_log_2026_q4 PARTITION OF audit_log
    FOR VALUES FROM ('2026-10-01') TO ('2027-01-01');
EOF

# Create partitions for calendar_metrics
echo "📊 Creating calendar_metrics partitions..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<'EOF'
-- January 2026
CREATE TABLE IF NOT EXISTS calendar_metrics_2026_01 PARTITION OF calendar_metrics
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

-- February 2026
CREATE TABLE IF NOT EXISTS calendar_metrics_2026_02 PARTITION OF calendar_metrics
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

-- December 2026 (and beyond)
CREATE TABLE IF NOT EXISTS calendar_metrics_2026_12 PARTITION OF calendar_metrics
    FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');
EOF

# Seed test data
echo "🌱 Seeding test data..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<'EOF'
-- Test tenant
INSERT INTO tenants (id, name, allowed_regions, data_residency_policy)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000', 
    'Test Tenant', 
    ARRAY['us-east-1', 'eu-west-1', 'ap-southeast-1'], 
    'strict'
)
ON CONFLICT (id) DO NOTHING;

-- USA Federal Holidays Calendar
INSERT INTO calendars (tenant_id, name, timezone, holidays, valid_from)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'USA Federal Holidays',
    'America/New_York',
    jsonb_build_array(
        jsonb_build_object('date', '2026-01-01', 'name', 'New Year', 'severity', 'HIGH'),
        jsonb_build_object('date', '2026-07-04', 'name', 'Independence Day', 'severity', 'HIGH'),
        jsonb_build_object('date', '2026-12-25', 'name', 'Christmas', 'severity', 'HIGH')
    ),
    NOW()
)
ON CONFLICT DO NOTHING;

-- EU Public Holidays Calendar
INSERT INTO calendars (tenant_id, name, timezone, holidays, valid_from)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'EU Public Holidays',
    'Europe/London',
    jsonb_build_array(
        jsonb_build_object('date', '2026-01-01', 'name', 'New Year', 'severity', 'HIGH'),
        jsonb_build_object('date', '2026-12-25', 'name', 'Christmas', 'severity', 'HIGH'),
        jsonb_build_object('date', '2026-12-26', 'name', 'Boxing Day', 'severity', 'MEDIUM')
    ),
    NOW()
)
ON CONFLICT DO NOTHING;

-- Default schedule profile
INSERT INTO schedule_profiles (tenant_id, name, timezone, conflict_resolution, valid_from)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'default',
    'UTC',
    'union',
    NOW()
)
ON CONFLICT DO NOTHING;

-- Link calendars to profile
INSERT INTO profile_calendars (profile_id, calendar_id, priority_weight, valid_from)
SELECT sp.id, c.id, 5, NOW()
FROM schedule_profiles sp, calendars c
WHERE sp.tenant_id = '550e8400-e29b-41d4-a716-446655440000'
  AND c.tenant_id = '550e8400-e29b-41d4-a716-446655440000'
  AND sp.name = 'default'
ON CONFLICT DO NOTHING;

-- Test blackout period
INSERT INTO blackouts (profile_id, start_time, end_time, reason, valid_from)
SELECT sp.id, NOW() + INTERVAL '2 days', NOW() + INTERVAL '3 days', 'Maintenance Window', NOW()
FROM schedule_profiles sp
WHERE sp.tenant_id = '550e8400-e29b-41d4-a716-446655440000'
  AND sp.name = 'default'
ON CONFLICT DO NOTHING;
EOF

# Verify indexes
echo "🔍 Verifying indexes..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT indexname FROM pg_indexes WHERE schemaname = 'public' LIMIT 10;"

# Verify tables
echo "📋 Verifying tables..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\dt public.*" | head -20

# Verify RLS policies
echo "🔐 Verifying RLS policies..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT tablename, rowsecurity FROM pg_tables WHERE schemaname = 'public' AND rowsecurity ORDER BY tablename;"

echo ""
echo "✅ Migration complete!"
echo ""
echo "Next steps:"
echo "  1. Start services: make dev"
echo "  2. Test health: curl http://localhost:8081/health"
echo "  3. List calendars: curl http://localhost:8081/api/v1/calendars -H 'X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000'"
echo ""

unset PGPASSWORD
