package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// HireEmployeeParams contains parameters for hiring an employee
type HireEmployeeParams struct {
	FirstName  string                 `json:"first_name"`
	LastName   string                 `json:"last_name"`
	Email      string                 `json:"email"`
	Department string                 `json:"department"`
	JobTitle   string                 `json:"job_title"`
	ManagerID  string                 `json:"manager_id"`
	StartDate  time.Time              `json:"start_date"`
	Salary     float64                `json:"salary"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// HireEmployeeResult contains the result of hiring workflow
type HireEmployeeResult struct {
	EmployeeID         string    `json:"employee_id"`
	Status             string    `json:"status"` // hired, rejected, pending
	RejectionReason    string    `json:"rejection_reason,omitempty"`
	ProvisionedSystems []string  `json:"provisioned_systems"`
	CompletedAt        time.Time `json:"completed_at"`
}

// HireEmployeeWorkflow orchestrates the employee hiring process
// Steps:
// 1. Create employee record
// 2. Request manager approval (with timeout)
// 3. Request HR approval (with timeout)
// 4. Provision systems (email, Slack, GitHub, etc.)
// 5. Send welcome email
// 6. Schedule onboarding
func HireEmployeeWorkflow(ctx workflow.Context, params HireEmployeeParams) (*HireEmployeeResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting HireEmployee workflow",
		"name", params.FirstName+" "+params.LastName,
		"email", params.Email)

	// Activity options with retries
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Create Employee Record
	logger.Info("Step 1: Creating employee record")
	var employeeID string
	err := workflow.ExecuteActivity(ctx, "CreateEmployeeActivity", params).Get(ctx, &employeeID)
	if err != nil {
		logger.Error("Failed to create employee record", "error", err)
		return nil, fmt.Errorf("failed to create employee: %w", err)
	}
	logger.Info("Employee record created", "employee_id", employeeID)

	// Step 2: Manager Approval (with 24h timeout)
	logger.Info("Step 2: Requesting manager approval", "manager_id", params.ManagerID)
	var managerApproved bool

	approvalCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 24 * time.Hour, // Long timeout for human approval
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Minute,
			MaximumAttempts: 1, // No retries for approval
		},
	})

	err = workflow.ExecuteActivity(approvalCtx, "RequestManagerApprovalActivity",
		employeeID, params.ManagerID).Get(approvalCtx, &managerApproved)
	if err != nil {
		logger.Error("Manager approval activity failed", "error", err)
		return nil, err
	}

	if !managerApproved {
		logger.Info("Manager rejected hire request")
		return &HireEmployeeResult{
			EmployeeID:      employeeID,
			Status:          "rejected",
			RejectionReason: "Manager declined",
			CompletedAt:     time.Now(),
		}, nil
	}
	logger.Info("Manager approved hire request")

	// Step 3: HR Approval (with 48h timeout)
	logger.Info("Step 3: Requesting HR approval")
	var hrApproved bool

	err = workflow.ExecuteActivity(approvalCtx, "RequestHRApprovalActivity",
		employeeID).Get(approvalCtx, &hrApproved)
	if err != nil {
		logger.Error("HR approval activity failed", "error", err)
		return nil, err
	}

	if !hrApproved {
		logger.Info("HR rejected hire request")
		return &HireEmployeeResult{
			EmployeeID:      employeeID,
			Status:          "rejected",
			RejectionReason: "HR declined",
			CompletedAt:     time.Now(),
		}, nil
	}
	logger.Info("HR approved hire request")

	// Step 4: Provision Systems (parallel execution for speed)
	logger.Info("Step 4: Provisioning systems")
	var systems []string

	provisionCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 5,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 2,
			MaximumAttempts:    5, // Systems can be flaky
		},
	})

	// Execute provisioning activities in parallel
	var futures []workflow.Future
	systemTypes := []string{"Email", "Slack", "GitHub", "Jira", "AWS"}

	for _, systemType := range systemTypes {
		future := workflow.ExecuteActivity(provisionCtx, "ProvisionSystemActivity",
			employeeID, systemType, params)
		futures = append(futures, future)
	}

	// Wait for all provisioning to complete
	for i, future := range futures {
		var result map[string]interface{}
		if err := future.Get(provisionCtx, &result); err != nil {
			logger.Warn("Failed to provision system", "system", systemTypes[i], "error", err)
			// Continue with other systems
		} else {
			systems = append(systems, systemTypes[i])
			logger.Info("System provisioned successfully", "system", systemTypes[i])
		}
	}

	// Step 5: Send Welcome Email (non-critical)
	logger.Info("Step 5: Sending welcome email")
	err = workflow.ExecuteActivity(ctx, "SendWelcomeEmailActivity",
		employeeID, params.Email, params.StartDate).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to send welcome email, continuing", "error", err)
		// Non-critical, don't fail workflow
	}

	// Step 6: Schedule Onboarding (async, fire and forget)
	logger.Info("Step 6: Scheduling onboarding")
	workflow.ExecuteActivity(ctx, "ScheduleOnboardingActivity",
		employeeID, params.StartDate).Get(ctx, nil)

	logger.Info("HireEmployee workflow completed successfully",
		"employee_id", employeeID,
		"systems_provisioned", len(systems))

	return &HireEmployeeResult{
		EmployeeID:         employeeID,
		Status:             "hired",
		ProvisionedSystems: systems,
		CompletedAt:        time.Now(),
	}, nil
}

// HandleApprovalSignal handles manual approval signals
func HandleApprovalSignal(ctx workflow.Context, approved bool, reason string) {
	// This would be called when someone approves/rejects from the UI
	logger := workflow.GetLogger(ctx)
	logger.Info("Received approval signal", "approved", approved, "reason", reason)
}
