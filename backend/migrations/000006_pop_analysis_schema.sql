-- Period-over-Period (PoP) Analysis and Anomaly Detection Schema
-- Created: 2025-09-10
-- Description: Tables for PoP metrics, anomaly detection, and steward governance

-- ===========================================
-- PoP METRIC DEFINITIONS
-- ===========================================

-- Core PoP metric definitions
CREATE TABLE IF NOT EXISTS public.pop_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    domain TEXT NOT NULL, -- e.g., 'finance', 'operations', 'marketing'
    category TEXT NOT NULL, -- e.g., 'revenue', 'users', 'performance'
    metric_type TEXT NOT NULL, -- 'count', 'sum', 'avg', 'ratio', 'percentage'

    -- Formula and computation
    base_query TEXT NOT NULL, -- SQL query template
    aggregation_function TEXT NOT NULL, -- 'SUM', 'COUNT', 'AVG', etc.
    date_column TEXT NOT NULL, -- Column containing the date
    value_column TEXT NOT NULL, -- Column containing the metric value

    -- PoP configuration
    granularity TEXT NOT NULL DEFAULT 'month', -- 'day', 'week', 'month', 'quarter', 'year'
    comparison_periods JSONB NOT NULL DEFAULT '["previous_period", "year_over_year"]',

    -- Governance metadata
    owner_user_id TEXT NOT NULL,
    steward_group TEXT NOT NULL,
    data_source TEXT NOT NULL, -- Reference to datasource
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,

    -- Quality and SLA
    sla_freshness_hours INTEGER NOT NULL DEFAULT 24,
    sla_completeness_threshold DECIMAL(5,2) NOT NULL DEFAULT 0.95,
    data_quality_checks JSONB,

    -- Status and lifecycle
    status TEXT NOT NULL DEFAULT 'draft', -- 'draft', 'active', 'deprecated', 'golden'
    golden_path BOOLEAN NOT NULL DEFAULT FALSE,
    version INTEGER NOT NULL DEFAULT 1,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by TEXT NOT NULL,
    updated_by TEXT,

    UNIQUE(name, version)
);

-- PoP metric tags for categorization and filtering
CREATE TABLE IF NOT EXISTS public.pop_metric_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,
    tag_name TEXT NOT NULL,
    tag_value TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(metric_id, tag_name)
);

-- ===========================================
-- PoP COMPUTATION RESULTS
-- ===========================================

-- Computed PoP values for each metric and period
CREATE TABLE IF NOT EXISTS public.pop_computations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,

    -- Time period
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    granularity TEXT NOT NULL,
    period_label TEXT NOT NULL, -- e.g., '2024-Q1', '2024-09'

    -- Values
    current_value DECIMAL(20,6),
    previous_value DECIMAL(20,6),
    delta DECIMAL(20,6),
    percent_change DECIMAL(10,4),

    -- Metadata
    record_count INTEGER,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    computation_status TEXT NOT NULL DEFAULT 'success', -- 'success', 'error', 'partial'

    UNIQUE(metric_id, period_start, period_end, granularity)
);

-- ===========================================
-- ANOMALY DETECTION
-- ===========================================

-- Anomaly detection results
CREATE TABLE IF NOT EXISTS public.pop_anomalies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,
    computation_id UUID REFERENCES public.pop_computations(id) ON DELETE CASCADE,

    -- Anomaly details
    anomaly_type TEXT NOT NULL, -- 'z_score', 'iqr', 'trend_break', 'threshold'
    severity TEXT NOT NULL, -- 'low', 'medium', 'high', 'critical'
    confidence DECIMAL(5,4), -- 0.0 to 1.0

    -- Statistical measures
    z_score DECIMAL(10,4),
    expected_value DECIMAL(20,6),
    expected_range_min DECIMAL(20,6),
    expected_range_max DECIMAL(20,6),
    actual_value DECIMAL(20,6),

    -- Detection metadata
    detection_method TEXT NOT NULL,
    detection_params JSONB,
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Status and resolution
    status TEXT NOT NULL DEFAULT 'open', -- 'open', 'investigating', 'resolved', 'false_positive'
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by TEXT,
    resolution_notes TEXT,

    UNIQUE(metric_id, computation_id, anomaly_type)
);

-- ===========================================
-- STEWARD REVIEW AND GOVERNANCE
-- ===========================================

-- Steward review sessions
CREATE TABLE IF NOT EXISTS public.pop_steward_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,

    -- Review details
    review_period_start DATE NOT NULL,
    review_period_end DATE NOT NULL,
    reviewer_user_id TEXT NOT NULL,
    review_type TEXT NOT NULL, -- 'regular', 'anomaly_investigation', 'golden_path_review'

    -- Review content
    overall_rating TEXT, -- 'excellent', 'good', 'needs_attention', 'critical'
    review_notes TEXT,
    action_items JSONB, -- Array of action items with status

    -- Status
    status TEXT NOT NULL DEFAULT 'in_progress', -- 'in_progress', 'completed', 'overdue'
    due_date DATE,
    completed_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Steward comments and feedback
CREATE TABLE IF NOT EXISTS public.pop_steward_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id UUID REFERENCES public.pop_steward_reviews(id) ON DELETE CASCADE,
    anomaly_id UUID REFERENCES public.pop_anomalies(id) ON DELETE SET NULL,

    commenter_user_id TEXT NOT NULL,
    comment_type TEXT NOT NULL, -- 'general', 'anomaly_feedback', 'action_item', 'approval'
    comment_text TEXT NOT NULL,

    -- For threaded discussions
    parent_comment_id UUID REFERENCES public.pop_steward_comments(id) ON DELETE CASCADE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ===========================================
-- DASHBOARD AND REPORTING
-- ===========================================

-- Dashboard configurations for PoP cockpit
CREATE TABLE IF NOT EXISTS public.pop_dashboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    owner_user_id TEXT NOT NULL,

    -- Configuration
    config JSONB NOT NULL, -- Dashboard layout, filters, widgets
    default_filters JSONB, -- Default filter values
    refresh_schedule TEXT, -- Cron expression for auto-refresh

    -- Access control
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    allowed_users TEXT[], -- Array of user IDs
    allowed_groups TEXT[], -- Array of group names

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Dashboard widgets
CREATE TABLE IF NOT EXISTS public.pop_dashboard_widgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID REFERENCES public.pop_dashboards(id) ON DELETE CASCADE,

    widget_type TEXT NOT NULL, -- 'metric_table', 'trend_chart', 'anomaly_heatmap', 'kpi_cards'
    title TEXT NOT NULL,
    position JSONB NOT NULL, -- {x, y, width, height}

    -- Widget configuration
    config JSONB NOT NULL, -- Widget-specific settings
    metric_ids UUID[], -- For widgets that show specific metrics

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ===========================================
-- INDEXES AND CONSTRAINTS
-- ===========================================

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_pop_metrics_domain ON public.pop_metrics(domain);
CREATE INDEX IF NOT EXISTS idx_pop_metrics_status ON public.pop_metrics(status);
CREATE INDEX IF NOT EXISTS idx_pop_metrics_golden_path ON public.pop_metrics(golden_path);
CREATE INDEX IF NOT EXISTS idx_pop_metrics_owner ON public.pop_metrics(owner_user_id);

CREATE INDEX IF NOT EXISTS idx_pop_computations_metric_period ON public.pop_computations(metric_id, period_start, period_end);
CREATE INDEX IF NOT EXISTS idx_pop_computations_granularity ON public.pop_computations(granularity);

CREATE INDEX IF NOT EXISTS idx_pop_anomalies_metric ON public.pop_anomalies(metric_id);
CREATE INDEX IF NOT EXISTS idx_pop_anomalies_status ON public.pop_anomalies(status);
CREATE INDEX IF NOT EXISTS idx_pop_anomalies_severity ON public.pop_anomalies(severity);

CREATE INDEX IF NOT EXISTS idx_pop_steward_reviews_metric ON public.pop_steward_reviews(metric_id);
CREATE INDEX IF NOT EXISTS idx_pop_steward_reviews_reviewer ON public.pop_steward_reviews(reviewer_user_id);
CREATE INDEX IF NOT EXISTS idx_pop_steward_reviews_status ON public.pop_steward_reviews(status);

-- ===========================================
-- VIEWS FOR COMMON QUERIES
-- ===========================================

-- View for active PoP metrics with latest computation
CREATE OR REPLACE VIEW public.pop_metrics_with_latest AS
SELECT
    m.*,
    c.current_value,
    c.previous_value,
    c.delta,
    c.percent_change,
    c.period_start,
    c.period_end,
    c.last_updated as last_computed_at,
    CASE
        WHEN a.id IS NOT NULL THEN TRUE
        ELSE FALSE
    END as has_anomalies,
    COUNT(a.id) as anomaly_count
FROM public.pop_metrics m
LEFT JOIN public.pop_computations c ON m.id = c.metric_id
    AND c.id = (
        SELECT id FROM public.pop_computations
        WHERE metric_id = m.id
        ORDER BY period_end DESC, last_updated DESC
        LIMIT 1
    )
LEFT JOIN public.pop_anomalies a ON m.id = a.metric_id
    AND a.status = 'open'
WHERE m.status = 'active'
GROUP BY m.id, c.id, c.current_value, c.previous_value, c.delta, c.percent_change, c.period_start, c.period_end, c.last_updated, a.id;

-- View for anomaly summary by domain and severity
CREATE OR REPLACE VIEW public.pop_anomaly_summary AS
SELECT
    m.domain,
    m.category,
    a.severity,
    a.anomaly_type,
    COUNT(*) as anomaly_count,
    MAX(a.detected_at) as latest_detection,
    ARRAY_AGG(DISTINCT m.name) as affected_metrics
FROM public.pop_anomalies a
JOIN public.pop_metrics m ON a.metric_id = m.id
WHERE a.status = 'open'
GROUP BY m.domain, m.category, a.severity, a.anomaly_type
ORDER BY m.domain, m.category, a.severity;

-- ===========================================
-- FUNCTIONS AND TRIGGERS
-- ===========================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION public.update_pop_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
DROP TRIGGER IF EXISTS pop_metrics_updated_at ON public;
CREATE TRIGGER pop_metrics_updated_at
    BEFORE UPDATE ON public.pop_metrics
    FOR EACH ROW EXECUTE FUNCTION public.update_pop_updated_at();

DROP TRIGGER IF EXISTS pop_steward_reviews_updated_at ON public;
CREATE TRIGGER pop_steward_reviews_updated_at
    BEFORE UPDATE ON public.pop_steward_reviews
    FOR EACH ROW EXECUTE FUNCTION public.update_pop_updated_at();

DROP TRIGGER IF EXISTS pop_dashboards_updated_at ON public;
CREATE TRIGGER pop_dashboards_updated_at
    BEFORE UPDATE ON public.pop_dashboards
    FOR EACH ROW EXECUTE FUNCTION public.update_pop_updated_at();

-- Function to automatically create steward review for anomalies
CREATE OR REPLACE FUNCTION public.create_anomaly_review()
RETURNS TRIGGER AS $$
BEGIN
    -- Only create review for high/critical anomalies
    IF NEW.severity IN ('high', 'critical') AND NEW.status = 'open' THEN
        INSERT INTO public.pop_steward_reviews (
            metric_id,
            review_period_start,
            review_period_end,
            reviewer_user_id,
            review_type,
            due_date
        )
        SELECT
            NEW.metric_id,
            c.period_start,
            c.period_end,
            m.owner_user_id,
            'anomaly_investigation',
            NOW() + INTERVAL '7 days'
        FROM public.pop_computations c
        JOIN public.pop_metrics m ON c.metric_id = m.id
        WHERE c.id = NEW.computation_id
        ON CONFLICT DO NOTHING;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for automatic anomaly review creation
CREATE TRIGGER create_anomaly_review_trigger
    AFTER INSERT ON public.pop_anomalies
    FOR EACH ROW EXECUTE FUNCTION public.create_anomaly_review();
