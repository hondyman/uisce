package worker

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/calc-engine/activities"
	"github.com/hondyman/semlayer/backend/internal/calc-engine/workflows"
	"github.com/hondyman/semlayer/backend/internal/logging"
	local_temporal "github.com/hondyman/semlayer/backend/internal/temporal"
	local_activities "github.com/hondyman/semlayer/backend/internal/temporal/activities"
	kafka "github.com/segmentio/kafka-go"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// TemporalWorkerConfig holds configuration for Temporal worker initialization
type TemporalWorkerConfig struct {
	TemporalHostPort string
	TaskQueue        string
	DB               *sql.DB
	KafkaBrokers     string // e.g. "redpanda:9092"
	RabbitURL        string // Deprecated: Kept for compatibility if caller hasn't updated
	AuditService     *audit.TrinoAuditService
}

// InitializeTemporalWorker sets up Temporal client, registers workflows/activities, and starts worker
func InitializeTemporalWorker(cfg TemporalWorkerConfig) (client.Client, worker.Worker, error) {
	// Create Temporal client
	temporalClient, err := client.Dial(client.Options{
		HostPort: cfg.TemporalHostPort,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	// Create worker
	w := worker.New(temporalClient, cfg.TaskQueue, worker.Options{})

	// Initialize Kafka writer if brokers provided
	var kafkaWriter *kafka.Writer
	brokers := cfg.KafkaBrokers
	// Fallback to RabbitURL value if set and KafkaBrokers not (assuming migration hasn't updated config source yet, though it should)
	// Actually, RabbitURL is amqp://... so we can't use it.
	// We'll rely on cfg.KafkaBrokers.
	if brokers == "" {
		brokers = "redpanda:9092" // Default
	}

	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(strings.Split(brokers, ",")...),
		Balancer: &kafka.LeastBytes{},
	}

	// Initialize activity config with dependencies
	activityCfg := &activities.ActivityConfig{
		DB:           cfg.DB,
		KafkaBrokers: brokers,
		KafkaWriter:  kafkaWriter,
	}
	activities.Initialize(activityCfg)

	// Register workflow
	w.RegisterWorkflow(workflows.MetricComputeWorkflow)
	w.RegisterWorkflow(local_temporal.GoldCopyConnectionPropagation)

	// Register activities
	w.RegisterActivity(activities.UpsertRunStatus)
	w.RegisterActivity(activities.ComputeAndMergePoP)
	w.RegisterActivity(activities.ComputeAndMergeAnomalies)
	w.RegisterActivity(activities.PublishCompletionEvent)
	w.RegisterActivity(activities.RefreshCubePartitions)

	// Register Gold Copy Activities
	var auditSvc local_activities.AuditService
	// Use injected audit service if available, otherwise try to init legacy or skip
	if cfg.AuditService != nil {
		auditSvc = cfg.AuditService
	} else {
		// Fallback or skip
		// auditSvc, err := audit.NewIcebergAuditService()
		logging.GetLogger().Sugar().Warn("⚠️ No Audit Service provided to Temporal Worker. Audit logging disabled.")
	}

	goldCopyActs := local_activities.NewGoldCopyActivities(cfg.DB, logging.GetLogger().Sugar(), auditSvc)
	w.RegisterActivity(goldCopyActs.PropagateConnectionActivity)
	w.RegisterActivity(goldCopyActs.LogConnectionAuditActivity)

	// Start worker
	err = w.Start()
	if err != nil {
		temporalClient.Close()
		return nil, nil, fmt.Errorf("failed to start Temporal worker: %w", err)
	}

	log.Printf("✅ Temporal worker started on task queue '%s'", cfg.TaskQueue)
	return temporalClient, w, nil
}

// Shutdown gracefully closes Temporal worker and Kafka writer
func Shutdown(tc client.Client, w worker.Worker, actCfg *activities.ActivityConfig) {
	if w != nil {
		w.Stop()
	}
	if tc != nil {
		tc.Close()
	}
	if actCfg != nil && actCfg.KafkaWriter != nil {
		actCfg.KafkaWriter.Close()
	}
}
