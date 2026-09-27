-- Seeds the 12 mdm party reference tables with standard institutional/retail/HNWI values.
-- Idempotent — uses ON CONFLICT DO NOTHING on the (tenant_id, type_cd) unique key.
-- Seeded for the gold-copy tenant (00000000-0000-0000-0000-000000000001) so it
-- is visible to all tenants via the shared-reference RLS pattern.
-- Each tenant can supplement via direct INSERT after onboarding.

-- ════════════════════════════════════════════════════════════════════════
-- party_type
-- ════════════════════════════════════════════════════════════════════════
INSERT INTO mdm.party_type (type_cd, name, category, is_individual, is_legal_entity, is_trust, is_government, is_fund, is_spv, requires_lei, requires_kyc, requires_tax_residency, requires_ubo_lookup, display_order, tenant_id) VALUES
  ('INDIVIDUAL',         'Individual',                   'PERSON',  true,  false, false, false, false, false, false, true,  true,  false, 10, '00000000-0000-0000-0000-000000000001'),
  ('JOINT_ACCOUNT',      'Joint Account Holders',        'PERSON',  true,  false, false, false, false, false, false, true,  true,  false, 11, '00000000-0000-0000-0000-000000000001'),
  ('TRUST',              'Trust',                        'VEHICLE', false, true,  true,  false, false, false, false, true,  true,  true,  20, '00000000-0000-0000-0000-000000000001'),
  ('ESTATE',             'Estate',                       'VEHICLE', false, true,  true,  false, false, false, false, true,  true,  true,  21, '00000000-0000-0000-0000-000000000001'),
  ('CORPORATION',        'Corporation',                  'ENTITY',  false, true,  false, false, false, false, true,  true,  true,  true,  30, '00000000-0000-0000-0000-000000000001'),
  ('LLC',                'Limited Liability Company',    'ENTITY',  false, true,  false, false, false, false, false, true,  true,  true,  31, '00000000-0000-0000-0000-000000000001'),
  ('PARTNERSHIP',        'Partnership',                  'ENTITY',  false, true,  false, false, false, false, false, true,  true,  true,  32, '00000000-0000-0000-0000-000000000001'),
  ('LP',                 'Limited Partnership',          'ENTITY',  false, true,  false, false, false, false, false, true,  true,  true,  33, '00000000-0000-0000-0000-000000000001'),
  ('LLP',                'Limited Liability Partnership','ENTITY',  false, true,  false, false, false, false, false, true,  true,  true,  34, '00000000-0000-0000-0000-000000000001'),
  ('FUND',               'Investment Fund',              'FUND',    false, true,  false, false, true,  false, true,  true,  true,  true,  40, '00000000-0000-0000-0000-000000000001'),
  ('ETF',                'Exchange-Traded Fund',         'FUND',    false, true,  false, false, true,  false, false, true,  false, false, 41, '00000000-0000-0000-0000-000000000001'),
  ('MUTUAL_FUND',        'Mutual Fund',                  'FUND',    false, true,  false, false, true,  false, true,  true,  false, false, 42, '00000000-0000-0000-0000-000000000001'),
  ('HEDGE_FUND',         'Hedge Fund',                   'FUND',    false, true,  false, false, true,  false, true,  true,  true,  true,  43, '00000000-0000-0000-0000-000000000001'),
  ('PENSION_FUND',       'Pension Fund',                 'FUND',    false, true,  false, false, true,  false, true,  true,  true,  true,  44, '00000000-0000-0000-0000-000000000001'),
  ('SOVEREIGN',          'Sovereign State',             'GOVT',    false, true,  false, true,  false, false, false, true,  false, false, 50, '00000000-0000-0000-0000-000000000001'),
  ('GOVERNMENT_AGENCY',  'Government Agency',            'GOVT',    false, true,  false, true,  false, false, false, true,  false, false, 51, '00000000-0000-0000-0000-000000000001'),
  ('CENTRAL_BANK',       'Central Bank',                 'GOVT',    false, true,  false, true,  false, false, false, true,  false, false, 52, '00000000-0000-0000-0000-000000000001'),
  ('INSURANCE_COMPANY',  'Insurance Company',            'FIN_INST',false, true,  false, false, false, false, true,  true,  true,  true,  60, '00000000-0000-0000-0000-000000000001'),
  ('BANK',               'Bank',                         'FIN_INST',false, true,  false, false, false, false, true,  true,  true,  true,  61, '00000000-0000-0000-0000-000000000001'),
  ('BROKER_DEALER',      'Broker-Dealer',                'FIN_INST',false, true,  false, false, false, false, true,  true,  true,  true,  62, '00000000-0000-0000-0000-000000000001'),
  ('CUSTODIAN',          'Custodian',                    'FIN_INST',false, true,  false, false, false, false, true,  true,  true,  true,  63, '00000000-0000-0000-0000-000000000001'),
  ('ASSET_MANAGER',      'Asset Manager',                'FIN_INST',false, true,  false, false, false, false, true,  true,  true,  true,  64, '00000000-0000-0000-0000-000000000001'),
  ('SPV',                'Special Purpose Vehicle',      'VEHICLE', false, true,  false, false, false, true,  false, true,  true,  true,  70, '00000000-0000-0000-0000-000000000001'),
  ('CHARITY',            'Charity / Non-Profit',         'VEHICLE', false, true,  false, false, false, false, false, true,  true,  true,  71, '00000000-0000-0000-0000-000000000001'),
  ('FOUNDATION',         'Foundation',                   'VEHICLE', false, true,  false, false, false, false, false, true,  true,  true,  72, '00000000-0000-0000-0000-000000000001'),
  ('CLUB',               'Club / Membership Body',       'VEHICLE', false, true,  false, false, false, false, false, true,  true,  false, 73, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, type_cd) DO NOTHING;

-- ════════════════════════════════════════════════════════════════════════
-- party_sub_type
-- ════════════════════════════════════════════════════════════════════════
INSERT INTO mdm.party_sub_type (sub_type_cd, name, parent_type_cd, description, tenant_id) VALUES
  ('RETAIL',          'Retail',         'INDIVIDUAL', 'Standard retail client',                            '00000000-0000-0000-0000-000000000001'),
  ('HNWI',            'High Net Worth', 'INDIVIDUAL', 'High net worth individual',                         '00000000-0000-0000-0000-000000000001'),
  ('UHNWI',           'Ultra HNWI',     'INDIVIDUAL', 'Ultra high net worth individual',                   '00000000-0000-0000-0000-000000000001'),
  ('VHNWI',           'Very HNWI',      'INDIVIDUAL', 'Very high net worth individual ($5M-$30M)',         '00000000-0000-0000-0000-000000000001'),
  ('MASS_AFFLUENT',   'Mass Affluent',  'INDIVIDUAL', 'Mass affluent retail segment',                       '00000000-0000-0000-0000-000000000001'),
  ('FAMILY_OFFICE',   'Family Office',  'INDIVIDUAL', 'Family office principal',                           '00000000-0000-0000-0000-000000000001'),
  ('PROFESSIONAL',    'Professional',   'INDIVIDUAL', 'Professional / corporate executive',                 '00000000-0000-0000-0000-000000000001'),
  ('INSTITUTIONAL',   'Institutional',  'CORPORATION','Institutional corporate client',                    '00000000-0000-0000-0000-000000000001'),
  ('SOVEREIGN_WEALTH','Sovereign Wealth','CORPORATION','Sovereign wealth fund',                            '00000000-0000-0000-0000-000000000001'),
  ('ENDOWMENT',       'Endowment',      'CORPORATION','University / institutional endowment',              '00000000-0000-0000-0000-000000000001'),
  ('FAMILY_INVESTMENT','Family Investment','LLC',    'Single-family investment vehicle',                  '00000000-0000-0000-0000-000000000001'),
  ('MANAGEMENT_CO',   'Management Co',  'LLC',        'Management company / GP',                            '00000000-0000-0000-0000-000000000001'),
  ('FUND_SPV',        'Fund SPV',       'SPV',        'Special purpose vehicle for a single fund',          '00000000-0000-0000-0000-000000000001'),
  ('BLIND_TRUST',     'Blind Trust',    'TRUST',      'Blind trust (UBO not disclosed)',                    '00000000-0000-0000-0000-000000000001'),
  ('REVOCABLE_TRUST', 'Revocable Trust','TRUST',      'Revocable living trust',                             '00000000-0000-0000-0000-000000000001'),
  ('IRREVOCABLE_TRUST','Irrevocable Trust','TRUST',   'Irrevocable trust',                                  '00000000-0000-0000-0000-000000000001'),
  ('CHARITABLE_TRUST','Charitable Trust','TRUST',     'Charitable remainder trust',                         '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, sub_type_cd) DO NOTHING;

-- ════════════════════════════════════════════════════════════════════════
-- party_status
-- ════════════════════════════════════════════════════════════════════════
INSERT INTO mdm.party_status (status_cd, name, is_active_status, is_terminal, requires_kyc_review, is_blocked, display_order, tenant_id) VALUES
  ('PROSPECT',     'Prospect',           true,  false, false, false, 10, '00000000-0000-0000-0000-000000000001'),
  ('ONBOARDING',   'Onboarding',         true,  false, true,  false, 20, '00000000-0000-0000-0000-000000000001'),
  ('ACTIVE',       'Active',             true,  false, false, false, 30, '00000000-0000-0000-0000-000000000001'),
  ('DORMANT',      'Dormant',            false, false, true,  false, 40, '00000000-0000-0000-0000-000000000001'),
  ('INACTIVE',     'Inactive',           false, false, true,  false, 50, '00000000-0000-0000-0000-000000000001'),
  ('SUSPENDED',    'Suspended',          false, false, true,  true,  60, '00000000-0000-0000-0000-000000000001'),
  ('CLOSED',       'Closed',             false, true,  false, false, 70, '00000000-0000-0000-0000-000000000001'),
  ('DISSOLVED',    'Dissolved',          false, true,  false, false, 71, '00000000-0000-0000-0000-000000000001'),
  ('DECEASED',     'Deceased',           false, true,  false, false, 72, '00000000-0000-0000-0000-000000000001'),
  ('MERGED',       'Merged',             false, true,  false, false, 73, '00000000-0000-0000-0000-000000000001'),
  ('ACQUIRED',     'Acquired',           false, true,  false, false, 74, '00000000-0000-0000-0000-000000000001'),
  ('IN_LIQUIDATION','In Liquidation',    false, false, true,  true,  80, '00000000-0000-0000-0000-000000000001'),
  ('IN_RECEIVERSHIP','In Receivership',   false, false, true,  true,  81, '00000000-0000-0000-0000-000000000001'),
  ('IN_ADMINISTRATION','In Administration',false,false, true,  true,  82, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, status_cd) DO NOTHING;

-- ════════════════════════════════════════════════════════════════════════
-- party_segment
-- ════════════════════════════════════════════════════════════════════════
INSERT INTO mdm.party_segment (segment_cd, name, segment_type, min_aum_usd, max_aum_usd, display_order, tenant_id) VALUES
  ('RETAIL',         'Retail',           'RETAIL',         0,         100000,      10, '00000000-0000-0000-0000-000000000001'),
  ('MASS_AFFLUENT',  'Mass Affluent',    'RETAIL',         100000,    1000000,     20, '00000000-0000-0000-0000-000000000001'),
  ('AFFLUENT',       'Affluent',         'RETAIL',         1000000,   5000000,     30, '00000000-0000-0000-0000-000000000001'),
  ('HNWI',           'HNWI',             'HNWI',           5000000,   30000000,    40, '00000000-0000-0000-0000-000000000001'),
  ('VHNWI',          'VHNWI',            'HNWI',           30000000,  100000000,   50, '00000000-0000-0000-0000-000000000001'),
  ('UHNWI',          'UHNWI',            'UHNWI',          100000000, 500000000,   60, '00000000-0000-0000-0000-000000000001'),
  ('FAMILY_OFFICE',  'Family Office',    'FAMILY_OFFICE',  50000000,  NULL,        70, '00000000-0000-0000-0000-000000000001'),
  ('INSTITUTIONAL',  'Institutional',    'INSTITUTIONAL',  10000000,  NULL,        80, '00000000-0000-0000-0000-000000000001'),
  ('SOVEREIGN',      'Sovereign',        'SOVEREIGN',      1000000000,NULL,        90, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, segment_cd) DO NOTHING;

-- ════════════════════════════════════════════════════════════════════════
-- party_role
-- ════════════════════════════════════════════════════════════════════════
INSERT INTO mdm.party_role (role_cd, name, role_category, applies_to_entity_type, requires_kyc, display_order, tenant_id) VALUES
  ('CLIENT',                'Client',                  'RELATIONSHIP', NULL,        true,  10, '00000000-0000-0000-0000-000000000001'),
  ('PROSPECT',              'Prospect',                'RELATIONSHIP', NULL,        false, 20, '00000000-0000-0000-0000-000000000001'),
  ('FORMER_CLIENT',         'Former Client',           'RELATIONSHIP', NULL,        false, 30, '00000000-0000-0000-0000-000000000001'),
  ('EMPLOYEE',              'Employee',                'INTERNAL',     'CORPORATION',false, 40, '00000000-0000-0000-0000-000000000001'),
  ('DIRECTOR',              'Director',                'GOVERNANCE',   'CORPORATION',true,  50, '00000000-0000-0000-0000-000000000001'),
  ('OFFICER',               'Officer',                 'GOVERNANCE',   'CORPORATION',true,  51, '00000000-0000-0000-0000-000000000001'),
  ('SHAREHOLDER',           'Shareholder',             'OWNERSHIP',    'CORPORATION',false, 60, '00000000-0000-0000-0000-000000000001'),
  ('UBO',                   'Ultimate Beneficial Owner','OWNERSHIP',   NULL,        true,  70, '00000000-0000-0000-0000-000000000001'),
  ('BENEFICIAL_OWNER',      'Beneficial Owner',        'OWNERSHIP',    NULL,        true,  71, '00000000-0000-0000-0000-000000000001'),
  ('TRUSTEE',               'Trustee',                 'TRUST',        'TRUST',     true,  80, '00000000-0000-0000-0000-000000000001'),
  ('SETTLOR',               'Settlor',                 'TRUST',        'TRUST',     true,  81, '00000000-0000-0000-0000-000000000001'),
  ('PROTECTOR',             'Protector',               'TRUST',        'TRUST',     true,  82, '00000000-0000-0000-0000-000000000001'),
  ('BENEFICIARY',           'Beneficiary',             'TRUST',        'TRUST',     false, 83, '00000000-0000-0000-0000-000000000001'),
  ('GUARDIAN',              'Guardian',                'PERSONAL',     'INDIVIDUAL',true,  90, '00000000-0000-0000-0000-000000000001'),
  ('POWER_OF_ATTORNEY',     'Power of Attorney',       'PERSONAL',     'INDIVIDUAL',true,  91, '00000000-0000-0000-0000-000000000001'),
  ('AUTHORIZED_SIGNATORY',  'Authorized Signatory',    'GOVERNANCE',   'CORPORATION',true, 100, '00000000-0000-0000-0000-000000000001'),
  ('PRIMARY_CONTACT',       'Primary Contact',         'RELATIONSHIP', NULL,        false,110, '00000000-0000-0000-0000-000000000001'),
  ('COUNTERPARTY',          'Counterparty',            'RELATIONSHIP', NULL,        true, 120, '00000000-0000-0000-0000-000000000001'),
  ('VENDOR',                'Vendor',                  'RELATIONSHIP', NULL,        false,130, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, role_cd) DO NOTHING;

-- ════════════════════════════════════════════════════════════════════════
-- party_relationship_type
-- ════════════════════════════════════════════════════════════════════════
INSERT INTO mdm.party_relationship_type (relationship_cd, name, category, is_directional, inverse_relationship_cd, tenant_id) VALUES
  ('SPOUSE',              'Spouse',                'FAMILY',    false, NULL,                       '00000000-0000-0000-0000-000000000001'),
  ('PARENT_OF',           'Parent of',             'FAMILY',    true,  'CHILD_OF',                 '00000000-0000-0000-0000-000000000001'),
  ('CHILD_OF',            'Child of',              'FAMILY',    true,  'PARENT_OF',                '00000000-0000-0000-0000-000000000001'),
  ('SIBLING_OF',          'Sibling of',            'FAMILY',    false, NULL,                       '00000000-0000-0000-0000-000000000001'),
  ('EMPLOYER_OF',         'Employer of',           'EMPLOYMENT',true,  'EMPLOYEE_OF',              '00000000-0000-0000-0000-000000000001'),
  ('EMPLOYEE_OF',         'Employee of',           'EMPLOYMENT',true,  'EMPLOYER_OF',              '00000000-0000-0000-0000-000000000001'),
  ('TRUSTEE_OF',          'Trustee of',            'TRUST',     true,  'TRUST_OF',                 '00000000-0000-0000-0000-000000000001'),
  ('SETTLOR_OF',          'Settlor of',            'TRUST',     true,  'TRUST_OF',                 '00000000-0000-0000-0000-000000000001'),
  ('BENEFICIARY_OF',      'Beneficiary of',        'TRUST',     true,  'TRUST_OF',                 '00000000-0000-0000-0000-000000000001'),
  ('GUARDIAN_OF',         'Guardian of',           'PERSONAL',  true,  'WARD_OF',                  '00000000-0000-0000-0000-000000000001'),
  ('ATTORNEY_FOR',        'Attorney for',          'PERSONAL',  true,  'CLIENT_OF',                '00000000-0000-0000-0000-000000000001'),
  ('CLIENT_OF',           'Client of',             'PERSONAL',  true,  'ATTORNEY_FOR',             '00000000-0000-0000-0000-000000000001'),
  ('ADVISOR_TO',          'Advisor to',            'PROFESSIONAL',true,'CLIENT_OF',                '00000000-0000-0000-0000-000000000001'),
  ('DIRECTOR_OF',         'Director of',           'GOVERNANCE',true,  'BOARD_OF',                 '00000000-0000-0000-0000-000000000001'),
  ('OFFICER_OF',          'Officer of',            'GOVERNANCE',true,  'COMPANY',                  '00000000-0000-0000-0000-000000000001'),
  ('SHAREHOLDER_OF',      'Shareholder of',        'OWNERSHIP', true,  'COMPANY',                  '00000000-0000-0000-0000-000000000001'),
  ('SUBSIDIARY_OF',       'Subsidiary of',         'CORPORATE', true,  'PARENT_OF',                '00000000-0000-0000-0000-000000000001'),
  ('AFFILIATE_OF',        'Affiliate of',          'CORPORATE', true,  'AFFILIATE_OF',             '00000000-0000-0000-0000-000000000001'),
  ('JOINT_VENTURE',       'Joint venture with',    'CORPORATE', true,  'JOINT_VENTURE',            '00000000-0000-0000-0000-000000000001'),
  ('COUNTERPARTY_OF',     'Counterparty of',       'BUSINESS',  true,  'COUNTERPARTY_OF',          '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, relationship_cd) DO NOTHING;

-- ════════════════════════════════════════════════════════════════════════
-- kyc_status
-- ════════════════════════════════════════════════════════════════════════
INSERT INTO mdm.kyc_status (status_cd, name, is_complete, is_pending, is_failed, allows_transactions, display_order, tenant_id) VALUES
  ('NOT_STARTED',       'Not Started',          false, true,  false, false, 10, '00000000-0000-0000-0000-000000000001'),
  ('PENDING_DOCUMENTS', 'Pending Documents',    false, true,  false, false, 20, '00000000-0000-0000-0000-000000000001'),
  ('IN_REVIEW',         'In Review',            false, true,  false, false, 30, '00000000-0000-0000-0000-000000000001'),
  ('APPROVED',          'Approved',             true,  false, false, true,  40, '00000000-0000-0000-0000-000000000001'),
  ('REJECTED',          'Rejected',             true,  false, true,  false, 50, '00000000-0000-0000-0000-000000000001'),
  ('EXPIRED',           'Expired',              false, true,  false, false, 60, '00000000-0000-0000-0000-000000000001'),
  ('SUSPENDED',         'Suspended',            false, false, false, false, 70, '00000000-0000-0000-0000-000000000001'),
  ('WAIVED',            'Waived (Low Risk)',    true,  false, false, true,  80, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, status_cd) DO NOTHING;

-- ════════════════════════════════════════════════════════════════════════
-- risk_rating
-- ════════════════════════════════════════════════════════════════════════
INSERT INTO mdm.risk_rating (rating_cd, name, rating_type, rating_level, review_frequency_months, requires_edd, requires_senior_approval, display_order, tenant_id) VALUES
  ('LOW',          'Low Risk',            'AML',        1,  36, false, false, 10, '00000000-0000-0000-0000-000000000001'),
  ('MEDIUM_LOW',   'Medium-Low Risk',     'AML',        2,  24, false, false, 20, '00000000-0000-0000-0000-000000000001'),
  ('MEDIUM',       'Medium Risk',         'AML',        3,  18, false, false, 30, '00000000-0000-0000-0000-000000000001'),
  ('MEDIUM_HIGH',  'Medium-High Risk',    'AML',        4,  12, true,  false, 40, '00000000-0000-0000-0000-000000000001'),
  ('HIGH',         'High Risk',           'AML',        5,  12, true,  true,  50, '00000000-0000-0000-0000-000000000001'),
  ('VERY_HIGH',    'Very High Risk',      'AML',        6,  6,  true,  true,  60, '00000000-0000-0000-0000-000000000001'),
  ('PROHIBITED',   'Prohibited',          'AML',        7,  NULL,true,  true,  70, '00000000-0000-0000-0000-000000000001'),
  ('LOW_CREDIT',   'Low Credit Risk',     'CREDIT',     1,  36, false, false, 110, '00000000-0000-0000-0000-000000000001'),
  ('MEDIUM_CREDIT','Medium Credit Risk',  'CREDIT',     2,  24, false, false, 120, '00000000-0000-0000-0000-000000000001'),
  ('HIGH_CREDIT',  'High Credit Risk',    'CREDIT',     3,  12, true,  true,  130, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, rating_cd, rating_type) DO NOTHING;

-- ════════════════════════════════════════════════════════════════════════
-- source_of_wealth_type
-- ════════════════════════════════════════════════════════════════════════
INSERT INTO mdm.source_of_wealth_type (source_cd, name, category, requires_documentation, risk_weight, tenant_id) VALUES
  ('SALARY',                 'Salary / Employment Income',         'EMPLOYMENT',  true,  0.5, '00000000-0000-0000-0000-000000000001'),
  ('BONUS',                  'Bonus / Commission',                 'EMPLOYMENT',  true,  0.5, '00000000-0000-0000-0000-000000000001'),
  ('BUSINESS_INCOME',        'Business Income',                    'BUSINESS',    true,  1.5, '00000000-0000-0000-0000-000000000001'),
  ('BUSINESS_SALE',          'Business Sale Proceeds',             'BUSINESS',    true,  2.0, '00000000-0000-0000-0000-000000000001'),
  ('INVESTMENT_GAINS',       'Investment Gains',                   'INVESTMENT',  true,  0.5, '00000000-0000-0000-0000-000000000001'),
  ('DIVIDENDS',              'Dividend Income',                    'INVESTMENT',  true,  0.3, '00000000-0000-0000-0000-000000000001'),
  ('INTEREST',               'Interest Income',                    'INVESTMENT',  true,  0.3, '00000000-0000-0000-0000-000000000001'),
  ('RENTAL_INCOME',          'Rental Income',                      'INVESTMENT',  true,  1.0, '00000000-0000-0000-0000-000000000001'),
  ('REAL_ESTATE_SALE',       'Real Estate Sale',                   'INVESTMENT',  true,  1.5, '00000000-0000-0000-0000-000000000001'),
  ('INHERITANCE',            'Inheritance',                        'PERSONAL',    true,  1.0, '00000000-0000-0000-0000-000000000001'),
  ('GIFT',                   'Gift',                               'PERSONAL',    true,  1.0, '00000000-0000-0000-0000-000000000001'),
  ('LOTTERY_WINNING',        'Lottery / Gambling Winnings',        'PERSONAL',    true,  3.0, '00000000-0000-0000-0000-000000000001'),
  ('LAWSUIT_SETTLEMENT',     'Lawsuit Settlement',                 'PERSONAL',    true,  2.5, '00000000-0000-0000-0000-000000000001'),
  ('INSURANCE_PAYOUT',       'Insurance Payout',                   'PERSONAL',    true,  0.5, '00000000-0000-0000-0000-000000000001'),
  ('RETIREMENT_BENEFITS',    'Retirement / Pension Benefits',      'PERSONAL',    true,  0.5, '00000000-0000-0000-0000-000000000001'),
  ('PRIOR_ACCUMULATED_WEALTH','Prior Accumulated Wealth',          'OTHER',       true,  1.0, '00000000-0000-0000-0000-000000000001'),
  ('FAMILY_FUNDING',         'Family Funding',                     'OTHER',       true,  1.0, '00000000-0000-0000-0000-000000000001'),
  ('IPO_PROCEEDS',           'IPO Proceeds',                       'BUSINESS',    true,  1.0, '00000000-0000-0000-0000-000000000001'),
  ('CRYPTO_GAINS',           'Cryptocurrency Gains',               'INVESTMENT',  true,  3.0, '00000000-0000-0000-0000-000000000001'),
  ('UNKNOWN',                'Unknown / Not Disclosed',            'OTHER',       false, 5.0, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, source_cd) DO NOTHING;

-- ════════════════════════════════════════════════════════════════════════
-- sanctions_list_source
-- ════════════════════════════════════════════════════════════════════════
INSERT INTO mdm.sanctions_list_source (list_cd, name, jurisdiction, issuing_authority, list_type, update_frequency, tenant_id) VALUES
  ('OFAC_SDN',            'OFAC SDN',              'US', 'US Treasury OFAC',           'INDIVIDUAL', 'DAILY',   '00000000-0000-0000-0000-000000000001'),
  ('OFAC_CONSOLIDATED',   'OFAC Consolidated',     'US', 'US Treasury OFAC',           'COMBINED',   'DAILY',   '00000000-0000-0000-0000-000000000001'),
  ('OFAC_FSEIR',          'OFAC FSE-IR',           'US', 'US Treasury OFAC',           'ENTITY',     'DAILY',   '00000000-0000-0000-0000-000000000001'),
  ('OFAC_SSI',            'OFAC Sectoral SSI',     'US', 'US Treasury OFAC',           'SECTOR',     'AS_NEEDED','00000000-0000-0000-0000-000000000001'),
  ('EU_SANCTIONS',        'EU Consolidated List',  'EU', 'EU Council',                  'COMBINED',   'DAILY',   '00000000-0000-0000-0000-000000000001'),
  ('UN_SANCTIONS',        'UN Consolidated List',  'UN', 'UN Security Council',         'COMBINED',   'DAILY',   '00000000-0000-0000-0000-000000000001'),
  ('UK_OFSI',             'UK OFSI Consolidated',  'GB', 'UK OFSI',                     'COMBINED',   'DAILY',   '00000000-0000-0000-0000-000000000001'),
  ('UK_HMT',              'UK HMT Financial',      'GB', 'UK HM Treasury',              'INDIVIDUAL', 'DAILY',   '00000000-0000-0000-0000-000000000001'),
  ('DFAT_AUSTRALIA',      'DFAT Australia',        'AU', 'Australian DFAT',             'COMBINED',   'WEEKLY',  '00000000-0000-0000-0000-000000000001'),
  ('DFATD_CANADA',        'DFATD Canada',          'CA', 'Canadian Government',         'COMBINED',   'WEEKLY',  '00000000-0000-0000-0000-000000000001'),
  ('SECO_SWITZERLAND',    'SECO Switzerland',      'CH', 'Swiss SECO',                  'COMBINED',   'WEEKLY',  '00000000-0000-0000-0000-000000000001'),
  ('PEP_GLOBAL',          'Global PEP Database',   NULL,'WorldCompliance / Dow Jones','INDIVIDUAL', 'DAILY',   '00000000-0000-0000-0000-000000000001'),
  ('ADVERSE_MEDIA',       'Adverse Media Database',NULL,'Refinitiv / Dow Jones',       'INDIVIDUAL', 'DAILY',   '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, list_cd) DO NOTHING;

-- ════════════════════════════════════════════════════════════════════════
-- party_document_type
-- ════════════════════════════════════════════════════════════════════════
INSERT INTO mdm.party_document_type (doc_type_cd, name, category, applies_to, is_mandatory_for_kyc, has_expiry, validity_months, requires_certification, display_order, tenant_id) VALUES
  ('PASSPORT',            'Passport',                       'IDENTITY',  'INDIVIDUAL',  true,  true,  120, true,  10, '00000000-0000-0000-0000-000000000001'),
  ('NATIONAL_ID',         'National ID Card',               'IDENTITY',  'INDIVIDUAL',  true,  true,  120, false, 20, '00000000-0000-0000-0000-000000000001'),
  ('DRIVERS_LICENSE',     'Drivers License',                'IDENTITY',  'INDIVIDUAL',  false, true,  60,  false, 30, '00000000-0000-0000-0000-000000000001'),
  ('RESIDENCE_PERMIT',    'Residence Permit',               'IDENTITY',  'INDIVIDUAL',  false, true,  60,  false, 40, '00000000-0000-0000-0000-000000000001'),
  ('UTILITY_BILL',        'Utility Bill (Address Proof)',   'ADDRESS',   'BOTH',       true,  true,  6,   false, 50, '00000000-0000-0000-0000-000000000001'),
  ('BANK_STATEMENT',      'Bank Statement',                 'FINANCIAL', 'BOTH',       false, true,  6,   false, 60, '00000000-0000-0000-0000-000000000001'),
  ('TAX_RETURN',          'Tax Return',                     'FINANCIAL', 'INDIVIDUAL', false, true,  12,  true,  70, '00000000-0000-0000-0000-000000000001'),
  ('W9',                  'IRS Form W-9',                   'TAX',       'INDIVIDUAL',  false, false, NULL, true,  80, '00000000-0000-0000-0000-000000000001'),
  ('W8BEN',               'IRS Form W-8BEN',                'TAX',       'INDIVIDUAL',  false, false, NULL, true,  81, '00000000-0000-0000-0000-000000000001'),
  ('W8BEN_E',             'IRS Form W-8BEN-E',              'TAX',       'ENTITY',      false, false, NULL, true,  82, '00000000-0000-0000-0000-000000000001'),
  ('CRS_SELF_CERT',       'CRS Self-Certification',         'TAX',       'BOTH',       true,  true,  36,  true,  90, '00000000-0000-0000-0000-000000000001'),
  ('INCORPORATION_CERT',  'Certificate of Incorporation',   'CORPORATE', 'ENTITY',     true,  false, NULL, true,  100, '00000000-0000-0000-0000-000000000001'),
  ('ARTICLES_ASSOC',      'Articles of Association',        'CORPORATE', 'ENTITY',     false, false, NULL, true,  110, '00000000-0000-0000-0000-000000000001'),
  ('MEMORANDUM_ASSOC',    'Memorandum of Association',      'CORPORATE', 'ENTITY',     false, false, NULL, true,  111, '00000000-0000-0000-0000-000000000001'),
  ('BOARD_RESOLUTION',    'Board Resolution',               'CORPORATE', 'ENTITY',     false, false, NULL, true,  120, '00000000-0000-0000-0000-000000000001'),
  ('CERT_GOOD_STANDING',  'Certificate of Good Standing',   'CORPORATE', 'ENTITY',     false, true,  12,  true,  130, '00000000-0000-0000-0000-000000000001'),
  ('LEI_CERTIFICATE',     'LEI Certificate',                'CORPORATE', 'ENTITY',     false, true,  12,  false, 140, '00000000-0000-0000-0000-000000000001'),
  ('TRUST_DEED',          'Trust Deed',                     'TRUST',     'TRUST',      true,  false, NULL, true,  200, '00000000-0000-0000-0000-000000000001'),
  ('LETTER_OF_WISHES',    'Letter of Wishes',               'TRUST',     'TRUST',      false, false, NULL, false, 210, '00000000-0000-0000-0000-000000000001'),
  ('PROSPECTUS',          'Fund Prospectus',                'FUND',      'FUND',       true,  true,  24,  false, 300, '00000000-0000-0000-0000-000000000001'),
  ('AUDITED_FINANCIALS',  'Audited Financials',             'FINANCIAL', 'BOTH',       false, true,  12,  true,  400, '00000000-0000-0000-0000-000000000001'),
  ('UNAUDITED_FINANCIALS','Unaudited Financials',           'FINANCIAL', 'BOTH',       false, true,  12,  false, 401, '00000000-0000-0000-0000-000000000001'),
  ('SOURCE_OF_WEALTH_DOC','Source of Wealth Documentation', 'FINANCIAL', 'BOTH',       true,  false, NULL, true,  500, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, doc_type_cd) DO NOTHING;

-- ════════════════════════════════════════════════════════════════════════
-- fatca_crs_status
-- ════════════════════════════════════════════════════════════════════════
INSERT INTO mdm.fatca_crs_status (status_cd, name, regime, description, requires_reporting, requires_withholding, display_order, tenant_id) VALUES
  ('US_SPECIFIED_PERSON',     'US Specified Person',         'FATCA', 'US person subject to FATCA reporting',                 true,  true,  10, '00000000-0000-0000-0000-000000000001'),
  ('US_CITIZEN',             'US Citizen / Resident',       'FATCA', 'US citizen or resident for tax purposes',              true,  true,  11, '00000000-0000-0000-0000-000000000001'),
  ('NON_US_PERSON',          'Non-US Person',               'FATCA', 'Person not subject to FATCA reporting',                false, false, 20, '00000000-0000-0000-0000-000000000001'),
  ('RECALCITRANT',           'Recalcitrant Account Holder', 'FATCA', 'Account holder refusing FATCA disclosure',             true,  true,  30, '00000000-0000-0000-0000-000000000001'),
  ('PARTICIPATING_FFI',      'Participating FFI',           'FATCA', 'Participating Foreign Financial Institution',          true,  false, 40, '00000000-0000-0000-0000-000000000001'),
  ('REPORTING_FIM_FFI',      'Reporting Model 1 FFI',       'FATCA', 'FFI reporting under Model 1 IGA',                       true,  false, 41, '00000000-0000-0000-0000-000000000001'),
  ('REPORTING_FIM2_FFI',     'Reporting Model 2 FFI',       'FATCA', 'FFI reporting under Model 2 IGA',                       true,  false, 42, '00000000-0000-0000-0000-000000000001'),
  ('NON_REPORTING_FFI',      'Non-Reporting FFI',           'FATCA', 'Non-reporting FFI under IGA',                           false, false, 43, '00000000-0000-0000-0000-000000000001'),
  ('EXEMPT_BENEFICIAL_OWNER','Exempt Beneficial Owner',     'FATCA', 'EBO under FATCA regulations',                           false, false, 50, '00000000-0000-0000-0000-000000000001'),
  ('CRS_REPORTABLE',         'CRS Reportable Person',       'CRS',   'Person subject to CRS reporting',                       true,  false, 100, '00000000-0000-0000-0000-000000000001'),
  ('CRS_NON_REPORTABLE',     'CRS Non-Reportable',          'CRS',   'Person not subject to CRS',                             false, false, 110, '00000000-0000-0000-0000-000000000001'),
  ('CRS_RECIPIENT',          'CRS Recipient',               'CRS',   'Reportable recipient under CRS',                        true,  false, 120, '00000000-0000-0000-0000-000000000001'),
  ('CRS_CONTROLLING_PERSON', 'CRS Controlling Person',      'CRS',   'Controlling person of a passive NFE under CRS',         true,  false, 130, '00000000-0000-0000-0000-000000000001'),
  ('UK_CDOT_REPORTABLE',     'UK CDOT Reportable',          'UK_CDOT','UK Common Reporting Standard reportable account',       true,  false, 200, '00000000-0000-0000-0000-000000000001'),
  ('UK_CDOT_NON_REPORTABLE', 'UK CDOT Non-Reportable',      'UK_CDOT','UK CDOT non-reportable',                                false, false, 210, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, status_cd, regime) DO NOTHING;

-- Migration log: this file is the canonical party reference seed.
-- Once applied, individual tenants can either (a) inherit these via the
-- shared-reference RLS pattern, or (b) INSERT their own overrides.
