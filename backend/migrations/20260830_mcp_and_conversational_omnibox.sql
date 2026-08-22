-- backend/migrations/20260830_mcp_and_conversational_omnibox.sql
-- Model Context Protocol (MCP) Server Tool Registry, AI Execution Logs & Omnibox Sessions

CREATE SCHEMA IF NOT EXISTS catalog_ai;

-- 1. Conversational Query & Omnibox Search Sessions (Rule 6: Semantic/OLTP Boundary)
CREATE TABLE IF NOT EXISTS catalog_ai.conversational_query_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id VARCHAR(100) NOT NULL,
    user_prompt TEXT NOT NULL,
    resolved_intent VARCHAR(50) NOT NULL, -- DATA_QUERY, MDM_TRIAGE, CATALOG_SEARCH, SCHEMA_DRIFT
    generated_ast JSONB,
    generated_sql TEXT,
    complexity_score INT DEFAULT 0,
    cost_band VARCHAR(20) DEFAULT 'LOW',
    execution_status VARCHAR(30) DEFAULT 'PLANNED', -- PLANNED, EXECUTED, REJECTED, BLOCKED
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_sessions_tenant 
ON catalog_ai.conversational_query_sessions (tenant_id, user_id, created_at DESC);

-- 2. Model Context Protocol (MCP) Tool Execution Ledger (SEC Rule 17a-4 Compliance)
CREATE TABLE IF NOT EXISTS catalog_ai.mcp_tool_execution_logs (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    session_id UUID REFERENCES catalog_ai.conversational_query_sessions(session_id) ON DELETE SET NULL,
    tool_name VARCHAR(100) NOT NULL,      -- "triage_mdm_exception", "explain_survivorship", "execute_semantic_query"
    invoked_by_actor VARCHAR(100) NOT NULL, -- "AI_ASSISTANT", "DATA_STEWARD", "API_CLIENT"
    input_parameters JSONB NOT NULL,
    output_result JSONB NOT NULL,
    execution_duration_ms INT NOT NULL,
    is_success BOOLEAN NOT NULL DEFAULT TRUE,
    error_message TEXT,
    payload_sha256 VARCHAR(64) NOT NULL,
    executed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mcp_tool_logs 
ON catalog_ai.mcp_tool_execution_logs (tenant_id, tool_name, executed_at DESC);
