package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExplorerAuditRun represents a single query execution event for auditing.
type ExplorerAuditRun struct {
	ID          uuid.UUID       `db:"id"`
	OccurredAt  time.Time       `db:"occurred_at"`
	UserID      string          `db:"user_id"`
	TenantID    string          `db:"tenant_id"`
	Model       string          `db:"model"`
	Measures    json.RawMessage `db:"measures"`
	Dimensions  json.RawMessage `db:"dimensions"`
	Filters     json.RawMessage `db:"filters"`
	OrderBy     json.RawMessage `db:"order_by"`
	Limit       *int            `db:"limit"`
	Offset      *int            `db:"offset"`
	Timezone    string          `db:"timezone"`
	SQL         string          `db:"sql"`
	SQLHash     string          `db:"sql_hash"`
	UsedPreagg  *string         `db:"used_preagg"`
	PlanningMS  *int64          `db:"planning_ms"`
	CompileMS   *int64          `db:"compile_ms"`
	DBElapsedMS *int64          `db:"db_elapsed_ms"`
	RowCount    *int64          `db:"row_count"`
	RouteReason *string         `db:"route_reason"`
	RuleID      *string         `db:"rule_id"`
	Provenance  *string         `db:"provenance"`
}
