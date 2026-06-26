package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

type BOExportService struct {
	db *sqlx.DB
}

func NewBOExportService(db *sqlx.DB) *BOExportService {
	return &BOExportService{
		db: db,
	}
}

// ExportBO generates a complete export package for a single BO
func (s *BOExportService) ExportBO(ctx context.Context, tenantID string, boID string) (*models.ExportBundle, error) {
	bundle := &models.ExportBundle{
		Version:        "1.0",
		IcebergVersion: "snapshot-latest", // TODO: Fetch real snapshot ID from Iceberg Service
		ExportedAt:     time.Now(),
		TenantID:       &tenantID,
	}

	// 1. Fetch the BO Node
	boNode, err := s.fetchNodeExport(ctx, boID, "business_object")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch BO node: %w", err)
	}
	bundle.BusinessObjects = append(bundle.BusinessObjects, *boNode)

	// 2. Identify Dependencies (Terms, Calculations) using Edges
	// We need to traverse:
	// BO -> HAS_TERM -> SemanticTerm
	// BO -> HAS_CALC -> CalculationTerm
	// BO -> DRIVES -> Table (Export as edge, but maybe not the table config itself unless we want to export table metadata? Spec says BO/Term/Calc)
	// BO -> RELATES_TO -> BusinessObject (Optional: do we export related BOs recursively? Spec implies "Select BOs", so maybe we only reference them via edges or export if selected. For now, let's just export edges and minimal reference, user must select them to include full definition.)

	// Let's implement a graph traversal to find "owned" items like Terms and Calcs associated with this BO.
	// Actually, Semantic Terms are often shared. Calculations can be BO-specific or shared.
	// The export should probably include the Terms used by the BO.

	// Helper to fetch connected nodes
	terms, err := s.fetchConnectedNodes(ctx, boID, "HAS_TERM", "semantic_term")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch semantic terms: %w", err)
	}
	bundle.SemanticTerms = terms

	calcs, err := s.fetchConnectedNodes(ctx, boID, "HAS_CALC", "calculation_term")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch calculation terms: %w", err)
	}
	bundle.CalculationTerms = calcs

	return bundle, nil
}

// ExportMultipleBOs exports multiple BOs into one bundle
func (s *BOExportService) ExportMultipleBOs(ctx context.Context, tenantID string, boIDs []string) (*models.ExportBundle, error) {
	bundle := &models.ExportBundle{
		Version:        "1.0",
		IcebergVersion: "snapshot-latest", // TODO: Fetch real snapshot ID
		ExportedAt:     time.Now(),
		TenantID:       &tenantID,
	}

	seenTerms := make(map[string]bool)
	seenCalcs := make(map[string]bool)

	for _, boID := range boIDs {
		// Fetch BO
		boNode, err := s.fetchNodeExport(ctx, boID, "business_object")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch BO %s: %w", boID, err)
		}
		bundle.BusinessObjects = append(bundle.BusinessObjects, *boNode)

		// Fetch Terms
		terms, err := s.fetchConnectedNodes(ctx, boID, "HAS_TERM", "semantic_term")
		if err != nil {
			return nil, err
		}
		for _, t := range terms {
			if !seenTerms[t.Node.NodeName] { // Dedupe by name? or ID? Export relies on Name usually for portability.
				bundle.SemanticTerms = append(bundle.SemanticTerms, t)
				seenTerms[t.Node.NodeName] = true
			}
		}

		// Fetch Calcs
		calcs, err := s.fetchConnectedNodes(ctx, boID, "HAS_CALC", "calculation_term")
		if err != nil {
			return nil, err
		}
		for _, c := range calcs {
			if !seenCalcs[c.Node.NodeName] {
				bundle.CalculationTerms = append(bundle.CalculationTerms, c)
				seenCalcs[c.Node.NodeName] = true
			}
		}
	}

	return bundle, nil
}

// Helpers

func (s *BOExportService) fetchNodeExport(ctx context.Context, nodeID string, expectedType string) (*models.NodeExport, error) {
	// 1. Get Node Data
	var node models.CatalogNodeExport
	query := `
		SELECT 
			nt.catalog_type_name as node_type_id,
			n.node_name,
			COALESCE(n.qualified_path, '') as qualified_path,
			n.properties,
			n.config
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE n.id = $1
	`
	err := s.db.GetContext(ctx, &node, query, nodeID)
	if err != nil {
		return nil, err
	}

	if expectedType != "" && node.NodeTypeID != expectedType {
		// Log warning or strict check? For now loose.
	}

	// 2. Get Edges (Outgoing)
	edges, err := s.fetchEdges(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	return &models.NodeExport{
		Node:  node,
		Edges: edges,
	}, nil
}

func (s *BOExportService) fetchConnectedNodes(ctx context.Context, sourceID string, edgeType string, targetNodeType string) ([]models.NodeExport, error) {
	// Find IDs of connected nodes
	var targetIDs []string
	query := `
		SELECT e.target_node_id
		FROM catalog_edge e
		JOIN catalog_edge_type et ON e.edge_type_id = et.id
		JOIN catalog_node n ON e.target_node_id = n.id
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE e.source_node_id = $1 
		  AND et.type_name = $2
		  AND nt.catalog_type_name = $3
	`
	err := s.db.SelectContext(ctx, &targetIDs, query, sourceID, edgeType, targetNodeType)
	if err != nil {
		return nil, err
	}

	var nodes []models.NodeExport
	for _, id := range targetIDs {
		n, err := s.fetchNodeExport(ctx, id, targetNodeType)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *n)
	}
	return nodes, nil
}

func (s *BOExportService) fetchEdges(ctx context.Context, sourceID string) ([]models.EdgeExport, error) {
	// We want to export edges where this node is the source.
	// And we resolve names of targets.
	query := `
		SELECT 
			sn.node_name as source_name,
			snt.catalog_type_name as source_type,
			tn.node_name as target_name,
			tnt.catalog_type_name as target_type,
			et.type_name as edge_type,
			e.properties
		FROM catalog_edge e
		JOIN catalog_node sn ON e.source_node_id = sn.id
		JOIN catalog_node_type snt ON sn.node_type_id = snt.id
		JOIN catalog_node tn ON e.target_node_id = tn.id
		JOIN catalog_node_type tnt ON tn.node_type_id = tnt.id
		JOIN catalog_edge_type et ON e.edge_type_id = et.id
		WHERE e.source_node_id = $1
	`

	var edges []models.EdgeExport
	err := s.db.SelectContext(ctx, &edges, query, sourceID)
	if err != nil {
		return nil, err
	}
	return edges, nil
}
