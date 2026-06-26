-- 1. AI Usage Logs (Usage Analytics)
CREATE TABLE ai_usage_logs (
    log_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(100) NOT NULL,
    session_id UUID NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    query_type VARCHAR(50), -- 'portfolio_review', 'client_question', 'research'
    tokens_used INTEGER,
    latency_ms INTEGER,
    data_sources_accessed TEXT[], -- ['crm', 'portfolio_db', 'market_data']
    guardrails_triggered TEXT[],
    client_id VARCHAR(100),
    cost_estimate DECIMAL(10,4)
);

CREATE INDEX idx_usage_user_time ON ai_usage_logs (user_id, timestamp);
CREATE INDEX idx_usage_session ON ai_usage_logs (session_id);

-- 2. Compliance Summary View (Reporting)
-- Note: This assumes existence of 'events_raw' from 012_glass_box.sql
-- We map 'events_raw' fields to the blueprint's expected structure
CREATE VIEW compliance_summary AS
SELECT 
    DATE_TRUNC('day', timestamp) as date,
    COUNT(*) FILTER (WHERE event_type = 'GUARDRAILS_EVALUATED' AND payload_canon LIKE '%"allowed":false%') as blocked_requests,
    COUNT(*) FILTER (WHERE event_type = 'GUARDRAILS_EVALUATED' AND payload_canon LIKE '%"requires_human":true%') as flagged_requests,
    COUNT(*) FILTER (WHERE event_type = 'GUARDRAILS_EVALUATED' AND payload_canon LIKE '%"violations":[%') as violation_events,
    COUNT(DISTINCT run_id) as unique_workflows
FROM events_raw
GROUP BY DATE_TRUNC('day', timestamp);

-- 3. SEC Report Generation Function
-- Returns a table of generated recommendations and their approvals
CREATE OR REPLACE FUNCTION generate_sec_report(start_date TIMESTAMP WITH TIME ZONE, end_date TIMESTAMP WITH TIME ZONE)
RETURNS TABLE(
    workflow_id UUID,
    client_id TEXT,
    recommendation TEXT,
    approval_timestamp TIMESTAMP WITH TIME ZONE,
    approver_id TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        e.run_id as workflow_id,
        e.payload_canon::jsonb->>'client_id' as client_id,
        e.payload_canon::jsonb->>'draft_content' as recommendation,
        a.timestamp as approval_timestamp,
        a.payload_canon::jsonb->>'actor_id' as approver_id
    FROM events_raw e
    JOIN events_raw a ON e.run_id = a.run_id
    WHERE e.event_type = 'WORKFLOW_STARTED' -- Using start event to get client_id
      AND a.event_type = 'ADVISOR_SIGNAL_RECEIVED'
      AND a.payload_canon::jsonb->>'action' = 'approve'
      AND a.timestamp BETWEEN start_date AND end_date;
END;
$$ LANGUAGE plpgsql;
