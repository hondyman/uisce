-- Migration 038: Advanced Security & Session Management
-- Biometric auth, device recognition, and GDPR consent

-- =============================================================================
-- 1. BIOMETRIC AUTHENTICATION
-- =============================================================================

CREATE TYPE biometric_auth_type AS ENUM ('FACE_ID', 'FINGERPRINT', 'VOICE', 'PIN');

CREATE TABLE client_biometric_auth (
    auth_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Biometric type
    auth_type biometric_auth_type NOT NULL,
    device_id TEXT NOT NULL,
    device_name TEXT,
    
    -- WebAuthn credential (for biometric)
    credential_id TEXT,
    public_key TEXT,
    
    -- Enrollment
    enrolled_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    use_count INTEGER DEFAULT 0,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT,
    
    UNIQUE (client_id, device_id, auth_type),
    INDEX idx_bio_auth_client (client_id, is_active)
);

-- =============================================================================
-- 2. CLIENT SESSIONS
-- =============================================================================

CREATE TABLE client_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Session details
    session_token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE,
    
    -- Device fingerprint
    device_fingerprint TEXT,
    device_name TEXT,
    device_type VARCHAR(20), -- 'DESKTOP', 'MOBILE', 'TABLET'
    
    -- Browser info
    browser VARCHAR(100),
    browser_version VARCHAR(50),
    os VARCHAR(100),
    os_version VARCHAR(50),
    
    -- Location
    ip_address VARCHAR(45),
    location_city TEXT,
    location_region TEXT,
    location_country VARCHAR(2),
    
    -- Security flags
    is_trusted_device BOOLEAN DEFAULT FALSE,
    requires_2fa BOOLEAN DEFAULT FALSE,
    passed_2fa BOOLEAN DEFAULT FALSE,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '30 days',
    
    -- Termination
    terminated_at TIMESTAMPTZ,
    termination_reason VARCHAR(50), -- 'LOGOUT', 'TIMEOUT', 'FORCED', 'SUSPICIOUS'
    
    INDEX idx_sessions_client_active (client_id, is_active, last_activity_at DESC),
    INDEX idx_sessions_token (session_token),
    INDEX idx_sessions_expiry (expires_at) WHERE is_active = TRUE
);

-- =============================================================================
-- 3. SECURITY EVENTS
-- =============================================================================

CREATE TYPE security_event_type AS ENUM (
    'LOGIN_SUCCESS', 'LOGIN_FAILURE', 'LOGOUT',
    'PASSWORD_CHANGE', 'EMAIL_CHANGE',
    '2FA_ENABLED', '2FA_DISABLED', '2FA_VERIFIED',
    'SUSPICIOUS_ACTIVITY', 'DEVICE_ADDED', 'DEVICE_REMOVED',
    'SESSION_TERMINATED', 'PASSWORD_RESET_REQUEST'
);

CREATE TABLE security_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    event_type security_event_type NOT NULL,
    event_details JSONB DEFAULT '{}',
    
    -- Context
    session_id UUID REFERENCES client_sessions(session_id),
    ip_address VARCHAR(45),
    device_fingerprint TEXT,
    
    -- Risk assessment
    risk_score DECIMAL(3,2), -- 0.00 to 1.00
    flagged_as_suspicious BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_security_events_client (client_id, created_at DESC),
    INDEX idx_security_events_suspicious (flagged_as_suspicious, created_at DESC) WHERE flagged_as_suspicious = TRUE
);

-- =============================================================================
-- 4. GDPR CONSENT MANAGEMENT
-- =============================================================================

CREATE TYPE consent_type AS ENUM (
    'TERMS_OF_SERVICE',
    'PRIVACY_POLICY',
    'DATA_SHARING',
    'MARKETING_EMAIL',
    'MARKETING_SMS',
    'ANALYTICS',
    'THIRD_PARTY_SHARING'
);

CREATE TABLE client_consent (
    consent_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    consent_type consent_type NOT NULL,
    consent_version VARCHAR(20) NOT NULL, -- e.g., "v2.1.0"
    
    -- Consent status
    consent_given BOOLEAN NOT NULL,
    
    -- Timestamps
    granted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    
    -- Compliance (GDPR Article 7)
    ip_address VARCHAR(45),
    user_agent TEXT,
    consent_method VARCHAR(50), -- 'CHECKBOX', 'SIGNATURE', 'VERBAL', 'IMPLICIT'
    
    -- Verification
    verified BOOLEAN DEFAULT FALSE,
    verification_method VARCHAR(50),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_consent_client_type (client_id, consent_type),
    INDEX idx_consent_active (client_id, consent_type, consent_given) WHERE consent_given = TRUE AND revoked_at IS NULL
);

-- =============================================================================
-- 5. TRUSTED DEVICES
-- =============================================================================

CREATE TABLE trusted_devices (
    device_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    
    device_fingerprint TEXT UNIQUE NOT NULL,
    device_name TEXT,
    device_type VARCHAR(20),
    
    -- Trust
    trusted_at TIMESTAMPTZ DEFAULT NOW(),
    trust_expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '90 days',
    
    -- Usage
    last_used_at TIMESTAMPTZ,
    use_count INTEGER DEFAULT 0,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    revoked_at TIMESTAMPTZ,
    
    INDEX idx_trusted_devices_client (client_id, is_active)
);

-- =============================================================================
-- 6. HELPER FUNCTIONS
-- =============================================================================

-- Create new session
CREATE OR REPLACE FUNCTION create_client_session(
    p_client_id UUID,
    p_tenant_id UUID,
    p_device_fingerprint TEXT,
    p_ip_address VARCHAR
) RETURNS UUID AS $$
DECLARE
    v_session_id UUID;
    v_is_trusted BOOLEAN;
BEGIN
    -- Check if device is trusted
    SELECT is_active INTO v_is_trusted
    FROM trusted_devices
    WHERE client_id = p_client_id
    AND device_fingerprint = p_device_fingerprint
    AND trust_expires_at > NOW()
    LIMIT 1;
    
    -- Create session
    INSERT INTO client_sessions (
        client_id, tenant_id, session_token, refresh_token,
        device_fingerprint, ip_address, is_trusted_device
    ) VALUES (
        p_client_id, p_tenant_id,
        encode(gen_random_bytes(32), 'hex'),
        encode(gen_random_bytes(32), 'hex'),
        p_device_fingerprint, p_ip_address,
        COALESCE(v_is_trusted, FALSE)
    )
    RETURNING session_id INTO v_session_id;
    
    -- Log security event
    INSERT INTO security_events (
        client_id, tenant_id, event_type, session_id, ip_address, device_fingerprint
    ) VALUES (
        p_client_id, p_tenant_id, 'LOGIN_SUCCESS', v_session_id, p_ip_address, p_device_fingerprint
    );
    
    RETURN v_session_id;
END;
$$ LANGUAGE plpgsql;

-- Terminate session
CREATE OR REPLACE FUNCTION terminate_session(p_session_id UUID, p_reason VARCHAR)
RETURNS VOID AS $$
BEGIN
    UPDATE client_sessions
    SET is_active = FALSE,
        terminated_at = NOW(),
        termination_reason = p_reason
    WHERE session_id = p_session_id;
    
    INSERT INTO security_events (
        client_id, tenant_id, event_type, event_details, session_id
    )
    SELECT 
        client_id, tenant_id, 'SESSION_TERMINATED',
        jsonb_build_object('reason', p_reason),
        session_id
    FROM client_sessions
    WHERE session_id = p_session_id;
END;
$$ LANGUAGE plpgsql;

-- Detect suspicious activity
CREATE OR REPLACE FUNCTION detect_suspicious_activity(p_client_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_failed_logins INTEGER;
    v_different_locations INTEGER;
BEGIN
    -- Check failed login attempts in last hour
    SELECT COUNT(*) INTO v_failed_logins
    FROM security_events
    WHERE client_id = p_client_id
    AND event_type = 'LOGIN_FAILURE'
    AND created_at > NOW() - INTERVAL '1 hour';
    
    IF v_failed_logins >= 5 THEN
        RETURN TRUE;
    END IF;
    
    -- Check logins from different countries in last 24 hours
    SELECT COUNT(DISTINCT COALESCE((event_details->>'location_country')::TEXT, '')) INTO v_different_locations
    FROM security_events
    WHERE client_id = p_client_id
    AND event_type = 'LOGIN_SUCCESS'
    AND created_at > NOW() - INTERVAL '24 hours';
    
    IF v_different_locations >= 3 THEN
        RETURN TRUE;
    END IF;
    
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 7. TRIGGERS
-- =============================================================================

-- Auto-expire sessions
CREATE OR REPLACE FUNCTION expire_old_sessions() RETURNS VOID AS $$
BEGIN
    UPDATE client_sessions
    SET is_active = FALSE,
        terminated_at = NOW(),
        termination_reason = 'TIMEOUT'
    WHERE is_active = TRUE
    AND expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 8. RLS POLICIES
-- =============================================================================

ALTER TABLE client_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE security_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_consent ENABLE ROW LEVEL SECURITY;

CREATE POLICY sessions_tenant_isolation ON client_sessions
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

CREATE POLICY security_events_tenant_isolation ON security_events
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

CREATE POLICY consent_tenant_isolation ON client_consent
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 9. COMMENTS
-- =============================================================================

COMMENT ON TABLE client_sessions IS 'Client sessions with device fingerprinting and security tracking';
COMMENT ON TABLE security_events IS 'Comprehensive security event log for compliance and threat detection';
COMMENT ON TABLE client_consent IS 'GDPR-compliant consent management with versioning';
COMMENT ON FUNCTION create_client_session IS 'Create new session with automatic trusted device detection';
COMMENT ON FUNCTION detect_suspicious_activity IS 'Detect suspicious login patterns for security alerts';
