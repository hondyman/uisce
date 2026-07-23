-- backend/migrations/017_security_master_seed.sql
-- Seeds:
--   1. Prerequisites: Missing gold copy tables (survivorship_rules, dq_rules)
--   2. Semantic Terms for Security Master in edm.semantic_terms
--   3. Survivorship rules in edm.survivorship_rules
--   4. DQ rules in edm.dq_rules
--   5. Demo issuers and securities

-- ============================================================
-- SECTION 0: PREREQUISITES
-- ============================================================

CREATE TABLE IF NOT EXISTS edm.survivorship_rules (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    core_id                  UUID REFERENCES edm.survivorship_rules(id),
    entity_type              VARCHAR(50) NOT NULL,
    field_name               VARCHAR(100) NOT NULL,
    strategy                 VARCHAR(50) NOT NULL,
    preferred_sources        TEXT[] NOT NULL DEFAULT '{}',
    condition_expression     TEXT,
    time_field               VARCHAR(100),
    priority                 INT NOT NULL DEFAULT 1,
    is_active                BOOLEAN NOT NULL DEFAULT true,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by               UUID NOT NULL,
    CONSTRAINT uq_survivorship_rule UNIQUE (tenant_id, entity_type, field_name, priority)
);

CREATE TABLE IF NOT EXISTS edm.dq_rules (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    entity_type              VARCHAR(50) NOT NULL,
    rule_name                VARCHAR(100) NOT NULL,
    rule_expression          TEXT NOT NULL,
    severity                 VARCHAR(20) DEFAULT 'Soft' CHECK (severity IN ('Hard', 'Soft', 'Info')),
    is_active                BOOLEAN DEFAULT true,
    created_at               TIMESTAMPTZ DEFAULT now(),
    updated_at               TIMESTAMPTZ DEFAULT now(),
    created_by               UUID,
    CONSTRAINT uq_dq_rule UNIQUE (tenant_id, entity_type, rule_name)
);

-- ============================================================
-- SECTION 1: SEMANTIC TERMS (edm.semantic_terms)
-- ============================================================

INSERT INTO edm.semantic_terms (name, data_type, definition)
VALUES
    ('SecurityID',                'uuid',    'Unique internal identifier for the security'),
    ('PrimaryIdentifier',         'text',    'Preferred business identifier (e.g. ISIN or Ticker)'),
    ('ISIN',                      'text',    'International Securities Identification Number'),
    ('CUSIP',                     'text',    'Committee on Uniform Securities Identification Procedures'),
    ('SEDOL',                     'text',    'Stock Exchange Daily Official List'),
    ('FIGI',                      'text',    'Financial Instrument Global Identifier'),
    ('Ticker',                    'text',    'Market ticker symbol'),
    ('SecurityName',              'text',    'Full legal name of the security'),
    ('AssetClass',                'text',    'Top-level asset classification (Equity, FI, etc.)'),
    ('SubAssetClass',             'text',    'Detailed asset classification'),
    ('InstrumentType',            'text',    'Specific instrument type (Common Stock, Corp Bond, etc.)'),
    ('CurrencyCode',              'text',    'Three-letter ISO currency code'),
    ('CountryOfIssue',            'text',    'ISO country code of issuance'),
    ('IssueDate',                 'date',    'Date the security was issued'),
    ('MaturityDate',              'date',    'Date the security matures'),
    ('IssuerID',                  'uuid',    'Reference to the legal entity issuer'),
    ('SecurityStatus',            'text',    'Operational status (Active, Matured, etc.)'),
    -- Fixed Income attributes
    ('CouponType',                'text',    'Type of coupon (Fixed, Floating, Zero)'),
    ('CouponRate',                'numeric', 'Annual coupon rate as a percentage'),
    ('CouponFrequency',           'text',    'Frequency of coupon payments'),
    ('DayCountConvention',        'text',    'Method for calculating interest (e.g. 30/360)'),
    ('ParValue',                  'numeric', 'Face value of the instrument'),
    ('IssueSize',                 'numeric', 'Total amount issued'),
    ('RatingComposite',           'text',    'Aggregated credit rating'),
    ('Seniority',                 'text',    'Debt seniority level'),
    -- Equity attributes
    ('ShareClass',                'text',    'Class of shares (Common A, Preferred, etc.)'),
    ('SharesOutstanding',         'numeric', 'Total number of shares issued'),
    ('DividendYield',             'numeric', 'Trailing 12-month dividend yield'),
    ('DividendFrequency',         'text',    'Frequency of dividend payments'),
    -- Fund attributes
    ('FundType',                  'text',    'Type of fund (Mutual, ETF, etc.)'),
    ('ManagementCompany',         'text',    'Company managing the fund portfolio'),
    ('TotalExpenseRatio',         'numeric', 'Annual cost of owning the fund'),
    ('DistributionPolicy',        'text',    'Policy for distributing income (Accumulating, Distributing)'),
    -- Derivative attributes
    ('UnderlierSecurityID',       'uuid',    'Reference to the underlying instrument'),
    ('StrikePrice',               'numeric', 'Price at which the derivative can be exercised'),
    ('OptionType',                'text',    'Call or Put'),
    ('ExerciseStyle',             'text',    'American, European, etc.'),
    ('ExpiryDate',                'date',    'Date the contract expires')
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- SECTION 2: SURVIVORSHIP RULES
-- ============================================================

INSERT INTO edm.survivorship_rules
    (id, tenant_id, entity_type, field_name, strategy, preferred_sources, priority, is_active, created_by)
VALUES
    -- Core Security root
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'security', 'isin',            'prefer_source',    ARRAY['Bloomberg','Refinitiv','ICE'],            1, true, '00000000-0000-0000-0000-000000000001'),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'security', 'cusip',           'prefer_source',    ARRAY['Bloomberg','Refinitiv'],                  2, true, '00000000-0000-0000-0000-000000000001'),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'security', 'figi',            'prefer_source',    ARRAY['Bloomberg'],                              3, true, '00000000-0000-0000-0000-000000000001'),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'security', 'security_name',   'prefer_source',    ARRAY['Bloomberg','Refinitiv'],                  4, true, '00000000-0000-0000-0000-000000000001'),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'security', 'asset_class',     'prefer_source',    ARRAY['Bloomberg','Refinitiv','ICE'],            5, true, '00000000-0000-0000-0000-000000000001'),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'security', 'currency',        'prefer_source',    ARRAY['Bloomberg','Custodian'],                  6, true, '00000000-0000-0000-0000-000000000001'),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'security', 'issuer_id',       'prefer_source',    ARRAY['Bloomberg','Refinitiv','Admin'],           7, true, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, entity_type, field_name, priority) DO NOTHING;

-- ============================================================
-- SECTION 3: DQ RULES
-- ============================================================

INSERT INTO edm.dq_rules
    (id, tenant_id, entity_type, rule_name, rule_expression, severity, is_active, created_by)
VALUES
    -- Core Security DQ
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'security',
     'Security_RequiredIdentifiers',
     '{"require_any": ["isin","cusip","figi"], "require": ["security_name"]}',
     'Hard', true, '00000000-0000-0000-0000-000000000001'),

    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'security',
     'Security_AssetClassValidity',
     '{"field":"asset_class","allowed_values":["Equity","FixedIncome","Fund","Derivative","FX","Commodity"]}',
     'Hard', true, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, entity_type, rule_name) DO NOTHING;

-- ============================================================
-- SECTION 4: DEMO DATA
-- ============================================================

INSERT INTO edm.issuer_master
    (id, tenant_id, issuer_id, issuer_name, short_name, lei, country_of_incorporation,
     sector, industry, rating_composite, status, created_by)
VALUES
    ('11100000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001',
     'ISS-AAPL', 'Apple Inc.', 'Apple', '549300P5JKL7MXQ0N014', 'USA',
     'Technology', 'Consumer Electronics', 'AA+', 'Active', '00000000-0000-0000-0000-000000000001'),
    ('11100000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001',
     'ISS-GS', 'Goldman Sachs Group Inc.', 'Goldman', 'W22LROWP2IHZNBB6K528', 'USA',
     'Financials', 'Investment Banking', 'A+', 'Active', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;

INSERT INTO edm.security_master
    (id, tenant_id, security_id, primary_identifier, isin, cusip, figi, ticker,
     bbg_id, security_name, short_name, asset_class, sub_asset_class, instrument_type,
     sector, currency, country_of_issue, region, issuer_id, listing_exchange, exchange_code,
     status, liquidity_profile, confidence_score, created_by)
VALUES
    ('22200000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001',
     'SEC-AAPL', 'US0378331005', 'US0378331005', '037833100', 'BBG000B9XRY4', 'AAPL',
     'AAPL US Equity', 'Apple Inc.', 'Apple', 'Equity', 'Large Cap Equity', 'Common Stock',
     'Technology', 'USD', 'USA', 'NAM', '11100000-0000-0000-0000-000000000001', 'NASDAQ', 'XNAS',
     'Active', 'High', 98, '00000000-0000-0000-0000-000000000001'),
    ('22200000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001',
     'SEC-GS-4.25-27', 'US38141GXB92', 'US38141GXB92', '38141GXB9', 'BBG00KZFQ4F5', 'GS',
     'GS 4.25 10/21/27 Corp', 'Goldman Sachs 4.25% 2027', 'GS 4.25 27', 'FixedIncome', 'IG Corp', 'Corporate Bond',
     'Financials', 'USD', 'USA', 'NAM', '11100000-0000-0000-0000-000000000002', 'NYSE', 'XNYS',
     'Active', 'High', 95, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;
