package workers

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/activities"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/workflows"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// RegisterSecurityWorkflows registers security-related workflows and activities with a Temporal worker.
func RegisterSecurityWorkflows(w worker.Worker, db *sqlx.DB) {
	// Create security service components
	securityRepo := security.NewAccessRuleRepository(db)
	securityValidator := security.NewDslValidator(db)
	securityAnalyzer := security.NewImpactAnalyzer(db, securityRepo)

	// Create activities handler
	securityActivities := activities.NewAccessRuleActivities(
		securityRepo,
		securityValidator,
		securityAnalyzer,
	)

	// Register workflow
	w.RegisterWorkflow(workflows.PromoteAccessRuleWorkflow)

	// Register all activities
	w.RegisterActivity(securityActivities.LoadRuleActivity)
	w.RegisterActivity(securityActivities.ValidateRuleSyntaxActivity)
	w.RegisterActivity(securityActivities.ImpactAnalysisActivity)
	w.RegisterActivity(securityActivities.RunSecurityTestsActivity)
	w.RegisterActivity(securityActivities.PromoteRuleActivity)
	w.RegisterActivity(securityActivities.EmitAuditAndInvalidateCacheActivity)
}

// StartSecurityWorker creates and starts a dedicated Temporal worker for security workflows.
func StartSecurityWorker(ctx context.Context, db *sqlx.DB, temporalHost string) error {
	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		return fmt.Errorf("unable to create Temporal client: %w", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, "security-task-queue", worker.Options{})

	// Register workflows and activities
	RegisterSecurityWorkflows(w, db)

	fmt.Println("Security Temporal worker started")

	// Start listening to the Task Queue
	return w.Run(worker.InterruptCh())
}
