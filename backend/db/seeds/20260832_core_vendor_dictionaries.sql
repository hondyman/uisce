-- 20260832_core_vendor_dictionaries.sql
-- Seed Canonical Bloomberg, Refinitiv/LSEG, and FactSet Data Dictionaries into Core Gold Copy

INSERT INTO catalog_vendor.vendor_data_dictionary (
    vendor_name, field_mnemonic, field_name, category,
    feed_type, data_type, description, aliases, standards_mapping
) VALUES 
(
    'BLOOMBERG', 'PX_LAST', 'Last Price', 'Pricing & Valuation',
    'Data License / B-PIPE', 'NUMERIC(18,6)',
    'Official closing price or last traded market price across primary listed exchange.',
    ARRAY['last_price', 'close_px', 'closing_price', 'latest_market_price'],
    '{"fibo_uri": "https://spec.edmcouncil.org/fibo/ontology/SEC/Securities/SecuritiesPricing/ClosingPrice"}'::jsonb
),
(
    'BLOOMBERG', 'ID_ISIN', 'ISIN Identifier', 'Symbology',
    'Data License / B-PIPE', 'VARCHAR(12)',
    'International Securities Identification Number compliant with ISO 6166.',
    ARRAY['isin', 'isin_code', 'security_isin', 'instrument_isin'],
    '{"iso_standard": "ISO 6166"}'::jsonb
),
(
    'BLOOMBERG', 'ID_CUSIP', 'CUSIP Identifier', 'Symbology',
    'Data License / B-PIPE', 'VARCHAR(9)',
    'Committee on Uniform Security Identification Procedures 9-character code.',
    ARRAY['cusip', 'cusip_code', 'security_cusip'],
    '{"standard": "ANSI X9.6"}'::jsonb
),
(
    'BLOOMBERG', 'ID_SEDOL1', 'SEDOL Identifier', 'Symbology',
    'Data License / B-PIPE', 'VARCHAR(7)',
    'Stock Exchange Daily Official List 7-character alphanumeric code.',
    ARRAY['sedol', 'sedol_code', 'sedol1'],
    '{"standard": "LSE SEDOL"}'::jsonb
),
(
    'BLOOMBERG', 'ID_BB_GLOBAL', 'FIGI (Financial Instrument Global Identifier)', 'Symbology',
    'Data License / OpenFIGI', 'VARCHAR(12)',
    'Open, persistent 12-character semantic identifier for financial instruments (OMG standard).',
    ARRAY['figi', 'bbgid', 'openfigi', 'global_id'],
    '{"standard": "OMG FIGI"}'::jsonb
),
(
    'BLOOMBERG', 'YLD_YTM_MID', 'Yield to Maturity (Mid)', 'Fixed Income',
    'Data License / B-PIPE', 'NUMERIC(10,6)',
    'Annualized rate of return anticipated on a bond held until maturity based on mid-market price.',
    ARRAY['ytm', 'yield_to_maturity', 'yield_mid', 'ytm_mid'],
    '{"fibo_uri": "https://spec.edmcouncil.org/fibo/ontology/SEC/Debt/Bonds/YieldToMaturity"}'::jsonb
),
(
    'BLOOMBERG', 'DUR_ADJ_MID', 'Modified Duration (Mid)', 'Fixed Income',
    'Data License / B-PIPE', 'NUMERIC(10,4)',
    'Price sensitivity measure of a bond to interest rate fluctuations based on mid-market price.',
    ARRAY['modified_duration', 'mod_dur', 'duration_mid', 'dur_adj'],
    '{"fibo_uri": "https://spec.edmcouncil.org/fibo/ontology/SEC/Debt/Bonds/ModifiedDuration"}'::jsonb
),
(
    'BLOOMBERG', 'CONVEXITY_MID', 'Convexity (Mid)', 'Fixed Income',
    'Data License / B-PIPE', 'NUMERIC(10,4)',
    'Second derivative curvature measure of price-yield relationship for debt securities.',
    ARRAY['convexity', 'convexity_mid', 'bond_convexity'],
    '{"fibo_uri": "https://spec.edmcouncil.org/fibo/ontology/SEC/Debt/Bonds/Convexity"}'::jsonb
),
(
    'BLOOMBERG', 'EQY_SH_OUT', 'Shares Outstanding', 'Equity & Fundamentals',
    'Data License / Fundamentals', 'NUMERIC(18,2)',
    'Total number of authorized and issued shares currently held by all shareholders.',
    ARRAY['shares_outstanding', 'shares_out', 'total_shares_outstanding'],
    '{"fibo_uri": "https://spec.edmcouncil.org/fibo/ontology/SEC/Equities/EquityInstruments/SharesOutstanding"}'::jsonb
),
(
    'BLOOMBERG', 'CRNCY', 'Pricing Currency', 'Reference Data',
    'Data License / B-PIPE', 'VARCHAR(3)',
    'Three-letter ISO 4217 currency code in which the instrument is priced and traded.',
    ARRAY['currency', 'quote_currency', 'pricing_ccy', 'ccy'],
    '{"iso_standard": "ISO 4217"}'::jsonb
),
(
    'REFINITIV', 'TR.PriceClose', 'Closing Price', 'Pricing & Valuation',
    'DataScope / Elektron', 'NUMERIC(18,6)',
    'Refinitiv official closing market trade price for the instrument.',
    ARRAY['refinitiv_close', 'ric_close', 'price_close'],
    '{"fibo_uri": "https://spec.edmcouncil.org/fibo/ontology/SEC/Securities/SecuritiesPricing/ClosingPrice"}'::jsonb
),
(
    'REFINITIV', 'TR.RIC', 'Reuters Instrument Code (RIC)', 'Symbology',
    'DataScope / Elektron', 'VARCHAR(32)',
    'Ticker-like code used by Refinitiv/LSEG to identify financial instruments and market data.',
    ARRAY['ric', 'reuters_code', 'instrument_ric'],
    '{"vendor": "REFINITIV"}'::jsonb
)
ON CONFLICT (vendor_name, field_mnemonic) DO UPDATE
SET field_name = EXCLUDED.field_name,
    category = EXCLUDED.category,
    description = EXCLUDED.description,
    aliases = EXCLUDED.aliases,
    standards_mapping = EXCLUDED.standards_mapping;
