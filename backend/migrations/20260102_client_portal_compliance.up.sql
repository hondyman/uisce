-- Client Portal & Compliance Database Schema
-- Migration: Add tables for messaging, e-signature, meetings, Form ADV, GIPS, and compliance

-- ==============================================================================
-- CLIENT PORTAL - MESSAGING
-- ==============================================================================

CREATE TABLE IF NOT EXISTS client_messages (
    message_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id UUID NOT NULL,
    family_id UUID NOT NULL,
    sender_id VARCHAR(255) NOT NULL,
    sender_type VARCHAR(20) NOT NULL, -- CLIENT, ADVISOR, SYSTEM
    recipient_id VARCHAR(255) NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    encrypted BOOLEAN DEFAULT TRUE,
    read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP WITH TIME ZONE,
    priority VARCHAR(20) DEFAULT 'NORMAL', -- LOW, NORMAL, HIGH, URGENT
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_message_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_messages_thread') THEN
    CREATE INDEX IF NOT EXISTS idx_messages_thread ON client_messages(thread_id);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_messages_family') THEN
    CREATE INDEX IF NOT EXISTS idx_messages_family ON client_messages(family_id);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_messages_recipient') THEN
    CREATE INDEX IF NOT EXISTS idx_messages_recipient ON client_messages(recipient_id, read);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_messages_created') THEN
    CREATE INDEX IF NOT EXISTS idx_messages_created ON client_messages(created_at DESC);
END IF; END $$;

CREATE TABLE IF NOT EXISTS message_attachments (
    attachment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL,
    file_name VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    storage_path TEXT NOT NULL,
    encrypted BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_attachment_message FOREIGN KEY (message_id) 
        REFERENCES client_messages(message_id) ON DELETE CASCADE
);

DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_attachments_message') THEN
    CREATE INDEX IF NOT EXISTS idx_attachments_message ON message_attachments(message_id);
END IF; END $$;

-- ==============================================================================
-- CLIENT PORTAL - E-SIGNATURE
-- ==============================================================================

CREATE TABLE IF NOT EXISTS signature_requests (
    request_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    document_name VARCHAR(500) NOT NULL,
    document_type VARCHAR(100) NOT NULL, -- IPS, ACCOUNT_AGREEMENT, AMENDMENT
    document_url TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING', -- PENDING, SIGNED, REJECTED, EXPIRED
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_signature_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_signature_family') THEN
    CREATE INDEX IF NOT EXISTS idx_signature_family ON signature_requests(family_id);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_signature_status') THEN
    CREATE INDEX IF NOT EXISTS idx_signature_status ON signature_requests(status);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_signature_expires') THEN
    CREATE INDEX IF NOT EXISTS idx_signature_expires ON signature_requests(expires_at);
END IF; END $$;

CREATE TABLE IF NOT EXISTS signature_signers (
    signer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID NOT NULL,
    member_id UUID,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    signing_order INT NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING', -- PENDING, SIGNED, DECLINED
    signed_at TIMESTAMP WITH TIME ZONE,
    ip_address VARCHAR(45),
    signature_data TEXT, -- Base64 signature image
    CONSTRAINT fk_signer_request FOREIGN KEY (request_id) 
        REFERENCES signature_requests(request_id) ON DELETE CASCADE
);

DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_signer_request') THEN
    CREATE INDEX IF NOT EXISTS idx_signer_request ON signature_signers(request_id);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_signer_status') THEN
    CREATE INDEX IF NOT EXISTS idx_signer_status ON signature_signers(status);
END IF; END $$;

-- ==============================================================================
-- CLIENT PORTAL - VIDEO MEETINGS
-- ==============================================================================

CREATE TABLE IF NOT EXISTS video_meetings (
    meeting_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    advisor_id VARCHAR(255) NOT NULL,
    meeting_type VARCHAR(50) NOT NULL, -- QUARTERLY_REVIEW, ANNUAL_PLANNING, AD_HOC
    title VARCHAR(500) NOT NULL,
    description TEXT,
    scheduled_start TIMESTAMP WITH TIME ZONE NOT NULL,
    scheduled_end TIMESTAMP WITH TIME ZONE NOT NULL,
    time_zone VARCHAR(50) DEFAULT 'America/New_York',
    video_provider VARCHAR(20) DEFAULT 'ZOOM', -- ZOOM, TEAMS, GOOGLE_MEET
    meeting_url TEXT,
    status VARCHAR(20) DEFAULT 'SCHEDULED', -- SCHEDULED, COMPLETED, CANCELLED
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_meeting_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_meeting_family') THEN
    CREATE INDEX IF NOT EXISTS idx_meeting_family ON video_meetings(family_id);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_meeting_scheduled') THEN
    CREATE INDEX IF NOT EXISTS idx_meeting_scheduled ON video_meetings(scheduled_start);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_meeting_status') THEN
    CREATE INDEX IF NOT EXISTS idx_meeting_status ON video_meetings(status);
END IF; END $$;

CREATE TABLE IF NOT EXISTS meeting_participants (
    participant_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id UUID NOT NULL,
    member_id UUID,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'ATTENDEE', -- ATTENDEE, ORGANIZER
    status VARCHAR(20) DEFAULT 'NO_RESPONSE', -- ACCEPTED, TENTATIVE, DECLINED, NO_RESPONSE
    CONSTRAINT fk_participant_meeting FOREIGN KEY (meeting_id) 
        REFERENCES video_meetings(meeting_id) ON DELETE CASCADE
);

DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_participant_meeting') THEN
    CREATE INDEX IF NOT EXISTS idx_participant_meeting ON meeting_participants(meeting_id);
END IF; END $$;

-- ==============================================================================
-- CLIENT PORTAL - ACTIVITY FEED
-- ==============================================================================

CREATE TABLE IF NOT EXISTS activity_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL, -- DOCUMENT_UPLOADED, MESSAGE_SENT, etc.
    title VARCHAR(500) NOT NULL,
    description TEXT,
    actor_id VARCHAR(255),
    actor_name VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_activity_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_activity_family') THEN
    CREATE INDEX IF NOT EXISTS idx_activity_family ON activity_events(family_id, created_at DESC);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_activity_type') THEN
    CREATE INDEX IF NOT EXISTS idx_activity_type ON activity_events(event_type);
END IF; END $$;

-- ==============================================================================
-- COMPLIANCE - FORM ADV
-- ==============================================================================

CREATE TABLE IF NOT EXISTS form_adv_filings (
    form_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    firm_id VARCHAR(255) NOT NULL,
    form_type VARCHAR(50) NOT NULL, -- INITIAL, AMENDMENT, ANNUAL_UPDATE
    filing_date TIMESTAMP WITH TIME ZONE NOT NULL,
    effective_date TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) DEFAULT 'DRAFT', -- DRAFT, FILED, APPROVED
    iard_number VARCHAR(50),
    part1_data JSONB NOT NULL,
    part2_data JSONB NOT NULL,
    schedules JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_form_adv_firm') THEN
    CREATE INDEX IF NOT EXISTS idx_form_adv_firm ON form_adv_filings(firm_id);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_form_adv_status') THEN
    CREATE INDEX IF NOT EXISTS idx_form_adv_status ON form_adv_filings(status);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_form_adv_filing_date') THEN
    CREATE INDEX IF NOT EXISTS idx_form_adv_filing_date ON form_adv_filings(filing_date DESC);
END IF; END $$;

-- ==============================================================================
-- COMPLIANCE - GIPS
-- ==============================================================================

CREATE TABLE IF NOT EXISTS gips_compliance_reports (
    compliance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    firm_id VARCHAR(255) NOT NULL,
    verification_date TIMESTAMP WITH TIME ZONE NOT NULL,
    verifier VARCHAR(255),
    compliance_status VARCHAR(50) NOT NULL, -- COMPLIANT, NON_COMPLIANT, IN_REVIEW
    composites JSONB NOT NULL,
    violations JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_gips_firm') THEN
    CREATE INDEX IF NOT EXISTS idx_gips_firm ON gips_compliance_reports(firm_id);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_gips_status') THEN
    CREATE INDEX IF NOT EXISTS idx_gips_status ON gips_compliance_reports(compliance_status);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_gips_date') THEN
    CREATE INDEX IF NOT EXISTS idx_gips_date ON gips_compliance_reports(verification_date DESC);
END IF; END $$;

-- ==============================================================================
-- COMPLIANCE - TRADE SURVEILLANCE
-- ==============================================================================

CREATE TABLE IF NOT EXISTS trade_surveillance_alerts (
    alert_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    firm_id VARCHAR(255) NOT NULL,
    alert_type VARCHAR(100) NOT NULL, -- FRONT_RUNNING, MARKET_MANIPULATION, etc.
    severity VARCHAR(20) NOT NULL, -- LOW, MEDIUM, HIGH, CRITICAL
    description TEXT NOT NULL,
    trade_details JSONB,
    status VARCHAR(50) DEFAULT 'OPEN', -- OPEN, INVESTIGATING, RESOLVED, FALSE_POSITIVE
    assigned_to VARCHAR(255),
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolution TEXT
);

DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_surveillance_firm') THEN
    CREATE INDEX IF NOT EXISTS idx_surveillance_firm ON trade_surveillance_alerts(firm_id);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_surveillance_type') THEN
    CREATE INDEX IF NOT EXISTS idx_surveillance_type ON trade_surveillance_alerts(alert_type);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_surveillance_severity') THEN
    CREATE INDEX IF NOT EXISTS idx_surveillance_severity ON trade_surveillance_alerts(severity);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_surveillance_status') THEN
    CREATE INDEX IF NOT EXISTS idx_surveillance_status ON trade_surveillance_alerts(status);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_surveillance_detected') THEN
    CREATE INDEX IF NOT EXISTS idx_surveillance_detected ON trade_surveillance_alerts(detected_at DESC);
END IF; END $$;

-- ==============================================================================
-- COMPLIANCE - SUITABILITY ANALYSIS
-- ==============================================================================

CREATE TABLE IF NOT EXISTS suitability_analyses (
    analysis_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id VARCHAR(255) NOT NULL,
    family_id UUID NOT NULL,
    member_id UUID,
    analysis_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    client_profile JSONB NOT NULL,
    portfolio_allocation JSONB NOT NULL,
    suitability_score NUMERIC(5,2) NOT NULL, -- 0-100
    suitability_status VARCHAR(20) NOT NULL, -- SUITABLE, WARNING, UNSUITABLE
    violations JSONB,
    recommendations JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_suitability_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_suitability_account') THEN
    CREATE INDEX IF NOT EXISTS idx_suitability_account ON suitability_analyses(account_id);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_suitability_family') THEN
    CREATE INDEX IF NOT EXISTS idx_suitability_family ON suitability_analyses(family_id);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_suitability_status') THEN
    CREATE INDEX IF NOT EXISTS idx_suitability_status ON suitability_analyses(suitability_status);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_suitability_date') THEN
    CREATE INDEX IF NOT EXISTS idx_suitability_date ON suitability_analyses(analysis_date DESC);
END IF; END $$;

-- ==============================================================================
-- COMPLIANCE - AUDIT TRAIL
-- ==============================================================================

CREATE TABLE IF NOT EXISTS audit_trail (
    entry_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    user_id VARCHAR(255) NOT NULL,
    user_email VARCHAR(255),
    action VARCHAR(50) NOT NULL, -- CREATE, UPDATE, DELETE, VIEW, EXPORT
    resource_type VARCHAR(100) NOT NULL, -- ACCOUNT, TRADE, REPORT, etc.
    resource_id VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    changes JSONB,
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT
);

DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_audit_user') THEN
    CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_trail(user_id);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_audit_resource') THEN
    CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_trail(resource_type, resource_id);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_audit_action') THEN
    CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_trail(action);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_audit_timestamp') THEN
    CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_trail(timestamp DESC);
END IF; END $$;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_audit_success') THEN
    CREATE INDEX IF NOT EXISTS idx_audit_success ON audit_trail(success, timestamp DESC);
END IF; END $$;

-- ==============================================================================
-- COMMENTS
-- ==============================================================================

COMMENT ON TABLE client_messages IS 'Secure encrypted messages between clients and advisors';
COMMENT ON TABLE message_attachments IS 'File attachments for client messages';
COMMENT ON TABLE signature_requests IS 'E-signature document requests';
COMMENT ON TABLE signature_signers IS 'Individual signers for e-signature requests';
COMMENT ON TABLE video_meetings IS 'Scheduled video meetings with clients';
COMMENT ON TABLE meeting_participants IS 'Participants in video meetings';
COMMENT ON TABLE activity_events IS 'Client portal activity feed';
COMMENT ON TABLE form_adv_filings IS 'SEC Form ADV filings';
COMMENT ON TABLE gips_compliance_reports IS 'GIPS compliance verification reports';
COMMENT ON TABLE trade_surveillance_alerts IS 'Trade compliance monitoring alerts';
COMMENT ON TABLE suitability_analyses IS 'Investment suitability assessments';
COMMENT ON TABLE audit_trail IS 'Complete audit log for regulatory compliance';
