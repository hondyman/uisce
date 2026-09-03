package workflows

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// BPWorker runs the Business Process Temporal workflows and activities
type BPWorker struct {
	client client.Client
	worker worker.Worker
}

// BPTaskQueue is the task queue for cmd/bp-worker (this file's NewBPWorker).
// IMPORTANT: cmd/bp-worker is not deployed anywhere today — no
// docker-compose file builds or runs it (grep confirms; backend/Dockerfile
// and every compose service that runs "./worker" build cmd/worker instead).
// A caller that wants to reach the actual production BP Temporal worker
// must use DeployedBPTaskQueue below, not this constant.
const BPTaskQueue = "bp-framework-queue"

// DeployedBPTaskQueue is the task queue the real, deployed BP Temporal
// worker (cmd/worker/main.go, built into backend/Dockerfile as "./worker"
// and run by docker-compose.hybrid.yml / docker-compose.starrocks.yml)
// actually listens on. cmd/worker registers RunStoredWorkflow and
// InterpreterWorkflow on this queue. Any code starting one of those
// workflows in production must target this constant — targeting
// BPTaskQueue instead sends the workflow to a queue nothing is listening
// on, where it will sit as an orphaned open workflow forever.
const DeployedBPTaskQueue = "bp_queue"

// NewBPWorker creates a new Business Process worker. deps may be nil, in
// which case activities requiring real dependencies (currently just
// ActivityCalculation, see calc_activities.go) are not registered — a
// workflow definition using a "Calculation" node against a worker started
// without deps will fail at runtime with "unable to find activityType",
// same as before ActivityCalculation existed at all.
func NewBPWorker(c client.Client, deps *ActivityDeps) *BPWorker {
	w := worker.New(c, BPTaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	// Register Workflows
	w.RegisterWorkflow(InterpreterWorkflow)
	w.RegisterWorkflow(RunStoredWorkflow)

	// ==================== LLM ACTIVITIES ====================
	w.RegisterActivity(LLMInterpretationActivity)
	w.RegisterActivity(LLMClassificationActivity)
	w.RegisterActivity(LLMDraftingActivity)
	w.RegisterActivity(LLMRecommendationActivity)
	w.RegisterActivity(LLMExplanationActivity)

	// ==================== ROUTING ACTIVITIES ====================
	w.RegisterActivity(RoutingExpressionActivity)
	w.RegisterActivity(LLMRoutingActivity)

	// ==================== CONDITION ACTIVITIES ====================
	w.RegisterActivity(SemanticConditionActivity)
	w.RegisterActivity(LLMConditionActivity)
	w.RegisterActivity(PolicyConditionActivity)

	// ==================== AUDIT ACTIVITIES ====================
	// ==================== AUDIT ACTIVITIES ====================
	w.RegisterActivity(RecordAuditEventActivity)
	w.RegisterActivity(ActivityCreateHumanTask)

	// ==================== EXTERNAL ACTIVITIES ====================
	w.RegisterActivity(ActivityCreateExternalTask)
	w.RegisterActivity(ActivityUpdateExternalTask)
	w.RegisterActivity(ActivityCloseExternalTask)
	w.RegisterActivity(ActivityWaitForExternalCallback)

	// ==================== SYSTEM ACTIVITIES ====================
	w.RegisterActivity(ActivityServiceCall)
	w.RegisterActivity(ActivitySemanticRollup)
	w.RegisterActivity(ActivityDataValidation)
	w.RegisterActivity(ActivityGenerateReport)
	w.RegisterActivity(ActivityNotification)

	// ==================== CALC ENGINE ACTIVITIES ====================
	// Requires deps (DB-backed SemanticCalculationService) — see
	// calc_activities.go. Without deps, a "Calculation" workflow node
	// fails at runtime rather than silently using a fake/stub result.
	if deps != nil && deps.CalcService != nil {
		w.RegisterActivity(deps.ActivityCalculation)
	} else {
		log.Println("WARNING: BP worker started without ActivityDeps.CalcService — \"Calculation\" workflow nodes will fail at runtime")
	}

	// Note: The following activities are referenced in dynamic_bp_workflow.go
	// but use string-based activity calls. They will be registered when implemented.
	// For now, we use placeholder stub functions.

	// Stub activities for publish event broker types
	// Prefer Kafka/Redpanda by default; keep RabbitMQ stub for legacy compatibility
	w.RegisterActivity(stubActivityPublishKafka)
	w.RegisterActivity(stubActivityPublishRabbitMQ) // DEPRECATED: legacy AMQP stub
	w.RegisterActivity(stubActivitySendAlert)
	w.RegisterActivity(stubActivityExecuteSteps)

	log.Printf("Registered BP Framework workflows and activities on queue: %s", BPTaskQueue)

	return &BPWorker{
		client: c,
		worker: w,
	}
}

// Start starts the BP worker
func (w *BPWorker) Start() error {
	log.Println("Starting BP Framework Temporal worker...")
	return w.worker.Start()
}

// Stop stops the BP worker
func (w *BPWorker) Stop() {
	log.Println("Stopping BP Framework Temporal worker...")
	w.worker.Stop()
}

// Run runs the worker and blocks until interrupted
func (w *BPWorker) Run() error {
	log.Println("Running BP Framework Temporal worker (blocking)...")
	return w.worker.Run(worker.InterruptCh())
}
