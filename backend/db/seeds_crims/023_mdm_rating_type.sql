-- 023_mdm_rating_type.sql
-- Shape B rating type + rating action type seeds (loaded from
-- 002c_rating_fixup.sql).
--
-- mdm.rating_type: column code -> type_cd; CHECK on applies_to now
-- includes 'TRANCHES' (not 'TRANCHE') per Shape B.
--
-- mdm.rating_action_type: column code -> action_cd; direction CHECK
-- is the new list (UPGRADE, DOWNGRADE, AFFIRMATION, INITIAL, WITHDRAWN,
-- PLACED_ON_WATCH, REMOVED_FROM_WATCH, DEFAULT, CURE, OTHER). AFFIRM is
-- not in the list — use AFFIRMATION.
--
-- Override codes (OVERRIDE_APPLIED, OVERRIDE_EXPIRED) are seeded in
-- 025_mdm_rating_internal_override_action_types.sql with direction='OTHER'.

\set ON_ERROR_STOP on
BEGIN;

SET LOCAL app.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';
SET LOCAL uisce.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';

INSERT INTO mdm.rating_type
    (id, tenant_id, type_cd, name, description, applies_to, rating_basis,
     is_active, custom_attributes)
VALUES
    ('a1a10230-0000-0000-0000-000000000001'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'ISSUER_LT', 'Issuer long-term', 'Long-term issuer credit rating.',
     'ISSUER', NULL,
     true, '{}'::jsonb),
    ('a1a10230-0000-0000-0000-000000000002'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'ISSUER_ST', 'Issuer short-term', 'Short-term issuer credit rating.',
     'ISSUER', NULL,
     true, '{}'::jsonb),
    ('a1a10230-0000-0000-0000-000000000003'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'ISSUE_LT', 'Issue long-term', 'Long-term debt issue rating.',
     'ISSUE', NULL,
     true, '{}'::jsonb),
    ('a1a10230-0000-0000-0000-000000000004'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'SOVEREIGN_LT', 'Sovereign long-term', 'Long-term sovereign credit rating.',
     'SOVEREIGN', NULL,
     true, '{}'::jsonb),
    ('a1a10230-0000-0000-0000-000000000005'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'COUNTERPARTY', 'Counterparty', 'Counterparty credit rating.',
     'COUNTERPARTY', NULL,
     true, '{}'::jsonb),
    ('a1a10230-0000-0000-0000-000000000006'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'FUND', 'Fund', 'Fund-level credit/quality rating.',
     'FUND', NULL,
     true, '{}'::jsonb),
    ('a1a10230-0000-0000-0000-000000000fff'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'INTERNAL_RATING', 'Internal rating', 'House internal rating model output.',
     'ISSUER', 'INTERNAL_MODEL',
     true, '{}'::jsonb)
ON CONFLICT (tenant_id, type_cd) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    applies_to = EXCLUDED.applies_to,
    rating_basis = EXCLUDED.rating_basis,
    is_active = EXCLUDED.is_active;

-- ── rating_action_type (FK target of mdm.rating_action.action_type_id) ──
INSERT INTO mdm.rating_action_type
    (id, tenant_id, action_cd, name, direction,
     is_credit_event, is_active, custom_attributes)
VALUES
    ('a1a10240-0000-0000-0000-000000000001'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'INITIAL', 'First rating assigned to a rated entity.',
     'INITIAL', false, true, '{}'::jsonb),
    ('a1a10240-0000-0000-0000-000000000002'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'UPGRADE', 'Rating value moved to a higher rank.',
     'UPGRADE', false, true, '{}'::jsonb),
    ('a1a10240-0000-0000-0000-000000000003'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'DOWNGRADE', 'Rating value moved to a lower rank.',
     'DOWNGRADE', false, true, '{}'::jsonb),
    ('a1a10240-0000-0000-0000-000000000004'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'AFFIRMATION', 'Rating value unchanged but reviewed.',
     'AFFIRMATION', false, true, '{}'::jsonb),
    ('a1a10240-0000-0000-0000-000000000005'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'WITHDRAW', 'Rating withdrawn by the agency.',
     'WITHDRAWN', false, true, '{}'::jsonb),
    ('a1a10240-0000-0000-0000-000000000006'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'DEFAULT', 'Issuer defaulted on rated obligations.',
     'DEFAULT', true, true, '{}'::jsonb)
ON CONFLICT (tenant_id, action_cd) DO UPDATE SET
    name = EXCLUDED.name,
    direction = EXCLUDED.direction,
    is_credit_event = EXCLUDED.is_credit_event,
    is_active = EXCLUDED.is_active;

COMMIT;
