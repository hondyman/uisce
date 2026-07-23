-- Security Rules System Migration
-- Creates tables for access rules and outbox pattern for event publishing

-- Create access_rule table
CREATE TABLE IF NOT EXISTS access_rule (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    business_object_id TEXT NOT NULL,
    group_dn TEXT NOT NULL,
    access_level TEXT NOT NULL CHECK (access_level IN ('NONE', 'READ', 'WRITE')),
    status TEXT NOT NULL CHECK (status IN ('DRAFT', 'REVIEW', 'APPROVED', 'DEPRECATED')),
    row_filter_dsl TEXT,
    column_masks JSONB DEFAULT '[]'::jsonb,
    applies_to_apis BOOLEAN DEFAULT true,
    applies_to_bi BOOLEAN DEFAULT true,
    applies_to_ai BOOLEAN DEFAULT true,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_by TEXT,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1,
    description TEXT
);

-- Create indexes for access_rule
CREATE INDEX IF NOT EXISTS idx_access_rule_tenant 
    ON access_rule(tenant_id);

CREATE INDEX IF NOT EXISTS idx_access_rule_bo 
    ON access_rule(business_object_id);

CREATE INDEX IF NOT EXISTS idx_access_rule_group 
    ON access_rule(group_dn);

CREATE INDEX IF NOT EXISTS idx_access_rule_status 
    ON access_rule(status);

CREATE INDEX IF NOT EXISTS idx_access_rule_tenant_bo 
    ON access_rule(tenant_id, business_object_id);

CREATE INDEX IF NOT EXISTS idx_access_rule_tenant_bo_group 
    ON access_rule(tenant_id, business_object_id, group_dn);

-- Create GIN index for JSONB column masks
CREATE INDEX IF NOT EXISTS idx_access_rule_column_masks 
    ON access_rule USING gin(column_masks);

-- Create outbox table for event publishing (if not exists)
CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    published BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP
);

-- Create index for outbox processing
CREATE INDEX IF NOT EXISTS idx_outbox_unpublished 
    ON outbox(published, created_at) 
    WHERE published = false;

CREATE INDEX IF NOT EXISTS idx_outbox_security_events 
    ON outbox(event_type, published) 
    WHERE event_type LIKE 'security.%' AND published = false;

-- Add comments
COMMENT ON TABLE access_rule IS 'Attribute-based access control rules binding LDAP groups to business objects';
COMMENT ON COLUMN access_rule.row_filter_dsl IS 'DSL expression for row-level filtering, e.g., "region = ''EMEA'' AND status = ''active''"';
COMMENT ON COLUMN access_rule.column_masks IS 'Array of column masking rules as JSONB: [{"semantic_term_id": "term:ssn", "mask_type": "HIDE"}]';
COMMENT ON TABLE outbox IS 'Transactional outbox for async event publishing to Kafka/Trino/Iceberg';

-- Sample data for testing (optional)
-- INSERT INTO access_rule (
--     id, tenant_id, business_object_id, group_dn, 
--     access_level, status, row_filter_dsl, 
--     column_masks, created_by
-- ) VALUES (
--     'rule-1',
--     'tenant-1',
--     'bo:portfolio',
--     'cn=wealth-advisors,ou=groups,dc=example,dc=com',
--     'READ',
--     'APPROVED',
--     'region = ''EMEA''',
--     '[{"semantic_term_id": "term:ssn", "mask_type": "HIDE"}]'::jsonb,
--     'admin@example.com'
-- );
