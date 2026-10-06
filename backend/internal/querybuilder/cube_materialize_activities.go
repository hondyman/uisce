package querybuilder

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Activity names for CubeMaterializeWorkflow (stable for registration + tests).
const (
	ActCubeValidateAndPlan    = "CubeValidateAndPlan"
	ActCubeBeginAttempt       = "CubeBeginAttempt"
	ActCubeExtractSources     = "CubeExtractSources"
	ActCubeDropStaging        = "CubeDropStaging"
	ActCubeApplyHot           = "CubeApplyHot"
	ActCubeApplyCold          = "CubeApplyCold"
	ActCubeCompensateHot      = "CubeCompensateHot"
	ActCubeCompleteDualCommit = "CubeCompleteDualCommit"
	ActCubeCompleteAttempt    = "CubeCompleteAttempt" // legacy hot-only; prefer dual-commit
	ActCubeFailAttempt        = "CubeFailAttempt"
)

// CubeMaterializeActivities wraps CubeMaterializer for Temporal.
// Lives in querybuilder (not temporal/activities) to avoid the
// querybuilder → handlers → … → temporal/activities → querybuilder import cycle.
type CubeMaterializeActivities struct {
	Materializer *CubeMaterializer
}

// NewCubeMaterializeActivities builds activities from control-plane + StarRocks DBs.
func NewCubeMaterializeActivities(db *sqlx.DB, starrocksDB *sql.DB) *CubeMaterializeActivities {
	return &CubeMaterializeActivities{
		Materializer: NewCubeMaterializer(db, starrocksDB),
	}
}

// CubeValidateAndPlan validates the cube contract and compiles hot DDL for one grain.
func (a *CubeMaterializeActivities) CubeValidateAndPlan(ctx context.Context, req CubeMaterializeRequest) (*CubeMaterializePlan, error) {
	if a == nil || a.Materializer == nil {
		return nil, fmt.Errorf("cube materialize activities: not configured")
	}
	return a.Materializer.ValidateAndPlan(ctx, req)
}

// CubeBeginAttempt marks the grain Materializing for the plan's attempt_id.
func (a *CubeMaterializeActivities) CubeBeginAttempt(ctx context.Context, plan *CubeMaterializePlan) error {
	if a == nil || a.Materializer == nil {
		return fmt.Errorf("cube materialize activities: not configured")
	}
	return a.Materializer.BeginAttempt(ctx, plan)
}

// CubeExtractSources CTAS federation sources into attempt-scoped StarRocks staging (Track C).
func (a *CubeMaterializeActivities) CubeExtractSources(ctx context.Context, plan *CubeMaterializePlan) (*CubeExtractResult, error) {
	if a == nil || a.Materializer == nil {
		return nil, fmt.Errorf("cube materialize activities: not configured")
	}
	return a.Materializer.ExtractSources(ctx, plan)
}

// CubeDropStaging drops attempt-scoped extract staging tables.
func (a *CubeMaterializeActivities) CubeDropStaging(ctx context.Context, plan *CubeMaterializePlan) error {
	if a == nil || a.Materializer == nil {
		return fmt.Errorf("cube materialize activities: not configured")
	}
	if plan == nil {
		return nil
	}
	return a.Materializer.DropStagingTables(ctx, plan.StagingTables)
}

// CubeApplyHot creates the StarRocks MV (single-BO extract+load via AS SELECT).
func (a *CubeMaterializeActivities) CubeApplyHot(ctx context.Context, plan *CubeMaterializePlan) (*CubeMaterializeHotResult, error) {
	if a == nil || a.Materializer == nil {
		return nil, fmt.Errorf("cube materialize activities: not configured")
	}
	return a.Materializer.ApplyHot(ctx, plan)
}

// CubeApplyCold commits Iceberg cold via StarRocks INSERT/CTAS (CUBE-1.3).
func (a *CubeMaterializeActivities) CubeApplyCold(ctx context.Context, plan *CubeMaterializePlan, hot *CubeMaterializeHotResult) (*CubeMaterializeColdResult, error) {
	if a == nil || a.Materializer == nil {
		return nil, fmt.Errorf("cube materialize activities: not configured")
	}
	return a.Materializer.ApplyCold(ctx, plan, hot)
}

// CubeCompensateHot drops the hot MV after a cold failure (CUBE-1.3).
func (a *CubeMaterializeActivities) CubeCompensateHot(ctx context.Context, plan *CubeMaterializePlan) error {
	if a == nil || a.Materializer == nil {
		return fmt.Errorf("cube materialize activities: not configured")
	}
	return a.Materializer.CompensateHot(ctx, plan)
}

// CubeCompleteDualCommit flips Active and stamps DualCommitWatermark (CUBE-1.3).
func (a *CubeMaterializeActivities) CubeCompleteDualCommit(
	ctx context.Context,
	plan *CubeMaterializePlan,
	hot *CubeMaterializeHotResult,
	cold *CubeMaterializeColdResult,
) error {
	if a == nil || a.Materializer == nil {
		return fmt.Errorf("cube materialize activities: not configured")
	}
	return a.Materializer.CompleteDualCommit(ctx, plan, hot, cold)
}

// CubeCompleteAttempt flips lifecycle Active (legacy hot-only path).
func (a *CubeMaterializeActivities) CubeCompleteAttempt(ctx context.Context, plan *CubeMaterializePlan, hot *CubeMaterializeHotResult) error {
	if a == nil || a.Materializer == nil {
		return fmt.Errorf("cube materialize activities: not configured")
	}
	return a.Materializer.CompleteAttempt(ctx, plan, hot)
}

// CubeFailAttemptInput carries the plan plus a serializable error message.
type CubeFailAttemptInput struct {
	Plan         *CubeMaterializePlan `json:"plan"`
	ErrorMessage string               `json:"error_message"`
}

// CubeFailAttempt marks Failed without advancing freshness or dual-commit watermark.
func (a *CubeMaterializeActivities) CubeFailAttempt(ctx context.Context, in CubeFailAttemptInput) error {
	if a == nil || a.Materializer == nil {
		return fmt.Errorf("cube materialize activities: not configured")
	}
	cause := fmt.Errorf("%s", in.ErrorMessage)
	if in.ErrorMessage == "" {
		cause = fmt.Errorf("cube materialize failed")
	}
	return a.Materializer.FailAttempt(ctx, in.Plan, cause)
}
