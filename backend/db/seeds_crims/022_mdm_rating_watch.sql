-- 022_mdm_rating_watch.sql
-- Shape B rating watch codes (loaded from 002c_rating_fixup.sql).
-- Idempotent on (tenant_id, watch_cd).
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 022_mdm_rating_watch.sql

\set ON_ERROR_STOP on
BEGIN;

SET LOCAL app.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';
SET LOCAL uisce.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';

INSERT INTO mdm.rating_watch
    (id, tenant_id, watch_cd, name, direction,
     horizon_days, is_active, custom_attributes)
VALUES
    ('a1a10220-0000-0000-0000-000000000000'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'NOT_ON_WATCH', 'Rating not under active review.',
     'EVOLVING', NULL,
     true, '{}'::jsonb),
    ('a1a10220-0000-0000-0000-000000000001'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'WATCH_POSITIVE', 'Possible upgrade under review.',
     'POSITIVE', 90,
     true, '{}'::jsonb),
    ('a1a10220-0000-0000-0000-000000000002'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'WATCH_NEGATIVE', 'Possible downgrade under review.',
     'NEGATIVE', 90,
     true, '{}'::jsonb),
    ('a1a10220-0000-0000-0000-000000000003'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'WATCH_EVOLVING', 'Direction unresolved.',
     'EVOLVING', 90,
     true, '{}'::jsonb)
ON CONFLICT (tenant_id, watch_cd) DO UPDATE SET
    name = EXCLUDED.name,
    direction = EXCLUDED.direction,
    horizon_days = EXCLUDED.horizon_days,
    is_active = EXCLUDED.is_active;

COMMIT;
