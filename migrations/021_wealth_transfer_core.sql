-- ============================================================================
-- WEALTH TRANSFER PLATFORM: CORE SCHEMA
-- ============================================================================
-- Migration: 021_wealth_transfer_core.sql
-- Purpose: Comprehensive family office, estate planning, and wealth transfer
--          management following metadata-first, configure-over-code principles
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For full-text search

-- ============================================================================
-- ENUMS FOR TYPE SAFETY
-- ============================================================================

CREATE TYPE family_office_status AS ENUM (
    'NOT_STARTED',
    'IN_PROGRESS', 
    'IMPLEMENTED',
    'REVIEW_NEEDED',
    'INACTIVE'
);

CREATE TYPE engagement_level AS ENUM (
    'UNENGAGED',
    'LOW',
    'MEDIUM',
    'HIGH'
);

CREATE TYPE marital_status AS ENUM (
    'SINGLE',
    'MARRIED',
    'DIVORCED',
    'WIDOWED',
    'SEPARATED'
);

CREATE TYPE employment_status AS ENUM (
    'EMPLOYED',
    'RETIRED',
    'STUDENT',
    'UNEMPLOYED',
    'SELF_EMPLOYED'
);

CREATE TYPE asset_class_enum AS ENUM (
    'REAL_ESTATE',
    'BUSINESS_INTEREST',
    'INVESTMENT_ACCOUNT',
    'RETIREMENT_ACCOUNT',
    'LIFE_INSURANCE',
    'TRUST_OWNERSHIP',
    'ART_COLLECTIBLES',
    'INTELLECTUAL_PROPERTY',
    'CRYPTOCURRENCY',
    'ALTERNATIVE_INVESTMENT',
    'CASH',
    'OTHER'
);

CREATE TYPE valuation_method AS ENUM (
    'MARKET_PRICE',
    'APPRAISAL',
    'BOOK_VALUE',
    'DCF',
    'COMPARABLE_SALES'
);

CREATE TYPE entity_type_enum AS ENUM (
    'REVOCABLE_TRUST',
    'IRREVOCABLE_TRUST',
    'SLAT',
    'GRAT',
    'QPRT',
    'ILIT',
    'DYNASTY_TRUST',
    'CHARITABLE_REMAINDER_TRUST',
    'CHARITABLE_LEAD_TRUST',
    'CRUMMEY_TRUST',
    'QTIP',
    'GENERATION_SKIPPING_TRUST',
    'SPECIAL_NEEDS_TRUST',
    'LLC',
    'FAMILY_LIMITED_PARTNERSHIP',
    'PRIVATE_FOUNDATION',
    'DONOR_ADVISED_FUND'
);

CREATE TYPE entity_status AS ENUM (
    'ACTIVE',
    'PENDING',
    'TERMINATED',
    'REVOKED'
);

CREATE TYPE gift_type_enum AS ENUM (
    'ANNUAL_EXCLUSION',
    'LIFETIME_EXEMPTION',
    'GENERATION_SKIPPING_TRANSFER',
    'CHARITABLE',
    'EDUCATIONAL_MEDICAL_EXCLUSION',
    'SPOUSAL_UNLIMITED',
    'PRESENT_INTEREST',
    'FUTURE_INTEREST'
);

CREATE TYPE strategy_type_enum AS ENUM (
    'NO_PLANNING',
    'ANNUAL_GIFTING',
    'SLAT',
    'GRAT',
    'QPRT',
    'ILIT',
    'DYNASTY_TRUST',
    'CHARITABLE_REMAINDER_TRUST',
    'CHARITABLE_LEAD_TRUST',
    'INSTALLMENT_SALE_IDGT',
    'FAMILY_LIMITED_PARTNERSHIP',
    'COMBINATION'
);

CREATE TYPE audit_risk_level AS ENUM (
    'LOW',
    'MEDIUM',
    'HIGH'
);

-- ============================================================================
-- FAMILY CORE ENTITIES
-- ============================================================================

CREATE TABLE family_offices (
    family_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL, -- Multi-tenant isolation
    
    -- Basic Information
    family_name TEXT NOT NULL,
    legal_entity_name TEXT,
    
    -- Advisor Assignment
    primary_advisor_id UUID, -- References users(id) in main system
    backup_advisor_id UUID,
    
    -- Financial Aggregates (denormalized for performance)
    total_estimated_networth DECIMAL(18,2) NOT NULL DEFAULT 0,
    total_liquid_assets DECIMAL(18,2) DEFAULT 0,
    total_illiquid_assets DECIMAL(18,2) DEFAULT 0,
    total_liabilities DECIMAL(18,2) DEFAULT 0,
    
    -- Estate Planning Status
    estate_plan_status family_office_status DEFAULT 'NOT_STARTED',
    last_plan_review_date DATE,
    next_plan_review_date DATE,
    
    -- Family Governance
    has_family_constitution BOOLEAN DEFAULT FALSE,
    family_constitution_document_id UUID, -- Link to document storage
    governance_structure JSONB, -- Flexible governance rules
    
    -- Multi-Generational Tracking
    patriarch_id UUID, -- References family_members(member_id)
    matriarch_id UUID,
    generation_count INTEGER DEFAULT 1,
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT valid_networth CHECK (total_estimated_networth >= 0),
    CONSTRAINT valid_generation_count CHECK (generation_count > 0)
);

-- Indexes for performance
CREATE INDEX idx_family_tenant ON family_offices(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_family_advisor ON family_offices(primary_advisor_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_family_plan_status ON family_offices(estate_plan_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_family_review_date ON family_offices(next_plan_review_date) 
    WHERE estate_plan_status = 'IMPLEMENTED' AND deleted_at IS NULL;

-- Full-text search on family name
CREATE INDEX idx_family_name_trgm ON family_offices USING gin(family_name gin_trgm_ops);

COMMENT ON TABLE family_offices IS 'Multi-generational family office entities with estate planning tracking';
COMMENT ON COLUMN family_offices.governance_structure IS 'JSONB: {board_members, voting_rules, meeting_frequency}';

-- ============================================================================
-- FAMILY MEMBERS
-- ============================================================================

CREATE TABLE family_members (
    member_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES family_offices(family_id) ON DELETE CASCADE,
    
    -- Personal Information
    legal_first_name TEXT NOT NULL,
    legal_middle_name TEXT,
    legal_last_name TEXT NOT NULL,
    preferred_name TEXT,
    suffix VARCHAR(10),
    
    date_of_birth DATE NOT NULL,
    ssn_encrypted TEXT, -- Encrypted at application layer
    citizenship TEXT[] DEFAULT ARRAY['US'],
    
    -- Residency (critical for estate tax)
    primary_state_residency VARCHAR(2) NOT NULL,
    secondary_residences JSONB, -- [{state, days_per_year, address}]
    domicile_state VARCHAR(2) NOT NULL,
    
    -- Generation Tracking
    generation INTEGER NOT NULL CHECK (generation >= 1 AND generation <= 10),
    parent_member_ids UUID[], -- Array of parent IDs
    spouse_member_id UUID REFERENCES family_members(member_id),
    children_member_ids UUID[],
    
    -- Financial Profile
    separate_networth DECIMAL(18,2) DEFAULT 0,
    annual_income DECIMAL(12,2),
    employment_status employment_status,
    occupation TEXT,
    
    -- Risk & Preferences
    risk_tolerance_score DECIMAL(3,2) CHECK (risk_tolerance_score >= 0 AND risk_tolerance_score <= 10),
    investment_philosophy TEXT,
    esg_preferences JSONB,
    
    -- Financial Literacy (AI-assessed)
    financial_literacy_score DECIMAL(3,2) CHECK (financial_literacy_score >= 0 AND financial_literacy_score <= 10),
    literacy_assessment_date DATE,
    literacy_assessment_method VARCHAR(50),
    
    -- Life Stage Events
    marital_status marital_status NOT NULL DEFAULT 'SINGLE',
    marriage_date DATE,
    prenuptial_agreement BOOLEAN DEFAULT FALSE,
    prenup_document_id UUID,
    divorce_date DATE,
    divorce_settlement_details JSONB,
    
    -- Children & Dependents
    children_count INTEGER DEFAULT 0,
    has_special_needs_dependents BOOLEAN DEFAULT FALSE,
    special_needs_details JSONB,
    
    -- Education
    education_level VARCHAR(50),
    current_student BOOLEAN DEFAULT FALSE,
    student_loan_balance DECIMAL(12,2),
    
    -- Health (for insurance & incapacity planning)
    has_chronic_health_conditions BOOLEAN,
    life_expectancy_estimate INTEGER,
    long_term_care_insurance BOOLEAN,
    
    -- Platform Engagement
    platform_user_id UUID, -- Link to auth system
    onboarding_status VARCHAR(50) DEFAULT 'NOT_INVITED',
    invitation_sent_date DATE,
    first_login_date DATE,
    last_login_date DATE,
    
    engagement_score DECIMAL(3,2) CHECK (engagement_score >= 0 AND engagement_score <= 1),
    engagement_last_calculated TIMESTAMPTZ,
    
    communication_preferences JSONB,
    
    -- Anticipated Life Events
    anticipated_major_expenses JSONB, -- [{type, child_name, start_year, estimated_cost}]
    retirement_target_age INTEGER,
    retirement_target_date DATE,
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT valid_generation CHECK (generation > 0),
    CONSTRAINT valid_dob CHECK (date_of_birth <= CURRENT_DATE),
    CONSTRAINT valid_spouse CHECK (spouse_member_id != member_id)
);

CREATE INDEX idx_member_family ON family_members(family_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_member_generation ON family_members(family_id, generation) WHERE deleted_at IS NULL;
CREATE INDEX idx_member_platform_user ON family_members(platform_user_id) WHERE platform_user_id IS NOT NULL;
CREATE INDEX idx_member_engagement ON family_members(engagement_score DESC) WHERE engagement_score IS NOT NULL;
CREATE INDEX idx_member_onboarding ON family_members(onboarding_status) WHERE onboarding_status != 'COMPLETE';

COMMENT ON TABLE family_members IS 'Individual family members with comprehensive financial and engagement tracking';
COMMENT ON COLUMN family_members.anticipated_major_expenses IS 'JSONB array of future expenses like college tuition';

-- ============================================================================
-- FAMILY ASSETS
-- ============================================================================

CREATE TABLE family_assets (
    asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES family_offices(family_id) ON DELETE CASCADE,
    
    -- Asset Identification
    asset_class asset_class_enum NOT NULL,
    asset_name TEXT NOT NULL,
    asset_description TEXT,
    asset_identifier TEXT, -- Account number, address, VIN, etc.
    
    -- Custodian/Location
    custodian_name TEXT,
    custodian_account_number TEXT,
    physical_location TEXT,
    
    -- Valuation
    current_valuation DECIMAL(18,2) NOT NULL,
    valuation_date DATE NOT NULL DEFAULT CURRENT_DATE,
    valuation_method valuation_method NOT NULL,
    valuation_firm TEXT,
    appraisal_document_id UUID,
    
    cost_basis DECIMAL(18,2),
    acquisition_date DATE,
    unrealized_gain_loss DECIMAL(18,2) GENERATED ALWAYS AS (current_valuation - COALESCE(cost_basis, 0)) STORED,
    
    -- Tax Attributes
    stepped_up_basis_eligible BOOLEAN DEFAULT TRUE,
    depreciation_eligible BOOLEAN DEFAULT FALSE,
    annual_depreciation DECIMAL(12,2),
    
    -- Estate Planning Attributes
    included_in_gross_estate BOOLEAN DEFAULT TRUE,
    estate_tax_discount_pct DECIMAL(5,2) DEFAULT 0,
    adjusted_estate_value DECIMAL(18,2) GENERATED ALWAYS AS (
        current_valuation * (1 - COALESCE(estate_tax_discount_pct, 0) / 100.0)
    ) STORED,
    
    -- Ownership Structure (critical for transfer planning)
    ownership_structure JSONB NOT NULL,
    /* Example:
    [
        {"owner_type": "INDIVIDUAL", "owner_id": "member-uuid", "ownership_pct": 50.0},
        {"owner_type": "TRUST", "owner_id": "trust-uuid", "ownership_pct": 50.0}
    ]
    */
    
    -- Liquidity Profile
    illiquid BOOLEAN DEFAULT FALSE,
    estimated_time_to_liquidate_days INTEGER,
    estimated_liquidation_cost_pct DECIMAL(5,2),
    
    -- Income Generation
    generates_income BOOLEAN DEFAULT FALSE,
    annual_income_generated DECIMAL(12,2),
    income_type VARCHAR(50),
    
    -- Debt Encumbrance
    has_debt BOOLEAN DEFAULT FALSE,
    outstanding_debt_balance DECIMAL(18,2),
    debt_interest_rate DECIMAL(5,4),
    debt_maturity_date DATE,
    
    -- Transfer Restrictions
    has_transfer_restrictions BOOLEAN DEFAULT FALSE,
    transfer_restriction_details TEXT,
    right_of_first_refusal BOOLEAN,
    buy_sell_agreement_exists BOOLEAN,
    buy_sell_agreement_document_id UUID,
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT valid_valuation CHECK (current_valuation >= 0),
    CONSTRAINT valid_ownership CHECK (jsonb_array_length(ownership_structure) > 0)
);

CREATE INDEX idx_asset_family ON family_assets(family_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_asset_class ON family_assets(family_id, asset_class) WHERE deleted_at IS NULL;
CREATE INDEX idx_asset_valuation_date ON family_assets(valuation_date DESC);
CREATE INDEX idx_asset_illiquid ON family_assets(family_id) WHERE illiquid = TRUE AND deleted_at IS NULL;
CREATE INDEX idx_asset_ownership ON family_assets USING GIN(ownership_structure);

COMMENT ON TABLE family_assets IS 'Family assets with ownership attribution and tax planning attributes';
COMMENT ON COLUMN family_assets.ownership_structure IS 'JSONB array defining fractional ownership by individuals/trusts';

-- ============================================================================
-- ESTATE ENTITIES (TRUSTS, LLCS, FOUNDATIONS)
-- ============================================================================

CREATE TABLE estate_entities (
    entity_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES family_offices(family_id) ON DELETE CASCADE,
    entity_type entity_type_enum NOT NULL,
    entity_name TEXT NOT NULL,
    entity_legal_name TEXT,
    
    -- Formation Details
    formation_date DATE NOT NULL,
    formation_state VARCHAR(2) NOT NULL,
    situs_state VARCHAR(2),
    governing_law_state VARCHAR(2),
    
    -- Tax Identification
    tax_id VARCHAR(20),
    tax_id_application_date DATE,
    tax_classification VARCHAR(50),
    
    -- Legal Documentation
    formation_document_id UUID,
    trust_agreement_document_id UUID,
    operating_agreement_document_id UUID,
    amendment_document_ids UUID[],
    
    -- Parties (Role-based)
    grantor_member_ids UUID[] NOT NULL,
    trustee_member_ids UUID[],
    trustee_entity_ids UUID[],
    successor_trustee_ids UUID[],
    beneficiary_member_ids UUID[] NOT NULL,
    contingent_beneficiary_member_ids UUID[],
    
    -- Trust-Specific Terms (polymorphic)
    terms JSONB,
    /* Example structure:
    {
        "distribution_rules": {
            "income_distribution": "MANDATORY_ANNUAL",
            "principal_distribution": "DISCRETIONARY",
            "distribution_age": 25,
            "staggered_distribution": [
                {"age": 25, "pct": 33},
                {"age": 30, "pct": 33},
                {"age": 35, "pct": 34}
            ]
        },
        "spendthrift_clause": true,
        "generation_skipping": true,
        "special_provisions": "HEMS standard"
    }
    */
    
    termination_date DATE,
    termination_event TEXT,
    
    -- Asset Holdings
    current_total_value DECIMAL(18,2) DEFAULT 0,
    asset_allocation JSONB,
    
    -- GRAT-Specific
    grat_annuity_amount DECIMAL(15,2),
    grat_annuity_frequency VARCHAR(20),
    grat_term_years INTEGER,
    grat_remainder_beneficiaries UUID[],
    
    -- ILIT-Specific
    ilit_life_insurance_policy_id UUID,
    ilit_crummey_withdrawal_rights BOOLEAN,
    
    -- Dynasty Trust
    dynasty_perpetual BOOLEAN DEFAULT FALSE,
    dynasty_generation_limit INTEGER,
    
    -- Foundation-Specific
    foundation_annual_distribution_requirement DECIMAL(5,4),
    foundation_tax_year_end DATE,
    foundation_irs_determination_letter_id UUID,
    
    -- Compliance & Filing
    annual_tax_filing_required BOOLEAN DEFAULT TRUE,
    last_tax_filing_date DATE,
    next_tax_filing_due_date DATE,
    requires_state_registration BOOLEAN,
    state_registration_number TEXT,
    
    -- Banking
    bank_account_info JSONB,
    
    -- Status
    entity_status entity_status DEFAULT 'ACTIVE',
    termination_date_actual DATE,
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT valid_entity_value CHECK (current_total_value >= 0)
);

CREATE INDEX idx_entity_family ON estate_entities(family_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_entity_type ON estate_entities(entity_type, entity_status);
CREATE INDEX idx_entity_grantor ON estate_entities USING GIN(grantor_member_ids);
CREATE INDEX idx_entity_beneficiary ON estate_entities USING GIN(beneficiary_member_ids);
CREATE INDEX idx_entity_tax_filing ON estate_entities(next_tax_filing_due_date) WHERE entity_status = 'ACTIVE';

COMMENT ON TABLE estate_entities IS 'Trusts, LLCs, and foundations with polymorphic terms structure';
COMMENT ON COLUMN estate_entities.terms IS 'JSONB flexible structure for trust-specific rules and provisions';

-- Continued in part 2...

-- ============================================================================
-- GIFT HISTORY & EXEMPTION TRACKING
-- ============================================================================

CREATE TABLE gift_history (
    gift_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES family_offices(family_id) ON DELETE CASCADE,
    
    -- Transaction Details
    donor_member_id UUID NOT NULL REFERENCES family_members(member_id),
    recipient_member_id UUID REFERENCES family_members(member_id),
    recipient_entity_id UUID REFERENCES estate_entities(entity_id),
    
    gift_date DATE NOT NULL,
    gift_type gift_type_enum NOT NULL,
    
    -- Asset Transferred
    asset_id UUID REFERENCES family_assets(asset_id),
    asset_description TEXT NOT NULL,
    
    -- Valuation
    fair_market_value DECIMAL(18,2) NOT NULL,
    valuation_method valuation_method NOT NULL,
    valuation_document_id UUID,
    
    -- Discounts (critical for estate planning)
    valuation_discount_pct DECIMAL(5,2) DEFAULT 0,
    net_gift_value DECIMAL(18,2) GENERATED ALWAYS AS (
        fair_market_value * (1 - COALESCE(valuation_discount_pct, 0) / 100.0)
    ) STORED,
    
    -- Exemption Utilization
    annual_exclusion_utilized DECIMAL(12,2) DEFAULT 0,
    lifetime_exemption_utilized DECIMAL(12,2) DEFAULT 0,
    gst_exemption_utilized DECIMAL(12,2) DEFAULT 0,
    
    -- Spousal Split Election
    spousal_split_election BOOLEAN DEFAULT FALSE,
    spouse_member_id UUID REFERENCES family_members(member_id),
    
    -- Gift Tax Filing (Form 709)
    requires_gift_tax_return BOOLEAN DEFAULT FALSE,
    form_709_filed BOOLEAN DEFAULT FALSE,
    form_709_filing_date DATE,
    form_709_document_id UUID,
    form_709_due_date DATE,
    
    -- Generation-Skipping Transfer
    is_generation_skipping BOOLEAN DEFAULT FALSE,
    generation_skip_count INTEGER,
    
    -- Gift Conditions
    gift_structure VARCHAR(50),
    gift_restrictions TEXT,
    
    -- Notes
    gift_purpose TEXT,
    advisor_notes TEXT,
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT valid_gift_value CHECK (fair_market_value > 0),
    CONSTRAINT valid_recipient CHECK (
        (recipient_member_id IS NOT NULL AND recipient_entity_id IS NULL) OR
        (recipient_member_id IS NULL AND recipient_entity_id IS NOT NULL)
    )
);

CREATE INDEX idx_gift_family ON gift_history(family_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_gift_donor ON gift_history(donor_member_id);
CREATE INDEX idx_gift_date ON gift_history(gift_date DESC);
CREATE INDEX idx_gift_form709_pending ON gift_history(form_709_due_date) 
    WHERE requires_gift_tax_return = TRUE AND form_709_filed = FALSE;

COMMENT ON TABLE gift_history IS 'Complete gift transaction history for lifetime exemption tracking';

-- ============================================================================
-- TAX JURISDICTIONS (METADATA-FIRST CONFIGURATION)
-- ============================================================================

CREATE TABLE tax_jurisdictions (
    jurisdiction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    jurisdiction_code VARCHAR(10) NOT NULL, -- 'US-FEDERAL', 'US-CA', 'UK', etc.
    jurisdiction_name TEXT NOT NULL,
    jurisdiction_type VARCHAR(20) NOT NULL, -- 'FEDERAL', 'STATE', 'COUNTRY'
    
    -- Estate Tax Configuration
    estate_tax_applies BOOLEAN DEFAULT TRUE,
    estate_tax_exemption DECIMAL(15,2),
    estate_tax_rate_schedule JSONB,
    /* Example:
    [
        {"threshold": 0, "rate": 0.18},
        {"threshold": 10000, "rate": 0.20},
        {"threshold": 20000, "rate": 0.22}
    ]
    */
    
    -- Gift Tax Configuration
    gift_tax_applies BOOLEAN DEFAULT TRUE,
    annual_gift_exclusion DECIMAL(12,2),
    lifetime_gift_exemption DECIMAL(15,2),
    
    -- Generation-Skipping Transfer Tax
    gst_tax_applies BOOLEAN DEFAULT FALSE,
    gst_tax_exemption DECIMAL(15,2),
    gst_tax_rate DECIMAL(5,4),
    
    -- Effective Dates
    effective_date DATE NOT NULL,
    expiration_date DATE,
    sunset_provisions JSONB, -- Scheduled changes
    
    -- Metadata
    notes TEXT,
    source_url TEXT,
    last_verified_date DATE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_jurisdiction_code_date UNIQUE(jurisdiction_code, effective_date)
);

CREATE INDEX idx_jurisdiction_code ON tax_jurisdictions(jurisdiction_code);
CREATE INDEX idx_jurisdiction_effective ON tax_jurisdictions(effective_date DESC, expiration_date);

COMMENT ON TABLE tax_jurisdictions IS 'Configurable tax rates and exemptions - metadata-first approach';

-- Seed federal tax data (2025 values)
INSERT INTO tax_jurisdictions (jurisdiction_code, jurisdiction_name, jurisdiction_type, estate_tax_exemption, annual_gift_exclusion, lifetime_gift_exemption, gst_tax_exemption, gst_tax_rate, effective_date, estate_tax_rate_schedule) VALUES
('US-FEDERAL', 'United States Federal', 'FEDERAL', 13990000, 18500, 13990000, 13990000, 0.40, '2025-01-01', '[
    {"threshold": 0, "rate": 0.18},
    {"threshold": 10000, "rate": 0.20},
    {"threshold": 20000, "rate": 0.22},
    {"threshold": 40000, "rate": 0.24},
    {"threshold": 60000, "rate": 0.26},
    {"threshold": 80000, "rate": 0.28},
    {"threshold": 100000, "rate": 0.30},
    {"threshold": 150000, "rate": 0.32},
    {"threshold": 250000, "rate": 0.34},
    {"threshold": 500000, "rate": 0.37},
    {"threshold": 750000, "rate": 0.39},
    {"threshold": 1000000, "rate": 0.40}
]'::jsonb);

-- Seed state tax data (example states with estate tax)
INSERT INTO tax_jurisdictions (jurisdiction_code, jurisdiction_name, jurisdiction_type, estate_tax_exemption, annual_gift_exclusion, effective_date, estate_tax_rate_schedule) VALUES
('US-CA', 'California', 'STATE', 0, 0, '2025-01-01', '[]'::jsonb), -- No estate tax
('US-NY', 'New York', 'STATE', 6940000, 0, '2025-01-01', '[{"threshold": 0, "rate": 0.034}, {"threshold": 500000, "rate": 0.05}, {"threshold": 1000000, "rate": 0.065}, {"threshold": 3100000, "rate": 0.10}, {"threshold": 5100000, "rate": 0.112}, {"threshold": 6100000, "rate": 0.144}, {"threshold": 10100000, "rate": 0.16}]'::jsonb),
('US-MA', 'Massachusetts', 'STATE', 2000000, 0, '2025-01-01', '[{"threshold": 0, "rate": 0.08}, {"threshold": 40000, "rate": 0.10}, {"threshold": 140000, "rate": 0.12}, {"threshold": 440000, "rate": 0.14}, {"threshold": 940000, "rate": 0.16}]'::jsonb),
('US-WA', 'Washington', 'STATE', 2193000, 0, '2025-01-01', '[{"threshold": 0, "rate": 0.10}, {"threshold": 1000000, "rate": 0.14}, {"threshold": 2000000, "rate": 0.15}, {"threshold": 3000000, "rate": 0.16}, {"threshold": 4000000, "rate": 0.17}, {"threshold": 6000000, "rate": 0.18}, {"threshold": 7000000, "rate": 0.19}, {"threshold": 9000000, "rate": 0.20}]'::jsonb),
('US-OR', 'Oregon', 'STATE', 1000000, 0, '2025-01-01', '[{"threshold": 0, "rate": 0.10}, {"threshold": 1000000, "rate": 0.105}, {"threshold": 1500000, "rate": 0.11}, {"threshold": 2500000, "rate": 0.115}, {"threshold": 3500000, "rate": 0.12}, {"threshold": 4500000, "rate": 0.13}, {"threshold": 5500000, "rate": 0.14}, {"threshold": 6500000, "rate": 0.15}, {"threshold": 7500000, "rate": 0.16}]'::jsonb);

-- ============================================================================
-- ESTATE PLANNING SCENARIOS
-- ============================================================================

CREATE TABLE estate_plan_scenarios (
    scenario_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES family_offices(family_id) ON DELETE CASCADE,
    
    -- Scenario Identification
    scenario_name TEXT NOT NULL,
    scenario_description TEXT,
    strategy_type strategy_type_enum NOT NULL,
    strategies_used TEXT[],
    
    -- Tax Impact Projections
    baseline_estate_tax DECIMAL(15,2) NOT NULL,
    projected_estate_tax DECIMAL(15,2) NOT NULL,
    tax_savings DECIMAL(15,2) GENERATED ALWAYS AS (baseline_estate_tax - projected_estate_tax) STORED,
    tax_savings_pct DECIMAL(5,2) GENERATED ALWAYS AS (
        CASE 
            WHEN baseline_estate_tax > 0 THEN ((baseline_estate_tax - projected_estate_tax) / baseline_estate_tax) * 100
            ELSE 0
        END
    ) STORED,
    
    -- Wealth Transfer Projections
    baseline_net_to_heirs DECIMAL(18,2) NOT NULL,
    projected_net_to_heirs DECIMAL(18,2) NOT NULL,
    additional_wealth_transferred DECIMAL(18,2) GENERATED ALWAYS AS (
        projected_net_to_heirs - baseline_net_to_heirs
    ) STORED,
    
    -- Multi-Generational Impact
    generation_count INTEGER DEFAULT 1,
    compounded_benefit_30yr DECIMAL(18,2),
    dynasty_trust_perpetual_benefit DECIMAL(18,2),
    
    -- Implementation Complexity
    complexity_score INTEGER NOT NULL CHECK (complexity_score BETWEEN 1 AND 10),
    implementation_time_weeks INTEGER NOT NULL,
    estimated_implementation_cost DECIMAL(12,2),
    annual_maintenance_cost DECIMAL(12,2),
    
    -- Requirements & Prerequisites
    requires_spousal_cooperation BOOLEAN DEFAULT FALSE,
    requires_gift_tax_filing BOOLEAN DEFAULT FALSE,
    requires_appraisal BOOLEAN DEFAULT FALSE,
    requires_life_insurance BOOLEAN DEFAULT FALSE,
    minimum_networth_required DECIMAL(15,2),
    
    -- Structures Created
    entities_to_create JSONB,
    /* Example:
    [
        {"type": "SLAT", "funding_amount": 13990000, "beneficiaries": ["child1", "child2"]},
        {"type": "ILIT", "insurance_amount": 5000000, "beneficiaries": ["all_children"]}
    ]
    */
    
    -- Gifting Strategy
    annual_gifts_total DECIMAL(12,2),
    lifetime_exemption_utilized DECIMAL(15,2),
    gst_exemption_utilized DECIMAL(15,2),
    
    -- Risk Assessment
    irs_audit_risk audit_risk_level,
    valuation_challenge_risk audit_risk_level,
    legislative_change_risk audit_risk_level,
    
    -- ML Confidence Score
    confidence_score DECIMAL(3,2) CHECK (confidence_score BETWEEN 0 AND 1),
    confidence_factors JSONB,
    
    -- Client Suitability
    suitable_for_risk_tolerance TEXT[],
    suitable_for_age_range JSONB, -- {min, max}
    suitable_for_networth_range JSONB, -- {min, max}
    
    -- Assumptions
    assumed_growth_rate DECIMAL(5,4) DEFAULT 0.07,
    assumed_tax_law_changes JSONB,
    assumed_life_expectancy INTEGER,
    
    -- Ranking
    rank_by_tax_savings INTEGER,
    rank_by_simplicity INTEGER,
    rank_by_overall_score INTEGER,
    
    -- Narrative (AI-generated)
    narrative_explanation TEXT,
    implementation_checklist JSONB,
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX idx_scenario_family ON estate_plan_scenarios(family_id);
CREATE INDEX idx_scenario_strategy ON estate_plan_scenarios(strategy_type);
CREATE INDEX idx_scenario_savings ON estate_plan_scenarios(tax_savings DESC);
CREATE INDEX idx_scenario_confidence ON estate_plan_scenarios(confidence_score DESC);
CREATE INDEX idx_scenario_rank ON estate_plan_scenarios(family_id, rank_by_overall_score);

COMMENT ON TABLE estate_plan_scenarios IS 'AI-generated estate planning scenarios with tax impact analysis';

-- ============================================================================
-- TRIGGERS FOR DENORMALIZED DATA
-- ============================================================================

-- Trigger to update family aggregates when member data changes
CREATE OR REPLACE FUNCTION update_family_aggregates()
RETURNS TRIGGER AS $$
BEGIN
    -- Recalculate family totals
    UPDATE family_offices fo
    SET 
        total_estimated_networth = (
            SELECT COALESCE(SUM(fm.separate_networth), 0)
            FROM family_members fm
            WHERE fm.family_id = COALESCE(NEW.family_id, OLD.family_id)
              AND fm.deleted_at IS NULL
        ),
        generation_count = (
            SELECT COALESCE(MAX(fm.generation), 1)
            FROM family_members fm
            WHERE fm.family_id = COALESCE(NEW.family_id, OLD.family_id)
              AND fm.deleted_at IS NULL
        ),
        updated_at = NOW()
    WHERE fo.family_id = COALESCE(NEW.family_id, OLD.family_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_family_aggregates
AFTER INSERT OR UPDATE OR DELETE ON family_members
FOR EACH ROW 
EXECUTE FUNCTION update_family_aggregates();

-- Trigger to update entity total value when assets change
CREATE OR REPLACE FUNCTION update_entity_value()
RETURNS TRIGGER AS $$
DECLARE
    entity_record RECORD;
BEGIN
    -- Recalculate for all entities that own this asset
    FOR entity_record IN
        SELECT DISTINCT (ownership->>'owner_id')::UUID as entity_id
        FROM family_assets fa,
        jsonb_array_elements(fa.ownership_structure) as ownership
        WHERE fa.asset_id = COALESCE(NEW.asset_id, OLD.asset_id)
          AND ownership->>'owner_type' = 'TRUST'
    LOOP
        UPDATE estate_entities ee
        SET current_total_value = (
            SELECT COALESCE(SUM(
                fa.current_valuation * (ownership->>'ownership_pct')::DECIMAL / 100.0
            ), 0)
            FROM family_assets fa,
            jsonb_array_elements(fa.ownership_structure) as ownership
            WHERE ownership->>'owner_id' = entity_record.entity_id::TEXT
              AND ownership->>'owner_type' = 'TRUST'
              AND fa.deleted_at IS NULL
        ),
        updated_at = NOW()
        WHERE ee.entity_id = entity_record.entity_id;
    END LOOP;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_entity_value
AFTER INSERT OR UPDATE OR DELETE ON family_assets
FOR EACH ROW 
EXECUTE FUNCTION update_entity_value();

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to get assets by owner (individual or trust)
CREATE OR REPLACE FUNCTION get_assets_by_owner(
    p_owner_id UUID,
    p_owner_type TEXT
)
RETURNS TABLE(
    asset_id UUID,
    asset_name TEXT,
    asset_class asset_class_enum,
    owned_value DECIMAL(18,2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        fa.asset_id,
        fa.asset_name,
        fa.asset_class,
        (fa.current_valuation * (ownership->>'ownership_pct')::DECIMAL / 100.0) as owned_value
    FROM family_assets fa,
    jsonb_array_elements(fa.ownership_structure) as ownership
    WHERE ownership->>'owner_id' = p_owner_id::TEXT
      AND ownership->>'owner_type' = p_owner_type
      AND fa.deleted_at IS NULL;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate lifetime exemption used
CREATE OR REPLACE FUNCTION calculate_lifetime_exemption_used(
    p_family_id UUID,
    p_member_id UUID
)
RETURNS DECIMAL(15,2) AS $$
DECLARE
    total_used DECIMAL(15,2);
BEGIN
    SELECT COALESCE(SUM(lifetime_exemption_utilized), 0)
    INTO total_used
    FROM gift_history
    WHERE family_id = p_family_id
      AND donor_member_id = p_member_id
      AND deleted_at IS NULL;
    
    RETURN total_used;
END;
$$ LANGUAGE plpgsql;

-- Function to check if gift requires Form 709
CREATE OR REPLACE FUNCTION requires_form_709(
    p_gift_value DECIMAL(18,2),
    p_annual_exclusion_used DECIMAL(12,2),
    p_gift_type gift_type_enum
)
RETURNS BOOLEAN AS $$
BEGIN
    -- Form 709 required if:
    -- 1. Gift exceeds annual exclusion
    -- 2. Gift uses lifetime exemption
    -- 3. Gift is generation-skipping
    -- 4. Spousal split election
    
    IF p_annual_exclusion_used > 0 AND p_gift_value > p_annual_exclusion_used THEN
        RETURN TRUE;
    END IF;
    
    IF p_gift_type IN ('LIFETIME_EXEMPTION', 'GENERATION_SKIPPING_TRANSFER') THEN
        RETURN TRUE;
    END IF;
    
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- ROW-LEVEL SECURITY FOR MULTI-TENANCY
-- ============================================================================

-- Enable RLS on all tables
ALTER TABLE family_offices ENABLE ROW LEVEL SECURITY;
ALTER TABLE family_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE family_assets ENABLE ROW LEVEL SECURITY;
ALTER TABLE estate_entities ENABLE ROW LEVEL SECURITY;
ALTER TABLE gift_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE estate_plan_scenarios ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only access their tenant's data
CREATE POLICY tenant_isolation_family_offices ON family_offices
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- Similar policies for other tables (simplified - check family's tenant)
CREATE POLICY tenant_isolation_family_members ON family_members
FOR ALL
USING (
    family_id IN (
        SELECT family_id FROM family_offices 
        WHERE tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID
    )
);

CREATE POLICY tenant_isolation_family_assets ON family_assets
FOR ALL
USING (
    family_id IN (
        SELECT family_id FROM family_offices 
        WHERE tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID
    )
);

CREATE POLICY tenant_isolation_estate_entities ON estate_entities
FOR ALL
USING (
    family_id IN (
        SELECT family_id FROM family_offices 
        WHERE tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID
    )
);

CREATE POLICY tenant_isolation_gift_history ON gift_history
FOR ALL
USING (
    family_id IN (
        SELECT family_id FROM family_offices 
        WHERE tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID
    )
);

CREATE POLICY tenant_isolation_estate_plan_scenarios ON estate_plan_scenarios
FOR ALL
USING (
    family_id IN (
        SELECT family_id FROM family_offices 
        WHERE tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID
    )
);

-- ============================================================================
-- SAMPLE DATA (FOR TESTING)
-- ============================================================================

-- Sample family office
INSERT INTO family_offices (family_id, tenant_id, family_name, total_estimated_networth, primary_state_residency, estate_plan_status) 
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Smith Family Office', 25000000, 'CA', 'NOT_STARTED');

-- Sample family members
INSERT INTO family_members (member_id, family_id, legal_first_name, legal_last_name, date_of_birth, generation, primary_state_residency, domicile_state, separate_networth)
VALUES 
('00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000001', 'John', 'Smith', '1954-05-15', 1, 'CA', 'CA', 15000000),
('00000000-0000-0000-0001-000000000002', '00000000-0000-0000-0000-000000000001', 'Mary', 'Smith', '1956-08-22', 1, 'CA', 'CA', 10000000),
('00000000-0000-0000-0001-000000000003', '00000000-0000-0000-0000-000000000001', 'Robert', 'Smith', '1980-03-10', 2, 'NY', 'NY', 0),
('00000000-0000-0000-0001-000000000004', '00000000-0000-0000-0000-000000000001', 'Jennifer', 'Smith', '1982-11-05', 2, 'CA', 'CA', 0);

-- Migration complete
COMMENT ON SCHEMA public IS 'Wealth Transfer Platform v1.0 - Core schema deployed';
