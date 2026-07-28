package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type AuditChannel string

const (
	ChannelREST        AuditChannel = "REST_API"
	ChannelJDBCPGWire  AuditChannel = "JDBC_PGWIRE"
	ChannelMCPAI       AuditChannel = "MCP_AI"
	ChannelGraphQL     AuditChannel = "GRAPHQL"
	ChannelUIDashboard AuditChannel = "UI_DASHBOARD"
)

type QueryAuditRecord struct {
	TelemetryID         string       `db:"telemetry_id" json:"telemetryId"`
	TenantID            string       `db:"tenant_id" json:"tenantId"`
	UserID              string       `db:"user_id" json:"userId"`
	BOID                string       `db:"bo_id" json:"boId"`
	Channel             AuditChannel `db:"channel" json:"channel"`
	ExecutionEngine     string       `db:"execution_engine" json:"executionEngine"`
	ExecutionDurationMs int          `db:"execution_duration_ms" json:"executionDurationMs"`
	EstimatedBytes      int64        `db:"estimated_bytes_scanned" json:"estimatedBytesScanned"`
	ExecutedSQL         string       `db:"executed_sql" json:"executedSql"`
	ClientIP            string       `db:"client_ip" json:"clientIp"`
	ComputeUnitsBilled  float64      `db:"compute_units_billed" json:"computeUnitsBilled"`
	EstimatedCostUSD    float64      `db:"estimated_cost_usd" json:"estimatedCostUsd"`
	CreatedAt           time.Time    `db:"created_at" json:"createdAt"`
}

type ChannelSummary struct {
	Channel        AuditChannel `db:"channel" json:"channel"`
	TotalQueries   int64        `db:"total_queries" json:"totalQueries"`
	TotalBytes     int64        `db:"total_bytes" json:"totalBytes"`
	TotalUnits     float64      `db:"total_units" json:"totalUnits"`
	TotalCostUSD   float64      `db:"total_cost_usd" json:"totalCostUsd"`
	AvgDurationMs  float64      `db:"avg_duration_ms" json:"avgDurationMs"`
}

type ChannelAuditService struct {
	db *sqlx.DB
}

func NewChannelAuditService(db *sqlx.DB) *ChannelAuditService {
	return &ChannelAuditService{db: db}
}

func (s *ChannelAuditService) LogQueryAudit(ctx context.Context, rec QueryAuditRecord) error {
	if rec.TelemetryID == "" {
		rec.TelemetryID = uuid.New().String()
	}
	if rec.Channel == "" {
		rec.Channel = ChannelREST
	}
	if rec.ComputeUnitsBilled <= 0 {
		rec.ComputeUnitsBilled = 1.0
	}
	if rec.EstimatedCostUSD <= 0 {
		rec.EstimatedCostUSD = rec.ComputeUnitsBilled * 0.0001
	}

	query := `
		INSERT INTO security.query_execution_telemetry (
			telemetry_id, tenant_id, user_id, bo_id, channel, execution_engine,
			execution_duration_ms, estimated_bytes_scanned, executed_sql,
			client_ip, compute_units_billed, estimated_cost_usd, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
	`
	_, err := s.db.ExecContext(ctx, query,
		rec.TelemetryID, rec.TenantID, rec.UserID, rec.BOID, rec.Channel, rec.ExecutionEngine,
		rec.ExecutionDurationMs, rec.EstimatedBytes, rec.ExecutedSQL,
		rec.ClientIP, rec.ComputeUnitsBilled, rec.EstimatedCostUSD,
	)
	return err
}

func (s *ChannelAuditService) GetChannelBillingSummary(ctx context.Context, tenantID string) ([]ChannelSummary, error) {
	query := `
		SELECT 
			channel,
			COUNT(*) as total_queries,
			COALESCE(SUM(estimated_bytes_scanned), 0) as total_bytes,
			COALESCE(SUM(compute_units_billed), 0) as total_units,
			COALESCE(SUM(estimated_cost_usd), 0) as total_cost_usd,
			COALESCE(AVG(execution_duration_ms), 0) as avg_duration_ms
		FROM security.query_execution_telemetry
		WHERE tenant_id = $1
		GROUP BY channel
		ORDER BY total_cost_usd DESC
	`
	var summaries []ChannelSummary
	err := s.db.SelectContext(ctx, &summaries, query, tenantID)
	return summaries, err
}

func (s *ChannelAuditService) GetRecentAuditLogs(ctx context.Context, tenantID string, limit int) ([]QueryAuditRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT 
			telemetry_id, tenant_id, user_id, bo_id, channel, execution_engine,
			execution_duration_ms, estimated_bytes_scanned, executed_sql,
			client_ip, compute_units_billed, estimated_cost_usd, created_at
		FROM security.query_execution_telemetry
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	var records []QueryAuditRecord
	err := s.db.SelectContext(ctx, &records, query, tenantID, limit)
	return records, err
}
