package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Edge Queue Worker
// ============================================================================

// EdgeQueueWorker processes pending edge creation requests from the queue
type EdgeQueueWorker struct {
	db           *sqlx.DB
	pollInterval time.Duration
	batchSize    int
	maxRetries   int
	running      bool
}

// NewEdgeQueueWorker creates a new edge queue worker
func NewEdgeQueueWorker(db *sqlx.DB) *EdgeQueueWorker {
	return &EdgeQueueWorker{
		db:           db,
		pollInterval: 5 * time.Second,
		batchSize:    50,
		maxRetries:   3,
		running:      false,
	}
}

// QueueItem represents an item in the edge creation queue
type QueueItem struct {
	ID           string          `db:"id"`
	TenantID     string          `db:"tenant_id"`
	DatasourceID string          `db:"datasource_id"`
	SourceID     string          `db:"source_node_id"`
	TargetID     string          `db:"target_node_id"`
	EdgeTypeName string          `db:"edge_type_name"`
	Properties   json.RawMessage `db:"properties"`
	Status       string          `db:"status"`
	Attempts     int             `db:"attempts"`
}

// Run starts the worker loop
func (w *EdgeQueueWorker) Run(ctx context.Context) {
	if w.running {
		logging.GetLogger().Sugar().Warn("Edge queue worker already running")
		return
	}

	w.running = true
	logging.GetLogger().Sugar().Info("🚀 Edge queue worker started")

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.running = false
			logging.GetLogger().Sugar().Info("Edge queue worker stopped")
			return
		case <-ticker.C:
			processed, failed, err := w.processBatch(ctx)
			if err != nil {
				logging.GetLogger().Sugar().Errorf("Edge queue worker error: %v", err)
			} else if processed > 0 || failed > 0 {
				logging.GetLogger().Sugar().Infof("Edge queue: processed=%d, failed=%d", processed, failed)
			}
		}
	}
}

// processBatch processes a batch of pending queue items
func (w *EdgeQueueWorker) processBatch(ctx context.Context) (processed, failed int, err error) {
	// Fetch pending items
	query := `
		SELECT id, tenant_id, datasource_id, source_node_id, target_node_id, edge_type_name, properties, status, attempts
		FROM edge_creation_queue
		WHERE status = 'pending' AND attempts < $1
		ORDER BY created_at
		LIMIT $2
	`

	var items []QueueItem
	if err := w.db.SelectContext(ctx, &items, query, w.maxRetries, w.batchSize); err != nil {
		return 0, 0, fmt.Errorf("failed to fetch queue items: %w", err)
	}

	if len(items) == 0 {
		return 0, 0, nil
	}

	for _, item := range items {
		// Mark as processing
		if _, err := w.db.ExecContext(ctx,
			`UPDATE edge_creation_queue SET status = 'processing', attempts = attempts + 1 WHERE id = $1`,
			item.ID); err != nil {
			continue
		}

		// Process the item
		if err := w.processItem(ctx, item); err != nil {
			failed++
			// Check if max retries exceeded
			if item.Attempts+1 >= w.maxRetries {
				// Move to DLQ
				w.moveToDLQ(ctx, item, err.Error())
			} else {
				// Mark as failed for retry
				_, _ = w.db.ExecContext(ctx,
					`UPDATE edge_creation_queue SET status = 'pending', error_message = $2 WHERE id = $1`,
					item.ID, err.Error())
			}
		} else {
			processed++
			// Mark as completed
			_, _ = w.db.ExecContext(ctx,
				`UPDATE edge_creation_queue SET status = 'completed', processed_at = $2 WHERE id = $1`,
				item.ID, time.Now())
		}
	}

	return processed, failed, nil
}

// processItem creates the actual edge in catalog_edge
func (w *EdgeQueueWorker) processItem(ctx context.Context, item QueueItem) error {
	// Validate source and target nodes exist
	var sourceExists, targetExists bool

	if err := w.db.GetContext(ctx, &sourceExists,
		`SELECT EXISTS(SELECT 1 FROM catalog_node WHERE id = $1)`, item.SourceID); err != nil {
		return fmt.Errorf("failed to check source node: %w", err)
	}
	if !sourceExists {
		return fmt.Errorf("source node not found: %s", item.SourceID)
	}

	if err := w.db.GetContext(ctx, &targetExists,
		`SELECT EXISTS(SELECT 1 FROM catalog_node WHERE id = $1)`, item.TargetID); err != nil {
		return fmt.Errorf("failed to check target node: %w", err)
	}
	if !targetExists {
		return fmt.Errorf("target node not found: %s", item.TargetID)
	}

	// Get edge type ID if available
	var edgeTypeID *string
	err := w.db.GetContext(ctx, &edgeTypeID,
		`SELECT id FROM catalog_edge_type WHERE edge_type_name = $1 LIMIT 1`, item.EdgeTypeName)
	if err != nil {
		// Edge type not found, will be null
		edgeTypeID = nil
	}

	// Create the edge
	edgeID := uuid.New().String()
	now := time.Now()

	_, err = w.db.ExecContext(ctx, `
		INSERT INTO catalog_edge (
			id, tenant_id, tenant_datasource_id, source_node_id, target_node_id, 
			edge_type_name, edge_type_id, properties, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_name, target_node_id) 
		DO UPDATE SET properties = EXCLUDED.properties, updated_at = EXCLUDED.updated_at
	`, edgeID, item.TenantID, item.DatasourceID, item.SourceID, item.TargetID,
		item.EdgeTypeName, edgeTypeID, item.Properties, now)

	if err != nil {
		return fmt.Errorf("failed to create edge: %w", err)
	}

	return nil
}

// moveToDLQ moves a failed item to the dead letter queue
func (w *EdgeQueueWorker) moveToDLQ(ctx context.Context, item QueueItem, errorMessage string) {
	dlqID := uuid.New().String()

	_, err := w.db.ExecContext(ctx, `
		INSERT INTO edge_creation_dlq (
			id, original_message_id, tenant_id, datasource_id, 
			source_node_id, target_node_id, edge_type_name, properties,
			error_code, error_message, attempts, created_at, failed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 
		          (SELECT created_at FROM edge_creation_queue WHERE id = $2),
		          NOW())
	`, dlqID, item.ID, item.TenantID, item.DatasourceID,
		item.SourceID, item.TargetID, item.EdgeTypeName, item.Properties,
		"MAX_RETRIES_EXCEEDED", errorMessage, item.Attempts+1)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to move item to DLQ: %v", err)
		return
	}

	// Mark original as dlq'd
	_, _ = w.db.ExecContext(ctx,
		`UPDATE edge_creation_queue SET status = 'dlq' WHERE id = $1`, item.ID)
}

// GetPendingCount returns the number of pending items in the queue
func (w *EdgeQueueWorker) GetPendingCount(ctx context.Context) (int, error) {
	var count int
	err := w.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM edge_creation_queue WHERE status = 'pending'`)
	return count, err
}

// GetDLQCount returns the number of items in the dead letter queue
func (w *EdgeQueueWorker) GetDLQCount(ctx context.Context) (int, error) {
	var count int
	err := w.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM edge_creation_dlq`)
	return count, err
}
