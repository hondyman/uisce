-- =============================================================================
-- Seed Alpha Wealth users + tenant assignments for Northwind tenant
-- Users: pat.egan (owner), sarah.c, david.v, marcus.b, sophia.l, jonah.p,
--        elena.r (direct employees), olivia.chen (contractor/entitlement),
--        support@uisce.io (Uisce Support/entitlement), alex.reyes (inactive),
--        sam.patel (inactive contractor/entitlement)
-- All passwords: password123  (bcrypt cost 10)
-- Tenant: northwind (99e99e99-99e9-49e9-89e9-99e99e99e999)
-- =============================================================================

DO $$
DECLARE
    -- 11 user UUIDs (explicit so they are stable across runs)
    u_pat    CONSTANT uuid := 'a0000001-0001-0001-0001-000000000001';
    u_sarah  CONSTANT uuid := 'a0000002-0002-0002-0002-000000000002';
    u_david  CONSTANT uuid := 'a0000003-0003-0003-0003-000000000003';
    u_marcus CONSTANT uuid := 'a0000004-0004-0004-0004-000000000004';
    u_sophia CONSTANT uuid := 'a0000005-0005-0005-0005-000000000005';
    u_jonah  CONSTANT uuid := 'a0000006-0006-0006-0006-000000000006';
    u_elena  CONSTANT uuid := 'a0000007-0007-0007-0007-000000000007';
    u_olivia CONSTANT uuid := 'a0000008-0008-0008-0008-000000000008';
    u_support CONSTANT uuid := 'a0000009-0009-0009-0009-000000000009';
    u_alex   CONSTANT uuid := 'a000000a-000a-000a-000a-00000000000a';
    u_sam    CONSTANT uuid := 'a000000b-000b-000b-000b-00000000000b';

    -- bcrypt hash of 'password123' (cost 10)
    pw_hash CONSTANT text := '$2b$10$.YhljjcLXk439fwbnFXMw.opnYzgqOZEBWtptNgMWCI.KM2P0LxZm';

    -- Northwind tenant ID
    northwind_id uuid := '99e99e99-99e9-49e9-89e9-99e99e99e999';
BEGIN

    -- -------------------------------------------------------------------------
    -- 1. app_user rows (idempotent by email)
    -- -------------------------------------------------------------------------

    INSERT INTO public.app_user (id, email, name, first_name, last_name, role, organization,
                                 password_hash, is_active, status, tenant_id, created_at, updated_at)
    VALUES
      (u_pat, 'pat.egan@alpha-wealth.com', 'Pat Egan', 'Pat', 'Egan',
       'admin', 'alpha-wealth', pw_hash, true, 'active', northwind_id, NOW(), NOW())
    ON CONFLICT (email) DO UPDATE SET
      name        = EXCLUDED.name,
      first_name  = EXCLUDED.first_name,
      last_name   = EXCLUDED.last_name,
      role        = EXCLUDED.role,
      organization= EXCLUDED.organization,
      is_active   = EXCLUDED.is_active,
      status      = EXCLUDED.status,
      tenant_id   = EXCLUDED.tenant_id,
      updated_at  = NOW();

    INSERT INTO public.app_user (id, email, name, first_name, last_name, role, organization,
                                 password_hash, is_active, status, tenant_id, created_at, updated_at)
    VALUES
      (u_sarah, 'sarah.c@alpha-wealth.com', 'Sarah Connor', 'Sarah', 'Connor',
       'portfolio_manager', 'alpha-wealth', pw_hash, true, 'active', northwind_id, NOW(), NOW())
    ON CONFLICT (email) DO UPDATE SET
      name=EXCLUDED.name, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name,
      role=EXCLUDED.role, organization=EXCLUDED.organization, is_active=EXCLUDED.is_active,
      status=EXCLUDED.status, tenant_id=EXCLUDED.tenant_id, updated_at=NOW();

    INSERT INTO public.app_user (id, email, name, first_name, last_name, role, organization,
                                 password_hash, is_active, status, tenant_id, created_at, updated_at)
    VALUES
      (u_david, 'david.v@alpha-wealth.com', 'David Vance', 'David', 'Vance',
       'portfolio_manager', 'alpha-wealth', pw_hash, true, 'active', northwind_id, NOW(), NOW())
    ON CONFLICT (email) DO UPDATE SET
      name=EXCLUDED.name, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name,
      role=EXCLUDED.role, organization=EXCLUDED.organization, is_active=EXCLUDED.is_active,
      status=EXCLUDED.status, tenant_id=EXCLUDED.tenant_id, updated_at=NOW();

    INSERT INTO public.app_user (id, email, name, first_name, last_name, role, organization,
                                 password_hash, is_active, status, tenant_id, created_at, updated_at)
    VALUES
      (u_marcus, 'marcus.b@alpha-wealth.com', 'Marcus Bell', 'Marcus', 'Bell',
       'client_advisor', 'alpha-wealth', pw_hash, true, 'active', northwind_id, NOW(), NOW())
    ON CONFLICT (email) DO UPDATE SET
      name=EXCLUDED.name, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name,
      role=EXCLUDED.role, organization=EXCLUDED.organization, is_active=EXCLUDED.is_active,
      status=EXCLUDED.status, tenant_id=EXCLUDED.tenant_id, updated_at=NOW();

    INSERT INTO public.app_user (id, email, name, first_name, last_name, role, organization,
                                 password_hash, is_active, status, tenant_id, created_at, updated_at)
    VALUES
      (u_sophia, 'sophia.l@alpha-wealth.com', 'Sophia Lin', 'Sophia', 'Lin',
       'compliance_officer', 'alpha-wealth', pw_hash, true, 'active', northwind_id, NOW(), NOW())
    ON CONFLICT (email) DO UPDATE SET
      name=EXCLUDED.name, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name,
      role=EXCLUDED.role, organization=EXCLUDED.organization, is_active=EXCLUDED.is_active,
      status=EXCLUDED.status, tenant_id=EXCLUDED.tenant_id, updated_at=NOW();

    INSERT INTO public.app_user (id, email, name, first_name, last_name, role, organization,
                                 password_hash, is_active, status, tenant_id, created_at, updated_at)
    VALUES
      (u_jonah, 'jonah.pierce@alpha-wealth.com', 'Jonathan Pierce', 'Jonathan', 'Pierce',
       'trade_execution', 'alpha-wealth', pw_hash, true, 'active', northwind_id, NOW(), NOW())
    ON CONFLICT (email) DO UPDATE SET
      name=EXCLUDED.name, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name,
      role=EXCLUDED.role, organization=EXCLUDED.organization, is_active=EXCLUDED.is_active,
      status=EXCLUDED.status, tenant_id=EXCLUDED.tenant_id, updated_at=NOW();

    INSERT INTO public.app_user (id, email, name, first_name, last_name, role, organization,
                                 password_hash, is_active, status, tenant_id, created_at, updated_at)
    VALUES
      (u_elena, 'elena.rostova@alpha-wealth.com', 'Elena Rostova', 'Elena', 'Rostova',
       'portfolio_manager', 'alpha-wealth', pw_hash, true, 'active', northwind_id, NOW(), NOW())
    ON CONFLICT (email) DO UPDATE SET
      name=EXCLUDED.name, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name,
      role=EXCLUDED.role, organization=EXCLUDED.organization, is_active=EXCLUDED.is_active,
      status=EXCLUDED.status, tenant_id=EXCLUDED.tenant_id, updated_at=NOW();

    -- Contractor via entitlement
    INSERT INTO public.app_user (id, email, name, first_name, last_name, role, organization,
                                 password_hash, is_active, status, tenant_id, created_at, updated_at)
    VALUES
      (u_olivia, 'olivia.chen@contractor.uisce.io', 'Olivia Chen', 'Olivia', 'Chen',
       'viewer', 'contractor.uisce.io', pw_hash, true, 'active', NULL, NOW(), NOW())
    ON CONFLICT (email) DO UPDATE SET
      name=EXCLUDED.name, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name,
      role=EXCLUDED.role, organization=EXCLUDED.organization, is_active=EXCLUDED.is_active,
      status=EXCLUDED.status, updated_at=NOW();

    -- Uisce Support entitlement
    INSERT INTO public.app_user (id, email, name, first_name, last_name, role, organization,
                                 password_hash, is_active, status, tenant_id, created_at, updated_at)
    VALUES
      (u_support, 'support@uisce.io', 'Uisce Support Bot', 'Uisce', 'Support',
       'helpdesk', 'uisce', pw_hash, true, 'active', NULL, NOW(), NOW())
    ON CONFLICT (email) DO UPDATE SET
      name=EXCLUDED.name, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name,
      role=EXCLUDED.role, organization=EXCLUDED.organization, is_active=EXCLUDED.is_active,
      status=EXCLUDED.status, updated_at=NOW();

    -- Inactive former employee
    INSERT INTO public.app_user (id, email, name, first_name, last_name, role, organization,
                                 password_hash, is_active, status, tenant_id, created_at, updated_at)
    VALUES
      (u_alex, 'alex.reyes@alpha-wealth.com', 'Alex Reyes', 'Alex', 'Reyes',
       'portfolio_manager', 'alpha-wealth', pw_hash, false, 'inactive', northwind_id, NOW(), NOW())
    ON CONFLICT (email) DO UPDATE SET
      name=EXCLUDED.name, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name,
      role=EXCLUDED.role, organization=EXCLUDED.organization, is_active=EXCLUDED.is_active,
      status=EXCLUDED.status, tenant_id=EXCLUDED.tenant_id, updated_at=NOW();

    -- Inactive contractor
    INSERT INTO public.app_user (id, email, name, first_name, last_name, role, organization,
                                 password_hash, is_active, status, tenant_id, created_at, updated_at)
    VALUES
      (u_sam, 'sam.patel@contractor.uisce.io', 'Sam Patel', 'Sam', 'Patel',
       'viewer', 'contractor.uisce.io', pw_hash, false, 'inactive', NULL, NOW(), NOW())
    ON CONFLICT (email) DO UPDATE SET
      name=EXCLUDED.name, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name,
      role=EXCLUDED.role, organization=EXCLUDED.organization, is_active=EXCLUDED.is_active,
      status=EXCLUDED.status, updated_at=NOW();

    RAISE NOTICE 'Seed users inserted/updated successfully.';

END $$;

-- -------------------------------------------------------------------------
-- 2. user_tenant rows (direct employees + pat as owner)
-- -------------------------------------------------------------------------

INSERT INTO public.user_tenant (user_id, tenant_id, access_role, created_at, updated_at)
VALUES
  ('a0000001-0001-0001-0001-000000000001', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'owner', NOW(), NOW()),
  ('a0000002-0002-0002-0002-000000000002', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'viewer', NOW(), NOW()),
  ('a0000003-0003-0003-0003-000000000003', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'viewer', NOW(), NOW()),
  ('a0000004-0004-0004-0004-000000000004', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'viewer', NOW(), NOW()),
  ('a0000005-0005-0005-0005-000000000005', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'viewer', NOW(), NOW()),
  ('a0000006-0006-0006-0006-000000000006', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'viewer', NOW(), NOW()),
  ('a0000007-0007-0007-0007-000000000007', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'viewer', NOW(), NOW()),
  ('a000000a-000a-000a-000a-00000000000a', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'viewer', NOW(), NOW())
ON CONFLICT (user_id, tenant_id) DO UPDATE SET
  access_role = EXCLUDED.access_role,
  updated_at  = NOW();

-- -------------------------------------------------------------------------
-- 3. user_tenant_access rows (entitlement-based access: contractor / uisce)
-- -------------------------------------------------------------------------

INSERT INTO public.user_tenant_access (id, user_id, tenant_id, role, created_at, created_by, updated_at)
VALUES
  ('b0000001-0001-0001-0001-000000000001', 'a0000008-0008-0008-0008-000000000008',
   '99e99e99-99e9-49e9-89e9-99e99e99e999', 'contractor', NOW(), 'system', NOW()),
  ('b0000002-0002-0002-0002-000000000002', 'a0000009-0009-0009-0009-000000000009',
   '99e99e99-99e9-49e9-89e9-99e99e99e999', 'support_entitlement', NOW(), 'system', NOW()),
  ('b0000003-0003-0003-0003-000000000003', 'a000000b-000b-000b-000b-00000000000b',
   '99e99e99-99e9-49e9-89e9-99e99e99e999', 'contractor', NOW(), 'system', NOW())
ON CONFLICT (user_id, tenant_id) DO UPDATE SET
  role       = EXCLUDED.role,
  updated_at = NOW();

-- -------------------------------------------------------------------------
-- 4. Add last_login_time for existing test user so Pat Egan shows a real value
-- -------------------------------------------------------------------------

UPDATE public.app_user
SET last_login_time = NOW() - INTERVAL '1 day'
WHERE id = 'a0000001-0001-0001-0001-000000000001';
