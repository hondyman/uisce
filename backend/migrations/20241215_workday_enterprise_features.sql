-- ============================================================================
-- MULTILINGUAL & INTERNATIONALIZATION (i18n)
-- Workday-style translation and locale management
-- ============================================================================

-- Supported Locales
CREATE TABLE IF NOT EXISTS locales (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    code TEXT NOT NULL UNIQUE, -- en-US, es-MX, fr-FR, de-DE, ja-JP, zh-CN
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    direction TEXT DEFAULT 'ltr', -- ltr or rtl
    date_format TEXT DEFAULT 'MM/DD/YYYY',
    time_format TEXT DEFAULT 'HH:mm',
    number_format JSONB DEFAULT '{"decimal": ".", "thousand": ",", "precision": 2}',
    currency_format JSONB DEFAULT '{"symbol": "$", "position": "before"}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_locales_code ON locales(code);
CREATE INDEX IF NOT EXISTS idx_locales_active ON locales(is_active);

-- Translations
CREATE TABLE IF NOT EXISTS translations (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT DEFAULT 'default-tenant',
    locale_code TEXT NOT NULL REFERENCES locales(code),
    namespace TEXT NOT NULL, -- ui, reports, errors, business_objects
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    context TEXT, -- Additional context for translators
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT translations_unique UNIQUE (tenant_id, locale_code, namespace, key)
);

CREATE INDEX IF NOT EXISTS idx_translations_lookup ON translations(tenant_id, locale_code, namespace, key);
CREATE INDEX IF NOT EXISTS idx_translations_namespace ON translations(namespace);

-- ============================================================================
-- AUDIT LOGGING (Comprehensive Change Tracking)
-- ============================================================================

CREATE TABLE IF NOT EXISTS audit_log (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    entity_type TEXT NOT NULL, -- business_object, report, user, etc.
    entity_id TEXT NOT NULL,
    entity_name TEXT,
    action TEXT NOT NULL, -- create, update, delete, approve, reject, execute
    actor TEXT NOT NULL,
    actor_type TEXT DEFAULT 'user', -- user, system, api
    ip_address TEXT,
    user_agent TEXT,
    changes JSONB, -- Before/after values
    metadata JSONB, -- Additional context
    severity TEXT DEFAULT 'info', -- info, warning, critical
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_tenant ON audit_log(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_log(actor);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);

-- ============================================================================
-- SECURITY GROUPS & PERMISSIONS
-- ============================================================================

-- Security Groups
CREATE TABLE IF NOT EXISTS security_groups (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    group_type TEXT NOT NULL DEFAULT 'role', -- role, team, department
    parent_group_id TEXT REFERENCES security_groups(id),
    is_system BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT sg_unique_code UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS idx_sg_tenant ON security_groups(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sg_type ON security_groups(group_type);

-- Group Memberships
CREATE TABLE IF NOT EXISTS security_group_members (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    group_id TEXT NOT NULL REFERENCES security_groups(id) ON DELETE CASCADE,
    member_id TEXT NOT NULL, -- user_id
    member_type TEXT DEFAULT 'user',
    granted_by TEXT,
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    CONSTRAINT sgm_unique UNIQUE (group_id, member_id)
);

CREATE INDEX IF NOT EXISTS idx_sgm_group ON security_group_members(group_id);
CREATE INDEX IF NOT EXISTS idx_sgm_member ON security_group_members(member_id);

-- Permissions
CREATE TABLE IF NOT EXISTS permissions (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    security_group_id TEXT NOT NULL REFERENCES security_groups(id) ON DELETE CASCADE,
    resource_type TEXT NOT NULL, -- business_object, report, portfolio, etc.
    resource_id TEXT, -- NULL for type-level permissions
    action TEXT NOT NULL, -- view, edit, delete, execute, approve
    grant_type TEXT DEFAULT 'allow', -- allow, deny
    conditions JSONB, -- Conditional access rules
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_perm_group ON permissions(security_group_id);
CREATE INDEX IF NOT EXISTS idx_perm_resource ON permissions(resource_type, resource_id);

-- ============================================================================
-- DOCUMENT ATTACHMENTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS attachments (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    entity_type TEXT NOT NULL, -- portfolio, account, client, report, etc.
    entity_id TEXT NOT NULL,
    file_name TEXT NOT NULL,
    file_type TEXT NOT NULL, -- pdf, excel, image, etc.
    file_size BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    storage_provider TEXT DEFAULT 's3', -- s3, azure, gcs, local
    title TEXT,
    description TEXT,
    category TEXT, -- statement, contract, kyc, tax_document
    is_confidential BOOLEAN DEFAULT false,
    uploaded_by TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_att_entity ON attachments(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_att_tenant ON attachments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_att_category ON attachments(category);

-- ============================================================================
-- NOTIFICATIONS & ALERTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS notification_templates (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    category TEXT NOT NULL, -- report, approval, alert, system
    channels TEXT[] DEFAULT ARRAY['email', 'portal'], -- email, sms, portal, slack
    subject_template TEXT NOT NULL,
    body_template TEXT NOT NULL,
    variables JSONB DEFAULT '[]', -- Available template variables
    is_system BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT nt_unique_key UNIQUE (tenant_id, key)
);

CREATE TABLE IF NOT EXISTS notifications (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    template_id TEXT REFERENCES notification_templates(id),
    recipient_id TEXT NOT NULL,
    recipient_type TEXT DEFAULT 'user', -- user, role, email
    channel TEXT NOT NULL, -- email, portal, sms
    priority TEXT DEFAULT 'normal', -- low, normal, high, urgent
    category TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    data JSONB, -- Additional data
    link_url TEXT,
    link_text TEXT,
    status TEXT DEFAULT 'pending', -- pending, sent, delivered, read, failed
    sent_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notif_recipient ON notifications(recipient_id, status);
CREATE INDEX IF NOT EXISTS idx_notif_created ON notifications(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notif_status ON notifications(status);

-- ============================================================================
-- SEED DATA
-- ============================================================================

-- Seed Locales
INSERT INTO locales (code, name, display_name, date_format, time_format)
VALUES
    ('en-US', 'English (United States)', 'English', 'MM/DD/YYYY', 'h:mm A'),
    ('en-GB', 'English (United Kingdom)', 'English (UK)', 'DD/MM/YYYY', 'HH:mm'),
    ('es-MX', 'Spanish (Mexico)', 'Español', 'DD/MM/YYYY', 'HH:mm'),
    ('fr-FR', 'French (France)', 'Français', 'DD/MM/YYYY', 'HH:mm'),
    ('de-DE', 'German (Germany)', 'Deutsch', 'DD.MM.YYYY', 'HH:mm'),
    ('ja-JP', 'Japanese (Japan)', '日本語', 'YYYY/MM/DD', 'HH:mm'),
    ('zh-CN', 'Chinese (Simplified)', '简体中文', 'YYYY/MM/DD', 'HH:mm')
ON CONFLICT (code) DO NOTHING;

-- Seed System Security Groups
INSERT INTO security_groups (id, tenant_id, code, name, display_name, description, is_system)
VALUES
    ('sg-admin', 'default-tenant', 'ADMINISTRATORS', 'Administrators', 'System Administrators', 'Full system access', true),
    ('sg-portfolio-mgr', 'default-tenant', 'PORTFOLIO_MANAGERS', 'Portfolio Managers', 'Portfolio Managers', 'Manage portfolios and trading', true),
    ('sg-analysts', 'default-tenant', 'ANALYSTS', 'Analysts', 'Investment Analysts', 'View and analyze data', true),
    ('sg-compliance', 'default-tenant', 'COMPLIANCE', 'Compliance Officers', 'Compliance', 'Compliance oversight', true),
    ('sg-clients', 'default-tenant', 'CLIENTS', 'Clients', 'Client Portal Users', 'Client portal access', true)
ON CONFLICT (tenant_id, code) DO NOTHING;

-- Seed Notification Templates
INSERT INTO notification_templates (key, name, category, subject_template, body_template, variables)
VALUES
    ('report_ready', 'Report Ready', 'report', 'Your {{report_name}} report is ready', 'Hello {{user_name}},\n\nYour {{report_name}} report has been generated and is ready to view.\n\nClick here to view: {{report_url}}', '["user_name", "report_name", "report_url"]'),
    ('approval_required', 'Approval Required', 'approval', 'Approval needed: {{process_name}}', 'Hello {{user_name}},\n\n{{actor_name}} has submitted {{process_name}} for your approval.\n\nDetails: {{details}}\n\nClick to review: {{approval_url}}', '["user_name", "actor_name", "process_name", "details", "approval_url"]'),
    ('portfolio_alert', 'Portfolio Alert', 'alert', 'Portfolio Alert: {{alert_type}}', 'Portfolio: {{portfolio_name}}\nAlert: {{alert_message}}\nTriggered: {{triggered_at}}', '["portfolio_name", "alert_type", "alert_message", "triggered_at"]'),
    ('process_completed', 'Process Completed', 'approval', '{{process_name}} completed', 'Your {{process_name}} has been {{status}}.\n\nCompleted by: {{completed_by}}\nComments: {{comments}}', '["process_name", "status", "completed_by", "comments"]')
ON CONFLICT (tenant_id, key) DO NOTHING;

-- ============================================================================
-- SUCCESS MESSAGE
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✓ Multilingual schema created: 7 locales seeded';
    RAISE NOTICE '✓ Audit logging system created';
    RAISE NOTICE '✓ Security groups & permissions created: 5 system groups';
    RAISE NOTICE '✓ Document attachments system created';
    RAISE NOTICE '✓ Notifications system created: 4 templates seeded';
END $$;
