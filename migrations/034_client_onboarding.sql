-- Migration 034: Client Onboarding System
-- Guided digital onboarding with progress tracking and document management

-- =============================================================================
-- 1. ENUM TYPES
-- =============================================================================

CREATE TYPE onboarding_status AS ENUM (
    'NOT_STARTED',
    'IN_PROGRESS',
    'PENDING_REVIEW',
    'APPROVED',
    'REJECTED',
    'COMPLETED',
    'ABANDONED'
);

CREATE TYPE onboarding_step AS ENUM (
    'PERSONAL_INFO',
    'FINANCIAL_GOALS',
    'RISK_ASSESSMENT',
    'DOCUMENT_UPLOAD',
    'ACCOUNT_SELECTION',
    'E_SIGNATURE',
    'FUNDING'
);

CREATE TYPE document_type AS ENUM (
    'DRIVERS_LICENSE',
    'PASSPORT',
    'SSN_CARD',
    'PROOF_OF_ADDRESS',
    'TAX_RETURN',
    'BANK_STATEMENT',
    'INVESTMENT_STATEMENT',
    'OTHER'
);

CREATE TYPE document_status AS ENUM (
    'PENDING_UPLOAD',
    'UPLOADED',
    'PROCESSING',
    'VERIFIED',
    'REJECTED',
    'EXPIRED'
);

-- =============================================================================
-- 2. ONBOARDING SESSIONS
-- =============================================================================

CREATE TABLE onboarding_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Client identification (email until account created)
    email VARCHAR(255) NOT NULL,
    client_id UUID REFERENCES clients(client_id), -- NULL until account created
    
    -- Session tracking
    resume_token UUID UNIQUE DEFAULT gen_random_uuid(),
    current_step onboarding_step DEFAULT 'PERSONAL_INFO',
    completed_steps onboarding_step[] DEFAULT '{}',
    onboarding_status onboarding_status DEFAULT 'IN_PROGRESS',
    
    -- Progress data (JSONB for flexibility)
    personal_info JSONB DEFAULT '{}',
    /* Example:
    {
        "first_name": "John",
        "last_name": "Doe",
        "dob": "1990-01-15",
        "ssn_encrypted": "...",
        "phone": "+1234567890",
        "address": {...}
    }
    */
    
    financial_goals JSONB DEFAULT '[]',
    /* Example:
    [
        {
            "goal_type": "RETIREMENT",
            "target_amount": 2000000,
            "target_date": "2050-01-01",
            "priority": 1
        }
    ]
    */
    
    risk_assessment JSONB DEFAULT '{}',
    /* Example:
    {
        "questionnaire_responses": [...],
        "risk_score": 7.5,
        "risk_category": "MODERATE_AGGRESSIVE",
        "recommended_allocation": {...}
    }
    */
    
    account_selection JSONB DEFAULT '{}',
    /* Example:
    {
        "account_type": "INDIVIDUAL_BROKERAGE",
        "funding_method": "ACH_TRANSFER",
        "initial_investment": 50000
    }
    */
    
    -- Tracking
    started_at TIMESTAMPTZ DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    abandoned_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '30 days',
    
    -- Referral source
    referral_source VARCHAR(100),
    utm_params JSONB,
    
    INDEX idx_onboarding_email (email),
    INDEX idx_onboarding_token (resume_token),
    INDEX idx_onboarding_status (onboarding_status, tenant_id)
);

ALTER TABLE onboarding_sessions ENABLE ROW LEVEL SECURITY;

CREATE POLICY onboarding_tenant_isolation ON onboarding_sessions
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 3. DOCUMENT UPLOADS
-- =============================================================================

CREATE TABLE onboarding_documents (
    document_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES onboarding_sessions(session_id) ON DELETE CASCADE,
    
    -- Document details
    document_type document_type NOT NULL,
    document_name TEXT NOT NULL,
    file_size_bytes BIGINT,
    mime_type VARCHAR(100),
    
    -- Storage
    storage_path TEXT NOT NULL, -- S3/GCS path
    storage_bucket VARCHAR(255),
    
    -- OCR extraction
    ocr_text TEXT,
    extracted_data JSONB, -- Structured data from OCR
    /* Example for ID:
    {
        "id_number": "D1234567",
        "full_name": "John Doe",
        "dob": "1990-01-15",
        "address": "123 Main St",
        "expiration_date": "2030-01-15"
    }
    */
    
    -- Verification
    document_status document_status DEFAULT 'UPLOADED',
    verified_at TIMESTAMPTZ,
    verified_by UUID REFERENCES users(user_id),
    rejection_reason TEXT,
    
    -- Audit
    uploaded_at TIMESTAMPTZ DEFAULT NOW(),
    uploaded_by_ip VARCHAR(45),
    
    INDEX idx_docs_session (session_id),
    INDEX idx_docs_type (document_type),
    INDEX idx_docs_status (document_status)
);

ALTER TABLE onboarding_documents ENABLE ROW LEVEL SECURITY;

CREATE POLICY documents_via_session ON onboarding_documents
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM onboarding_sessions
            WHERE onboarding_sessions.session_id = onboarding_documents.session_id
            AND onboarding_sessions.tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID
        )
    );

-- =============================================================================
-- 4. E-SIGNATURE TRACKING
-- =============================================================================

CREATE TABLE onboarding_signatures (
    signature_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES onboarding_sessions(session_id) ON DELETE CASCADE,
    
    -- Document signed
    document_name TEXT NOT NULL,
    document_type VARCHAR(100), -- 'TERMS_OF_SERVICE', 'ACCOUNT_AGREEMENT', 'DISCLOSURE'
    
    -- E-signature provider
    provider VARCHAR(50), -- 'DOCUSIGN', 'ADOBE_SIGN', 'INTERNAL'
    provider_envelope_id TEXT,
    
    -- Signature details
    signed_at TIMESTAMPTZ,
    signer_name TEXT,
    signer_email VARCHAR(255),
    signer_ip VARCHAR(45),
    
    -- Document storage
    signed_document_path TEXT,
    
    -- Status
    signature_status VARCHAR(20) DEFAULT 'PENDING', -- 'PENDING', 'SENT', 'VIEWED', 'SIGNED', 'DECLINED'
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_signatures_session (session_id),
    INDEX idx_signatures_status (signature_status)
);

-- =============================================================================
-- 5. VALIDATION ERRORS
-- =============================================================================

CREATE TABLE onboarding_validation_errors (
    error_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES onboarding_sessions(session_id) ON DELETE CASCADE,
    
    -- Error details
    step onboarding_step NOT NULL,
    field_name TEXT,
    error_type VARCHAR(50), -- 'REQUIRED', 'INVALID_FORMAT', 'OUT_OF_RANGE', etc.
    error_message TEXT NOT NULL,
    
    -- Resolution
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_errors_session (session_id, resolved)
);

-- =============================================================================
-- 6. HELPER FUNCTIONS
-- =============================================================================

-- Update session activity timestamp
CREATE OR REPLACE FUNCTION update_onboarding_activity() RETURNS TRIGGER AS $$
BEGIN
    NEW.last_activity_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER onboarding_activity_trigger
BEFORE UPDATE ON onboarding_sessions
FOR EACH ROW
EXECUTE FUNCTION update_onboarding_activity();

-- Mark abandoned sessions
CREATE OR REPLACE FUNCTION mark_abandoned_onboarding_sessions() RETURNS void AS $$
BEGIN
    UPDATE onboarding_sessions
    SET onboarding_status = 'ABANDONED',
        abandoned_at = NOW()
    WHERE onboarding_status = 'IN_PROGRESS'
    AND last_activity_at < NOW() - INTERVAL '7 days'
    AND abandoned_at IS NULL;
END;
$$ LANGUAGE plpgsql;

-- Calculate onboarding completion percentage
CREATE OR REPLACE FUNCTION calculate_onboarding_progress(p_session_id UUID)
RETURNS INTEGER AS $$
DECLARE
    v_total_steps INTEGER := 7; -- Total steps in onboarding
    v_completed_steps INTEGER;
BEGIN
    SELECT array_length(completed_steps, 1)
    INTO v_completed_steps
    FROM onboarding_sessions
    WHERE session_id = p_session_id;
    
    IF v_completed_steps IS NULL THEN
        v_completed_steps := 0;
    END IF;
    
    RETURN (v_completed_steps * 100 / v_total_steps);
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- 7. INDEXES FOR ANALYTICS
-- =============================================================================

-- Conversion funnel analysis
CREATE INDEX idx_onboarding_funnel ON onboarding_sessions(current_step, onboarding_status, tenant_id);

-- Abandonment analysis
CREATE INDEX idx_onboarding_abandonment ON onboarding_sessions(last_activity_at, onboarding_status)
WHERE onboarding_status = 'IN_PROGRESS';

-- =============================================================================
-- 8. COMMENTS
-- =============================================================================

COMMENT ON TABLE onboarding_sessions IS 'Client onboarding sessions with progress tracking and resume capability';
COMMENT ON TABLE onboarding_documents IS 'Document uploads with OCR extraction and verification status';
COMMENT ON TABLE onboarding_signatures IS 'E-signature tracking for account agreements and disclosures';
COMMENT ON FUNCTION mark_abandoned_onboarding_sessions IS 'Automatically mark sessions abandoned after 7 days of inactivity';
COMMENT ON FUNCTION calculate_onboarding_progress IS 'Calculate onboarding completion percentage (0-100)';
