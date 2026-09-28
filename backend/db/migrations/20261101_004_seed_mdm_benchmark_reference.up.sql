-- 20261101_004_seed_mdm_benchmark_reference.up.sql
-- Seeds benchmark reference tables under the shared-reference tenant.
-- Idempotent. Total ~85 rows.

DO $seed$
DECLARE gold uuid := '00000000-0000-0000-0000-000000000001'::uuid;
BEGIN

-- ── benchmark_provider ─────────────────────────────────────────────────
INSERT INTO mdm.benchmark_provider
    (tenant_id, provider_cd, name, short_name, provider_type, domicile,
     is_administrator, administrator_status, is_iosco_compliant,
     is_esma_registered, is_uk_fca_registered, is_sec_registered, is_active)
VALUES
    (gold,'MSCI','MSCI Inc.','MSCI','INDEX_PROVIDER','US',true,'AUTHORISED',true,true,true,true,true),
    (gold,'FTSE_RUSSELL','FTSE Russell','FTSE','INDEX_PROVIDER','GB',true,'AUTHORISED',true,true,true,true,true),
    (gold,'SP_DJI','S&P Dow Jones Indices','S&P DJI','INDEX_PROVIDER','US',true,'AUTHORISED',true,true,true,true,true),
    (gold,'BLOOMBERG','Bloomberg L.P.','Bloomberg','INDEX_PROVIDER','US',true,'AUTHORISED',true,true,true,true,true),
    (gold,'ICE','Intercontinental Exchange','ICE','INDEX_PROVIDER','US',true,'AUTHORISED',true,true,true,true,true),
    (gold,'REFINITIV','Refinitiv','Refinitiv','INDEX_PROVIDER','GB',true,'AUTHORISED',true,true,true,true,true),
    (gold,'SOLACTIVE','Solactive AG','Solactive','INDEX_PROVIDER','DE',true,'AUTHORISED',true,true,true,true,true),
    (gold,'STOXX','STOXX Ltd.','STOXX','INDEX_PROVIDER','CH',true,'AUTHORISED',true,true,true,true,true),
    (gold,'WILSHIRE','Wilshire Associates','Wilshire','INDEX_PROVIDER','US',true,'AUTHORISED',true,true,true,true,true),
    (gold,'CRSP','Center for Research in Security Prices','CRSP','INDEX_PROVIDER','US',true,'AUTHORISED',true,true,true,true,true),
    (gold,'NASDAQ','Nasdaq Inc.','Nasdaq','INDEX_PROVIDER','US',true,'AUTHORISED',true,true,true,true,true),
    (gold,'MORNINGSTAR','Morningstar Inc.','Morningstar','INDEX_PROVIDER','US',true,'AUTHORISED',true,true,true,true,true),
    (gold,'EVESTMENT','eVestment Alliance','eVestment','PEER_GROUP_PROVIDER','US',false,NULL,false,false,false,false,true),
    (gold,'LIPPER','Lipper (LSEG)','Lipper','PEER_GROUP_PROVIDER','GB',false,NULL,false,false,false,false,true),
    (gold,'INTERNAL','Internal Composite','INTERNAL','INTERNAL','US',true,'EXEMPT',true,false,false,false,true)
ON CONFLICT (tenant_id, provider_cd) DO NOTHING;

-- ── benchmark_type ─────────────────────────────────────────────────────
INSERT INTO mdm.benchmark_type
    (tenant_id, type_cd, name, category, is_investable, is_publicly_available,
     requires_methodology, requires_regulatory_registration, display_order)
VALUES
    (gold,'MARKET_INDEX','Market Index','MARKET_INDEX',true,true,true,true,10),
    (gold,'COMPOSITE','Composite Benchmark','COMPOSITE',false,false,true,true,20),
    (gold,'PEER_GROUP','Peer Group','PEER_GROUP',false,true,true,false,30),
    (gold,'POLICY_BENCHMARK','Policy Benchmark','POLICY',false,false,true,false,40),
    (gold,'CUSTOM','Custom Benchmark','CUSTOM',false,false,true,false,50),
    (gold,'REGULATORY','Regulatory Benchmark','REGULATORY',false,false,true,true,60),
    (gold,'STRATEGY','Strategy Benchmark','STRATEGY',false,false,true,false,70),
    (gold,'HYBRID','Hybrid Benchmark','HYBRID',false,false,true,false,80),
    (gold,'CASH','Cash Benchmark','CASH',true,true,false,false,90),
    (gold,'ZERO','Zero Benchmark','ZERO',false,false,false,false,100)
ON CONFLICT (tenant_id, type_cd) DO NOTHING;

-- ── benchmark_return_variant ───────────────────────────────────────────
INSERT INTO mdm.benchmark_return_variant
    (tenant_id, variant_cd, name, variant_type, dividend_treatment,
     withholding_tax_applied, withholding_tax_basis, is_default)
VALUES
    (gold,'PRICE_RETURN','Price Return','PRICE_RETURN','EXCLUDED',false,NULL,false),
    (gold,'GROSS_TOTAL_RETURN','Gross Total Return','GROSS_TOTAL_RETURN','REINVESTED_GROSS',false,NULL,false),
    (gold,'NET_TOTAL_RETURN','Net Total Return','NET_TOTAL_RETURN','REINVESTED_NET',true,'MAX_RATE',true),
    (gold,'EXCESS_RETURN','Excess Return','EXCESS_RETURN','REINVESTED_NET',true,'MAX_RATE',false),
    (gold,'TOTAL_RETURN','Total Return (Unspecified)','TOTAL_RETURN','REINVESTED_GROSS',false,NULL,false)
ON CONFLICT (tenant_id, variant_cd) DO NOTHING;

-- ── benchmark_currency_variant ─────────────────────────────────────────
INSERT INTO mdm.benchmark_currency_variant
    (tenant_id, variant_cd, name, base_currency, is_hedged, hedge_currency,
     hedging_frequency, hedging_method, hedge_ratio_pct, is_default)
VALUES
    (gold,'USD_UNHEDGED','USD Unhedged','USD',false,NULL,NULL,NULL,NULL,true),
    (gold,'USD_HEDGED','USD Hedged','USD',true,'USD','MONTHLY','FORWARD_CONTRACT',100.00,false),
    (gold,'EUR_UNHEDGED','EUR Unhedged','EUR',false,NULL,NULL,NULL,NULL,true),
    (gold,'EUR_HEDGED','EUR Hedged','EUR',true,'EUR','MONTHLY','FORWARD_CONTRACT',100.00,false),
    (gold,'GBP_UNHEDGED','GBP Unhedged','GBP',false,NULL,NULL,NULL,NULL,true),
    (gold,'GBP_HEDGED','GBP Hedged','GBP',true,'GBP','MONTHLY','FORWARD_CONTRACT',100.00,false),
    (gold,'JPY_UNHEDGED','JPY Unhedged','JPY',false,NULL,NULL,NULL,NULL,true),
    (gold,'CHF_UNHEDGED','CHF Unhedged','CHF',false,NULL,NULL,NULL,NULL,true),
    (gold,'AUD_UNHEDGED','AUD Unhedged','AUD',false,NULL,NULL,NULL,NULL,true),
    (gold,'CAD_UNHEDGED','CAD Unhedged','CAD',false,NULL,NULL,NULL,NULL,true)
ON CONFLICT (tenant_id, variant_cd) DO NOTHING;

-- ── benchmark_weighting_method ─────────────────────────────────────────
INSERT INTO mdm.benchmark_weighting_method
    (tenant_id, method_cd, name, requires_rebalance)
VALUES
    (gold,'MARKET_CAP_FLOAT_ADJ','Market Cap (Float-Adjusted)',true),
    (gold,'MARKET_CAP_FULL','Market Cap (Full)',true),
    (gold,'EQUAL_WEIGHTED','Equal Weighted',true),
    (gold,'PRICE_WEIGHTED','Price Weighted',true),
    (gold,'FUNDAMENTAL_WEIGHTED','Fundamental Weighted',true),
    (gold,'GDP_WEIGHTED','GDP Weighted',true),
    (gold,'REVENUE_WEIGHTED','Revenue Weighted',true),
    (gold,'DIVIDEND_WEIGHTED','Dividend Weighted',true),
    (gold,'VOLATILITY_WEIGHTED','Volatility Weighted',true),
    (gold,'RISK_PARITY','Risk Parity',true),
    (gold,'MIN_VARIANCE','Minimum Variance',true),
    (gold,'MAX_SHARPE','Maximum Sharpe',true),
    (gold,'FACTOR_TILTED','Factor Tilted',true),
    (gold,'ESG_TILTED','ESG Tilted',true),
    (gold,'CUSTOM','Custom Methodology',true)
ON CONFLICT (tenant_id, method_cd) DO NOTHING;

-- ── benchmark_rebalance_frequency ──────────────────────────────────────
INSERT INTO mdm.benchmark_rebalance_frequency
    (tenant_id, frequency_cd, name, periods_per_year)
VALUES
    (gold,'DAILY','Daily',365.00),
    (gold,'WEEKLY','Weekly',52.00),
    (gold,'MONTHLY','Monthly',12.00),
    (gold,'QUARTERLY','Quarterly',4.00),
    (gold,'SEMI_ANNUAL','Semi-Annual',2.00),
    (gold,'ANNUAL','Annual',1.00),
    (gold,'EVENT_DRIVEN','Event-Driven',NULL),
    (gold,'AD_HOC','Ad Hoc',NULL),
    (gold,'NONE','No Rebalance',NULL)
ON CONFLICT (tenant_id, frequency_cd) DO NOTHING;

-- ── benchmark_regulatory_regime ────────────────────────────────────────
INSERT INTO mdm.benchmark_regulatory_regime
    (tenant_id, regime_cd, name, jurisdiction, regulator_cd, regulation_name,
     effective_from, requires_registration, requires_authorisation, requires_esg_disclosure)
VALUES
    (gold,'EU_BMR','EU Benchmarks Regulation','EU','ESMA','Regulation (EU) 2016/1011',
     '2018-01-01',true,true,false),
    (gold,'UK_BMR','UK Benchmarks Regulation','GB','FCA','UK Benchmarks Regulation 2021',
     '2021-01-01',true,true,false),
    (gold,'IOSCO_PRINCIPLES','IOSCO Principles for Financial Benchmarks','INT','IOSCO','IOSCO Principles',
     '2013-07-01',false,false,false),
    (gold,'SEC_40_ACT','US Investment Company Act 1940','US','SEC','40-Act',
     '1940-01-01',false,false,false),
    (gold,'UCITS_ESG','UCITS ESG Disclosure','EU','ESMA','UCITS Directive 2009/65/EC + ESG',
     '2021-03-10',false,false,true),
    (gold,'AIFMD','Alternative Investment Fund Managers Directive','EU','ESMA','AIFMD 2011/61/EU',
     '2013-07-22',false,false,false),
    (gold,'SFDR_PAI','SFDR Principal Adverse Impact','EU','ESMA','SFDR 2019/2088',
     '2021-03-10',false,false,true),
    (gold,'CFTC','US Commodity Futures Trading Commission','US','CFTC','CFTC Regulations',
     '1936-01-01',false,false,false),
    (gold,'MAS_BMR','Singapore Benchmarks Regulation','SG','MAS','MAS Notices on Benchmark',
     '2018-01-01',true,true,false),
    (gold,'ASIC_BMR','Australia Benchmarks Regulation','AU','ASIC','ASIC Benchmark Rules',
     '2018-04-01',true,true,false)
ON CONFLICT (tenant_id, regime_cd) DO NOTHING;

-- ── benchmark_use_case ─────────────────────────────────────────────────
INSERT INTO mdm.benchmark_use_case
    (tenant_id, use_case_cd, name, is_regulated)
VALUES
    (gold,'PERFORMANCE_MEASUREMENT','Performance Measurement',false),
    (gold,'ATTRIBUTION','Attribution Analysis',false),
    (gold,'FEE_CALCULATION','Fee Calculation',true),
    (gold,'MANDATE_COMPLIANCE','Mandate Compliance',true),
    (gold,'RISK_MONITORING','Risk Monitoring',false),
    (gold,'CAPITAL_ALLOCATION','Capital Allocation',false),
    (gold,'REGULATORY_REPORTING','Regulatory Reporting',true),
    (gold,'PRODUCT_DISCLOSURE','Product Disclosure',true),
    (gold,'SALES_COMPARISON','Sales Comparison',false),
    (gold,'IMA_DEFINITION','Investment Management Agreement Definition',true),
    (gold,'PORTFOLIO_CONSTRUCTION','Portfolio Construction',false)
ON CONFLICT (tenant_id, use_case_cd) DO NOTHING;

END
$seed$;
