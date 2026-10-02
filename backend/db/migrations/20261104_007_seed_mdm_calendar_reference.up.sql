-- 20261104_007_seed_mdm_calendar_reference.up.sql
-- Seeds calendar reference tables under the gold tenant.
-- Rows: calendar_type 25, time_zone 25, calendar_source 14, holiday_type 20,
--       holiday_rule_type 14, rolling_convention 10, business_day_definition 8,
--       calendar_hierarchy_type 10, calendar_regulatory_regime 11. Total 137.

DO $seed$
DECLARE gold uuid := '00000000-0000-0000-0000-000000000001'::uuid;
BEGIN

-- ── calendar_type ──────────────────────────────────────────────────────
INSERT INTO mdm.calendar_type
    (tenant_id, calendar_type_cd, name, category, applies_to_entity_type,
     required_for_settlement, required_for_valuation, required_for_dealing, display_order)
VALUES
    (gold,'EXCHANGE_TRADING','Exchange Trading Calendar','MARKET','MARKET',false,false,true,10),
    (gold,'EXCHANGE_SETTLEMENT','Exchange Settlement Calendar','SETTLEMENT','MARKET',true,false,false,20),
    (gold,'EXCHANGE_CLEARING','Exchange Clearing Calendar','SETTLEMENT','MARKET',true,false,false,30),
    (gold,'SIFMA_BOND','SIFMA Bond Market Calendar','MARKET','MARKET',false,false,true,40),
    (gold,'SIFMA_MUNI','SIFMA Municipal Market Calendar','MARKET','MARKET',false,false,true,50),
    (gold,'FEDWIRE','Fedwire Calendar','CENTRAL_BANK','CURRENCY',true,false,false,60),
    (gold,'TARGET2','TARGET2 Calendar','CENTRAL_BANK','CURRENCY',true,false,false,70),
    (gold,'CHAPS','CHAPS Calendar','CENTRAL_BANK','CURRENCY',true,false,false,80),
    (gold,'BANK_HOLIDAY_US','US Bank Holiday Calendar','PAYMENT','COUNTRY',true,false,false,90),
    (gold,'BANK_HOLIDAY_UK','UK Bank Holiday Calendar','PAYMENT','COUNTRY',true,false,false,100),
    (gold,'BANK_HOLIDAY_EU','EU Bank Holiday Calendar','PAYMENT','COUNTRY',true,false,false,110),
    (gold,'TAX_CALENDAR','Tax Calendar','TAX','COUNTRY',false,false,false,120),
    (gold,'FISCAL_CALENDAR','Fiscal Calendar','FISCAL','COUNTRY',false,false,false,130),
    (gold,'MUTUAL_FUND_DEALING','Mutual Fund Dealing Calendar','FUND','FUND',false,false,true,140),
    (gold,'MUTUAL_FUND_VALUATION','Mutual Fund Valuation Calendar','FUND','FUND',false,true,false,150),
    (gold,'ETF_CREATION','ETF Creation/Redemption Calendar','FUND','FUND',false,false,true,160),
    (gold,'BENCHMARK_FIXING','Benchmark Fixing Calendar','BENCHMARK','BENCHMARK',false,false,false,170),
    (gold,'BENCHMARK_REBALANCE','Benchmark Rebalance Calendar','BENCHMARK','BENCHMARK',false,false,false,180),
    (gold,'INDEX_CALCULATION','Index Calculation Calendar','BENCHMARK','BENCHMARK',false,false,false,190),
    (gold,'FX_SETTLEMENT','FX Settlement Calendar','SETTLEMENT','CURRENCY',true,false,false,200),
    (gold,'FX_FIXING','FX Fixing Calendar','BENCHMARK','CURRENCY',false,false,false,210),
    (gold,'MONEY_MARKET_DEALING','Money Market Dealing Calendar','MARKET','CURRENCY',false,false,true,220),
    (gold,'SWIFT_MESSAGE','SWIFT Message Calendar','OPERATIONAL','MARKET',false,false,false,230),
    (gold,'CLS_SETTLEMENT','CLS Settlement Calendar','SETTLEMENT','CURRENCY',true,false,false,240),
    (gold,'DEPOSITORY_TRUST','Depository Trust Calendar','SETTLEMENT','MARKET',true,false,false,250)
ON CONFLICT (tenant_id, calendar_type_cd) DO NOTHING;

-- ── time_zone (major financial centers) ────────────────────────────────
INSERT INTO mdm.time_zone
    (tenant_id, tz_name, display_name, utc_offset_standard_minutes, utc_offset_dst_minutes, observes_dst)
VALUES
    (gold,'America/New_York','New York',-300,-240,true),
    (gold,'America/Chicago','Chicago',-360,-300,true),
    (gold,'America/Los_Angeles','Los Angeles',-480,-420,true),
    (gold,'America/Toronto','Toronto',-300,-240,true),
    (gold,'America/Sao_Paulo','Sao Paulo',-180,-180,false),
    (gold,'Europe/London','London',0,60,true),
    (gold,'Europe/Paris','Paris',60,120,true),
    (gold,'Europe/Frankfurt','Frankfurt',60,120,true),
    (gold,'Europe/Zurich','Zurich',60,120,true),
    (gold,'Europe/Luxembourg','Luxembourg',60,120,true),
    (gold,'Europe/Dublin','Dublin',0,60,true),
    (gold,'Europe/Amsterdam','Amsterdam',60,120,true),
    (gold,'Europe/Madrid','Madrid',60,120,true),
    (gold,'Europe/Milan','Milan',60,120,true),
    (gold,'Europe/Stockholm','Stockholm',60,120,true),
    (gold,'Asia/Tokyo','Tokyo',540,540,false),
    (gold,'Asia/Hong_Kong','Hong Kong',480,480,false),
    (gold,'Asia/Singapore','Singapore',480,480,false),
    (gold,'Asia/Shanghai','Shanghai',480,480,false),
    (gold,'Asia/Seoul','Seoul',540,540,false),
    (gold,'Asia/Mumbai','Mumbai',330,330,false),
    (gold,'Asia/Dubai','Dubai',240,240,false),
    (gold,'Australia/Sydney','Sydney',600,660,true),
    (gold,'Pacific/Auckland','Auckland',720,780,true),
    (gold,'UTC','UTC',0,0,false)
ON CONFLICT (tenant_id, tz_name) DO NOTHING;

-- ── calendar_source ────────────────────────────────────────────────────
INSERT INTO mdm.calendar_source
    (tenant_id, source_cd, name, short_name, source_type, vendor_type,
     authority_level, coverage_scope, is_active)
VALUES
    (gold,'ICE_EXCHANGE_CALENDARS','ICE Exchange Calendar Service','ICE','EXCHANGE','ICE','AUTHORITATIVE','VENUE',true),
    (gold,'NYSE_CALENDARS','NYSE Calendar Service','NYSE','EXCHANGE','ICE','AUTHORITATIVE','VENUE',true),
    (gold,'NASDAQ_CALENDARS','Nasdaq Calendar Service','Nasdaq','EXCHANGE','NASDAQ','AUTHORITATIVE','VENUE',true),
    (gold,'LSE_CALENDARS','London Stock Exchange Calendar','LSE','EXCHANGE',NULL,'AUTHORITATIVE','VENUE',true),
    (gold,'TARGET2_CALENDARS','TARGET2 Calendar','TARGET2','CENTRAL_BANK',NULL,'AUTHORITATIVE','COUNTRY',true),
    (gold,'FEDWIRE_CALENDARS','Federal Reserve Fedwire Calendar','FED','CENTRAL_BANK',NULL,'AUTHORITATIVE','COUNTRY',true),
    (gold,'SIFMA_CALENDARS','SIFMA Holiday Recommendations','SIFMA','REGULATOR',NULL,'RECOMMENDED','COUNTRY',true),
    (gold,'BLOOMBERG_CALENDARS','Bloomberg Calendar Service','BBG','DATA_VENDOR','BLOOMBERG','AUTHORITATIVE','GLOBAL',true),
    (gold,'REFINITIV_CALENDARS','Refinitiv Calendar Service','RTRS','DATA_VENDOR','REFINITIV','AUTHORITATIVE','GLOBAL',true),
    (gold,'WM_FX_CALENDARS','WM FX Fixing Calendar','WM','DATA_VENDOR','REFINITIV','AUTHORITATIVE','GLOBAL',true),
    (gold,'ISDA_CALENDARS','ISDA Calendar Book','ISDA','REGULATOR',NULL,'AUTHORITATIVE','GLOBAL',true),
    (gold,'ICMA_CALENDARS','ICMA Calendar Book','ICMA','REGULATOR',NULL,'AUTHORITATIVE','REGIONAL',true),
    (gold,'INTERNAL_CALENDARS','Internal Calendar Master','INT','INTERNAL',NULL,'AUTHORITATIVE','GLOBAL',true),
    (gold,'MANUAL','Manual Override','MAN','INTERNAL',NULL,'AUTHORITATIVE','GLOBAL',true)
ON CONFLICT (tenant_id, source_cd) DO NOTHING;

-- ── holiday_type ───────────────────────────────────────────────────────
INSERT INTO mdm.holiday_type
    (tenant_id, holiday_type_cd, name, category, is_market_closure,
     is_bank_closure, is_settlement_closure, is_observed_when_weekend, display_order)
VALUES
    (gold,'NEW_YEAR','New Year''s Day','NATIONAL',true,true,true,true,10),
    (gold,'MLK_DAY','Martin Luther King Jr. Day','NATIONAL',true,true,true,true,20),
    (gold,'PRESIDENTS_DAY','Presidents'' Day','NATIONAL',true,true,true,true,30),
    (gold,'GOOD_FRIDAY','Good Friday','RELIGIOUS',true,true,true,false,40),
    (gold,'EASTER_MONDAY','Easter Monday','RELIGIOUS',true,true,true,false,50),
    (gold,'MEMORIAL_DAY','Memorial Day','NATIONAL',true,true,true,true,60),
    (gold,'JUNETEENTH','Juneteenth','NATIONAL',true,true,true,true,70),
    (gold,'INDEPENDENCE_DAY','Independence Day','NATIONAL',true,true,true,true,80),
    (gold,'LABOR_DAY','Labor Day','NATIONAL',true,true,true,true,90),
    (gold,'THANKSGIVING','Thanksgiving','NATIONAL',true,true,true,true,100),
    (gold,'CHRISTMAS','Christmas Day','RELIGIOUS',true,true,true,true,110),
    (gold,'BOXING_DAY','Boxing Day','NATIONAL',true,true,true,true,120),
    (gold,'LUNAR_NEW_YEAR','Chinese New Year','CULTURAL',true,true,true,false,130),
    (gold,'EID_AL_FITR','Eid al-Fitr','RELIGIOUS',true,true,true,false,140),
    (gold,'EID_AL_ADHA','Eid al-Adha','RELIGIOUS',true,true,true,false,150),
    (gold,'ROSH_HASHANAH','Rosh Hashanah','RELIGIOUS',true,true,true,false,160),
    (gold,'YOM_KIPPUR','Yom Kippur','RELIGIOUS',true,true,true,false,170),
    (gold,'DIWALI','Diwali','RELIGIOUS',true,true,true,false,180),
    (gold,'EMERGENCY_CLOSURE','Emergency Closure','EMERGENCY',true,true,true,false,190),
    (gold,'SUBSTITUTE','Substitute Holiday','SUBSTITUTE',true,true,true,false,200)
ON CONFLICT (tenant_id, holiday_type_cd) DO NOTHING;

-- ── holiday_rule_type ──────────────────────────────────────────────────
INSERT INTO mdm.holiday_rule_type
    (tenant_id, rule_type_cd, name, requires_params, requires_manual_dates,
     is_deterministic, description)
VALUES
    (gold,'FIXED_DATE','Fixed Date',true,false,true,'Same calendar date every year'),
    (gold,'NTH_WEEKDAY_OF_MONTH','Nth Weekday of Month',true,false,true,'e.g. 3rd Monday of January'),
    (gold,'LAST_WEEKDAY_OF_MONTH','Last Weekday of Month',true,false,true,'e.g. last Monday of May'),
    (gold,'FIRST_WEEKDAY_OF_MONTH','First Weekday of Month',true,false,true,NULL),
    (gold,'EASTER_BASED','Easter Based',true,false,true,'Good Friday, Easter Monday'),
    (gold,'ORTHODOX_EASTER_BASED','Orthodox Easter Based',true,false,true,NULL),
    (gold,'ISLAMIC_LUNAR','Islamic Lunar',true,true,false,'Eid al-Fitr, Eid al-Adha'),
    (gold,'JEWISH_LUNAR','Jewish Lunar',true,true,false,'Rosh Hashanah, Yom Kippur'),
    (gold,'CHINESE_LUNAR','Chinese Lunar',true,true,false,'Chinese New Year'),
    (gold,'HINDU_LUNAR','Hindu Lunar',true,true,false,'Diwali, Holi'),
    (gold,'BUDDHIST_LUNAR','Buddhist Lunar',true,true,false,'Vesak'),
    (gold,'SEQUENCE','Sequence (relative to anchor)',true,false,true,NULL),
    (gold,'MANUAL','Manual Dates Only',false,true,false,NULL),
    (gold,'COMPOSITE','Composite Rule',true,false,false,NULL)
ON CONFLICT (tenant_id, rule_type_cd) DO NOTHING;

-- ── rolling_convention ─────────────────────────────────────────────────
INSERT INTO mdm.rolling_convention
    (tenant_id, convention_cd, name, description, direction, is_modified)
VALUES
    (gold,'UNADJUSTED','Unadjusted','Do not roll',NULL,false),
    (gold,'FOLLOWING','Following','Roll forward to next business day','FORWARD',false),
    (gold,'MODIFIED_FOLLOWING','Modified Following','Roll forward but stay in month','FORWARD',true),
    (gold,'PRECEDING','Preceding','Roll backward to previous business day','BACKWARD',false),
    (gold,'MODIFIED_PRECEDING','Modified Preceding','Roll backward but stay in month','BACKWARD',true),
    (gold,'END_OF_MONTH','End of Month','Last business day of month','FORWARD',false),
    (gold,'IMM','IMM','3rd Wednesday of delivery month','FORWARD',false),
    (gold,'IMM_MODIFIED_FOLLOWING','IMM Modified Following','IMM with modification','FORWARD',true),
    (gold,'CDS_IMM','CDS IMM','CDS IMM date','FORWARD',false),
    (gold,'TIBOR_END_OF_MONTH','TIBOR End of Month','TIBOR EOM','FORWARD',false)
ON CONFLICT (tenant_id, convention_cd) DO NOTHING;

-- ── business_day_definition ────────────────────────────────────────────
INSERT INTO mdm.business_day_definition
    (tenant_id, definition_cd, name, is_settlement_day, is_trading_day,
     is_payment_day, is_dealing_day, is_valuation_day, is_notification_day, is_reporting_day)
VALUES
    (gold,'TRADING','Trading Day',false,true,false,false,false,false,false),
    (gold,'SETTLEMENT','Settlement Day',true,false,false,false,false,false,false),
    (gold,'PAYMENT','Payment Day',false,false,true,false,false,false,false),
    (gold,'DEALING','Dealing Day',false,false,false,true,false,false,false),
    (gold,'VALUATION','Valuation Day',false,false,false,false,true,false,false),
    (gold,'NOTIFICATION','Notification Day',false,false,false,false,false,true,false),
    (gold,'REPORTING','Reporting Day',false,false,false,false,false,false,true),
    (gold,'CLEARING','Clearing Day',true,false,false,false,false,false,false)
ON CONFLICT (tenant_id, definition_cd) DO NOTHING;

-- ── calendar_hierarchy_type ────────────────────────────────────────────
INSERT INTO mdm.calendar_hierarchy_type
    (tenant_id, hierarchy_type_cd, name, is_inherited)
VALUES
    (gold,'COUNTRY','Country',true),
    (gold,'REGION','Region',true),
    (gold,'EXCHANGE_FAMILY','Exchange Family',true),
    (gold,'EXCHANGE_GROUP','Exchange Group',true),
    (gold,'CURRENCY','Currency',true),
    (gold,'CLEARING_HOUSE','Clearing House',true),
    (gold,'CENTRAL_BANK_FAMILY','Central Bank Family',true),
    (gold,'FUND_FAMILY','Fund Family',true),
    (gold,'INDEX_FAMILY','Index Family',true),
    (gold,'PROVIDER_FAMILY','Provider Family',true)
ON CONFLICT (tenant_id, hierarchy_type_cd) DO NOTHING;

-- ── calendar_regulatory_regime ─────────────────────────────────────────
INSERT INTO mdm.calendar_regulatory_regime
    (tenant_id, regime_cd, name, jurisdiction, regulator_cd,
     regulation_name, requires_specific_calendar, required_calendar_type_cd, effective_from)
VALUES
    (gold,'TARGET2_MANDATORY','TARGET2 Mandatory','EU','ECB','TARGET2 Guideline',
     true,'TARGET2','2007-11-19'),
    (gold,'FEDWIRE_HOURS','Fedwire Operating Hours','US','FEDERAL_RESERVE','Fedwire Operating Circular',
     true,'FEDWIRE','1918-01-01'),
    (gold,'UCITS_DEALING','UCITS Dealing Calendar','EU','ESMA','UCITS Directive',
     true,'MUTUAL_FUND_DEALING','1985-12-20'),
    (gold,'MIFID_II','MiFID II Trading','EU','ESMA','MiFID II',
     false,NULL,'2018-01-03'),
    (gold,'AIFMD','AIFMD Valuation','EU','ESMA','AIFMD',
     true,'MUTUAL_FUND_VALUATION','2013-07-22'),
    (gold,'SOLVENCY_II','Solvency II Reporting','EU','EIOPA','Solvency II Directive',
     false,NULL,'2016-01-01'),
    (gold,'NAIC_SCHEDULE','NAIC Schedule Filing','US','NAIC','NAIC Annual Statement Instructions',
     false,NULL,'1990-01-01'),
    (gold,'SEC_TRADING','SEC Trading','US','SEC','Securities Exchange Act 1934',
     false,NULL,'1934-06-06'),
    (gold,'CFTC_TRADING','CFTC Trading','US','CFTC','Commodity Exchange Act',
     false,NULL,'1936-01-01'),
    (gold,'EMIR_CLEARING','EMIR Clearing','EU','ESMA','EMIR',
     true,'EXCHANGE_CLEARING','2012-08-16'),
    (gold,'BASEL_III_SETTLEMENT','Basel III Settlement','INT','BIS','Basel III Framework',
     true,'EXCHANGE_SETTLEMENT','2013-01-01')
ON CONFLICT (tenant_id, regime_cd) DO NOTHING;

END
$seed$;
