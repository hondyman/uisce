-- Idempotent migration: set a default tenant_id for existing business_objects rows that have NULL tenant_id
-- This prevents joins from failing to match display names in tenant-scoped queries during development.
-- Run with: psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" -f backend/db/migrations/20251112_set_business_objects_tenant_id.sql

BEGIN;

-- Use a well-known dev tenant id used in local development
DO $$
BEGIN
    -- Only update rows that are missing tenant_id
    UPDATE business_objects
    SET tenant_id = '00000000-0000-0000-0000-000000000000'
    WHERE tenant_id IS NULL;
END$$;

COMMIT;

-- Note: This migration is intentionally simple and idempotent. If you have a
-- dedicated migration tracking system, add this file to that system as needed.
