-- 025_mdm_rating_internal_override_action_types.sql
-- Seed two new mdm.rating_action_type codes for the override flow.
-- Shape B column names: action_cd (not code); direction='OTHER' is in
-- the CHECK list (the override codes' direction is "uncategorized" —
-- the action_cd value carries the fine-grained meaning).
--
-- Run AFTER 023_mdm_rating_type.sql.

\set ON_ERROR_STOP on
BEGIN;

SET LOCAL app.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';
SET LOCAL uisce.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';

INSERT INTO mdm.rating_action_type
    (id, tenant_id, action_cd, name, direction,
     is_credit_event, is_active, custom_attributes)
VALUES
    ('a1a10240-0000-0000-0000-000000000007'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'OVERRIDE_APPLIED', 'Override applied on top of model output.',
     'OTHER', false, true, '{}'::jsonb),
    ('a1a10240-0000-0000-0000-000000000008'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'OVERRIDE_EXPIRED', 'Override window closed; rating reverts to model.',
     'OTHER', false, true, '{}'::jsonb)
ON CONFLICT (tenant_id, action_cd) DO UPDATE SET
    name = EXCLUDED.name,
    direction = EXCLUDED.direction,
    is_credit_event = EXCLUDED.is_credit_event,
    is_active = EXCLUDED.is_active;

COMMIT;
