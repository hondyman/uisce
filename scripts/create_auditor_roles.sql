-- ============================================================================
-- AUDITOR ROLES & STARROCKS SECURE AUDIT VIEWS
-- Target: PostgreSQL (alpha DB on 100.84.50.65:5432) & StarRocks FE (9030)
-- ============================================================================

-- 1. PostgreSQL RBAC: Create tenant_auditor and internal_auditor roles
INSERT INTO bp_roles (id, tenant_id, role_key, role_name, description, is_active, is_system_role, created_at, updated_at)
VALUES 
  ('a1b2c3d4-e5f6-7890-abcd-111111111111'::uuid, '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'tenant_auditor', 'Client External Auditor', 'Read-only access strictly isolated to client tenant audit history', true, true, NOW(), NOW()),
  ('a1b2c3d4-e5f6-7890-abcd-222222222222'::uuid, '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'internal_auditor', 'Uisce Internal Auditor', 'System-wide read-only access across all tenant audit ledgers', true, true, NOW(), NOW())
ON CONFLICT (role_key) DO UPDATE SET description = EXCLUDED.description, updated_at = NOW();

-- 2. StarRocks SQL: Create Secure External Audit Trail View with Email Masking
-- Database: iceberg_lakehouse
CREATE DATABASE IF NOT EXISTS iceberg_lakehouse.audit_marts;

USE iceberg_lakehouse.audit_marts;

CREATE OR REPLACE VIEW v_external_audit_trail AS
SELECT 
    timestamp,
    tenant_id,
    tenant_instance_id,
    -- Email Masking: Transforms 'john.doe@bank.com' into 'j***@bank.com'
    CASE 
        WHEN user_id LIKE '%@%' THEN CONCAT(SUBSTRING(user_id, 1, 1), '***@', SUBSTRING(user_id, INSTR(user_id, '@') + 1))
        ELSE user_id 
    END AS masked_user_id,
    action,
    entity_type,
    entity_id,
    role_key,
    chain_of_custody_hash
FROM iceberg_lakehouse.default.fact_user_access_history;
