-- Migration 039: Self-Service Account Management
-- Client self-service for account updates, documents, and transactions

-- =============================================================================
-- 1. ACCOUNT UPDATE REQUESTS
-- =============================================================================

CREATE TYPE update_request_type AS ENUM (
    'CONTACT_INFO',
    'BENEFICIARY',
    'COMMUNICATION_PREFERENCES',
    'INVESTMENT_PREFERENCES',
    'AUTHORIZED_USER',
    'ACCOUNT_RESTRICTION'
);

CREATE TYPE request_status AS ENUM ('PENDING', 'APPROVED', 'REJECTED', 'COMPLETED');

CREATE TABLE account_update_requests (
    request_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Request details
    request_type update_request_type NOT NULL,
    current_values JSONB,
    requested_values JSONB NOT NULL,
    
    -- Justification
    reason TEXT,
    
    -- Status
    request_status request_status DEFAULT 'PENDING',
    
    -- Approval workflow
    requires_approval BOOLEAN DEFAULT TRUE,
    approved_by UUID REFERENCES users(user_id),
    approved_at TIMESTAMPTZ,
    rejection_reason TEXT,
    
    -- Completion
    completed_at TIMESTAMPTZ,
    applied_by UUID REFERENCES users(user_id),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_update_requests_client (client_id, request_status),
    INDEX idx_update_requests_pending (request_status, created_at) WHERE request_status = 'PENDING'
);

-- =============================================================================
-- 2. DOCUMENT SIGNATURE REQUESTS
-- =============================================================================

CREATE TYPE signature_request_status AS ENUM ('SENT', 'VIEWED', 'SIGNED', 'DECLINED', 'EXPIRED');

CREATE TABLE signature_requests (
    request_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Document details
    document_type VARCHAR(50) NOT NULL, -- 'BENEFICIARY_UPDATE', 'ACCOUNT_CHANGE', 'IPS_AGREEMENT', etc.
    document_name TEXT NOT NULL,
    document_description TEXT,
    
    -- E-signature provider
    provider VARCHAR(50) DEFAULT 'DOCUSIGN', -- 'DOCUSIGN', 'ADOBE_SIGN', 'INTERNAL'
    provider_envelope_id TEXT,
    provider_document_id TEXT,
    
    -- Signature details
    sent_at TIMESTAMPTZ DEFAULT NOW(),
    viewed_at TIMESTAMPTZ,
    signed_at TIMESTAMPTZ,
    declined_at TIMESTAMPTZ,
    
    -- Signer info
    signer_name TEXT,
    signer_email VARCHAR(255),
    signer_ip VARCHAR(45),
    
    -- Document storage
    unsigned_document_url TEXT,
    signed_document_url TEXT,
    
    -- Status
    signature_status signature_request_status DEFAULT 'SENT',
    expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '30 days',
    
    -- Related
    related_request_id UUID REFERENCES account_update_requests(request_id),
    
    INDEX idx_signature_requests_client (client_id, signature_status),
    INDEX idx_signature_requests_pending (signature_status, expires_at) WHERE signature_status IN ('SENT', 'VIEWED')
);

-- =============================================================================
-- 3. TRANSACTION REQUESTS (Client-Initiated)
-- =============================================================================

CREATE TYPE transaction_request_type AS ENUM (
    'ONE_TIME_CONTRIBUTION',
    'RECURRING_CONTRIBUTION',
    'DISTRIBUTION',
    'TRANSFER_IN',
    'TRANSFER_OUT',
    'REBALANCE'
);

CREATE TYPE transaction_request_status AS ENUM (
    'DRAFT',
    'SUBMITTED',
    'PENDING_APPROVAL',
    'APPROVED',
    'IN_PROGRESS',
    'COMPLETED',
    'CANCELLED',
    'REJECTED'
);

CREATE TABLE transaction_requests (
    request_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    account_id UUID, -- Can reference portfolio accounts
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Transaction details
    transaction_type transaction_request_type NOT NULL,
    amount DECIMAL(15,2),
    
    -- Transaction data
    transaction_data JSONB NOT NULL,
    /* Example for CONTRIBUTION:
    {
        "funding_source": "BANK_ACCOUNT",
        "bank_account_id": "uuid",
        "effective_date": "2025-01-15",
        "allocation": {...}
    }
    */
    /* Example for RECURRING:
    {
        "frequency": "MONTHLY",
        "day_of_month": 15,
        "start_date": "2025-01-15",
        "end_date": "2030-01-15",
        "amount_per_contribution": 5000
    }
    */
    
    -- Scheduling
    requested_execution_date DATE,
    actual_execution_date DATE,
    
    -- Status
    transaction_status transaction_request_status DEFAULT 'DRAFT',
    
    -- Approval
    requires_approval BOOLEAN DEFAULT TRUE,
    approved_by UUID REFERENCES users(user_id),
    approved_at TIMESTAMPTZ,
    rejection_reason TEXT,
    
    -- Processing
    processed_by UUID REFERENCES users(user_id),
    transaction_id UUID, -- Link to actual executed transaction
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_transaction_requests_client (client_id, transaction_status),
    INDEX idx_transaction_requests_pending (transaction_status, created_at) WHERE transaction_status = 'SUBMITTED'
);

-- =============================================================================
-- 4. LINKED BANK ACCOUNTS
-- =============================================================================

CREATE TABLE client_bank_accounts (
    bank_account_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Bank details
    bank_name TEXT NOT NULL,
    account_type VARCHAR(20), -- 'CHECKING', 'SAVINGS'
    account_number_last4 VARCHAR(4),
    routing_number VARCHAR(9),
    
    -- Plaid integration (for ACH)
    plaid_access_token TEXT,
    plaid_account_id TEXT,
    
    -- Verification
    verified BOOLEAN DEFAULT FALSE,
    verification_method VARCHAR(50), -- 'MICRO_DEPOSITS', 'INSTANT_VERIFY', 'MANUAL'
    verified_at TIMESTAMPTZ,
    
    -- Status
    is_primary BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    
    added_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    
    INDEX idx_bank_accounts_client (client_id, is_active)
);

-- =============================================================================
-- 5. HELPER FUNCTIONS
-- =============================================================================

-- Submit account update request
CREATE OR REPLACE FUNCTION submit_account_update_request(
    p_client_id UUID,
    p_tenant_id UUID,
    p_request_type update_request_type,
    p_current_values JSONB,
    p_requested_values JSONB,
    p_reason TEXT DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_request_id UUID;
    v_requires_approval BOOLEAN;
BEGIN
    -- Determine if approval is required
    v_requires_approval := CASE 
        WHEN p_request_type IN ('BENEFICIARY', 'AUTHORIZED_USER') THEN TRUE
        ELSE FALSE
    END;
    
    INSERT INTO account_update_requests (
        client_id, tenant_id, request_type,
        current_values, requested_values, reason,
        requires_approval
    ) VALUES (
        p_client_id, p_tenant_id, p_request_type,
        p_current_values, p_requested_values, p_reason,
        v_requires_approval
    )
    RETURNING request_id INTO v_request_id;
    
    -- If no approval required, auto-approve
    IF NOT v_requires_approval THEN
        UPDATE account_update_requests
        SET request_status = 'APPROVED',
            approved_at = NOW()
        WHERE request_id = v_request_id;
    END IF;
    
    RETURN v_request_id;
END;
$$ LANGUAGE plpgsql;

-- Process approved update request
CREATE OR REPLACE FUNCTION process_update_request(p_request_id UUID, p_applied_by UUID)
RETURNS VOID AS $$
DECLARE
    v_request RECORD;
BEGIN
    SELECT * INTO v_request FROM account_update_requests WHERE request_id = p_request_id;
    
    IF v_request.request_status != 'APPROVED' THEN
        RAISE EXCEPTION 'Request must be approved before processing';
    END IF;
    
    -- Apply changes based on request type
    -- In real implementation, this would update relevant tables
    
    UPDATE account_update_requests
    SET request_status = 'COMPLETED',
        completed_at = NOW(),
        applied_by = p_applied_by
    WHERE request_id = p_request_id;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 6. TRIGGERS
-- ==============================================================================

CREATE OR REPLACE FUNCTION update_transaction_request_timestamp() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER transaction_request_update_trigger
BEFORE UPDATE ON transaction_requests
FOR EACH ROW
EXECUTE FUNCTION update_transaction_request_timestamp();

-- =============================================================================
-- 7. RLS POLICIES
-- =============================================================================

ALTER TABLE account_update_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE transaction_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_bank_accounts ENABLE ROW LEVEL SECURITY;

CREATE POLICY update_requests_tenant_isolation ON account_update_requests
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

CREATE POLICY transaction_requests_tenant_isolation ON transaction_requests
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

CREATE POLICY bank_accounts_tenant_isolation ON client_bank_accounts
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 8. COMMENTS
-- =============================================================================

COMMENT ON TABLE account_update_requests IS 'Self-service account update requests with approval workflow';
COMMENT ON TABLE signature_requests IS 'E-signature requests for account changes and agreements';
COMMENT ON TABLE transaction_requests IS 'Client-initiated transaction requests (contributions, distributions, transfers)';
COMMENT ON TABLE client_bank_accounts IS 'Linked bank accounts for ACH transfers with Plaid integration';
COMMENT ON FUNCTION submit_account_update_request IS 'Submit account update request with automatic approval for low-risk changes';
