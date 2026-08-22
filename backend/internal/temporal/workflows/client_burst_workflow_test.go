package workflows_test

import (
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestClientBurstReportWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	params := workflows.ClientBurstReportWorkflowParams{
		TenantID:   "tenant-123",
		ScheduleID: "sched-456",
		EvalTime:   time.Now(),
	}

	evalResp := struct {
		Allowed       bool      `json:"allowed"`
		EffectiveDate time.Time `json:"effective_date"`
		BurstDim      string    `json:"burst_dimension"`
		ExportFormat  string    `json:"export_format"`
	}{
		Allowed:       true,
		EffectiveDate: params.EvalTime,
		BurstDim:      "client_id",
		ExportFormat:  "PDF",
	}

	reportActs := &activities.ReportActivities{}
	env.RegisterActivity(reportActs)

	env.OnActivity(reportActs.EvaluateReportCalendarActivity, mock.Anything, params.TenantID, params.ScheduleID, mock.Anything).Return(evalResp, nil)
	env.OnActivity(reportActs.ResolveClientSlicesActivity, mock.Anything, params.TenantID, "client_id").Return([]string{"c1", "c2"}, nil)
	env.OnActivity(reportActs.InitBurstBatchActivity, mock.Anything, params.TenantID, params.ScheduleID, mock.Anything).Return("batch-789", nil)
	env.OnActivity(reportActs.RenderAndStoreClientArtifactActivity, mock.Anything, params.TenantID, "batch-789", params.ScheduleID, "c1", "PDF", mock.Anything).Return(true, nil)
	env.OnActivity(reportActs.RenderAndStoreClientArtifactActivity, mock.Anything, params.TenantID, "batch-789", params.ScheduleID, "c2", "PDF", mock.Anything).Return(true, nil)
	env.OnActivity(reportActs.FinalizeBurstBatchActivity, mock.Anything, params.TenantID, "batch-789", "COMPLETED", 2, 2, 0).Return(nil)
	env.OnActivity(reportActs.DispatchClientDistributionsActivity, mock.Anything, params.TenantID, "batch-789").Return(nil)

	env.ExecuteWorkflow(workflows.ClientBurstReportWorkflow, params)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.ClientBurstReportWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "batch-789", result.BatchID)
	require.Equal(t, 2, result.TotalClients)
	require.Equal(t, 2, result.SuccessfulRenders)
	require.Equal(t, 0, result.FailedRenders)
	require.Equal(t, "COMPLETED", result.Status)
}
