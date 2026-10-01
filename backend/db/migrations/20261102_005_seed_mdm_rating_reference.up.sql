-- 20261102_005_seed_mdm_rating_reference.up.sql
-- Seed rating reference data for the gold-copy tenant.
-- Rows: rating_agency 12, rating_type 16, rating_action_type 11,
--       rating_outlook 6, rating_watch 4, rating_scale 31, rating_scale_map 63
-- Total: 143

-- ── rating_agency (12) ─────────────────────────────────────────────────
INSERT INTO mdm.rating_agency
    (agency_cd, name, short_name, agency_type, domicile,
     is_nrsro, is_ecai, is_naic_acceptable, is_active, status, tenant_id)
VALUES
    ('SP',          'S&P Global Ratings',        'S&P',          'NRSRO',           'US', true,  true,  true,  true, 'ACTIVE', '00000000-0000-0000-0000-000000000001'),
    ('MOODY',       'Moody''s Investors Service','Moody''s',     'NRSRO',           'US', true,  true,  true,  true, 'ACTIVE', '00000000-0000-0000-0000-000000000001'),
    ('FITCH',       'Fitch Ratings',             'Fitch',        'NRSRO',           'US', true,  true,  true,  true, 'ACTIVE', '00000000-0000-0000-0000-000000000001'),
    ('DBRS',        'DBRS Morningstar',          'DBRS',         'NRSRO',           'CA', true,  true,  false, true, 'ACTIVE', '00000000-0000-0000-0000-000000000001'),
    ('AMBEST',      'AM Best',                   'AM Best',      'NRSRO',           'US', true,  true,  false, true, 'ACTIVE', '00000000-0000-0000-0000-000000000001'),
    ('EJR',         'Egan-Jones Ratings Company','Egan-Jones',   'NRSRO',           'US', true,  true,  false, true, 'ACTIVE', '00000000-0000-0000-0000-000000000001'),
    ('KBRA',        'Kroll Bond Rating Agency',  'KBRA',         'NRSRO',           'US', true,  true,  false, true, 'ACTIVE', '00000000-0000-0000-0000-000000000001'),
    ('SCOPE',       'Scope Ratings GmbH',        'Scope',        'REGIONAL',        'DE', false, true,  false, true, 'ACTIVE', '00000000-0000-0000-0000-000000000001'),
    ('CRISIL',      'CRISIL Ratings',            'CRISIL',       'NATIONAL',        'IN', false, false, false, true, 'ACTIVE', '00000000-0000-0000-0000-000000000001'),
    ('MORNINGSTAR', 'Morningstar Ratings',       'Morningstar',  'ECAI',            'US', false, true,  false, true, 'ACTIVE', '00000000-0000-0000-0000-000000000001'),
    ('INTERNAL',    'Internal Ratings Model',    'Internal',     'INTERNAL',        NULL, false, false, false, true, 'ACTIVE', '00000000-0000-0000-0000-000000000001'),
    ('MKTIMPL',     'Market Implied Ratings',    'Market-Implied','MARKET_IMPLIED', NULL, false, false, false, true, 'ACTIVE', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, agency_cd) DO NOTHING;

-- ── rating_type (16) ───────────────────────────────────────────────────
INSERT INTO mdm.rating_type
    (type_cd, name, applies_to, rating_basis, is_active, tenant_id)
VALUES
    ('ISSUER_LT', 'Issuer Long-Term Rating',      'ISSUER',           'LONG_TERM',    true, '00000000-0000-0000-0000-000000000001'),
    ('ISSUER_ST', 'Issuer Short-Term Rating',     'ISSUER',           'SHORT_TERM',   true, '00000000-0000-0000-0000-000000000001'),
    ('ISSUE_SEN', 'Senior Unsecured Issue',       'ISSUE',            'LONG_TERM',    true, '00000000-0000-0000-0000-000000000001'),
    ('ISSUE_SUB', 'Subordinated Issue',           'ISSUE',            'LONG_TERM',    true, '00000000-0000-0000-0000-000000000001'),
    ('ISSUE_SR',  'Secured Issue',                'ISSUE',            'LONG_TERM',    true, '00000000-0000-0000-0000-000000000001'),
    ('TRANCHE',   'Tranche Rating',               'TRANCHES',         'LONG_TERM',    true, '00000000-0000-0000-0000-000000000001'),
    ('CPTY',      'Counterparty Credit Rating',   'COUNTERPARTY',     'LONG_TERM',    true, '00000000-0000-0000-0000-000000000001'),
    ('SOV',       'Sovereign Long-Term Rating',   'SOVEREIGN',        'LONG_TERM',    true, '00000000-0000-0000-0000-000000000001'),
    ('SUBSOV',    'Sub-Sovereign Rating',         'SUB_SOVEREIGN',    'LONG_TERM',    true, '00000000-0000-0000-0000-000000000001'),
    ('BFSR',      'Bank Financial Strength',      'BANK',             'POINT_IN_TIME',true, '00000000-0000-0000-0000-000000000001'),
    ('IFS',       'Insurer Financial Strength',   'INSURER',          'LONG_TERM',    true, '00000000-0000-0000-0000-000000000001'),
    ('FCR',       'Fund Credit Risk Rating',      'FUND',             'LONG_TERM',    true, '00000000-0000-0000-0000-000000000001'),
    ('FUND_OVR',  'Fund Overall Rating',          'FUND',             'LONG_TERM',    true, '00000000-0000-0000-0000-000000000001'),
    ('FS',        'Financial Strength Generic',   'FINANCIAL_STRENGTH','POINT_IN_TIME',true,'00000000-0000-0000-0000-000000000001'),
    ('ISS_GRP',   'Issuer Group Rating',          'ISSUER',           'LONG_TERM',    true, '00000000-0000-0000-0000-000000000001'),
    ('CP',        'Commercial Paper Rating',      'ISSUE',            'SHORT_TERM',   true, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, type_cd) DO NOTHING;

-- ── rating_action_type (11) ────────────────────────────────────────────
INSERT INTO mdm.rating_action_type
    (action_cd, name, direction, is_credit_event, is_active, tenant_id)
VALUES
    ('UPG', 'Upgrade',           'UPGRADE',             false, true, '00000000-0000-0000-0000-000000000001'),
    ('DWN', 'Downgrade',         'DOWNGRADE',           false, true, '00000000-0000-0000-0000-000000000001'),
    ('AFF', 'Affirmation',       'AFFIRMATION',         false, true, '00000000-0000-0000-0000-000000000001'),
    ('INI', 'Initial Rating',    'INITIAL',             false, true, '00000000-0000-0000-0000-000000000001'),
    ('WDR', 'Withdrawn Rating',  'WITHDRAWN',           false, true, '00000000-0000-0000-0000-000000000001'),
    ('PW',  'Placed on Watch',   'PLACED_ON_WATCH',     false, true, '00000000-0000-0000-0000-000000000001'),
    ('RW',  'Removed from Watch','REMOVED_FROM_WATCH',  false, true, '00000000-0000-0000-0000-000000000001'),
    ('DEF', 'Default',           'DEFAULT',             true,  true, '00000000-0000-0000-0000-000000000001'),
    ('CUR', 'Cure',              'CURE',                false, true, '00000000-0000-0000-0000-000000000001'),
    ('REC', 'Recovery',          'OTHER',               false, true, '00000000-0000-0000-0000-000000000001'),
    ('OLC', 'Outlook Change',    'OTHER',               false, true, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, action_cd) DO NOTHING;

-- ── rating_outlook (6) ─────────────────────────────────────────────────
INSERT INTO mdm.rating_outlook
    (outlook_cd, name, direction, horizon_months, is_active, tenant_id)
VALUES
    ('POS',  'Positive',        'POSITIVE', 12, true, '00000000-0000-0000-0000-000000000001'),
    ('NEG',  'Negative',        'NEGATIVE', 12, true, '00000000-0000-0000-0000-000000000001'),
    ('STA',  'Stable',          'STABLE',   12, true, '00000000-0000-0000-0000-000000000001'),
    ('DEV',  'Developing',      'DEVELOPING',24, true, '00000000-0000-0000-0000-000000000001'),
    ('NEU',  'Neutral',         'NEUTRAL',  12, true, '00000000-0000-0000-0000-000000000001'),
    ('IMPR', 'Improving',       'POSITIVE',  6, true, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, outlook_cd) DO NOTHING;

-- ── rating_watch (4) ───────────────────────────────────────────────────
INSERT INTO mdm.rating_watch
    (watch_cd, name, direction, horizon_days, is_active, tenant_id)
VALUES
    ('POS',  'Watch Positive',    'POSITIVE',    90, true, '00000000-0000-0000-0000-000000000001'),
    ('NEG',  'Watch Negative',    'NEGATIVE',    90, true, '00000000-0000-0000-0000-000000000001'),
    ('DEV',  'CreditWatch Develop','DEVELOPING',180, true, '00000000-0000-0000-0000-000000000001'),
    ('EVOL', 'Watch Evolving',    'EVOLVING',    90, true, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, watch_cd) DO NOTHING;

-- ── rating_scale (31) ──────────────────────────────────────────────────
-- S&P long-term, 21 notches (CC omitted): ranks 1-21
INSERT INTO mdm.rating_scale
    (agency_id, agency_cd, scale_cd, scale_name, value, rank_no,
     rating_category, display_order, is_active, tenant_id)
SELECT
    ra.id, 'SP', 'SP_LT', 'S&P Long-Term', v.value, v.rank_no,
    CASE WHEN v.rank_no <= 10 THEN 'IG'
         WHEN v.rank_no <= 16 THEN 'HY'
         WHEN v.rank_no <= 19 THEN 'CCC'
         WHEN v.rank_no = 20   THEN 'DISTRESSED'
         ELSE 'DEFAULT' END,
    v.rank_no, true, '00000000-0000-0000-0000-000000000001'
FROM mdm.rating_agency ra
CROSS JOIN (VALUES
    ('AAA', 1),('AA+', 2),('AA', 3),('AA-', 4),
    ('A+',  5),('A',  6),('A-', 7),
    ('BBB+',8),('BBB',9),('BBB-',10),
    ('BB+',11),('BB',12),('BB-',13),
    ('B+', 14),('B', 15),('B-', 16),
    ('CCC+',17),('CCC',18),('CCC-',19),
    ('C',  20),('D',  21)
) AS v(value, rank_no)
WHERE ra.tenant_id = '00000000-0000-0000-0000-000000000001'
  AND ra.agency_cd = 'SP'
ON CONFLICT (tenant_id, scale_cd, value) DO NOTHING;

-- Moody's long-term investment grade, 10 notches: ranks 1-10
INSERT INTO mdm.rating_scale
    (agency_id, agency_cd, scale_cd, scale_name, value, rank_no,
     rating_category, display_order, is_active, tenant_id)
SELECT
    ra.id, 'MOODY', 'MOODY_LT', 'Moody''s Long-Term', v.value, v.rank_no,
    'IG', v.rank_no, true, '00000000-0000-0000-0000-000000000001'
FROM mdm.rating_agency ra
CROSS JOIN (VALUES
    ('Aaa', 1),('Aa1', 2),('Aa2', 3),('Aa3', 4),
    ('A1',  5),('A2',  6),('A3',  7),
    ('Baa1',8),('Baa2',9),('Baa3',10)
) AS v(value, rank_no)
WHERE ra.tenant_id = '00000000-0000-0000-0000-000000000001'
  AND ra.agency_cd = 'MOODY'
ON CONFLICT (tenant_id, scale_cd, value) DO NOTHING;

-- ── rating_scale_map (63) ──────────────────────────────────────────────
-- Moody's ↔ S&P IG, both directions: 20
INSERT INTO mdm.rating_scale_map
    (from_agency_cd, from_scale_cd, from_value,
     to_agency_cd,   to_scale_cd,   to_value,
     mapping_type, confidence, is_active, tenant_id)
SELECT fa.from_agency, 'MOODY_LT', v.from_value, 'SP', 'SP_LT', v.to_value,
       'EXACT', 100.00, true, '00000000-0000-0000-0000-000000000001'::uuid
FROM (VALUES
    ('Aaa','AAA'),('Aa1','AA+'),('Aa2','AA'),('Aa3','AA-'),
    ('A1','A+'),('A2','A'),('A3','A-'),
    ('Baa1','BBB+'),('Baa2','BBB'),('Baa3','BBB-')
) AS v(from_value, to_value),
LATERAL (SELECT 'MOODY' AS from_agency) fa
UNION ALL
SELECT 'SP', 'SP_LT', v.from_value, 'MOODY', 'MOODY_LT', v.to_value,
       'EXACT', 100.00, true, '00000000-0000-0000-0000-000000000001'::uuid
FROM (VALUES
    ('AAA','Aaa'),('AA+','Aa1'),('AA','Aa2'),('AA-','Aa3'),
    ('A+','A1'),('A','A2'),('A-','A3'),
    ('BBB+','Baa1'),('BBB','Baa2'),('BBB-','Baa3')
) AS v(from_value, to_value);

-- S&P ↔ Fitch IG (Fitch uses S&P-style letters), both directions: 20
INSERT INTO mdm.rating_scale_map
    (from_agency_cd, from_scale_cd, from_value,
     to_agency_cd,   to_scale_cd,   to_value,
     mapping_type, confidence, is_active, tenant_id)
SELECT v.val, 'SP_LT', v.val, 'FITCH', 'FITCH_LT', v.val,
       'EXACT', 100.00, true, '00000000-0000-0000-0000-000000000001'::uuid
FROM (VALUES ('AAA'),('AA+'),('AA'),('AA-'),('A+'),('A'),('A-'),
             ('BBB+'),('BBB'),('BBB-')) AS v(val)
UNION ALL
SELECT 'FITCH', 'FITCH_LT', v.val, 'SP', 'SP_LT', v.val,
       'EXACT', 100.00, true, '00000000-0000-0000-0000-000000000001'::uuid
FROM (VALUES ('AAA'),('AA+'),('AA'),('AA-'),('A+'),('A'),('A-'),
             ('BBB+'),('BBB'),('BBB-')) AS v(val);

-- Moody's ↔ Fitch IG, both directions: 20
INSERT INTO mdm.rating_scale_map
    (from_agency_cd, from_scale_cd, from_value,
     to_agency_cd,   to_scale_cd,   to_value,
     mapping_type, confidence, is_active, tenant_id)
SELECT 'MOODY', 'MOODY_LT', v.m, 'FITCH', 'FITCH_LT', v.f,
       'EXACT', 98.00, true, '00000000-0000-0000-0000-000000000001'::uuid
FROM (VALUES
    ('Aaa','AAA'),('Aa1','AA+'),('Aa2','AA'),('Aa3','AA-'),
    ('A1','A+'),('A2','A'),('A3','A-'),
    ('Baa1','BBB+'),('Baa2','BBB'),('Baa3','BBB-')
) AS v(m, f)
UNION ALL
SELECT 'FITCH', 'FITCH_LT', v.f, 'MOODY', 'MOODY_LT', v.m,
       'EXACT', 98.00, true, '00000000-0000-0000-0000-000000000001'::uuid
FROM (VALUES
    ('AAA','Aaa'),('AA+','Aa1'),('AA','Aa2'),('AA-','Aa3'),
    ('A+','A1'),('A','A2'),('A-','A3'),
    ('BBB+','Baa1'),('BBB','Baa2'),('BBB-','Baa3')
) AS v(f, m);

-- AM Best → S&P approximations: 3
INSERT INTO mdm.rating_scale_map
    (from_agency_cd, from_scale_cd, from_value,
     to_agency_cd, to_scale_cd, to_value,
     mapping_type, confidence, is_active, tenant_id)
VALUES
    ('AMBEST', 'AMBEST_LT', 'A++', 'SP', 'SP_LT', 'AAA', 'APPROXIMATE', 90.00, true, '00000000-0000-0000-0000-000000000001'),
    ('AMBEST', 'AMBEST_LT', 'A+',  'SP', 'SP_LT', 'AA',  'APPROXIMATE', 90.00, true, '00000000-0000-0000-0000-000000000001'),
    ('AMBEST', 'AMBEST_LT', 'A',   'SP', 'SP_LT', 'A',   'APPROXIMATE', 85.00, true, '00000000-0000-0000-0000-000000000001');
