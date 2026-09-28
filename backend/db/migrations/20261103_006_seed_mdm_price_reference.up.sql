-- 20261103_006_seed_mdm_price_reference.up.sql
-- Seeds price_source, price_type, price_quality_tier, fair_value_level,
-- price_observation_type, curve_category, curve_interpolation_method,
-- curve_compounding_frequency under the shared-reference tenant.
-- Idempotent. 90 rows.

DO $seed$
DECLARE gold uuid := '00000000-0000-0000-0000-000000000001'::uuid;
BEGIN

-- ── price_source ───────────────────────────────────────────────────────
INSERT INTO mdm.price_source
    (tenant_id, source_cd, name, short_name, source_type, vendor_type,
     is_primary_source, priority, is_active)
VALUES
    (gold,'BLOOMBERG','Bloomberg L.P.','BBG','EVALUATOR','BLOOMBERG',true,10,true),
    (gold,'REFINITIV','Refinitiv','RTRS','EVALUATOR','REFINITIV',false,20,true),
    (gold,'ICE','Intercontinental Exchange','ICE','EVALUATOR','ICE',false,30,true),
    (gold,'MARKIT','S&P Global Market Intelligence (Markit)','Markit','EVALUATOR','MARKIT',false,40,true),
    (gold,'IDC','ICE Data Services','IDC','EVALUATOR','IDC',false,50,true),
    (gold,'SIX','SIX Financial Information','SIX','EVALUATOR','SIX',false,60,true),
    (gold,'WM_REFINITIV','WM/Refinitiv FX Fixing','WM','PRICING_SERVICE','REFINITIV',true,10,true),
    (gold,'ECB','European Central Bank Reference Rates','ECB','INTERNAL',NULL,false,20,true),
    (gold,'BFIX','Bloomberg FX Fixings','BFIX','PRICING_SERVICE','BLOOMBERG',false,30,true),
    (gold,'BOE','Bank of England','BOE','INTERNAL',NULL,false,40,true),
    (gold,'FED_H10','Federal Reserve H.10','FED','INTERNAL',NULL,false,50,true),
    (gold,'EXCHANGE_CONSOLIDATED','Consolidated Exchange Feed','EXCH','CONSENSUS',NULL,true,5,true),
    (gold,'INTERNAL_MODEL','Internal Valuation Model','INTMDL','MODEL',NULL,false,100,true),
    (gold,'TRADE_CAPTURE','Captured Trade Price','TRD','TRADE_CAPTURE',NULL,true,1,true),
    (gold,'MANUAL','Manual Override','MAN','INTERNAL',NULL,false,999,true)
ON CONFLICT (tenant_id, source_cd) DO NOTHING;

-- ── price_type ─────────────────────────────────────────────────────────
INSERT INTO mdm.price_type
    (tenant_id, price_type_cd, name, category, price_side, is_executable,
     is_official, requires_model, display_order)
VALUES
    (gold,'BID','Bid Price','QUOTE','BID',true,false,false,10),
    (gold,'ASK','Ask Price','QUOTE','ASK',true,false,false,20),
    (gold,'MID','Mid Price','QUOTE','MID',true,false,false,30),
    (gold,'LAST','Last Traded Price','TRADE','LAST',true,false,false,40),
    (gold,'TRADE','Trade Price','TRADE','LAST',true,false,false,50),
    (gold,'OFFICIAL_CLOSE','Official Close','OFFICIAL','LAST',false,true,false,60),
    (gold,'OFFICIAL_OPEN','Official Open','OFFICIAL','LAST',false,true,false,70),
    (gold,'EVALUATED','Evaluated Price','EVALUATED','MID',false,true,false,80),
    (gold,'MATRIX','Matrix Price','EVALUATED','MID',false,true,false,90),
    (gold,'MODEL','Model Price','MODEL','MID',false,false,true,100),
    (gold,'INTERPOLATED','Interpolated Price','DERIVED','MID',false,false,true,110),
    (gold,'STRESSED','Stressed Price','DERIVED','MID',false,false,true,120),
    (gold,'INDICATIVE','Indicative Price','INDICATIVE','MID',false,false,false,130),
    (gold,'NAV','Net Asset Value','OFFICIAL','MID',false,true,false,140),
    (gold,'PRIOR_CLOSE','Prior Close','OFFICIAL','LAST',false,true,false,150),
    (gold,'ADJUSTED_CLOSE','Adjusted Close','OFFICIAL','LAST',false,true,false,160),
    (gold,'SETTLEMENT','Settlement Price','OFFICIAL','LAST',false,true,false,170),
    (gold,'CLEAN_PRICE','Clean Price','EVALUATED','MID',false,false,false,180),
    (gold,'DIRTY_PRICE','Dirty Price','EVALUATED','MID',false,false,false,190)
ON CONFLICT (tenant_id, price_type_cd) DO NOTHING;

-- ── price_quality_tier ─────────────────────────────────────────────────
INSERT INTO mdm.price_quality_tier
    (tenant_id, tier_cd, name, tier_level, is_executable, is_observable,
     is_level_1, is_level_2, is_level_3, max_staleness_minutes, requires_review)
VALUES
    (gold,'EXCHANGE_TRADED','Exchange Traded',1,true,true,true,false,false,1,false),
    (gold,'DEALER_QUOTED','Dealer Quoted',2,true,true,false,true,false,15,false),
    (gold,'BROKER_QUOTED','Broker Quoted',3,true,true,false,true,false,30,false),
    (gold,'EVALUATED','Evaluated',4,false,true,false,true,false,60,false),
    (gold,'MATRIX','Matrix',5,false,true,false,true,false,1440,true),
    (gold,'MODEL','Model',6,false,false,false,false,true,4320,true),
    (gold,'INDICATIVE','Indicative',7,false,false,false,false,true,1440,false)
ON CONFLICT (tenant_id, tier_cd) DO NOTHING;

-- ── fair_value_level ───────────────────────────────────────────────────
INSERT INTO mdm.fair_value_level
    (tenant_id, level_cd, name, level_number, description)
VALUES
    (gold,'LEVEL_1','Level 1','1','Quoted prices in active markets for identical assets'),
    (gold,'LEVEL_2','Level 2','2','Observable inputs other than Level 1 quoted prices'),
    (gold,'LEVEL_3','Level 3','3','Unobservable inputs for the asset')
ON CONFLICT (tenant_id, level_cd) DO NOTHING;

-- ── price_observation_type ─────────────────────────────────────────────
INSERT INTO mdm.price_observation_type
    (tenant_id, observation_type_cd, name, is_close, is_intraday, is_snapshot)
VALUES
    (gold,'TRADE','Trade',false,false,false),
    (gold,'QUOTE','Quote',false,true,false),
    (gold,'SNAPSHOT','Snapshot',false,true,true),
    (gold,'OFFICIAL_CLOSE','Official Close',true,false,false),
    (gold,'SETTLEMENT','Settlement',true,false,false),
    (gold,'AUCTION','Auction',true,false,false),
    (gold,'REFERENCE','Reference',false,false,false),
    (gold,'FIXING','Fixing',false,false,false),
    (gold,'MARK_TO_MARKET','Mark to Market',true,false,false)
ON CONFLICT (tenant_id, observation_type_cd) DO NOTHING;

-- ── curve_category ─────────────────────────────────────────────────────
INSERT INTO mdm.curve_category
    (tenant_id, category_cd, name, curve_class)
VALUES
    (gold,'OIS_DISCOUNT','OIS Discount Curve','RATE'),
    (gold,'SOFR_DISCOUNT','SOFR Discount Curve','RATE'),
    (gold,'EURIBOR_FORWARD','EURIBOR Forward Curve','RATE'),
    (gold,'LIBOR_LEGACY','LIBOR Legacy Curve','RATE'),
    (gold,'GOVT_YIELD','Government Yield Curve','RATE'),
    (gold,'CORP_CREDIT_SPREAD','Corporate Credit Spread Curve','CREDIT'),
    (gold,'CDS_SPREAD','CDS Spread Curve','CREDIT'),
    (gold,'INFLATION_BREAKEVEN','Inflation Breakeven Curve','INFLATION'),
    (gold,'INFLATION_ZERO','Inflation Zero Curve','INFLATION'),
    (gold,'FX_FORWARD','FX Forward Curve','FX'),
    (gold,'FX_SWAP','FX Swap Curve','FX'),
    (gold,'EQUITY_INDEX_VOL','Equity Index Vol Surface','VOLATILITY'),
    (gold,'SINGLE_NAME_VOL','Single Name Vol Surface','VOLATILITY'),
    (gold,'FX_VOL','FX Vol Surface','VOLATILITY'),
    (gold,'IR_VOL','Interest Rate Vol Surface','VOLATILITY'),
    (gold,'COMMODITY_FORWARD','Commodity Forward Curve','COMMODITY'),
    (gold,'COMMODITY_VOL','Commodity Vol Surface','VOLATILITY'),
    (gold,'MUNICIPAL_AAA','Municipal AAA Curve','MUNICIPAL'),
    (gold,'MUNICIPAL_AA','Municipal AA Curve','MUNICIPAL'),
    (gold,'MUNICIPAL_A','Municipal A Curve','MUNICIPAL'),
    (gold,'FUNDING_FC','Funding Curve','FUNDING'),
    (gold,'COLLATERAL_CSA','Collateral CSA Curve','COLLATERAL')
ON CONFLICT (tenant_id, category_cd) DO NOTHING;

-- ── curve_interpolation_method ─────────────────────────────────────────
INSERT INTO mdm.curve_interpolation_method
    (tenant_id, method_cd, name)
VALUES
    (gold,'LINEAR','Linear'),
    (gold,'LOG_LINEAR','Log-Linear'),
    (gold,'CUBIC_SPLINE','Cubic Spline'),
    (gold,'MONOTONE_CUBIC','Monotone Cubic'),
    (gold,'FLAT_FORWARD','Flat Forward'),
    (gold,'LOG_CUBIC','Log-Cubic'),
    (gold,'NELSON_SIEGEL','Nelson-Siegel'),
    (gold,'SVENSSON','Svensson'),
    (gold,'PIECEWISE_CONSTANT','Piecewise Constant')
ON CONFLICT (tenant_id, method_cd) DO NOTHING;

-- ── curve_compounding_frequency ────────────────────────────────────────
INSERT INTO mdm.curve_compounding_frequency
    (tenant_id, compounding_cd, name, periods_per_year)
VALUES
    (gold,'CONTINUOUS','Continuous',NULL),
    (gold,'ANNUAL','Annual',1),
    (gold,'SEMI_ANNUAL','Semi-Annual',2),
    (gold,'QUARTERLY','Quarterly',4),
    (gold,'MONTHLY','Monthly',12),
    (gold,'SIMPLE','Simple',NULL)
ON CONFLICT (tenant_id, compounding_cd) DO NOTHING;

END
$seed$;
