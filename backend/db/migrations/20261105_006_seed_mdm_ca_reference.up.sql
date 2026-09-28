-- 20261105_006_seed_mdm_ca_reference.up.sql
-- Seeds CA reference tables under the gold tenant. Idempotent.

DO $seed$
DECLARE gold uuid := '00000000-0000-0000-0000-000000000001'::uuid;
BEGIN

-- ── ca_event_type ──────────────────────────────────────────────────────
INSERT INTO mdm.ca_event_type
    (tenant_id, event_type_cd, name, category, is_income_event, is_principal_event,
     is_mandatory, is_voluntary, requires_election, requires_proxy_vote,
     affects_cost_basis, affects_nav, affects_tax_lots, display_order)
VALUES
    (gold,'CASH_DIVIDEND','Cash Dividend','INCOME',true,false,true,false,false,false,false,true,true,10),
    (gold,'SPECIAL_DIVIDEND','Special Dividend','INCOME',true,false,true,false,false,false,false,true,true,20),
    (gold,'INTERIM_DIVIDEND','Interim Dividend','INCOME',true,false,true,false,false,false,false,true,true,30),
    (gold,'FINAL_DIVIDEND','Final Dividend','INCOME',true,false,true,false,false,false,false,true,true,40),
    (gold,'SUPPLEMENTAL_DIVIDEND','Supplemental Dividend','INCOME',true,false,true,false,false,false,false,true,true,50),
    (gold,'DIVIDEND_REINVESTMENT','Dividend Reinvestment','INCOME',true,false,false,true,true,false,false,true,true,60),
    (gold,'STOCK_DIVIDEND','Stock Dividend','EQUITY',false,false,true,false,false,false,true,true,true,70),
    (gold,'STOCK_SPLIT','Stock Split','EQUITY',false,false,true,false,false,false,true,true,true,80),
    (gold,'REVERSE_SPLIT','Reverse Split','EQUITY',false,false,true,false,false,false,true,true,true,90),
    (gold,'BONUS_ISSUE','Bonus Issue','EQUITY',false,false,true,false,false,false,true,true,true,100),
    (gold,'SCRIP_DIVIDEND','Scrip Dividend','EQUITY',false,false,false,true,true,false,true,true,true,110),
    (gold,'CAPITALIZATION_ISSUE','Capitalization Issue','EQUITY',false,false,true,false,false,false,true,true,true,120),
    (gold,'MERGER','Merger','MANDATORY',false,true,true,false,false,false,true,true,true,130),
    (gold,'ACQUISITION','Acquisition','MANDATORY',false,true,true,false,false,false,true,true,true,140),
    (gold,'SPIN_OFF','Spin-Off','MANDATORY',false,true,true,false,false,false,true,true,true,150),
    (gold,'SPLIT_OFF','Split-Off','MANDATORY',false,true,true,false,false,false,true,true,true,160),
    (gold,'CARVE_OUT','Carve-Out','MANDATORY',false,true,true,false,false,false,true,true,true,170),
    (gold,'TAKEOVER','Takeover','VOLUNTARY',false,true,false,true,true,false,true,true,true,180),
    (gold,'DELISTING','Delisting','MANDATORY',false,true,true,false,false,false,false,true,false,190),
    (gold,'NAME_CHANGE','Name Change','MANDATORY',false,false,true,false,false,false,false,false,false,200),
    (gold,'TICKER_CHANGE','Ticker Change','MANDATORY',false,false,true,false,false,false,false,false,false,210),
    (gold,'CUSIP_CHANGE','CUSIP Change','MANDATORY',false,false,true,false,false,false,false,false,false,220),
    (gold,'ISIN_CHANGE','ISIN Change','MANDATORY',false,false,true,false,false,false,false,false,false,230),
    (gold,'RIGHTS_ISSUE','Rights Issue','VOLUNTARY',false,true,false,true,true,false,true,true,true,240),
    (gold,'RIGHTS_SUBSCRIPTION','Rights Subscription','VOLUNTARY',false,true,false,true,true,false,true,true,true,250),
    (gold,'WARRANTS_ISSUE','Warrants Issue','VOLUNTARY',false,false,false,true,true,false,true,true,true,260),
    (gold,'WARRANT_EXERCISE','Warrant Exercise','VOLUNTARY',false,true,false,true,true,false,true,true,true,270),
    (gold,'TENDER_OFFER','Tender Offer','VOLUNTARY',false,true,false,true,true,false,true,true,true,280),
    (gold,'DUTCH_AUCTION','Dutch Auction','VOLUNTARY',false,true,false,true,true,false,true,true,true,290),
    (gold,'EXCHANGE_OFFER','Exchange Offer','VOLUNTARY',false,true,false,true,true,false,true,true,true,300),
    (gold,'BUYBACK','Buyback','VOLUNTARY',false,true,false,true,true,false,true,true,true,310),
    (gold,'SELF_TENDER','Self-Tender','VOLUNTARY',false,true,false,true,true,false,true,true,true,320),
    (gold,'COUPON_PAYMENT','Coupon Payment','FIXED_INCOME',true,false,true,false,false,false,false,true,true,330),
    (gold,'PRINCIPAL_PAYMENT','Principal Payment','FIXED_INCOME',false,true,true,false,false,false,false,true,true,340),
    (gold,'BOND_CALL','Bond Call','FIXED_INCOME',false,true,true,false,false,false,false,true,true,350),
    (gold,'BOND_PUT','Bond Put','VOLUNTARY',false,true,false,true,true,false,false,true,true,360),
    (gold,'SINKING_FUND','Sinking Fund','FIXED_INCOME',false,true,true,false,false,false,false,true,true,370),
    (gold,'AMORTIZATION','Amortization','FIXED_INCOME',false,true,true,false,false,false,false,true,true,380),
    (gold,'MATURITY','Maturity','FIXED_INCOME',false,true,true,false,false,false,false,true,true,390),
    (gold,'CONVERSION','Conversion','VOLUNTARY',false,true,false,true,true,false,true,true,true,400),
    (gold,'DEFAULT','Default','FIXED_INCOME',false,false,true,false,false,false,false,true,false,410),
    (gold,'CONSENT_SOLICITATION','Consent Solicitation','VOLUNTARY',false,false,false,true,true,false,false,false,false,420),
    (gold,'FUND_DISTRIBUTION_INCOME','Fund Distribution - Income','FUND',true,false,true,false,false,false,false,true,true,430),
    (gold,'FUND_DISTRIBUTION_CAPITAL_GAIN','Fund Distribution - Capital Gain','FUND',true,false,true,false,false,false,false,true,true,440),
    (gold,'FUND_DISTRIBUTION_ROC','Fund Distribution - ROC','FUND',true,false,true,false,false,false,false,true,true,450),
    (gold,'FUND_SPLIT','Fund Split','FUND',false,false,true,false,false,false,true,true,true,460),
    (gold,'FUND_MERGER','Fund Merger','FUND',false,true,true,false,false,false,true,true,true,470),
    (gold,'FUND_REORGANIZATION','Fund Reorganization','FUND',false,true,true,false,false,false,true,true,true,480),
    (gold,'FUND_LIQUIDATION','Fund Liquidation','FUND',false,true,true,false,false,false,true,true,true,490),
    (gold,'FUND_RECLASSIFICATION','Fund Reclassification','FUND',false,false,true,false,false,false,false,false,false,500),
    (gold,'NAV_REVISION','NAV Revision','FUND',false,false,true,false,false,false,false,true,false,510),
    (gold,'RETURN_OF_CAPITAL','Return of Capital','TAX',true,false,true,false,false,false,true,true,true,520),
    (gold,'WITHHOLDING_TAX','Withholding Tax','TAX',false,false,true,false,false,false,false,true,true,530),
    (gold,'FOREIGN_TAX_CREDIT','Foreign Tax Credit','TAX',false,false,true,false,false,false,false,false,true,540),
    (gold,'ANNUAL_MEETING','Annual Meeting','PROXY',false,false,true,false,false,true,false,false,false,550),
    (gold,'SPECIAL_MEETING','Special Meeting','PROXY',false,false,false,true,false,true,false,false,false,560),
    (gold,'PROXY_CONTEST','Proxy Contest','PROXY',false,false,false,true,false,true,false,false,false,570),
    (gold,'CLASS_ACTION','Class Action','CLASS_ACTION',false,false,false,true,false,false,false,false,false,580),
    (gold,'BANKRUPTCY','Bankruptcy','LEGAL',false,false,true,false,false,false,false,true,false,590),
    (gold,'RECEIVERSHIP','Receivership','LEGAL',false,false,true,false,false,false,false,true,false,600),
    (gold,'LIQUIDATION','Liquidation','LEGAL',false,true,true,false,false,false,true,true,true,610),
    (gold,'REGULATORY_NOTIFICATION','Regulatory Notification','REGULATORY',false,false,true,false,false,false,false,false,false,620),
    (gold,'RESTRICTED_SECURITY','Restricted Security','REGULATORY',false,false,true,false,false,false,false,false,false,630),
    (gold,'INSIDER_TRANSACTION','Insider Transaction','REGULATORY',false,false,true,false,false,false,false,false,false,640)
ON CONFLICT (tenant_id, event_type_cd) DO NOTHING;

-- ── ca_status ──────────────────────────────────────────────────────────
INSERT INTO mdm.ca_status
    (tenant_id, status_cd, name, status_category, is_terminal, requires_processing, display_order)
VALUES
    (gold,'ANNOUNCED','Announced','ANNOUNCED',false,false,10),
    (gold,'TENTATIVE','Tentative','ANNOUNCED',false,false,20),
    (gold,'PENDING_APPROVAL','Pending Approval','ANNOUNCED',false,false,30),
    (gold,'APPROVED','Approved','ACTIVE',false,false,40),
    (gold,'EFFECTIVE','Effective','ACTIVE',false,true,50),
    (gold,'EX_DATE_PASSED','Ex-Date Passed','ACTIVE',false,true,60),
    (gold,'RECORD_DATE_PASSED','Record Date Passed','ACTIVE',false,true,70),
    (gold,'PAYMENT_PENDING','Payment Pending','ACTIVE',false,true,80),
    (gold,'PAID','Paid','COMPLETED',true,false,90),
    (gold,'COMPLETED','Completed','COMPLETED',true,false,100),
    (gold,'CANCELLED','Cancelled','CANCELLED',true,false,110),
    (gold,'WITHDRAWN','Withdrawn','CANCELLED',true,false,120),
    (gold,'SUPERSEDED','Superseded','SUPERSEDED',true,false,130),
    (gold,'AMENDED','Amended','ACTIVE',false,false,140),
    (gold,'FAILED','Failed','FAILED',true,false,150),
    (gold,'LAPSED','Lapsed','FAILED',true,false,160),
    (gold,'EXPIRED','Expired','FAILED',true,false,170),
    (gold,'DEFAULTED','Defaulted','FAILED',true,true,180)
ON CONFLICT (tenant_id, status_cd) DO NOTHING;

-- ── ca_election_type ───────────────────────────────────────────────────
INSERT INTO mdm.ca_election_type
    (tenant_id, election_type_cd, name, requires_instruction, has_default, proration_possible)
VALUES
    (gold,'CASH','Cash',true,false,false),
    (gold,'STOCK','Stock',true,false,true),
    (gold,'MIXED','Mixed Cash and Stock',true,false,true),
    (gold,'CASH_OR_STOCK','Cash or Stock',true,false,true),
    (gold,'ALL_STOCK','All Stock',true,false,true),
    (gold,'ALL_CASH','All Cash',true,false,false),
    (gold,'REINVEST','Reinvest',true,true,false),
    (gold,'DO_NOT_REINVEST','Do Not Reinvest',true,true,false),
    (gold,'PARTIAL_CASH','Partial Cash',true,false,false),
    (gold,'PARTIAL_STOCK','Partial Stock',true,false,true),
    (gold,'TENDER_ALL','Tender All',true,false,false),
    (gold,'TENDER_PARTIAL','Tender Partial',true,false,false),
    (gold,'TENDER_NONE','Tender None',true,false,false),
    (gold,'EXERCISE_ALL','Exercise All',true,false,false),
    (gold,'EXERCISE_PARTIAL','Exercise Partial',true,false,false),
    (gold,'EXERCISE_NONE','Exercise None',true,false,false),
    (gold,'ABSTAIN','Abstain',true,false,false),
    (gold,'VOTE_FOR','Vote For',true,false,false),
    (gold,'VOTE_AGAINST','Vote Against',true,false,false),
    (gold,'VOTE_WITHHELD','Vote Withheld',true,false,false)
ON CONFLICT (tenant_id, election_type_cd) DO NOTHING;

-- ── ca_payment_type ────────────────────────────────────────────────────
INSERT INTO mdm.ca_payment_type
    (tenant_id, payment_type_cd, name, payment_method, is_income, is_principal, is_interest, is_dividend)
VALUES
    (gold,'CASH','Cash','CASH',false,false,false,false),
    (gold,'STOCK','Stock','STOCK',false,false,false,false),
    (gold,'BOND','Bond','BOND',false,false,false,false),
    (gold,'MIXED','Mixed','MIXED',false,false,false,false),
    (gold,'IN_KIND','In Kind','IN_KIND',false,false,false,false),
    (gold,'DIVIDEND','Dividend','CASH',true,false,false,true),
    (gold,'INTEREST','Interest','CASH',true,false,true,false),
    (gold,'PRINCIPAL','Principal','CASH',false,true,false,false),
    (gold,'PREMIUM','Premium','CASH',true,false,false,false),
    (gold,'RETURN_OF_CAPITAL','Return of Capital','CASH',true,false,false,false),
    (gold,'CAPITAL_GAIN','Capital Gain Distribution','CASH',true,false,false,true),
    (gold,'WITHHOLDING_TAX','Withholding Tax','CASH',false,false,false,false),
    (gold,'FOREIGN_TAX','Foreign Tax','CASH',false,false,false,false)
ON CONFLICT (tenant_id, payment_type_cd) DO NOTHING;

-- ── ca_tax_treatment ───────────────────────────────────────────────────
INSERT INTO mdm.ca_tax_treatment
    (tenant_id, treatment_cd, name, is_ordinary_income, is_qualified_dividend,
     is_capital_gain, is_return_of_capital, is_tax_free, is_deferred,
     affects_cost_basis, withholding_required)
VALUES
    (gold,'ORDINARY_INCOME','Ordinary Income',true,false,false,false,false,false,false,false),
    (gold,'QUALIFIED_DIVIDEND','Qualified Dividend',false,true,false,false,false,false,false,false),
    (gold,'SHORT_TERM_CAPITAL_GAIN','Short-Term Capital Gain',false,false,true,false,false,false,false,false),
    (gold,'LONG_TERM_CAPITAL_GAIN','Long-Term Capital Gain',false,false,true,false,false,false,false,false),
    (gold,'RETURN_OF_CAPITAL','Return of Capital',false,false,false,true,false,false,true,false),
    (gold,'TAX_FREE','Tax Free',false,false,false,false,true,false,false,false),
    (gold,'TAX_DEFERRED','Tax Deferred',false,false,false,false,false,true,false,false),
    (gold,'SECTION_1250_GAIN','Section 1250 Gain',false,false,true,false,false,false,false,false),
    (gold,'SECTION_1231_GAIN','Section 1231 Gain',false,false,true,false,false,false,false,false),
    (gold,'UNRECAPTURED_1250','Unrecaptured Section 1250',true,false,false,false,false,false,false,false),
    (gold,'FOREIGN_SOURCE','Foreign Source Income',true,false,false,false,false,false,false,true),
    (gold,'US_SOURCE','US Source Income',true,false,false,false,false,false,false,false),
    (gold,'PFIC_INCOME','PFIC Income',true,false,false,false,false,false,false,false),
    (gold,'CFC_SUBPART_F','CFC Subpart F Income',true,false,false,false,false,false,false,false),
    (gold,'TREATY_RATE','Treaty Rate Withholding',true,false,false,false,false,false,false,true),
    (gold,'NON_TREATY_RATE','Non-Treaty Withholding',true,false,false,false,false,false,false,true)
ON CONFLICT (tenant_id, treatment_cd) DO NOTHING;

-- ── ca_regulatory_regime ───────────────────────────────────────────────
INSERT INTO mdm.ca_regulatory_regime
    (tenant_id, regime_cd, name, jurisdiction, regulator_cd,
     regulation_name, requires_notification, requires_filing,
     requires_shareholder_approval, notification_deadline_days, effective_from)
VALUES
    (gold,'SEC_13D','SEC Schedule 13D','US','SEC','Securities Exchange Act 1934 s.13(d)',true,true,false,10,'1968-07-29'),
    (gold,'SEC_13G','SEC Schedule 13G','US','SEC','Securities Exchange Act 1934 s.13(g)',true,true,false,45,'1978-01-01'),
    (gold,'SEC_13F','SEC Form 13F','US','SEC','Securities Exchange Act 1934 s.13(f)',true,true,false,45,'1975-01-01'),
    (gold,'SEC_16','SEC Section 16','US','SEC','Securities Exchange Act 1934 s.16',true,true,false,2,'1934-06-06'),
    (gold,'HSR_ACT','Hart-Scott-Rodino','US','FTC','HSR Act 1976',true,true,false,30,'1978-01-01'),
    (gold,'EU_MARKET_ABUSE','EU Market Abuse Regulation','EU','ESMA','Regulation 596/2014',true,true,false,1,'2016-07-03'),
    (gold,'UK_TAKEOVER_CODE','UK Takeover Code','GB','TAKEOVER_PANEL','City Code on Takeovers',true,true,true,7,'1968-01-01'),
    (gold,'IRL_TAKEOVER_RULES','Irish Takeover Rules','IE','IRISH_TAKEOVER_PANEL','Irish Takeover Rules',true,true,true,7,'1997-01-01'),
    (gold,'HK_TAKEOVER_CODE','Hong Kong Takeover Code','HK','SFC','Code on Takeovers and Mergers',true,true,true,7,'1975-01-01'),
    (gold,'JP_LARGE_SHAREHOLDING','Japan Large Shareholding','JP','JFSA','FIEA Article 27',true,true,false,5,'1990-01-01'),
    (gold,'DE_BAFIN_WPUE','German BaFin WpÜG','DE','BAFIN','Wertpapiererwerbs- und Übernahmegesetz',true,true,true,7,'2002-01-01'),
    (gold,'FR_AMF_DECLARATION','French AMF Declaration','FR','AMF','RG AMF Article 231',true,true,false,5,'2006-01-01'),
    (gold,'CA_EARLY_WARNING','Canada Early Warning','CA','CSA','NI 62-103',true,true,false,2,'2000-01-01'),
    (gold,'AU_SUBSTANTIAL_HOLDING','Australia Substantial Holding','AU','ASIC','Corporations Act Chapter 6C',true,true,false,2,'2000-01-01')
ON CONFLICT (tenant_id, regime_cd) DO NOTHING;

-- ── ca_mandatory_type ──────────────────────────────────────────────────
INSERT INTO mdm.ca_mandatory_type
    (tenant_id, mandatory_type_cd, name, is_mandatory, is_voluntary,
     is_mandatory_with_options, requires_election, requires_response_by_deadline, default_action)
VALUES
    (gold,'MANDATORY','Mandatory',true,false,false,false,false,NULL),
    (gold,'VOLUNTARY','Voluntary',false,true,false,true,true,'DECLINE'),
    (gold,'MANDATORY_WITH_OPTIONS','Mandatory with Options',false,false,true,true,true,'DEFAULT'),
    (gold,'VOLUNTARY_WITH_DEFAULT','Voluntary with Default',false,true,false,true,true,'DEFAULT'),
    (gold,'MANDATORY_WITH_ELECTION','Mandatory with Election',true,false,false,true,true,'DEFAULT')
ON CONFLICT (tenant_id, mandatory_type_cd) DO NOTHING;

-- ── ca_entitlement_basis ───────────────────────────────────────────────
INSERT INTO mdm.ca_entitlement_basis
    (tenant_id, basis_cd, name, valuation_date_basis, requires_position_snapshot)
VALUES
    (gold,'EX_DATE_POSITION','Ex-Date Position','EX_DATE',true),
    (gold,'RECORD_DATE_POSITION','Record Date Position','RECORD_DATE',true),
    (gold,'TRADE_DATE_POSITION','Trade Date Position','TRADE_DATE',true),
    (gold,'SETTLEMENT_DATE_POSITION','Settlement Date Position','TRADE_DATE',true),
    (gold,'END_OF_DAY_POSITION','End of Day Position','RECORD_DATE',true),
    (gold,'START_OF_DAY_POSITION','Start of Day Position','RECORD_DATE',true),
    (gold,'AVERAGE_POSITION','Average Position','RECORD_DATE',true)
ON CONFLICT (tenant_id, basis_cd) DO NOTHING;

END
$seed$;
