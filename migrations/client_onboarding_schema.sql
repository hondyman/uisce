-- ============================================================================
-- Client Onboarding Database Schema
-- ============================================================================
-- This migration creates the complete data model for wealth management client
-- onboarding including clients, documents, contacts, accounts, portfolios, and
-- workflow tracking. Designed for multi-tenant deployments with ABAC support.
-- ============================================================================

-- ============================================================================
-- 1. CLIENTS TABLE
-- ============================================================================
-- Core client entity with risk profiling and KYC/AML attributes
CREATE TABLE IF NOT EXISTS clients (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  
  -- Basic Information
  first_name VARCHAR(255) NOT NULL,
  last_name VARCHAR(255) NOT NULL,
  email VARCHAR(255) NOT NULL,
  phone_number VARCHAR(20),
  
  -- KYC/AML Information
  identification_number VARCHAR(50) UNIQUE,
  identification_type VARCHAR(50), -- 'SSN', 'EIN', 'PASSPORT', 'DRIVER_LICENSE'
  date_of_birth DATE,
  country_of_citizenship VARCHAR(100),
  tax_residency_country VARCHAR(100),
  
  -- Risk & Wealth Profile
  risk_profile VARCHAR(50) NOT NULL DEFAULT 'moderate', -- 'low', 'moderate', 'high', 'very_high'
  net_worth DECIMAL(18, 2),
  annual_income DECIMAL(18, 2),
  investment_experience VARCHAR(50), -- 'none', 'beginner', 'intermediate', 'advanced'
  
  -- Onboarding Status
  onboarding_status VARCHAR(50) NOT NULL DEFAULT 'pending_validation', 
  -- 'pending_validation', 'pending_review', 'pending_approval', 'pending_agreements', 
  -- 'pending_account_creation', 'pending_notification', 'active', 'rejected', 'suspended'
  onboarding_stage INT DEFAULT 1, -- Current step in 5-step process
  
  -- Workflow Reference
  temporal_workflow_id VARCHAR(500),
  
  -- Assigned Advisor
  assigned_advisor_id UUID,
  
  -- Compliance & Documentation
  kyc_status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'approved', 'rejected', 'escalated'
  aml_status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'approved', 'rejected', 'escalated'
  aml_screening_provider VARCHAR(100), -- 'lexis_nexis', 'worldcheck', 'internal'
  aml_screening_result JSONB, -- Store external API response
  
  -- Legal & Agreements
  agreements_sent_date TIMESTAMP,
  agreements_signed_date TIMESTAMP,
  
  -- Audit Trail
  created_by UUID,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by UUID,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  
  -- Multi-tenancy & Access Control
  is_active BOOLEAN DEFAULT TRUE,
  
  CONSTRAINT clients_tenant_datasource_fk FOREIGN KEY (tenant_id, datasource_id) 
    REFERENCES datasources(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_clients_tenant ON clients(tenant_id);
CREATE INDEX idx_clients_datasource ON clients(datasource_id);
CREATE INDEX idx_clients_onboarding_status ON clients(onboarding_status);
CREATE INDEX idx_clients_risk_profile ON clients(risk_profile);
CREATE INDEX idx_clients_email ON clients(email);
CREATE INDEX idx_clients_kyc_status ON clients(kyc_status);
CREATE INDEX idx_clients_aml_status ON clients(aml_status);


-- ============================================================================
-- 2. DOCUMENTS TABLE
-- ============================================================================
-- Client documentation (ID proofs, proof of address, agreements, etc.)
CREATE TABLE IF NOT EXISTS client_documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  client_id UUID NOT NULL,
  
  -- Document Classification
  document_type VARCHAR(100) NOT NULL, 
  -- 'id_proof', 'proof_of_address', 'proof_of_funds', 'bank_statement', 
  -- 'tax_return', 'employment_letter', 'agreement', 'beneficiary_form'
  document_name VARCHAR(255) NOT NULL,
  
  -- Document Content & Storage
  document_path VARCHAR(500), -- S3/storage path
  file_size BIGINT, -- Size in bytes
  file_type VARCHAR(50), -- 'pdf', 'jpg', 'png', 'docx'
  
  -- Document Status & Verification
  status VARCHAR(50) NOT NULL DEFAULT 'pending_review', -- 'pending_review', 'approved', 'rejected', 'expired'
  verification_status VARCHAR(50) DEFAULT 'unverified', -- 'unverified', 'verified', 'failed'
  verified_by UUID,
  verified_at TIMESTAMP,
  verification_notes TEXT,
  
  -- Expiration & Compliance
  issue_date DATE,
  expiry_date DATE,
  is_expired BOOLEAN GENERATED ALWAYS AS (expiry_date IS NOT NULL AND expiry_date < CURRENT_DATE) STORED,
  
  -- For agreements
  e_signature_request_id VARCHAR(500), -- DocuSign request ID
  e_signature_status VARCHAR(50), -- 'pending', 'sent', 'signed', 'declined'
  signed_at TIMESTAMP,
  signed_by_ip VARCHAR(45), -- IPv4 or IPv6
  
  -- Audit Trail
  created_by UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  
  CONSTRAINT fk_client_documents_client FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE,
  CONSTRAINT fk_client_documents_tenant FOREIGN KEY (tenant_id, datasource_id) 
    REFERENCES datasources(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_client_documents_client ON client_documents(client_id);
CREATE INDEX idx_client_documents_tenant ON client_documents(tenant_id);
CREATE INDEX idx_client_documents_type ON client_documents(document_type);
CREATE INDEX idx_client_documents_status ON client_documents(status);
CREATE INDEX idx_client_documents_verification ON client_documents(verification_status);


-- ============================================================================
-- 3. CONTACTS TABLE
-- ============================================================================
-- Client contacts (emergency contacts, authorized representatives, advisors)
CREATE TABLE IF NOT EXISTS client_contacts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  client_id UUID NOT NULL,
  
  -- Contact Information
  contact_type VARCHAR(50) NOT NULL, 
  -- 'emergency_contact', 'authorized_representative', 'advisor', 'beneficiary', 'accountant'
  first_name VARCHAR(255) NOT NULL,
  last_name VARCHAR(255) NOT NULL,
  email VARCHAR(255),
  phone_number VARCHAR(20),
  
  -- Relationship
  relationship VARCHAR(100), -- 'spouse', 'parent', 'child', 'sibling', 'friend', 'attorney'
  
  -- For Advisors
  employee_id VARCHAR(100),
  department VARCHAR(100),
  is_primary_advisor BOOLEAN DEFAULT FALSE,
  
  -- Permissions & Access
  can_access_accounts BOOLEAN DEFAULT FALSE,
  can_make_trades BOOLEAN DEFAULT FALSE,
  can_request_distributions BOOLEAN DEFAULT FALSE,
  
  -- Audit Trail
  created_by UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  is_active BOOLEAN DEFAULT TRUE,
  
  CONSTRAINT fk_client_contacts_client FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE,
  CONSTRAINT fk_client_contacts_tenant FOREIGN KEY (tenant_id, datasource_id) 
    REFERENCES datasources(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_client_contacts_client ON client_contacts(client_id);
CREATE INDEX idx_client_contacts_tenant ON client_contacts(tenant_id);
CREATE INDEX idx_client_contacts_type ON client_contacts(contact_type);


-- ============================================================================
-- 4. ACCOUNTS TABLE
-- ============================================================================
-- Investment accounts created during onboarding
CREATE TABLE IF NOT EXISTS client_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  client_id UUID NOT NULL,
  
  -- Account Information
  account_number VARCHAR(100) UNIQUE NOT NULL,
  account_type VARCHAR(50) NOT NULL, -- 'brokerage', 'ira', 'sep_ira', 'rollover_ira', 'trust'
  account_title VARCHAR(255),
  
  -- Account Status
  status VARCHAR(50) NOT NULL DEFAULT 'pending_funding',
  -- 'pending_funding', 'active', 'suspended', 'closed'
  
  -- Financial Information
  initial_balance DECIMAL(18, 2) DEFAULT 0,
  current_balance DECIMAL(18, 2) DEFAULT 0,
  currency VARCHAR(3) DEFAULT 'USD',
  
  -- Custodian Information
  custodian_name VARCHAR(255),
  custodian_account_id VARCHAR(100),
  
  -- Account Features
  allows_margin BOOLEAN DEFAULT FALSE,
  allows_options BOOLEAN DEFAULT FALSE,
  allows_cryptocurrency BOOLEAN DEFAULT FALSE,
  
  -- Audit Trail
  created_by UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  funding_date TIMESTAMP,
  
  CONSTRAINT fk_client_accounts_client FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE,
  CONSTRAINT fk_client_accounts_tenant FOREIGN KEY (tenant_id, datasource_id) 
    REFERENCES datasources(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_client_accounts_client ON client_accounts(client_id);
CREATE INDEX idx_client_accounts_tenant ON client_accounts(tenant_id);
CREATE INDEX idx_client_accounts_account_number ON client_accounts(account_number);
CREATE INDEX idx_client_accounts_status ON client_accounts(status);
CREATE INDEX idx_client_accounts_type ON client_accounts(account_type);


-- ============================================================================
-- 5. PORTFOLIOS TABLE
-- ============================================================================
-- Investment portfolios and asset allocation
CREATE TABLE IF NOT EXISTS client_portfolios (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  account_id UUID NOT NULL,
  
  -- Portfolio Information
  portfolio_name VARCHAR(255) NOT NULL,
  portfolio_type VARCHAR(50), -- 'model', 'custom', 'uma', 'tactical'
  status VARCHAR(50) NOT NULL DEFAULT 'active',
  
  -- Asset Allocation (in percentages)
  allocation_json JSONB NOT NULL, -- Store asset class allocations
  -- Example: { "equities": 50, "fixed_income": 30, "alternatives": 10, "cash": 10 }
  
  -- Portfolio Characteristics
  target_return DECIMAL(5, 2), -- Target annual return percentage
  risk_level VARCHAR(50), -- 'conservative', 'moderate', 'aggressive'
  rebalance_frequency VARCHAR(50) DEFAULT 'quarterly', -- 'monthly', 'quarterly', 'semi-annual', 'annual'
  last_rebalance_date DATE,
  next_rebalance_date DATE,
  
  -- Performance Tracking
  inception_date DATE NOT NULL,
  total_market_value DECIMAL(18, 2),
  total_gain_loss DECIMAL(18, 2),
  ytd_return DECIMAL(5, 2),
  
  -- Holdings
  holdings_count INT DEFAULT 0,
  
  -- Audit Trail
  created_by UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by UUID,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  
  CONSTRAINT fk_client_portfolios_account FOREIGN KEY (account_id) REFERENCES client_accounts(id) ON DELETE CASCADE,
  CONSTRAINT fk_client_portfolios_tenant FOREIGN KEY (tenant_id, datasource_id) 
    REFERENCES datasources(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_client_portfolios_account ON client_portfolios(account_id);
CREATE INDEX idx_client_portfolios_tenant ON client_portfolios(tenant_id);
CREATE INDEX idx_client_portfolios_status ON client_portfolios(status);


-- ============================================================================
-- 6. PORTFOLIO HOLDINGS TABLE
-- ============================================================================
-- Individual securities in a portfolio
CREATE TABLE IF NOT EXISTS portfolio_holdings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  portfolio_id UUID NOT NULL,
  
  -- Security Information
  security_id VARCHAR(100) NOT NULL, -- Ticker, CUSIP, or ISIN
  security_name VARCHAR(255) NOT NULL,
  security_type VARCHAR(50), -- 'stock', 'bond', 'etf', 'mutual_fund', 'commodity'
  
  -- Position Information
  quantity DECIMAL(18, 8) NOT NULL,
  unit_price DECIMAL(18, 2) NOT NULL,
  market_value DECIMAL(18, 2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
  
  -- Allocation
  target_allocation DECIMAL(5, 2), -- Percentage of portfolio
  actual_allocation DECIMAL(5, 2), -- Actual percentage
  
  -- Performance
  cost_basis DECIMAL(18, 2),
  gain_loss DECIMAL(18, 2),
  gain_loss_percentage DECIMAL(5, 2),
  
  -- Audit Trail
  added_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  
  CONSTRAINT fk_portfolio_holdings_portfolio FOREIGN KEY (portfolio_id) REFERENCES client_portfolios(id) ON DELETE CASCADE,
  CONSTRAINT fk_portfolio_holdings_tenant FOREIGN KEY (tenant_id, datasource_id) 
    REFERENCES datasources(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_portfolio_holdings_portfolio ON portfolio_holdings(portfolio_id);
CREATE INDEX idx_portfolio_holdings_tenant ON portfolio_holdings(tenant_id);


-- ============================================================================
-- 7. ONBOARDING WORKFLOW STATE TABLE
-- ============================================================================
-- Tracks the state and progress of the onboarding workflow
CREATE TABLE IF NOT EXISTS onboarding_workflows (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  client_id UUID NOT NULL,
  
  -- Workflow Identity
  workflow_id VARCHAR(500) UNIQUE NOT NULL, -- Temporal workflow ID
  
  -- Step Tracking (1-5)
  current_step INT NOT NULL DEFAULT 1,
  step_1_validation_status VARCHAR(50) DEFAULT 'pending',
  step_2_routing_status VARCHAR(50) DEFAULT 'pending',
  step_3_agreements_status VARCHAR(50) DEFAULT 'pending',
  step_4_accounts_status VARCHAR(50) DEFAULT 'pending',
  step_5_notification_status VARCHAR(50) DEFAULT 'pending',
  
  -- Step Completion Times
  step_1_completed_at TIMESTAMP,
  step_2_completed_at TIMESTAMP,
  step_3_completed_at TIMESTAMP,
  step_4_completed_at TIMESTAMP,
  step_5_completed_at TIMESTAMP,
  
  -- Overall Status
  overall_status VARCHAR(50) NOT NULL DEFAULT 'in_progress',
  -- 'in_progress', 'completed', 'failed', 'rejected', 'suspended'
  
  -- Approval & Rejection
  approved_by UUID, -- Advisor who approved
  approved_at TIMESTAMP,
  rejected_by UUID,
  rejected_at TIMESTAMP,
  rejection_reason TEXT,
  
  -- Timeout Escalation
  timeout_escalation_workflow_id VARCHAR(500),
  escalation_status VARCHAR(50), -- 'not_escalated', 'escalated', 'resolved'
  escalation_action VARCHAR(50), -- 'notify', 'escalate', 'auto_approve', 'auto_reject'
  
  -- Workflow Data
  validation_errors JSONB, -- Store validation failures
  workflow_context JSONB, -- Store workflow-specific data
  
  -- Audit Trail
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  
  CONSTRAINT fk_onboarding_workflows_client FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE,
  CONSTRAINT fk_onboarding_workflows_tenant FOREIGN KEY (tenant_id, datasource_id) 
    REFERENCES datasources(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_onboarding_workflows_client ON onboarding_workflows(client_id);
CREATE INDEX idx_onboarding_workflows_tenant ON onboarding_workflows(tenant_id);
CREATE INDEX idx_onboarding_workflows_workflow_id ON onboarding_workflows(workflow_id);
CREATE INDEX idx_onboarding_workflows_status ON onboarding_workflows(overall_status);
CREATE INDEX idx_onboarding_workflows_current_step ON onboarding_workflows(current_step);


-- ============================================================================
-- 8. ONBOARDING WORKFLOW EVENTS TABLE
-- ============================================================================
-- Audit log for all events in the onboarding workflow
CREATE TABLE IF NOT EXISTS onboarding_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  workflow_id UUID NOT NULL, -- References onboarding_workflows.id
  
  -- Event Information
  event_type VARCHAR(100) NOT NULL,
  -- 'validation_started', 'validation_completed', 'validation_failed',
  -- 'routing_started', 'routing_completed', 'review_assigned',
  -- 'agreement_generated', 'agreement_sent', 'agreement_signed',
  -- 'account_created', 'portfolio_created',
  -- 'notification_sent', 'onboarding_completed', 'onboarding_rejected',
  -- 'escalation_triggered', 'escalation_resolved'
  
  event_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  event_data JSONB, -- Flexible storage for event-specific data
  
  -- Actor Information
  triggered_by UUID, -- User or system that triggered event
  actor_type VARCHAR(50), -- 'user', 'system', 'workflow'
  actor_role VARCHAR(50),
  
  -- Step Reference
  step_number INT,
  
  -- Audit Trail
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  
  CONSTRAINT fk_onboarding_events_workflow FOREIGN KEY (workflow_id) REFERENCES onboarding_workflows(id) ON DELETE CASCADE,
  CONSTRAINT fk_onboarding_events_tenant FOREIGN KEY (tenant_id, datasource_id) 
    REFERENCES datasources(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_onboarding_events_workflow ON onboarding_events(workflow_id);
CREATE INDEX idx_onboarding_events_tenant ON onboarding_events(tenant_id);
CREATE INDEX idx_onboarding_events_type ON onboarding_events(event_type);
CREATE INDEX idx_onboarding_events_timestamp ON onboarding_events(event_timestamp);


-- ============================================================================
-- 9. KYC/AML VALIDATION RESULTS TABLE
-- ============================================================================
-- Store detailed KYC/AML screening results for audit trail
CREATE TABLE IF NOT EXISTS kyc_aml_results (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  client_id UUID NOT NULL,
  
  -- Screening Details
  screening_type VARCHAR(50) NOT NULL, -- 'kyc', 'aml'
  screening_provider VARCHAR(100), -- 'lexis_nexis', 'worldcheck', 'internal'
  screening_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  
  -- Results
  status VARCHAR(50) NOT NULL, -- 'pass', 'fail', 'review_required'
  risk_score DECIMAL(5, 2),
  risk_level VARCHAR(50), -- 'low', 'medium', 'high', 'critical'
  
  -- Detailed Findings
  findings JSONB NOT NULL, -- Store detailed results from provider
  matches JSONB, -- Watchlist matches if any
  
  -- Follow-up
  requires_review BOOLEAN DEFAULT FALSE,
  reviewed_by UUID,
  reviewed_at TIMESTAMP,
  review_notes TEXT,
  
  -- Escalation
  escalation_level VARCHAR(50), -- 'none', 'supervisor', 'director', 'compliance_officer'
  escalated_to UUID,
  escalation_reason TEXT,
  
  -- Audit Trail
  created_by UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  
  CONSTRAINT fk_kyc_aml_results_client FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE,
  CONSTRAINT fk_kyc_aml_results_tenant FOREIGN KEY (tenant_id, datasource_id) 
    REFERENCES datasources(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_kyc_aml_results_client ON kyc_aml_results(client_id);
CREATE INDEX idx_kyc_aml_results_tenant ON kyc_aml_results(tenant_id);
CREATE INDEX idx_kyc_aml_results_status ON kyc_aml_results(status);
CREATE INDEX idx_kyc_aml_results_type ON kyc_aml_results(screening_type);


-- ============================================================================
-- 10. COMMENTS/NOTES TABLE
-- ============================================================================
-- Internal notes and comments on client onboarding
CREATE TABLE IF NOT EXISTS onboarding_notes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  client_id UUID NOT NULL,
  
  -- Note Information
  note_type VARCHAR(50), -- 'internal_note', 'compliance_note', 'advisor_note', 'system_message'
  content TEXT NOT NULL,
  
  -- Visibility & Access Control
  is_internal BOOLEAN DEFAULT TRUE,
  visible_to_client BOOLEAN DEFAULT FALSE,
  required_role VARCHAR(100), -- ABAC role requirement
  
  -- Author Information
  created_by UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  
  -- Reference
  related_step INT, -- Which step this note relates to
  related_document_id UUID, -- Link to document if applicable
  
  CONSTRAINT fk_onboarding_notes_client FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE,
  CONSTRAINT fk_onboarding_notes_tenant FOREIGN KEY (tenant_id, datasource_id) 
    REFERENCES datasources(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_onboarding_notes_client ON onboarding_notes(client_id);
CREATE INDEX idx_onboarding_notes_tenant ON onboarding_notes(tenant_id);
CREATE INDEX idx_onboarding_notes_type ON onboarding_notes(note_type);


-- ============================================================================
-- VIEWS FOR COMMON QUERIES
-- ============================================================================

-- Active clients in onboarding
CREATE OR REPLACE VIEW active_onboarding_clients AS
SELECT 
  c.id,
  c.tenant_id,
  c.datasource_id,
  c.first_name,
  c.last_name,
  c.email,
  c.onboarding_status,
  c.onboarding_stage,
  ow.current_step,
  ow.overall_status,
  ow.workflow_id,
  c.risk_profile,
  c.kyc_status,
  c.aml_status,
  c.assigned_advisor_id
FROM clients c
LEFT JOIN onboarding_workflows ow ON c.id = ow.client_id
WHERE c.is_active = TRUE 
  AND c.onboarding_status != 'active'
ORDER BY c.created_at DESC;

-- Client summary view
CREATE OR REPLACE VIEW client_onboarding_summary AS
SELECT 
  c.id,
  c.tenant_id,
  c.first_name || ' ' || c.last_name AS full_name,
  c.email,
  c.onboarding_status,
  c.risk_profile,
  c.net_worth,
  COALESCE(COUNT(DISTINCT ca.id), 0) AS accounts_count,
  COALESCE(SUM(ca.current_balance), 0) AS total_balance,
  c.kyc_status,
  c.aml_status,
  ow.overall_status AS workflow_status,
  c.created_at
FROM clients c
LEFT JOIN client_accounts ca ON c.id = ca.client_id
LEFT JOIN onboarding_workflows ow ON c.id = ow.client_id
GROUP BY c.id, ow.id;

-- Pending approvals view
CREATE OR REPLACE VIEW pending_advisor_approvals AS
SELECT 
  c.id AS client_id,
  c.tenant_id,
  c.first_name || ' ' || c.last_name AS client_name,
  c.email,
  c.assigned_advisor_id,
  c.risk_profile,
  c.net_worth,
  ow.workflow_id,
  ow.current_step,
  ow.updated_at AS last_update
FROM clients c
LEFT JOIN onboarding_workflows ow ON c.id = ow.client_id
WHERE c.onboarding_status = 'pending_review'
  AND c.is_active = TRUE
ORDER BY ow.updated_at ASC;

-- Grant appropriate permissions (adjust as needed for your setup)
-- GRANT SELECT ON active_onboarding_clients TO app_user;
-- GRANT SELECT ON client_onboarding_summary TO app_user;
-- GRANT SELECT ON pending_advisor_approvals TO app_user;
