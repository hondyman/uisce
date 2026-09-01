-- 20260922_mcp_agentic_audit.up.sql
-- MCP Tool Execution Audit Ledger for AI Agentic Governance

CREATE SCHEMA IF NOT EXISTS catalog_mdm_ai;

CREATE TABLE IF NOT EXISTS catalog_mdm_ai.mcp_tool_execution_logs (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    tool_name VARCHAR(100) NOT NULL,
    request_parameters JSONB NOT NULL,
    response_data JSONB,
    execution_duration_ms INT NOT NULL,
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error_message TEXT,
    merkle_receipt VARCHAR(64) NOT NULL,
    executed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mcp_tool_audit 
ON catalog_mdm_ai.mcp_tool_execution_logs (tenant_id, tool_name, executed_at DESC);
