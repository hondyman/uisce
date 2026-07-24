-- +goose Up
-- Migration: Add ABAC (Attribute-Based Access Control) schema
-- Created: 2025-09-09
-- Description: Adds tables for a comprehensive ABAC system, including users, resources, policies, and audit logs.

-- Generic trigger function to set updated_at timestamp
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Extend existing users table with attributes
ALTER TABLE app_user ADD COLUMN IF NOT EXISTS attributes JSONB DEFAULT '{}';
ALTER TABLE app_user ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

CREATE TRIGGER trigger_app_user_updated_at
    BEFORE UPDATE ON app_user
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Core ABAC Tables
CREATE TABLE resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    attributes JSONB DEFAULT '{}',  -- e.g., {"type": "semantic_model", "owner": "user1"}
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER trigger_resources_updated_at
    BEFORE UPDATE ON resources
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    rules JSONB NOT NULL,  -- e.g., {"subject": {"role": "admin"}, "action": "read", "resource": {"type": "model"}, "effect": "allow"}
    start_date TIMESTAMP WITH TIME ZONE,
    end_date TIMESTAMP WITH TIME ZONE,
    schedule JSONB,  -- e.g., {"days": ["mon", "tue"], "time_window": "09:00-17:00"}
    location_rules JSONB,  -- e.g., {"ip_range": "192.168.0.0/24", "geofence": {"lat": 37.77, "long": -122.42, "radius": 1000}}
    priority INTEGER DEFAULT 0,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER trigger_policies_updated_at
    BEFORE UPDATE ON policies
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(50) NOT NULL,  -- e.g., "policy_eval", "admin_action"
    user_id TEXT REFERENCES app_user(id) ON DELETE SET NULL,
    details JSONB NOT NULL,  -- e.g., {"decision": "allow", "reason": "matched policy X"}
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE delegations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delegator_id TEXT REFERENCES app_user(id) ON DELETE CASCADE,
    delegatee_id TEXT REFERENCES app_user(id) ON DELETE CASCADE,
    policy_id UUID REFERENCES policies(id) ON DELETE CASCADE,
    expiration TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Advanced Tables for Policy Management
CREATE TABLE policy_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID REFERENCES policies(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    rules JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE policy_conflicts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id1 UUID REFERENCES policies(id) ON DELETE CASCADE,
    policy_id2 UUID REFERENCES policies(id) ON DELETE CASCADE,
    conflict_type VARCHAR(50) NOT NULL,  -- e.g., "permit_deny", "overlapping"
    severity VARCHAR(20) NOT NULL,  -- e.g., "critical"
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE conflict_resolution_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conflict_id UUID REFERENCES policy_conflicts(id) ON DELETE CASCADE,
    action_taken VARCHAR(255) NOT NULL,  -- e.g., "merged policies"
    resolved_by TEXT REFERENCES app_user(id) ON DELETE SET NULL,
    resolved_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_app_user_attributes ON app_user USING GIN(attributes);
CREATE INDEX IF NOT EXISTS idx_resources_attributes ON resources USING GIN(attributes);
CREATE INDEX IF NOT EXISTS idx_policies_rules ON policies USING GIN(rules);
CREATE INDEX IF NOT EXISTS idx_policies_active ON policies(active, priority DESC);

-- +goose Down
DROP TABLE IF EXISTS conflict_resolution_actions;
DROP TABLE IF EXISTS policy_conflicts;
DROP TABLE IF EXISTS policy_versions;
DROP TABLE IF EXISTS delegations;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS policies;
DROP TABLE IF EXISTS resources;

DROP TRIGGER IF EXISTS trigger_app_user_updated_at ON app_user;
ALTER TABLE app_user DROP COLUMN IF EXISTS attributes;
ALTER TABLE app_user DROP COLUMN IF EXISTS updated_at;

DROP FUNCTION IF EXISTS set_updated_at();