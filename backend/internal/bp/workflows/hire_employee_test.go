package workflows_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hondyman/semlayer/backend/internal/bp/activities"
	"github.com/hondyman/semlayer/backend/internal/bp/workflows"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestHireEmployeeWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Temporal's test environment requires activities to be registered
	// before they can be mocked by name.
	bp := activities.NewBPActivities(nil)
	env.RegisterActivity(bp.CreateEmployeeActivity)
	env.RegisterActivity(bp.RequestManagerApprovalActivity)
	env.RegisterActivity(bp.RequestHRApprovalActivity)
	env.RegisterActivity(bp.ProvisionSystemActivity)
	env.RegisterActivity(bp.SendWelcomeEmailActivity)
	env.RegisterActivity(bp.ScheduleOnboardingActivity)

	// Mock activities
	// Mock activities
	env.OnActivity("CreateEmployeeActivity", mock.Anything, mock.Anything).Return("emp-123", nil)
	env.OnActivity("RequestManagerApprovalActivity", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	env.OnActivity("RequestHRApprovalActivity", mock.Anything, mock.Anything).Return(true, nil)
	env.OnActivity("ProvisionSystemActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		map[string]interface{}{"status": "provisioned"}, nil,
	)
	env.OnActivity("SendWelcomeEmailActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity("ScheduleOnboardingActivity", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Execute workflow
	params := workflows.HireEmployeeParams{
		FirstName:  "John",
		LastName:   "Doe",
		Email:      "john.doe@example.com",
		Department: "Engineering",
		JobTitle:   "Software Engineer",
		ManagerID:  "mgr-123",
		StartDate:  time.Now().AddDate(0, 0, 14),
		Salary:     120000,
	}

	env.ExecuteWorkflow(workflows.HireEmployeeWorkflow, params)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.HireEmployeeResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "emp-123", result.EmployeeID)
	require.Equal(t, "hired", result.Status)
	require.NotEmpty(t, result.ProvisionedSystems)
}

func TestHireEmployeeWorkflow_ManagerRejects(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	bp := activities.NewBPActivities(nil)
	env.RegisterActivity(bp.CreateEmployeeActivity)
	env.RegisterActivity(bp.RequestManagerApprovalActivity)
	env.RegisterActivity(bp.RequestHRApprovalActivity)
	env.RegisterActivity(bp.ProvisionSystemActivity)
	env.RegisterActivity(bp.SendWelcomeEmailActivity)
	env.RegisterActivity(bp.ScheduleOnboardingActivity)

	env.OnActivity("CreateEmployeeActivity", mock.Anything, mock.Anything).Return("emp-123", nil)
	env.OnActivity("RequestManagerApprovalActivity", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)

	params := workflows.HireEmployeeParams{
		FirstName: "Jane",
		LastName:  "Smith",
		Email:     "jane.smith@example.com",
	}

	env.ExecuteWorkflow(workflows.HireEmployeeWorkflow, params)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.HireEmployeeResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "rejected", result.Status)
	require.Contains(t, result.RejectionReason, "Manager declined")
}

func TestHireEmployeeWorkflow_HRRejects(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	bp := activities.NewBPActivities(nil)
	env.RegisterActivity(bp.CreateEmployeeActivity)
	env.RegisterActivity(bp.RequestManagerApprovalActivity)
	env.RegisterActivity(bp.RequestHRApprovalActivity)
	env.RegisterActivity(bp.ProvisionSystemActivity)
	env.RegisterActivity(bp.SendWelcomeEmailActivity)
	env.RegisterActivity(bp.ScheduleOnboardingActivity)

	env.OnActivity("CreateEmployeeActivity", mock.Anything, mock.Anything).Return("emp-123", nil)
	env.OnActivity("RequestManagerApprovalActivity", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	env.OnActivity("RequestHRApprovalActivity", mock.Anything, mock.Anything).Return(false, nil)

	params := workflows.HireEmployeeParams{
		FirstName: "Bob",
		LastName:  "Johnson",
		Email:     "bob.johnson@example.com",
	}

	env.ExecuteWorkflow(workflows.HireEmployeeWorkflow, params)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.HireEmployeeResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "rejected", result.Status)
	require.Contains(t, result.RejectionReason, "HR declined")
}

// ...
func TestHireEmployeeWorkflow_ProvisioningFailure(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	bp := activities.NewBPActivities(nil)
	env.RegisterActivity(bp.CreateEmployeeActivity)
	env.RegisterActivity(bp.RequestManagerApprovalActivity)
	env.RegisterActivity(bp.RequestHRApprovalActivity)
	env.RegisterActivity(bp.ProvisionSystemActivity)
	env.RegisterActivity(bp.SendWelcomeEmailActivity)
	env.RegisterActivity(bp.ScheduleOnboardingActivity)

	env.OnActivity("CreateEmployeeActivity", mock.Anything, mock.Anything).Return("emp-123", nil)
	env.OnActivity("RequestManagerApprovalActivity", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	env.OnActivity("RequestHRApprovalActivity", mock.Anything, mock.Anything).Return(true, nil)

	// Simulate partial provisioning failure
	call := 0
	env.OnActivity("ProvisionSystemActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, empID, systemType string, params interface{}) (map[string]interface{}, error) {
			call++
			if call == 3 { // Fail GitHub provisioning
				return nil, fmt.Errorf("timeout")
			}
			return map[string]interface{}{"status": "provisioned"}, nil
		},
	)

	env.OnActivity("SendWelcomeEmailActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity("ScheduleOnboardingActivity", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	params := workflows.HireEmployeeParams{
		FirstName: "Alice",
		LastName:  "Williams",
		Email:     "alice.williams@example.com",
	}

	env.ExecuteWorkflow(workflows.HireEmployeeWorkflow, params)

	require.True(t, env.IsWorkflowCompleted())

	var result workflows.HireEmployeeResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "hired", result.Status)
	// Should have some systems provisioned despite one failure
	require.NotEmpty(t, result.ProvisionedSystems)
}
