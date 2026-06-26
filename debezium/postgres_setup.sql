-- PostgreSQL Setup for Debezium CDC
-- This script configures PostgreSQL for logical replication with Debezium

-- ============================================================================
-- STEP 1: Enable Logical Replication
-- ============================================================================

-- Check current wal_level (must be 'logical')
SHOW wal_level;

-- If wal_level is not 'logical', you need to update postgresql.conf:
-- wal_level = logical
-- max_replication_slots = 4  (or higher)
-- max_wal_senders = 4  (or higher)
-- Then restart PostgreSQL

-- ============================================================================
-- STEP 2: Create Debezium User (if not using postgres superuser)
-- ============================================================================

-- Create dedicated user for Debezium
CREATE USER debezium_user WITH REPLICATION PASSWORD 'debezium_password';

-- Grant necessary permissions
GRANT USAGE ON SCHEMA iam TO debezium_user;
GRANT SELECT ON ALL TABLES IN SCHEMA iam TO debezium_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA iam GRANT SELECT ON TABLES TO debezium_user;

-- Grant replication permissions
ALTER USER debezium_user WITH REPLICATION;

-- ============================================================================
-- STEP 3: Create Publication for IAM Tables
-- ============================================================================

-- Create publication that Debezium will subscribe to
CREATE PUBLICATION iam_security_publication FOR TABLE 
    iam.roles,
    iam.permissions,
    iam.role_permissions,
    iam.user_roles,
    iam.security_events;

-- Verify publication
SELECT * FROM pg_publication WHERE pubname = 'iam_security_publication';

-- View published tables
SELECT * FROM pg_publication_tables WHERE pubname = 'iam_security_publication';

-- ============================================================================
-- STEP 4: Create Replication Slot (Optional - Debezium can create it)
-- ============================================================================

-- Debezium will create this automatically, but you can pre-create it:
-- SELECT pg_create_logical_replication_slot('iam_security_slot', 'pgoutput');

-- View existing replication slots
SELECT * FROM pg_replication_slots;

-- ============================================================================
-- STEP 5: Configure pg_hba.conf for Replication
-- ============================================================================

-- Add this line to pg_hba.conf to allow replication connections:
-- host    replication     debezium_user   0.0.0.0/0               md5

-- Or if using postgres user:
-- host    replication     postgres        0.0.0.0/0               md5

-- Then reload PostgreSQL configuration:
-- SELECT pg_reload_conf();

-- ============================================================================
-- VERIFICATION
-- ============================================================================

-- Check wal_level
SELECT name, setting FROM pg_settings WHERE name = 'wal_level';

-- Check max_replication_slots
SELECT name, setting FROM pg_settings WHERE name = 'max_replication_slots';

-- Check max_wal_senders
SELECT name, setting FROM pg_settings WHERE name = 'max_wal_senders';

-- Check publications
SELECT * FROM pg_publication;

-- Check replication slots
SELECT slot_name, plugin, slot_type, database, active 
FROM pg_replication_slots;

-- Test replication connection (run from debezium server)
-- psql "host=postgres port=5432 dbname=alpha user=debezium_user replication=database"

-- ============================================================================
-- MONITORING
-- ============================================================================

-- Monitor replication lag
SELECT 
    slot_name,
    pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)) AS replication_lag
FROM pg_replication_slots
WHERE slot_name = 'iam_security_slot';

-- Monitor WAL size
SELECT pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), '0/0')) AS wal_size;

-- ============================================================================
-- CLEANUP (if needed)
-- ============================================================================

-- Drop publication
-- DROP PUBLICATION IF EXISTS iam_security_publication;

-- Drop replication slot
-- SELECT pg_drop_replication_slot('iam_security_slot');

-- Revoke permissions
-- REVOKE ALL ON SCHEMA iam FROM debezium_user;
-- DROP USER debezium_user;
