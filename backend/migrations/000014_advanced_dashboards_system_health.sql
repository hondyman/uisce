-- Advanced Dashboard Features and System Health Monitoring
-- Created: 2025-09-10
-- Description: Enhanced dashboards, automated reporting, and comprehensive system monitoring

-- ===========================================
-- ADVANCED DASHBOARD FEATURES
-- =========================================--

-- CREATE TABLE IF NOT EXISTS for dashboard templates
CREATE TABLE IF NOT EXISTS public.pop_dashboard_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name TEXT NOT NULL,
    template_description TEXT,
    template_category TEXT NOT NULL, -- 'executive', 'operational', 'risk', 'compliance', 'custom'
    default_config JSONB NOT NULL,
    required_metrics TEXT[] NOT NULL,
    is_public BOOLEAN NOT NULL DEFAULT false,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- CREATE TABLE IF NOT EXISTS for custom dashboard widgets
CREATE TABLE IF NOT EXISTS public.pop_custom_widgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    widget_name TEXT NOT NULL,
    widget_type TEXT NOT NULL, -- 'custom_chart', 'custom_table', 'custom_kpi', 'custom_alert'
    widget_config JSONB NOT NULL,
    sql_query TEXT,
    refresh_interval_seconds INTEGER DEFAULT 300,
    data_source TEXT NOT NULL,
    created_by TEXT NOT NULL,
    is_public BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- CREATE TABLE IF NOT EXISTS for dashboard sharing and permissions
CREATE TABLE IF NOT EXISTS public.pop_dashboard_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID REFERENCES public.pop_dashboards(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    permission_level TEXT NOT NULL, -- 'view', 'edit', 'admin', 'owner'
    granted_by TEXT NOT NULL,
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(dashboard_id, user_id)
);

-- Sample dashboard templates
INSERT INTO public.pop_dashboard_templates (
    template_name, template_description, template_category, default_config,
    required_metrics, created_by
) VALUES
(
    'Executive Summary Dashboard',
    'High-level overview for executive leadership with key KPIs and trends',
    'executive',
    '{
        "layout": "grid",
        "theme": "corporate",
        "auto_refresh": true,
        "refresh_interval": 300,
        "widgets": [
            {"type": "kpi_cards", "position": {"x": 0, "y": 0, "width": 12, "height": 2}},
            {"type": "trend_chart", "position": {"x": 0, "y": 2, "width": 8, "height": 4}},
            {"type": "alert_summary", "position": {"x": 8, "y": 2, "width": 4, "height": 4}},
            {"type": "risk_heatmap", "position": {"x": 0, "y": 6, "width": 6, "height": 4}},
            {"type": "compliance_status", "position": {"x": 6, "y": 6, "width": 6, "height": 4}}
        ]
    }',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d479', 'f47ac10b-58cc-4372-a567-0e02b2c3d480', 'f47ac10b-58cc-4372-a567-0e02b2c3d482', 'f47ac10b-58cc-4372-a567-0e02b2c3d486']::uuid[],
    'dashboard.admin@company.com'
),
(
    'Risk Management Command Center',
    'Comprehensive risk monitoring and anomaly detection dashboard',
    'risk',
    '{
        "layout": "operational",
        "theme": "risk",
        "auto_refresh": true,
        "refresh_interval": 60,
        "widgets": [
            {"type": "risk_metrics_table", "position": {"x": 0, "y": 0, "width": 12, "height": 4}},
            {"type": "anomaly_timeline", "position": {"x": 0, "y": 4, "width": 8, "height": 4}},
            {"type": "stress_test_results", "position": {"x": 8, "y": 4, "width": 4, "height": 4}},
            {"type": "alert_panel", "position": {"x": 0, "y": 8, "width": 6, "height": 3}},
            {"type": "prediction_panel", "position": {"x": 6, "y": 8, "width": 6, "height": 3}}
        ]
    }',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d482', 'f47ac10b-58cc-4372-a567-0e02b2c3d491', 'f47ac10b-58cc-4372-a567-0e02b2c3d483', 'f47ac10b-58cc-4372-a567-0e02b2c3d490']::uuid[],
    'steward.risk@company.com'
);

-- Sample custom widgets
INSERT INTO public.pop_custom_widgets (
    widget_name, widget_type, widget_config, sql_query, data_source, created_by
) VALUES
(
    'AUM Growth Forecast',
    'custom_chart',
    '{
        "chart_type": "combo",
        "show_historical": true,
        "show_forecast": true,
        "forecast_periods": 12,
        "confidence_intervals": true
    }',
    'SELECT * FROM pop_predictions WHERE metric_id = (SELECT id FROM pop_metrics WHERE name = ''f47ac10b-58cc-4372-a567-0e02b2c3d479'') ORDER BY prediction_date',
    'analytics_db',
    'ml.engineer@company.com'
),
(
    'Compliance Risk Heatmap',
    'custom_chart',
    '{
        "chart_type": "heatmap",
        "color_scheme": "red_yellow_green",
        "group_by": "metric_category",
        "time_period": "quarter"
    }',
    'SELECT metric_id, compliance_score, risk_level FROM pop_compliance_predictions WHERE prediction_date >= CURRENT_DATE - INTERVAL ''90 days''',
    'compliance_db',
    'compliance@company.com'
);

-- ===========================================
-- AUTOMATED REPORTING SYSTEM
-- =========================================--

-- CREATE TABLE IF NOT EXISTS for report templates
CREATE TABLE IF NOT EXISTS public.pop_report_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name TEXT NOT NULL,
    template_description TEXT,
    report_type TEXT NOT NULL, -- 'daily', 'weekly', 'monthly', 'quarterly', 'annual'
    output_format TEXT NOT NULL, -- 'pdf', 'excel', 'html', 'json'
    schedule_config JSONB NOT NULL,
    recipient_list TEXT[] NOT NULL,
    metric_ids UUID[] NOT NULL,
    template_content JSONB,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- CREATE TABLE IF NOT EXISTS for generated reports
CREATE TABLE IF NOT EXISTS public.pop_generated_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID REFERENCES public.pop_report_templates(id) ON DELETE CASCADE,
    report_period_start DATE NOT NULL,
    report_period_end DATE NOT NULL,
    generation_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    file_path TEXT,
    file_size_bytes INTEGER,
    checksum TEXT,
    delivery_status TEXT DEFAULT 'pending', -- 'pending', 'sent', 'delivered', 'failed'
    delivery_attempts INTEGER DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample report templates
INSERT INTO public.pop_report_templates (
    template_name, template_description, report_type, output_format,
    schedule_config, recipient_list, metric_ids, created_by
) VALUES
(
    'Daily Risk Report',
    'Daily summary of risk metrics and anomalies',
    'daily',
    'pdf',
    '{
        "frequency": "daily",
        "time": "08:00",
        "timezone": "America/New_York",
        "business_days_only": true
    }',
    ARRAY['risk.team@company.com', 'ceo@company.com'],
    ARRAY[
        'f47ac10b-58cc-4372-a567-0e02b2c3d482',
        'f47ac10b-58cc-4372-a567-0e02b2c3d491',
        'f47ac10b-58cc-4372-a567-0e02b2c3d483'
    ]::uuid[]::UUID[],
    'reporting.admin@company.com'
),
(
    'Monthly Performance Report',
    'Comprehensive monthly performance analysis',
    'monthly',
    'excel',
    '{
        "frequency": "monthly",
        "day_of_month": 5,
        "time": "09:00",
        "timezone": "America/New_York"
    }',
    ARRAY['executives@company.com', 'board@company.com', 'investors@company.com'],
    ARRAY[
        'f47ac10b-58cc-4372-a567-0e02b2c3d479',
        'f47ac10b-58cc-4372-a567-0e02b2c3d480',
        'f47ac10b-58cc-4372-a567-0e02b2c3d489',
        'f47ac10b-58cc-4372-a567-0e02b2c3d498'
    ]::uuid[]::UUID[],
    'reporting.admin@company.com'
),
(
    'Quarterly Compliance Report',
    'Regulatory compliance status and findings',
    'quarterly',
    'pdf',
    '{
        "frequency": "quarterly",
        "day_of_quarter": 15,
        "time": "10:00",
        "timezone": "America/New_York"
    }',
    ARRAY['compliance@company.com', 'auditors@company.com', 'regulators@company.com'],
    ARRAY[
        'f47ac10b-58cc-4372-a567-0e02b2c3d486',
        'f47ac10b-58cc-4372-a567-0e02b2c3d495',
        'f47ac10b-58cc-4372-a567-0e02b2c3d496'
    ]::uuid[]::UUID[],
    'compliance@company.com'
);

-- ===========================================
-- SYSTEM HEALTH MONITORING
-- =========================================--

-- CREATE TABLE IF NOT EXISTS for system health metrics
CREATE TABLE IF NOT EXISTS public.pop_system_health (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    component_name TEXT NOT NULL,
    component_type TEXT NOT NULL, -- 'database', 'api', 'ml_model', 'external_api', 'dashboard'
    health_status TEXT NOT NULL, -- 'healthy', 'warning', 'critical', 'offline'
    health_score DECIMAL(5,4),
    response_time_ms INTEGER,
    error_rate DECIMAL(5,4),
    throughput INTEGER,
    last_check TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    next_check TIMESTAMP WITH TIME ZONE,
    alert_threshold JSONB,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- CREATE TABLE IF NOT EXISTS for system health alerts
CREATE TABLE IF NOT EXISTS public.pop_system_health_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    component_id UUID REFERENCES public.pop_system_health(id) ON DELETE CASCADE,
    alert_type TEXT NOT NULL, -- 'status_change', 'performance_degradation', 'error_rate_spike'
    severity TEXT NOT NULL,
    alert_message TEXT NOT NULL,
    threshold_breached TEXT,
    current_value TEXT,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_by TEXT,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample system health monitoring
INSERT INTO public.pop_system_health (
    component_name, component_type, health_status, health_score,
    response_time_ms, error_rate, throughput, alert_threshold
) VALUES
-- Database health
(
    'Primary PostgreSQL Database',
    'database',
    'healthy',
    0.98,
    45,
    0.002,
    1250,
    '{
        "response_time_ms": {"warning": 100, "critical": 500},
        "error_rate": {"warning": 0.01, "critical": 0.05},
        "health_score": {"warning": 0.95, "critical": 0.90}
    }'
),
-- API health
(
    'PoP Metrics API',
    'api',
    'healthy',
    0.97,
    125,
    0.005,
    450,
    '{
        "response_time_ms": {"warning": 200, "critical": 1000},
        "error_rate": {"warning": 0.02, "critical": 0.10},
        "health_score": {"warning": 0.95, "critical": 0.90}
    }'
),
-- ML model health
(
    'AUM Prediction Model',
    'ml_model',
    'healthy',
    0.94,
    NULL,
    NULL,
    NULL,
    '{
        "accuracy": {"warning": 0.85, "critical": 0.80},
        "drift_score": {"warning": 0.15, "critical": 0.25}
    }'
),
-- External API health
(
    'Bloomberg Market Data API',
    'external_api',
    'warning',
    0.89,
    850,
    0.025,
    95,
    '{
        "response_time_ms": {"warning": 500, "critical": 2000},
        "error_rate": {"warning": 0.03, "critical": 0.10},
        "health_score": {"warning": 0.90, "critical": 0.80}
    }'
);

-- ===========================================
-- PERFORMANCE ANALYTICS
-- =========================================--

-- CREATE TABLE IF NOT EXISTS for performance benchmarks
CREATE TABLE IF NOT EXISTS public.pop_performance_benchmarks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    benchmark_name TEXT NOT NULL,
    benchmark_category TEXT NOT NULL, -- 'system_performance', 'user_experience', 'data_quality'
    metric_name TEXT NOT NULL,
    target_value DECIMAL(20,6),
    warning_threshold DECIMAL(20,6),
    critical_threshold DECIMAL(20,6),
    comparison_operator TEXT NOT NULL, -- 'gt', 'lt', 'eq', 'gte', 'lte'
    measurement_period TEXT NOT NULL, -- 'real-time', 'hourly', 'daily', 'weekly'
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- CREATE TABLE IF NOT EXISTS for performance measurements
CREATE TABLE IF NOT EXISTS public.pop_performance_measurements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    benchmark_id UUID REFERENCES public.pop_performance_benchmarks(id) ON DELETE CASCADE,
    measured_value DECIMAL(20,6) NOT NULL,
    measurement_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status TEXT NOT NULL, -- 'normal', 'warning', 'critical'
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample performance benchmarks
INSERT INTO public.pop_performance_benchmarks (
    benchmark_name, benchmark_category, metric_name, target_value,
    warning_threshold, critical_threshold, comparison_operator, measurement_period, created_by
) VALUES
-- System performance benchmarks
(
    'API Response Time',
    'system_performance',
    'avg_response_time_ms',
    100.0,
    200.0,
    500.0,
    'lt',
    'real-time',
    'devops@company.com'
),
(
    'Dashboard Load Time',
    'user_experience',
    'dashboard_load_time_seconds',
    2.0,
    5.0,
    10.0,
    'lt',
    'real-time',
    'devops@company.com'
),
(
    'Data Freshness',
    'data_quality',
    'data_age_hours',
    4.0,
    12.0,
    24.0,
    'lt',
    'hourly',
    'data.engineer@company.com'
),
(
    'Report Generation Time',
    'system_performance',
    'report_generation_minutes',
    15.0,
    30.0,
    60.0,
    'lt',
    'daily',
    'reporting.admin@company.com'
);

-- ===========================================
-- AUDIT AND GOVERNANCE ENHANCEMENTS
-- =========================================--

-- CREATE TABLE IF NOT EXISTS for audit trails
CREATE TABLE IF NOT EXISTS public.pop_audit_trail (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL, -- 'metric', 'computation', 'anomaly', 'dashboard', 'report'
    resource_id UUID NOT NULL,
    action_details JSONB,
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    session_id TEXT
);

-- CREATE TABLE IF NOT EXISTS for data lineage tracking
CREATE TABLE IF NOT EXISTS public.pop_data_lineage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_table TEXT NOT NULL,
    source_column TEXT,
    target_table TEXT NOT NULL,
    target_column TEXT,
    transformation_rule TEXT,
    data_flow_direction TEXT NOT NULL, -- 'upstream', 'downstream'
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample audit trail entries
INSERT INTO public.pop_audit_trail (
    user_id, action, resource_type, resource_id, action_details, timestamp
) VALUES
(
    'steward.finance@company.com',
    'promote_to_golden_path',
    'metric',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    '{"previous_status": "active", "new_status": "golden_path", "reason": "Critical regulatory metric"}',
    '2024-09-09 10:30:00+00'
),
(
    'ml.engineer@company.com',
    'create_ml_model',
    'ml_model',
    (SELECT id FROM public.pop_predictive_models WHERE model_name = 'AUM Growth Predictor'),
    '{"model_type": "xgboost", "accuracy": 0.89}',
    '2024-09-09 14:15:00+00'
);

-- ===========================================
-- SUCCESS MESSAGE
-- =========================================--

DO $$
BEGIN
    RAISE NOTICE 'Advanced Dashboard Features and System Health Monitoring have been successfully implemented!';
    RAISE NOTICE 'Dashboard templates created: %', (SELECT COUNT(*) FROM public.pop_dashboard_templates);
    RAISE NOTICE 'Custom widgets defined: %', (SELECT COUNT(*) FROM public.pop_custom_widgets);
    RAISE NOTICE 'Report templates configured: %', (SELECT COUNT(*) FROM public.pop_report_templates);
    RAISE NOTICE 'System health monitoring: % components', (SELECT COUNT(*) FROM public.pop_system_health);
    RAISE NOTICE 'Performance benchmarks set: %', (SELECT COUNT(*) FROM public.pop_performance_benchmarks);
    RAISE NOTICE 'Audit trail initialized with % entries', (SELECT COUNT(*) FROM public.pop_audit_trail);
END $$;
