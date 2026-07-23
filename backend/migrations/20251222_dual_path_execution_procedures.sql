-- =====================================================
-- Real-Time Atomic Metric Refresh Lane
-- Ingests daily/hourly metrics with freshness gates
-- =====================================================

-- =====================================================
-- 1. REAL-TIME ATOMIC REFRESH PROCEDURE
-- =====================================================

CREATE OR REPLACE FUNCTION public.refresh_atomic_metrics(
  p_metric_id UUID DEFAULT NULL,
  p_execution_type TEXT DEFAULT 'refresh'
)
RETURNS TABLE (
  execution_id UUID,
  metric_id UUID,
  metric_name TEXT,
  records_processed INT,
  records_finalized INT,
  sla_met BOOLEAN,
  error_message TEXT
) AS $$
DECLARE
  v_execution_id UUID;
  v_metric_id UUID;
  v_metric_name TEXT;
  v_registry RECORD;
  v_total_records INT := 0;
  v_finalized_count INT := 0;
  v_error_msg TEXT;
  v_success BOOLEAN;
BEGIN
  v_execution_id := gen_random_uuid();
  
  -- Get metrics to refresh (all or specific)
  FOR v_registry IN
    SELECT 
      metric_id, name, sla_freshness_hours, sla_completeness_threshold,
      source_system, refresh_schedule
    FROM semantic_layer.metric_registry
    WHERE (p_metric_id IS NULL OR metric_id = p_metric_id)
      AND status = 'active'
      AND granularity && ARRAY['date']  -- Real-time lane handles daily+ grains
  LOOP
    v_metric_id := v_registry.metric_id;
    v_metric_name := v_registry.name;
    v_total_records := 0;
    v_finalized_count := 0;
    v_success := TRUE;
    v_error_msg := NULL;
    
    BEGIN
      -- Ingest new raw metrics from source
      INSERT INTO public.metrics_finalized (
        metric_id, metric_name, as_of_date, value, source_system,
        completeness_score, freshness_status
      )
      SELECT
        v_metric_id,
        v_metric_name,
        m.metric_time::DATE,
        m.value,
        v_registry.source_system,
        ((m.details->>'completeness_score')::NUMERIC),
        CASE 
          WHEN (NOW() - m.metric_time) <= (v_registry.sla_freshness_hours || ' hours')::INTERVAL 
          THEN 'fresh'
          ELSE 'stale'
        END as freshness_status
      FROM public.metrics m
      WHERE m.metric_type = v_metric_name
        AND m.metric_time >= NOW() - INTERVAL '1 day'
      ON CONFLICT (metric_id, as_of_date) DO UPDATE SET
        value = EXCLUDED.value,
        completeness_score = EXCLUDED.completeness_score,
        freshness_status = EXCLUDED.freshness_status,
        updated_at = NOW();
      
      GET DIAGNOSTICS v_finalized_count = ROW_COUNT;
      
      -- Validate SLA compliance
      v_success := (
        SELECT COUNT(*) >= 1
        FROM public.metrics_finalized
        WHERE metric_id = v_metric_id
          AND as_of_date = CURRENT_DATE
          AND completeness_score >= v_registry.sla_completeness_threshold
          AND freshness_status = 'fresh'
      );
      
      IF NOT v_success THEN
        -- Log SLA violation
        INSERT INTO public.sla_violations (
          metric_id, violation_type, expected_threshold, 
          actual_value, details
        )
        SELECT
          v_metric_id,
          'completeness',
          v_registry.sla_completeness_threshold,
          MAX(mf.completeness_score),
          JSONB_BUILD_OBJECT(
            'metric_name', v_metric_name,
            'as_of_date', CURRENT_DATE,
            'reason', 'completeness_below_threshold'
          )
        FROM public.metrics_finalized mf
        WHERE mf.metric_id = v_metric_id
          AND mf.as_of_date = CURRENT_DATE;
      END IF;
      
    EXCEPTION WHEN OTHERS THEN
      v_success := FALSE;
      v_error_msg := SQLERRM;
    END;
    
    -- Log execution
    INSERT INTO semantic_layer.metric_execution_log (
      execution_id, metric_id, lane, execution_type,
      status, record_count, success_count, error_count,
      completeness_score, error_message, completed_at, duration_ms
    )
    VALUES (
      v_execution_id,
      v_metric_id,
      'real-time',
      p_execution_type,
      CASE WHEN v_success THEN 'completed' ELSE 'failed' END,
      v_finalized_count,
      CASE WHEN v_success THEN v_finalized_count ELSE 0 END,
      CASE WHEN v_success THEN 0 ELSE v_finalized_count END,
      CASE WHEN v_success THEN 100.0 ELSE 0.0 END,
      v_error_msg,
      NOW(),
      0
    );
    
    RETURN QUERY
    SELECT
      v_execution_id,
      v_metric_id,
      v_metric_name,
      v_total_records,
      v_finalized_count,
      v_success,
      v_error_msg;
  END LOOP;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- 2. BATCH MONTHLY PoP COMPUTATION
-- =====================================================

CREATE OR REPLACE FUNCTION public.compute_monthly_pop(
  p_metric_id UUID DEFAULT NULL,
  p_period_start DATE DEFAULT NULL,
  p_period_end DATE DEFAULT NULL
)
RETURNS TABLE (
  execution_id UUID,
  metric_id UUID,
  period_label TEXT,
  records_computed INT,
  computation_status TEXT,
  error_message TEXT
) AS $$
DECLARE
  v_execution_id UUID;
  v_metric_id UUID;
  v_metric_name TEXT;
  v_period_start DATE;
  v_period_end DATE;
  v_period_label TEXT;
  v_records INT := 0;
  v_error_msg TEXT;
  v_success BOOLEAN;
BEGIN
  v_execution_id := gen_random_uuid();
  
  -- Default to prior month if not specified
  IF p_period_start IS NULL THEN
    v_period_start := DATE_TRUNC('month', CURRENT_DATE - INTERVAL '1 month')::DATE;
    v_period_end := (DATE_TRUNC('month', CURRENT_DATE) - INTERVAL '1 day')::DATE;
  ELSE
    v_period_start := p_period_start;
    v_period_end := p_period_end;
  END IF;
  
  v_period_label := TO_CHAR(v_period_start, 'YYYY-MM');
  
  BEGIN
    WITH monthly_current AS (
      SELECT
        metric_id,
        v_period_start as period_start,
        v_period_end as period_end,
        v_period_label as period_label,
        COUNT(*) as record_count,
        SUM(value::NUMERIC) as current_value,
        AVG(value::NUMERIC) as avg_value,
        STDDEV_POP(value::NUMERIC) as stddev_value
      FROM public.metrics
      WHERE (p_metric_id IS NULL OR metric_id = p_metric_id)
        AND metric_time >= v_period_start::TIMESTAMPTZ
        AND metric_time < (v_period_end::TIMESTAMPTZ + INTERVAL '1 day')
        AND metric_type IN (
          SELECT name FROM semantic_layer.metric_registry
          WHERE granularity && ARRAY['month']
        )
      GROUP BY metric_id
    ),
    lagged_periods AS (
      SELECT
        m.*,
        LAG(current_value) OVER (
          PARTITION BY metric_id 
          ORDER BY period_start
        ) as previous_value,
        LAG(period_label) OVER (
          PARTITION BY metric_id 
          ORDER BY period_start
        ) as previous_period_label
      FROM monthly_current m
    )
    INSERT INTO public.pop_computations (
      metric_id, period_start, period_end, granularity, period_label,
      current_value, previous_value, delta, percent_change,
      record_count, computation_status, last_updated
    )
    SELECT
      metric_id,
      period_start,
      period_end,
      'month',
      period_label,
      current_value,
      previous_value,
      ROUND(current_value - COALESCE(previous_value, 0), 4),
      ROUND(CASE 
        WHEN previous_value IS NULL OR previous_value = 0 THEN NULL
        ELSE (current_value - previous_value) / ABS(previous_value) * 100
      END, 4),
      record_count,
      'success',
      NOW()
    FROM lagged_periods
    ON CONFLICT (metric_id, period_start, period_end, granularity) 
    DO UPDATE SET
      current_value = EXCLUDED.current_value,
      previous_value = EXCLUDED.previous_value,
      delta = EXCLUDED.delta,
      percent_change = EXCLUDED.percent_change,
      record_count = EXCLUDED.record_count,
      computation_status = 'success',
      last_updated = NOW();
    
    GET DIAGNOSTICS v_records = ROW_COUNT;
    v_success := TRUE;
    
  EXCEPTION WHEN OTHERS THEN
    v_success := FALSE;
    v_error_msg := SQLERRM;
  END;
  
  -- Log execution
  INSERT INTO semantic_layer.metric_execution_log (
    execution_id, metric_id, lane, execution_type,
    period_start, period_end, period_label,
    status, record_count, success_count, error_count,
    completeness_score, error_message, completed_at
  )
  VALUES (
    v_execution_id,
    p_metric_id,
    'batch',
    'refresh',
    v_period_start,
    v_period_end,
    v_period_label,
    CASE WHEN v_success THEN 'completed' ELSE 'failed' END,
    v_records,
    CASE WHEN v_success THEN v_records ELSE 0 END,
    CASE WHEN v_success THEN 0 ELSE 1 END,
    CASE WHEN v_success THEN 100.0 ELSE 0.0 END,
    v_error_msg,
    NOW()
  );
  
  RETURN QUERY
  SELECT
    v_execution_id,
    COALESCE(p_metric_id, gen_random_uuid()),
    v_period_label,
    v_records,
    CASE WHEN v_success THEN 'success' ELSE 'error' END,
    v_error_msg;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- 3. COMPUTE COMPARISON PERIODS (YoY, QoQ, PoP)
-- =====================================================

CREATE OR REPLACE FUNCTION public.compute_comparison_periods(
  p_metric_id UUID DEFAULT NULL
)
RETURNS TABLE (
  execution_id UUID,
  metric_id UUID,
  period_label TEXT,
  comparison_periods_computed INT
) AS $$
DECLARE
  v_execution_id UUID;
BEGIN
  v_execution_id := gen_random_uuid();
  
  WITH base AS (
    SELECT
      metric_id, period_label, current_value,
      LAG(current_value) OVER (PARTITION BY metric_id ORDER BY period_label) as previous_period,
      LAG(current_value, 12) OVER (PARTITION BY metric_id ORDER BY period_label) as yoy_prior,
      LAG(current_value, 3) OVER (PARTITION BY metric_id ORDER BY period_label) as qoq_prior
    FROM public.pop_computations
    WHERE (p_metric_id IS NULL OR metric_id = p_metric_id)
      AND granularity = 'month'
      AND computation_status = 'success'
  )
  INSERT INTO public.metrics_comparison_periods (
    metric_id, period_label, 
    current_value, previous_period_value, yoy_value, qoq_value,
    previous_period_delta, previous_period_percent_change,
    yoy_delta, yoy_percent_change,
    qoq_delta, qoq_percent_change
  )
  SELECT
    metric_id, period_label,
    current_value, 
    previous_period, 
    yoy_prior, 
    qoq_prior,
    ROUND(current_value - previous_period, 4),
    ROUND(CASE 
      WHEN previous_period IS NULL OR previous_period = 0 THEN NULL
      ELSE (current_value - previous_period) / ABS(previous_period) * 100
    END, 4),
    ROUND(current_value - yoy_prior, 4),
    ROUND(CASE 
      WHEN yoy_prior IS NULL OR yoy_prior = 0 THEN NULL
      ELSE (current_value - yoy_prior) / ABS(yoy_prior) * 100
    END, 4),
    ROUND(current_value - qoq_prior, 4),
    ROUND(CASE 
      WHEN qoq_prior IS NULL OR qoq_prior = 0 THEN NULL
      ELSE (current_value - qoq_prior) / ABS(qoq_prior) * 100
    END, 4)
  FROM base
  ON CONFLICT (metric_id, period_label) DO UPDATE SET
    current_value = EXCLUDED.current_value,
    previous_period_value = EXCLUDED.previous_period_value,
    yoy_value = EXCLUDED.yoy_value,
    qoq_value = EXCLUDED.qoq_value,
    previous_period_delta = EXCLUDED.previous_period_delta,
    previous_period_percent_change = EXCLUDED.previous_period_percent_change,
    yoy_delta = EXCLUDED.yoy_delta,
    yoy_percent_change = EXCLUDED.yoy_percent_change,
    qoq_delta = EXCLUDED.qoq_delta,
    qoq_percent_change = EXCLUDED.qoq_percent_change,
    updated_at = NOW();
  
  RETURN QUERY
  SELECT
    v_execution_id,
    COALESCE(p_metric_id, gen_random_uuid()),
    TO_CHAR(CURRENT_DATE, 'YYYY-MM'),
    (SELECT COUNT(*) FROM public.metrics_comparison_periods 
     WHERE p_metric_id IS NULL OR metric_id = p_metric_id);
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- 4. Z-SCORE ANOMALY DETECTION (WINDOWED)
-- =====================================================

CREATE OR REPLACE FUNCTION public.detect_zscore_anomalies(
  p_metric_id UUID DEFAULT NULL,
  p_zscore_threshold NUMERIC DEFAULT 2.5,
  p_window_days INT DEFAULT 90,
  p_min_data_points INT DEFAULT 7
)
RETURNS TABLE (
  execution_id UUID,
  metric_id UUID,
  computation_id UUID,
  anomaly_type TEXT,
  severity TEXT,
  confidence NUMERIC,
  z_score NUMERIC,
  error_message TEXT
) AS $$
DECLARE
  v_execution_id UUID;
  v_anomaly_count INT := 0;
  v_error_msg TEXT;
BEGIN
  v_execution_id := gen_random_uuid();
  
  BEGIN
    WITH windowed_history AS (
      SELECT
        c.metric_id,
        c.id as computation_id,
        c.current_value::NUMERIC as x,
        c.period_label,
        AVG(c.current_value::NUMERIC) OVER w as mu,
        STDDEV_POP(c.current_value::NUMERIC) OVER w as sigma,
        COUNT(*) OVER w as window_count
      FROM public.pop_computations c
      WHERE (p_metric_id IS NULL OR c.metric_id = p_metric_id)
        AND c.computation_status = 'success'
        AND c.period_end >= CURRENT_DATE - (p_window_days || ' days')::INTERVAL
      WINDOW w AS (
        PARTITION BY c.metric_id
        ORDER BY c.period_end DESC
        ROWS BETWEEN (p_window_days) PRECEDING AND CURRENT ROW
      )
    ),
    scored AS (
      SELECT
        *,
        CASE 
          WHEN sigma = 0 THEN NULL 
          ELSE (x - mu) / sigma 
        END as z_score
      FROM windowed_history
      WHERE window_count >= p_min_data_points
    ),
    flagged AS (
      SELECT
        metric_id,
        computation_id,
        z_score,
        CASE 
          WHEN ABS(z_score) >= 3.0 THEN 'high'
          WHEN ABS(z_score) >= p_zscore_threshold THEN 'medium'
          ELSE 'low'
        END as severity,
        (1 / (1 + EXP(-ABS(z_score))))::NUMERIC(5,4) as confidence,
        'z_score' as anomaly_type,
        NOW() as detected_at,
        'open' as status
      FROM scored
      WHERE ABS(z_score) >= p_zscore_threshold
    )
    INSERT INTO public.pop_anomalies (
      metric_id, computation_id, anomaly_type, severity, confidence,
      z_score, expected_value, actual_value, detection_method,
      detection_params, detected_at, status
    )
    SELECT
      f.metric_id,
      f.computation_id,
      f.anomaly_type,
      f.severity,
      f.confidence,
      f.z_score,
      (SELECT mu FROM scored WHERE computation_id = f.computation_id LIMIT 1),
      (SELECT x FROM scored WHERE computation_id = f.computation_id LIMIT 1),
      'z_score',
      JSONB_BUILD_OBJECT(
        'threshold', p_zscore_threshold,
        'window_days', p_window_days,
        'method', 'z_score',
        'sigma', (SELECT sigma FROM scored WHERE computation_id = f.computation_id LIMIT 1)
      ),
      f.detected_at,
      f.status
    FROM flagged f
    ON CONFLICT (metric_id, computation_id, anomaly_type) DO NOTHING;
    
    GET DIAGNOSTICS v_anomaly_count = ROW_COUNT;
    
  EXCEPTION WHEN OTHERS THEN
    v_error_msg := SQLERRM;
  END;
  
  -- Log execution
  INSERT INTO semantic_layer.metric_execution_log (
    execution_id, metric_id, lane, execution_type,
    status, record_count, success_count, error_message, completed_at
  )
  VALUES (
    v_execution_id,
    p_metric_id,
    'batch',
    'refresh',
    CASE WHEN v_error_msg IS NULL THEN 'completed' ELSE 'failed' END,
    v_anomaly_count,
    v_anomaly_count,
    v_error_msg,
    NOW()
  );
  
  RETURN QUERY
  SELECT
    v_execution_id,
    COALESCE(p_metric_id, gen_random_uuid()),
    gen_random_uuid()::UUID,
    'z_score',
    'medium',
    0.8::NUMERIC,
    2.5::NUMERIC,
    v_error_msg;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- 5. GRANT EXECUTE PERMISSIONS
-- =====================================================

GRANT EXECUTE ON FUNCTION public.refresh_atomic_metrics TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.compute_monthly_pop TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.compute_comparison_periods TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.detect_zscore_anomalies TO PUBLIC;
