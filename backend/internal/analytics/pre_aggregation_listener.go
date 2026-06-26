package analytics

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PreAggInvalidationListener creates a listener function that invalidates pre-aggs when graph changes occur.
// This should be registered with SemanticGraphService.RegisterChangeListener().
func PreAggInvalidationListener(db *sqlx.DB, invalidationSvc *PreAggInvalidationService) func(nodeID uuid.UUID) {
	return func(nodeID uuid.UUID) {
		ctx := context.Background()

		// Get node type
		var nodeType string
		err := db.GetContext(ctx, &nodeType, `
			SELECT nt.catalog_type_name 
			FROM catalog_node n
			JOIN catalog_node_type nt ON n.node_type_id = nt.id
			WHERE n.id = $1
		`, nodeID)
		if err != nil {
			return // Node not found or deleted
		}

		// Invalidate based on node type
		switch nodeType {
		case "business_object":
			_ = invalidationSvc.InvalidateByBO(ctx, nodeID)
		case "semantic_term":
			_ = invalidationSvc.InvalidateByTerm(ctx, nodeID)
		case "calculation_term":
			_ = invalidationSvc.InvalidateByCalculation(ctx, nodeID)
		}
	}
}
