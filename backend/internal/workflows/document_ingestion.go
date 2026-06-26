package workflows

import (
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

// DocumentIngestionWorkflowParam defines the input for the workflow
type DocumentIngestionWorkflowParam struct {
	TenantID   uuid.UUID
	DocumentID uuid.UUID
	SourcePath string
}

// DocumentIngestionWorkflow orchestrates the document processing pipeline
func DocumentIngestionWorkflow(ctx workflow.Context, params DocumentIngestionWorkflowParam) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var activities *DocumentActivities // Used for name resolution only

	// 1. Extract Text
	var text string
	err := workflow.ExecuteActivity(ctx, activities.ExtractTextActivity, params.SourcePath).Get(ctx, &text)
	if err != nil {
		return err
	}

	// 2. Chunk Document
	var chunks []string
	err = workflow.ExecuteActivity(ctx, activities.ChunkDocumentActivity, text).Get(ctx, &chunks)
	if err != nil {
		return err
	}

	// 3. Generate Embeddings
	var embeddings [][]float32
	err = workflow.ExecuteActivity(ctx, activities.GenerateEmbeddingsActivity, chunks).Get(ctx, &embeddings)
	if err != nil {
		return err
	}

	// 4. Store Chunks
	err = workflow.ExecuteActivity(ctx, activities.StoreChunksActivity, params.TenantID, params.DocumentID, chunks, embeddings).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
