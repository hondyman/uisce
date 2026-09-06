-- Seed john.b for e2e a11y crawl (idempotent, re-runnable in CI).
-- Three-table provision: app_user.tenant_id (single) + user_tenant (multi-tenant
-- membership) + app_user.role/is_core_admin/permissions (auth model).
--
-- Verification oracle after running: SELECT should return exactly 1 row with
-- role='admin', is_core_admin=true, tenant_id='99e99e99-...'.

-- Step 1: Set john.b's primary tenant (was NULL — causes RLS to deny everything)
UPDATE app_user
SET tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'
WHERE id = '113d0169-4819-42ff-968b-778f72af79e9'
  AND tenant_id IS NULL;

-- Step 2: Add john.b to user_tenant with admin role on Northwind
INSERT INTO user_tenant (user_id, tenant_id, access_role)
VALUES ('113d0169-4819-42ff-968b-778f72af79e9', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'admin')
ON CONFLICT (user_id, tenant_id) DO UPDATE SET access_role = 'admin';

-- Step 3: Set app_user.role for the public.users view read path
UPDATE app_user
SET role = 'admin',
    is_core_admin = true,
    permissions = '["*"]'::jsonb
WHERE id = '113d0169-4819-42ff-968b-778f72af79e9';

-- CI-verify assertion (run as: psql < this_file 2>&1 | grep 'seed OK'; assert 0 on success)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM app_user
        WHERE id = '113d0169-4819-42ff-968b-778f72af79e9'
          AND role = 'admin'
          AND is_core_admin = true
          AND tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'
    ) AND EXISTS (
        SELECT 1 FROM user_tenant
        WHERE user_id = '113d0169-4819-42ff-968b-778f72af79e9'
          AND tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'
          AND access_role = 'admin'
    ) THEN
        RAISE NOTICE 'john.b seed OK';
    ELSE
        RAISE NOTICE 'john.b seed FAILED';
    END IF;
END $$;
