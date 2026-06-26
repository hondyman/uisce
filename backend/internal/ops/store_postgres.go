package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PostgresStore implements the Store interface using PostgreSQL
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new Postgres store
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// ========== Alerts ==========

// ListAlerts returns all alerts, optionally filtered by enabled status
func (p *PostgresStore) ListAlerts(ctx context.Context, enabled *bool) ([]Alert, error) {
	var alerts []Alert
	query := `
		SELECT id, name, scope, metric, threshold, comparison, window_secs, enabled, created_at, updated_at
		FROM ops_alerts
	`
	args := []interface{}{}

	if enabled != nil {
		query += " WHERE enabled = $1"
		args = append(args, *enabled)
	}

	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var alert Alert
		if err := rows.Scan(&alert.ID, &alert.Name, &alert.Scope, &alert.Metric, &alert.Threshold,
			&alert.Comparison, &alert.WindowSecs, &alert.Enabled, &alert.CreatedAt, &alert.UpdatedAt); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	return alerts, rows.Err()
}

// GetAlert returns a single alert
func (p *PostgresStore) GetAlert(ctx context.Context, id uuid.UUID) (*Alert, error) {
	var alert Alert
	err := p.db.QueryRowContext(ctx,
		`SELECT id, name, scope, metric, threshold, comparison, window_secs, enabled, created_at, updated_at
		 FROM ops_alerts WHERE id = $1`,
		id).Scan(&alert.ID, &alert.Name, &alert.Scope, &alert.Metric, &alert.Threshold,
		&alert.Comparison, &alert.WindowSecs, &alert.Enabled, &alert.CreatedAt, &alert.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &alert, err
}

// CreateAlert creates a new alert
func (p *PostgresStore) CreateAlert(ctx context.Context, alert Alert) (*Alert, error) {
	alert.ID = uuid.New()
	alert.CreatedAt = time.Now().UTC()
	alert.UpdatedAt = time.Now().UTC()

	err := p.db.QueryRowContext(ctx,
		`INSERT INTO ops_alerts (id, name, scope, metric, threshold, comparison, window_secs, enabled, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, created_at, updated_at`,
		alert.ID, alert.Name, alert.Scope, alert.Metric, alert.Threshold, alert.Comparison, alert.WindowSecs, alert.Enabled, alert.CreatedAt, alert.UpdatedAt).
		Scan(&alert.ID, &alert.CreatedAt, &alert.UpdatedAt)

	return &alert, err
}

// UpdateAlert updates an existing alert
func (p *PostgresStore) UpdateAlert(ctx context.Context, id uuid.UUID, alert Alert) error {
	alert.UpdatedAt = time.Now().UTC()
	_, err := p.db.ExecContext(ctx,
		`UPDATE ops_alerts SET name=$1, scope=$2, metric=$3, threshold=$4, comparison=$5, window_secs=$6, enabled=$7, updated_at=$8
		 WHERE id=$9`,
		alert.Name, alert.Scope, alert.Metric, alert.Threshold, alert.Comparison, alert.WindowSecs, alert.Enabled, alert.UpdatedAt, id)
	return err
}

// DeleteAlert deletes an alert
func (p *PostgresStore) DeleteAlert(ctx context.Context, id uuid.UUID) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM ops_alerts WHERE id=$1", id)
	return err
}

// InsertAlertEvent records an alert event
func (p *PostgresStore) InsertAlertEvent(ctx context.Context, event AlertEvent) error {
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO ops_alert_events (id, alert_id, scope_id, endpoint, value, triggered_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		event.ID, event.AlertID, event.ScopeID, event.Endpoint, event.Value, event.TriggeredAt)
	return err
}

// GetAlertEvents returns recent events for an alert
func (p *PostgresStore) GetAlertEvents(ctx context.Context, alertID uuid.UUID, limit int) ([]AlertEvent, error) {
	var events []AlertEvent
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, alert_id, scope_id, endpoint, value, triggered_at
		 FROM ops_alert_events
		 WHERE alert_id = $1
		 ORDER BY triggered_at DESC LIMIT $2`,
		alertID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var event AlertEvent
		if err := rows.Scan(&event.ID, &event.AlertID, &event.ScopeID, &event.Endpoint, &event.Value, &event.TriggeredAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// ========== Error Fingerprints ==========

// GetOrCreateFingerprint gets or creates an error fingerprint
func (p *PostgresStore) GetOrCreateFingerprint(ctx context.Context, fingerprint, path string, statusCode int, sample string) (*ErrorFingerprint, error) {
	var fp ErrorFingerprint
	err := p.db.QueryRowContext(ctx,
		`INSERT INTO ops_error_fingerprints (id, fingerprint, path, status_code, sample_message, first_seen, last_seen, count)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 1)
		 ON CONFLICT (fingerprint) DO UPDATE
		 SET last_seen = EXCLUDED.last_seen, count = count + 1
		 RETURNING id, fingerprint, path, status_code, sample_message, first_seen, last_seen, count, created_at`,
		uuid.New(), fingerprint, path, statusCode, sample, time.Now().UTC(), time.Now().UTC()).
		Scan(&fp.ID, &fp.Fingerprint, &fp.Path, &fp.StatusCode, &fp.SampleMessage, &fp.FirstSeen, &fp.LastSeen, &fp.Count, &fp.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &fp, nil
}

// UpdateFingerprintCount updates the count for a fingerprint
func (p *PostgresStore) UpdateFingerprintCount(ctx context.Context, fingerprintID uuid.UUID, newCount int64) error {
	_, err := p.db.ExecContext(ctx,
		`UPDATE ops_error_fingerprints SET count=$1, last_seen=$2, updated_at=$3 WHERE id=$4`,
		newCount, time.Now().UTC(), time.Now().UTC(), fingerprintID)
	return err
}

// InsertErrorEvent records an error event
func (p *PostgresStore) InsertErrorEvent(ctx context.Context, event ErrorEvent) error {
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO ops_error_events (id, fingerprint_id, tenant_id, endpoint, status_code, message, request_id, occurred_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		event.ID, event.FingerprintID, event.TenantID, event.Endpoint, event.StatusCode, event.Message, event.RequestID, event.OccurredAt)
	return err
}

// ListFingerprints returns recent fingerprints
func (p *PostgresStore) ListFingerprints(ctx context.Context, limit int) ([]ErrorFingerprint, error) {
	var fingerprints []ErrorFingerprint
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, fingerprint, path, status_code, sample_message, first_seen, last_seen, count, created_at
		 FROM ops_error_fingerprints
		 ORDER BY last_seen DESC LIMIT $1`,
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var fp ErrorFingerprint
		if err := rows.Scan(&fp.ID, &fp.Fingerprint, &fp.Path, &fp.StatusCode, &fp.SampleMessage, &fp.FirstSeen, &fp.LastSeen, &fp.Count, &fp.CreatedAt); err != nil {
			return nil, err
		}
		fingerprints = append(fingerprints, fp)
	}

	return fingerprints, rows.Err()
}

// GetFingerprintEvents returns events for a fingerprint
func (p *PostgresStore) GetFingerprintEvents(ctx context.Context, fingerprintID uuid.UUID, limit int) ([]ErrorEvent, error) {
	var events []ErrorEvent
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, fingerprint_id, tenant_id, endpoint, status_code, message, request_id, occurred_at
		 FROM ops_error_events
		 WHERE fingerprint_id = $1
		 ORDER BY occurred_at DESC LIMIT $2`,
		fingerprintID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var event ErrorEvent
		if err := rows.Scan(&event.ID, &event.FingerprintID, &event.TenantID, &event.Endpoint, &event.StatusCode, &event.Message, &event.RequestID, &event.OccurredAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// ========== Health Scores ==========

// UpsertTenantHealth inserts or updates tenant health
func (p *PostgresStore) UpsertTenantHealth(ctx context.Context, health TenantHealth) error {
	componentsJSON, _ := json.Marshal(health.Components)

	_, err := p.db.ExecContext(ctx,
		`INSERT INTO ops_tenant_health_cache (id, tenant_id, health_score, components, computed_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (tenant_id) DO UPDATE
		 SET health_score = EXCLUDED.health_score, components = EXCLUDED.components, computed_at = EXCLUDED.computed_at, updated_at = EXCLUDED.updated_at`,
		uuid.New(), health.TenantID, health.Score, componentsJSON, health.ComputedAt, health.UpdatedAt)

	return err
}

// GetTenantHealth retrieves cached tenant health
func (p *PostgresStore) GetTenantHealth(ctx context.Context, tenantID uuid.UUID) (*TenantHealth, error) {
	var health TenantHealth
	var componentsJSON []byte

	err := p.db.QueryRowContext(ctx,
		`SELECT tenant_id, health_score, components, computed_at, updated_at
		 FROM ops_tenant_health_cache
		 WHERE tenant_id = $1`,
		tenantID).Scan(&health.TenantID, &health.Score, &componentsJSON, &health.ComputedAt, &health.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(componentsJSON, &health.Components); err != nil {
		return nil, err
	}

	return &health, nil
}

// GetTenantHealths retrieves multiple tenant health scores
func (p *PostgresStore) GetTenantHealths(ctx context.Context, limit int) ([]TenantHealth, error) {
	var healths []TenantHealth
	rows, err := p.db.QueryContext(ctx,
		`SELECT tenant_id, health_score, components, computed_at, updated_at
		 FROM ops_tenant_health_cache
		 ORDER BY health_score ASC LIMIT $1`,
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var health TenantHealth
		var componentsJSON []byte
		if err := rows.Scan(&health.TenantID, &health.Score, &componentsJSON, &health.ComputedAt, &health.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(componentsJSON, &health.Components); err != nil {
			return nil, err
		}
		healths = append(healths, health)
	}

	return healths, rows.Err()
}

// UpsertEndpointHealth inserts or updates endpoint health
func (p *PostgresStore) UpsertEndpointHealth(ctx context.Context, health EndpointHealth) error {
	componentsJSON, _ := json.Marshal(health.Components)

	_, err := p.db.ExecContext(ctx,
		`INSERT INTO ops_endpoint_health_cache (id, endpoint, health_score, error_rate, p95_ms, requests_1h, components, computed_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (endpoint) DO UPDATE
		 SET health_score = EXCLUDED.health_score, error_rate = EXCLUDED.error_rate, p95_ms = EXCLUDED.p95_ms, requests_1h = EXCLUDED.requests_1h, components = EXCLUDED.components, computed_at = EXCLUDED.computed_at, updated_at = EXCLUDED.updated_at`,
		uuid.New(), health.Endpoint, health.Score, health.ErrorRate, health.P95MS, health.Requests1H, componentsJSON, health.ComputedAt, health.UpdatedAt)

	return err
}

// GetEndpointHealth retrieves cached endpoint health
func (p *PostgresStore) GetEndpointHealth(ctx context.Context, endpoint string) (*EndpointHealth, error) {
	var health EndpointHealth
	var componentsJSON sql.NullString

	err := p.db.QueryRowContext(ctx,
		`SELECT endpoint, health_score, error_rate, p95_ms, requests_1h, components, computed_at, updated_at
		 FROM ops_endpoint_health_cache
		 WHERE endpoint = $1`,
		endpoint).Scan(&health.Endpoint, &health.Score, &health.ErrorRate, &health.P95MS, &health.Requests1H, &componentsJSON, &health.ComputedAt, &health.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if componentsJSON.Valid {
		if err := json.Unmarshal([]byte(componentsJSON.String), &health.Components); err != nil {
			return nil, err
		}
	}

	return &health, nil
}

// GetEndpointHealths retrieves multiple endpoint health scores
func (p *PostgresStore) GetEndpointHealths(ctx context.Context, limit int) ([]EndpointHealth, error) {
	var healths []EndpointHealth
	rows, err := p.db.QueryContext(ctx,
		`SELECT endpoint, health_score, error_rate, p95_ms, requests_1h, components, computed_at, updated_at
		 FROM ops_endpoint_health_cache
		 ORDER BY health_score ASC LIMIT $1`,
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var health EndpointHealth
		var componentsJSON sql.NullString
		if err := rows.Scan(&health.Endpoint, &health.Score, &health.ErrorRate, &health.P95MS, &health.Requests1H, &componentsJSON, &health.ComputedAt, &health.UpdatedAt); err != nil {
			return nil, err
		}
		if componentsJSON.Valid {
			if err := json.Unmarshal([]byte(componentsJSON.String), &health.Components); err != nil {
				return nil, err
			}
		}
		healths = append(healths, health)
	}

	return healths, rows.Err()
}

// ========== Heatmap Data ==========

// InsertHeatmapBucket inserts a latency heatmap bucket
func (p *PostgresStore) InsertHeatmapBucket(ctx context.Context, bucketTime time.Time, dimensionType, dimensionValue string, p50, p95, p99 int, requestCount int) error {
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO ops_latency_heatmap (id, bucket_time, dimension_type, dimension_value, p50_ms, p95_ms, p99_ms, request_count, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		uuid.New(), bucketTime, dimensionType, dimensionValue, p50, p95, p99, requestCount, time.Now().UTC())
	return err
}

// GetHeatmapData retrieves heatmap data for a specific dimension
func (p *PostgresStore) GetHeatmapData(ctx context.Context, dimensionType, dimensionValue string, bucketSize, window time.Duration) ([]HeatmapSeriesPoint, error) {
	var points []HeatmapSeriesPoint
	since := time.Now().UTC().Add(-window)

	rows, err := p.db.QueryContext(ctx,
		`SELECT bucket_time, p50_ms, p95_ms, p99_ms FROM ops_latency_heatmap
		 WHERE dimension_type = $1 AND dimension_value = $2 AND bucket_time >= $3
		 ORDER BY bucket_time ASC`,
		dimensionType, dimensionValue, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var point HeatmapSeriesPoint
		if err := rows.Scan(&point.Time, &point.Value, &point.P95MS, &point.P99MS); err != nil {
			return nil, err
		}
		points = append(points, point)
	}

	return points, rows.Err()
}

// GetHeatmapSeries retrieves multiple series for a dimension type
func (p *PostgresStore) GetHeatmapSeries(ctx context.Context, dimensionType string, limit int, bucketSize, window time.Duration) ([]HeatmapSeries, error) {
	var series []HeatmapSeries
	since := time.Now().UTC().Add(-window)

	// Get distinct dimension values
	rows, err := p.db.QueryContext(ctx,
		`SELECT DISTINCT dimension_value FROM ops_latency_heatmap
		 WHERE dimension_type = $1 AND bucket_time >= $2
		 LIMIT $3`,
		dimensionType, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dimensionValue string
		if err := rows.Scan(&dimensionValue); err != nil {
			return nil, err
		}

		points, err := p.GetHeatmapData(ctx, dimensionType, dimensionValue, bucketSize, window)
		if err != nil {
			return nil, err
		}

		series = append(series, HeatmapSeries{
			Key:    dimensionValue,
			Values: points,
		})
	}

	return series, nil
}

// ========== Metrics (stub implementations) ==========

// GetMetricValue retrieves a computed metric value
func (p *PostgresStore) GetMetricValue(ctx context.Context, metric, scope string, since time.Time) (float64, error) {
	query := `
		SELECT COALESCE(AVG(value), 0) as avg_value
		FROM public.metrics
		WHERE metric_type = $1 
		AND tags->>'scope' = $2
		AND metric_time >= $3
	`
	var value float64
	err := p.db.QueryRowContext(ctx, query, metric, scope, since).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return value, err
}

// GetTenantMetrics retrieves all metrics for a tenant
func (p *PostgresStore) GetTenantMetrics(ctx context.Context, tenantID uuid.UUID, since time.Time) (*TenantMetrics, error) {
	query := `
		SELECT
			COALESCE(AVG(CASE WHEN metric_type = 'availability' THEN value ELSE NULL END), 99) as availability,
			COALESCE(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY value) FILTER (WHERE metric_type = 'latency'), 100) as p50,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY value) FILTER (WHERE metric_type = 'latency'), 150) as p95,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY value) FILTER (WHERE metric_type = 'latency'), 200) as p99,
			COALESCE(SUM(CASE WHEN metric_type = 'requests' THEN value ELSE 0 END), 0) as requests,
			COALESCE(SUM(CASE WHEN metric_type = 'rate_limited' THEN value ELSE 0 END), 0) as rate_limited,
			COALESCE(AVG(CASE WHEN metric_type = 'error_rate' THEN value ELSE NULL END), 0) as error_rate
		FROM public.metrics
		WHERE tags->>'tenant_id' = $1
		AND metric_time >= $2
	`

	var availPct, p50, p95, p99, requests, rateLimited, errorRate float64

	err := p.db.QueryRowContext(ctx, query, tenantID.String(), since).Scan(
		&availPct, &p50, &p95, &p99, &requests, &rateLimited, &errorRate,
	)
	if err == sql.ErrNoRows || err != nil {
		// Return default healthy metrics if no data
		return &TenantMetrics{
			TenantID:        tenantID,
			AvailabilityPct: 99.0,
			P50:             100,
			P95:             150,
			P99:             200,
			Requests:        0,
			RateLimited:     0,
			ErrorRate:       0.0,
		}, nil
	}

	return &TenantMetrics{
		TenantID:        tenantID,
		AvailabilityPct: availPct,
		P50:             int(p50),
		P95:             int(p95),
		P99:             int(p99),
		Requests:        int64(requests),
		RateLimited:     int64(rateLimited),
		ErrorRate:       errorRate,
	}, nil
}

// GetEndpointMetrics retrieves all metrics for an endpoint
func (p *PostgresStore) GetEndpointMetrics(ctx context.Context, endpoint string, since time.Time) (*EndpointMetrics, error) {
	query := `
		SELECT
			COALESCE(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY value) FILTER (WHERE metric_type = 'latency'), 100) as p50,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY value) FILTER (WHERE metric_type = 'latency'), 150) as p95,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY value) FILTER (WHERE metric_type = 'latency'), 200) as p99,
			COALESCE(SUM(CASE WHEN metric_type = 'requests' THEN value ELSE 0 END), 0) as requests,
			COALESCE(AVG(CASE WHEN metric_type = 'error_rate' THEN value ELSE NULL END), 0) as error_rate
		FROM public.metrics
		WHERE tags->>'endpoint' = $1
		AND metric_time >= $2
	`

	var p50, p95, p99, requests, errorRate float64

	err := p.db.QueryRowContext(ctx, query, endpoint, since).Scan(
		&p50, &p95, &p99, &requests, &errorRate,
	)
	if err == sql.ErrNoRows || err != nil {
		// Return default healthy metrics if no data
		return &EndpointMetrics{
			Path:        endpoint,
			P50:         100,
			P95:         150,
			P99:         200,
			Requests:    0,
			ErrorRate:   0.0,
			StatusCodes: make(map[int]int64),
		}, nil
	}

	return &EndpointMetrics{
		Path:        endpoint,
		P50:         int(p50),
		P95:         int(p95),
		P99:         int(p99),
		Requests:    int64(requests),
		ErrorRate:   errorRate,
		StatusCodes: make(map[int]int64), // Could query separately for status code breakdown
	}, nil
}

// GetGlobalMetrics retrieves global metrics
func (p *PostgresStore) GetGlobalMetrics(ctx context.Context, since time.Time) (*TenantMetrics, error) {
	query := `
		SELECT
			COALESCE(AVG(CASE WHEN metric_type = 'availability' THEN value ELSE NULL END), 99) as availability,
			COALESCE(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY value) FILTER (WHERE metric_type = 'latency'), 100) as p50,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY value) FILTER (WHERE metric_type = 'latency'), 150) as p95,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY value) FILTER (WHERE metric_type = 'latency'), 200) as p99,
			COALESCE(SUM(CASE WHEN metric_type = 'requests' THEN value ELSE 0 END), 0) as requests,
			COALESCE(SUM(CASE WHEN metric_type = 'rate_limited' THEN value ELSE 0 END), 0) as rate_limited,
			COALESCE(AVG(CASE WHEN metric_type = 'error_rate' THEN value ELSE NULL END), 0) as error_rate
		FROM public.metrics
		WHERE metric_time >= $1
	`

	var availPct, p50, p95, p99, requests, rateLimited, errorRate float64

	err := p.db.QueryRowContext(ctx, query, since).Scan(
		&availPct, &p50, &p95, &p99, &requests, &rateLimited, &errorRate,
	)
	if err == sql.ErrNoRows || err != nil {
		// Return default healthy metrics if no data
		return &TenantMetrics{
			AvailabilityPct: 99.0,
			P50:             100,
			P95:             150,
			P99:             200,
			Requests:        0,
			RateLimited:     0,
			ErrorRate:       0.0,
		}, nil
	}

	return &TenantMetrics{
		AvailabilityPct: availPct,
		P50:             int(p50),
		P95:             int(p95),
		P99:             int(p99),
		Requests:        int64(requests),
		RateLimited:     int64(rateLimited),
		ErrorRate:       errorRate,
	}, nil
}

// ========== Audit Log (Phase 2.4c) ==========

// InsertAuditLog inserts an audit log entry into the database
func (p *PostgresStore) InsertAuditLog(ctx context.Context, auditLog *AuditLog) error {
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO ops_audit_log 
		 (id, incident_id, user_id, user_role, action_type, status, parameters, result, error_msg, executed_at, duration_ms, source_ip, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		auditLog.ID, auditLog.IncidentID, auditLog.UserID, auditLog.UserRole, auditLog.ActionType,
		auditLog.Status, auditLog.Parameters, json.RawMessage(jsonifyResult(auditLog.Result)),
		auditLog.ErrorMsg, auditLog.ExecutedAt, auditLog.DurationMs, auditLog.SourceIP, auditLog.CreatedAt)
	return err
}

// GetAuditLog retrieves a single audit log entry by ID
func (p *PostgresStore) GetAuditLog(ctx context.Context, id uuid.UUID) (*AuditLog, error) {
	var auditLog AuditLog
	var resultJSON []byte

	err := p.db.QueryRowContext(ctx,
		`SELECT id, incident_id, user_id, user_role, action_type, status, parameters, result, error_msg, executed_at, duration_ms, source_ip, created_at
		 FROM ops_audit_log WHERE id = $1`,
		id).Scan(
		&auditLog.ID, &auditLog.IncidentID, &auditLog.UserID, &auditLog.UserRole, &auditLog.ActionType,
		&auditLog.Status, &auditLog.Parameters, &resultJSON, &auditLog.ErrorMsg,
		&auditLog.ExecutedAt, &auditLog.DurationMs, &auditLog.SourceIP, &auditLog.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if resultJSON != nil {
		if err := json.Unmarshal(resultJSON, &auditLog.Result); err != nil {
			return nil, err
		}
	}

	return &auditLog, nil
}

// ListAuditLogs retrieves audit logs with optional filtering
func (p *PostgresStore) ListAuditLogs(ctx context.Context, filters AuditLogFilters, limit int, offset int) ([]AuditLog, error) {
	query := `
		SELECT id, incident_id, user_id, user_role, action_type, status, parameters, result, error_msg, executed_at, duration_ms, source_ip, created_at
		FROM ops_audit_log
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	// Apply filters
	if filters.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filters.UserID)
		argCount++
	}

	if filters.ActionType != nil {
		query += fmt.Sprintf(" AND action_type = $%d", argCount)
		args = append(args, *filters.ActionType)
		argCount++
	}

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filters.Status)
		argCount++
	}

	if filters.IncidentID != nil {
		query += fmt.Sprintf(" AND incident_id = $%d", argCount)
		args = append(args, *filters.IncidentID)
		argCount++
	}

	if filters.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filters.StartTime)
		argCount++
	}

	if filters.EndTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filters.EndTime)
		argCount++
	}

	// Add ordering and pagination
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var auditLogs []AuditLog
	for rows.Next() {
		var auditLog AuditLog
		var resultJSON []byte

		if err := rows.Scan(&auditLog.ID, &auditLog.IncidentID, &auditLog.UserID, &auditLog.UserRole, &auditLog.ActionType,
			&auditLog.Status, &auditLog.Parameters, &resultJSON, &auditLog.ErrorMsg,
			&auditLog.ExecutedAt, &auditLog.DurationMs, &auditLog.SourceIP, &auditLog.CreatedAt); err != nil {
			return nil, err
		}

		if resultJSON != nil {
			if err := json.Unmarshal(resultJSON, &auditLog.Result); err != nil {
				return nil, err
			}
		}

		auditLogs = append(auditLogs, auditLog)
	}

	return auditLogs, rows.Err()
}

// ListIncidentAuditLogs retrieves all audit log entries for a specific incident
func (p *PostgresStore) ListIncidentAuditLogs(ctx context.Context, incidentID uuid.UUID, limit int) ([]AuditLog, error) {
	query := `
		SELECT id, incident_id, user_id, user_role, action_type, status, parameters, result, error_msg, executed_at, duration_ms, source_ip, created_at
		FROM ops_audit_log
		WHERE incident_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, incidentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var auditLogs []AuditLog
	for rows.Next() {
		var auditLog AuditLog
		var resultJSON []byte

		if err := rows.Scan(&auditLog.ID, &auditLog.IncidentID, &auditLog.UserID, &auditLog.UserRole, &auditLog.ActionType,
			&auditLog.Status, &auditLog.Parameters, &resultJSON, &auditLog.ErrorMsg,
			&auditLog.ExecutedAt, &auditLog.DurationMs, &auditLog.SourceIP, &auditLog.CreatedAt); err != nil {
			return nil, err
		}

		if resultJSON != nil {
			if err := json.Unmarshal(resultJSON, &auditLog.Result); err != nil {
				return nil, err
			}
		}

		auditLogs = append(auditLogs, auditLog)
	}

	return auditLogs, rows.Err()
}

// Helper function to convert map[string]interface{} to JSON
func jsonifyResult(result map[string]interface{}) json.RawMessage {
	if result == nil {
		return json.RawMessage([]byte("null"))
	}
	data, _ := json.Marshal(result)
	return json.RawMessage(data)
}

// ========== Region Metadata (Phase 3.1) ==========

// GetRegionConfig retrieves a region configuration by code
func (p *PostgresStore) GetRegionConfig(ctx context.Context, regionCode string) (*RegionConfig, error) {
	var config RegionConfig
	err := p.db.QueryRowContext(ctx,
		`SELECT id, region_code, region_name, description, is_active, created_at, updated_at
		 FROM region_config WHERE region_code = $1`,
		regionCode).
		Scan(&config.ID, &config.RegionCode, &config.RegionName, &config.Description, &config.IsActive, &config.CreatedAt, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &config, err
}

// ListRegionConfigs retrieves all region configurations
func (p *PostgresStore) ListRegionConfigs(ctx context.Context, activeOnly bool) ([]RegionConfig, error) {
	query := `
		SELECT id, region_code, region_name, description, is_active, created_at, updated_at
		FROM region_config
	`
	args := []interface{}{}

	if activeOnly {
		query += " WHERE is_active = true"
	}

	query += " ORDER BY region_code"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []RegionConfig
	for rows.Next() {
		var config RegionConfig
		if err := rows.Scan(&config.ID, &config.RegionCode, &config.RegionName, &config.Description, &config.IsActive, &config.CreatedAt, &config.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return configs, rows.Err()
}

// InsertRegionRouting inserts or updates a region routing configuration
func (p *PostgresStore) InsertRegionRouting(ctx context.Context, routing *RegionRouting) error {
	if routing.ID == uuid.Nil {
		routing.ID = uuid.New()
	}
	routing.CreatedAt = time.Now().UTC()
	routing.UpdatedAt = time.Now().UTC()

	_, err := p.db.ExecContext(ctx,
		`INSERT INTO region_routing (id, tenant_id, region, starrocks_cluster, redpanda_broker, temporal_namespace, ops_worker_pool, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (tenant_id, region) DO UPDATE SET
		    starrocks_cluster = EXCLUDED.starrocks_cluster,
		    redpanda_broker = EXCLUDED.redpanda_broker,
		    temporal_namespace = EXCLUDED.temporal_namespace,
		    ops_worker_pool = EXCLUDED.ops_worker_pool,
		    updated_at = EXCLUDED.updated_at`,
		routing.ID, routing.TenantID, routing.Region, routing.StarRocksCluster, routing.RedpandaBroker, routing.TemporalNamespace, routing.OpsWorkerPool, routing.CreatedAt, routing.UpdatedAt)

	return err
}

// GetRegionRouting retrieves region routing for a tenant and region
func (p *PostgresStore) GetRegionRouting(ctx context.Context, tenantID uuid.UUID, region string) (*RegionRouting, error) {
	var routing RegionRouting
	err := p.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, region, starrocks_cluster, redpanda_broker, temporal_namespace, ops_worker_pool, created_at, updated_at
		 FROM region_routing WHERE tenant_id = $1 AND region = $2`,
		tenantID, region).
		Scan(&routing.ID, &routing.TenantID, &routing.Region, &routing.StarRocksCluster, &routing.RedpandaBroker, &routing.TemporalNamespace, &routing.OpsWorkerPool, &routing.CreatedAt, &routing.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &routing, err
}

// ListRegionRoutings retrieves all region routings for a tenant
func (p *PostgresStore) ListRegionRoutings(ctx context.Context, tenantID uuid.UUID) ([]RegionRouting, error) {
	query := `
		SELECT id, tenant_id, region, starrocks_cluster, redpanda_broker, temporal_namespace, ops_worker_pool, created_at, updated_at
		FROM region_routing
		WHERE tenant_id = $1
		ORDER BY region
	`

	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routings []RegionRouting
	for rows.Next() {
		var routing RegionRouting
		if err := rows.Scan(&routing.ID, &routing.TenantID, &routing.Region, &routing.StarRocksCluster, &routing.RedpandaBroker, &routing.TemporalNamespace, &routing.OpsWorkerPool, &routing.CreatedAt, &routing.UpdatedAt); err != nil {
			return nil, err
		}
		routings = append(routings, routing)
	}

	return routings, rows.Err()
}

// ============================================================================
// Phase 3.5: Regional Metrics & SLA Tracking
// ============================================================================

// UpsertRegionalMetrics inserts or updates regional performance metrics
func (p *PostgresStore) UpsertRegionalMetrics(ctx context.Context, metrics *RegionalMetrics) error {
	query := `
		INSERT INTO regional_metrics (id, region, error_rate, p50_latency_ms, p95_latency_ms, p99_latency_ms, 
			availability_pct, request_count, incident_count, components, computed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (region, computed_at) DO UPDATE SET
			error_rate = EXCLUDED.error_rate,
			p50_latency_ms = EXCLUDED.p50_latency_ms,
			p95_latency_ms = EXCLUDED.p95_latency_ms,
			p99_latency_ms = EXCLUDED.p99_latency_ms,
			availability_pct = EXCLUDED.availability_pct,
			request_count = EXCLUDED.request_count,
			incident_count = EXCLUDED.incident_count,
			components = EXCLUDED.components,
			updated_at = CURRENT_TIMESTAMP
	`

	componentsJSON := "{}"
	if metrics.Components != nil {
		data, _ := json.Marshal(metrics.Components)
		componentsJSON = string(data)
	}

	_, err := p.db.ExecContext(ctx, query,
		metrics.ID, metrics.Region, metrics.ErrorRate, metrics.P50Latency,
		metrics.P95Latency, metrics.P99Latency, metrics.Availability,
		metrics.RequestCount, metrics.IncidentCount, componentsJSON,
		metrics.ComputedAt, metrics.CreatedAt, metrics.UpdatedAt)

	return err
}

// GetRegionalMetrics retrieves the latest metrics for a region
func (p *PostgresStore) GetRegionalMetrics(ctx context.Context, region string) (*RegionalMetrics, error) {
	query := `
		SELECT id, region, error_rate, p50_latency_ms, p95_latency_ms, p99_latency_ms, 
			availability_pct, request_count, incident_count, components, computed_at, created_at, updated_at
		FROM regional_metrics
		WHERE region = $1
		ORDER BY computed_at DESC
		LIMIT 1
	`

	metrics := &RegionalMetrics{}
	var componentsJSON string

	err := p.db.QueryRowContext(ctx, query, region).Scan(
		&metrics.ID, &metrics.Region, &metrics.ErrorRate, &metrics.P50Latency,
		&metrics.P95Latency, &metrics.P99Latency, &metrics.Availability,
		&metrics.RequestCount, &metrics.IncidentCount, &componentsJSON,
		&metrics.ComputedAt, &metrics.CreatedAt, &metrics.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if componentsJSON != "" && componentsJSON != "{}" {
		json.Unmarshal([]byte(componentsJSON), &metrics.Components)
	}

	return metrics, nil
}

// ListRegionalMetrics retrieves recent metrics for all regions
func (p *PostgresStore) ListRegionalMetrics(ctx context.Context, limit int) ([]RegionalMetrics, error) {
	query := `
		SELECT DISTINCT ON (region) id, region, error_rate, p50_latency_ms, p95_latency_ms, p99_latency_ms,
			availability_pct, request_count, incident_count, components, computed_at, created_at, updated_at
		FROM regional_metrics
		ORDER BY region, computed_at DESC
		LIMIT $1
	`

	rows, err := p.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allMetrics []RegionalMetrics
	for rows.Next() {
		var metrics RegionalMetrics
		var componentsJSON string

		if err := rows.Scan(&metrics.ID, &metrics.Region, &metrics.ErrorRate, &metrics.P50Latency,
			&metrics.P95Latency, &metrics.P99Latency, &metrics.Availability,
			&metrics.RequestCount, &metrics.IncidentCount, &componentsJSON,
			&metrics.ComputedAt, &metrics.CreatedAt, &metrics.UpdatedAt); err != nil {
			return nil, err
		}

		if componentsJSON != "" && componentsJSON != "{}" {
			json.Unmarshal([]byte(componentsJSON), &metrics.Components)
		}

		allMetrics = append(allMetrics, metrics)
	}

	return allMetrics, rows.Err()
}

// UpsertRegionalHealth inserts or updates regional health scores
func (p *PostgresStore) UpsertRegionalHealth(ctx context.Context, health *RegionalHealth) error {
	query := `
		INSERT INTO regional_health (id, region, health_score, status, computed_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (region) DO UPDATE SET
			health_score = EXCLUDED.health_score,
			status = EXCLUDED.status,
			computed_at = EXCLUDED.computed_at,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := p.db.ExecContext(ctx, query,
		health.ID, health.Region, health.Score, health.Status, health.ComputedAt, health.UpdatedAt)

	return err
}

// GetRegionalHealth retrieves current health for a region
func (p *PostgresStore) GetRegionalHealth(ctx context.Context, region string) (*RegionalHealth, error) {
	query := `
		SELECT id, region, health_score, status, computed_at, updated_at
		FROM regional_health
		WHERE region = $1
	`

	health := &RegionalHealth{}
	err := p.db.QueryRowContext(ctx, query, region).Scan(
		&health.ID, &health.Region, &health.Score, &health.Status, &health.ComputedAt, &health.UpdatedAt)

	return health, err
}

// ListRegionalHealth retrieves health scores for all regions
func (p *PostgresStore) ListRegionalHealth(ctx context.Context, limit int) ([]RegionalHealth, error) {
	query := `
		SELECT id, region, health_score, status, computed_at, updated_at
		FROM regional_health
		ORDER BY updated_at DESC
		LIMIT $1
	`

	rows, err := p.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allHealth []RegionalHealth
	for rows.Next() {
		var health RegionalHealth
		if err := rows.Scan(&health.ID, &health.Region, &health.Score, &health.Status, &health.ComputedAt, &health.UpdatedAt); err != nil {
			return nil, err
		}
		allHealth = append(allHealth, health)
	}

	return allHealth, rows.Err()
}

// UpsertRegionalSLA inserts or updates regional SLA definitions
func (p *PostgresStore) UpsertRegionalSLA(ctx context.Context, sla *RegionalSLA) error {
	query := `
		INSERT INTO regional_sla (id, region, availability_sla_pct, p95_latency_sla_ms, error_rate_sla_pct, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (region) DO UPDATE SET
			availability_sla_pct = EXCLUDED.availability_sla_pct,
			p95_latency_sla_ms = EXCLUDED.p95_latency_sla_ms,
			error_rate_sla_pct = EXCLUDED.error_rate_sla_pct,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := p.db.ExecContext(ctx, query,
		sla.ID, sla.Region, sla.AvailabilitySLA, sla.P95LatencySLA, sla.ErrorRateSLA, sla.CreatedAt, sla.UpdatedAt)

	return err
}

// GetRegionalSLA retrieves the SLA for a region
func (p *PostgresStore) GetRegionalSLA(ctx context.Context, region string) (*RegionalSLA, error) {
	query := `
		SELECT id, region, availability_sla_pct, p95_latency_sla_ms, error_rate_sla_pct, created_at, updated_at
		FROM regional_sla
		WHERE region = $1
	`

	sla := &RegionalSLA{}
	err := p.db.QueryRowContext(ctx, query, region).Scan(
		&sla.ID, &sla.Region, &sla.AvailabilitySLA, &sla.P95LatencySLA, &sla.ErrorRateSLA, &sla.CreatedAt, &sla.UpdatedAt)

	return sla, err
}

// ListRegionalSLAs retrieves all regional SLA definitions
func (p *PostgresStore) ListRegionalSLAs(ctx context.Context, limit int) ([]RegionalSLA, error) {
	query := `
		SELECT id, region, availability_sla_pct, p95_latency_sla_ms, error_rate_sla_pct, created_at, updated_at
		FROM regional_sla
		ORDER BY region
		LIMIT $1
	`

	rows, err := p.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allSLAs []RegionalSLA
	for rows.Next() {
		var sla RegionalSLA
		if err := rows.Scan(&sla.ID, &sla.Region, &sla.AvailabilitySLA, &sla.P95LatencySLA, &sla.ErrorRateSLA, &sla.CreatedAt, &sla.UpdatedAt); err != nil {
			return nil, err
		}
		allSLAs = append(allSLAs, sla)
	}

	return allSLAs, rows.Err()
}

// InsertRegionalSLAStatus records SLA compliance check results
func (p *PostgresStore) InsertRegionalSLAStatus(ctx context.Context, status *RegionalSLAStatus) error {
	query := `
		INSERT INTO regional_sla_status (id, region, sla_id, availability_met, latency_met, error_rate_met, compliance_pct, checked_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := p.db.ExecContext(ctx, query,
		status.ID, status.Region, status.SLAID, status.AvailabilityMet, status.LatencyMet,
		status.ErrorRateMet, status.CompliancePct, status.CheckedAt, status.CreatedAt)

	return err
}

// GetRegionalSLAStatus retrieves the latest SLA status for a region
func (p *PostgresStore) GetRegionalSLAStatus(ctx context.Context, region string) (*RegionalSLAStatus, error) {
	query := `
		SELECT id, region, sla_id, availability_met, latency_met, error_rate_met, compliance_pct, checked_at, created_at
		FROM regional_sla_status
		WHERE region = $1
		ORDER BY checked_at DESC
		LIMIT 1
	`

	status := &RegionalSLAStatus{}
	err := p.db.QueryRowContext(ctx, query, region).Scan(
		&status.ID, &status.Region, &status.SLAID, &status.AvailabilityMet, &status.LatencyMet,
		&status.ErrorRateMet, &status.CompliancePct, &status.CheckedAt, &status.CreatedAt)

	return status, err
}

// ListRegionalSLAStatuses retrieves historical SLA compliance for a region
func (p *PostgresStore) ListRegionalSLAStatuses(ctx context.Context, region string, limit int) ([]RegionalSLAStatus, error) {
	query := `
		SELECT id, region, sla_id, availability_met, latency_met, error_rate_met, compliance_pct, checked_at, created_at
		FROM regional_sla_status
		WHERE region = $1
		ORDER BY checked_at DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, region, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allStatuses []RegionalSLAStatus
	for rows.Next() {
		var status RegionalSLAStatus
		if err := rows.Scan(&status.ID, &status.Region, &status.SLAID, &status.AvailabilityMet, &status.LatencyMet,
			&status.ErrorRateMet, &status.CompliancePct, &status.CheckedAt, &status.CreatedAt); err != nil {
			return nil, err
		}
		allStatuses = append(allStatuses, status)
	}

	return allStatuses, rows.Err()
}

// ========== Phase 3.9: Region-Aware API Layer ==========

// ListLatestRegionalSLAStatuses returns the latest SLA status for each region
func (p *PostgresStore) ListLatestRegionalSLAStatuses(ctx context.Context) ([]RegionalSLAStatus, error) {
	query := `
		SELECT DISTINCT ON (region) id, region, sla_id, availability_met, latency_met, error_rate_met, compliance_pct, checked_at, created_at
		FROM regional_sla_status
		ORDER BY region, checked_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []RegionalSLAStatus
	for rows.Next() {
		var status RegionalSLAStatus
		if err := rows.Scan(&status.ID, &status.Region, &status.SLAID, &status.AvailabilityMet, &status.LatencyMet,
			&status.ErrorRateMet, &status.CompliancePct, &status.CheckedAt, &status.CreatedAt); err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}

	return statuses, rows.Err()
}

// ListRegionalIncidentCounts returns incident count per region in a time window
func (p *PostgresStore) ListRegionalIncidentCounts(ctx context.Context, since, until time.Time) ([]RegionalIncidentCount, error) {
	query := `
		SELECT region, COUNT(*) as count
		FROM ops_incidents
		WHERE created_at >= $1 AND created_at <= $2 AND region IS NOT NULL
		GROUP BY region
		ORDER BY count DESC
	`

	rows, err := p.db.QueryContext(ctx, query, since, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []RegionalIncidentCount
	for rows.Next() {
		var c RegionalIncidentCount
		if err := rows.Scan(&c.Region, &c.Count); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}

	return counts, rows.Err()
}

// ListOpsEventsByRegion returns ops events for a specific region
func (p *PostgresStore) ListOpsEventsByRegion(ctx context.Context, region string, limit int) ([]Event, error) {
	query := `
		SELECT id, incident_id, event_type, scope, tenant_id, endpoint_path, region,
		       fingerprint_id, alert_id, severity, title, details, occurred_at, created_at
		FROM ops_events
		WHERE region = $1
		ORDER BY occurred_at DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, region, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(
			&e.ID, &e.IncidentID, &e.EventType, &e.Scope, &e.TenantID, &e.EndpointPath, &e.Region,
			&e.FingerprintID, &e.AlertID, &e.Severity, &e.Title, &e.Details, &e.OccurredAt, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

// ListAuditLogsByRegion returns audit logs for a specific region
func (p *PostgresStore) ListAuditLogsByRegion(ctx context.Context, region string, limit int) ([]AuditLog, error) {
	query := `
		SELECT id, user_id, user_role, action_type, status, parameters, result, error_msg,
		       incident_id, executed_at, source_ip, created_at
		FROM ops_audit_log
		WHERE region = $1
		ORDER BY executed_at DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, region, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		var resultJSON []byte
		if err := rows.Scan(
			&log.ID, &log.UserID, &log.UserRole, &log.ActionType, &log.Status, &log.Parameters, &resultJSON, &log.ErrorMsg,
			&log.IncidentID, &log.ExecutedAt, &log.SourceIP, &log.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(resultJSON) > 0 {
			if err := json.Unmarshal(resultJSON, &log.Result); err != nil {
				return nil, err
			}
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// ========== Phase 3.10: Failover Policies & Automated Regional Failover ==========

// InsertFailoverPolicy inserts a new failover policy
func (p *PostgresStore) InsertFailoverPolicy(ctx context.Context, policy *FailoverPolicy) error {
	query := `
		INSERT INTO failover_policies (id, tenant_id, name, source_region, target_regions, trigger_health_score, 
		                                trigger_error_rate, trigger_latency_ms, is_automatic, cooldown_minutes, 
		                                priority, is_enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	policy.ID = uuid.New()
	policy.CreatedAt = time.Now().UTC()
	policy.UpdatedAt = time.Now().UTC()

	_, err := p.db.ExecContext(ctx, query,
		policy.ID, policy.TenantID, policy.Name, policy.SourceRegion, policy.TargetRegions,
		policy.TriggerHealthScore, policy.TriggerErrorRate, policy.TriggerLatency,
		policy.IsAutomatic, policy.CooldownMinutes, policy.Priority, policy.IsEnabled,
		policy.CreatedAt, policy.UpdatedAt,
	)
	return err
}

// GetFailoverPolicy retrieves a failover policy by ID
func (p *PostgresStore) GetFailoverPolicy(ctx context.Context, id uuid.UUID) (*FailoverPolicy, error) {
	query := `
		SELECT id, tenant_id, name, source_region, target_regions, trigger_health_score, 
		       trigger_error_rate, trigger_latency_ms, is_automatic, cooldown_minutes, 
		       priority, is_enabled, created_at, updated_at
		FROM failover_policies
		WHERE id = $1
	`

	var policy FailoverPolicy
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&policy.ID, &policy.TenantID, &policy.Name, &policy.SourceRegion, &policy.TargetRegions,
		&policy.TriggerHealthScore, &policy.TriggerErrorRate, &policy.TriggerLatency,
		&policy.IsAutomatic, &policy.CooldownMinutes, &policy.Priority, &policy.IsEnabled,
		&policy.CreatedAt, &policy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &policy, nil
}

// ListFailoverPolicies retrieves all failover policies for a tenant
func (p *PostgresStore) ListFailoverPolicies(ctx context.Context, tenantID uuid.UUID) ([]FailoverPolicy, error) {
	query := `
		SELECT id, tenant_id, name, source_region, target_regions, trigger_health_score, 
		       trigger_error_rate, trigger_latency_ms, is_automatic, cooldown_minutes, 
		       priority, is_enabled, created_at, updated_at
		FROM failover_policies
		WHERE tenant_id = $1 AND is_enabled = true
		ORDER BY priority ASC, created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []FailoverPolicy
	for rows.Next() {
		var policy FailoverPolicy
		if err := rows.Scan(
			&policy.ID, &policy.TenantID, &policy.Name, &policy.SourceRegion, &policy.TargetRegions,
			&policy.TriggerHealthScore, &policy.TriggerErrorRate, &policy.TriggerLatency,
			&policy.IsAutomatic, &policy.CooldownMinutes, &policy.Priority, &policy.IsEnabled,
			&policy.CreatedAt, &policy.UpdatedAt,
		); err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}

	return policies, rows.Err()
}

// UpdateFailoverPolicy updates an existing failover policy
func (p *PostgresStore) UpdateFailoverPolicy(ctx context.Context, id uuid.UUID, policy *FailoverPolicy) error {
	query := `
		UPDATE failover_policies
		SET name = $1, source_region = $2, target_regions = $3, trigger_health_score = $4,
		    trigger_error_rate = $5, trigger_latency_ms = $6, is_automatic = $7, cooldown_minutes = $8,
		    priority = $9, is_enabled = $10, updated_at = $11
		WHERE id = $12
	`

	policy.UpdatedAt = time.Now().UTC()
	_, err := p.db.ExecContext(ctx, query,
		policy.Name, policy.SourceRegion, policy.TargetRegions, policy.TriggerHealthScore,
		policy.TriggerErrorRate, policy.TriggerLatency, policy.IsAutomatic, policy.CooldownMinutes,
		policy.Priority, policy.IsEnabled, policy.UpdatedAt, id,
	)
	return err
}

// DeleteFailoverPolicy soft-deletes a failover policy by setting is_enabled to false
func (p *PostgresStore) DeleteFailoverPolicy(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE failover_policies SET is_enabled = false, updated_at = $1 WHERE id = $2`
	_, err := p.db.ExecContext(ctx, query, time.Now().UTC(), id)
	return err
}

// InsertFailoverEvent inserts a new failover event
func (p *PostgresStore) InsertFailoverEvent(ctx context.Context, event *FailoverEvent) error {
	query := `
		INSERT INTO failover_events (id, incident_id, policy_id, tenant_id, source_region, target_region, 
		                              trigger_reason, trigger_value, status, rollback_needed, error_msg, 
		                              triggered_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	event.ID = uuid.New()
	event.CreatedAt = time.Now().UTC()
	event.UpdatedAt = time.Now().UTC()

	_, err := p.db.ExecContext(ctx, query,
		event.ID, event.IncidentID, event.PolicyID, event.TenantID, event.SourceRegion, event.TargetRegion,
		event.TriggerReason, event.TriggerValue, event.Status, event.RollbackNeeded, event.ErrorMsg,
		event.TriggeredAt, event.CreatedAt, event.UpdatedAt,
	)
	return err
}

// UpdateFailoverEvent updates the status of a failover event
func (p *PostgresStore) UpdateFailoverEvent(ctx context.Context, id uuid.UUID, status string, errorMsg *string, completedAt *time.Time) error {
	query := `
		UPDATE failover_events
		SET status = $1, error_msg = $2, completed_at = $3, updated_at = $4
		WHERE id = $5
	`

	_, err := p.db.ExecContext(ctx, query, status, errorMsg, completedAt, time.Now().UTC(), id)
	return err
}

// ListFailoverEvents retrieves failover events for a policy
func (p *PostgresStore) ListFailoverEvents(ctx context.Context, policyID uuid.UUID, limit int) ([]FailoverEvent, error) {
	query := `
		SELECT id, incident_id, policy_id, tenant_id, source_region, target_region, trigger_reason, 
		       trigger_value, status, rollback_needed, error_msg, triggered_at, completed_at, created_at, updated_at
		FROM failover_events
		WHERE policy_id = $1
		ORDER BY triggered_at DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, policyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []FailoverEvent
	for rows.Next() {
		var event FailoverEvent
		if err := rows.Scan(
			&event.ID, &event.IncidentID, &event.PolicyID, &event.TenantID, &event.SourceRegion, &event.TargetRegion,
			&event.TriggerReason, &event.TriggerValue, &event.Status, &event.RollbackNeeded, &event.ErrorMsg,
			&event.TriggeredAt, &event.CompletedAt, &event.CreatedAt, &event.UpdatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// ListIncidentFailoverEvents retrieves failover events for an incident
func (p *PostgresStore) ListIncidentFailoverEvents(ctx context.Context, incidentID uuid.UUID) ([]FailoverEvent, error) {
	query := `
		SELECT id, incident_id, policy_id, tenant_id, source_region, target_region, trigger_reason, 
		       trigger_value, status, rollback_needed, error_msg, triggered_at, completed_at, created_at, updated_at
		FROM failover_events
		WHERE incident_id = $1
		ORDER BY triggered_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []FailoverEvent
	for rows.Next() {
		var event FailoverEvent
		if err := rows.Scan(
			&event.ID, &event.IncidentID, &event.PolicyID, &event.TenantID, &event.SourceRegion, &event.TargetRegion,
			&event.TriggerReason, &event.TriggerValue, &event.Status, &event.RollbackNeeded, &event.ErrorMsg,
			&event.TriggeredAt, &event.CompletedAt, &event.CreatedAt, &event.UpdatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// UpsertFailoverMetrics inserts or updates failover metrics
func (p *PostgresStore) UpsertFailoverMetrics(ctx context.Context, metrics *FailoverMetrics) error {
	query := `
		INSERT INTO failover_metrics (id, policy_id, total_failovers, successful_count, failed_count, 
		                               avg_duration_ms, last_failover_at, success_rate_pct, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (policy_id) DO UPDATE SET
			total_failovers = $3,
			successful_count = $4,
			failed_count = $5,
			avg_duration_ms = $6,
			last_failover_at = $7,
			success_rate_pct = $8,
			updated_at = $9
	`

	if metrics.ID == uuid.Nil {
		metrics.ID = uuid.New()
	}
	metrics.UpdatedAt = time.Now().UTC()

	_, err := p.db.ExecContext(ctx, query,
		metrics.ID, metrics.PolicyID, metrics.TotalFailovers, metrics.SuccessfulCount, metrics.FailedCount,
		metrics.AvgDurationMs, metrics.LastFailoverAt, metrics.SuccessRatePct, metrics.UpdatedAt,
	)
	return err
}

// GetFailoverMetrics retrieves failover metrics for a policy
func (p *PostgresStore) GetFailoverMetrics(ctx context.Context, policyID uuid.UUID) (*FailoverMetrics, error) {
	query := `
		SELECT id, policy_id, total_failovers, successful_count, failed_count, avg_duration_ms, 
		       last_failover_at, success_rate_pct, updated_at
		FROM failover_metrics
		WHERE policy_id = $1
	`

	var metrics FailoverMetrics
	err := p.db.QueryRowContext(ctx, query, policyID).Scan(
		&metrics.ID, &metrics.PolicyID, &metrics.TotalFailovers, &metrics.SuccessfulCount, &metrics.FailedCount,
		&metrics.AvgDurationMs, &metrics.LastFailoverAt, &metrics.SuccessRatePct, &metrics.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}

// ========== Phase 3.11: Failover Chain Orchestration ==========

// InsertFailoverChain inserts a new failover chain
func (p *PostgresStore) InsertFailoverChain(ctx context.Context, chain *FailoverChain) error {
	query := `
		INSERT INTO failover_chains (id, tenant_id, name, source_region, chain_targets, 
		                             trigger_health_score, trigger_error_rate, trigger_latency_ms,
		                             max_chain_depth, cooldown_minutes, priority, is_enabled, 
		                             created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	chain.ID = uuid.New()
	chain.CreatedAt = time.Now().UTC()
	chain.UpdatedAt = time.Now().UTC()

	if chain.MaxChainDepth == 0 {
		chain.MaxChainDepth = 3 // Default max depth
	}

	_, err := p.db.ExecContext(ctx, query,
		chain.ID, chain.TenantID, chain.Name, chain.SourceRegion, chain.ChainTargets,
		chain.TriggerHealthScore, chain.TriggerErrorRate, chain.TriggerLatency,
		chain.MaxChainDepth, chain.CooldownMinutes, chain.Priority, chain.IsEnabled,
		chain.CreatedAt, chain.UpdatedAt,
	)
	return err
}

// GetFailoverChain retrieves a failover chain by ID
func (p *PostgresStore) GetFailoverChain(ctx context.Context, id uuid.UUID) (*FailoverChain, error) {
	query := `
		SELECT id, tenant_id, name, source_region, chain_targets, trigger_health_score,
		       trigger_error_rate, trigger_latency_ms, max_chain_depth, cooldown_minutes,
		       priority, is_enabled, created_at, updated_at
		FROM failover_chains
		WHERE id = $1
	`

	var chain FailoverChain
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&chain.ID, &chain.TenantID, &chain.Name, &chain.SourceRegion, &chain.ChainTargets,
		&chain.TriggerHealthScore, &chain.TriggerErrorRate, &chain.TriggerLatency,
		&chain.MaxChainDepth, &chain.CooldownMinutes, &chain.Priority, &chain.IsEnabled,
		&chain.CreatedAt, &chain.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &chain, nil
}

// ListFailoverChains retrieves all failover chains for a tenant
func (p *PostgresStore) ListFailoverChains(ctx context.Context, tenantID uuid.UUID) ([]FailoverChain, error) {
	query := `
		SELECT id, tenant_id, name, source_region, chain_targets, trigger_health_score,
		       trigger_error_rate, trigger_latency_ms, max_chain_depth, cooldown_minutes,
		       priority, is_enabled, created_at, updated_at
		FROM failover_chains
		WHERE tenant_id = $1 AND is_enabled = true
		ORDER BY priority ASC, created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []FailoverChain
	for rows.Next() {
		var chain FailoverChain
		if err := rows.Scan(
			&chain.ID, &chain.TenantID, &chain.Name, &chain.SourceRegion, &chain.ChainTargets,
			&chain.TriggerHealthScore, &chain.TriggerErrorRate, &chain.TriggerLatency,
			&chain.MaxChainDepth, &chain.CooldownMinutes, &chain.Priority, &chain.IsEnabled,
			&chain.CreatedAt, &chain.UpdatedAt,
		); err != nil {
			return nil, err
		}
		chains = append(chains, chain)
	}

	return chains, rows.Err()
}

// UpdateFailoverChain updates an existing failover chain
func (p *PostgresStore) UpdateFailoverChain(ctx context.Context, id uuid.UUID, chain *FailoverChain) error {
	query := `
		UPDATE failover_chains
		SET name = $1, source_region = $2, chain_targets = $3, trigger_health_score = $4,
		    trigger_error_rate = $5, trigger_latency_ms = $6, max_chain_depth = $7,
		    cooldown_minutes = $8, priority = $9, is_enabled = $10, updated_at = $11
		WHERE id = $12
	`

	chain.UpdatedAt = time.Now().UTC()
	_, err := p.db.ExecContext(ctx, query,
		chain.Name, chain.SourceRegion, chain.ChainTargets, chain.TriggerHealthScore,
		chain.TriggerErrorRate, chain.TriggerLatency, chain.MaxChainDepth,
		chain.CooldownMinutes, chain.Priority, chain.IsEnabled, chain.UpdatedAt, id,
	)
	return err
}

// DeleteFailoverChain soft-deletes a failover chain
func (p *PostgresStore) DeleteFailoverChain(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE failover_chains SET is_enabled = false, updated_at = $1 WHERE id = $2`
	_, err := p.db.ExecContext(ctx, query, time.Now().UTC(), id)
	return err
}

// InsertFailoverChainExecution inserts a new chain execution record
func (p *PostgresStore) InsertFailoverChainExecution(ctx context.Context, execution *FailoverChainExecution) error {
	query := `
		INSERT INTO failover_chain_executions (id, chain_id, incident_id, tenant_id, source_region,
		                                        current_step, current_target, previous_target, status,
		                                        steps_executed, failure_reasons, triggered_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	execution.ID = uuid.New()
	execution.CreatedAt = time.Now().UTC()
	execution.UpdatedAt = time.Now().UTC()

	stepsJSON, _ := json.Marshal(execution.StepsExecuted)
	reasonsJSON, _ := json.Marshal(execution.FailureReasons)

	_, err := p.db.ExecContext(ctx, query,
		execution.ID, execution.ChainID, execution.IncidentID, execution.TenantID, execution.SourceRegion,
		execution.CurrentStep, execution.CurrentTarget, execution.PreviousTarget, execution.Status,
		stepsJSON, reasonsJSON, execution.TriggeredAt, execution.CreatedAt, execution.UpdatedAt,
	)
	return err
}

// UpdateFailoverChainExecution updates chain execution status
func (p *PostgresStore) UpdateFailoverChainExecution(ctx context.Context, id uuid.UUID, status string, stepsExecuted []string, failureReasons []string, completedAt *time.Time) error {
	query := `
		UPDATE failover_chain_executions
		SET status = $1, steps_executed = $2, failure_reasons = $3, completed_at = $4, updated_at = $5
		WHERE id = $6
	`

	stepsJSON, _ := json.Marshal(stepsExecuted)
	reasonsJSON, _ := json.Marshal(failureReasons)

	_, err := p.db.ExecContext(ctx, query, status, stepsJSON, reasonsJSON, completedAt, time.Now().UTC(), id)
	return err
}

// ListFailoverChainExecutions retrieves executions for a chain
func (p *PostgresStore) ListFailoverChainExecutions(ctx context.Context, chainID uuid.UUID, limit int) ([]FailoverChainExecution, error) {
	query := `
		SELECT id, chain_id, incident_id, tenant_id, source_region, current_step, current_target,
		       previous_target, status, steps_executed, failure_reasons, triggered_at, completed_at,
		       created_at, updated_at
		FROM failover_chain_executions
		WHERE chain_id = $1
		ORDER BY triggered_at DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, chainID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []FailoverChainExecution
	for rows.Next() {
		var execution FailoverChainExecution
		var stepsJSON, reasonsJSON []byte

		if err := rows.Scan(
			&execution.ID, &execution.ChainID, &execution.IncidentID, &execution.TenantID, &execution.SourceRegion,
			&execution.CurrentStep, &execution.CurrentTarget, &execution.PreviousTarget, &execution.Status,
			&stepsJSON, &reasonsJSON, &execution.TriggeredAt, &execution.CompletedAt,
			&execution.CreatedAt, &execution.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if len(stepsJSON) > 0 {
			_ = json.Unmarshal(stepsJSON, &execution.StepsExecuted)
		}
		if len(reasonsJSON) > 0 {
			_ = json.Unmarshal(reasonsJSON, &execution.FailureReasons)
		}

		executions = append(executions, execution)
	}

	return executions, rows.Err()
}

// ListIncidentChainExecutions retrieves chain executions for an incident
func (p *PostgresStore) ListIncidentChainExecutions(ctx context.Context, incidentID uuid.UUID) ([]FailoverChainExecution, error) {
	query := `
		SELECT id, chain_id, incident_id, tenant_id, source_region, current_step, current_target,
		       previous_target, status, steps_executed, failure_reasons, triggered_at, completed_at,
		       created_at, updated_at
		FROM failover_chain_executions
		WHERE incident_id = $1
		ORDER BY triggered_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []FailoverChainExecution
	for rows.Next() {
		var execution FailoverChainExecution
		var stepsJSON, reasonsJSON []byte

		if err := rows.Scan(
			&execution.ID, &execution.ChainID, &execution.IncidentID, &execution.TenantID, &execution.SourceRegion,
			&execution.CurrentStep, &execution.CurrentTarget, &execution.PreviousTarget, &execution.Status,
			&stepsJSON, &reasonsJSON, &execution.TriggeredAt, &execution.CompletedAt,
			&execution.CreatedAt, &execution.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if len(stepsJSON) > 0 {
			_ = json.Unmarshal(stepsJSON, &execution.StepsExecuted)
		}
		if len(reasonsJSON) > 0 {
			_ = json.Unmarshal(reasonsJSON, &execution.FailureReasons)
		}

		executions = append(executions, execution)
	}

	return executions, rows.Err()
}

// UpsertFailoverChainMetrics inserts or updates failover chain metrics
func (p *PostgresStore) UpsertFailoverChainMetrics(ctx context.Context, metrics *FailoverChainMetrics) error {
	query := `
		INSERT INTO failover_chain_metrics (id, chain_id, total_executions, successful_count, 
		                                     partial_success_count, failed_count, avg_steps_needed,
		                                     avg_duration_ms, last_execution_at, success_rate_pct, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (chain_id) DO UPDATE SET
			total_executions = $3,
			successful_count = $4,
			partial_success_count = $5,
			failed_count = $6,
			avg_steps_needed = $7,
			avg_duration_ms = $8,
			last_execution_at = $9,
			success_rate_pct = $10,
			updated_at = $11
	`

	if metrics.ID == uuid.Nil {
		metrics.ID = uuid.New()
	}
	metrics.UpdatedAt = time.Now().UTC()

	_, err := p.db.ExecContext(ctx, query,
		metrics.ID, metrics.ChainID, metrics.TotalExecutions, metrics.SuccessfulCount,
		metrics.PartialSuccessCount, metrics.FailedCount, metrics.AvgStepsNeeded,
		metrics.AvgDurationMs, metrics.LastExecutionAt, metrics.SuccessRatePct, metrics.UpdatedAt,
	)
	return err
}

// GetFailoverChainMetrics retrieves metrics for a chain
func (p *PostgresStore) GetFailoverChainMetrics(ctx context.Context, chainID uuid.UUID) (*FailoverChainMetrics, error) {
	query := `
		SELECT id, chain_id, total_executions, successful_count, partial_success_count, failed_count,
		       avg_steps_needed, avg_duration_ms, last_execution_at, success_rate_pct, updated_at
		FROM failover_chain_metrics
		WHERE chain_id = $1
	`

	var metrics FailoverChainMetrics
	err := p.db.QueryRowContext(ctx, query, chainID).Scan(
		&metrics.ID, &metrics.ChainID, &metrics.TotalExecutions, &metrics.SuccessfulCount,
		&metrics.PartialSuccessCount, &metrics.FailedCount, &metrics.AvgStepsNeeded,
		&metrics.AvgDurationMs, &metrics.LastExecutionAt, &metrics.SuccessRatePct, &metrics.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}

// ========== Phase 3.12: Multi-Tenant & Priority Failover ==========

// InsertFailoverChainState inserts a new chain state record
func (p *PostgresStore) InsertFailoverChainState(ctx context.Context, state *FailoverChainState) error {
	state.ID = uuid.New()
	query := `
		INSERT INTO failover_chain_states
		(id, chain_id, tenant_id, last_executed_at, next_eligible_at, current_step_index,
		 is_executing, execution_lock_at, last_error, consecutive_failures, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := p.db.ExecContext(ctx, query,
		state.ID, state.ChainID, state.TenantID, state.LastExecutedAt, state.NextEligibleAt,
		state.CurrentStepIndex, state.IsExecuting, state.ExecutionLockAt, state.LastError,
		state.ConsecutiveFailures, time.Now().UTC(),
	)

	return err
}

// UpdateFailoverChainState updates an existing chain state
func (p *PostgresStore) UpdateFailoverChainState(ctx context.Context, id uuid.UUID, state *FailoverChainState) error {
	query := `
		UPDATE failover_chain_states
		SET last_executed_at = $2, next_eligible_at = $3, current_step_index = $4,
		    is_executing = $5, execution_lock_at = $6, last_error = $7,
		    consecutive_failures = $8, updated_at = $9
		WHERE id = $1
	`

	_, err := p.db.ExecContext(ctx, query,
		id, state.LastExecutedAt, state.NextEligibleAt, state.CurrentStepIndex,
		state.IsExecuting, state.ExecutionLockAt, state.LastError,
		state.ConsecutiveFailures, time.Now().UTC(),
	)

	return err
}

// GetFailoverChainState retrieves state for a specific chain/tenant
func (p *PostgresStore) GetFailoverChainState(ctx context.Context, chainID uuid.UUID, tenantID uuid.UUID) (*FailoverChainState, error) {
	query := `
		SELECT id, chain_id, tenant_id, last_executed_at, next_eligible_at, current_step_index,
		       is_executing, execution_lock_at, last_error, consecutive_failures, updated_at
		FROM failover_chain_states
		WHERE chain_id = $1 AND tenant_id = $2
	`

	var state FailoverChainState
	err := p.db.QueryRowContext(ctx, query, chainID, tenantID).Scan(
		&state.ID, &state.ChainID, &state.TenantID, &state.LastExecutedAt, &state.NextEligibleAt,
		&state.CurrentStepIndex, &state.IsExecuting, &state.ExecutionLockAt, &state.LastError,
		&state.ConsecutiveFailures, &state.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &state, nil
}

// ListFailoverChainStates lists all states for a tenant
func (p *PostgresStore) ListFailoverChainStates(ctx context.Context, tenantID uuid.UUID) ([]FailoverChainState, error) {
	query := `
		SELECT id, chain_id, tenant_id, last_executed_at, next_eligible_at, current_step_index,
		       is_executing, execution_lock_at, last_error, consecutive_failures, updated_at
		FROM failover_chain_states
		WHERE tenant_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []FailoverChainState
	for rows.Next() {
		var state FailoverChainState
		err := rows.Scan(
			&state.ID, &state.ChainID, &state.TenantID, &state.LastExecutedAt, &state.NextEligibleAt,
			&state.CurrentStepIndex, &state.IsExecuting, &state.ExecutionLockAt, &state.LastError,
			&state.ConsecutiveFailures, &state.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}

	return states, rows.Err()
}

// LockChainForExecution acquires exclusive execution lock on a chain
func (p *PostgresStore) LockChainForExecution(ctx context.Context, chainID uuid.UUID, tenantID uuid.UUID, lockDurationMs int) error {
	lockTime := time.Now().UTC().Add(time.Duration(lockDurationMs) * time.Millisecond)
	query := `
		UPDATE failover_chain_states
		SET is_executing = true, execution_lock_at = $2, updated_at = $3
		WHERE chain_id = $1 AND tenant_id = $4 AND (is_executing = false OR execution_lock_at < $3)
	`

	_, err := p.db.ExecContext(ctx, query, chainID, lockTime, time.Now().UTC(), tenantID)
	return err
}

// UnlockChainForExecution releases execution lock on a chain
func (p *PostgresStore) UnlockChainForExecution(ctx context.Context, chainID uuid.UUID, tenantID uuid.UUID) error {
	query := `
		UPDATE failover_chain_states
		SET is_executing = false, execution_lock_at = NULL, updated_at = $3
		WHERE chain_id = $1 AND tenant_id = $2
	`

	_, err := p.db.ExecContext(ctx, query, chainID, tenantID, time.Now().UTC())
	return err
}

// InsertFailoverChainConflict records a conflict between two chains
func (p *PostgresStore) InsertFailoverChainConflict(ctx context.Context, conflict *FailoverChainConflict) error {
	conflict.ID = uuid.New()
	now := time.Now().UTC()

	query := `
		INSERT INTO failover_chain_conflicts
		(id, tenant_id, chain_id_1, chain_id_2, conflict_type, source_region_1, source_region_2,
		 shared_targets, resolution_rule, is_resolved, resolved_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := p.db.ExecContext(ctx, query,
		conflict.ID, conflict.TenantID, conflict.ChainID1, conflict.ChainID2, conflict.ConflictType,
		conflict.SourceRegion1, conflict.SourceRegion2, conflict.SharedTargets, conflict.ResolutionRule,
		conflict.IsResolved, conflict.ResolvedAt, now, now,
	)

	return err
}

// ListFailoverChainConflicts lists conflicts for a chain
func (p *PostgresStore) ListFailoverChainConflicts(ctx context.Context, tenantID uuid.UUID, chainID uuid.UUID) ([]FailoverChainConflict, error) {
	query := `
		SELECT id, tenant_id, chain_id_1, chain_id_2, conflict_type, source_region_1, source_region_2,
		       shared_targets, resolution_rule, is_resolved, resolved_at, created_at, updated_at
		FROM failover_chain_conflicts
		WHERE tenant_id = $1 AND (chain_id_1 = $2 OR chain_id_2 = $2)
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, tenantID, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []FailoverChainConflict
	for rows.Next() {
		var conflict FailoverChainConflict
		err := rows.Scan(
			&conflict.ID, &conflict.TenantID, &conflict.ChainID1, &conflict.ChainID2, &conflict.ConflictType,
			&conflict.SourceRegion1, &conflict.SourceRegion2, &conflict.SharedTargets, &conflict.ResolutionRule,
			&conflict.IsResolved, &conflict.ResolvedAt, &conflict.CreatedAt, &conflict.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		conflicts = append(conflicts, conflict)
	}

	return conflicts, rows.Err()
}

// UpdateConflictResolution marks a conflict as resolved
func (p *PostgresStore) UpdateConflictResolution(ctx context.Context, conflictID uuid.UUID, resolved bool, rule string) error {
	query := `
		UPDATE failover_chain_conflicts
		SET is_resolved = $2, resolution_rule = $3, resolved_at = $4, updated_at = $4
		WHERE id = $1
	`

	now := time.Now().UTC()
	_, err := p.db.ExecContext(ctx, query, conflictID, resolved, rule, now)
	return err
}

// GetConflictingChains returns all chains conflicting with a given chain
func (p *PostgresStore) GetConflictingChains(ctx context.Context, tenantID uuid.UUID, chainID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT CASE
		       WHEN chain_id_1 = $2 THEN chain_id_2
		       ELSE chain_id_1
		END as conflicting_chain_id
		FROM failover_chain_conflicts
		WHERE tenant_id = $1 AND (chain_id_1 = $2 OR chain_id_2 = $2) AND is_resolved = false
	`

	rows, err := p.db.QueryContext(ctx, query, tenantID, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflictingChains []uuid.UUID
	for rows.Next() {
		var conflictID uuid.UUID
		if err := rows.Scan(&conflictID); err != nil {
			return nil, err
		}
		conflictingChains = append(conflictingChains, conflictID)
	}

	return conflictingChains, rows.Err()
}

// UpsertChainExecutionMetricsAdvanced inserts or updates advanced execution metrics
func (p *PostgresStore) UpsertChainExecutionMetricsAdvanced(ctx context.Context, metrics *ChainExecutionMetricsAdvanced) error {
	metrics.ID = uuid.New()
	now := time.Now().UTC()

	query := `
		INSERT INTO chain_execution_metrics_advanced
		(id, chain_id, total_executions, p50_duration_ms, p75_duration_ms, p95_duration_ms, p99_duration_ms,
		 max_duration_ms, min_duration_ms, std_dev_duration_ms, success_rate_99th, avg_steps_needed, p95_steps_needed,
		 most_common_failure, sla_compliance, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (chain_id) DO UPDATE SET
			total_executions = $3, p50_duration_ms = $4, p75_duration_ms = $5, p95_duration_ms = $6,
			p99_duration_ms = $7, max_duration_ms = $8, min_duration_ms = $9, std_dev_duration_ms = $10,
			success_rate_99th = $11, avg_steps_needed = $12, p95_steps_needed = $13, most_common_failure = $14,
			sla_compliance = $15, updated_at = $16
	`

	_, err := p.db.ExecContext(ctx, query,
		metrics.ID, metrics.ChainID, metrics.TotalExecutions, metrics.P50DurationMs, metrics.P75DurationMs,
		metrics.P95DurationMs, metrics.P99DurationMs, metrics.MaxDurationMs, metrics.MinDurationMs,
		metrics.StdDevDurationMs, metrics.SuccessRate99th, metrics.AvgStepsNeeded, metrics.P95StepsNeeded,
		metrics.MostCommonFailure, metrics.SLACompliance, now,
	)

	return err
}

// GetChainExecutionMetricsAdvanced retrieves advanced metrics for a chain
func (p *PostgresStore) GetChainExecutionMetricsAdvanced(ctx context.Context, chainID uuid.UUID) (*ChainExecutionMetricsAdvanced, error) {
	query := `
		SELECT id, chain_id, total_executions, p50_duration_ms, p75_duration_ms, p95_duration_ms, p99_duration_ms,
		       max_duration_ms, min_duration_ms, std_dev_duration_ms, success_rate_99th, avg_steps_needed, p95_steps_needed,
		       most_common_failure, sla_compliance, updated_at
		FROM chain_execution_metrics_advanced
		WHERE chain_id = $1
	`

	var metrics ChainExecutionMetricsAdvanced
	err := p.db.QueryRowContext(ctx, query, chainID).Scan(
		&metrics.ID, &metrics.ChainID, &metrics.TotalExecutions, &metrics.P50DurationMs, &metrics.P75DurationMs,
		&metrics.P95DurationMs, &metrics.P99DurationMs, &metrics.MaxDurationMs, &metrics.MinDurationMs,
		&metrics.StdDevDurationMs, &metrics.SuccessRate99th, &metrics.AvgStepsNeeded, &metrics.P95StepsNeeded,
		&metrics.MostCommonFailure, &metrics.SLACompliance, &metrics.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}

// ListChainsSortedBySLACompliance lists chains sorted by SLA compliance (highest first)
func (p *PostgresStore) ListChainsSortedBySLACompliance(ctx context.Context, tenantID uuid.UUID) ([]ChainExecutionMetricsAdvanced, error) {
	query := `
		SELECT acema.id, acema.chain_id, acema.total_executions, acema.p50_duration_ms, acema.p75_duration_ms,
		       acema.p95_duration_ms, acema.p99_duration_ms, acema.max_duration_ms, acema.min_duration_ms,
		       acema.std_dev_duration_ms, acema.success_rate_99th, acema.avg_steps_needed, acema.p95_steps_needed,
		       acema.most_common_failure, acema.sla_compliance, acema.updated_at
		FROM chain_execution_metrics_advanced acema
		JOIN failover_chains fc ON acema.chain_id = fc.id
		WHERE fc.tenant_id = $1
		ORDER BY acema.sla_compliance DESC
	`

	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []ChainExecutionMetricsAdvanced
	for rows.Next() {
		var m ChainExecutionMetricsAdvanced
		err := rows.Scan(
			&m.ID, &m.ChainID, &m.TotalExecutions, &m.P50DurationMs, &m.P75DurationMs,
			&m.P95DurationMs, &m.P99DurationMs, &m.MaxDurationMs, &m.MinDurationMs,
			&m.StdDevDurationMs, &m.SuccessRate99th, &m.AvgStepsNeeded, &m.P95StepsNeeded,
			&m.MostCommonFailure, &m.SLACompliance, &m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, rows.Err()
}

// InsertChainPriorityExecution inserts a new priority execution queue
func (p *PostgresStore) InsertChainPriorityExecution(ctx context.Context, execution *ChainPriorityExecution) error {
	execution.ID = uuid.New()
	now := time.Now().UTC()

	query := `
		INSERT INTO chain_priority_executions
		(id, tenant_id, incident_id, chains_to_execute, execution_order, current_chain_idx,
		 status, completed_chains, failed_chains, started_at, completed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := p.db.ExecContext(ctx, query,
		execution.ID, execution.TenantID, execution.IncidentID, execution.ChainsToExecute,
		execution.ExecutionOrder, execution.CurrentChainIdx, execution.Status, execution.CompletedChains,
		execution.FailedChains, execution.StartedAt, execution.CompletedAt, now, now,
	)

	return err
}

// UpdateChainPriorityExecution updates execution queue progress
func (p *PostgresStore) UpdateChainPriorityExecution(ctx context.Context, id uuid.UUID, currentIdx int, status string, completedChains []string, failedChains []string) error {
	completedJSON, _ := json.Marshal(completedChains)
	failedJSON, _ := json.Marshal(failedChains)

	query := `
		UPDATE chain_priority_executions
		SET current_chain_idx = $2, status = $3, completed_chains = $4, failed_chains = $5,
		    completed_at = CASE WHEN $3 = 'completed' OR $3 = 'failed' THEN $6 ELSE completed_at END,
		    updated_at = $6
		WHERE id = $1
	`

	_, err := p.db.ExecContext(ctx, query,
		id, currentIdx, status, string(completedJSON), string(failedJSON), time.Now().UTC(),
	)

	return err
}

// GetChainPriorityExecution retrieves a priority execution queue
func (p *PostgresStore) GetChainPriorityExecution(ctx context.Context, id uuid.UUID) (*ChainPriorityExecution, error) {
	query := `
		SELECT id, tenant_id, incident_id, chains_to_execute, execution_order, current_chain_idx,
		       status, completed_chains, failed_chains, started_at, completed_at, created_at, updated_at
		FROM chain_priority_executions
		WHERE id = $1
	`

	var execution ChainPriorityExecution
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&execution.ID, &execution.TenantID, &execution.IncidentID, &execution.ChainsToExecute,
		&execution.ExecutionOrder, &execution.CurrentChainIdx, &execution.Status, &execution.CompletedChains,
		&execution.FailedChains, &execution.StartedAt, &execution.CompletedAt, &execution.CreatedAt, &execution.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &execution, nil
}

// ListPendingChainQueues lists pending priority execution queues for a tenant
func (p *PostgresStore) ListPendingChainQueues(ctx context.Context, tenantID uuid.UUID) ([]ChainPriorityExecution, error) {
	query := `
		SELECT id, tenant_id, incident_id, chains_to_execute, execution_order, current_chain_idx,
		       status, completed_chains, failed_chains, started_at, completed_at, created_at, updated_at
		FROM chain_priority_executions
		WHERE tenant_id = $1 AND status IN ('pending', 'in_progress')
		ORDER BY started_at ASC
	`

	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []ChainPriorityExecution
	for rows.Next() {
		var execution ChainPriorityExecution
		err := rows.Scan(
			&execution.ID, &execution.TenantID, &execution.IncidentID, &execution.ChainsToExecute,
			&execution.ExecutionOrder, &execution.CurrentChainIdx, &execution.Status, &execution.CompletedChains,
			&execution.FailedChains, &execution.StartedAt, &execution.CompletedAt, &execution.CreatedAt, &execution.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}

	return executions, rows.Err()
}

// ========== Phase 3.14: Analytics & Trends Methods ==========

// UpsertSLAComplianceTrend inserts or updates SLA compliance trend
func (p *PostgresStore) UpsertSLAComplianceTrend(ctx context.Context, trend *SLAComplianceTrend) error {
	query := `
		INSERT INTO sla_compliance_trends (id, chain_id, tenant_id, compliance_score, success_rate_trend, latency_trend, percentile_99, status, reported_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (chain_id) DO UPDATE SET
		compliance_score = $4, success_rate_trend = $5, latency_trend = $6, percentile_99 = $7, status = $8, reported_at = $9
	`

	return p.db.QueryRowContext(ctx, query,
		trend.ID, trend.ChainID, trend.TenantID, trend.ComplianceScore, trend.SuccessRateTrend,
		trend.LatencyTrend, trend.Percentile99, trend.Status, trend.ReportedAt, trend.CreatedAt,
	).Err()
}

// ListSLAComplianceTrends retrieves SLA trends for a tenant
func (p *PostgresStore) ListSLAComplianceTrends(ctx context.Context, tenantID uuid.UUID, limit int) ([]SLAComplianceTrend, error) {
	query := `
		SELECT id, chain_id, tenant_id, compliance_score, success_rate_trend, latency_trend, percentile_99, status, reported_at, created_at
		FROM sla_compliance_trends
		WHERE tenant_id = $1
		ORDER BY reported_at DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []SLAComplianceTrend
	for rows.Next() {
		var t SLAComplianceTrend
		err := rows.Scan(&t.ID, &t.ChainID, &t.TenantID, &t.ComplianceScore, &t.SuccessRateTrend, &t.LatencyTrend, &t.Percentile99, &t.Status, &t.ReportedAt, &t.CreatedAt)
		if err != nil {
			return nil, err
		}
		trends = append(trends, t)
	}

	return trends, rows.Err()
}

// UpsertConflictResolutionTrend inserts or updates conflict resolution trend
func (p *PostgresStore) UpsertConflictResolutionTrend(ctx context.Context, trend *ConflictResolutionTrend) error {
	query := `
		INSERT INTO conflict_resolution_trends (id, tenant_id, total_conflicts, resolved_count, failed_count, resolution_rate, avg_resolution_ms, most_common_rule, period_start, period_end, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (tenant_id, period_start) DO UPDATE SET
		total_conflicts = $3, resolved_count = $4, failed_count = $5, resolution_rate = $6, avg_resolution_ms = $7, most_common_rule = $8
	`

	return p.db.QueryRowContext(ctx, query,
		trend.ID, trend.TenantID, trend.TotalConflicts, trend.ResolvedCount, trend.FailedCount,
		trend.ResolutionRate, trend.AvgResolutionMs, trend.MostCommonRule, trend.PeriodStart, trend.PeriodEnd, trend.CreatedAt,
	).Err()
}

// GetConflictResolutionTrend retrieves a conflict resolution trend
func (p *PostgresStore) GetConflictResolutionTrend(ctx context.Context, tenantID uuid.UUID, periodStart time.Time) (*ConflictResolutionTrend, error) {
	query := `
		SELECT id, tenant_id, total_conflicts, resolved_count, failed_count, resolution_rate, avg_resolution_ms, most_common_rule, period_start, period_end, created_at
		FROM conflict_resolution_trends
		WHERE tenant_id = $1 AND period_start = $2
	`

	var trend ConflictResolutionTrend
	err := p.db.QueryRowContext(ctx, query, tenantID, periodStart).Scan(
		&trend.ID, &trend.TenantID, &trend.TotalConflicts, &trend.ResolvedCount, &trend.FailedCount,
		&trend.ResolutionRate, &trend.AvgResolutionMs, &trend.MostCommonRule, &trend.PeriodStart, &trend.PeriodEnd, &trend.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &trend, err
}

// UpsertChainExecutionStats inserts or updates chain execution stats
func (p *PostgresStore) UpsertChainExecutionStats(ctx context.Context, stats *ChainExecutionStats) error {
	query := `
		INSERT INTO chain_execution_stats (id, chain_id, tenant_id, total_executions, successful_executions, failed_executions, success_rate_pct, avg_execution_ms, max_execution_ms, min_execution_ms, last_success_at, last_failure_at, period_start, period_end, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (chain_id, period_start) DO UPDATE SET
		total_executions = $4, successful_executions = $5, failed_executions = $6, success_rate_pct = $7, avg_execution_ms = $8, max_execution_ms = $9, min_execution_ms = $10, last_success_at = $11, last_failure_at = $12
	`

	return p.db.QueryRowContext(ctx, query,
		stats.ID, stats.ChainID, stats.TenantID, stats.TotalExecutions, stats.SuccessfulExecutions, stats.FailedExecutions,
		stats.SuccessRatePct, stats.AvgExecutionMs, stats.MaxExecutionMs, stats.MinExecutionMs, stats.LastSuccessAt, stats.LastFailureAt,
		stats.PeriodStart, stats.PeriodEnd, stats.CreatedAt,
	).Err()
}

// GetChainExecutionStats retrieves execution stats for a chain
func (p *PostgresStore) GetChainExecutionStats(ctx context.Context, chainID uuid.UUID) (*ChainExecutionStats, error) {
	query := `
		SELECT id, chain_id, tenant_id, total_executions, successful_executions, failed_executions, success_rate_pct, avg_execution_ms, max_execution_ms, min_execution_ms, last_success_at, last_failure_at, period_start, period_end, created_at
		FROM chain_execution_stats
		WHERE chain_id = $1
		ORDER BY period_end DESC
		LIMIT 1
	`

	var stats ChainExecutionStats
	err := p.db.QueryRowContext(ctx, query, chainID).Scan(
		&stats.ID, &stats.ChainID, &stats.TenantID, &stats.TotalExecutions, &stats.SuccessfulExecutions, &stats.FailedExecutions,
		&stats.SuccessRatePct, &stats.AvgExecutionMs, &stats.MaxExecutionMs, &stats.MinExecutionMs, &stats.LastSuccessAt, &stats.LastFailureAt,
		&stats.PeriodStart, &stats.PeriodEnd, &stats.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &stats, err
}

// UpsertChainHealthReport inserts or updates chain health report
func (p *PostgresStore) UpsertChainHealthReport(ctx context.Context, report *ChainHealthReport) error {
	query := `
		INSERT INTO chain_health_reports (id, chain_id, tenant_id, overall_health, last_execution_status, consecutive_failures, is_healthy, recommended_action, reported_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (chain_id) DO UPDATE SET
		overall_health = $4, last_execution_status = $5, consecutive_failures = $6, is_healthy = $7, recommended_action = $8, reported_at = $9
	`

	return p.db.QueryRowContext(ctx, query,
		report.ID, report.ChainID, report.TenantID, report.OverallHealth, report.LastExecutionStatus, report.ConsecutiveFailures,
		report.IsHealthy, report.RecommendedAction, report.ReportedAt, report.CreatedAt,
	).Err()
}

// GetChainHealthReport retrieves health report for a chain
func (p *PostgresStore) GetChainHealthReport(ctx context.Context, chainID uuid.UUID) (*ChainHealthReport, error) {
	query := `
		SELECT id, chain_id, tenant_id, overall_health, last_execution_status, consecutive_failures, is_healthy, recommended_action, reported_at, created_at
		FROM chain_health_reports
		WHERE chain_id = $1
	`

	var report ChainHealthReport
	err := p.db.QueryRowContext(ctx, query, chainID).Scan(
		&report.ID, &report.ChainID, &report.TenantID, &report.OverallHealth, &report.LastExecutionStatus, &report.ConsecutiveFailures,
		&report.IsHealthy, &report.RecommendedAction, &report.ReportedAt, &report.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &report, err
}

// ListChainsByFilter implements advanced filtering for chains
func (p *PostgresStore) ListChainsByFilter(ctx context.Context, criteria *ChainFilterCriteria) ([]FailoverChain, error) {
	query := `SELECT id, name, source_region, chain_targets, trigger_health_score, trigger_error_rate, trigger_latency, max_chain_depth, cooldown_minutes, priority, is_enabled, created_at, updated_at FROM failover_chains WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if criteria.TenantID != nil {
		query += fmt.Sprintf(` AND tenant_id = $%d`, argIdx)
		args = append(args, *criteria.TenantID)
		argIdx++
	}

	if criteria.SourceRegion != nil {
		query += fmt.Sprintf(` AND source_region = $%d`, argIdx)
		args = append(args, *criteria.SourceRegion)
		argIdx++
	}

	if criteria.IsEnabled != nil {
		query += fmt.Sprintf(` AND is_enabled = $%d`, argIdx)
		args = append(args, *criteria.IsEnabled)
		argIdx++
	}

	if criteria.SortBy != "" {
		sortCol := "created_at"
		if criteria.SortBy == "sla_compliance" {
			sortCol = "priority"
		}
		sortOrder := "DESC"
		if criteria.SortOrder == "asc" {
			sortOrder = "ASC"
		}
		query += fmt.Sprintf(` ORDER BY %s %s`, sortCol, sortOrder)
	}

	if criteria.Limit > 0 {
		query += fmt.Sprintf(` LIMIT $%d`, argIdx)
		args = append(args, criteria.Limit)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []FailoverChain
	for rows.Next() {
		var chain FailoverChain
		err := rows.Scan(&chain.ID, &chain.Name, &chain.SourceRegion, &chain.ChainTargets, &chain.TriggerHealthScore, &chain.TriggerErrorRate, &chain.TriggerLatency, &chain.MaxChainDepth, &chain.CooldownMinutes, &chain.Priority, &chain.IsEnabled, &chain.CreatedAt, &chain.UpdatedAt)
		if err != nil {
			return nil, err
		}
		chains = append(chains, chain)
	}

	return chains, rows.Err()
}

// SearchChains implements full-text-like search for chains
func (p *PostgresStore) SearchChains(ctx context.Context, tenantID uuid.UUID, searchTerm string, limit int) ([]FailoverChain, error) {
	query := `
		SELECT id, name, source_region, chain_targets, trigger_health_score, trigger_error_rate, trigger_latency, max_chain_depth, cooldown_minutes, priority, is_enabled, created_at, updated_at
		FROM failover_chains
		WHERE tenant_id = $1 AND (name ILIKE $2 OR source_region ILIKE $2)
		ORDER BY name ASC
		LIMIT $3
	`

	searchPattern := "%" + searchTerm + "%"
	rows, err := p.db.QueryContext(ctx, query, tenantID, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []FailoverChain
	for rows.Next() {
		var chain FailoverChain
		err := rows.Scan(&chain.ID, &chain.Name, &chain.SourceRegion, &chain.ChainTargets, &chain.TriggerHealthScore, &chain.TriggerErrorRate, &chain.TriggerLatency, &chain.MaxChainDepth, &chain.CooldownMinutes, &chain.Priority, &chain.IsEnabled, &chain.CreatedAt, &chain.UpdatedAt)
		if err != nil {
			return nil, err
		}
		chains = append(chains, chain)
	}

	return chains, rows.Err()
}

// InsertBatchConflictResolution creates a batch conflict resolution operation
func (p *PostgresStore) InsertBatchConflictResolution(ctx context.Context, batch *BatchConflictResolution) error {
	query := `
		INSERT INTO batch_conflict_resolutions (id, tenant_id, conflict_ids, resolution_rule, status, total_conflicts, resolved_count, failed_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	return p.db.QueryRowContext(ctx, query,
		batch.ID, batch.TenantID, batch.ConflictIDs, batch.ResolutionRule, batch.Status, batch.TotalConflicts,
		batch.ResolvedCount, batch.FailedCount, batch.CreatedAt, batch.UpdatedAt,
	).Err()
}

// UpdateBatchConflictResolution updates a batch operation's progress
func (p *PostgresStore) UpdateBatchConflictResolution(ctx context.Context, id uuid.UUID, resolvedCount int, failedCount int, status string) error {
	query := `
		UPDATE batch_conflict_resolutions
		SET resolved_count = $1, failed_count = $2, status = $3, executed_at = NOW(), updated_at = NOW()
		WHERE id = $4
	`

	return p.db.QueryRowContext(ctx, query, resolvedCount, failedCount, status, id).Err()
}

// GetBatchConflictResolution retrieves a batch conflict resolution operation
func (p *PostgresStore) GetBatchConflictResolution(ctx context.Context, id uuid.UUID) (*BatchConflictResolution, error) {
	query := `
		SELECT id, tenant_id, conflict_ids, resolution_rule, status, total_conflicts, resolved_count, failed_count, executed_at, created_at, updated_at
		FROM batch_conflict_resolutions
		WHERE id = $1
	`

	var batch BatchConflictResolution
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&batch.ID, &batch.TenantID, &batch.ConflictIDs, &batch.ResolutionRule, &batch.Status, &batch.TotalConflicts,
		&batch.ResolvedCount, &batch.FailedCount, &batch.ExecutedAt, &batch.CreatedAt, &batch.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &batch, err
}
