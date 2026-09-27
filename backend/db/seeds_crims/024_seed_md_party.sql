-- 024_seed_md_party.sql
-- Seed the parties referenced by the rating slice's CSV. mdm.party is the
-- alpha-shape master (db/migrations/20261028_001_mdm_party_expand.up.sql);
-- it has lei, registration_number, name, and other columns but no
-- party_cd. rated_party_key on staging.rating_incoming is interpreted as
-- an LEI; this file populates the parties the slice CSV references.
--
-- Apply AFTER 0010_staging_rating_incoming + 019..023 rating reference seeds.
-- Idempotent on (tenant_id, id).

\set ON_ERROR_STOP on
BEGIN;

SET LOCAL app.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';
SET LOCAL uisce.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';

INSERT INTO mdm.party (
    id, tenant_id,
    name, lei, status, lifecycle_stage, is_legal_entity, is_golden_record,
    created_at, updated_at
) VALUES
    -- Acme Holdings (rated by SP, Moody's, Internal — exercised in CSV)
    ('c0c0a000-0000-0000-0000-00000000a001'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'Acme Holdings', '529900T8BM49AURSDO55',
     'ACTIVE', 'ACTIVE', true, true, now(), now()),
    -- Global Holdings
    ('c0c0a000-0000-0000-0000-00000000a002'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'Global Holdings', '529900T8BM49AURSDO56',
     'ACTIVE', 'ACTIVE', true, true, now(), now()),
    -- EU Sovereign
    ('c0c0a000-0000-0000-0000-00000000a003'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'Federal Republic of Europe', '529900T8BM49AURSDO57',
     'ACTIVE', 'ACTIVE', true, true, now(), now())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    lei = EXCLUDED.lei,
    updated_at = now();

COMMIT;
