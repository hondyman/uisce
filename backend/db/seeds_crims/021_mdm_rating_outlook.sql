-- 021_mdm_rating_outlook.sql
-- Shape B rating outlook codes (loaded from 002c_rating_fixup.sql).
-- Idempotent on (tenant_id, outlook_cd).
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 021_mdm_rating_outlook.sql

\set ON_ERROR_STOP on
BEGIN;

SET LOCAL app.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';
SET LOCAL uisce.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';

INSERT INTO mdm.rating_outlook
    (id, tenant_id, outlook_cd, name, direction,
     horizon_months, is_active, custom_attributes)
VALUES
    ('a1a10210-0000-0000-0000-000000000001'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'POSITIVE', 'Upgrade likely in 6-24 months.', 'POSITIVE', 18,
     true, '{}'::jsonb),
    ('a1a10210-0000-0000-0000-000000000002'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'STABLE', 'No change expected.', 'STABLE', 18,
     true, '{}'::jsonb),
    ('a1a10210-0000-0000-0000-000000000003'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'NEGATIVE', 'Downgrade likely in 6-24 months.', 'NEGATIVE', 18,
     true, '{}'::jsonb),
    ('a1a10210-0000-0000-0000-000000000004'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'DEVELOPING', 'Direction unresolved.', 'DEVELOPING', 12,
     true, '{}'::jsonb),
    ('a1a10210-0000-0000-0000-000000000005'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'NEUTRAL', 'No directional signal.', 'NEUTRAL', 18,
     true, '{}'::jsonb)
ON CONFLICT (tenant_id, outlook_cd) DO UPDATE SET
    name = EXCLUDED.name,
    direction = EXCLUDED.direction,
    horizon_months = EXCLUDED.horizon_months,
    is_active = EXCLUDED.is_active;

COMMIT;
