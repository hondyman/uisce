package activities

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	kafka "github.com/segmentio/kafka-go"
)

// ComputeRequest mirrors workflows.ComputeRequest for activity params
type ComputeRequest struct {
	TenantID    string    `json:"tenant_id"`
	MetricID    string    `json:"metric_id"`
	CalcType    string    `json:"calc_type"`    // "pop" | "anomaly"
	PeriodLabel string    `json:"period_label"` // e.g., "2024-08"
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	RunID       string    `json:"run_id"`
}

// ActivityConfig holds dependencies for activities
type ActivityConfig struct {
	DB           *sql.DB
	KafkaBrokers string
	KafkaWriter  *kafka.Writer
}

var globalConfig *ActivityConfig

// Initialize sets up the global activity config
func Initialize(cfg *ActivityConfig) {
	globalConfig = cfg
}

// ============================================================================
// ACTIVITY IMPLEMENTATIONS
// ============================================================================

// UpsertRunStatus updates metric_job_runs status in Postgres
func UpsertRunStatus(ctx context.Context, req ComputeRequest, status string) error {
	if globalConfig == nil || globalConfig.DB == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `
    INSERT INTO metric_job_runs(
      run_id, tenant_id, metric_id, calc_type, period_label, 
      period_start, period_end, status, started_at
    )
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE(
      (SELECT started_at FROM metric_job_runs WHERE run_id=$1), 
      now()
    ))
    ON CONFLICT (tenant_id, metric_id, calc_type, period_label) 
    DO UPDATE SET 
      status=$8, 
      ended_at = CASE WHEN $8 IN ('success','failed') THEN now() ELSE NULL END
  `

	_, err := globalConfig.DB.ExecContext(ctx,
		query,
		req.RunID, req.TenantID, req.MetricID, req.CalcType,
		req.PeriodLabel, req.PeriodStart, req.PeriodEnd, status)

	if err != nil {
		return fmt.Errorf("failed to upsert run status: %w", err)
	}

	return nil
}

// ComputeAndMergePoP orchestrates PoP calculation via Trino
func ComputeAndMergePoP(ctx context.Context, req ComputeRequest) error {
	if globalConfig == nil || globalConfig.DB == nil {
		return fmt.Errorf("dependencies not initialized")
	}

	// 1) Fetch metric definition from Postgres
	var compLogic string
	err := globalConfig.DB.QueryRowContext(ctx,
		`SELECT computation_logic FROM metric_registry WHERE metric_id=$1 AND tenant_id=$2`,
		req.MetricID, req.TenantID).Scan(&compLogic)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("metric not found: tenant_id=%s, metric_id=%s", req.TenantID, req.MetricID)
		}
		return fmt.Errorf("failed to fetch metric: %w", err)
	}

	// 2) Generate MERGE SQL using template
	mergeSQL := generatePopMergeSQL(req.TenantID, req.MetricID, req.PeriodLabel,
		req.PeriodStart, req.PeriodEnd, req.RunID)

	// 3) Execute MERGE against Trino/Iceberg
	// For now, log the SQL and simulate execution
	fmt.Printf("Executing PoP MERGE SQL:\n%s\n", mergeSQL)

	stats := map[string]interface{}{
		"success":      true,
		"record_count": 100,
		"duration_ms":  2500,
	}

	// 4) Update run with stats
	statsJSON, _ := json.Marshal(stats)
	_, err = globalConfig.DB.ExecContext(ctx, `
    UPDATE metric_job_runs SET stats=$1 WHERE run_id=$2
  `, statsJSON, req.RunID)
	if err != nil {
		return fmt.Errorf("failed to update run stats: %w", err)
	}

	return nil
}

// generatePopMergeSQL generates MERGE SQL for PoP calculations
func generatePopMergeSQL(tenantID, metricID, periodLabel string, start, end time.Time, runID string) string {
	return fmt.Sprintf(`
    MERGE INTO iceberg.metrics_pop t
    USING (
      WITH monthly AS (
        SELECT
          '%s' AS tenant_id,
          '%s' AS metric_id,
          date_trunc('month', as_of_date)::date AS period_start,
          (date_trunc('month', as_of_date) + interval '1 month' - interval '1 day')::date AS period_end,
          '%s' AS period_label,
          count(*) AS record_count,
          sum(CAST(value AS decimal(38,10))) AS current_value
        FROM iceberg.metrics_atomic
        WHERE tenant_id = '%s'
          AND metric_id = '%s'
          AND as_of_date >= '%s'
          AND as_of_date <= '%s'
        GROUP BY 1,2,3,4,5
      ),
      lagged AS (
        SELECT m.*,
               lag(current_value) OVER (
                 PARTITION BY tenant_id, metric_id
                 ORDER BY period_start
               ) AS previous_value
        FROM monthly m
      )
      SELECT tenant_id, metric_id, period_start, period_end, period_label,
             record_count, current_value, previous_value,
             (current_value - previous_value) AS delta,
             CASE WHEN previous_value = 0 THEN NULL
                  ELSE ROUND(100 * (current_value - previous_value) / previous_value, 4) END AS percent_change,
             'success' AS computation_status,
             '%s' AS computation_id,
             now() AS last_updated,
             now() AS created_at
      FROM lagged
    ) s
    ON  t.tenant_id = s.tenant_id
    AND t.metric_id = s.metric_id
    AND t.period_label = s.period_label
    WHEN MATCHED THEN UPDATE SET
      record_count = s.record_count,
      current_value = s.current_value,
      previous_value = s.previous_value,
      delta = s.delta,
      percent_change = s.percent_change,
      computation_status = s.computation_status,
      last_updated = s.last_updated
    WHEN NOT MATCHED THEN INSERT *
  `, tenantID, metricID, periodLabel, tenantID, metricID,
		start.Format("2006-01-02"), end.Format("2006-01-02"), runID)
}

// ComputeAndMergeAnomalies orchestrates z-score anomaly detection via Trino
func ComputeAndMergeAnomalies(ctx context.Context, req ComputeRequest) error {
	if globalConfig == nil || globalConfig.DB == nil {
		return fmt.Errorf("dependencies not initialized")
	}

	anomalySQL := generateAnomalyMergeSQL(req.TenantID, req.MetricID,
		req.PeriodStart, req.PeriodEnd, req.RunID)

	// For now, log the SQL and simulate execution
	fmt.Printf("Executing Anomaly MERGE SQL:\n%s\n", anomalySQL)

	return nil
}

// generateAnomalyMergeSQL generates MERGE SQL for anomaly detection
func generateAnomalyMergeSQL(tenantID, metricID string, start, end time.Time, runID string) string {
	return fmt.Sprintf(`
    MERGE INTO iceberg.metrics_anomalies t
    USING (
      WITH windowed AS (
        SELECT
          tenant_id,
          metric_id,
          as_of_date::timestamp AS detected_at,
          value::decimal(38,10) AS x,
          avg(value::decimal(38,10)) OVER w AS mu,
          stddev_pop(value::decimal(38,10)) OVER w AS sigma
        FROM iceberg.metrics_atomic
        WHERE tenant_id = '%s'
          AND metric_id = '%s'
          AND as_of_date >= '%s'::date
          AND as_of_date <= '%s'::date
        WINDOW w AS (
          PARTITION BY tenant_id, metric_id
          ORDER BY as_of_date
          RANGE BETWEEN INTERVAL '90 days' PRECEDING AND CURRENT ROW
        )
      ),
      scored AS (
        SELECT *,
          CASE WHEN sigma = 0 THEN NULL
               ELSE (x - mu) / sigma END AS z_score
        FROM windowed
      )
      SELECT
        tenant_id,
        metric_id,
        'z_score' AS anomaly_type,
        detected_at,
        CASE WHEN abs(z_score) >= 2.5 THEN 'high'
             WHEN abs(z_score) >= 1.5 THEN 'medium'
             ELSE 'low' END AS severity,
        0.95 AS confidence,
        x AS actual_value,
        mu AS expected_value,
        (mu - 3 * sigma) AS expected_range_min,
        (mu + 3 * sigma) AS expected_range_max,
        '{\"threshold\": \"2.5\", \"window_days\": \"90\"}' AS detection_params,
        NULL AS computation_id,
        'open' AS status,
        now() AS created_at
      FROM scored
      WHERE z_score IS NOT NULL
        AND abs(z_score) >= 2.5
    ) s
    ON  t.tenant_id = s.tenant_id
    AND t.metric_id = s.metric_id
    AND t.anomaly_type = s.anomaly_type
    AND t.detected_at = s.detected_at
    WHEN MATCHED THEN UPDATE SET severity = s.severity, confidence = s.confidence
    WHEN NOT MATCHED THEN INSERT *
  `, tenantID, metricID, start.Format("2006-01-02"), end.Format("2006-01-02"))
}

// PublishCompletionEvent emits Kafka event for downstream systems
func PublishCompletionEvent(ctx context.Context, req ComputeRequest) error {
	if globalConfig == nil || globalConfig.KafkaWriter == nil {
		fmt.Printf("Warning: Kafka not configured, skipping event publication\n")
		return nil
	}

	// Build event payload
	event := map[string]interface{}{
		"event_id":     uuid.NewString(),
		"tenant_id":    req.TenantID,
		"metric_id":    req.MetricID,
		"calc_type":    req.CalcType,
		"period_label": req.PeriodLabel,
		"completed_at": time.Now().UTC(),
		"run_id":       req.RunID,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("Error marshaling event: %v\n", err)
		return err
	}

	msg := kafka.Message{
		Topic: "metrics.computations",
		Value: payload,
		Time:  time.Now(),
	}

	err = globalConfig.KafkaWriter.WriteMessages(ctx, msg)
	if err != nil {
		fmt.Printf("Error publishing to Kafka: %v (continuing anyway)\n", err)
		// Don't fail the workflow for event publication issues
		return nil
	}
	fmt.Printf("Published event to Kafka: %s\n", string(payload))

	return nil
}

// RefreshCubePartitions calls Cube.dev API to refresh specific partitions
func RefreshCubePartitions(ctx context.Context, req ComputeRequest) error {
	// TODO: Implement Cube.dev API call
	// For now, just log that refresh would happen
	fmt.Printf("Would refresh Cube partitions for tenant=%s, metric=%s\n",
		req.TenantID, req.MetricID)

	return nil
}
