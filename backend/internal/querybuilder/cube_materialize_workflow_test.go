package querybuilder

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func containsStr(s, sub string) bool { return strings.Contains(s, sub) }

func registerCubeMaterializeTestActs(env *testsuite.TestWorkflowEnvironment, acts *CubeMaterializeActivities) {
	env.RegisterActivityWithOptions(acts.CubeValidateAndPlan, activity.RegisterOptions{Name: ActCubeValidateAndPlan})
	env.RegisterActivityWithOptions(acts.CubeBeginAttempt, activity.RegisterOptions{Name: ActCubeBeginAttempt})
	env.RegisterActivityWithOptions(acts.CubeApplyHot, activity.RegisterOptions{Name: ActCubeApplyHot})
	env.RegisterActivityWithOptions(acts.CubeCompleteAttempt, activity.RegisterOptions{Name: ActCubeCompleteAttempt})
	env.RegisterActivityWithOptions(acts.CubeFailAttempt, activity.RegisterOptions{Name: ActCubeFailAttempt})
}

func TestCubeMaterializeWorkflow_HappyPath(t *testing.T) {
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
		DDL:                 "CREATE MATERIALIZED VIEW cube_t_account_smoke AS SELECT 1;",
	}
	hot := &CubeMaterializeHotResult{
		MaterializationName: plan.MaterializationName,
		TargetDatabase:      plan.TargetDatabase,
		AppliedDDL:          true,
		RowCount:            42,
	}

	acts := &CubeMaterializeActivities{}
	registerCubeMaterializeTestActs(env, acts)

	env.OnActivity(ActCubeValidateAndPlan, mock.Anything, req).Return(plan, nil)
	env.OnActivity(ActCubeBeginAttempt, mock.Anything, plan).Return(nil)
	env.OnActivity(ActCubeApplyHot, mock.Anything, plan).Return(hot, nil)
	env.OnActivity(ActCubeCompleteAttempt, mock.Anything, plan, hot).Return(nil)

	env.ExecuteWorkflow(CubeMaterializeWorkflow, req)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result CubeMaterializeWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.True(t, result.HotApplied)
	require.Equal(t, int64(42), result.RowCount)
	require.Equal(t, plan.AttemptID, result.AttemptID)
	require.False(t, result.Noop)
	require.False(t, result.ColdCommitted) // CUBE-1.3
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
		NoopReason:      "content_hash unchanged and grain already Active",
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
	require.Equal(t, plan.NoopReason, result.NoopReason)
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
			(in.ErrorMessage == "starrocks down" || containsStr(in.ErrorMessage, "starrocks down"))
	})).Return(nil)

	env.ExecuteWorkflow(CubeMaterializeWorkflow, req)
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}
