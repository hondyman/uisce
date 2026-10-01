-- 20261106_006_seed_mdm_product_reference.up.sql

DO $seed$
DECLARE gold uuid := '00000000-0000-0000-0000-000000000001'::uuid;
BEGIN

-- ── product_type ───────────────────────────────────────────────────────
INSERT INTO mdm.product_type
    (tenant_id, type_cd, name, category, is_registered, is_security,
     requires_vehicle, requires_security_class, has_share_classes, display_order)
VALUES
    (gold,'MUTUAL_FUND','Mutual Fund','REGISTERED_FUND',true,false,true,true,true,10),
    (gold,'ETF','Exchange Traded Fund','REGISTERED_FUND',true,true,true,true,true,20),
    (gold,'CEF','Closed-End Fund','REGISTERED_FUND',true,true,true,true,true,30),
    (gold,'UNIT_TRUST','Unit Investment Trust','REGISTERED_FUND',true,false,true,false,false,40),
    (gold,'CIT','Collective Investment Trust','UNREGISTERED_FUND',false,false,true,false,false,50),
    (gold,'SMA','Separately Managed Account','MANAGED_ACCOUNT',false,false,false,false,false,60),
    (gold,'UMA','Unified Managed Account','MANAGED_ACCOUNT',false,false,false,false,false,70),
    (gold,'MODEL_PORTFOLIO','Model Portfolio','MANAGED_ACCOUNT',false,false,false,false,true,80),
    (gold,'WRAP_ACCOUNT','Wrap Account','MANAGED_ACCOUNT',false,false,false,false,false,90),
    (gold,'VARIABLE_ANNUITY','Variable Annuity','INSURANCE',true,false,true,false,true,100),
    (gold,'FIXED_ANNUITY','Fixed Annuity','INSURANCE',true,false,true,false,false,110),
    (gold,'INDEXED_ANNUITY','Indexed Annuity','INSURANCE',true,false,true,false,false,120),
    (gold,'VUL','Variable Universal Life','INSURANCE',true,false,true,false,true,130),
    (gold,'GIC','Guaranteed Investment Contract','INSURANCE',false,false,false,false,false,140),
    (gold,'SYNTHETIC_GIC','Synthetic GIC','INSURANCE',false,false,false,false,false,150),
    (gold,'FUNDING_AGREEMENT','Funding Agreement','INSURANCE',false,false,false,false,false,160),
    (gold,'STABLE_VALUE_FUND','Stable Value Fund','INSURANCE',false,false,false,false,false,170),
    (gold,'STRUCTURED_NOTE','Structured Note','STRUCTURED',true,true,false,false,false,180),
    (gold,'MARKET_LINKED_CD','Market-Linked CD','STRUCTURED',true,true,false,false,false,190),
    (gold,'PRINCIPAL_PROTECTED','Principal Protected Note','STRUCTURED',true,true,false,false,false,200),
    (gold,'HEDGE_FUND','Hedge Fund','ALTERNATIVE',false,false,true,false,true,210),
    (gold,'PE_FUND','Private Equity Fund','ALTERNATIVE',false,false,true,false,true,220),
    (gold,'VC_FUND','Venture Capital Fund','ALTERNATIVE',false,false,true,false,true,230),
    (gold,'REAL_ESTATE_FUND','Real Estate Fund','ALTERNATIVE',false,false,true,false,true,240),
    (gold,'INFRASTRUCTURE_FUND','Infrastructure Fund','ALTERNATIVE',false,false,true,false,true,250),
    (gold,'MONEY_MARKET_FUND','Money Market Fund','REGISTERED_FUND',true,false,true,false,true,260),
    (gold,'TARGET_DATE_FUND','Target Date Fund','REGISTERED_FUND',true,false,true,true,true,270),
    (gold,'SEPARATE_ACCOUNT','Separate Account','INSURANCE',false,false,false,false,false,280),
    (gold,'INSURANCE_SEPARATE_ACCOUNT','Insurance Separate Account','INSURANCE',false,false,false,false,false,290),
    (gold,'ADVISORY_MANDATE','Advisory Mandate','ADVISORY',false,false,false,false,false,300)
ON CONFLICT (tenant_id, type_cd) DO NOTHING;

-- ── product_sub_type ───────────────────────────────────────────────────
INSERT INTO mdm.product_sub_type
    (tenant_id, sub_type_cd, name, parent_type_cd)
VALUES
    (gold,'US_MUTUAL_FUND','US Mutual Fund','MUTUAL_FUND'),
    (gold,'UCITS','UCITS','MUTUAL_FUND'),
    (gold,'AIF','Alternative Investment Fund','MUTUAL_FUND'),
    (gold,'SICAV','SICAV','MUTUAL_FUND'),
    (gold,'OEIC','OEIC','MUTUAL_FUND'),
    (gold,'UNIT_TRUST_UK','UK Unit Trust','MUTUAL_FUND'),
    (gold,'INDEX_ETF','Index ETF','ETF'),
    (gold,'ACTIVE_ETF','Active ETF','ETF'),
    (gold,'LEVERAGED_ETF','Leveraged ETF','ETF'),
    (gold,'INVERSE_ETF','Inverse ETF','ETF'),
    (gold,'COMMODITY_ETF','Commodity ETF','ETF'),
    (gold,'CURRENCY_ETF','Currency ETF','ETF'),
    (gold,'SINGLE_STOCK_ETF','Single-Stock ETF','ETF'),
    (gold,'DISCIPLINARY_SMA','Discretionary SMA','SMA'),
    (gold,'NON_DISCIPLINARY_SMA','Non-Discretionary SMA','SMA'),
    (gold,'WRAP_SMA','Wrap SMA','SMA'),
    (gold,'VARIABLE_ANNUITY_VA','Variable Annuity','VARIABLE_ANNUITY'),
    (gold,'REGISTERED_INDEX_LINKED','Registered Index-Linked','INDEXED_ANNUITY'),
    (gold,'TRADITIONAL_GIC','Traditional GIC','GIC'),
    (gold,'FHLB_FUNDING_AGREEMENT','FHLB Funding Agreement','FUNDING_AGREEMENT'),
    (gold,'BOOK_VALUE_WRAP','Book-Value Wrap','SYNTHETIC_GIC')
ON CONFLICT (tenant_id, sub_type_cd) DO NOTHING;

-- ── product_status ─────────────────────────────────────────────────────
INSERT INTO mdm.product_status
    (tenant_id, status_cd, name, is_live, is_open_to_new, is_terminal, display_order)
VALUES
    (gold,'CONCEPT','Concept',false,false,false,10),
    (gold,'DEVELOPMENT','Development',false,false,false,20),
    (gold,'PILOT','Pilot',true,true,false,30),
    (gold,'REGISTERED','Registered',true,true,false,40),
    (gold,'LIVE','Live',true,true,false,50),
    (gold,'CLOSED_TO_NEW','Closed to New Investors',true,false,false,60),
    (gold,'SOFT_CLOSED','Soft Closed',true,false,false,70),
    (gold,'HARD_CLOSED','Hard Closed',true,false,false,80),
    (gold,'SUSPENDED','Suspended',false,false,false,90),
    (gold,'MERGED','Merged',false,false,true,100),
    (gold,'LIQUIDATED','Liquidated',false,false,true,110),
    (gold,'WITHDRAWN','Withdrawn',false,false,true,120)
ON CONFLICT (tenant_id, status_cd) DO NOTHING;

-- ── product_distribution_channel ───────────────────────────────────────
INSERT INTO mdm.product_distribution_channel
    (tenant_id, channel_cd, name, channel_type)
VALUES
    (gold,'DIRECT','Direct to Investor','DIRECT'),
    (gold,'RIA','Registered Investment Advisors','RIA'),
    (gold,'BROKER_DEALER','Broker-Dealers','BROKER_DEALER'),
    (gold,'BANK','Banks','BANK'),
    (gold,'INSURANCE','Insurance Companies','INSURANCE'),
    (gold,'WRAP_PLATFORM','Wrap Platforms','WRAP'),
    (gold,'RIA_PLATFORM','RIA Platforms','PLATFORM'),
    (gold,'SUPERMARKET','Fund Supermarkets','PLATFORM'),
    (gold,'INSTITUTIONAL_DIRECT','Institutional Direct','INSTITUTIONAL'),
    (gold,'PENSION','Pension Plans','INSTITUTIONAL'),
    (gold,'RETIREMENT_PLATFORM','Retirement Platforms','RETIREMENT'),
    (gold,'INTERMEDIARY','Intermediaries','INTERMEDIARY')
ON CONFLICT (tenant_id, channel_cd) DO NOTHING;

-- ── product_registration_type ──────────────────────────────────────────
INSERT INTO mdm.product_registration_type
    (tenant_id, reg_type_cd, name)
VALUES
    (gold,'REGISTERED','Registered'),
    (gold,'AUTHORISED','Authorised'),
    (gold,'RECOGNISED','Recognised'),
    (gold,'NOTIFIED','Notified'),
    (gold,'PASSIVE_MARKETING','Passive Marketing'),
    (gold,'EXEMPT','Exempt'),
    (gold,'PRIVATE_PLACEMENT','Private Placement')
ON CONFLICT (tenant_id, reg_type_cd) DO NOTHING;

-- ── product_document_type ──────────────────────────────────────────────
INSERT INTO mdm.product_document_type
    (tenant_id, doc_type_cd, name, category, is_mandatory, is_public)
VALUES
    (gold,'PROSPECTUS','Prospectus','REGULATORY',true,true),
    (gold,'SUMMARY_PROSPECTUS','Summary Prospectus','REGULATORY',true,true),
    (gold,'SAI','Statement of Additional Information','REGULATORY',true,true),
    (gold,'KID','Key Information Document','REGULATORY',true,true),
    (gold,'KIID','Key Investor Information Document','REGULATORY',true,true),
    (gold,'FACTSHEET','Factsheet','MARKETING',false,true),
    (gold,'SEMI_ANNUAL_REPORT','Semi-Annual Report','REGULATORY',true,true),
    (gold,'ANNUAL_REPORT','Annual Report','REGULATORY',true,true),
    (gold,'FINANCIAL_STATEMENTS','Financial Statements','REGULATORY',true,true),
    (gold,'PRIVACY_NOTICE','Privacy Notice','COMPLIANCE',true,true),
    (gold,'DISCLOSURE','Disclosure','COMPLIANCE',true,true),
    (gold,'METHODOLOGY','Methodology','LEGAL',false,true),
    (gold,'REGISTRATION_STATEMENT','Registration Statement','REGULATORY',true,false),
    (gold,'ESG_DISCLOSURE','ESG Disclosure','COMPLIANCE',false,true),
    (gold,'SFDR_ANNEX','SFDR Annex','COMPLIANCE',true,true),
    (gold,'PROXY_POLICY','Proxy Voting Policy','COMPLIANCE',true,true),
    (gold,'BROCHURE','Brochure','MARKETING',false,true),
    (gold,'FORM_ADV','Form ADV','REGULATORY',true,true)
ON CONFLICT (tenant_id, doc_type_cd) DO NOTHING;

-- ── product_share_class_type ───────────────────────────────────────────
INSERT INTO mdm.product_share_class_type
    (tenant_id, class_type_cd, name, has_distribution, has_load)
VALUES
    (gold,'INSTITUTIONAL','Institutional Class',false,false),
    (gold,'ADVISOR','Advisor Class',true,false),
    (gold,'RETAIL','Retail Class',true,true),
    (gold,'A_CLASS','A Class (Front Load)',true,true),
    (gold,'B_CLASS','B Class (Back Load)',true,true),
    (gold,'C_CLASS','C Class (Level Load)',true,true),
    (gold,'I_CLASS','I Class (Institutional)',false,false),
    (gold,'R_CLASS','R Class (Retirement)',false,false),
    (gold,'Z_CLASS','Z Class (No Load)',false,false),
    (gold,'ETF_CLASS','ETF Class',false,false),
    (gold,'ADMIRAL','Admiral Class',false,false),
    (gold,'INVESTOR','Investor Class',true,false)
ON CONFLICT (tenant_id, class_type_cd) DO NOTHING;

-- ── product_target_market_type ─────────────────────────────────────────
INSERT INTO mdm.product_target_market_type
    (tenant_id, tm_type_cd, name, is_positive)
VALUES
    (gold,'RETAIL','Retail Investors',true),
    (gold,'MASS_AFFLUENT','Mass Affluent',true),
    (gold,'HNWI','High Net Worth',true),
    (gold,'UHNWI','Ultra High Net Worth',true),
    (gold,'INSTITUTIONAL','Institutional',true),
    (gold,'SOVEREIGN','Sovereign',true),
    (gold,'PENSION','Pension Plans',true),
    (gold,'INSURANCE','Insurance',true),
    (gold,'EXECUTION_ONLY','Execution Only',false),
    (gold,'NO_ADVICE','No Advice',false),
    (gold,'NO_DISTRIBUTION','No Distribution',false)
ON CONFLICT (tenant_id, tm_type_cd) DO NOTHING;

-- ── product_lifecycle_event_type ───────────────────────────────────────
INSERT INTO mdm.product_lifecycle_event_type
    (tenant_id, event_cd, name, is_final, requires_approval, display_order)
VALUES
    (gold,'CONCEPT','Concept',false,false,10),
    (gold,'DEVELOPMENT_START','Development Start',false,true,20),
    (gold,'PILOT_START','Pilot Start',false,true,30),
    (gold,'REGISTRATION_FILED','Registration Filed',false,true,40),
    (gold,'REGISTRATION_APPROVED','Registration Approved',false,true,50),
    (gold,'LAUNCH','Launch',false,false,60),
    (gold,'SOFT_CLOSE','Soft Close',false,true,70),
    (gold,'HARD_CLOSE','Hard Close',false,true,80),
    (gold,'REOPEN','Reopen',false,true,90),
    (gold,'SUSPENSION','Suspension',false,true,100),
    (gold,'RESUMPTION','Resumption',false,true,110),
    (gold,'MERGER_IN','Merger In',true,true,120),
    (gold,'MERGER_OUT','Merger Out',true,true,130),
    (gold,'LIQUIDATION_ANNOUNCED','Liquidation Announced',false,true,140),
    (gold,'LIQUIDATION_COMPLETE','Liquidation Complete',true,true,150),
    (gold,'WITHDRAWAL','Withdrawal',true,true,160)
ON CONFLICT (tenant_id, event_cd) DO NOTHING;

END
$seed$;
