-- Client Portal & Digital Onboarding Schema
-- Phase 6: Client-Facing Digital Experience

-- ===========================
-- ONBOARDING SESSIONS
-- ===========================
CREATE TABLE IF NOT EXISTS onboarding_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID,
    email TEXT NOT NULL,
    current_step INTEGER DEFAULT 1,
    total_steps INTEGER DEFAULT 7,
    step_data JSONB DEFAULT '{}'::jsonb,
    status VARCHAR(50) NOT NULL DEFAULT 'IN_PROGRESS' CHECK (status IN (
        'IN_PROGRESS',
        'COMPLETED',
        'ABANDONED',
        'EXPIRED'
    )),
    last_active_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_onboarding_client' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_onboarding_client ON onboarding_sessions(client_id);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_onboarding_email' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_onboarding_email ON onboarding_sessions(email);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_onboarding_status' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_onboarding_status ON onboarding_sessions(status) WHERE status = 'IN_PROGRESS';
  END IF;
END$$; 

-- ===========================
-- UPLOADED DOCUMENTS
-- ===========================
CREATE TABLE IF NOT EXISTS uploaded_documents (
    document_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    onboarding_session_id UUID REFERENCES onboarding_sessions(session_id) ON DELETE CASCADE,
    document_type VARCHAR(50) NOT NULL CHECK (document_type IN (
        'DRIVERS_LICENSE',
        'PASSPORT',
        'W9',
        'W8',
        'BANK_STATEMENT',
        'PROOF_OF_ADDRESS',
        'TAX_RETURN',
        'OTHER'
    )),
    file_url TEXT NOT NULL,
    file_name TEXT,
    file_size_bytes BIGINT,
    mime_type VARCHAR(100),
    
    -- OCR extraction
    ocr_extracted_data JSONB,
    ocr_confidence DECIMAL(3,2),
    
    -- Verification
    verification_status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (verification_status IN (
        'PENDING',
        'IN_REVIEW',
        'VERIFIED',
        'REJECTED',
        'EXPIRED'
    )),
    verification_notes TEXT,
    verified_by UUID,
    verified_at TIMESTAMPTZ,
    
    uploaded_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_documents_client' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_documents_client ON uploaded_documents(client_id);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_documents_session' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_documents_session ON uploaded_documents(onboarding_session_id);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_documents_verification' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_documents_verification ON uploaded_documents(verification_status) WHERE verification_status = 'PENDING';
  END IF;
END$$; 

-- ===========================
-- E-SIGNATURES
-- ===========================
CREATE TABLE IF NOT EXISTS e_signatures (
    signature_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    onboarding_session_id UUID REFERENCES onboarding_sessions(session_id),
    
    document_name TEXT NOT NULL,
    document_url TEXT NOT NULL,
    document_type VARCHAR(50), -- 'CLIENT_AGREEMENT', 'PRIVACY_POLICY', 'DISCLOSURE', 'IPS'
    
    -- Provider integration
    signature_provider VARCHAR(50) DEFAULT 'INTERNAL' CHECK (signature_provider IN (
        'DOCUSIGN',
        'ADOBE_SIGN',
        'INTERNAL'
    )),
    provider_envelope_id TEXT,
    provider_metadata JSONB,
    
    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'SENT' CHECK (status IN (
        'SENT',
        'VIEWED',
        'SIGNED',
        'DECLINED',
        'EXPIRED',
        'VOIDED'
    )),
    
    sent_at TIMESTAMPTZ DEFAULT NOW(),
    viewed_at TIMESTAMPTZ,
    signed_at TIMESTAMPTZ,
    
    -- Audit trail
    ip_address INET,
    user_agent TEXT,
    signature_image_url TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_esignatures_client' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_esignatures_client ON e_signatures(client_id);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_esignatures_status' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_esignatures_status ON e_signatures(status);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_esignatures_pending' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_esignatures_pending ON e_signatures(status) WHERE status IN ('SENT', 'VIEWED');
  END IF;
END$$; 

-- ===========================
-- DASHBOARD WIDGETS
-- ===========================
CREATE TABLE IF NOT EXISTS dashboard_widgets (
    widget_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    
    widget_type VARCHAR(50) NOT NULL CHECK (widget_type IN (
        'PORTFOLIO_SUMMARY',
        'GOALS_PROGRESS',
        'RECENT_TRANSACTIONS',
        'MESSAGES_INBOX',
        'BILLING_STATUS',
        'UPCOMING_MEETINGS',
        'MARKET_NEWS',
        'RECOMMENDED_ACTIONS',
        'NET_WORTH_TREND',
        'ASSET_ALLOCATION'
    )),
    
    position INTEGER NOT NULL, -- Display order (1 = top-left)
    size VARCHAR(20) DEFAULT 'MEDIUM' CHECK (size IN ('SMALL', 'MEDIUM', 'LARGE', 'FULL_WIDTH')),
    config JSONB DEFAULT '{}'::jsonb, -- Widget-specific settings
    is_visible BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(client_id, position)
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_widgets_client' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_widgets_client ON dashboard_widgets(client_id, position);
  END IF;
END$$; 

-- ===========================
-- CLIENT GOALS
-- ===========================
CREATE TABLE IF NOT EXISTS client_goals (
    goal_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    
    goal_type VARCHAR(50) NOT NULL CHECK (goal_type IN (
        'RETIREMENT',
        'EDUCATION',
        'HOME_PURCHASE',
        'DEBT_PAYOFF',
        'LEGACY',
        'MAJOR_PURCHASE',
        'EMERGENCY_FUND',
        'CUSTOM'
    )),
    goal_name TEXT NOT NULL,
    description TEXT,
    
    -- Financial targets
    target_amount DECIMAL(15,2) NOT NULL,
    target_date DATE NOT NULL,
    current_progress DECIMAL(15,2) DEFAULT 0,
    
    -- Planning
    monthly_contribution DECIMAL(10,2),
    assumed_return_rate DECIMAL(5,4), -- e.g., 0.07 = 7%
    projected_completion_date DATE,
    confidence_level DECIMAL(3,2) CHECK (confidence_level BETWEEN 0 AND 1),
    
    -- Status
    status VARCHAR(50) DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'COMPLETED', 'PAUSED', 'ABANDONED')),
    completed_at TIMESTAMPTZ,
    
    notes TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_goals_client' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_goals_client ON client_goals(client_id, status);
  END IF;
END$$; 

-- ===========================
-- SECURE MESSAGES
-- ===========================
CREATE TABLE IF NOT EXISTS secure_messages (
    message_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL,
    
    sender_id UUID NOT NULL,
    sender_type VARCHAR(20) NOT NULL CHECK (sender_type IN ('CLIENT', 'ADVISOR', 'SYSTEM')),
    
    recipient_id UUID NOT NULL,
    recipient_type VARCHAR(20) NOT NULL CHECK (recipient_type IN ('CLIENT', 'ADVISOR')),
    
    subject TEXT,
    message_text_encrypted TEXT NOT NULL, -- AES-256 encrypted
    encryption_key_id TEXT, -- Reference to key management system
    
    attachments JSONB DEFAULT '[]'::jsonb, -- Array of {filename, url, size}
    
    -- Status
    read_at TIMESTAMPTZ,
    archived BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_messages_conversation' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_messages_conversation ON secure_messages(conversation_id, created_at DESC);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_messages_recipient_unread' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_messages_recipient_unread ON secure_messages(recipient_id, created_at DESC) WHERE read_at IS NULL;
  END IF;
END$$; 

-- ===========================
-- NOTIFICATIONS
-- ===========================
CREATE TABLE IF NOT EXISTS notifications (
    notification_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    
    notification_type VARCHAR(50) NOT NULL CHECK (notification_type IN (
        'MEETING_REMINDER',
        'DOCUMENT_REQUEST',
        'DOCUMENT_READY',
        'MESSAGE_RECEIVED',
        'BILLING_DUE',
        'PAYMENT_CONFIRMED',
        'MARKET_ALERT',
        'GOAL_MILESTONE',
        'PERFORMANCE_UPDATE',
        'CAPITAL_CALL',
        'SYSTEM_UPDATE'
    )),
    
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    
    priority VARCHAR(20) DEFAULT 'MEDIUM' CHECK (priority IN ('LOW', 'MEDIUM', 'HIGH', 'URGENT')),
    
    -- Multi-channel delivery
    channels TEXT[] DEFAULT ARRAY['IN_APP'], -- 'EMAIL', 'SMS', 'PUSH', 'IN_APP'
    sent_via JSONB DEFAULT '{}'::jsonb, -- Track which channels actually sent
    
    -- Action
    action_url TEXT,
    action_label TEXT,
    
    -- Status
    read_at TIMESTAMPTZ,
    dismissed_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'notifications' AND column_name = 'client_id') THEN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_notifications_client' AND relkind = 'i') THEN
      CREATE INDEX IF NOT EXISTS idx_notifications_client ON notifications(client_id, created_at DESC);
    END IF;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'notifications' AND column_name = 'client_id')
     AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'notifications' AND column_name = 'read_at') THEN
    -- Create a conservative partial index only on read_at NULL (skip dismissed_at if not present)
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_notifications_unread' AND relkind = 'i') THEN
      CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications(client_id, created_at DESC) WHERE read_at IS NULL;
    END IF;
  END IF;
END$$; 

-- ===========================
-- SCENARIO SIMULATIONS
-- ===========================
CREATE TABLE IF NOT EXISTS scenario_simulations (
    simulation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    
    scenario_name TEXT NOT NULL,
    scenario_type VARCHAR(50) CHECK (scenario_type IN (
        'RETIREMENT',
        'MARKET_SHOCK',
        'JOB_CHANGE',
        'EARLY_RETIREMENT',
        'INHERITANCE',
        'MAJOR_EXPENSE',
        'CUSTOM'
    )),
    
    -- Inputs
    input_parameters JSONB NOT NULL, -- {savings_rate, return, inflation, years, etc.}
    
    -- Outputs
    projected_outcomes JSONB NOT NULL, -- {age_60: $2.5M, age_65: $3.8M, probability: 0.87}
    monte_carlo_results JSONB, -- Distribution of outcomes
    
    -- Metadata
    created_by UUID, -- Could be client or advisor
    shared_with_client BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_simulations_client' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_simulations_client ON scenario_simulations(client_id, created_at DESC);
  END IF;
END$$; 

-- ===========================
-- CLIENT SESSIONS
-- ===========================
CREATE TABLE IF NOT EXISTS client_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    
    -- Device fingerprinting
    device_fingerprint TEXT,
    device_name TEXT,
    ip_address INET NOT NULL,
    user_agent TEXT,
    
    -- Authentication
    login_method VARCHAR(50) NOT NULL CHECK (login_method IN (
        'PASSWORD',
        'BIOMETRIC',
        'SSO',
        'MFA',
        'MAGIC_LINK'
    )),
    mfa_verified BOOLEAN DEFAULT FALSE,
    
    -- Session lifecycle
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Audit
    logout_reason VARCHAR(50), -- 'USER_LOGOUT', 'TIMEOUT', 'FORCED', 'EXPIRED'
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_sessions_client' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_sessions_client ON client_sessions(client_id, is_active) WHERE is_active = TRUE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_sessions_active' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_sessions_active ON client_sessions(expires_at) WHERE is_active = TRUE;
  END IF;
END$$; 

-- ===========================
-- CONSENT PREFERENCES
-- ===========================
CREATE TABLE IF NOT EXISTS consent_preferences (
    consent_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    
    consent_type VARCHAR(50) NOT NULL CHECK (consent_type IN (
        'MARKETING_EMAIL',
        'MARKETING_SMS',
        'PERFORMANCE_REPORTS',
        'ADVISOR_ACCESS_FULL',
        'ADVISOR_ACCESS_LIMITED',
        'DATA_SHARING_THIRD_PARTY',
        'DATA_SHARING_AFFILIATES',
        'ANALYTICS_TRACKING',
        'BIOMETRIC_DATA'
    )),
    
    granted BOOLEAN NOT NULL,
    scope JSONB, -- Specific permissions within type
    
    -- Lifecycle
    granted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    
    -- Audit
    ip_address INET,
    user_agent TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(client_id, consent_type)
);

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'consent_preferences' AND column_name = 'client_id') THEN
    CREATE INDEX IF NOT EXISTS idx_consent_client ON consent_preferences(client_id);
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'consent_preferences' AND column_name = 'client_id')
     AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'consent_preferences' AND column_name = 'granted') THEN
    -- Use a simple predicate that is immutable-friendly (avoid NOW())
    CREATE INDEX IF NOT EXISTS idx_consent_active ON consent_preferences(client_id, consent_type) WHERE granted = TRUE;
  END IF;
END$$; 

-- ===========================
-- RECOMMENDED ACTIONS
-- ===========================
CREATE TABLE IF NOT EXISTS recommended_actions (
    action_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    
    action_type VARCHAR(50) NOT NULL CHECK (action_type IN (
        'REBALANCE_PORTFOLIO',
        'TAX_LOSS_HARVEST',
        'INCREASE_CONTRIBUTION',
        'REDUCE_EXPENSES',
        'SCHEDULE_REVIEW',
        'UPDATE_BENEFICIARIES',
        'REVIEW_INSURANCE',
        'ROTH_CONVERSION',
        'ESTATE_PLANNING',
        'CHARITABLE_GIVING'
    )),
    
    priority INTEGER CHECK (priority BETWEEN 1 AND 10),
    
    -- Impact estimation
    estimated_impact_dollars DECIMAL(12,2),
    estimated_impact_description TEXT,
    time_sensitivity VARCHAR(50) CHECK (time_sensitivity IN (
        'IMMEDIATE',
        'THIS_WEEK',
        'THIS_MONTH',
        'THIS_QUARTER',
        'ANYTIME'
    )),
    
    -- Details
    action_details JSONB NOT NULL,
    rationale TEXT,
    
    -- Status
    status VARCHAR(50) DEFAULT 'PENDING' CHECK (status IN (
        'PENDING',
        'PRESENTED',
        'ACCEPTED',
        'DECLINED',
        'COMPLETED',
        'EXPIRED'
    )),
    
    presented_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_actions_client_pending' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_actions_client_pending ON recommended_actions(client_id, priority DESC) 
      WHERE status IN ('PENDING', 'PRESENTED');
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_actions_type' AND relkind = 'i') THEN
    CREATE INDEX IF NOT EXISTS idx_actions_type ON recommended_actions(action_type, status);
  END IF;
END$$; 

-- ===========================
-- UPDATE TRIGGERS
-- ===========================
CREATE OR REPLACE FUNCTION update_portal_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'onboarding_sessions_updated_at') THEN
    DROP TRIGGER IF EXISTS onboarding_sessions_updated_at ON onboarding_sessions;
CREATE TRIGGER onboarding_sessions_updated_at
      BEFORE UPDATE ON onboarding_sessions
      FOR EACH ROW
      EXECUTE FUNCTION update_portal_timestamp();
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'dashboard_widgets_updated_at') THEN
    DROP TRIGGER IF EXISTS dashboard_widgets_updated_at ON dashboard_widgets;
CREATE TRIGGER dashboard_widgets_updated_at
      BEFORE UPDATE ON dashboard_widgets
      FOR EACH ROW
      EXECUTE FUNCTION update_portal_timestamp();
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'client_goals_updated_at') THEN
    DROP TRIGGER IF EXISTS client_goals_updated_at ON client_goals;
CREATE TRIGGER client_goals_updated_at
      BEFORE UPDATE ON client_goals
      FOR EACH ROW
      EXECUTE FUNCTION update_portal_timestamp();
  END IF;
END$$; 

-- ===========================
-- VIEWS
-- ===========================

-- Active onboarding sessions with progress
CREATE OR REPLACE VIEW active_onboarding_sessions AS
SELECT 
    s.session_id,
    s.client_id,
    s.email,
    s.current_step,
    s.total_steps,
    ROUND((s.current_step::decimal / s.total_steps) * 100, 2) as completion_percentage,
    s.last_active_at,
    EXTRACT(EPOCH FROM (NOW() - s.last_active_at))/3600 as hours_since_activity,
    COUNT  (d.document_id) as documents_uploaded,
    COUNT(e.signature_id) as signatures_completed
FROM onboarding_sessions s
LEFT JOIN uploaded_documents d ON s.session_id = d.onboarding_session_id
LEFT JOIN e_signatures e ON s.session_id = e.onboarding_session_id AND e.status = 'SIGNED'
WHERE s.status = 'IN_PROGRESS'
GROUP BY s.session_id;

-- Client dashboard summary
DO $$
BEGIN
  DROP VIEW IF EXISTS client_dashboard_summary;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'clients' AND column_name = 'id') THEN
    EXECUTE $exec$
      CREATE OR REPLACE VIEW client_dashboard_summary AS
      SELECT 
          c.id AS client_id,
          (SELECT COUNT(*) FROM secure_messages WHERE recipient_id = c.id AND read_at IS NULL) as unread_messages,
          (SELECT COUNT(*) FROM notifications WHERE recipient_id = c.id AND read_at IS NULL AND dismissed_at IS NULL) as unread_notifications,
          (SELECT COUNT(*) FROM recommended_actions WHERE client_id = c.id AND status IN ('PENDING', 'PRESENTED')) as pending_actions,
          (SELECT COUNT(*) FROM client_goals WHERE client_id = c.id AND status = 'ACTIVE') as active_goals
      FROM clients c;
    $exec$;
  ELSE
    EXECUTE $exec$
      CREATE OR REPLACE VIEW client_dashboard_summary AS
      SELECT NULL::uuid AS client_id, 0 AS unread_messages, 0 AS unread_notifications, 0 AS pending_actions, 0 AS active_goals WHERE FALSE;
    $exec$;
  END IF;
END$$;
COMMENT ON TABLE onboarding_sessions IS 'Multi-step digital onboarding with progress tracking and resume capability';
COMMENT ON TABLE uploaded_documents IS 'Client documents with OCR extraction and verification workflow';
COMMENT ON TABLE e_signatures IS 'Electronic signature tracking for compliance';
COMMENT ON TABLE dashboard_widgets IS 'Personalized dashboard widget configurations';
COMMENT ON TABLE client_goals IS 'Financial goals with milestone tracking and projections';
COMMENT ON TABLE secure_messages IS 'Encrypted client-advisor messaging';
COMMENT ON TABLE notifications IS 'Multi-channel notification delivery tracking';
COMMENT ON TABLE recommended_actions IS 'AI-generated actionable insights prioritized by impact';
