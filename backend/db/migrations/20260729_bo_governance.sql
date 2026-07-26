-- Uisce World-Class Business Object Governance Engine
-- Rule 1 (Config-Before-Code) & Rule 7 (Security Mandate)

BEGIN;

CREATE SCHEMA IF NOT EXISTS bo;

-- 1. CEL-based Field Validation Rules
CREATE TABLE IF NOT EXISTS bo.validation_rule (
    rule_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    bo_key           VARCHAR(150) NOT NULL,
    field_key        VARCHAR(150),                        -- NULL = cross-field rule
    rule_name        VARCHAR(200) NOT NULL,
    description      TEXT,
    expression       TEXT NOT NULL,                       -- CEL expression e.g. "value.len() > 3"
    error_message    TEXT NOT NULL,
    severity         VARCHAR(20) DEFAULT 'ERROR',         -- ERROR, WARNING, INFO
    priority         INT DEFAULT 100,
    is_active        BOOLEAN DEFAULT TRUE,
    is_core          BOOLEAN DEFAULT FALSE,               -- inherited from gold_copy
    created_by       UUID,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_validation_rule_tenant_bo ON bo.validation_rule(tenant_id, bo_key, is_active);

-- 2. Declarative WHEN/THEN Policy Rules
CREATE TABLE IF NOT EXISTS bo.policy_rule (
    policy_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    bo_key           VARCHAR(150) NOT NULL,
    policy_name      VARCHAR(200) NOT NULL,
    description      TEXT,
    trigger_event    VARCHAR(50) NOT NULL,                -- ON_SAVE, ON_SUBMIT, ON_READ, ON_DELETE, ON_FIELD_CHANGE
    condition_expr   TEXT NOT NULL,                       -- CEL: e.g. "record.status == 'PENDING' && record.amount > 10000"
    action_type      VARCHAR(50) NOT NULL,                -- BLOCK, REQUIRE_APPROVAL, NOTIFY_ROLE, ESCALATE, COMPUTE_FIELD
    action_config    JSONB DEFAULT '{}',                  -- { "role": "COMPLIANCE_OFFICER", "message": "..." }
    priority         INT DEFAULT 100,
    is_active        BOOLEAN DEFAULT TRUE,
    is_core          BOOLEAN DEFAULT FALSE,
    created_by       UUID,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_policy_rule_tenant_bo ON bo.policy_rule(tenant_id, bo_key, trigger_event, is_active);

-- 3. RBAC + ABAC Access Policy Matrix
CREATE TABLE IF NOT EXISTS bo.access_policy (
    access_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    bo_key           VARCHAR(150) NOT NULL,
    role_key         VARCHAR(150) NOT NULL,               -- e.g. "PORTFOLIO_MANAGER", "COMPLIANCE_OFFICER"
    operation        VARCHAR(50) NOT NULL,                -- READ, WRITE, DELETE, EXECUTE, ADMIN
    is_allowed       BOOLEAN DEFAULT FALSE,
    condition_expr   TEXT,                                -- ABAC: optional CEL e.g. "principal.department == record.owner_dept"
    row_filter_expr  TEXT,                                -- Row-level: e.g. "tenant_id = :tenant_id AND owner_id = :principal_id"
    is_core          BOOLEAN DEFAULT FALSE,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, bo_key, role_key, operation)
);
CREATE INDEX IF NOT EXISTS idx_access_policy_tenant_bo ON bo.access_policy(tenant_id, bo_key, role_key);

-- 4. Field-Level Security & Masking Config
CREATE TABLE IF NOT EXISTS bo.field_security (
    security_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    bo_key           VARCHAR(150) NOT NULL,
    field_key        VARCHAR(150) NOT NULL,
    classification   VARCHAR(50) DEFAULT 'PUBLIC',        -- PUBLIC, MASKED, ENCRYPTED, RESTRICTED
    mask_pattern     VARCHAR(200),                        -- e.g. "***-**-####" for SSN
    visible_to_roles TEXT[] DEFAULT ARRAY['ADMIN'],       -- roles that see full value
    mask_for_roles   TEXT[] DEFAULT ARRAY[]::TEXT[],      -- roles that see masked value
    deny_to_roles    TEXT[] DEFAULT ARRAY[]::TEXT[],      -- roles that see nothing
    is_core          BOOLEAN DEFAULT FALSE,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, bo_key, field_key)
);
CREATE INDEX IF NOT EXISTS idx_field_security_tenant_bo ON bo.field_security(tenant_id, bo_key);

-- 5. Immutable BO Audit Event Log
CREATE TABLE IF NOT EXISTS bo.audit_event (
    event_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    bo_key           VARCHAR(150) NOT NULL,
    entity_id        TEXT,
    operation        VARCHAR(50) NOT NULL,                -- CREATE, UPDATE, DELETE, READ, POLICY_TRIGGERED, ACCESS_DENIED
    actor_id         TEXT NOT NULL,
    actor_role       TEXT,
    before_value     JSONB,
    after_value      JSONB,
    policy_triggered TEXT,                               -- policy_name if a policy fired
    ip_address       INET,
    session_id       TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_event_tenant_bo ON bo.audit_event(tenant_id, bo_key, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_event_entity ON bo.audit_event(tenant_id, entity_id, created_at DESC);

COMMIT;
