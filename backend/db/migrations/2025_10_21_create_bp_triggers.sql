-- Migration: create BP triggers, executions, steps and business process tables
-- Run this migration when enabling the BP triggers feature

CREATE TABLE IF NOT EXISTS business_processes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    process_name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bp_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    process_id UUID NOT NULL REFERENCES business_processes(id) ON DELETE CASCADE,
    step_order INT NOT NULL,
    step_type TEXT NOT NULL,
    step_name TEXT NOT NULL,
    duration_hours INT DEFAULT 0,
    assignee_role TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bp_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    trigger_name TEXT NOT NULL,
    trigger_type TEXT NOT NULL,
    enabled BOOLEAN DEFAULT true,
    event_config JSONB,
    condition_config JSONB,
    target_process_id UUID,
    priority INT DEFAULT 100,
    notification_config JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bp_trigger_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger_id UUID NOT NULL REFERENCES bp_triggers(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    workflow_id TEXT,
    execution_status TEXT NOT NULL,
    trigger_payload JSONB,
    error_message TEXT,
    execution_time_ms BIGINT,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_bp_triggers_tenant ON bp_triggers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bp_triggers_process ON bp_triggers(target_process_id);
CREATE INDEX IF NOT EXISTS idx_bp_executions_trigger ON bp_trigger_executions(trigger_id);
-- Migration: Create BP Triggers and Execution Logs
-- Date: 2025-10-21
-- Purpose: Store Business Process triggers, executions, and metrics

CREATE TABLE IF NOT EXISTS bp_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    trigger_name VARCHAR(255) NOT NULL,
    trigger_type VARCHAR(30) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    event_config JSONB,
    schedule_config JSONB,
    threshold_config JSONB,
    condition_config JSONB,
    escalation_config JSONB,
    dependency_config JSONB,
    sentiment_config JSONB,
    external_config JSONB,
    target_process_id UUID,
    action_type VARCHAR(20) DEFAULT 'start',
    priority INT DEFAULT 5,
    retry_config JSONB DEFAULT '{"max_attempts":3, "backoff_multiplier":2}',
    rate_limit_config JSONB,
    notification_config JSONB,
    execution_count BIGINT DEFAULT 0,
    last_executed_at TIMESTAMP WITH TIME ZONE,
    avg_execution_time_ms INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bp_trigger_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    workflow_id VARCHAR(255),
    execution_status VARCHAR(20),
    trigger_payload JSONB,
    result JSONB,
    execution_time_ms INT,
    error_message TEXT,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_bp_trigger FOREIGN KEY (trigger_id) REFERENCES bp_triggers(id) ON DELETE CASCADE
);

-- Materialized view for trigger metrics (refresh as needed)
CREATE MATERIALIZED VIEW IF NOT EXISTS bp_trigger_metrics AS
SELECT
  t.id,
  t.trigger_name,
  t.trigger_type,
  COUNT(e.id) as total_executions,
  COUNT(CASE WHEN e.execution_status = 'completed' THEN 1 END) as successful_executions,
  COUNT(CASE WHEN e.execution_status = 'failed' THEN 1 END) as failed_executions,
  AVG(e.execution_time_ms) as avg_execution_time_ms,
  MAX(e.executed_at) as last_execution
FROM bp_triggers t
LEFT JOIN bp_trigger_executions e ON t.id = e.trigger_id
GROUP BY t.id, t.trigger_name, t.trigger_type;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_bp_triggers_tenant ON bp_triggers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bp_triggers_type ON bp_triggers(trigger_type) WHERE enabled = true;
CREATE INDEX IF NOT EXISTS idx_bp_trigger_executions_status ON bp_trigger_executions(execution_status, executed_at);
CREATE INDEX IF NOT EXISTS idx_bp_trigger_executions_workflow ON bp_trigger_executions(workflow_id);

-- Notes: Refresh materialized view periodically or on-demand:
-- REFRESH MATERIALIZED VIEW CONCURRENTLY bp_trigger_metrics;
