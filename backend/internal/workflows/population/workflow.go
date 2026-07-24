package population

import (
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// KGPopulationInput defines the input for the population workflow
type KGPopulationInput struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	BatchSize    int       `json:"batch_size"`
	ParallelJobs int       `json:"parallel_jobs"`
}

// KGPopulationResult defines the output
type KGPopulationResult struct {
	TenantID             uuid.UUID `json:"tenant_id"`
	EntitiesCreated      int       `json:"entities_created"`
	RelationshipsCreated int       `json:"relationships_created"`
	ProcessingTimeMs     int64     `json:"processing_time_ms"`
}

// KnowledgeGraphPopulationWorkflow orchestrates the extraction and loading of entities into Postgres
func KnowledgeGraphPopulationWorkflow(ctx workflow.Context, input KGPopulationInput) (*KGPopulationResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting KG population workflow (Postgres)", "tenantID", input.TenantID)

	startTime := workflow.Now(ctx)
	result := &KGPopulationResult{TenantID: input.TenantID}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Phase 1: Extract entities (Mocked for now, would call NER)
	// In a real scenario, this would call the Python NER service or an internal Go NER activity
	logger.Info("Phase 1: Extracting entities")
	var extractedEntities []ExtractedEntity
	// Using a stub activity for now to simulate extraction
	err := workflow.ExecuteActivity(ctx, ExtractEntitiesActivity, input.TenantID).Get(ctx, &extractedEntities)
	if err != nil {
		return nil, err
	}

	// Phase 2: Deduplicate and Link (Logic remains similar, just in-memory or DB-assisted)
	logger.Info("Phase 2: Deduplicating entities")
	var linkedEntities []LinkedEntity
	err = workflow.ExecuteActivity(ctx, DeduplicateEntitiesActivity, extractedEntities).Get(ctx, &linkedEntities)
	if err != nil {
		return nil, err
	}

	// Phase 3: Persist Nodes to Postgres (Replacing Neo4j)
	logger.Info("Phase 3: Persisting nodes to Postgres")
	var nodeStats NodeCreationStats
	err = workflow.ExecuteActivity(ctx, PersistNodesPostgresActivity, PersistNodesInput{
		TenantID: input.TenantID,
		Entities: linkedEntities,
	}).Get(ctx, &nodeStats)
	if err != nil {
		return nil, err
	}
	result.EntitiesCreated = nodeStats.NodesCreated

	// Phase 4: Persist Relationships to Postgres (Replacing Neo4j)
	logger.Info("Phase 4: Persisting relationships to Postgres")
	var relStats RelationshipStats
	err = workflow.ExecuteActivity(ctx, PersistRelationshipsPostgresActivity, PersistRelationshipsInput{
		TenantID: input.TenantID,
		Entities: linkedEntities,
	}).Get(ctx, &relStats)
	if err != nil {
		return nil, err
	}
	result.RelationshipsCreated = relStats.RelationshipsCreated

	result.ProcessingTimeMs = workflow.Now(ctx).Sub(startTime).Milliseconds()
	return result, nil
}
