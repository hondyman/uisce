-- PoP Cockpit Dashboard Configuration
-- Created: 2025-09-10
-- Description: Complete dashboard setup for the Period-over-Period metrics cockpit

-- ===========================================
-- DASHBOARD CONFIGURATION
-- ===========================================

-- Main PoP Cockpit Dashboard
INSERT INTO public.pop_dashboards (
    id, name, description, owner_user_id,
    config, default_filters, refresh_schedule,
    is_public, allowed_users, allowed_groups
) VALUES (
    'd142512a-3505-4c6e-8121-5079540b7274',
    'PoP Metrics Cockpit',
    'Comprehensive dashboard for monitoring Period-over-Period metrics, anomalies, and governance',
    'admin@company.com',
    '{
        "layout": "responsive",
        "theme": "corporate",
        "auto_refresh": true,
        "refresh_interval": 300,
        "alerts_enabled": true,
        "notification_channels": ["email", "slack"],
        "timezone": "America/New_York",
        "widgets": {
            "header": {
                "show_title": true,
                "show_last_updated": true,
                "show_user_info": true
            },
            "navigation": {
                "show_domain_filter": true,
                "show_status_filter": true,
                "show_time_range": true
            }
        }
    }',
    '{"domain": "finance", "status": "active"}',
    '*/5 * * * *',
    false,
    ARRAY['stewards', 'analysts', 'managers'],
    ARRAY['finance_team', 'risk_team', 'operations_team']
);

-- ===========================================
-- DASHBOARD WIDGETS
-- ===========================================

-- KPI Summary Cards
INSERT INTO public.pop_dashboard_widgets (
    dashboard_id, widget_type, title, position, config, metric_ids
) VALUES
(
    'd142512a-3505-4c6e-8121-5079540b7274',
    'kpi_cards',
    'Executive Summary',
    '{"x": 0, "y": 0, "width": 12, "height": 3}',
    '{
        "layout": "horizontal",
        "show_trend": true,
        "show_comparison": true,
        "show_targets": true,
        "cards": [
            {
                "metric": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
                "title": "Total AUM",
                "format": "currency",
                "unit": "M",
                "target": 120000,
                "good_threshold": 110000
            },
            {
                "metric": "f47ac10b-58cc-4372-a567-0e02b2c3d480",
                "title": "NAV Growth",
                "format": "percentage",
                "unit": "%",
                "target": 2.0,
                "good_threshold": 1.0
            },
            {
                "metric": "f47ac10b-58cc-4372-a567-0e02b2c3d481",
                "title": "Net Inflows",
                "format": "currency",
                "unit": "M",
                "target": 5000,
                "good_threshold": 1000
            }
        ]
    }',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d479', 'f47ac10b-58cc-4372-a567-0e02b2c3d480', 'f47ac10b-58cc-4372-a567-0e02b2c3d481']::uuid[]
),

-- Anomaly Heatmap
(
    'd142512a-3505-4c6e-8121-5079540b7274',
    'anomaly_heatmap',
    'Anomaly Overview',
    '{"x": 0, "y": 3, "width": 6, "height": 4}',
    '{
        "group_by": "domain",
        "severity_levels": ["critical", "high", "medium", "low"],
        "color_scheme": "red_green",
        "show_trend": true,
        "time_window": "30d",
        "auto_refresh": true
    }',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d479', 'f47ac10b-58cc-4372-a567-0e02b2c3d480', 'f47ac10b-58cc-4372-a567-0e02b2c3d481', 'f47ac10b-58cc-4372-a567-0e02b2c3d482', 'f47ac10b-58cc-4372-a567-0e02b2c3d483']::uuid[]
),

-- Trend Charts
(
    'd142512a-3505-4c6e-8121-5079540b7274',
    'trend_chart',
    'AUM Growth Trends',
    '{"x": 6, "y": 3, "width": 6, "height": 4}',
    '{
        "chart_type": "line",
        "period": "12months",
        "show_forecast": true,
        "show_bands": true,
        "confidence_interval": 0.95,
        "annotations": {
            "show_anomalies": true,
            "show_events": true
        }
    }',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d479']::uuid[]
),

-- Risk Metrics Panel
(
    'd142512a-3505-4c6e-8121-5079540b7274',
    'metric_table',
    'Risk & Performance Metrics',
    '{"x": 0, "y": 7, "width": 8, "height": 5}',
    '{
        "columns": ["name", "current_value", "change_pct", "anomaly_status", "last_updated"],
        "sortable": true,
        "filterable": true,
        "show_anomalies": true,
        "alert_on_threshold": true,
        "thresholds": {
            "f47ac10b-58cc-4372-a567-0e02b2c3d482": {"warning": 18, "critical": 22},
            "f47ac10b-58cc-4372-a567-0e02b2c3d483": {"warning": 0.8, "critical": 0.5}
        }
    }',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d482', 'f47ac10b-58cc-4372-a567-0e02b2c3d483', 'f47ac10b-58cc-4372-a567-0e02b2c3d480']::uuid[]
),

-- Operations Metrics
(
    'd142512a-3505-4c6e-8121-5079540b7274',
    'combo_chart',
    'Operational Efficiency',
    '{"x": 8, "y": 7, "width": 4, "height": 5}',
    '{
        "primary_metric": "f47ac10b-58cc-4372-a567-0e02b2c3d484",
        "secondary_metric": "f47ac10b-58cc-4372-a567-0e02b2c3d485",
        "chart_type": "bar_line",
        "period": "7days",
        "show_sla_targets": true,
        "targets": {
            "f47ac10b-58cc-4372-a567-0e02b2c3d485": 3.0
        }
    }',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d484', 'f47ac10b-58cc-4372-a567-0e02b2c3d485']::uuid[]
),

-- Governance Status
(
    'd142512a-3505-4c6e-8121-5079540b7274',
    'status_panel',
    'Governance Overview',
    '{"x": 0, "y": 12, "width": 6, "height": 3}',
    '{
        "show_golden_path": true,
        "show_anomaly_count": true,
        "show_review_status": true,
        "show_compliance_score": true,
        "group_by": "domain"
    }',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d479', 'f47ac10b-58cc-4372-a567-0e02b2c3d480', 'f47ac10b-58cc-4372-a567-0e02b2c3d482', 'f47ac10b-58cc-4372-a567-0e02b2c3d486']::uuid[]
),

-- Recent Activity Feed
(
    'd142512a-3505-4c6e-8121-5079540b7274',
    'activity_feed',
    'Recent Activity',
    '{"x": 6, "y": 12, "width": 6, "height": 3}',
    '{
        "max_items": 10,
        "show_types": ["anomaly", "review", "comment", "promotion"],
        "time_window": "7days",
        "group_by_type": true
    }',
    NULL
);

-- ===========================================
-- ALERT CONFIGURATIONS
-- ===========================================

-- Create a table for dashboard alerts (if not exists)
CREATE TABLE IF NOT EXISTS public.pop_dashboard_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID REFERENCES public.pop_dashboards(id) ON DELETE CASCADE,
    alert_name TEXT NOT NULL,
    alert_type TEXT NOT NULL, -- 'threshold', 'anomaly', 'trend', 'missing_data'
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,
    condition_config JSONB NOT NULL,
    severity TEXT NOT NULL DEFAULT 'medium',
    enabled BOOLEAN NOT NULL DEFAULT true,
    notification_channels TEXT[] DEFAULT ARRAY['dashboard'],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample alerts for the cockpit
INSERT INTO public.pop_dashboard_alerts (
    dashboard_id, alert_name, alert_type, metric_id, condition_config, severity, notification_channels
) VALUES
(
    'd142512a-3505-4c6e-8121-5079540b7274',
    'High AUM Volatility',
    'threshold',
    'f47ac10b-58cc-4372-a567-0e02b2c3d482',
    '{
        "operator": ">",
        "value": 20.0,
        "duration": "2periods",
        "cooldown": "1hour"
    }',
    'high',
    ARRAY['dashboard', 'email', 'slack']
),
(
    'd142512a-3505-4c6e-8121-5079540b7274',
    'Negative Net Inflows',
    'threshold',
    'f47ac10b-58cc-4372-a567-0e02b2c3d481',
    '{
        "operator": "<",
        "value": 0,
        "duration": "1period",
        "cooldown": "4hours"
    }',
    'medium',
    ARRAY['dashboard', 'email']
),
(
    'd142512a-3505-4c6e-8121-5079540b7274',
    'Processing Time SLA Breach',
    'threshold',
    'f47ac10b-58cc-4372-a567-0e02b2c3d485',
    '{
        "operator": ">",
        "value": 3.5,
        "duration": "3periods",
        "cooldown": "30minutes"
    }',
    'high',
    ARRAY['dashboard', 'slack']
),
(
    'd142512a-3505-4c6e-8121-5079540b7274',
    'Compliance Filing Delay',
    'threshold',
    'f47ac10b-58cc-4372-a567-0e02b2c3d486',
    '{
        "operator": "<",
        "value": 95.0,
        "duration": "1period",
        "cooldown": "1day"
    }',
    'critical',
    ARRAY['dashboard', 'email', 'slack']
);

-- ===========================================
-- USER PREFERENCES
-- ===========================================

-- Create a table for user dashboard preferences
CREATE TABLE IF NOT EXISTS public.pop_user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    dashboard_id UUID REFERENCES public.pop_dashboards(id) ON DELETE CASCADE,
    preferences JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, dashboard_id)
);

-- Sample user preferences
INSERT INTO public.pop_user_preferences (user_id, dashboard_id, preferences) VALUES
(
    'steward.finance@company.com',
    'd142512a-3505-4c6e-8121-5079540b7274',
    '{
        "theme": "dark",
        "default_domain": "finance",
        "auto_refresh": true,
        "alert_preferences": {
            "email": true,
            "slack": false,
            "dashboard": true
        },
        "widget_layout": {
            "kpi_cards": {"visible": true, "position": "top"},
            "anomaly_heatmap": {"visible": true, "expanded": true},
            "trend_charts": {"visible": true, "default_period": "6months"}
        }
    }'
),
(
    'steward.risk@company.com',
    'd142512a-3505-4c6e-8121-5079540b7274',
    '{
        "theme": "corporate",
        "default_domain": "finance",
        "auto_refresh": true,
        "alert_preferences": {
            "email": true,
            "slack": true,
            "dashboard": true
        },
        "widget_layout": {
            "risk_metrics": {"visible": true, "expanded": true},
            "anomaly_heatmap": {"visible": true, "filter_severity": ["high", "critical"]}
        }
    }'
);

-- ===========================================
-- DASHBOARD ACCESS POLICIES
-- ===========================================

-- Create a view for dashboard access control
CREATE OR REPLACE VIEW public.dashboard_access_view AS
SELECT
    d.id,
    d.name,
    d.owner_user_id,
    d.is_public,
    d.allowed_users,
    d.allowed_groups,
    CASE
        WHEN d.is_public THEN true
        WHEN current_user_id = ANY(d.allowed_users) THEN true
        WHEN current_user_groups && d.allowed_groups THEN true
        ELSE false
    END as has_access
FROM public.pop_dashboards d,
LATERAL (
    SELECT
        session_user as current_user_id,
        ARRAY[]::TEXT[] as current_user_groups -- In practice, this would be populated from user management system
) as user_info;

-- ===========================================
-- DASHBOARD PERFORMANCE METRICS
-- ===========================================

-- Create a table to track dashboard usage and performance
CREATE TABLE IF NOT EXISTS public.pop_dashboard_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID REFERENCES public.pop_dashboards(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL, -- 'view', 'refresh', 'export', 'alert_acknowledged'
    session_id TEXT,
    user_agent TEXT,
    ip_address INET,
    response_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for performance
CREATE INDEX IF NOT EXISTS idx_dashboard_usage_dashboard_user ON public.pop_dashboard_usage(dashboard_id, user_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_usage_created_at ON public.pop_dashboard_usage(created_at);

-- ===========================================
-- SUCCESS MESSAGE
-- ===========================================

DO $$
BEGIN
    RAISE NOTICE 'PoP Cockpit Dashboard Configuration has been successfully created!';
    RAISE NOTICE 'Dashboard ID: d142512a-3505-4c6e-8121-5079540b7274';
    RAISE NOTICE 'Configured widgets: KPI Cards, Anomaly Heatmap, Trend Charts, Risk Metrics, Operations Metrics, Governance Status, Activity Feed';
    RAISE NOTICE 'Configured alerts: 4 threshold-based alerts';
    RAISE NOTICE 'User preferences: Configured for finance and risk stewards';
END $$;
