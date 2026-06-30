-- Advanced PoP Metrics Enhancements for Mutual Fund Company
-- Created: 2025-09-10
-- Description: Additional sophisticated metrics and capabilities

-- ===========================================
-- ADVANCED FINANCIAL METRICS
-- ===========================================

-- Additional sophisticated metrics for mutual fund analysis
INSERT INTO public.pop_metrics (
    id, name, display_name, description, domain, category, metric_type,
    base_query, aggregation_function, date_column, value_column,
    granularity, comparison_periods,
    owner_user_id, steward_group, data_source, schema_name, table_name,
    sla_freshness_hours, sla_completeness_threshold,
    status, golden_path, version, created_by
) VALUES
-- Advanced Performance Metrics
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d489',
    'fund_alpha',
    'Fund Alpha',
    'Risk-adjusted excess return over benchmark',
    'finance',
    'performance',
    'ratio',
    'SELECT date, alpha_value FROM advanced_performance_metrics',
    'AVG',
    'date',
    'alpha_value',
    'month',
    '["previous_period", "year_over_year"]',
    'steward.investment@company.com',
    'Investment Operations',
    'fund_database',
    'analytics',
    'advanced_performance_metrics',
    48,
    0.95,
    'active',
    true,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d490',
    'fund_beta',
    'Fund Beta',
    'Measure of systematic risk relative to market',
    'finance',
    'risk',
    'ratio',
    'SELECT date, beta_value FROM risk_factors',
    'AVG',
    'date',
    'beta_value',
    'month',
    '["previous_period", "year_over_year"]',
    'steward.risk@company.com',
    'Risk Management',
    'fund_database',
    'analytics',
    'risk_factors',
    72,
    0.90,
    'active',
    true,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d491',
    'maximum_drawdown',
    'Maximum Drawdown',
    'Largest peak-to-trough decline in portfolio value',
    'finance',
    'risk',
    'ratio',
    'SELECT date, max_drawdown_pct FROM drawdown_analysis',
    'MAX',
    'date',
    'max_drawdown_pct',
    'month',
    '["previous_period", "year_over_year"]',
    'steward.risk@company.com',
    'Risk Management',
    'fund_database',
    'analytics',
    'drawdown_analysis',
    72,
    0.85,
    'active',
    true,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d492',
    'tracking_error',
    'Tracking Error',
    'Standard deviation of difference between fund and benchmark returns',
    'finance',
    'performance',
    'ratio',
    'SELECT date, tracking_error FROM benchmark_comparison',
    'AVG',
    'date',
    'tracking_error',
    'month',
    '["previous_period", "year_over_year"]',
    'steward.investment@company.com',
    'Investment Operations',
    'fund_database',
    'analytics',
    'benchmark_comparison',
    48,
    0.90,
    'active',
    false,
    1,
    'system'
),
-- Operational Excellence Metrics
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d493',
    'client_satisfaction_score',
    'Client Satisfaction Score',
    'Average client satisfaction rating from surveys',
    'operations',
    'customer_experience',
    'avg',
    'SELECT date, satisfaction_score FROM client_feedback',
    'AVG',
    'date',
    'satisfaction_score',
    'month',
    '["previous_period", "year_over_year"]',
    'steward.client@company.com',
    'Client Operations',
    'crm_database',
    'analytics',
    'client_feedback',
    168,
    0.80,
    'active',
    true,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d494',
    'expense_ratio',
    'Fund Expense Ratio',
    'Total annual fund operating expenses as percentage of assets',
    'finance',
    'efficiency',
    'ratio',
    'SELECT date, expense_ratio_pct FROM fund_expenses',
    'AVG',
    'date',
    'expense_ratio_pct',
    'quarter',
    '["previous_period", "year_over_year"]',
    'steward.finance@company.com',
    'Finance Operations',
    'fund_database',
    'analytics',
    'fund_expenses',
    720,
    0.95,
    'active',
    true,
    1,
    'system'
),
-- Regulatory and Compliance Metrics
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d495',
    'regulatory_fines',
    'Regulatory Fines',
    'Total regulatory fines and penalties incurred',
    'compliance',
    'regulatory',
    'sum',
    'SELECT date, fine_amount FROM regulatory_actions',
    'SUM',
    'date',
    'fine_amount',
    'month',
    '["previous_period", "year_over_year"]',
    'steward.compliance@company.com',
    'Compliance',
    'compliance_db',
    'compliance',
    'regulatory_actions',
    720,
    0.99,
    'active',
    true,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d496',
    'audit_findings_count',
    'Audit Findings Count',
    'Number of findings from internal and external audits',
    'compliance',
    'audit',
    'count',
    'SELECT date, findings_count FROM audit_results',
    'SUM',
    'date',
    'findings_count',
    'quarter',
    '["previous_period", "year_over_year"]',
    'steward.compliance@company.com',
    'Compliance',
    'compliance_db',
    'compliance',
    'audit_results',
    2160,
    0.95,
    'active',
    true,
    1,
    'system'
),
-- Market and Competitive Metrics
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d497',
    'market_share',
    'Market Share',
    'Percentage of total mutual fund market assets managed',
    'finance',
    'market_position',
    'percentage',
    'SELECT date, market_share_pct FROM market_intelligence',
    'AVG',
    'date',
    'market_share_pct',
    'quarter',
    '["previous_period", "year_over_year"]',
    'steward.strategy@company.com',
    'Strategy',
    'market_data',
    'intelligence',
    'market_intelligence',
    168,
    0.85,
    'active',
    true,
    1,
    'system'
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d498',
    'peer_performance_rank',
    'Peer Performance Rank',
    'Percentile ranking against peer group performance',
    'finance',
    'performance',
    'ratio',
    'SELECT date, performance_percentile FROM peer_analysis',
    'AVG',
    'date',
    'performance_percentile',
    'quarter',
    '["previous_period", "year_over_year"]',
    'steward.investment@company.com',
    'Investment Operations',
    'peer_database',
    'analytics',
    'peer_analysis',
    168,
    0.80,
    'active',
    false,
    1,
    'system'
);

-- ===========================================
-- ENHANCED METRIC TAGS
-- ===========================================

INSERT INTO public.pop_metric_tags (metric_id, tag_name, tag_value) VALUES
-- Alpha Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d489', 'risk_adjusted', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d489', 'benchmark_relative', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d489', 'investment_performance', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d489', 'attribution', 'true'),

-- Beta Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d490', 'systematic_risk', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d490', 'market_risk', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d490', 'volatility', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d490', 'correlation', 'true'),

-- Maximum Drawdown Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d491', 'risk_measure', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d491', 'worst_case', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d491', 'stress_testing', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d491', 'recovery_analysis', 'true'),

-- Tracking Error Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d492', 'benchmark_tracking', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d492', 'active_risk', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d492', 'deviation', 'true'),

-- Client Satisfaction Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d493', 'customer_experience', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d493', 'nps', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d493', 'retention', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d493', 'loyalty', 'true'),

-- Expense Ratio Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d494', 'cost_efficiency', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d494', 'fee_analysis', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d494', 'competitiveness', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d494', 'regulatory', 'true'),

-- Regulatory Fines Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d495', 'compliance_cost', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d495', 'regulatory_risk', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d495', 'reputation', 'true'),

-- Audit Findings Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d496', 'internal_controls', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d496', 'governance', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d496', 'remediation', 'true'),

-- Market Share Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d497', 'competition', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d497', 'growth', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d497', 'market_position', 'true'),

-- Peer Performance Tags
('f47ac10b-58cc-4372-a567-0e02b2c3d498', 'benchmarking', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d498', 'relative_performance', 'true'),
('f47ac10b-58cc-4372-a567-0e02b2c3d498', 'quartile_analysis', 'true');

-- ===========================================
-- SAMPLE COMPUTATIONS FOR NEW METRICS
-- ===========================================

INSERT INTO public.pop_computations (
    metric_id, period_start, period_end, granularity, period_label,
    current_value, previous_value, delta, percent_change,
    record_count, computation_status
) VALUES
-- Alpha Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d489', '2024-08-01', '2024-08-31', 'month', '2024-08',
    1.25, 0.85, 0.40, 47.06, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d489', '2024-07-01', '2024-07-31', 'month', '2024-07',
    0.85, 1.15, -0.30, -26.09, 1, 'success'),

-- Beta Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d490', '2024-08-01', '2024-08-31', 'month', '2024-08',
    0.95, 1.05, -0.10, -9.52, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d490', '2024-07-01', '2024-07-31', 'month', '2024-07',
    1.05, 0.98, 0.07, 7.14, 1, 'success'),

-- Maximum Drawdown Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d491', '2024-08-01', '2024-08-31', 'month', '2024-08',
    -8.45, -12.20, 3.75, -30.74, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d491', '2024-07-01', '2024-07-31', 'month', '2024-07',
    -12.20, -9.85, -2.35, 23.86, 1, 'success'),

-- Tracking Error Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d492', '2024-08-01', '2024-08-31', 'month', '2024-08',
    2.15, 2.45, -0.30, -12.24, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d492', '2024-07-01', '2024-07-31', 'month', '2024-07',
    2.45, 2.10, 0.35, 16.67, 1, 'success'),

-- Client Satisfaction Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d493', '2024-08-01', '2024-08-31', 'month', '2024-08',
    8.7, 8.4, 0.3, 3.57, 1250, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d493', '2024-07-01', '2024-07-31', 'month', '2024-07',
    8.4, 8.6, -0.2, -2.33, 1180, 'success'),

-- Expense Ratio Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d494', '2024-06-01', '2024-08-31', 'quarter', '2024-Q3',
    0.85, 0.88, -0.03, -3.41, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d494', '2024-03-01', '2024-05-31', 'quarter', '2024-Q2',
    0.88, 0.90, -0.02, -2.22, 1, 'success'),

-- Regulatory Fines Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d495', '2024-08-01', '2024-08-31', 'month', '2024-08',
    0.00, 25000.00, -25000.00, -100.00, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d495', '2024-07-01', '2024-07-31', 'month', '2024-07',
    25000.00, 0.00, 25000.00, 100.00, 1, 'success'),

-- Audit Findings Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d496', '2024-06-01', '2024-08-31', 'quarter', '2024-Q3',
    3, 5, -2, -40.00, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d496', '2024-03-01', '2024-05-31', 'quarter', '2024-Q2',
    5, 7, -2, -28.57, 1, 'success'),

-- Market Share Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d497', '2024-06-01', '2024-08-31', 'quarter', '2024-Q3',
    4.25, 4.15, 0.10, 2.41, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d497', '2024-03-01', '2024-05-31', 'quarter', '2024-Q2',
    4.15, 4.05, 0.10, 2.47, 1, 'success'),

-- Peer Performance Rank Computations
('f47ac10b-58cc-4372-a567-0e02b2c3d498', '2024-06-01', '2024-08-31', 'quarter', '2024-Q3',
    65.5, 58.2, 7.3, 12.54, 1, 'success'),
('f47ac10b-58cc-4372-a567-0e02b2c3d498', '2024-03-01', '2024-05-31', 'quarter', '2024-Q2',
    58.2, 62.1, -3.9, -6.28, 1, 'success');

-- ===========================================
-- ADVANCED ANOMALY PATTERNS
-- ===========================================

INSERT INTO public.pop_anomalies (
    metric_id, computation_id,
    anomaly_type, severity, confidence,
    z_score, expected_value, expected_range_min, expected_range_max, actual_value,
    detection_method, detection_params, status
) VALUES
-- Alpha anomaly (significant outperformance)
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d489',
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d489' AND period_label = '2024-08'),
    'z_score',
    'high',
    0.92,
    2.85,
    0.50,
    -0.20,
    1.20,
    1.25,
    'z_score',
    '{"threshold": 2.0, "window_days": 180}',
    'open'
),
-- Beta anomaly (unusual market sensitivity)
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d490',
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d490' AND period_label = '2024-08'),
    'iqr',
    'medium',
    0.78,
    NULL,
    1.00,
    0.85,
    1.15,
    0.95,
    'iqr',
    '{"multiplier": 1.5}',
    'investigating'
),
-- Maximum drawdown anomaly (severe loss)
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d491',
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d491' AND period_label = '2024-08'),
    'threshold',
    'critical',
    0.95,
    NULL,
    -5.00,
    -15.00,
    -2.00,
    -8.45,
    'threshold',
    '{"threshold_value": -10.0, "direction": "below"}',
    'open'
),
-- Client satisfaction anomaly (significant drop)
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d493',
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d493' AND period_label = '2024-08'),
    'trend_break',
    'medium',
    0.85,
    -1.95,
    8.6,
    8.2,
    9.0,
    8.7,
    'trend_analysis',
    '{"slope_threshold": -0.5, "min_periods": 3}',
    'open'
),
-- Regulatory fine anomaly (unexpected fine)
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d495',
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d495' AND period_label = '2024-07'),
    'threshold',
    'high',
    0.98,
    NULL,
    0.00,
    0.00,
    1000.00,
    25000.00,
    'threshold',
    '{"threshold_value": 0, "direction": "above"}',
    'open'
);

-- ===========================================
-- ENHANCED STEWARD REVIEWS
-- ===========================================

INSERT INTO public.pop_steward_reviews (
    metric_id, review_period_start, review_period_end,
    reviewer_user_id, review_type,
    overall_rating, review_notes,
    status, due_date
) VALUES
-- Alpha performance review
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d489',
    '2024-08-01',
    '2024-08-31',
    'steward.investment@company.com',
    'anomaly_investigation',
    'excellent',
    'Strong alpha generation in August. Portfolio managers successfully identified undervalued securities. Need to analyze if this is sustainable or luck-based.',
    'completed',
    '2024-09-15'
),
-- Risk metrics review
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d491',
    '2024-08-01',
    '2024-08-31',
    'steward.risk@company.com',
    'anomaly_investigation',
    'needs_attention',
    'Maximum drawdown improved but still concerning. Risk management protocols need review. Consider implementing additional downside protection measures.',
    'in_progress',
    '2024-09-20'
),
-- Client experience review
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d493',
    '2024-08-01',
    '2024-08-31',
    'steward.client@company.com',
    'regular',
    'good',
    'Client satisfaction showing positive trend. Recent improvements in digital platform have been well-received. Continue monitoring for sustained improvement.',
    'completed',
    '2024-09-10'
),
-- Compliance review
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d495',
    '2024-07-01',
    '2024-07-31',
    'steward.compliance@company.com',
    'anomaly_investigation',
    'critical',
    'Unexpected regulatory fine incurred. Immediate investigation required. Need to review compliance procedures and identify root cause.',
    'in_progress',
    '2024-09-05'
);

-- ===========================================
-- ENHANCED STEWARD COMMENTS
-- ===========================================

INSERT INTO public.pop_steward_comments (
    review_id, anomaly_id, commenter_user_id, comment_type, comment_text
) VALUES
-- Investment team comments on alpha
(
    (SELECT id FROM public.pop_steward_reviews WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d489' LIMIT 1),
    (SELECT id FROM public.pop_anomalies WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d489' LIMIT 1),
    'portfolio.manager@company.com',
    'anomaly_feedback',
    'Alpha generation driven by successful short positions in overvalued tech stocks. Risk management protocols were followed throughout.'
),
(
    (SELECT id FROM public.pop_steward_reviews WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d489' LIMIT 1),
    NULL,
    'chief.investment.officer@company.com',
    'general',
    'Excellent work by the investment team. This level of alpha generation is exceptional and should be highlighted in client communications.'
),

-- Risk team comments on drawdown
(
    (SELECT id FROM public.pop_steward_reviews WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d491' LIMIT 1),
    (SELECT id FROM public.pop_anomalies WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d491' LIMIT 1),
    'chief.risk.officer@company.com',
    'anomaly_feedback',
    'Drawdown was within acceptable limits but triggers review of stop-loss procedures. Will implement additional monitoring for high-risk positions.'
),

-- Client operations comments
(
    (SELECT id FROM public.pop_steward_reviews WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d493' LIMIT 1),
    NULL,
    'client.services@company.com',
    'general',
    'Satisfaction improvement attributed to faster response times and improved mobile app functionality. Continue investing in client experience.'
),

-- Compliance team comments on fine
(
    (SELECT id FROM public.pop_steward_reviews WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d495' LIMIT 1),
    (SELECT id FROM public.pop_anomalies WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d495' LIMIT 1),
    'compliance.officer@company.com',
    'anomaly_feedback',
    'Fine related to late filing of Form 13F. Root cause analysis shows system processing delay. Implementing additional controls and monitoring.'
);

-- ===========================================
-- PREDICTIVE ANALYTICS TABLES
-- =========================================--

-- CREATE TABLE IF NOT EXISTS for predictive models
CREATE TABLE IF NOT EXISTS public.pop_predictive_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,
    model_name TEXT NOT NULL,
    model_type TEXT NOT NULL, -- 'linear_regression', 'arima', 'prophet', 'neural_network'
    model_params JSONB NOT NULL,
    training_period_start DATE NOT NULL,
    training_period_end DATE NOT NULL,
    accuracy_score DECIMAL(5,4),
    last_trained TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- CREATE TABLE IF NOT EXISTS for predictions
CREATE TABLE IF NOT EXISTS public.pop_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,
    model_id UUID REFERENCES public.pop_predictive_models(id) ON DELETE CASCADE,
    prediction_date DATE NOT NULL,
    predicted_value DECIMAL(20,6),
    confidence_interval_lower DECIMAL(20,6),
    confidence_interval_upper DECIMAL(20,6),
    prediction_horizon_days INTEGER NOT NULL,
    actual_value DECIMAL(20,6),
    prediction_error DECIMAL(20,6),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample predictive models
INSERT INTO public.pop_predictive_models (
    metric_id, model_name, model_type, model_params,
    training_period_start, training_period_end, accuracy_score, is_active
) VALUES
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'AUM Growth Predictor',
    'linear_regression',
    '{
        "features": ["market_performance", "inflows", "economic_indicators"],
        "coefficients": {"market_performance": 0.65, "inflows": 0.25, "economic_indicators": 0.10},
        "intercept": 1000000.0
    }',
    '2023-01-01',
    '2024-06-30',
    0.87,
    true
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d482',
    'Volatility Forecaster',
    'arima',
    '{
        "order": [2, 1, 1],
        "seasonal_order": [1, 1, 1, 12],
        "trend": "c"
    }',
    '2023-01-01',
    '2024-06-30',
    0.78,
    true
);

-- Sample predictions
INSERT INTO public.pop_predictions (
    metric_id, model_id, prediction_date, predicted_value,
    confidence_interval_lower, confidence_interval_upper, prediction_horizon_days
) VALUES
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    (SELECT id FROM public.pop_predictive_models WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d479' LIMIT 1),
    '2024-09-30',
    132000.00,
    128000.00,
    136000.00,
    30
),
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d482',
    (SELECT id FROM public.pop_predictive_models WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d482' LIMIT 1),
    '2024-09-30',
    14.25,
    12.00,
    16.50,
    30
);

-- ===========================================
-- STRESS TESTING FRAMEWORK
-- =========================================--

-- CREATE TABLE IF NOT EXISTS for stress test scenarios
CREATE TABLE IF NOT EXISTS public.pop_stress_scenarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_name TEXT NOT NULL,
    scenario_description TEXT,
    scenario_type TEXT NOT NULL, -- 'market_crash', 'interest_rate_hike', 'liquidity_crisis', 'custom'
    parameters JSONB NOT NULL, -- Scenario-specific parameters
    created_by TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- CREATE TABLE IF NOT EXISTS for stress test results
CREATE TABLE IF NOT EXISTS public.pop_stress_test_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id UUID REFERENCES public.pop_stress_scenarios(id) ON DELETE CASCADE,
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,
    baseline_value DECIMAL(20,6),
    stressed_value DECIMAL(20,6),
    impact_percentage DECIMAL(10,4),
    recovery_time_days INTEGER,
    test_date TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample stress test scenarios
INSERT INTO public.pop_stress_scenarios (
    scenario_name, scenario_description, scenario_type, parameters, created_by
) VALUES
(
    '2020 Market Crash Scenario',
    'Simulates 30% market decline over 6 months',
    'market_crash',
    '{
        "market_decline_pct": 30.0,
        "duration_days": 180,
        "recovery_period_days": 365,
        "volatility_multiplier": 2.5
    }',
    'steward.risk@company.com'
),
(
    'Interest Rate Shock',
    'Simulates 200bps immediate rate increase',
    'interest_rate_hike',
    '{
        "rate_increase_bps": 200,
        "duration_days": 90,
        "bond_impact_multiplier": 1.8,
        "equity_impact_multiplier": 0.3
    }',
    'steward.risk@company.com'
);

-- ===========================================
-- PEER BENCHMARKING TABLES
-- =========================================--

-- CREATE TABLE IF NOT EXISTS for peer groups
CREATE TABLE IF NOT EXISTS public.pop_peer_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_name TEXT NOT NULL,
    group_description TEXT,
    criteria JSONB NOT NULL, -- Criteria for inclusion in peer group
    created_by TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- CREATE TABLE IF NOT EXISTS for peer comparisons
CREATE TABLE IF NOT EXISTS public.pop_peer_comparisons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,
    peer_group_id UUID REFERENCES public.pop_peer_groups(id) ON DELETE CASCADE,
    comparison_date DATE NOT NULL,
    our_value DECIMAL(20,6),
    peer_median DECIMAL(20,6),
    peer_quartile_25 DECIMAL(20,6),
    peer_quartile_75 DECIMAL(20,6),
    peer_count INTEGER,
    percentile_rank DECIMAL(5,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample peer groups
INSERT INTO public.pop_peer_groups (
    group_name, group_description, criteria, created_by
) VALUES
(
    'Large Cap Equity Funds',
    'Peer group of large capitalization equity mutual funds',
    '{
        "asset_size_min": 1000000000,
        "asset_size_max": 50000000000,
        "strategy": "large_cap_equity",
        "geography": "us_domestic"
    }',
    'steward.investment@company.com'
),
(
    'Balanced Funds',
    'Peer group of balanced allocation mutual funds',
    '{
        "asset_size_min": 500000000,
        "strategy": "balanced",
        "equity_allocation_min": 40,
        "equity_allocation_max": 70
    }',
    'steward.investment@company.com'
);

-- ===========================================
-- SUCCESS MESSAGE
-- =========================================--

DO $$
BEGIN
    RAISE NOTICE 'Advanced PoP Metrics enhancements have been successfully added!';
    RAISE NOTICE 'New metrics count: %', (SELECT COUNT(*) FROM public.pop_metrics);
    RAISE NOTICE 'New computations count: %', (SELECT COUNT(*) FROM public.pop_computations);
    RAISE NOTICE 'New anomalies count: %', (SELECT COUNT(*) FROM public.pop_anomalies);
    RAISE NOTICE 'Predictive models created: %', (SELECT COUNT(*) FROM public.pop_predictive_models);
    RAISE NOTICE 'Stress test scenarios added: %', (SELECT COUNT(*) FROM public.pop_stress_scenarios);
    RAISE NOTICE 'Peer groups defined: %', (SELECT COUNT(*) FROM public.pop_peer_groups);
END $$;
