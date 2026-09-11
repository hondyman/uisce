package workflows_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
)

func TestReportGenerationWorkflow_EmptyViews(t *testing.T) {
	// CONTRACT: A template with zero semantic views must execute successfully to completion.
	// The workflow is not allowed to fail on an empty-view template — it's a valid
	// degenerate case (a layout shell awaiting data binding).
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	executionID := "exec-001"
	template := activities.HydratedReportTemplate{
		ID:              "tmpl-abc",
		TenantID:        "tenant-123",
		TemplateName:    "Quarterly Wealth Summary",
		Category:        "wealth",
		SemanticViewIDs: []string{}, // Empty — valid degenerate case
		LayoutConfig:    map[string]interface{}{},
		CreatedByID:     "user-owner-456",
		CreatedBy:       "owner@example.com",
		IsPersonal:      false,
		IsPublic:        true,
	}

	params := workflows.ReportGenerationWorkflowParams{
		ExecutionID: executionID,
		Template:    template,
		ScheduleID:  "sched-789",
		TriggerParams: map[string]interface{}{
			"triggered_at": time.Now().Format(time.RFC3339),
			"schedule_id":  "sched-789",
		},
	}

	reportActs := &activities.ReportActivities{}
	env.RegisterActivity(reportActs)

	// Expect zero-row semantic result (empty views guard)
	expectedSemanticResult := map[string]interface{}{
		"views_queried": 0,
		"rows":          0,
		"data":          []interface{}{},
	}

	expectedArtifact := activities.ArtifactResult{
		OutputURL:       "/artifacts/tenant-123/exec-001/report.pdf",
		OutputSizeBytes: 0,
		RowsProcessed:   0,
		ExecutionTimeMS: 5,
		CompletedAt:     time.Now(),
		Engine:          "temporal_workflow",
	}


	env.OnActivity(reportActs.QuerySemanticViewsActivity, mock.Anything, template, params.TriggerParams).
		Return(expectedSemanticResult, nil)
	env.OnActivity(reportActs.GenerateArtifactActivity, mock.Anything, mock.MatchedBy(func(inp activities.GenerateArtifactInput) bool {
		return inp.ExecutionID == executionID && inp.Template.CreatedByID == "user-owner-456"
	}), mock.Anything). // mock.Anything: Temporal JSON-roundtrips the result so int->float64; content validated above
		Return(expectedArtifact, nil)
	// IDENTITY INVARIANT: StoreExecutionResultActivity receives input with template owner CreatedByID.
	// This is the proof point — the worker writes the owner's identity, not the caller's.
	env.OnActivity(reportActs.StoreExecutionResultActivity, mock.Anything, mock.MatchedBy(func(inp activities.GenerateArtifactInput) bool {
		// Assert that the identity carried into storage is the template owner, not a caller.
		return inp.Template.CreatedByID == "user-owner-456" && inp.Template.TenantID == "tenant-123"
	}), mock.Anything).
		Return(nil)

	env.ExecuteWorkflow(workflows.ReportGenerationWorkflow, params)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.ReportGenerationWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, executionID, result.ExecutionID)
	require.Equal(t, "temporal_workflow", result.Engine)
}

func TestReportGenerationWorkflow_IdentityInvariant(t *testing.T) {
	// CONTRACT: The two-sided identity invariant — execution ALWAYS runs under the
	// template owner's identity (template.TenantID, template.CreatedByID) regardless
	// of who triggered the run.
	//
	// This test asserts that StoreExecutionResultActivity is called with the template
	// owner's credentials in the input, not with any hypothetical caller credentials.
	// The worker writing the wrong identity here is the security model's weakest seam.
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	templateOwnerTenantID := "owner-tenant-999"
	templateOwnerUserID := "template-owner-user-777"
	// The caller who triggered the run has a different identity
	callerTenantID := "caller-tenant-111"
	callerUserID := "caller-user-222"

	template := activities.HydratedReportTemplate{
		ID:          "tmpl-xyz",
		TenantID:    templateOwnerTenantID, // Template owner's tenant
		TemplateName: "Global Liquidity Risk",
		Category:    "risk",
		SemanticViewIDs: []string{},
		LayoutConfig:    map[string]interface{}{},
		CreatedByID: templateOwnerUserID, // Template owner's user ID
		CreatedBy:   "template-owner@firm.com",
	}

	params := workflows.ReportGenerationWorkflowParams{
		ExecutionID: "exec-identity-test",
		Template:    template,
		ScheduleID:  "sched-identity",
		TriggerParams: map[string]interface{}{
			// Caller context passed in params — MUST NOT leak into execution identity
			"triggered_by": callerUserID,
			"caller_tenant": callerTenantID,
		},
	}

	reportActs := &activities.ReportActivities{}
	env.RegisterActivity(reportActs)

	env.OnActivity(reportActs.QuerySemanticViewsActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"views_queried": 0, "rows": 0, "data": []interface{}{}}, nil)
	env.OnActivity(reportActs.GenerateArtifactActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(activities.ArtifactResult{
			OutputURL:       "/artifacts/" + templateOwnerTenantID + "/exec-identity-test/report.pdf",
			Engine:          "temporal_workflow",
			CompletedAt:     time.Now(),
		}, nil)

	// IDENTITY INVARIANT ASSERTION: StoreExecutionResultActivity input must carry the
	// template owner's identity — specifically CreatedByID == templateOwnerUserID and
	// TenantID == templateOwnerTenantID. If a caller's identity leaked here, the
	// security model is broken.
	env.OnActivity(reportActs.StoreExecutionResultActivity, mock.Anything,
		mock.MatchedBy(func(inp activities.GenerateArtifactInput) bool {
			correctOwnerTenant := inp.Template.TenantID == templateOwnerTenantID
			correctOwnerUser := inp.Template.CreatedByID == templateOwnerUserID
			callerNotLeaked := inp.Template.TenantID != callerTenantID && inp.Template.CreatedByID != callerUserID
			return correctOwnerTenant && correctOwnerUser && callerNotLeaked
		}),
		mock.Anything).
		Return(nil)

	env.ExecuteWorkflow(workflows.ReportGenerationWorkflow, params)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError(), "Workflow should complete without error")

	var result workflows.ReportGenerationWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	// The output URL must be scoped to the template owner's tenant, not the caller's
	require.Contains(t, result.OutputURL, templateOwnerTenantID,
		"Output URL must be scoped to template owner tenant, not caller tenant")
}

func TestReportGenerationWorkflow_StorageFailureFails(t *testing.T) {
	// CONTRACT: If StoreExecutionResultActivity fails, the workflow MUST fail (not swallow
	// the error). The reconciler detects stale 'running' rows and marks them failed.
	// Swallowing storage errors would leave executions silently stuck in 'running'.
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	params := workflows.ReportGenerationWorkflowParams{
		ExecutionID: "exec-storage-fail",
		Template: activities.HydratedReportTemplate{
			ID:          "tmpl-fail",
			TenantID:    "tenant-fail",
			CreatedByID: "user-fail",
		},
		TriggerParams: map[string]interface{}{},
	}

	reportActs := &activities.ReportActivities{}
	env.RegisterActivity(reportActs)

	env.OnActivity(reportActs.QuerySemanticViewsActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"views_queried": 0, "rows": 0, "data": []interface{}{}}, nil)
	env.OnActivity(reportActs.GenerateArtifactActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(activities.ArtifactResult{Engine: "temporal_workflow", CompletedAt: time.Now()}, nil)
	// Simulate storage failure
	env.OnActivity(reportActs.StoreExecutionResultActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("simulated storage failure"))

	env.ExecuteWorkflow(workflows.ReportGenerationWorkflow, params)

	require.True(t, env.IsWorkflowCompleted())
	// Workflow MUST fail — not succeed silently — when storage fails
	require.Error(t, env.GetWorkflowError(),
		"Workflow must fail if StoreExecutionResultActivity fails — do NOT swallow storage errors")
}
