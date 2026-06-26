-- Machine Learning and Advanced Analytics Enhancements
-- Created: 2025-09-10
-- Description: ML-based anomaly detection, external data integration, and compliance automation

-- ===========================================
-- MACHINE LEARNING ANOMALY DETECTION
-- =========================================--

-- Create table for ML models
CREATE TABLE IF NOT EXISTS public.pop_ml_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,
    model_name TEXT NOT NULL,
    model_type TEXT NOT NULL, -- 'isolation_forest', 'autoencoder', 'lstm', 'prophet', 'xgboost'
    model_version TEXT NOT NULL,
    model_params JSONB NOT NULL,
    training_data_start DATE NOT NULL,
    training_data_end DATE NOT NULL,
    model_accuracy DECIMAL(5,4),
    model_precision DECIMAL(5,4),
    model_recall DECIMAL(5,4),
    model_f1_score DECIMAL(5,4),
    feature_importance JSONB,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_trained TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create table for ML predictions and anomalies
CREATE TABLE IF NOT EXISTS public.pop_ml_anomalies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,
    model_id UUID REFERENCES public.pop_ml_models(id) ON DELETE CASCADE,
    computation_id UUID REFERENCES public.pop_computations(id) ON DELETE CASCADE,
    anomaly_score DECIMAL(10,6) NOT NULL,
    anomaly_probability DECIMAL(5,4) NOT NULL,
    predicted_value DECIMAL(20,6),
    actual_value DECIMAL(20,6),
    confidence_interval_lower DECIMAL(20,6),
    confidence_interval_upper DECIMAL(20,6),
    detection_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    explanation TEXT,
    severity TEXT NOT NULL, -- 'low', 'medium', 'high', 'critical'
    status TEXT NOT NULL DEFAULT 'detected', -- 'detected', 'investigating', 'confirmed', 'false_positive'
    investigated_by TEXT,
    investigation_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample ML models
INSERT INTO public.pop_ml_models (
    metric_id, model_name, model_type, model_version, model_params,
    training_data_start, training_data_end, model_accuracy, model_precision,
    model_recall, model_f1_score, feature_importance, created_by
) VALUES
-- AUM prediction model
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'AUM Trend Predictor',
    'xgboost',
    '1.0.0',
    '{
        "n_estimators": 100,
        "max_depth": 6,
        "learning_rate": 0.1,
        "features": ["previous_aum", "market_performance", "inflows", "economic_indicators", "seasonal_factors"]
    }',
    '2022-01-01',
    '2024-06-30',
    0.89,
    0.91,
    0.87,
    0.89,
    '{
        "previous_aum": 0.35,
        "market_performance": 0.28,
        "inflows": 0.22,
        "economic_indicators": 0.10,
        "seasonal_factors": 0.05
    }',
    'ml.engineer@company.com'
),
-- Volatility anomaly detector
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d482',
    'Volatility Anomaly Detector',
    'isolation_forest',
    '1.0.0',
    '{
        "n_estimators": 100,
        "contamination": 0.1,
        "features": ["volatility", "market_volatility", "vix_index", "trading_volume", "economic_uncertainty"]
    }',
    '2022-01-01',
    '2024-06-30',
    0.94,
    0.96,
    0.92,
    0.94,
    '{
        "volatility": 0.40,
        "market_volatility": 0.30,
        "vix_index": 0.20,
        "trading_volume": 0.07,
        "economic_uncertainty": 0.03
    }',
    'ml.engineer@company.com'
),
-- Transaction volume forecaster
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d484',
    'Transaction Volume LSTM',
    'lstm',
    '1.0.0',
    '{
        "layers": [64, 32],
        "dropout": 0.2,
        "sequence_length": 30,
        "features": ["volume", "day_of_week", "month", "holiday_flag", "market_conditions"]
    }',
    '2023-01-01',
    '2024-06-30',
    0.87,
    0.89,
    0.85,
    0.87,
    '{
        "volume": 0.45,
        "day_of_week": 0.25,
        "month": 0.15,
        "holiday_flag": 0.10,
        "market_conditions": 0.05
    }',
    'ml.engineer@company.com'
);

-- Sample ML-detected anomalies
INSERT INTO public.pop_ml_anomalies (
    metric_id, model_id, computation_id, anomaly_score, anomaly_probability,
    predicted_value, actual_value, confidence_interval_lower, confidence_interval_upper,
    explanation, severity, status, investigation_notes
) VALUES
-- AUM anomaly
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    (SELECT id FROM public.pop_ml_models WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d479'),
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d479' AND period_label = '2024-08'),
    2.45,
    0.92,
    122500.00,
    125000.50,
    120000.00,
    125000.00,
    'AUM growth significantly higher than predicted based on market conditions and historical patterns. Possible explanation: Unexpected large inflows from institutional clients.',
    'high',
    'investigating',
    'Investigating large institutional inflows. Preliminary analysis shows $2.5M from pension fund commitments.'
),
-- Volatility anomaly
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d482',
    (SELECT id FROM public.pop_ml_models WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d482'),
    (SELECT id FROM public.pop_computations WHERE metric_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d482' AND period_label = '2024-08'),
    3.12,
    0.98,
    14.50,
    18.75,
    12.00,
    17.00,
    'Volatility spike detected using Isolation Forest algorithm. Anomaly score indicates this is a rare event (top 2% of historical observations).',
    'critical',
    'confirmed',
    'Confirmed anomaly due to geopolitical events and market uncertainty. Risk management protocols activated.'
);

-- ===========================================
-- EXTERNAL DATA INTEGRATION
-- =========================================--

-- Create table for external data sources
CREATE TABLE IF NOT EXISTS public.pop_external_data_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_name TEXT NOT NULL,
    source_type TEXT NOT NULL, -- 'api', 'database', 'file', 'stream'
    connection_params JSONB NOT NULL,
    data_frequency TEXT NOT NULL, -- 'real-time', 'daily', 'weekly', 'monthly'
    last_sync TIMESTAMP WITH TIME ZONE,
    sync_status TEXT DEFAULT 'pending', -- 'pending', 'success', 'failed'
    error_message TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create table for external data mappings
CREATE TABLE IF NOT EXISTS public.pop_external_data_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID REFERENCES public.pop_external_data_sources(id) ON DELETE CASCADE,
    metric_id UUID REFERENCES public.pop_metrics(id) ON DELETE CASCADE,
    external_field_name TEXT NOT NULL,
    transformation_rule TEXT,
    data_quality_score DECIMAL(3,2),
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample external data sources
INSERT INTO public.pop_external_data_sources (
    source_name, source_type, connection_params, data_frequency, last_sync, sync_status, created_by
) VALUES
-- Market data provider
(
    'Bloomberg Terminal',
    'api',
    '{
        "api_key": "encrypted_key",
        "endpoint": "https://api.bloomberg.com/v1",
        "authentication": "oauth2"
    }',
    'real-time',
    '2024-09-09 16:00:00+00',
    'success',
    'data.engineer@company.com'
),
-- Economic indicators
(
    'Federal Reserve Economic Data (FRED)',
    'api',
    '{
        "api_key": "fred_api_key",
        "endpoint": "https://api.stlouisfed.org/fred",
        "series": ["GDP", "UNRATE", "FEDFUNDS"]
    }',
    'monthly',
    '2024-09-01 08:00:00+00',
    'success',
    'data.engineer@company.com'
),
-- Peer benchmarking data
(
    'Morningstar Direct',
    'database',
    '{
        "connection_string": "encrypted_connection",
        "database": "morningstar_peer_data",
        "tables": ["fund_performance", "peer_rankings"]
    }',
    'daily',
    '2024-09-09 06:00:00+00',
    'success',
    'data.engineer@company.com'
),
-- Regulatory filings
(
    'SEC EDGAR Database',
    'api',
    '{
        "api_key": "sec_api_key",
        "endpoint": "https://www.sec.gov/edgar/searchedgar",
        "forms": ["13F", "N-PORT", "N-CEN"]
    }',
    'quarterly',
    '2024-07-15 10:00:00+00',
    'success',
    'compliance@company.com'
);

-- Sample external data mappings
INSERT INTO public.pop_external_data_mappings (
    source_id, metric_id, external_field_name, transformation_rule, data_quality_score
) VALUES
-- Bloomberg mappings
(
    (SELECT id FROM public.pop_external_data_sources WHERE source_name = 'Bloomberg Terminal'),
    'f47ac10b-58cc-4372-a567-0e02b2c3d482',
    'SPX_VOLATILITY_30D',
    'value * 100', -- Convert to percentage
    0.98
),
(
    (SELECT id FROM public.pop_external_data_sources WHERE source_name = 'Bloomberg Terminal'),
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'FUND_AUM_TOTAL',
    'value / 1000000', -- Convert to millions
    0.99
),

-- FRED mappings
(
    (SELECT id FROM public.pop_external_data_sources WHERE source_name = 'Federal Reserve Economic Data (FRED)'),
    'f47ac10b-58cc-4372-a567-0e02b2c3d489',
    'GDP_GROWTH_RATE',
    'value', -- Direct mapping
    0.95
),

-- Morningstar mappings
(
    (SELECT id FROM public.pop_external_data_sources WHERE source_name = 'Morningstar Direct'),
    'f47ac10b-58cc-4372-a567-0e02b2c3d498',
    'PEER_PERCENTILE_RANK',
    'value', -- Direct mapping
    0.92
);

-- ===========================================
-- COMPLIANCE REPORTING AUTOMATION
-- =========================================--

-- Create table for compliance reports
CREATE TABLE IF NOT EXISTS public.pop_compliance_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_name TEXT NOT NULL,
    report_type TEXT NOT NULL, -- 'regulatory', 'internal', 'client', 'audit'
    regulatory_body TEXT, -- 'SEC', 'FINRA', 'Internal', etc.
    report_period_start DATE NOT NULL,
    report_period_end DATE NOT NULL,
    generation_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    generated_by TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft', -- 'draft', 'review', 'approved', 'submitted', 'accepted'
    submission_deadline DATE,
    actual_submission_date DATE,
    review_notes TEXT,
    approval_date TIMESTAMP WITH TIME ZONE,
    approved_by TEXT,
    file_path TEXT,
    checksum TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create table for compliance report sections
CREATE TABLE IF NOT EXISTS public.pop_compliance_report_sections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID REFERENCES public.pop_compliance_reports(id) ON DELETE CASCADE,
    section_name TEXT NOT NULL,
    section_order INTEGER NOT NULL,
    metric_ids UUID[] NOT NULL,
    template TEXT,
    generated_content TEXT,
    data_sources JSONB,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create table for compliance requirements
CREATE TABLE IF NOT EXISTS public.pop_compliance_requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requirement_name TEXT NOT NULL,
    regulatory_body TEXT NOT NULL,
    requirement_type TEXT NOT NULL, -- 'reporting', 'disclosure', 'testing', 'monitoring'
    frequency TEXT NOT NULL, -- 'daily', 'weekly', 'monthly', 'quarterly', 'annual'
    deadline_day INTEGER, -- Day of month/quarter
    responsible_party TEXT NOT NULL,
    metric_ids UUID[],
    automation_level TEXT NOT NULL DEFAULT 'manual', -- 'manual', 'semi-automated', 'fully-automated'
    last_compliance_check TIMESTAMP WITH TIME ZONE,
    compliance_status TEXT DEFAULT 'pending', -- 'compliant', 'non-compliant', 'pending'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample compliance reports
INSERT INTO public.pop_compliance_reports (
    report_name, report_type, regulatory_body, report_period_start, report_period_end,
    generated_by, status, submission_deadline
) VALUES
(
    'Form N-PORT Quarterly Report',
    'regulatory',
    'SEC',
    '2024-07-01',
    '2024-09-30',
    'compliance@company.com',
    'approved',
    '2024-11-30'
),
(
    'Risk Management Quarterly Report',
    'internal',
    'Internal',
    '2024-07-01',
    '2024-09-30',
    'steward.risk@company.com',
    'review',
    '2024-10-15'
),
(
    'Client Performance Report Q3 2024',
    'client',
    'Client Reporting',
    '2024-07-01',
    '2024-09-30',
    'client.services@company.com',
    'draft',
    '2024-10-31'
);

-- Sample compliance requirements
INSERT INTO public.pop_compliance_requirements (
    requirement_name, regulatory_body, requirement_type, frequency, deadline_day,
    responsible_party, metric_ids, automation_level, compliance_status
) VALUES
(
    'Form 13F Institutional Holdings Report',
    'SEC',
    'reporting',
    'quarterly',
    45, -- 45 days after quarter end
    'compliance@company.com',
    ARRAY[
        'f47ac10b-58cc-4372-a567-0e02b2c3d479',
        'f47ac10b-58cc-4372-a567-0e02b2c3d486'
    ]::uuid[]::UUID[],
    'semi-automated',
    'compliant'
),
(
    'Risk Metrics Monitoring',
    'Internal',
    'monitoring',
    'daily',
    NULL,
    'steward.risk@company.com',
    ARRAY[
        'f47ac10b-58cc-4372-a567-0e02b2c3d482',
        'f47ac10b-58cc-4372-a567-0e02b2c3d491',
        'f47ac10b-58cc-4372-a567-0e02b2c3d483'
    ]::uuid[]::UUID[],
    'fully-automated',
    'compliant'
),
(
    'Client Disclosure Requirements',
    'FINRA',
    'disclosure',
    'monthly',
    15,
    'client.services@company.com',
    ARRAY[
        'f47ac10b-58cc-4372-a567-0e02b2c3d480',
        'f47ac10b-58cc-4372-a567-0e02b2c3d493'
    ]::uuid[]::UUID[],
    'semi-automated',
    'pending'
);

-- ===========================================
-- PREDICTIVE COMPLIANCE MONITORING
-- =========================================--

-- Create table for compliance predictions
CREATE TABLE IF NOT EXISTS public.pop_compliance_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requirement_id UUID REFERENCES public.pop_compliance_requirements(id) ON DELETE CASCADE,
    prediction_date DATE NOT NULL,
    risk_level TEXT NOT NULL, -- 'low', 'medium', 'high', 'critical'
    risk_score DECIMAL(5,4) NOT NULL,
    predicted_compliance_status TEXT NOT NULL,
    confidence_level DECIMAL(5,4) NOT NULL,
    risk_factors JSONB,
    mitigation_actions JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample compliance predictions
INSERT INTO public.pop_compliance_predictions (
    requirement_id, prediction_date, risk_level, risk_score,
    predicted_compliance_status, confidence_level, risk_factors, mitigation_actions
) VALUES
(
    (SELECT id FROM public.pop_compliance_requirements WHERE requirement_name = 'Form 13F Institutional Holdings Report'),
    '2024-11-15',
    'medium',
    0.65,
    'at_risk',
    0.82,
    '["Data processing delays", "Staffing constraints", "System integration issues"]',
    '["Implement automated data extraction", "Cross-train compliance staff", "Upgrade reporting systems"]'
),
(
    (SELECT id FROM public.pop_compliance_requirements WHERE requirement_name = 'Risk Metrics Monitoring'),
    '2024-09-15',
    'low',
    0.15,
    'compliant',
    0.94,
    '["Minor system latency"]',
    '["Monitor system performance", "Regular maintenance schedule"]'
);

-- ===========================================
-- SUCCESS MESSAGE
-- =========================================--

DO $$
BEGIN
    RAISE NOTICE 'Machine Learning and Advanced Analytics Enhancements have been successfully implemented!';
    RAISE NOTICE 'ML models created: %', (SELECT COUNT(*) FROM public.pop_ml_models);
    RAISE NOTICE 'ML-detected anomalies: %', (SELECT COUNT(*) FROM public.pop_ml_anomalies);
    RAISE NOTICE 'External data sources configured: %', (SELECT COUNT(*) FROM public.pop_external_data_sources);
    RAISE NOTICE 'Data mappings established: %', (SELECT COUNT(*) FROM public.pop_external_data_mappings);
    RAISE NOTICE 'Compliance reports framework: %', (SELECT COUNT(*) FROM public.pop_compliance_reports);
    RAISE NOTICE 'Compliance requirements defined: %', (SELECT COUNT(*) FROM public.pop_compliance_requirements);
    RAISE NOTICE 'Compliance predictions generated: %', (SELECT COUNT(*) FROM public.pop_compliance_predictions);
END $$;
