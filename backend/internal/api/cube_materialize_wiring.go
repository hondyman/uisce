package api

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/activity"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/querybuilder"
)

// registerCubeMaterializeWorker starts an in-process Temporal worker on
// uisce-cubes for CubeMaterializeWorkflow (CUBE-1.2). Designer Deploy and
// schedule kind=cube_refresh (CUBE-1.5) target this queue.
func (s *Server) registerCubeMaterializeWorker(sqlxDB *sqlx.DB, tc temporalclient.Client) {
	log := logging.GetLogger().Sugar()
	if tc == nil {
		log.Warnf("cubes: no Temporal client; CubeMaterializeWorkflow will not run")
		return
	}
	if sqlxDB == nil {
		log.Warnf("cubes: no metadata database; CubeMaterializeWorkflow worker not started")
		return
	}

	starrocksDB := analytics.OpenStarRocksDB()
	acts := querybuilder.NewCubeMaterializeActivities(sqlxDB, starrocksDB)
	s.cubeMaterializeActs = acts

	w := worker.New(tc, querybuilder.CubeTaskQueue, worker.Options{})
	w.RegisterWorkflowWithOptions(querybuilder.CubeMaterializeWorkflow, workflow.RegisterOptions{
		Name: querybuilder.CubeMaterializeWorkflowName,
	})
	w.RegisterActivityWithOptions(acts.CubeValidateAndPlan, activity.RegisterOptions{Name: querybuilder.ActCubeValidateAndPlan})
	w.RegisterActivityWithOptions(acts.CubeBeginAttempt, activity.RegisterOptions{Name: querybuilder.ActCubeBeginAttempt})
	w.RegisterActivityWithOptions(acts.CubeApplyHot, activity.RegisterOptions{Name: querybuilder.ActCubeApplyHot})
	w.RegisterActivityWithOptions(acts.CubeCompleteAttempt, activity.RegisterOptions{Name: querybuilder.ActCubeCompleteAttempt})
	w.RegisterActivityWithOptions(acts.CubeFailAttempt, activity.RegisterOptions{Name: querybuilder.ActCubeFailAttempt})

	if err := w.Start(); err != nil {
		log.Errorf("cubes: CubeMaterialize worker did not start: %v", err)
		return
	}
	log.Infof("cubes: worker started on task queue %s", querybuilder.CubeTaskQueue)
}

// StartCubeMaterialize starts CubeMaterializeWorkflow with REJECT_DUPLICATE.
// Caller must supply a grain that exists on the cube; contract_version and
// grain_hash are taken from the validated plan after a local ValidateAndPlan
// so the workflow ID matches the in-flight attempt.
func (s *Server) StartCubeMaterialize(ctx context.Context, req querybuilder.CubeMaterializeRequest) (string, *querybuilder.CubeMaterializePlan, error) {
	if s.TemporalClient == nil {
		return "", nil, fmt.Errorf("cubes: Temporal client not configured")
	}
	if s.cubeMaterializeActs == nil || s.cubeMaterializeActs.Materializer == nil {
		return "", nil, fmt.Errorf("cubes: materializer not configured")
	}
	plan, err := s.cubeMaterializeActs.Materializer.ValidateAndPlan(ctx, req)
	if err != nil {
		return "", nil, err
	}
	opts := querybuilder.CubeMaterializeStartOptions(plan.TenantID, plan.CubeID, plan.ContractVersion, plan.GrainHash)
	// Re-issue with the planned attempt_id so the workflow and catalog agree.
	req.AttemptID = plan.AttemptID
	req.Grain = plan.Grain
	run, err := s.TemporalClient.ExecuteWorkflow(ctx, opts, querybuilder.CubeMaterializeWorkflowName, req)
	if err != nil {
		return "", plan, err
	}
	return run.GetID(), plan, nil
}
