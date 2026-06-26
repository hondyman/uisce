-- PoP System Administration and Maintenance Scripts
-- Created: 2025-09-10
-- Description: Administrative scripts for managing the PoP metrics system

-- ===========================================
-- SYSTEM HEALTH CHECKS
-- ===========================================

-- Function to perform comprehensive system health check
CREATE OR REPLACE FUNCTION public.pop_system_health_check()
RETURNS TABLE (
    check_name TEXT,
    status TEXT,
    details TEXT,
    recommendation TEXT
) AS $$
DECLARE
    v_metric_count INTEGER;
    v_computation_count INTEGER;
    v_anomaly_count INTEGER;
    v_stale_metrics INTEGER;
    v_missing_data INTEGER;
BEGIN
    -- Check metric count
    SELECT COUNT(*) INTO v_metric_count FROM public.pop_metrics WHERE status = 'active';
    status := CASE WHEN v_metric_count > 0 THEN 'PASS' ELSE 'FAIL' END;
    RETURN QUERY SELECT
        'Active Metrics Count'::TEXT,
        status,
        'Found ' || v_metric_count || ' active metrics'::TEXT,
        CASE WHEN v_metric_count = 0 THEN 'Add metrics to the system' ELSE 'System healthy' END;

    -- Check recent computations
    SELECT COUNT(*) INTO v_computation_count
    FROM public.pop_computations
    WHERE last_updated >= CURRENT_DATE - INTERVAL '7 days';

    status := CASE WHEN v_computation_count > 0 THEN 'PASS' ELSE 'WARN' END;
    RETURN QUERY SELECT
        'Recent Computations'::TEXT,
        status,
        'Found ' || v_computation_count || ' computations in last 7 days'::TEXT,
        CASE WHEN v_computation_count = 0 THEN 'Check computation pipeline' ELSE 'System healthy' END;

    -- Check for stale metrics
    SELECT COUNT(*) INTO v_stale_metrics
    FROM public.pop_metrics m
    LEFT JOIN public.pop_computations c ON m.id = c.metric_id
        AND c.last_updated >= CURRENT_DATE - (m.sla_freshness_hours || ' hours')::INTERVAL
    WHERE m.status = 'active' AND c.id IS NULL;

    status := CASE WHEN v_stale_metrics = 0 THEN 'PASS' ELSE 'WARN' END;
    RETURN QUERY SELECT
        'Stale Metrics'::TEXT,
        status,
        v_stale_metrics || ' metrics have stale data'::TEXT,
        CASE WHEN v_stale_metrics > 0 THEN 'Review data pipeline for affected metrics' ELSE 'All metrics up to date' END;

    -- Check anomaly processing
    SELECT COUNT(*) INTO v_anomaly_count
    FROM public.pop_anomalies
    WHERE status = 'open' AND detected_at >= CURRENT_DATE - INTERVAL '30 days';

    RETURN QUERY SELECT
        'Anomaly Processing'::TEXT,
        'INFO'::TEXT,
        v_anomaly_count || ' open anomalies in last 30 days'::TEXT,
        'Monitor anomaly trends'::TEXT;

    -- Check data quality
    SELECT COUNT(*) INTO v_missing_data
    FROM public.pop_computations
    WHERE current_value IS NULL
    AND period_end >= CURRENT_DATE - INTERVAL '30 days';

    status := CASE WHEN v_missing_data = 0 THEN 'PASS' ELSE 'WARN' END;
    RETURN QUERY SELECT
        'Data Quality'::TEXT,
        status,
        v_missing_data || ' records with missing values in last 30 days'::TEXT,
        CASE WHEN v_missing_data > 0 THEN 'Review data sources for missing values' ELSE 'Data quality is good' END;

END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- METRIC ONBOARDING WIZARD
-- ===========================================

-- Function to create a new metric with validation
CREATE OR REPLACE FUNCTION public.create_pop_metric_wizard(
    p_name TEXT,
    p_display_name TEXT,
    p_description TEXT,
    p_domain TEXT,
    p_category TEXT,
    p_metric_type TEXT,
    p_base_query TEXT,
    p_aggregation_function TEXT,
    p_date_column TEXT,
    p_value_column TEXT,
    p_granularity TEXT,
    p_owner_user_id TEXT,
    p_steward_group TEXT,
    p_data_source TEXT,
    p_schema_name TEXT,
    p_table_name TEXT,
    p_sla_freshness_hours INTEGER DEFAULT 24,
    p_sla_completeness_threshold DECIMAL DEFAULT 0.95
)
RETURNS UUID AS $$
DECLARE
    v_metric_id UUID;
    v_validation_errors TEXT[] := ARRAY[];
BEGIN
    -- Validate required fields
    IF p_name IS NULL OR trim(p_name) = '' THEN
        v_validation_errors := v_validation_errors || 'Metric name is required';
    END IF;

    IF p_base_query IS NULL OR trim(p_base_query) = '' THEN
        v_validation_errors := v_validation_errors || 'Base query is required';
    END IF;

    IF p_date_column IS NULL OR trim(p_date_column) = '' THEN
        v_validation_errors := v_validation_errors || 'Date column is required';
    END IF;

    IF p_value_column IS NULL OR trim(p_value_column) = '' THEN
        v_validation_errors := v_validation_errors || 'Value column is required';
    END IF;

    -- Check for duplicate names
    IF EXISTS (SELECT 1 FROM public.pop_metrics WHERE name = p_name) THEN
        v_validation_errors := v_validation_errors || 'Metric name already exists';
    END IF;

    -- Validate domain and category
    IF p_domain NOT IN ('finance', 'operations', 'compliance', 'risk', 'marketing', 'sales') THEN
        v_validation_errors := v_validation_errors || 'Invalid domain. Must be one of: finance, operations, compliance, risk, marketing, sales';
    END IF;

    -- If validation errors exist, raise exception
    IF array_length(v_validation_errors, 1) > 0 THEN
        RAISE EXCEPTION 'Validation failed: %', array_to_string(v_validation_errors, '; ');
    END IF;

    -- Create the metric
    INSERT INTO public.pop_metrics (
        name, display_name, description, domain, category, metric_type,
        base_query, aggregation_function, date_column, value_column,
        granularity, owner_user_id, steward_group, data_source,
        schema_name, table_name, sla_freshness_hours, sla_completeness_threshold,
        created_by
    ) VALUES (
        p_name, p_display_name, p_description, p_domain, p_category, p_metric_type,
        p_base_query, p_aggregation_function, p_date_column, p_value_column,
        p_granularity, p_owner_user_id, p_steward_group, p_data_source,
        p_schema_name, p_table_name, p_sla_freshness_hours, p_sla_completeness_threshold,
        p_owner_user_id
    ) RETURNING id INTO v_metric_id;

    -- Add default tags
    INSERT INTO public.pop_metric_tags (metric_id, tag_name, tag_value) VALUES
    (v_metric_id, 'created_via', 'wizard'),
    (v_metric_id, 'domain', p_domain),
    (v_metric_id, 'category', p_category);

    -- Log the creation
    INSERT INTO public.pop_steward_comments (
        metric_id, commenter_user_id, comment_type, comment_text
    ) VALUES (
        v_metric_id, p_owner_user_id, 'general',
        'Metric created via onboarding wizard'
    );

    RETURN v_metric_id;
END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- BULK OPERATIONS
-- ===========================================

-- Function to bulk update metric status
CREATE OR REPLACE FUNCTION public.bulk_update_metric_status(
    p_metric_ids UUID[],
    p_new_status TEXT,
    p_user_id TEXT,
    p_reason TEXT DEFAULT NULL
)
RETURNS INTEGER AS $$
DECLARE
    v_updated_count INTEGER;
BEGIN
    -- Validate status
    IF p_new_status NOT IN ('draft', 'active', 'deprecated', 'golden') THEN
        RAISE EXCEPTION 'Invalid status. Must be one of: draft, active, deprecated, golden';
    END IF;

    -- Update metrics
    UPDATE public.pop_metrics
    SET status = p_new_status, updated_at = NOW(), updated_by = p_user_id
    WHERE id = ANY(p_metric_ids);

    GET DIAGNOSTICS v_updated_count = ROW_COUNT;

    -- Log the bulk update
    INSERT INTO public.pop_steward_comments (
        metric_id, commenter_user_id, comment_type, comment_text
    )
    SELECT
        unnest(p_metric_ids),
        p_user_id,
        'general',
        'Bulk status update to ' || p_new_status || COALESCE(': ' || p_reason, '')
    FROM unnest(p_metric_ids);

    RETURN v_updated_count;
END;
$$ LANGUAGE plpgsql;

-- Function to bulk promote metrics to golden path
CREATE OR REPLACE FUNCTION public.bulk_promote_to_golden_path(
    p_metric_ids UUID[],
    p_user_id TEXT,
    p_reason TEXT DEFAULT NULL
)
RETURNS INTEGER AS $$
DECLARE
    v_promoted_count INTEGER;
BEGIN
    -- Update metrics
    UPDATE public.pop_metrics
    SET golden_path = true, updated_at = NOW(), updated_by = p_user_id
    WHERE id = ANY(p_metric_ids);

    GET DIAGNOSTICS v_promoted_count = ROW_COUNT;

    -- Log the promotions
    INSERT INTO public.pop_steward_comments (
        metric_id, commenter_user_id, comment_type, comment_text
    )
    SELECT
        unnest(p_metric_ids),
        p_user_id,
        'golden_path',
        'Bulk promoted to golden path' || COALESCE(': ' || p_reason, '')
    FROM unnest(p_metric_ids);

    RETURN v_promoted_count;
END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- DATA EXPORT AND REPORTING
-- ===========================================

-- Function to export metrics data for reporting
CREATE OR REPLACE FUNCTION public.export_metrics_report(
    p_domain TEXT DEFAULT NULL,
    p_start_date DATE DEFAULT NULL,
    p_end_date DATE DEFAULT NULL,
    p_include_anomalies BOOLEAN DEFAULT true
)
RETURNS TABLE (
    metric_id UUID,
    metric_name TEXT,
    domain TEXT,
    category TEXT,
    period_start DATE,
    period_end DATE,
    current_value DECIMAL,
    previous_value DECIMAL,
    percent_change DECIMAL,
    anomaly_count INTEGER,
    health_status TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.id,
        m.display_name,
        m.domain,
        m.category,
        c.period_start,
        c.period_end,
        c.current_value,
        c.previous_value,
        c.percent_change,
        COALESCE(anomaly_counts.anomaly_count, 0),
        CASE
            WHEN m.golden_path THEN 'golden'
            WHEN COALESCE(anomaly_counts.anomaly_count, 0) > 0 THEN 'at_risk'
            ELSE 'healthy'
        END as health_status
    FROM public.pop_metrics m
    LEFT JOIN public.pop_computations c ON m.id = c.metric_id
    LEFT JOIN (
        SELECT
            metric_id,
            COUNT(*) as anomaly_count
        FROM public.pop_anomalies
        WHERE status = 'open'
        GROUP BY metric_id
    ) anomaly_counts ON m.id = anomaly_counts.metric_id
    WHERE m.status = 'active'
    AND (p_domain IS NULL OR m.domain = p_domain)
    AND (p_start_date IS NULL OR c.period_start >= p_start_date)
    AND (p_end_date IS NULL OR c.period_end <= p_end_date)
    ORDER BY m.domain, m.category, m.display_name, c.period_end DESC;
END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- MAINTENANCE TASKS
-- =========================================--

-- Function to archive old anomalies
CREATE OR REPLACE FUNCTION public.archive_old_anomalies(p_days_old INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    v_archived_count INTEGER;
BEGIN
    UPDATE public.pop_anomalies
    SET status = 'archived'
    WHERE status = 'resolved'
    AND resolved_at < CURRENT_DATE - (p_days_old || ' days')::INTERVAL;

    GET DIAGNOSTICS v_archived_count = ROW_COUNT;

    RAISE NOTICE 'Archived % old resolved anomalies', v_archived_count;

    RETURN v_archived_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up orphaned records
CREATE OR REPLACE FUNCTION public.cleanup_orphaned_records()
RETURNS TABLE (table_name TEXT, cleaned_count INTEGER) AS $$
DECLARE
    v_count INTEGER;
BEGIN
    -- Clean up orphaned computations
    DELETE FROM public.pop_computations
    WHERE metric_id NOT IN (SELECT id FROM public.pop_metrics);

    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN QUERY SELECT 'pop_computations'::TEXT, v_count;

    -- Clean up orphaned anomalies
    DELETE FROM public.pop_anomalies
    WHERE metric_id NOT IN (SELECT id FROM public.pop_metrics);

    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN QUERY SELECT 'pop_anomalies'::TEXT, v_count;

    -- Clean up orphaned reviews
    DELETE FROM public.pop_steward_reviews
    WHERE metric_id NOT IN (SELECT id FROM public.pop_metrics);

    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN QUERY SELECT 'pop_steward_reviews'::TEXT, v_count;

    -- Clean up orphaned comments
    DELETE FROM public.pop_steward_comments
    WHERE metric_id NOT IN (SELECT id FROM public.pop_metrics)
    AND review_id IS NULL;

    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN QUERY SELECT 'pop_steward_comments'::TEXT, v_count;

    -- Clean up orphaned tags
    DELETE FROM public.pop_metric_tags
    WHERE metric_id NOT IN (SELECT id FROM public.pop_metrics);

    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN QUERY SELECT 'pop_metric_tags'::TEXT, v_count;
END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- MONITORING AND ALERTING
-- =========================================--

-- Function to get system metrics for monitoring
CREATE OR REPLACE FUNCTION public.get_pop_system_metrics()
RETURNS TABLE (
    metric_name TEXT,
    metric_value INTEGER,
    metric_type TEXT,
    last_updated TIMESTAMP
) AS $$
BEGIN
    -- Total metrics
    RETURN QUERY SELECT
        'total_metrics'::TEXT,
        COUNT(*)::INTEGER,
        'count'::TEXT,
        MAX(updated_at)
    FROM public.pop_metrics;

    -- Active metrics
    RETURN QUERY SELECT
        'active_metrics'::TEXT,
        COUNT(*)::INTEGER,
        'count'::TEXT,
        MAX(updated_at)
    FROM public.pop_metrics WHERE status = 'active';

    -- Golden path metrics
    RETURN QUERY SELECT
        'golden_path_metrics'::TEXT,
        COUNT(*)::INTEGER,
        'count'::TEXT,
        MAX(updated_at)
    FROM public.pop_metrics WHERE golden_path = true;

    -- Open anomalies
    RETURN QUERY SELECT
        'open_anomalies'::TEXT,
        COUNT(*)::INTEGER,
        'count'::TEXT,
        MAX(detected_at)
    FROM public.pop_anomalies WHERE status = 'open';

    -- Recent computations
    RETURN QUERY SELECT
        'recent_computations'::TEXT,
        COUNT(*)::INTEGER,
        'count'::TEXT,
        MAX(last_updated)
    FROM public.pop_computations
    WHERE last_updated >= CURRENT_DATE - INTERVAL '24 hours';

    -- Pending reviews
    RETURN QUERY SELECT
        'pending_reviews'::TEXT,
        COUNT(*)::INTEGER,
        'count'::TEXT,
        MAX(created_at)
    FROM public.pop_steward_reviews
    WHERE status IN ('in_progress', 'overdue');
END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- SUCCESS MESSAGE
-- =========================================--

DO $$
BEGIN
    RAISE NOTICE 'PoP System Administration Scripts have been successfully created!';
    RAISE NOTICE 'Available functions:';
    RAISE NOTICE '  - pop_system_health_check() - Comprehensive system health check';
    RAISE NOTICE '  - create_pop_metric_wizard() - Guided metric creation with validation';
    RAISE NOTICE '  - bulk_update_metric_status() - Bulk status updates';
    RAISE NOTICE '  - bulk_promote_to_golden_path() - Bulk golden path promotions';
    RAISE NOTICE '  - export_metrics_report() - Data export for reporting';
    RAISE NOTICE '  - archive_old_anomalies() - Archive old resolved anomalies';
    RAISE NOTICE '  - cleanup_orphaned_records() - Clean up orphaned records';
    RAISE NOTICE '  - get_pop_system_metrics() - System metrics for monitoring';
END $$;
