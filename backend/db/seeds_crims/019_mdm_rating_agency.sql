-- 019_mdm_rating_agency.sql
-- Seed the rating agencies used by Rating, Benchmark, and (later) Security
-- reference data. Idempotent on (tenant_id, agency_cd).
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 019_mdm_rating_agency.sql

\set ON_ERROR_STOP on
BEGIN;

SET LOCAL app.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';
SET LOCAL uisce.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';

-- Stable IDs so other seeds (rating_scale.agency_id, JSON mapping config)
-- can reference them.
INSERT INTO mdm.rating_agency (
    id, tenant_id, agency_cd, name, short_name, agency_type,
    domicile, is_nrsro, is_ecai, is_naic_acceptable,
    is_active, effective_from, status
) VALUES
    ('a1a10000-0000-0000-0000-00000000a001'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'SP', 'S&P Global Ratings', 'S&P', 'NRSRO',
     'US', true, true, true,
     true, '2010-01-01', 'ACTIVE'),
    ('a1a10000-0000-0000-0000-00000000a002'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'MOODYS', 'Moody''s Investors Service', 'Moody''s', 'NRSRO',
     'US', true, true, true,
     true, '2010-01-01', 'ACTIVE'),
    ('a1a10000-0000-0000-0000-00000000a003'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'FITCH', 'Fitch Ratings', 'Fitch', 'NRSRO',
     'US', true, true, true,
     true, '2010-01-01', 'ACTIVE'),
    ('a1a10000-0000-0000-0000-00000000a004'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'DBRS', 'DBRS Morningstar', 'DBRS', 'NRSRO',
     'US', true, true, true,
     true, '2015-01-01', 'ACTIVE'),
    ('a1a10000-0000-0000-0000-00000000000a'::uuid,
     '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
     'INTERNAL', 'Internal Rating Model', 'Internal', 'INTERNAL',
     NULL, false, false, false,
     true, '2010-01-01', 'ACTIVE')
ON CONFLICT (tenant_id, agency_cd) DO UPDATE SET
    name = EXCLUDED.name,
    short_name = EXCLUDED.short_name,
    agency_type = EXCLUDED.agency_type,
    is_nrsro = EXCLUDED.is_nrsro,
    is_ecai = EXCLUDED.is_ecai;

DO $v$
DECLARE n int;
BEGIN
    SELECT count(*) INTO n FROM mdm.rating_agency
     WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid
       AND agency_cd IN ('SP','MOODYS','FITCH','DBRS','INTERNAL');
    IF n <> 5 THEN
        RAISE EXCEPTION 'expected 5 agencies, got %', n;
    END IF;
    RAISE NOTICE '019_mdm_rating_agency: 5 rows present';
END
$v$;

COMMIT;
