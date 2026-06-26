-- Risk Management Schema Extension
-- Integrates AI risk detection, ABAC controls, and automated mitigation
-- Created: October 30, 2025

-- ============================================================================
-- ENUMS FOR RISK MANAGEMENT
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'risk_event_type') THEN
        CREATE TYPE risk_event_type AS ENUM (
            'CONCENTRATION',
            'VAR_BREACH',
            'LIQUIDITY_SHORTAGE',
            'COMPLIANCE_VIOLATION',
            'ESG_ALERT',
            'GEOPOLITICAL',
            'MARKET_SHOCK',
            'COUNTERPARTY_RISK',
            'OPERATIONAL_RISK'
        );
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'risk_severity') THEN
        CREATE TYPE risk_severity AS ENUM (
            'LOW',
            'MEDIUM',
            'HIGH',
            'CRITICAL'
        );
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'risk_status') THEN
        CREATE TYPE risk_status AS ENUM (
            'DETECTED',
            'ACKNOWLEDGED',
            'MITIGATING',
            'MITIGATED',
            'ACCEPTED',
            'ESCALATED',
            'CLOSED'
        );
    END IF;
END$$;

-- ============================================================================
-- TABLE: risk_events
-- Purpose: Core risk detection events with AI scores and mitigation tracking
-- ============================================================================
CREATE TABLE IF NOT EXISTS risk_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Multi-tenant isolation
    tenant_id UUID NOT NULL,
    
    -- Entity relationships
    portfolio_entity_id UUID,
    position_id UUID,
    client_entity_id UUID,
    household_entity_id UUID,
    
    -- Risk classification
    event_type risk_event_type NOT NULL,
    severity risk_severity NOT NULL,
    status risk_status DEFAULT 'DETECTED',
    
    -- AI-generated scores (beats Addepar)
    risk_score NUMERIC(5, 2) NOT NULL,  -- 0-10 scale
    confidence_score NUMERIC(5, 2),      -- AI confidence 0-1
    var_95 NUMERIC(18, 2),               -- Value at Risk 95%
    cvar_95 NUMERIC(18, 2),              -- Conditional VaR 95%
    
    -- Metrics
    current_exposure NUMERIC(18, 2),
    concentration_pct NUMERIC(5, 2),
    liquidity_ratio NUMERIC(8, 4),
    
    -- AI insights
    ai_reasoning TEXT,
    ai_recommendations JSONB,
    
    -- Mitigation
    mitigation_strategy TEXT,
    mitigation_actions JSONB,
    auto_mitigated BOOLEAN DEFAULT FALSE,
    mitigated_at TIMESTAMP,
    mitigated_by VARCHAR(255),
    
    -- Workflow tracking
    workflow_id VARCHAR(100),  -- Temporal workflow ID
    workflow_run_id VARCHAR(100),
    business_process_instance_id UUID,
    
    -- Escalation
    escalated_to VARCHAR(255),
    escalated_at TIMESTAMP,
    escalation_reason TEXT,
    
    -- Metadata
    detected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMP,
    acknowledged_by VARCHAR(255),
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    
    CONSTRAINT risk_events_tenant_check CHECK (tenant_id IS NOT NULL)
);

CREATE INDEX idx_risk_events_tenant ON risk_events(tenant_id);
CREATE INDEX idx_risk_events_portfolio ON risk_events(portfolio_entity_id);
CREATE INDEX idx_risk_events_status ON risk_events(status);
CREATE INDEX idx_risk_events_severity ON risk_events(severity);
CREATE INDEX idx_risk_events_detected ON risk_events(detected_at DESC);
CREATE INDEX idx_risk_events_score ON risk_events(risk_score DESC);
CREATE INDEX idx_risk_events_workflow ON risk_events(workflow_id) WHERE workflow_id IS NOT NULL;
CREATE INDEX idx_risk_events_composite ON risk_events(tenant_id, portfolio_entity_id, status);

-- ============================================================================
-- TABLE: risk_thresholds
-- Purpose: Workday-style configurable thresholds for automated triggers
-- ============================================================================
CREATE TABLE IF NOT EXISTS risk_thresholds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Multi-tenant
    tenant_id UUID NOT NULL,
    
    -- Scope
    scope VARCHAR(50) NOT NULL,  -- 'GLOBAL', 'PORTFOLIO', 'CLIENT', 'ASSET_CLASS'
    scope_entity_id UUID,
    
    -- Threshold configuration
    risk_type risk_event_type NOT NULL,
    threshold_name VARCHAR(100) NOT NULL,
    
    -- Thresholds
    warning_threshold NUMERIC(18, 2),
    critical_threshold NUMERIC(18, 2),
    
    -- Actions
    auto_mitigate BOOLEAN DEFAULT FALSE,
    require_approval BOOLEAN DEFAULT TRUE,
    escalate_to_roles TEXT[],
    
    -- Temporal windows (Workday-style)
    active_hours INT[] DEFAULT ARRAY[9,10,11,12,13,14,15,16,17],
    active_days INT[] DEFAULT ARRAY[1,2,3,4,5],  -- Monday-Friday
    timezone VARCHAR(50) DEFAULT 'America/New_York',
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255)
);

CREATE INDEX idx_risk_thresholds_tenant ON risk_thresholds(tenant_id);
CREATE INDEX idx_risk_thresholds_scope ON risk_thresholds(scope, scope_entity_id);
CREATE INDEX idx_risk_thresholds_active ON risk_thresholds(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_risk_thresholds_type ON risk_thresholds(risk_type);

-- ============================================================================
-- TABLE: risk_mitigation_actions
-- Purpose: Track individual mitigation actions executed by workflows
-- ============================================================================
CREATE TABLE IF NOT EXISTS risk_mitigation_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    risk_event_id UUID NOT NULL,
    
    -- Action details
    action_type VARCHAR(50) NOT NULL,  -- 'REBALANCE', 'HEDGE', 'LIQUIDATE', 'NOTIFY', 'BLOCK_TRADES'
    action_description TEXT,
    action_parameters JSONB,
    
    -- Execution
    executed_at TIMESTAMP,
    executed_by VARCHAR(255),
    execution_result JSONB,
    
    -- Status
    status VARCHAR(20) DEFAULT 'PENDING',  -- 'PENDING', 'EXECUTING', 'COMPLETED', 'FAILED'
    
    -- Error tracking
    error_message TEXT,
    error_details JSONB,
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_risk_event FOREIGN KEY (risk_event_id) REFERENCES risk_events(id) ON DELETE CASCADE
);

CREATE INDEX idx_risk_mitigation_event ON risk_mitigation_actions(risk_event_id);
CREATE INDEX idx_risk_mitigation_status ON risk_mitigation_actions(status);
CREATE INDEX idx_risk_mitigation_type ON risk_mitigation_actions(action_type);

-- ============================================================================
-- TABLE: risk_metrics_history
-- Purpose: Store historical risk metrics snapshots for trending and analysis
-- ============================================================================
CREATE TABLE IF NOT EXISTS risk_metrics_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    tenant_id UUID NOT NULL,
    portfolio_entity_id UUID NOT NULL,
    
    -- Metrics snapshot
    as_of_date DATE NOT NULL DEFAULT CURRENT_DATE,
    as_of_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Risk scores
    overall_risk_score NUMERIC(5, 2),
    var_95 NUMERIC(18, 2),
    cvar_95 NUMERIC(18, 2),
    sharpe_ratio NUMERIC(8, 4),
    sortino_ratio NUMERIC(8, 4),
    max_drawdown NUMERIC(8, 4),
    
    -- Concentration metrics
    top_10_concentration NUMERIC(5, 2),
    herfindahl_index NUMERIC(8, 6),
    
    -- Liquidity metrics
    liquidity_ratio NUMERIC(8, 4),
    illiquid_percentage NUMERIC(5, 2),
    
    -- Additional metrics
    beta NUMERIC(8, 4),
    tracking_error NUMERIC(8, 4),
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_risk_history_portfolio ON risk_metrics_history(portfolio_entity_id);
CREATE INDEX idx_risk_history_date ON risk_metrics_history(as_of_date DESC);
CREATE INDEX idx_risk_history_tenant ON risk_metrics_history(tenant_id, as_of_date DESC);

-- ============================================================================
-- TABLE: risk_abac_policies
-- Purpose: ABAC policies for risk mitigation authorization
-- ============================================================================
CREATE TABLE IF NOT EXISTS risk_abac_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    tenant_id UUID NOT NULL,
    
    -- Policy definition
    policy_name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Attributes
    attributes JSONB NOT NULL,  -- Subject, resource, action, environment attributes
    effect VARCHAR(10) DEFAULT 'ALLOW',  -- 'ALLOW' or 'DENY'
    
    -- Priority for conflict resolution
    priority INTEGER DEFAULT 100,
    
    -- Temporal constraints
    valid_from TIMESTAMP,
    valid_until TIMESTAMP,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255)
);

CREATE INDEX idx_risk_abac_tenant ON risk_abac_policies(tenant_id);
CREATE INDEX idx_risk_abac_active ON risk_abac_policies(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_risk_abac_priority ON risk_abac_policies(priority DESC);

-- ============================================================================
-- TABLE: risk_event_audit_trail
-- Purpose: Immutable audit log for risk events and actions
-- ============================================================================
CREATE TABLE IF NOT EXISTS risk_event_audit_trail (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    risk_event_id UUID NOT NULL,
    
    -- Audit info
    action VARCHAR(100) NOT NULL,  -- 'CREATED', 'ACKNOWLEDGED', 'MITIGATED', 'ESCALATED', etc.
    actor_email VARCHAR(255) NOT NULL,
    actor_role VARCHAR(100),
    
    -- Details
    changes JSONB,  -- JSON diff of what changed
    reason TEXT,
    
    -- Metadata
    ip_address VARCHAR(45),
    user_agent TEXT,
    
    -- Timestamp
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_audit_risk_event FOREIGN KEY (risk_event_id) REFERENCES risk_events(id) ON DELETE CASCADE
);

CREATE INDEX idx_risk_audit_event ON risk_event_audit_trail(risk_event_id);
CREATE INDEX idx_risk_audit_action ON risk_event_audit_trail(action);
CREATE INDEX idx_risk_audit_actor ON risk_event_audit_trail(actor_email);
CREATE INDEX idx_risk_audit_created ON risk_event_audit_trail(created_at DESC);

-- ============================================================================
-- VIEW: v_portfolio_risk_dashboard
-- Purpose: Real-time risk dashboard data (matches provider dashboard patterns)
-- ============================================================================
DO $$
BEGIN
    -- Only create the view if the base table 'entities' exists to avoid errors when running against partial schemas.
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'entities') THEN
        EXECUTE $view$
        CREATE OR REPLACE VIEW v_portfolio_risk_dashboard AS
        SELECT 
                p.id AS portfolio_entity_id,
                p.display_name AS portfolio_name,
                p.tenant_id,
        
                -- Current risk score (last 24 hours)
                COALESCE(
                        (SELECT AVG(risk_score) 
                         FROM risk_events 
                         WHERE portfolio_entity_id = p.id 
                             AND status IN ('DETECTED', 'ACKNOWLEDGED')
                             AND detected_at > CURRENT_TIMESTAMP - INTERVAL '24 hours'
                        ), 0
                ) AS current_risk_score,
        
                -- Active alerts
                (SELECT COUNT(*) 
                 FROM risk_events 
                 WHERE portfolio_entity_id = p.id 
                     AND status IN ('DETECTED', 'ACKNOWLEDGED')
                ) AS active_alerts,
        
                -- Critical alerts
                (SELECT COUNT(*) 
                 FROM risk_events 
                 WHERE portfolio_entity_id = p.id 
                     AND severity = 'CRITICAL'
                     AND status IN ('DETECTED', 'ACKNOWLEDGED')
                ) AS critical_alerts,
        
                -- Mitigated count (last 30 days)
                (SELECT COUNT(*) 
                 FROM risk_events 
                 WHERE portfolio_entity_id = p.id 
                     AND status = 'MITIGATED'
                     AND mitigated_at > CURRENT_TIMESTAMP - INTERVAL '30 days'
                ) AS mitigated_last_30d,
        
                -- Latest metrics
                rmh.var_95,
                rmh.cvar_95,
                rmh.sharpe_ratio,
                rmh.liquidity_ratio,
                rmh.top_10_concentration,
                rmh.herfindahl_index,
        
                -- Latest risk event
                (SELECT json_build_object(
                        'id', id,
                        'event_type', event_type,
                        'severity', severity,
                        'risk_score', risk_score,
                        'detected_at', detected_at,
                        'ai_reasoning', ai_reasoning
                 )
                 FROM risk_events 
                 WHERE portfolio_entity_id = p.id 
                 ORDER BY detected_at DESC 
                 LIMIT 1
                ) AS latest_risk_event,
        
                -- Auto-mitigated percentage (last 90 days)
                COALESCE(
                        ROUND(
                                100.0 * (SELECT COUNT(*) FROM risk_events WHERE portfolio_entity_id = p.id AND auto_mitigated = true AND detected_at > CURRENT_TIMESTAMP - INTERVAL '90 days')
                                / NULLIF((SELECT COUNT(*) FROM risk_events WHERE portfolio_entity_id = p.id AND detected_at > CURRENT_TIMESTAMP - INTERVAL '90 days'), 0),
                                1
                        ), 0
                ) AS auto_mitigation_rate
        
        FROM entities p
        LEFT JOIN LATERAL (
                SELECT * FROM risk_metrics_history 
                WHERE portfolio_entity_id = p.id 
                ORDER BY as_of_date DESC, as_of_time DESC
                LIMIT 1
        ) rmh ON TRUE
        WHERE p.model_type = 'PORTFOLIO' OR p.entity_type = 'PORTFOLIO';
        $view$;
    ELSE
        RAISE NOTICE 'Skipping v_portfolio_risk_dashboard: base table "entities" not present';
    END IF;
END$$;

-- ============================================================================
-- FUNCTION: update_risk_event_timestamp
-- Purpose: Auto-update updated_at on risk_events
-- ============================================================================
CREATE OR REPLACE FUNCTION update_risk_event_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER risk_events_updated_at 
    BEFORE UPDATE ON risk_events
    FOR EACH ROW EXECUTE FUNCTION update_risk_event_timestamp();

-- ============================================================================
-- ROW-LEVEL SECURITY (if enabled in your Hasura)
-- ============================================================================
-- Uncomment if using Hasura RLS:
-- ALTER TABLE risk_events ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE risk_thresholds ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE risk_mitigation_actions ENABLE ROW LEVEL SECURITY;
-- 
-- CREATE POLICY risk_events_tenant_isolation ON risk_events
--     FOR SELECT
--     USING (tenant_id = current_setting('hasura.user.x-hasura-tenant-id', TRUE)::UUID);
-- 
-- CREATE POLICY risk_thresholds_tenant_isolation ON risk_thresholds
--     FOR SELECT
--     USING (tenant_id = current_setting('hasura.user.x-hasura-tenant-id', TRUE)::UUID);

-- ============================================================================
-- COMMENTS FOR DOCUMENTATION
-- ============================================================================
COMMENT ON TABLE risk_events IS 'Core risk detection events with AI scores and automated mitigation tracking';
COMMENT ON TABLE risk_thresholds IS 'Configurable thresholds for automated risk detection triggers';
COMMENT ON TABLE risk_mitigation_actions IS 'Individual mitigation actions executed during risk response';
COMMENT ON TABLE risk_metrics_history IS 'Historical snapshots for risk trending and analysis';
COMMENT ON TABLE risk_abac_policies IS 'ABAC policies for risk mitigation authorization';
COMMENT ON TABLE risk_event_audit_trail IS 'Immutable audit trail of all risk events and actions';

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.views WHERE table_name = 'v_portfolio_risk_dashboard') THEN
        EXECUTE 'COMMENT ON VIEW v_portfolio_risk_dashboard IS ''Real-time portfolio risk dashboard aggregation for UI'';';
    ELSE
        RAISE NOTICE 'Skipping COMMENT ON v_portfolio_risk_dashboard: view not present';
    END IF;
END$$;
