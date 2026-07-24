package optimize

import (
	"time"

	"github.com/google/uuid"
)

// QueryLog captures detailed telemetry for a single query execution.
type QueryLog struct {
	ID                 uuid.UUID `db:"id"`
	Timestamp          time.Time `db:"timestamp"`
	DatasourceID       string    `db:"datasource_id"`
	Models             []string  `db:"models"` // Using pq.Array for PostgreSQL
	Measures           []string  `db:"measures"`
	Dimensions         []string  `db:"dimensions"`
	Granularity        string    `db:"granularity"`
	FiltersHash        string    `db:"filters_hash"`
	UsedPreaggregation *string   `db:"used_preaggregation"`
	PlanningMS         int64     `db:"planning_ms"`
	CompileMS          int64     `db:"compile_ms"`
	DBElapsedMS        int64     `db:"db_elapsed_ms"`
	RowsReturned       int       `db:"rows_returned"`
	PartitionPruned    bool      `db:"partition_pruned"`
	FreshnessOK        bool      `db:"freshness_ok"`
	ReaggOK            bool      `db:"reagg_ok"`
	FallbackReason     *string   `db:"fallback_reason"`
	Error              *string   `db:"error"`
}
