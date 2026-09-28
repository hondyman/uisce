-- 20261030_005_seed_mdm_counterparty_reference.up.sql
-- Seeds counterparty_type, counterparty_status, counterparty_role, agreement_type
-- under the gold-copy tenant so they're visible to all real tenants.
-- Idempotent.

DO $seed$
DECLARE gold uuid := '00000000-0000-0000-0000-000000000001'::uuid;
BEGIN

-- ── counterparty_type ──────────────────────────────────────────────────
INSERT INTO mdm.counterparty_type
    (tenant_id, type_cd, name, category, requires_isda, requires_credit_limit,
     requires_due_diligence, requires_settlement_instructions, display_order)
VALUES
    (gold,'BANK','Bank','FINANCIAL_INSTITUTION',true,true,true,true,10),
    (gold,'INVESTMENT_BANK','Investment Bank','FINANCIAL_INSTITUTION',true,true,true,true,20),
    (gold,'CUSTODIAN','Custodian','FINANCIAL_INSTITUTION',false,true,true,true,30),
    (gold,'PRIME_BROKER','Prime Broker','FINANCIAL_INSTITUTION',true,true,true,true,40),
    (gold,'CLEARING_BROKER','Clearing Broker','FINANCIAL_INSTITUTION',true,true,true,true,50),
    (gold,'EXECUTING_BROKER','Executing Broker','FINANCIAL_INSTITUTION',false,true,true,true,60),
    (gold,'INTRODUCING_BROKER','Introducing Broker','FINANCIAL_INSTITUTION',false,true,true,false,70),
    (gold,'BROKER_DEALER','Broker-Dealer','FINANCIAL_INSTITUTION',true,true,true,true,80),
    (gold,'MARKET_MAKER','Market Maker','FINANCIAL_INSTITUTION',false,true,true,false,90),
    (gold,'LIQUIDITY_PROVIDER','Liquidity Provider','FINANCIAL_INSTITUTION',false,true,true,false,100),
    (gold,'CCP','Central Counterparty','INFRASTRUCTURE',false,true,true,true,110),
    (gold,'EXCHANGE','Exchange','INFRASTRUCTURE',false,false,true,false,120),
    (gold,'TRADING_VENUE','Trading Venue','INFRASTRUCTURE',false,false,true,false,130),
    (gold,'ASSET_MANAGER','Asset Manager','SERVICE_PROVIDER',false,false,true,false,140),
    (gold,'FUND_ADMINISTRATOR','Fund Administrator','SERVICE_PROVIDER',false,false,true,false,150),
    (gold,'TRANSFER_AGENT','Transfer Agent','SERVICE_PROVIDER',false,false,true,false,160),
    (gold,'DISTRIBUTOR','Distributor','SERVICE_PROVIDER',false,false,true,false,170),
    (gold,'PAYING_AGENT','Paying Agent','SERVICE_PROVIDER',false,false,true,false,180),
    (gold,'CORRESPONDENT_BANK','Correspondent Bank','FINANCIAL_INSTITUTION',false,true,true,true,190),
    (gold,'CASH_MANAGER','Cash Manager','SERVICE_PROVIDER',false,false,true,false,200),
    (gold,'AUDITOR','Auditor','SERVICE_PROVIDER',false,false,true,false,210),
    (gold,'LEGAL_COUNSEL','Legal Counsel','SERVICE_PROVIDER',false,false,true,false,220),
    (gold,'TAX_ADVISOR','Tax Advisor','SERVICE_PROVIDER',false,false,true,false,230),
    (gold,'CONSULTANT','Consultant','SERVICE_PROVIDER',false,false,true,false,240),
    (gold,'INSURANCE_COMPANY','Insurance Company','FINANCIAL_INSTITUTION',false,true,true,false,250),
    (gold,'REINSURER','Reinsurer','FINANCIAL_INSTITUTION',false,true,true,false,260),
    (gold,'DATA_VENDOR','Data Vendor','SERVICE_PROVIDER',false,false,false,false,270),
    (gold,'MARKET_DATA_VENDOR','Market Data Vendor','SERVICE_PROVIDER',false,false,false,false,280),
    (gold,'INDEX_PROVIDER','Index Provider','SERVICE_PROVIDER',false,false,false,false,290),
    (gold,'RATING_AGENCY','Rating Agency','SERVICE_PROVIDER',false,false,false,false,300),
    (gold,'REGULATORY','Regulatory Body','GOVERNMENT',false,false,false,false,310),
    (gold,'GOVERNMENT','Government','GOVERNMENT',false,false,false,false,320),
    (gold,'CORRESPONDENT','Correspondent','INTERMEDIARY',false,false,true,false,330),
    (gold,'INVESTOR','Investor','COMMERCIAL',false,false,true,false,340),
    (gold,'ISSUER','Issuer','COMMERCIAL',false,false,true,false,350),
    (gold,'SPONSOR','Sponsor','COMMERCIAL',false,false,true,false,360),
    (gold,'ARRANGER','Arranger','FINANCIAL_INSTITUTION',false,true,true,false,370)
ON CONFLICT (tenant_id, type_cd) DO NOTHING;

-- ── counterparty_status ────────────────────────────────────────────────
INSERT INTO mdm.counterparty_status
    (tenant_id, status_cd, name, is_active_status, is_terminal, allows_trading,
     allows_settlement, requires_remediation, display_order)
VALUES
    (gold,'PROSPECT','Prospect',true,false,false,false,false,10),
    (gold,'DUE_DILIGENCE','Due Diligence',true,false,false,false,false,20),
    (gold,'ONBOARDING','Onboarding',true,false,false,false,false,30),
    (gold,'APPROVED','Approved',true,false,true,true,false,40),
    (gold,'ACTIVE','Active',true,false,true,true,false,50),
    (gold,'RESTRICTED','Restricted',true,false,false,true,true,60),
    (gold,'SUSPENDED','Suspended',false,false,false,false,true,70),
    (gold,'UNDER_REVIEW','Under Review',true,false,false,true,true,80),
    (gold,'CREDIT_WATCH','Credit Watch',true,false,true,true,true,90),
    (gold,'DEFAULT','Default',false,true,false,false,true,100),
    (gold,'TERMINATED','Terminated',false,true,false,false,false,110),
    (gold,'OFFBOARDED','Offboarded',false,true,false,false,false,120),
    (gold,'BLACKLISTED','Blacklisted',false,true,false,false,true,130),
    (gold,'DECEASED','Deceased',false,true,false,false,false,140)
ON CONFLICT (tenant_id, status_cd) DO NOTHING;

-- ── counterparty_role ──────────────────────────────────────────────────
INSERT INTO mdm.counterparty_role
    (tenant_id, role_cd, name, role_category, requires_agreement,
     requires_credit_limit, requires_collateral, display_order)
VALUES
    (gold,'TRADING_COUNTERPARTY','Trading Counterparty','TRADING',true,true,true,10),
    (gold,'DERIVATIVE_COUNTERPARTY','Derivative Counterparty','TRADING',true,true,true,20),
    (gold,'REPO_COUNTERPARTY','Repo Counterparty','TRADING',true,true,true,30),
    (gold,'SEC_LENDING_COUNTERPARTY','Securities Lending Counterparty','TRADING',true,true,true,40),
    (gold,'FX_COUNTERPARTY','FX Counterparty','TRADING',true,true,false,50),
    (gold,'CLEARING_MEMBER','Clearing Member','CLEARING',true,true,true,60),
    (gold,'GIVE_UP_BROKER','Give-Up Broker','CLEARING',true,true,false,70),
    (gold,'SETTLEMENT_AGENT','Settlement Agent','SETTLEMENT',true,true,false,80),
    (gold,'CASH_CORRESPONDENT','Cash Correspondent','SETTLEMENT',true,true,false,90),
    (gold,'CUSTODIAN_ROLE','Custodian','CUSTODY',true,true,false,100),
    (gold,'SUB_CUSTODIAN','Sub-Custodian','CUSTODY',true,true,false,110),
    (gold,'PRIME_BROKER_ROLE','Prime Broker','CUSTODY',true,true,true,120),
    (gold,'FUND_ADMIN','Fund Administrator','ADMINISTRATION',true,false,false,130),
    (gold,'TRANSFER_AGENT_ROLE','Transfer Agent','ADMINISTRATION',true,false,false,140),
    (gold,'FUND_ACCOUNTANT','Fund Accountant','ADMINISTRATION',true,false,false,150),
    (gold,'REPO_FINANCING','Repo Financing','FINANCING',true,true,true,160),
    (gold,'MARGIN_LENDER','Margin Lender','FINANCING',true,true,true,170),
    (gold,'CREDIT_FACILITY','Credit Facility Provider','FINANCING',true,true,true,180),
    (gold,'INVESTMENT_ADVISOR','Investment Advisor','ADVISORY',true,false,false,190),
    (gold,'SUB_ADVISOR','Sub-Advisor','ADVISORY',true,false,false,200),
    (gold,'LEGAL_COUNSEL_ROLE','Legal Counsel','ADVISORY',true,false,false,210),
    (gold,'TAX_ADVISOR_ROLE','Tax Advisor','ADVISORY',true,false,false,220),
    (gold,'AUDITOR_ROLE','Auditor','ADVISORY',true,false,false,230),
    (gold,'MARKET_DATA','Market Data Provider','DATA',false,false,false,240),
    (gold,'PRICING_VENDOR','Pricing Vendor','DATA',false,false,false,250),
    (gold,'INDEX_VENDOR','Index Provider','DATA',false,false,false,260),
    (gold,'RATING_AGENCY_ROLE','Rating Agency','DATA',false,false,false,270),
    (gold,'REGULATOR','Regulator','REGULATORY',false,false,false,280),
    (gold,'DISTRIBUTOR_ROLE','Distributor','DISTRIBUTION',true,false,false,290),
    (gold,'PLATFORM','Platform','DISTRIBUTION',true,false,false,300)
ON CONFLICT (tenant_id, role_cd) DO NOTHING;

-- ── agreement_type ─────────────────────────────────────────────────────
INSERT INTO mdm.agreement_type
    (tenant_id, agreement_type_cd, name, category, is_master_agreement,
     requires_netting, requires_collateral, standard_template, regulatory_regime)
VALUES
    (gold,'ISDA_MASTER','ISDA Master Agreement','DERIVATIVE',true,true,false,'ISDA_2002','GLOBAL'),
    (gold,'ISDA_1992','ISDA 1992 Master Agreement','DERIVATIVE',true,true,false,'ISDA_1992','GLOBAL'),
    (gold,'ISDA_CSA','ISDA Credit Support Annex','DERIVATIVE',false,false,true,'ISDA_CSA','GLOBAL'),
    (gold,'ISDA_SCHEDULE','ISDA Schedule','DERIVATIVE',false,false,false,NULL,'GLOBAL'),
    (gold,'GMRA','Global Master Repurchase Agreement','REPO',true,true,true,'GMRA_2011','GLOBAL'),
    (gold,'GMSLA','Global Master Securities Lending Agreement','SECURITIES_LENDING',true,true,true,'GMSLA_2010','GLOBAL'),
    (gold,'MSFTA','Master Securities Forward Transaction Agreement','SECURITIES_LENDING',true,true,true,'MSFTA_2013','US'),
    (gold,'MSLA','Master Securities Lending Agreement','SECURITIES_LENDING',true,true,true,NULL,'GLOBAL'),
    (gold,'CLEARING_AGREEMENT','Clearing Agreement','CLEARING',true,true,true,NULL,'GLOBAL'),
    (gold,'FUTURES_ACCOUNT_AGREEMENT','Futures Account Agreement','CLEARING',true,true,true,NULL,'US'),
    (gold,'PRIME_BROKERAGE_AGREEMENT','Prime Brokerage Agreement','CUSTODY',true,true,true,NULL,'GLOBAL'),
    (gold,'CUSTODY_AGREEMENT','Custody Agreement','CUSTODY',true,false,false,NULL,'GLOBAL'),
    (gold,'ADMIN_AGREEMENT','Administration Agreement','SERVICE',true,false,false,NULL,'GLOBAL'),
    (gold,'TRANSFER_AGENCY_AGREEMENT','Transfer Agency Agreement','SERVICE',true,false,false,NULL,'GLOBAL'),
    (gold,'DISTRIBUTION_AGREEMENT','Distribution Agreement','SERVICE',true,false,false,NULL,'GLOBAL'),
    (gold,'IMA','Investment Management Agreement','SERVICE',true,false,false,NULL,'GLOBAL'),
    (gold,'SUB_ADVISORY_AGREEMENT','Sub-Advisory Agreement','SERVICE',true,false,false,NULL,'GLOBAL'),
    (gold,'MARKET_DATA_LICENSE','Market Data License','DATA',false,false,false,NULL,'GLOBAL'),
    (gold,'INDEX_LICENSE','Index License','DATA',false,false,false,NULL,'GLOBAL'),
    (gold,'SERVICES_AGREEMENT','Services Agreement','SERVICE',true,false,false,NULL,'GLOBAL'),
    (gold,'NDA','Non-Disclosure Agreement','OTHER',false,false,false,NULL,'GLOBAL'),
    (gold,'MSA','Master Services Agreement','SERVICE',true,false,false,NULL,'GLOBAL'),
    (gold,'SOW','Statement of Work','SERVICE',false,false,false,NULL,'GLOBAL'),
    (gold,'REFERRAL_AGREEMENT','Referral Agreement','OTHER',false,false,false,NULL,'GLOBAL')
ON CONFLICT (tenant_id, agreement_type_cd) DO NOTHING;

END
$seed$;
