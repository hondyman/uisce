-- PoP Metrics Enhancement Functions
-- Created: 2025-09-10
-- Description: Additional functions for data quality, anomaly detection, and dashboard utilities

-- ===========================================
-- DATA QUALITY CHECK FUNCTIONS
-- ===========================================

-- Function to check data completeness for a metric
CREATE OR REPLACE FUNCTION public.check_metric_data_completeness(
    p_metric_id UUID,
    p_start_date DATE,
    p_end_date DATE,
    p_expected_frequency TEXT DEFAULT 'daily'
)
RETURNS TABLE (
    metric_id UUID,
    period_start DATE,
    period_end DATE,
    expected_records INTEGER,
    actual_records INTEGER,
    completeness_percentage DECIMAL(5,2),
    status TEXT
) AS $$
DECLARE
    v_granularity TEXT;
    v_base_query TEXT;
    v_date_column TEXT;
    v_total_days INTEGER;
    v_actual_records INTEGER;
BEGIN
    -- Get metric configuration
    SELECT granularity, base_query, date_column
    INTO v_granularity, v_base_query, v_date_column
    FROM public.pop_metrics
    WHERE id = p_metric_id;

    -- Calculate expected records based on frequency
    v_total_days := p_end_date - p_start_date + 1;

    CASE p_expected_frequency
        WHEN 'daily' THEN
            expected_records := v_total_days;
        WHEN 'weekly' THEN
            expected_records := CEIL(v_total_days / 7.0);
        WHEN 'monthly' THEN
            expected_records := EXTRACT(MONTH FROM age(p_end_date, p_start_date)) + 1;
        ELSE
            expected_records := v_total_days;
    END CASE;

    -- Count actual records (this is a simplified version - in practice you'd execute the base_query)
    SELECT COUNT(*)
    INTO v_actual_records
    FROM public.pop_computations
    WHERE metric_id = p_metric_id
    AND period_start >= p_start_date
    AND period_end <= p_end_date;

    -- Calculate completeness
    completeness_percentage := CASE
        WHEN expected_records > 0 THEN (v_actual_records::DECIMAL / expected_records) * 100
        ELSE 0
    END;

    -- Determine status
    status := CASE
        WHEN completeness_percentage >= 95 THEN 'excellent'
        WHEN completeness_percentage >= 90 THEN 'good'
        WHEN completeness_percentage >= 80 THEN 'fair'
        ELSE 'poor'
    END;

    RETURN QUERY
    SELECT
        p_metric_id,
        p_start_date,
        p_end_date,
        expected_records,
        v_actual_records,
        ROUND(completeness_percentage, 2),
        status::TEXT;
END;
$$ LANGUAGE plpgsql;

-- Function to detect statistical anomalies using multiple methods
CREATE OR REPLACE FUNCTION public.detect_multimethod_anomalies(
    p_metric_id UUID,
    p_lookback_days INTEGER DEFAULT 90,
    p_sensitivity DECIMAL DEFAULT 0.8
)
RETURNS TABLE (
    metric_id UUID,
    period_start DATE,
    period_end DATE,
    current_value DECIMAL,
    z_score DECIMAL,
    iqr_score DECIMAL,
    trend_score DECIMAL,
    combined_score DECIMAL,
    anomaly_type TEXT,
    severity TEXT,
    confidence DECIMAL
) AS $$
DECLARE
    v_values DECIMAL[];
    v_dates DATE[];
    v_current_value DECIMAL;
    v_current_date DATE;
    v_mean DECIMAL;
    v_stddev DECIMAL;
    v_q1 DECIMAL;
    v_q3 DECIMAL;
    v_iqr DECIMAL;
    v_trend_slope DECIMAL;
BEGIN
    -- Get historical values for the lookback period
    SELECT
        ARRAY_AGG(current_value ORDER BY period_end),
        ARRAY_AGG(period_end ORDER BY period_end),
        (ARRAY_AGG(current_value ORDER BY period_end DESC))[1],
        (ARRAY_AGG(period_end ORDER BY period_end DESC))[1]
    INTO v_values, v_dates, v_current_value, v_current_date
    FROM public.pop_computations
    WHERE metric_id = p_metric_id
    AND period_end >= CURRENT_DATE - (p_lookback_days || ' days')::INTERVAL
    AND current_value IS NOT NULL;

    -- Skip if insufficient data
    IF array_length(v_values, 1) < 7 THEN
        RETURN;
    END IF;

    -- Calculate Z-Score
    SELECT AVG(v), STDDEV(v) INTO v_mean, v_stddev
    FROM unnest(v_values[1:array_length(v_values, 1)-1]) v;

    z_score := CASE
        WHEN v_stddev > 0 THEN (v_current_value - v_mean) / v_stddev
        ELSE 0
    END;

    -- Calculate IQR-based outlier detection
    SELECT
        percentile_cont(0.25) WITHIN GROUP (ORDER BY v),
        percentile_cont(0.75) WITHIN GROUP (ORDER BY v)
    INTO v_q1, v_q3
    FROM unnest(v_values[1:array_length(v_values, 1)-1]) v;

    v_iqr := v_q3 - v_q1;
    iqr_score := CASE
        WHEN v_iqr > 0 THEN
            CASE
                WHEN v_current_value < v_q1 - 1.5 * v_iqr THEN -2.0
                WHEN v_current_value > v_q3 + 1.5 * v_iqr THEN 2.0
                ELSE 0
            END
        ELSE 0
    END;

    -- Calculate trend-based anomaly (simplified linear trend)
    WITH trend_calc AS (
        SELECT
            regr_slope(v, row_number) as slope,
            regr_intercept(v, row_number) as intercept
        FROM (
            SELECT v, ROW_NUMBER() OVER (ORDER BY d) as row_number
            FROM unnest(v_values[1:array_length(v_values, 1)-1], v_dates[1:array_length(v_dates, 1)-1]) as t(v, d)
        ) t
    )
    SELECT slope INTO v_trend_slope FROM trend_calc;

    -- Expected value based on trend
    trend_score := CASE
        WHEN v_trend_slope IS NOT NULL THEN
            v_current_value - (v_trend_slope * array_length(v_values, 1) + (SELECT intercept FROM trend_calc))
        ELSE 0
    END;

    -- Combine scores (weighted average)
    combined_score := (ABS(z_score) * 0.4) + (ABS(iqr_score) * 0.3) + (ABS(trend_score / GREATEST(ABS(v_mean), 1)) * 0.3);

    -- Determine anomaly type and severity
    anomaly_type := CASE
        WHEN ABS(z_score) > ABS(iqr_score) AND ABS(z_score) > ABS(trend_score / GREATEST(ABS(v_mean), 1)) THEN 'z_score'
        WHEN ABS(iqr_score) > ABS(trend_score / GREATEST(ABS(v_mean), 1)) THEN 'iqr'
        ELSE 'trend_break'
    END;

    severity := CASE
        WHEN combined_score > 3.0 THEN 'critical'
        WHEN combined_score > 2.0 THEN 'high'
        WHEN combined_score > 1.5 THEN 'medium'
        WHEN combined_score > 1.0 THEN 'low'
        ELSE 'normal'
    END;

    confidence := GREATEST(0, LEAST(1, 1 - (1 / GREATEST(combined_score, 0.1))));

    -- Only return if it's an anomaly
    IF combined_score > 1.0 THEN
        RETURN QUERY
        SELECT
            p_metric_id,
            v_current_date - INTERVAL '1 day',
            v_current_date,
            v_current_value,
            ROUND(z_score, 3),
            ROUND(iqr_score, 3),
            ROUND(trend_score, 3),
            ROUND(combined_score, 3),
            anomaly_type,
            severity,
            ROUND(confidence, 3);
    END IF;
END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- DASHBOARD UTILITY FUNCTIONS
-- ===========================================

-- Function to get dashboard summary for a user
CREATE OR REPLACE FUNCTION public.get_user_dashboard_summary(p_user_id TEXT)
RETURNS TABLE (
    dashboard_name TEXT,
    total_metrics INTEGER,
    anomaly_count INTEGER,
    golden_path_count INTEGER,
    last_updated TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        d.name,
        COUNT(DISTINCT dw.metric_ids) as total_metrics,
        COUNT(DISTINCT CASE WHEN a.id IS NOT NULL THEN dw.metric_ids END) as anomaly_count,
        COUNT(DISTINCT CASE WHEN m.golden_path THEN dw.metric_ids END) as golden_path_count,
        MAX(m.updated_at) as last_updated
    FROM public.pop_dashboards d
    LEFT JOIN public.pop_dashboard_widgets dw ON d.id = dw.dashboard_id
    LEFT JOIN public.pop_metrics m ON m.id = ANY(dw.metric_ids)
    LEFT JOIN public.pop_anomalies a ON a.metric_id = ANY(dw.metric_ids) AND a.status = 'open'
    WHERE d.owner_user_id = p_user_id OR d.is_public = true OR p_user_id = ANY(d.allowed_users)
    GROUP BY d.id, d.name;
END;
$$ LANGUAGE plpgsql;

-- Function to get metric health score
CREATE OR REPLACE FUNCTION public.calculate_metric_health_score(p_metric_id UUID)
RETURNS TABLE (
    metric_id UUID,
    data_quality_score DECIMAL(5,2),
    anomaly_score DECIMAL(5,2),
    timeliness_score DECIMAL(5,2),
    overall_health_score DECIMAL(5,2),
    health_status TEXT
) AS $$
DECLARE
    v_metric_record RECORD;
    v_data_quality DECIMAL := 0;
    v_anomaly_score DECIMAL := 0;
    v_timeliness DECIMAL := 0;
    v_overall_score DECIMAL := 0;
BEGIN
    -- Get metric info
    SELECT * INTO v_metric_record
    FROM public.pop_metrics
    WHERE id = p_metric_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Calculate data quality score (simplified)
    SELECT AVG(CASE
        WHEN record_count > 0 THEN 100
        ELSE 0
    END) INTO v_data_quality
    FROM public.pop_computations
    WHERE metric_id = p_metric_id
    AND period_end >= CURRENT_DATE - INTERVAL '30 days';

    -- Calculate anomaly score (lower anomalies = higher score)
    SELECT GREATEST(0, 100 - (COUNT(*) * 10)) INTO v_anomaly_score
    FROM public.pop_anomalies
    WHERE metric_id = p_metric_id
    AND status = 'open'
    AND detected_at >= CURRENT_DATE - INTERVAL '30 days';

    -- Calculate timeliness score
    SELECT CASE
        WHEN last_updated >= CURRENT_DATE - (v_metric_record.sla_freshness_hours || ' hours')::INTERVAL THEN 100
        WHEN last_updated >= CURRENT_DATE - (v_metric_record.sla_freshness_hours * 2 || ' hours')::INTERVAL THEN 75
        WHEN last_updated >= CURRENT_DATE - (v_metric_record.sla_freshness_hours * 3 || ' hours')::INTERVAL THEN 50
        ELSE 25
    END INTO v_timeliness
    FROM (
        SELECT MAX(last_updated) as last_updated
        FROM public.pop_computations
        WHERE metric_id = p_metric_id
    ) t;

    -- Calculate overall score
    v_overall_score := (COALESCE(v_data_quality, 0) * 0.4) +
                      (COALESCE(v_anomaly_score, 100) * 0.4) +
                      (COALESCE(v_timeliness, 0) * 0.2);

    -- Determine health status
    health_status := CASE
        WHEN v_overall_score >= 90 THEN 'excellent'
        WHEN v_overall_score >= 80 THEN 'good'
        WHEN v_overall_score >= 70 THEN 'fair'
        WHEN v_overall_score >= 60 THEN 'needs_attention'
        ELSE 'critical'
    END;

    RETURN QUERY
    SELECT
        p_metric_id,
        ROUND(COALESCE(v_data_quality, 0), 2),
        ROUND(COALESCE(v_anomaly_score, 100), 2),
        ROUND(COALESCE(v_timeliness, 0), 2),
        ROUND(v_overall_score, 2),
        health_status;
END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- GOVERNANCE AND COMPLIANCE FUNCTIONS
-- ===========================================

-- Function to get governance summary for a domain
CREATE OR REPLACE FUNCTION public.get_domain_governance_summary(p_domain TEXT)
RETURNS TABLE (
    domain TEXT,
    total_metrics INTEGER,
    golden_path_metrics INTEGER,
    active_anomalies INTEGER,
    pending_reviews INTEGER,
    compliance_score DECIMAL(5,2)
) AS $$
DECLARE
    v_total_metrics INTEGER := 0;
    v_golden_path INTEGER := 0;
    v_anomalies INTEGER := 0;
    v_reviews INTEGER := 0;
    v_compliance DECIMAL := 0;
BEGIN
    -- Count metrics
    SELECT COUNT(*) INTO v_total_metrics
    FROM public.pop_metrics
    WHERE domain = p_domain AND status = 'active';

    -- Count golden path metrics
    SELECT COUNT(*) INTO v_golden_path
    FROM public.pop_metrics
    WHERE domain = p_domain AND golden_path = true;

    -- Count active anomalies
    SELECT COUNT(*) INTO v_anomalies
    FROM public.pop_anomalies a
    JOIN public.pop_metrics m ON a.metric_id = m.id
    WHERE m.domain = p_domain AND a.status = 'open';

    -- Count pending reviews
    SELECT COUNT(*) INTO v_reviews
    FROM public.pop_steward_reviews sr
    JOIN public.pop_metrics m ON sr.metric_id = m.id
    WHERE m.domain = p_domain AND sr.status IN ('in_progress', 'overdue');

    -- Calculate compliance score
    v_compliance := CASE
        WHEN v_total_metrics > 0 THEN
            ((v_golden_path::DECIMAL / v_total_metrics) * 60) +
            (GREATEST(0, 30 - v_anomalies) / 30.0 * 30) +
            (GREATEST(0, 10 - v_reviews) / 10.0 * 10)
        ELSE 0
    END;

    RETURN QUERY
    SELECT
        p_domain,
        v_total_metrics,
        v_golden_path,
        v_anomalies,
        v_reviews,
        ROUND(LEAST(100, GREATEST(0, v_compliance)), 2);
END;
$$ LANGUAGE plpgsql;

-- Function to promote metric to golden path with audit trail
CREATE OR REPLACE FUNCTION public.promote_metric_to_golden_path(
    p_metric_id UUID,
    p_user_id TEXT,
    p_reason TEXT
)
RETURNS BOOLEAN AS $$
DECLARE
    v_old_status BOOLEAN;
BEGIN
    -- Get current status
    SELECT golden_path INTO v_old_status
    FROM public.pop_metrics
    WHERE id = p_metric_id;

    -- Update metric
    UPDATE public.pop_metrics
    SET golden_path = true, updated_at = NOW(), updated_by = p_user_id
    WHERE id = p_metric_id;

    -- Add audit comment
    INSERT INTO public.pop_steward_comments (
        metric_id, commenter_user_id, comment_type, comment_text
    ) VALUES (
        p_metric_id, p_user_id, 'golden_path',
        'Promoted to golden path: ' || COALESCE(p_reason, 'No reason provided')
    );

    -- Log the change (using steward comments for audit trail)
    INSERT INTO public.pop_steward_comments (
        metric_id, commenter_user_id, comment_type, comment_text
    ) VALUES (
        p_metric_id, p_user_id, 'audit',
        'Status change - Golden Path: ' || CASE WHEN v_old_status THEN 'already true' ELSE 'false -> true' END ||
        '. Reason: ' || COALESCE(p_reason, 'No reason provided')
    );

    RETURN true;
EXCEPTION
    WHEN OTHERS THEN
        RETURN false;
END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- UTILITY FUNCTIONS
-- ===========================================

-- Function to clean up old computation data
CREATE OR REPLACE FUNCTION public.cleanup_old_computations(p_retention_days INTEGER DEFAULT 365)
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    DELETE FROM public.pop_computations
    WHERE period_end < CURRENT_DATE - (p_retention_days || ' days')::INTERVAL;

    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;

    RAISE NOTICE 'Cleaned up % old computation records', v_deleted_count;

    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to refresh materialized views
CREATE OR REPLACE FUNCTION public.refresh_pop_views()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY public.pop_metrics_with_latest;
    REFRESH MATERIALIZED VIEW CONCURRENTLY public.pop_anomaly_summary;

    RAISE NOTICE 'Refreshed PoP materialized views';
END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- SUCCESS MESSAGE
-- ===========================================

DO $$
BEGIN
    RAISE NOTICE 'PoP Metrics enhancement functions have been successfully created!';
    RAISE NOTICE 'Available functions:';
    RAISE NOTICE '  - check_metric_data_completeness()';
    RAISE NOTICE '  - detect_multimethod_anomalies()';
    RAISE NOTICE '  - get_user_dashboard_summary()';
    RAISE NOTICE '  - calculate_metric_health_score()';
    RAISE NOTICE '  - get_domain_governance_summary()';
    RAISE NOTICE '  - promote_metric_to_golden_path()';
    RAISE NOTICE '  - cleanup_old_computations()';
    RAISE NOTICE '  - refresh_pop_views()';
END $$;
