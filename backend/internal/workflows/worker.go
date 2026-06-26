package workflows

import (
	"log"

	"github.com/hondyman/semlayer/backend/internal/cbo"
	"github.com/hondyman/semlayer/backend/internal/observability/activities"
	obsWorkflows "github.com/hondyman/semlayer/backend/internal/observability/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Worker runs the Temporal workflows and activities
type Worker struct {
	client client.Client
	worker worker.Worker
}

func NewWorker(c client.Client, sloEvaluator *cbo.SLOEvaluator, sloProvider *cbo.DBSLOProvider) *Worker {
	w := worker.New(c, "wealth-stream-queue", worker.Options{})

	// Register Workflows
	w.RegisterWorkflow(DriftMonitorWorkflow)
	w.RegisterWorkflow(obsWorkflows.SLOEvaluationWorkflow)

	// Register Activities
	dataActivities := &Activities{}
	w.RegisterActivity(dataActivities.GetPortfolioData)
	w.RegisterActivity(dataActivities.CalculateDrift)
	w.RegisterActivity(dataActivities.SendAlert)
	w.RegisterActivity(dataActivities.ExecuteTrade)
	w.RegisterActivity(ComplianceActivity)

	if sloEvaluator != nil && sloProvider != nil {
		sloActivities := activities.NewSLOActivities(sloEvaluator, sloProvider)
		w.RegisterActivity(sloActivities.LoadActiveSLOsActivity)
		w.RegisterActivity(sloActivities.EvaluateSLOActivity)
		w.RegisterActivity(sloActivities.HandleSLOViolationActivity)
	}

	return &Worker{
		client: c,
		worker: w,
	}
}

func (w *Worker) Start() error {
	log.Println("Starting Temporal worker...")
	return w.worker.Start()
}

func (w *Worker) Stop() {
	log.Println("Stopping Temporal worker...")
	w.worker.Stop()
}
