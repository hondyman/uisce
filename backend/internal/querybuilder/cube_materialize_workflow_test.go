package querybuilder

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func containsStr(s, sub string) bool { return strings.Contains(s, sub) }

func registerCubeMaterializeTestActs(env *testsuite.TestWorkflowEnvironment, acts *CubeMaterializeActivities) {
	env.RegisterActivityWithOptions(acts.CubeValidateAndPlan, activity.RegisterOptions{Name: ActCubeValidateAndPlan})
	env.RegisterActivityWithOptions(acts.CubeBeginAttempt, activity.RegisterOptions{Name: ActCubeBeginAttempt})
	env.RegisterActivityWithOptions(acts.CubeExtractSources, activity.RegisterOptions{Name: ActCubeExtractSources})
	env.RegisterActivityWithOptions(acts.CubeDropStaging, activity.RegisterOptions{Name: ActCubeDropStaging})
	env.RegisterActivityWithOptions(acts.CubeApplyHot, activity.RegisterOptions{Name: ActCubeApplyHot})
	env.RegisterActivityWithOptions(acts.CubeApplyCold, activity.RegisterOptions{Name: ActCubeApplyCold})
	env.RegisterActivityWithOptions(acts.CubeCompensateHot, activity.RegisterOptions{Name: ActCubeCompensateHot})
	env.RegisterActivityWithOptions(acts.CubeCompleteDualCommit, activity.RegisterOptions{Name: ActCubeCompleteDualCommit})
	env.RegisterActivityWithOptions(acts.CubeFailAttempt, activity.RegisterOptions{Name: ActCubeFailAttempt})
}

func TestCubeMaterializeWorkflow_HappyPathDualCommit(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	req := CubeMaterializeRequest{
		TenantID: "99e99e99-99e9-49e9-89e9-99e99e99e999",
		CubeID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
		Grain:    []string{"account_id", "status"},
	}
	plan := &CubeMaterializePlan{
		TenantID:            req.TenantID,
		CubeID:              req.CubeID,
		ContractVersion:     1,
		Grain:               req.Grain,
		GrainHash:           GrainHash(req.Grain),
		NodeID:              "11111111-1111-1111-1111-111111111111",
		AttemptID:           "attempt-1",
		MaterializationName: "cube_t_account_smoke",
		TargetDatabase:      "tenant_99e99e99",
		IcebergTable:        "iceberg_catalog.cubes.cube_t_account_smoke",
		DDL:                 "CREATE MATERIALIZED VIEW cube_t_account_smoke AS SELECT 1;",
	}
	hotAt := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	coldAt := time.Date(2026, 10, 5, 12, 0, 5, 0, time.UTC)
	hot := &CubeMaterializeHotResult{
		MaterializationName: plan.MaterializationName,
		TargetDatabase:      plan.TargetDatabase,
		AppliedDDL:          true,
		RowCount:            42,
		CommittedAt:         hotAt,
	}
	cold := &CubeMaterializeColdResult{
		IcebergTable: plan.IcebergTable,
		Applied:      true,
		RowCount:     42,
		CommittedAt:  coldAt,
	}

	acts := &CubeMaterializeActivities{}
	registerCubeMaterializeTestActs(env, acts)

	env.OnActivity(ActCubeValidateAndPlan, mock.Anything, req).Return(plan, nil)
	env.OnActivity(ActCubeBeginAttempt, mock.Anything, plan).Return(nil)
	env.OnActivity(ActCubeApplyHot, mock.Anything, plan).Return(hot, nil)
	env.OnActivity(ActCubeApplyCold, mock.Anything, plan, hot).Return(cold, nil)
	env.OnActivity(ActCubeCompleteDualCommit, mock.Anything, plan, hot, cold).Return(nil)

	env.ExecuteWorkflow(CubeMaterializeWorkflow, req)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result CubeMaterializeWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.True(t, result.HotApplied)
	require.True(t, result.ColdCommitted)
	require.Equal(t, int64(42), result.RowCount)
	require.Equal(t, plan.AttemptID, result.AttemptID)
	require.Equal(t, plan.IcebergTable, result.IcebergTable)
	require.Equal(t, coldAt.Format(time.RFC3339Nano), result.DualCommitWatermark)
	require.False(t, result.Noop)
	require.False(t, result.CompensatedHot)
}

func TestCubeMaterializeWorkflow_NoopSkipsLoad(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	req := CubeMaterializeRequest{
		TenantID: "t1",
		CubeID:   "c1",
		Grain:    []string{"day"},
	}
	plan := &CubeMaterializePlan{
		TenantID:        req.TenantID,
		CubeID:          req.CubeID,
		ContractVersion: 3,
		GrainHash:       GrainHash(req.Grain),
		NodeID:          "22222222-2222-2222-2222-222222222222",
		AttemptID:       "attempt-noop",
		Noop:            true,
		NoopReason:      "content_hash unchanged and grain already dual-committed Active",
	}

	acts := &CubeMaterializeActivities{}
	registerCubeMaterializeTestActs(env, acts)

	env.OnActivity(ActCubeValidateAndPlan, mock.Anything, req).Return(plan, nil)

	env.ExecuteWorkflow(CubeMaterializeWorkflow, req)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result CubeMaterializeWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.True(t, result.Noop)
	require.False(t, result.HotApplied)
	require.False(t, result.ColdCommitted)
	require.Equal(t, plan.NoopReason, result.NoopReason)
}

func TestCubeMaterializeWorkflow_ColdFailureCompensatesHot(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	req := CubeMaterializeRequest{
		TenantID: "t1",
		CubeID:   "c1",
		Grain:    []string{"day"},
	}
	plan := &CubeMaterializePlan{
		TenantID:            req.TenantID,
		CubeID:              req.CubeID,
		ContractVersion:     1,
		GrainHash:           GrainHash(req.Grain),
		NodeID:              "33333333-3333-3333-3333-333333333333",
		AttemptID:           "attempt-fail-cold",
		MaterializationName: "cube_fail",
		TargetDatabase:      "tenant_t1",
		IcebergTable:        "iceberg_catalog.cubes.cube_fail",
	}
	hot := &CubeMaterializeHotResult{
		MaterializationName: plan.MaterializationName,
		TargetDatabase:      plan.TargetDatabase,
		AppliedDDL:          true,
		RowCount:            7,
		CommittedAt:         time.Now().UTC(),
	}

	acts := &CubeMaterializeActivities{}
	registerCubeMaterializeTestActs(env, acts)

	env.OnActivity(ActCubeValidateAndPlan, mock.Anything, req).Return(plan, nil)
	env.OnActivity(ActCubeBeginAttempt, mock.Anything, plan).Return(nil)
	env.OnActivity(ActCubeApplyHot, mock.Anything, plan).Return(hot, nil)
	env.OnActivity(ActCubeApplyCold, mock.Anything, plan, hot).Return(nil, errors.New("iceberg catalog unavailable"))
	env.OnActivity(ActCubeCompensateHot, mock.Anything, plan).Return(nil)
	env.OnActivity(ActCubeFailAttempt, mock.Anything, mock.MatchedBy(func(in CubeFailAttemptInput) bool {
		return in.Plan != nil && in.Plan.AttemptID == "attempt-fail-cold" &&
			containsStr(in.ErrorMessage, "iceberg catalog unavailable")
	})).Return(nil)

	env.ExecuteWorkflow(CubeMaterializeWorkflow, req)
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	// Temporal discards the workflow return value on error; the proof that cold
	// failure compensated hot is that CompensateHot + FailAttempt were invoked
	// (mocks would fail the test otherwise) and CompleteDualCommit was not.
	env.AssertExpectations(t)
}

func TestCubeMaterializeWorkflow_HotFailureMarksFailed(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	req := CubeMaterializeRequest{
		TenantID: "t1",
		CubeID:   "c1",
		Grain:    []string{"day"},
	}
	plan := &CubeMaterializePlan{
		TenantID:            req.TenantID,
		CubeID:              req.CubeID,
		ContractVersion:     1,
		GrainHash:           GrainHash(req.Grain),
		NodeID:              "33333333-3333-3333-3333-333333333333",
		AttemptID:           "attempt-fail",
		MaterializationName: "cube_fail",
		TargetDatabase:      "tenant_t1",
	}

	acts := &CubeMaterializeActivities{}
	registerCubeMaterializeTestActs(env, acts)

	env.OnActivity(ActCubeValidateAndPlan, mock.Anything, req).Return(plan, nil)
	env.OnActivity(ActCubeBeginAttempt, mock.Anything, plan).Return(nil)
	env.OnActivity(ActCubeApplyHot, mock.Anything, plan).Return(nil, errors.New("starrocks down"))
	env.OnActivity(ActCubeFailAttempt, mock.Anything, mock.MatchedBy(func(in CubeFailAttemptInput) bool {
		return in.Plan != nil && in.Plan.AttemptID == "attempt-fail" &&
			containsStr(in.ErrorMessage, "starrocks down")
	})).Return(nil)

	env.ExecuteWorkflow(CubeMaterializeWorkflow, req)
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func TestCubeMaterializeWorkflow_ExtractThenDualCommit(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	req := CubeMaterializeRequest{
		TenantID: "t1",
		CubeID:   "c1",
		Grain:    []string{"account_id"},
	}
	plan := &CubeMaterializePlan{
		TenantID:            req.TenantID,
		CubeID:              req.CubeID,
		ContractVersion:     1,
		GrainHash:           GrainHash(req.Grain),
		NodeID:              "44444444-4444-4444-4444-444444444444",
		AttemptID:           "attempt-extract",
		MaterializationName: "cube_fed",
		TargetDatabase:      "tenant_t1",
		IcebergTable:        "lakekeeper_iceberg.cubes.cube_fed",
		SourceTable:         "pg_alpha.oms.position AS pos\nINNER JOIN pg_alpha.oms.account AS acct ON pos.a = acct.a",
		DDL:                 "CREATE MATERIALIZED VIEW cube_fed AS SELECT 1 FROM pg_alpha.oms.position AS pos\nINNER JOIN pg_alpha.oms.account AS acct ON pos.a = acct.a GROUP BY 1;",
		ExtractEnabled:      true,
		FederationSources: []FederationSourcePlan{
			{Alias: "pos", DrivingTable: "pg_alpha.oms.position"},
			{Alias: "acct", DrivingTable: "pg_alpha.oms.account"},
		},
	}
	extractedPlan := *plan
	extractedPlan.ExtractApplied = true
	extractedPlan.StagingTables = []string{"`tenant_t1`.`cube_ext_x_y_pos`", "`tenant_t1`.`cube_ext_x_y_acct`"}
	extractedPlan.SourceTable = "`tenant_t1`.`cube_ext_x_y_pos` AS pos\nINNER JOIN `tenant_t1`.`cube_ext_x_y_acct` AS acct ON pos.a = acct.a"
	extracted := &CubeExtractResult{Plan: &extractedPlan, StagingTables: extractedPlan.StagingTables}

	hot := &CubeMaterializeHotResult{
		MaterializationName: plan.MaterializationName,
		TargetDatabase:      plan.TargetDatabase,
		AppliedDDL:          true,
		RowCount:            3,
		CommittedAt:         time.Now().UTC(),
	}
	cold := &CubeMaterializeColdResult{
		IcebergTable: plan.IcebergTable,
		Applied:      true,
		RowCount:     3,
		CommittedAt:  time.Now().UTC(),
	}

	acts := &CubeMaterializeActivities{}
	registerCubeMaterializeTestActs(env, acts)

	env.OnActivity(ActCubeValidateAndPlan, mock.Anything, req).Return(plan, nil)
	env.OnActivity(ActCubeBeginAttempt, mock.Anything, plan).Return(nil)
	env.OnActivity(ActCubeExtractSources, mock.Anything, plan).Return(extracted, nil)
	env.OnActivity(ActCubeApplyHot, mock.Anything, mock.MatchedBy(func(p *CubeMaterializePlan) bool {
		return p != nil && p.ExtractApplied && len(p.StagingTables) == 2
	})).Return(hot, nil)
	env.OnActivity(ActCubeApplyCold, mock.Anything, mock.Anything, hot).Return(cold, nil)
	env.OnActivity(ActCubeCompleteDualCommit, mock.Anything, mock.Anything, hot, cold).Return(nil)
	env.OnActivity(ActCubeDropStaging, mock.Anything, mock.MatchedBy(func(p *CubeMaterializePlan) bool {
		return p != nil && len(p.StagingTables) == 2
	})).Return(nil)

	env.ExecuteWorkflow(CubeMaterializeWorkflow, req)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}
