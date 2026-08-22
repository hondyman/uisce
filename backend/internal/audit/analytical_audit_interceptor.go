package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type QueryAuditTelemetry struct {
	TenantID          uuid.UUID              `json:"tenantId"`
	RequestID         string                 `json:"requestId"`
	UserID            string                 `json:"userId"`
	ServerHost        string                 `json:"serverHost"`
	ExecutionType     string                 `json:"executionType"` // SEMANTIC_QUERY, SSRS_REPORT_RENDER, VECTOR_CALC
	NormalizedQuery   string                 `json:"normalizedQuery"`
	ExecutionPlanJSON map[string]interface{} `json:"executionPlanJson"`
	RowCountReturned  int64                  `json:"rowCountReturned"` // Metrics only; NEVER stores result payload
	ScannedBytes      int64                  `json:"scannedBytes"`
	CPUDurationMs     int                    `json:"cpuDurationMs"`
	TotalLatencyMs    int                    `json:"totalLatencyMs"`
	EngineType        string                 `json:"engineType"` // STARROCKS, POSTGRES_ALPHA, ICEBERG, WASM_VECTOR
	AttributedCostUSD float64                `json:"attributedCostUSD"`
	Status            string                 `json:"status"` // COMPLETED, FAILED, BLOCKED_CIRCUIT_BREAKER
	ErrorSummary      string                 `json:"errorSummary,omitempty"`
}

type AnalyticalAuditInterceptor struct {
	db *sqlx.DB
}

func NewAnalyticalAuditInterceptor(db *sqlx.DB) *AnalyticalAuditInterceptor {
	return &AnalyticalAuditInterceptor{db: db}
}

// RecordQueryExecution logs execution metrics and plan DAG without storing customer result data
func (i *AnalyticalAuditInterceptor) RecordQueryExecution(
	ctx context.Context,
	telemetry QueryAuditTelemetry,
) error {
	if telemetry.TenantID == uuid.Nil {
		return fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	// 1. Calculate Normalized Query Fingerprint (SHA-256)
	h := sha256.Sum256([]byte(telemetry.NormalizedQuery))
	fingerprint := hex.EncodeToString(h[:])

	planJSONBytes, _ := json.Marshal(telemetry.ExecutionPlanJSON)

	if i.db == nil {
		return nil
	}

	// 2. Insert Telemetry into Audit Log Table
	query := `
		INSERT INTO audit.analytical_query_execution_logs (
			tenant_id, request_id, user_id, server_host, execution_type,
			query_fingerprint, normalized_query_text, execution_plan_json,
			row_count_returned, scanned_bytes, cpu_duration_ms, total_latency_ms,
			engine_type, attributed_cost_usd, status, error_summary, executed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW());
	`
	_, err := i.db.ExecContext(ctx, query,
		telemetry.TenantID,
		telemetry.RequestID,
		telemetry.UserID,
		telemetry.ServerHost,
		telemetry.ExecutionType,
		fingerprint,
		telemetry.NormalizedQuery,
		planJSONBytes,
		telemetry.RowCountReturned,
		telemetry.ScannedBytes,
		telemetry.CPUDurationMs,
		telemetry.TotalLatencyMs,
		telemetry.EngineType,
		telemetry.AttributedCostUSD,
		telemetry.Status,
		telemetry.ErrorSummary,
	)

	return err
}
