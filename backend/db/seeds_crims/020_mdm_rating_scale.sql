-- 020_mdm_rating_scale.sql
-- Shape B rating scale ladders. Loaded from db/manual_fixes/002c_rating_fixup.sql.
-- One row per (tenant, scale_cd, value) tuple. UNIQUE constraint on that
-- tuple enforces shape B.
--
-- Single source of truth for rank: rank_no (higher = better). The loader
-- derives mdm.rating.rating_rank from this column at write time.
--
-- Why the FULL ladder (not just the values the CSV uses): missing values
-- cause runtime OVERRIDE_VALUE_NOT_ON_SCALE rejections with no actionable
-- error. Seeding ~80 rows costs nothing and closes that class of bug.
--
-- Why ranks are contiguous: any future query joining mdm.rating against
-- mdm.rating_scale expects 1:1 mapping. Gaps in rank_no silently poison
-- survivorship orderings.
--
-- Apply AFTER 019_mdm_rating_agency.sql (for FK agency_id resolution).
-- Idempotent on (tenant_id, scale_cd, value).

\set ON_ERROR_STOP on
BEGIN;

SET LOCAL app.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';
SET LOCAL uisce.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';

-- ── SP_LT: 22 values, ranks 22..1 ─────────────────────────────────────
-- S&P long-term issuer rating scale. AAA = highest, D = default.
INSERT INTO mdm.rating_scale
    (tenant_id, scale_cd, value, rank_no, agency_cd,
     display_order, effective_from, is_active, custom_attributes)
VALUES
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'AAA',  22, 'SP',  22, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'AA+',  21, 'SP',  21, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'AA',   20, 'SP',  20, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'AA-',  19, 'SP',  19, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'A+',   18, 'SP',  18, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'A',    17, 'SP',  17, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'A-',   16, 'SP',  16, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'BBB+', 15, 'SP',  15, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'BBB',  14, 'SP',  14, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'BBB-', 13, 'SP',  13, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'BB+',  12, 'SP',  12, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'BB',   11, 'SP',  11, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'BB-',  10, 'SP',  10, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'B+',   9,  'SP',  9,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'B',    8,  'SP',  8,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'B-',   7,  'SP',  7,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'CCC+', 6,  'SP',  6,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'CCC',  5,  'SP',  5,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'CCC-', 4,  'SP',  4,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'CC',   3,  'SP',  3,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'C',    2,  'SP',  2,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'SP_LT', 'D',    1,  'SP',  1,  CURRENT_DATE, true, '{}'::jsonb)
ON CONFLICT (tenant_id, scale_cd, value) DO UPDATE SET
    rank_no = EXCLUDED.rank_no,
    is_active = EXCLUDED.is_active;

-- ── MOODYS_LT: 21 values, ranks 22..2 ─────────────────────────────────
-- Moody's long-term scale. No rank 1 — Moody's uses WD (withdrawn)
-- rather than D (default) for long-term scale; rank 1 is reserved for
-- future default-equivalent values.
INSERT INTO mdm.rating_scale
    (tenant_id, scale_cd, value, rank_no, agency_cd,
     display_order, effective_from, is_active, custom_attributes)
VALUES
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Aaa',  22, 'MOODYS', 22, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Aa1',  21, 'MOODYS', 21, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Aa2',  20, 'MOODYS', 20, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Aa3',  19, 'MOODYS', 19, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'A1',   18, 'MOODYS', 18, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'A2',   17, 'MOODYS', 17, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'A3',   16, 'MOODYS', 16, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Baa1', 15, 'MOODYS', 15, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Baa2', 14, 'MOODYS', 14, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Baa3', 13, 'MOODYS', 13, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Ba1',  12, 'MOODYS', 12, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Ba2',  11, 'MOODYS', 11, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Ba3',  10, 'MOODYS', 10, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'B1',   9,  'MOODYS', 9,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'B2',   8,  'MOODYS', 8,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'B3',   7,  'MOODYS', 7,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Caa1', 6,  'MOODYS', 6,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Caa2', 5,  'MOODYS', 5,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Caa3', 4,  'MOODYS', 4,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'Ca',   3,  'MOODYS', 3,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'MOODYS_LT', 'C',    2,  'MOODYS', 2,  CURRENT_DATE, true, '{}'::jsonb)
ON CONFLICT (tenant_id, scale_cd, value) DO UPDATE SET
    rank_no = EXCLUDED.rank_no,
    is_active = EXCLUDED.is_active;

-- ── FITCH_LT: 20 values, ranks 20..1 ──────────────────────────────────
-- Fitch long-term scale. Fitch collapses +/- at some bands; this ladder
-- reflects the 20-letter published scale.
INSERT INTO mdm.rating_scale
    (tenant_id, scale_cd, value, rank_no, agency_cd,
     display_order, effective_from, is_active, custom_attributes)
VALUES
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'AAA',  20, 'FITCH', 20, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'AA+',  19, 'FITCH', 19, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'AA',   18, 'FITCH', 18, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'AA-',  17, 'FITCH', 17, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'A+',   16, 'FITCH', 16, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'A',    15, 'FITCH', 15, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'A-',   14, 'FITCH', 14, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'BBB+', 13, 'FITCH', 13, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'BBB',  12, 'FITCH', 12, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'BBB-', 11, 'FITCH', 11, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'BB+',  10, 'FITCH', 10, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'BB',   9,  'FITCH', 9,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'BB-',  8,  'FITCH', 8,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'B+',   7,  'FITCH', 7,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'B',    6,  'FITCH', 6,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'B-',   5,  'FITCH', 5,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'CCC',  4,  'FITCH', 4,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'CC',   3,  'FITCH', 3,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'C',    2,  'FITCH', 2,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'FITCH_LT', 'D',    1,  'FITCH', 1,  CURRENT_DATE, true, '{}'::jsonb)
ON CONFLICT (tenant_id, scale_cd, value) DO UPDATE SET
    rank_no = EXCLUDED.rank_no,
    is_active = EXCLUDED.is_active;

-- ── INT_SCALE: 16 values, ranks 16..1 ────────────────────────────────
-- Internal rating model output. MODEL-DEPENDENT: if the model emits a
-- rank outside this 16..1 range, the loader rejects the row with
-- VALUE_NOT_ON_SCALE. Either add the seed row or remap the model.
-- Default choice: contiguous 16..1.
INSERT INTO mdm.rating_scale
    (tenant_id, scale_cd, value, rank_no, agency_cd,
     display_order, effective_from, is_active, custom_attributes)
VALUES
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'A+',   16, 'INTERNAL', 16, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'A',    15, 'INTERNAL', 15, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'A-',   14, 'INTERNAL', 14, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'BBB+', 13, 'INTERNAL', 13, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'BBB',  12, 'INTERNAL', 12, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'BBB-', 11, 'INTERNAL', 11, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'BB+',  10, 'INTERNAL', 10, CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'BB',   9,  'INTERNAL', 9,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'BB-',  8,  'INTERNAL', 8,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'B+',   7,  'INTERNAL', 7,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'B',    6,  'INTERNAL', 6,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'B-',   5,  'INTERNAL', 5,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'CCC',  4,  'INTERNAL', 4,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'CC',   3,  'INTERNAL', 3,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'C',    2,  'INTERNAL', 2,  CURRENT_DATE, true, '{}'::jsonb),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'INT_SCALE', 'D',    1,  'INTERNAL', 1,  CURRENT_DATE, true, '{}'::jsonb)
ON CONFLICT (tenant_id, scale_cd, value) DO UPDATE SET
    rank_no = EXCLUDED.rank_no,
    is_active = EXCLUDED.is_active;

-- Set agency_id (FK) on every seeded row to the corresponding mdm.rating_agency.
-- Idempotent.
UPDATE mdm.rating_scale rs
   SET agency_id = ra.id
  FROM mdm.rating_agency ra
 WHERE rs.tenant_id = ra.tenant_id
   AND rs.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid
   AND rs.agency_id IS NULL
   AND rs.agency_cd = ra.agency_cd;

DO $v$
DECLARE n int;
BEGIN
    SELECT count(*) INTO n FROM mdm.rating_scale
     WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
    IF n < 79 THEN
        RAISE EXCEPTION 'rating_scale under-seeded: % rows (expected 79 = 22+21+20+16)', n;
    END IF;
    RAISE NOTICE '020_mdm_rating_scale: % rows present', n;
END
$v$;

COMMIT;
