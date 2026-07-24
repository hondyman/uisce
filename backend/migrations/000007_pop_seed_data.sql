-- PoP Metrics Seed Data for Mutual Fund Company
-- Created: 2025-09-10
-- Description: Sample data for financial services PoP metrics, anomaly detection, and governance

-- ===========================================
-- MUTUAL FUND METRICS DEFINITIONS
-- ===========================================

-- Core AUM (Assets Under Management) Metrics
INSERT INTO public.pop_metrics (
    id, name, display_name, description, domain, category, metric_type,
    base_query, aggregation_function, date_column, value_column,
    granularity, comparison_periods,
    owner_user_id, steward_group, data_source, schema_name, table_name,
    sla_freshness_hours, sla_completeness_threshold,
    status, golden_path, version, created_by
) VALUES
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'total_aum',
    'Total Assets Under Management',
    'Total AUM across all funds and client accounts',
    'finance',
    'assets',
    'sum',
    'SELECT date, total_aum FROM daily_fund_metrics WHERE fund_type = ''all''',
    'SUM',
    'date',
    'total_aum',
    'month',
    '["previous_period", "year_over_year", "quarter_over_quarter"]',
    'steward.finance@company.com',
    'Finance Operations',
    'fund_database',
    'analytics',
    'daily_fund_metrics',
    24,
    0.99,
    'active',
    true,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d480',
    'nav_growth_rate',
    'NAV Growth Rate',
    'Monthly growth rate of Net Asset Value',
    'finance',
    'performance',
    'ratio',
    'SELECT date, nav_growth_pct FROM monthly_performance_metrics',
    'AVG',
    'date',
    'nav_growth_pct',
    'month',
    '["previous_period", "year_over_year"]',
    'steward.investment@company.com',
    'Investment Operations',
    'fund_database',
    'analytics',
    'monthly_performance_metrics',
    48,
    0.98,
    'active',
    true,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d481',
    'net_inflows',
    'Net Inflows/Outflows',
    'Net capital inflows minus outflows',
    'finance',
    'capital_flows',
    'sum',
    'SELECT date, net_inflows FROM daily_capital_flows',
    'SUM',
    'date',
    'net_inflows',
    'month',
    '["previous_period", "year_over_year"]',
    'steward.client@company.com',
    'Client Operations',
    'fund_database',
    'analytics',
    'daily_capital_flows',
    24,
    0.95,
    'active',
    false,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d482',
    'volatility_30d',
    '30-Day Volatility',
    '30-day rolling volatility of fund returns',
    'finance',
    'risk',
    'ratio',
    'SELECT date, volatility_30d FROM risk_metrics',
    'AVG',
    'date',
    'volatility_30d',
    'month',
    '["previous_period", "year_over_year"]',
    'steward.risk@company.com',
    'Risk Management',
    'fund_database',
    'analytics',
    'risk_metrics',
    72,
    0.90,
    'active',
    true,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d483',
    'sharpe_ratio',
    'Sharpe Ratio',
    'Risk-adjusted return measure',
    'finance',
    'performance',
    'ratio',
    'SELECT date, sharpe_ratio FROM performance_ratios',
    'AVG',
    'date',
    'sharpe_ratio',
    'quarter',
    '["previous_period", "year_over_year"]',
    'steward.investment@company.com',
    'Investment Operations',
    'fund_database',
    'analytics',
    'performance_ratios',
    168,
    0.85,
    'active',
    false,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d484',
    'transaction_volume',
    'Daily Transaction Volume',
    'Number of transactions processed daily',
    'operations',
    'transactions',
    'count',
    'SELECT date, transaction_count FROM operational_metrics',
    'SUM',
    'date',
    'transaction_count',
    'day',
    '["previous_period", "week_over_week"]',
    'steward.ops@company.com',
    'Operations',
    'transaction_db',
    'operations',
    'operational_metrics',
    6,
    0.99,
    'active',
    true,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d485',
    'avg_processing_time',
    'Average Processing Time',
    'Average time to process transactions',
    'operations',
    'efficiency',
    'avg',
    'SELECT date, avg_processing_seconds FROM operational_metrics',
    'AVG',
    'date',
    'avg_processing_seconds',
    'day',
    '["previous_period", "week_over_week"]',
    'steward.ops@company.com',
    'Operations',
    'transaction_db',
    'operations',
    'operational_metrics',
    6,
    0.98,
    'active',
    false,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d486',
    'compliance_filing_status',
    'Compliance Filing Status',
    'Percentage of required regulatory filings completed on time',
    'compliance',
    'regulatory',
    'percentage',
    'SELECT date, filing_completion_pct FROM compliance_metrics',
    'AVG',
    'date',
    'filing_completion_pct',
    'month',
    '["previous_period", "year_over_year"]',
    'steward.compliance@company.com',
    'Compliance',
    'compliance_db',
    'compliance',
    'compliance_metrics',
    720,
    0.95,
    'active',
    true,
    1,
    'system'
);

-- ===========================================
-- METRIC TAGS FOR CATEGORIZATION
-- ===========================================

INSERT INTO public.pop_metric_tags (metric_id, tag_name, tag_value) VALUES
-- AUM Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d479', 'business_critical', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d479', 'regulatory', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d479', 'kpi', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d479', 'board_reporting', 'true'),

-- NAV Growth Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d480', 'performance', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d480', 'investor_focus', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d480', 'benchmark', 'true'),

-- Net Inflows Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d481', 'growth', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d481', 'investor_sentiment', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d481', 'market_timing', 'true'),

-- Volatility Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d482', 'risk', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d482', 'stress_testing', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d482', 'regulatory', 'true'),

-- Sharpe Ratio Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d483', 'risk_adjusted', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d483', 'performance', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d483', 'peer_comparison', 'true'),

-- Transaction Volume Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d484', 'operational', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d484', 'capacity', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d484', 'scalability', 'true'),

-- Processing Time Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d485', 'efficiency', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d485', 'sla', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d485', 'customer_experience', 'true'),

-- Compliance Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d486', 'regulatory', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d486', 'audit', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d486', 'legal', 'true');

-- ===========================================
-- SAMPLE COMPUTATION RESULTS
-- ===========================================

INSERT INTO public.pop_computations (
    metric_id, period_start, period_end, granularity, period_label,
    current_value, previous_value, delta, percent_change,
    record_count, computation_status
) VALUES
-- AUM Computations (in millions)
('f47ac10b-58cc-4372-a567-0e02b2c3d479', '2024-08-01', '2024-08-31', 'month', '2024-08',
    125000.50, 118000.25, 7000.25, 5.93, 31, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d479', '2024-07-01', '2024-07-31', 'month', '2024-07',
    118000.25, 115500.00, 2500.25, 2.17, 31, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d479', '2024-06-01', '2024-06-30', 'month', '2024-06',
    115500.00, 112000.75, 3499.25, 3.12, 30, 'success'),

-- NAV Growth Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d480', '2024-08-01', '2024-08-31', 'month', '2024-08',
    2.45, 1.89, 0.56, 29.63, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d480', '2024-07-01', '2024-07-31', 'month', '2024-07',
    1.89, 2.12, -0.23, -10.85, 1, 'success'),

-- Net Inflows Computations (in millions)
('f47ac10b-58cc-4372-a567-0e02b2c3d481', '2024-08-01', '2024-08-31', 'month', '2024-08',
    8500.00, 6200.00, 2300.00, 37.10, 31, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d481', '2024-07-01', '2024-07-31', 'month', '2024-07',
    6200.00, 5800.00, 400.00, 6.90, 31, 'success'),

-- Volatility Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d482', '2024-08-01', '2024-08-31', 'month', '2024-08',
    12.45, 15.23, -2.78, -18.25, 31, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d482', '2024-07-01', '2024-07-31', 'month', '2024-07',
    15.23, 18.45, -3.22, -17.45, 31, 'success'),

-- Sharpe Ratio Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d483', '2024-06-01', '2024-08-31', 'quarter', '2024-Q3',
    1.85, 1.62, 0.23, 14.20, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d483', '2024-03-01', '2024-05-31', 'quarter', '2024-Q2',
    1.62, 1.45, 0.17, 11.72, 1, 'success'),

-- Transaction Volume Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d484', '2024-09-09', '2024-09-09', 'day', '2024-09-09',
    15420, 14850, 570, 3.84, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d484', '2024-09-08', '2024-09-08', 'day', '2024-09-08',
    14850, 15200, -350, -2.30, 1, 'success'),

-- Processing Time Computations (in seconds)
('f47ac10b-58cc-4372-a567-0e02b2c3d485', '2024-09-09', '2024-09-09', 'day', '2024-09-09',
    2.34, 2.67, -0.33, -12.36, 15420, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d485', '2024-09-08', '2024-09-08', 'day', '2024-09-08',
    2.67, 2.45, 0.22, 8.98, 14850, 'success'),

-- Compliance Filing Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d486', '2024-08-01', '2024-08-31', 'month', '2024-08',
    98.5, 97.2, 1.3, 1.34, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d486', '2024-07-01', '2024-07-31', 'month', '2024-07',
    97.2, 99.1, -1.9, -1.92, 1, 'success');

-- ===========================================
-- SAMPLE ANOMALIES
-- ===========================================

INSERT INTO public.pop_anomalies (
    metric_id, computation_id,
    anomaly_type, severity, confidence,
    z_score, expected_value, expected_range_min, expected_range_max, actual_value,
    detection_method, detection_params, status
) VALUES
-- High volatility anomaly
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d482',
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d482' AND period_label = '2024-08'),
    'z_score',
    'high',
    0.95,
    3.45,
    14.50,
    12.00,
    17.00,
    18.75,
    'z_score',
    '{"threshold": 2.5, "window_days": 90}',
    'open'
),
-- Negative inflows anomaly
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d481',
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d481' AND period_label = '2024-08'),
    'threshold',
    'medium',
    0.88,
    NULL,
    5000.00,
    3000.00,
    15000.00,
    -2500.00,
    'threshold',
    '{"threshold_value": 0, "direction": "below"}',
    'investigating'
),
-- Processing time degradation
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d485',
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d485' AND period_label = '2024-09-09'),
    'trend_break',
    'medium',
    0.82,
    2.1,
    2.45,
    2.20,
    2.70,
    3.15,
    'trend_analysis',
    '{"slope_threshold": 0.1, "min_periods": 7}',
    'open'
);

-- ===========================================
-- SAMPLE STEWARD REVIEWS
-- ===========================================

INSERT INTO public.pop_steward_reviews (
    metric_id, review_period_start, review_period_end,
    reviewer_user_id, review_type,
    overall_rating, review_notes,
    status, due_date
) VALUES
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d482',
    '2024-08-01',
    '2024-08-31',
    'steward.risk@company.com',
    'anomaly_investigation',
    'needs_attention',
    'High volatility detected in August. Market conditions appear normal, but need to investigate if this is due to specific fund composition changes.',
    'in_progress',
    '2024-09-15'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d481',
    '2024-08-01',
    '2024-08-31',
    'steward.client@company.com',
    'regular',
    'good',
    'Net inflows showing healthy growth. Some outflows in tech sector funds but overall positive momentum.',
    'completed',
    '2024-09-10'
);

-- ===========================================
-- SAMPLE STEWARD COMMENTS
-- ===========================================

INSERT INTO public.pop_steward_comments (
    review_id, anomaly_id, commenter_user_id, comment_type, comment_text
) VALUES
(
    (SELECT id FROM public.pop_steward_reviews WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d482' LIMIT 1),
    (SELECT id FROM public.pop_anomalies WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d482' LIMIT 1),
    'steward.risk@company.com',
    'anomaly_feedback',
    'Volatility spike appears to be driven by increased market uncertainty. Will monitor for next 2 weeks before taking action.'
),
(
    (SELECT id FROM public.pop_steward_reviews WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d481' LIMIT 1),
    NULL,
    'steward.client@company.com',
    'general',
    'Positive inflow trend continues. Marketing campaigns showing good ROI.'
),
(
    NULL,
    NULL,
    'steward.finance@company.com',
    'golden_path',
    'Promoted AUM metric to golden path due to its critical role in regulatory reporting and board communications.'
);

-- ===========================================
-- SAMPLE DASHBOARDS
-- ===========================================

INSERT INTO public.pop_dashboards (
    name, description, owner_user_id,
    config, default_filters,
    is_public, allowed_groups
) VALUES
(
    'Executive Dashboard',
    'High-level metrics for executive team and board reporting',
    'executive@company.com',
    '{
        "layout": "grid",
        "theme": "corporate",
        "auto_refresh": true,
        "refresh_interval": 300
    }',
    '{"domain": "finance", "golden_path": true}',
    false,
    ARRAY['executives', 'board_members']
),
(
    'Risk Management Cockpit',
    'Real-time risk metrics and anomaly monitoring',
    'steward.risk@company.com',
    '{
        "layout": "dashboard",
        "theme": "risk",
        "auto_refresh": true,
        "refresh_interval": 60,
        "alert_thresholds": {
            "volatility": 20.0,
            "sharpe_ratio": 0.5
        }
    }',
    '{"domain": "finance", "category": "risk"}',
    false,
    ARRAY['risk_team', 'compliance']
),
(
    'Operations Dashboard',
    'Operational efficiency and transaction monitoring',
    'steward.ops@company.com',
    '{
        "layout": "operational",
        "theme": "efficiency",
        "auto_refresh": true,
        "refresh_interval": 300
    }',
    '{"domain": "operations"}',
    true,
    ARRAY['operations', 'it', 'management']
);

-- ===========================================
-- SAMPLE DASHBOARD WIDGETS
-- ===========================================

INSERT INTO public.pop_dashboard_widgets (
    dashboard_id, widget_type, title, position, config, metric_ids
) VALUES
-- Executive Dashboard Widgets
(
    (SELECT id FROM public.pop_dashboards WHERE name = 'Executive Dashboard'),
    'kpi_cards',
    'Key Financial KPIs',
    '{"x": 0, "y": 0, "width": 12, "height": 2}',
    '{"show_trend": true, "show_comparison": true}',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d479', 'f47ac10b-58cc-4372-a567-0e02b2c3d480', 'f47ac10b-58cc-4372-a567-0e02b2c3d481']::uuid[]
),
(
    (SELECT id FROM public.pop_dashboards WHERE name = 'Executive Dashboard'),
    'trend_chart',
    'AUM Growth Trend',
    '{"x": 0, "y": 2, "width": 8, "height": 4}',
    '{"chart_type": "line", "period": "12months", "show_forecast": true}',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d479']::uuid[]
),
(
    (SELECT id FROM public.pop_dashboards WHERE name = 'Executive Dashboard'),
    'anomaly_heatmap',
    'Risk and Anomaly Overview',
    '{"x": 8, "y": 2, "width": 4, "height": 4}',
    '{"severity_filter": ["high", "critical"], "group_by": "domain"}',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d482', 'f47ac10b-58cc-4372-a567-0e02b2c3d483']::uuid[]
),

-- Risk Management Widgets
(
    (SELECT id FROM public.pop_dashboards WHERE name = 'Risk Management Cockpit'),
    'metric_table',
    'Risk Metrics Monitor',
    '{"x": 0, "y": 0, "width": 12, "height": 6}',
    '{"show_anomalies": true, "alert_on_threshold": true, "sort_by": "severity"}',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d482', 'f47ac10b-58cc-4372-a567-0e02b2c3d483']::uuid[]
),
(
    (SELECT id FROM public.pop_dashboards WHERE name = 'Risk Management Cockpit'),
    'trend_chart',
    'Volatility Trends',
    '{"x": 0, "y": 6, "width": 6, "height": 4}',
    '{"chart_type": "area", "period": "6months", "show_bands": true}',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d482']::uuid[]
),

-- Operations Dashboard Widgets
(
    (SELECT id FROM public.pop_dashboards WHERE name = 'Operations Dashboard'),
    'kpi_cards',
    'Operational KPIs',
    '{"x": 0, "y": 0, "width": 12, "height": 2}',
    '{"show_sla": true, "show_targets": true}',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d484', 'f47ac10b-58cc-4372-a567-0e02b2c3d485']::uuid[]
),
(
    (SELECT id FROM public.pop_dashboards WHERE name = 'Operations Dashboard'),
    'trend_chart',
    'Transaction Processing Efficiency',
    '{"x": 0, "y": 2, "width": 8, "height": 4}',
    '{"chart_type": "combo", "show_volume": true, "show_time": true}',
    ARRAY['f47ac10b-58cc-4372-a567-0e02b2c3d484', 'f47ac10b-58cc-4372-a567-0e02b2c3d485']::uuid[]
);

-- ===========================================
-- SUCCESS MESSAGE
-- ===========================================

DO $$
BEGIN
    RAISE NOTICE 'PoP Metrics seed data for Mutual Fund Company has been successfully inserted!';
    RAISE NOTICE 'Created % metrics, % computations, % anomalies, % reviews, and % dashboards',
        (SELECT COUNT(*) FROM public.pop_metrics),
        (SELECT COUNT(*) FROM public.pop_computations),
        (SELECT COUNT(*) FROM public.pop_anomalies),
        (SELECT COUNT(*) FROM public.pop_steward_reviews),
        (SELECT COUNT(*) FROM public.pop_dashboards);
END $$;
