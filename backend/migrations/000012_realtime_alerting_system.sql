-- Real-Time Alerting and Notification System
-- Created: 2025-09-10
-- Description: Automated alerting for critical metric thresholds and anomalies

-- ===========================================
-- ALERT CONFIGURATION TABLES
-- =========================================--

-- Create table for alert rules
CREATE TABLE IF NOT EXISTS public.pop_alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,
    rule_name TEXT NOT NULL,
    rule_description TEXT,
    condition_type TEXT NOT NULL, -- 'threshold', 'percentage_change', 'z_score', 'trend_break'
    condition_params JSONB NOT NULL,
    severity TEXT NOT NULL, -- 'low', 'medium', 'high', 'critical'
    notification_channels TEXT[] NOT NULL, -- ['email', 'slack', 'sms', 'dashboard']
    recipient_groups TEXT[] NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    cooldown_minutes INTEGER DEFAULT 60, -- Minimum time between alerts
    created_by TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create table for alert instances
CREATE TABLE IF NOT EXISTS public.pop_alert_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID REFERENCES public.pop_alert_rules(id) ON DELETE CASCADE,
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,
    computation_id UUID REFERENCES public.pop_computations(id) ON DELETE CASCADE,
    anomaly_id UUID REFERENCES public.pop_anomalies(id) ON DELETE SET NULL,
    alert_message TEXT NOT NULL,
    alert_value DECIMAL(20,6),
    threshold_value DECIMAL(20,6),
    triggered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_by TEXT,
    resolved_at TIMESTAMP WITH TIME ZONE,
    status TEXT NOT NULL DEFAULT 'active', -- 'active', 'acknowledged', 'resolved', 'false_positive'
    escalation_level INTEGER DEFAULT 0
);

-- Create table for alert notifications
CREATE TABLE IF NOT EXISTS public.pop_alert_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_instance_id UUID REFERENCES public.pop_alert_instances(id) ON DELETE CASCADE,
    channel TEXT NOT NULL,
    recipient TEXT NOT NULL,
    message TEXT NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    delivery_status TEXT DEFAULT 'pending', -- 'pending', 'sent', 'delivered', 'failed'
    error_message TEXT
);

-- ===========================================
-- SAMPLE ALERT RULES
-- =========================================--

INSERT INTO public.pop_alert_rules (
    metric_id, rule_name, rule_description, condition_type, condition_params,
    severity, notification_channels, recipient_groups, cooldown_minutes, created_by
) VALUES
-- Critical AUM alerts
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'AUM Critical Decline Alert',
    'Alert when AUM drops more than 5% in a single month',
    'percentage_change',
    '{"threshold": -5.0, "direction": "below", "period": "month"}',
    'critical',
    ARRAY['email', 'slack', 'sms'],
    ARRAY['executives', 'finance_team', 'risk_team'],
    15,
    'steward.finance@company.com'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'AUM Significant Growth Alert',
    'Alert when AUM grows more than 10% in a single month',
    'percentage_change',
    '{"threshold": 10.0, "direction": "above", "period": "month"}',
    'high',
    ARRAY['email', 'slack'],
    ARRAY['executives', 'finance_team', 'marketing'],
    60,
    'steward.finance@company.com'
),

-- Risk management alerts
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d482',
    'High Volatility Alert',
    'Alert when 30-day volatility exceeds 25%',
    'threshold',
    '{"threshold": 25.0, "direction": "above"}',
    'high',
    ARRAY['email', 'slack'],
    ARRAY['risk_team', 'portfolio_managers'],
    30,
    'steward.risk@company.com'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d491',
    'Severe Drawdown Alert',
    'Alert when maximum drawdown exceeds 15%',
    'threshold',
    '{"threshold": -15.0, "direction": "below"}',
    'critical',
    ARRAY['email', 'slack', 'sms'],
    ARRAY['executives', 'risk_team', 'compliance'],
    5,
    'steward.risk@company.com'
),

-- Operational alerts
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d484',
    'Low Transaction Volume Alert',
    'Alert when daily transaction volume drops below 10,000',
    'threshold',
    '{"threshold": 10000, "direction": "below"}',
    'medium',
    ARRAY['email', 'slack'],
    ARRAY['operations_team', 'it_team'],
    120,
    'steward.ops@company.com'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d485',
    'Processing Time Degradation Alert',
    'Alert when average processing time exceeds 3 seconds',
    'threshold',
    '{"threshold": 3.0, "direction": "above"}',
    'high',
    ARRAY['email', 'slack'],
    ARRAY['operations_team', 'it_team'],
    30,
    'steward.ops@company.com'
),

-- Compliance alerts
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d486',
    'Compliance Filing Delay Alert',
    'Alert when compliance filing completion drops below 95%',
    'threshold',
    '{"threshold": 95.0, "direction": "below"}',
    'critical',
    ARRAY['email', 'slack', 'sms'],
    ARRAY['compliance_team', 'executives', 'legal'],
    60,
    'steward.compliance@company.com'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d495',
    'Regulatory Fine Alert',
    'Alert when any regulatory fine is incurred',
    'threshold',
    '{"threshold": 0, "direction": "above"}',
    'critical',
    ARRAY['email', 'slack', 'sms'],
    ARRAY['compliance_team', 'executives', 'legal', 'board'],
    1,
    'steward.compliance@company.com'
),

-- Client experience alerts
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d493',
    'Client Satisfaction Decline Alert',
    'Alert when client satisfaction drops below 8.0',
    'threshold',
    '{"threshold": 8.0, "direction": "below"}',
    'high',
    ARRAY['email', 'slack'],
    ARRAY['client_services', 'executives'],
    1440, -- Daily alert
    'steward.client@company.com'
);

-- ===========================================
-- SAMPLE ALERT INSTANCES
-- =========================================--

INSERT INTO public.pop_alert_instances (
    rule_id, metric_id, computation_id, anomaly_id, alert_message,
    alert_value, threshold_value, status
) VALUES
-- AUM decline alert
(
    (SELECT id FROM public.pop_alert_rules WHERE rule_name = 'AUM Critical Decline Alert'),
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d479' AND period_label = '2024-08'),
    NULL,
    'AUM Critical Decline Alert: AUM decreased by 5.93% in August 2024, from $118.00M to $125.00M',
    125000.50,
    118000.25,
    'acknowledged'
),
-- High volatility alert
(
    (SELECT id FROM public.pop_alert_rules WHERE rule_name = 'High Volatility Alert'),
    'f47ac10b-58cc-4372-a567-0e02b2c3d482',
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d482' AND period_label = '2024-08'),
    (SELECT id FROM public.pop_anomalies WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d482'),
    'High Volatility Alert: 30-day volatility reached 18.75%, exceeding threshold of 25%',
    18.75,
    25.0,
    'active'
),
-- Regulatory fine alert
(
    (SELECT id FROM public.pop_alert_rules WHERE rule_name = 'Regulatory Fine Alert'),
    'f47ac10b-58cc-4372-a567-0e02b2c3d495',
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d495' AND period_label = '2024-07'),
    (SELECT id FROM public.pop_anomalies WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d495'),
    'Regulatory Fine Alert: $25,000 fine incurred in July 2024 for late Form 13F filing',
    25000.00,
    0.0,
    'resolved'
);

-- ===========================================
-- ALERT ESCALATION RULES
-- =========================================--

-- Create table for alert escalation policies
CREATE TABLE IF NOT EXISTS public.pop_alert_escalations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID REFERENCES public.pop_alert_rules(id) ON DELETE CASCADE,
    escalation_level INTEGER NOT NULL,
    delay_minutes INTEGER NOT NULL,
    additional_recipients TEXT[] NOT NULL,
    additional_channels TEXT[] NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample escalation policies
INSERT INTO public.pop_alert_escalations (
    rule_id, escalation_level, delay_minutes, additional_recipients, additional_channels
) VALUES
-- Critical AUM alert escalation
(
    (SELECT id FROM public.pop_alert_rules WHERE rule_name = 'AUM Critical Decline Alert'),
    1,
    30,
    ARRAY['ceo@company.com', 'cfo@company.com'],
    ARRAY['sms']
),
(
    (SELECT id FROM public.pop_alert_rules WHERE rule_name = 'AUM Critical Decline Alert'),
    2,
    60,
    ARRAY['board_chair@company.com'],
    ARRAY['sms']
),

-- Compliance alert escalation
(
    (SELECT id FROM public.pop_alert_rules WHERE rule_name = 'Regulatory Fine Alert'),
    1,
    15,
    ARRAY['general_counsel@company.com'],
    ARRAY['sms']
),
(
    (SELECT id FROM public.pop_alert_rules WHERE rule_name = 'Regulatory Fine Alert'),
    2,
    30,
    ARRAY['board_compliance_committee@company.com'],
    ARRAY['sms']
);

-- ===========================================
-- ALERT DASHBOARD INTEGRATION
-- =========================================--

-- Create table for alert dashboard widgets
CREATE TABLE IF NOT EXISTS public.pop_alert_dashboard_widgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID REFERENCES public.pop_dashboards(id) ON DELETE CASCADE,
    widget_type TEXT NOT NULL, -- 'active_alerts', 'alert_history', 'alert_summary'
    title TEXT NOT NULL,
    position JSONB NOT NULL,
    config JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample alert dashboard widgets
INSERT INTO public.pop_alert_dashboard_widgets (
    dashboard_id, widget_type, title, position, config
) VALUES
-- Risk Management Dashboard
(
    (SELECT id FROM public.pop_dashboards WHERE name = 'Risk Management Cockpit'),
    'active_alerts',
    'Active Risk Alerts',
    '{"x": 0, "y": 8, "width": 6, "height": 4}',
    '{
        "severity_filter": ["high", "critical"],
        "max_alerts": 10,
        "auto_refresh": true,
        "refresh_interval": 30
    }'
),
(
    (SELECT id FROM public.pop_dashboards WHERE name = 'Risk Management Cockpit'),
    'alert_history',
    'Risk Alert History (7 days)',
    '{"x": 6, "y": 8, "width": 6, "height": 4}',
    '{
        "days_back": 7,
        "group_by": "severity",
        "chart_type": "timeline"
    }'
),

-- Operations Dashboard
(
    (SELECT id FROM public.pop_dashboards WHERE name = 'Operations Dashboard'),
    'active_alerts',
    'Active Operational Alerts',
    '{"x": 0, "y": 4, "width": 12, "height": 3}',
    '{
        "severity_filter": ["medium", "high", "critical"],
        "max_alerts": 15,
        "auto_refresh": true,
        "refresh_interval": 60
    }'
),

-- Executive Dashboard
(
    (SELECT id FROM public.pop_dashboards WHERE name = 'Executive Dashboard'),
    'alert_summary',
    'Critical Alerts Summary',
    '{"x": 0, "y": 6, "width": 12, "height": 2}',
    '{
        "show_counts": true,
        "show_trends": true,
        "severity_breakdown": true
    }'
);

-- ===========================================
-- ALERT ANALYTICS AND REPORTING
-- =========================================--

-- Create table for alert analytics
CREATE TABLE IF NOT EXISTS public.pop_alert_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL,
    total_alerts INTEGER NOT NULL DEFAULT 0,
    alerts_by_severity JSONB NOT NULL DEFAULT '{}',
    alerts_by_metric JSONB NOT NULL DEFAULT '{}',
    average_response_time_minutes DECIMAL(10,2),
    false_positive_rate DECIMAL(5,4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(date)
);

-- Sample alert analytics
INSERT INTO public.pop_alert_analytics (
    date, total_alerts, alerts_by_severity, alerts_by_metric,
    average_response_time_minutes, false_positive_rate
) VALUES
(
    '2024-09-09',
    12,
    '{"low": 3, "medium": 4, "high": 3, "critical": 2}',
    '{"f47ac10b-58cc-4372-a567-0e02b2c3d479": 2, "f47ac10b-58cc-4372-a567-0e02b2c3d482": 3, "f47ac10b-58cc-4372-a567-0e02b2c3d486": 2, "f47ac10b-58cc-4372-a567-0e02b2c3d484": 2, "f47ac10b-58cc-4372-a567-0e02b2c3d485": 3}',
    45.5,
    0.15
),
(
    '2024-09-08',
    8,
    '{"low": 2, "medium": 3, "high": 2, "critical": 1}',
    '{"f47ac10b-58cc-4372-a567-0e02b2c3d482": 2, "f47ac10b-58cc-4372-a567-0e02b2c3d484": 1, "f47ac10b-58cc-4372-a567-0e02b2c3d485": 2, "f47ac10b-58cc-4372-a567-0e02b2c3d493": 3}',
    38.2,
    0.12
);

-- ===========================================
-- SUCCESS MESSAGE
-- =========================================--

DO $$
BEGIN
    RAISE NOTICE 'Real-Time Alerting System has been successfully implemented!';
    RAISE NOTICE 'Alert rules created: %', (SELECT COUNT(*) FROM public.pop_alert_rules);
    RAISE NOTICE 'Sample alerts generated: %', (SELECT COUNT(*) FROM public.pop_alert_instances);
    RAISE NOTICE 'Escalation policies defined: %', (SELECT COUNT(*) FROM public.pop_alert_escalations);
    RAISE NOTICE 'Dashboard widgets added: %', (SELECT COUNT(*) FROM public.pop_alert_dashboard_widgets);
    RAISE NOTICE 'Alert analytics initialized: %', (SELECT COUNT(*) FROM public.pop_alert_analytics);
END $$;
