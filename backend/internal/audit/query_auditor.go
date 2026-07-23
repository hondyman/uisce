package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// QueryAudit represents a tracked semantic query execution
type QueryAudit struct {
	ID                 uuid.UUID       `db:"id"`
	TenantID           uuid.UUID       `db:"tenant_id"`
	UserID             uuid.UUID       `db:"user_id"`
	SessionID          string          `db:"session_id"`
	ModelID            string          `db:"model_id"`
	ModelName          string          `db:"model_name"`
	SemanticQuery      json.RawMessage `db:"semantic_query"`
	CompiledSQL        string          `db:"compiled_sql"`
	SQLParameters      json.RawMessage `db:"sql_parameters"`
	ExecutionStartTime *time.Time      `db:"execution_start_time"`
	ExecutionEndTime   *time.Time      `db:"execution_end_time"`
	DurationMS         *int64          `db:"duration_ms"`
	RowsScanned        *int64          `db:"rows_scanned"`
	RowsReturned       *int64          `db:"rows_returned"`
	CacheHit           bool            `db:"cache_hit"`
	QueryPlan          json.RawMessage `db:"query_plan"`
	ErrorMessage       string          `db:"error_message"`
	Status             string          `db:"status"` // success, error, timeout
	SchemaVersion      int             `db:"schema_version"`
	DriftIndicators    json.RawMessage `db:"drift_indicators"`
	CreatedAt          time.Time       `db:"created_at"`
}

// QueryAuditor manages query auditing
type QueryAuditor struct {
	db *sqlx.DB
}

// NewQueryAuditor creates new auditor
func NewQueryAuditor(db *sqlx.DB) *QueryAuditor {
	return &QueryAuditor{db: db}
}

// RecordQueryExecution records a query execution
func (qa *QueryAuditor) RecordQueryExecution(ctx context.Context, audit *QueryAudit) error {
	query := `
		INSERT INTO semantic_query_audit (
			id, tenant_id, user_id, session_id, model_id, model_name,
			semantic_query, compiled_sql, sql_parameters,
			execution_start_time, execution_end_time, duration_ms,
			rows_scanned, rows_returned, cache_hit,
			query_plan, error_message, status, schema_version,
			drift_indicators, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, now()
		)
	`

	audit.ID = uuid.New()
	audit.CreatedAt = time.Now()

	err := qa.db.QueryRowContext(
		ctx, query,
		audit.ID, audit.TenantID, audit.UserID, audit.SessionID,
		audit.ModelID, audit.ModelName, audit.SemanticQuery, audit.CompiledSQL,
		audit.SQLParameters, audit.ExecutionStartTime, audit.ExecutionEndTime,
		audit.DurationMS, audit.RowsScanned, audit.RowsReturned, audit.CacheHit,
		audit.QueryPlan, audit.ErrorMessage, audit.Status, audit.SchemaVersion,
		audit.DriftIndicators,
	).Err()

	if err != nil {
		log.Printf("❌ Failed to record query audit: %v", err)
		return err
	}

	log.Printf("✅ Query audit recorded: %s (duration: %dms)", audit.ModelName, *audit.DurationMS)
	return nil
}

// GetQueryAuditTrail retrieves audit trail for a model
func (qa *QueryAuditor) GetQueryAuditTrail(ctx context.Context, tenantID, modelID string, limit int) ([]QueryAudit, error) {
	query := `
		SELECT id, tenant_id, user_id, session_id, model_id, model_name,
		       semantic_query, compiled_sql, sql_parameters,
		       execution_start_time, execution_end_time, duration_ms,
		       rows_scanned, rows_returned, cache_hit,
		       query_plan, error_message, status, schema_version,
		       drift_indicators, created_at
		FROM semantic_query_audit
		WHERE tenant_id = $1 AND model_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`

	var audits []QueryAudit
	err := qa.db.SelectContext(ctx, &audits, query, tenantID, modelID, limit)
	return audits, err
}

// GetSlowQueries retrieves slow queries
func (qa *QueryAuditor) GetSlowQueries(ctx context.Context, tenantID string, thresholdMS int64, limit int) ([]QueryAudit, error) {
	query := `
		SELECT id, tenant_id, user_id, session_id, model_id, model_name,
		       semantic_query, compiled_sql, sql_parameters,
		       execution_start_time, execution_end_time, duration_ms,
		       rows_scanned, rows_returned, cache_hit,
		       query_plan, error_message, status, schema_version,
		       drift_indicators, created_at
		FROM semantic_query_audit
		WHERE tenant_id = $1 AND duration_ms > $2 AND status = 'success'
		ORDER BY duration_ms DESC
		LIMIT $3
	`

	var audits []QueryAudit
	err := qa.db.SelectContext(ctx, &audits, query, tenantID, thresholdMS, limit)
	return audits, err
}

// GetQueryStats retrieves performance statistics for a model
func (qa *QueryAuditor) GetQueryStats(ctx context.Context, tenantID, modelID string, hoursBack int) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_queries,
			AVG(duration_ms) as avg_duration_ms,
			MAX(duration_ms) as max_duration_ms,
			MIN(duration_ms) as min_duration_ms,
			SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END)::float / COUNT(*) as cache_hit_rate,
			SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as error_count,
			MAX(CASE WHEN execution_start_time > now() - interval '1 hour' THEN duration_ms ELSE 0 END) as last_hour_max_ms,
			SUM(rows_returned) as total_rows_returned
		FROM semantic_query_audit
		WHERE tenant_id = $1 AND model_id = $2
		AND created_at > now() - interval '1 hour' * $3
	`

	var totalQueries int
	var avgDuration, maxDuration, minDuration *float64
	var cacheHitRate *float64
	var errorCount int
	var lastHourMax *int64
	var totalRowsReturned *int64

	err := qa.db.QueryRowContext(
		ctx, query, tenantID, modelID, hoursBack,
	).Scan(&totalQueries, &avgDuration, &maxDuration, &minDuration, &cacheHitRate, &errorCount, &lastHourMax, &totalRowsReturned)

	if err != nil {
		log.Printf("⚠️  Failed to get query stats: %v", err)
		return nil, err
	}

	stats := map[string]interface{}{
		"total_queries":       totalQueries,
		"avg_duration_ms":     avgDuration,
		"max_duration_ms":     maxDuration,
		"min_duration_ms":     minDuration,
		"cache_hit_rate":      cacheHitRate,
		"error_count":         errorCount,
		"last_hour_max_ms":    lastHourMax,
		"total_rows_returned": totalRowsReturned,
		"hours_analyzed":      hoursBack,
	}

	return stats, nil
}

// CompareQueriesForDrift compares two query compilations
func (qa *QueryAuditor) CompareQueriesForDrift(oldAudit, newAudit *QueryAudit) *DriftAnalysis {
	analysis := &DriftAnalysis{
		OldQueryID: oldAudit.ID.String(),
		NewQueryID: newAudit.ID.String(),
	}

	if oldAudit.ExecutionStartTime != nil && newAudit.ExecutionStartTime != nil {
		analysis.TimestampDiff = newAudit.ExecutionStartTime.Sub(*oldAudit.ExecutionStartTime)
	}

	if oldAudit.RowsScanned != nil && newAudit.RowsScanned != nil {
		analysis.RowsScannedDiff = *newAudit.RowsScanned - *oldAudit.RowsScanned
	}

	if oldAudit.RowsReturned != nil && newAudit.RowsReturned != nil {
		analysis.RowsReturnedDiff = *newAudit.RowsReturned - *oldAudit.RowsReturned
	}

	if oldAudit.DurationMS != nil && newAudit.DurationMS != nil {
		analysis.DurationDiff = *newAudit.DurationMS - *oldAudit.DurationMS
		analysis.DurationDiffPercent = float64(analysis.DurationDiff) / float64(*oldAudit.DurationMS) * 100
	}

	// Compare SQL
	if oldAudit.CompiledSQL != newAudit.CompiledSQL {
		analysis.SQLChanged = true
		analysis.SQLDiff = computeSQLDiff(oldAudit.CompiledSQL, newAudit.CompiledSQL)
	}

	return analysis
}

// DriftAnalysis represents comparison between query executions
type DriftAnalysis struct {
	OldQueryID          string
	NewQueryID          string
	TimestampDiff       time.Duration
	RowsScannedDiff     int64
	RowsReturnedDiff    int64
	DurationDiff        int64
	DurationDiffPercent float64
	SQLChanged          bool
	SQLDiff             string
}

func computeSQLDiff(oldSQL, newSQL string) string {
	// Simple implementation - just mark if different
	if len(oldSQL) > 50 {
		oldSQL = oldSQL[:50] + "..."
	}
	if len(newSQL) > 50 {
		newSQL = newSQL[:50] + "..."
	}
	return fmt.Sprintf("OLD: %s\nNEW: %s", oldSQL, newSQL)
}

// CleanupOldAudits removes audit records older than specified days
func (qa *QueryAuditor) CleanupOldAudits(ctx context.Context, daysOld int) (int64, error) {
	result, err := qa.db.ExecContext(
		ctx,
		"DELETE FROM semantic_query_audit WHERE created_at < now() - interval '1 day' * $1",
		daysOld,
	)

	if err != nil {
		log.Printf("❌ Failed to cleanup old audits: %v", err)
		return 0, err
	}

	rowsDeleted, _ := result.RowsAffected()
	log.Printf("✅ Cleaned up %d old audit records", rowsDeleted)
	return rowsDeleted, nil
}
