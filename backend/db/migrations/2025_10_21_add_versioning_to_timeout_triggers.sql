-- Migration: Add Versioning and Metadata to Workflow Timeout Triggers
-- Date: 2025-10-21
-- Purpose: Enable version control, audit trails, approvals, and collaboration features

-- Add versioning columns to main table
ALTER TABLE workflow_timeout_triggers
ADD COLUMN IF NOT EXISTS version INT DEFAULT 1,
ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('draft', 'active', 'deprecated')),
ADD COLUMN IF NOT EXISTS created_by UUID,
ADD COLUMN IF NOT EXISTS modified_by UUID,
ADD COLUMN IF NOT EXISTS description TEXT,
ADD COLUMN IF NOT EXISTS tags JSONB DEFAULT '[]',
ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';

-- Version History Table
CREATE TABLE IF NOT EXISTS workflow_timeout_trigger_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    version INT NOT NULL,
    
    -- Trigger data snapshot
    workflow_name VARCHAR(100) NOT NULL,
    step_name VARCHAR(100) NOT NULL,
    due_hours INT NOT NULL,
    trigger_percentages JSONB,
    actions_json JSONB NOT NULL,
    is_active BOOLEAN,
    
    -- Change tracking
    changes JSONB DEFAULT '[]',  -- Array of what changed
    change_summary TEXT,          -- Human-readable summary
    author_id UUID,
    author_email VARCHAR(255),
    author_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT fk_trigger_versions FOREIGN KEY (trigger_id) 
        REFERENCES workflow_timeout_triggers(id) ON DELETE CASCADE,
    UNIQUE(trigger_id, version)
);

-- Approval Requests Table
CREATE TABLE IF NOT EXISTS workflow_timeout_trigger_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    version INT NOT NULL,
    
    -- Approval workflow
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    requested_by_id UUID,
    requested_by_email VARCHAR(255),
    requested_by_name VARCHAR(100),
    requested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Approval chain
    approvers JSONB DEFAULT '[]',  -- Array of {id, email, name, status, timestamp}
    rejection_reason TEXT,
    approved_at TIMESTAMP WITH TIME ZONE,
    rejected_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT fk_approval_trigger FOREIGN KEY (trigger_id) 
        REFERENCES workflow_timeout_triggers(id) ON DELETE CASCADE
);

-- Comments/Collaboration Table
CREATE TABLE IF NOT EXISTS workflow_timeout_trigger_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    
    -- Comment data
    content TEXT NOT NULL,
    author_id UUID,
    author_email VARCHAR(255),
    author_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Threading
    parent_comment_id UUID REFERENCES workflow_timeout_trigger_comments(id) ON DELETE CASCADE,
    
    -- Mentions
    mentioned_users JSONB DEFAULT '[]',  -- Array of user IDs mentioned
    
    CONSTRAINT fk_comment_trigger FOREIGN KEY (trigger_id) 
        REFERENCES workflow_timeout_triggers(id) ON DELETE CASCADE
);

-- Audit Log Table
CREATE TABLE IF NOT EXISTS workflow_timeout_trigger_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    
    -- Action info
    action VARCHAR(50) NOT NULL,  -- 'create', 'update', 'delete', 'restore', 'approve', 'reject'
    details JSONB,
    
    -- Actor info
    actor_id UUID,
    actor_email VARCHAR(255),
    actor_name VARCHAR(100),
    actor_role VARCHAR(50),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT fk_audit_trigger FOREIGN KEY (trigger_id) 
        REFERENCES workflow_timeout_triggers(id) ON DELETE CASCADE
);

-- Test Results Table
CREATE TABLE IF NOT EXISTS workflow_timeout_trigger_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    
    -- Test info
    test_case_name VARCHAR(255) NOT NULL,
    input_data JSONB NOT NULL,
    expected_result VARCHAR(10),  -- 'pass' or 'fail'
    actual_result VARCHAR(10),
    status VARCHAR(20) DEFAULT 'pending',  -- 'pending', 'running', 'passed', 'failed'
    error_message TEXT,
    
    -- Execution info
    execution_time_ms INT,
    run_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    runner_id UUID,
    runner_email VARCHAR(255),
    
    CONSTRAINT fk_test_trigger FOREIGN KEY (trigger_id) 
        REFERENCES workflow_timeout_triggers(id) ON DELETE CASCADE
);

-- Test Suites Table
CREATE TABLE IF NOT EXISTS workflow_timeout_trigger_test_suites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    
    -- Suite info
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Results
    total_tests INT DEFAULT 0,
    passed_tests INT DEFAULT 0,
    failed_tests INT DEFAULT 0,
    pass_rate DECIMAL(5, 2),
    
    -- Execution
    last_run_at TIMESTAMP WITH TIME ZONE,
    last_run_duration_ms INT,
    created_by_id UUID,
    created_by_email VARCHAR(255),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT fk_suite_trigger FOREIGN KEY (trigger_id) 
        REFERENCES workflow_timeout_triggers(id) ON DELETE CASCADE
);

-- Analytics Table
CREATE TABLE IF NOT EXISTS workflow_timeout_trigger_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    
    -- Metrics
    total_invocations BIGINT DEFAULT 0,
    successful_invocations BIGINT DEFAULT 0,
    failed_invocations BIGINT DEFAULT 0,
    success_rate DECIMAL(5, 2),
    
    -- Performance
    avg_execution_time_ms DECIMAL(10, 2),
    min_execution_time_ms INT,
    max_execution_time_ms INT,
    
    -- Trends
    last_30_days_invocations BIGINT DEFAULT 0,
    last_30_days_success_rate DECIMAL(5, 2),
    
    measured_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT fk_analytics_trigger FOREIGN KEY (trigger_id) 
        REFERENCES workflow_timeout_triggers(id) ON DELETE CASCADE
);

-- Create Indexes
CREATE INDEX IF NOT EXISTS idx_versions_trigger_id ON workflow_timeout_trigger_versions(trigger_id);
CREATE INDEX IF NOT EXISTS idx_versions_tenant_id ON workflow_timeout_trigger_versions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_approvals_trigger_id ON workflow_timeout_trigger_approvals(trigger_id);
CREATE INDEX IF NOT EXISTS idx_approvals_status ON workflow_timeout_trigger_approvals(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_comments_trigger_id ON workflow_timeout_trigger_comments(trigger_id);
CREATE INDEX IF NOT EXISTS idx_audit_trigger_id ON workflow_timeout_trigger_audit(trigger_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON workflow_timeout_trigger_audit(tenant_id, action);
CREATE INDEX IF NOT EXISTS idx_tests_trigger_id ON workflow_timeout_trigger_tests(trigger_id);
CREATE INDEX IF NOT EXISTS idx_test_suites_trigger_id ON workflow_timeout_trigger_test_suites(trigger_id);
CREATE INDEX IF NOT EXISTS idx_analytics_trigger_id ON workflow_timeout_trigger_analytics(trigger_id);

-- Add index for version history
CREATE INDEX IF NOT EXISTS idx_trigger_status ON workflow_timeout_triggers(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_trigger_created_by ON workflow_timeout_triggers(created_by);

-- Update existing records with defaults
UPDATE workflow_timeout_triggers 
SET version = 1, status = 'active', created_by = '00000000-0000-0000-0000-000000000001'
WHERE version IS NULL;
