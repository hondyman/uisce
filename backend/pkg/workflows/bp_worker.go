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

const BPTaskQueue = "bp-framework-queue"

// NewBPWorker creates a new Business Process worker
func NewBPWorker(c client.Client) *BPWorker {
	w := worker.New(c, BPTaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	// Register Workflows
	w.RegisterWorkflow(InterpreterWorkflow)

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
