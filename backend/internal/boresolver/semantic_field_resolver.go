package boresolver

import (
	"context"
	"fmt"
)

// ============================================================================
// FIELD RESOLUTION PIPELINE
// ============================================================================

// FieldResolver handles the complete resolution chain:
// BO Field → Semantic Term → Catalog Edge → Physical Table + Column
type FieldResolver struct {
	boRepo       *BusinessObjectCachedRepository
	semanticRepo *SemanticTermRepository
	catalogRepo  *CatalogRepository
}

// NewFieldResolver creates a new field resolver with all required repositories.
func NewFieldResolver(
	boRepo *BusinessObjectCachedRepository,
	semanticRepo *SemanticTermRepository,
	catalogRepo *CatalogRepository,
) *FieldResolver {
	return &FieldResolver{
		boRepo:       boRepo,
		semanticRepo: semanticRepo,
		catalogRepo:  catalogRepo,
	}
}

// ResolveFieldToPhysical follows the complete resolution chain and returns the physical table + column.
//
// Resolution pipeline (canonical implementation - Part A):
//  1. Load BO field
//  2. Check for BO-level overrides (physical_table + physical_column)
//  3. If override found → return immediately with SourceType="OVERRIDE"
//  4. Load semantic term by semantic_term_id
//  5. Query catalog edges for (term, datasource)
//  6. Walk TERM_MAPS_TO_COLUMN edge to column node
//  7. Follow parent_id to table node
//  8. Return ResolvedField{Table, Column, SourceType="SEMANTIC"}
//
// This is the authoritative resolver that SQL generation calls for every field and filter.
func (r *FieldResolver) ResolveFieldToPhysical(
	ctx context.Context,
	fieldID string,
	datasourceID string,
) (*ResolvedField, error) {
	// STEP 1: BO Field
	field, err := r.boRepo.GetFieldByID(ctx, fieldID)
	if err != nil {
		return nil, fmt.Errorf("step 1: failed to load BO field %s: %w", fieldID, err)
	}

	// STEP 2-3: BO-level Overrides
	if field.PhysicalTable != nil && *field.PhysicalTable != "" &&
		field.PhysicalColumn != nil && *field.PhysicalColumn != "" {
		return &ResolvedField{
			FieldID:        field.ID,
			FieldName:      field.Name,
			SemanticTermID: field.SemanticTermID,
			Table:          *field.PhysicalTable,
			Column:         *field.PhysicalColumn,
			SemanticName:   field.Name,
			SourceType:     "OVERRIDE",
		}, nil
	}

	// STEP 4: Semantic Term
	if field.SemanticTermID == "" {
		return nil, fmt.Errorf(
			"step 4: field %s has no semantic term and no physical override",
			fieldID,
		)
	}

	term, err := r.semanticRepo.GetSemanticTerm(ctx, field.SemanticTermID)
	if err != nil {
		return nil, fmt.Errorf(
			"step 4: failed to load semantic term %s for field %s: %w",
			field.SemanticTermID, fieldID, err,
		)
	}

	// STEPS 5-7: Catalog Edges → Physical Mapping
	table, column, err := r.resolvePhysicalMapping(ctx, term.ID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf(
			"steps 5-7: no physical mapping for semantic term %s (field %s): %w",
			term.Name, fieldID, err,
		)
	}

	// STEP 8: Return Resolved Field
	return &ResolvedField{
		FieldID:        field.ID,
		FieldName:      field.Name,
		SemanticTermID: term.ID,
		Table:          table,
		Column:         column,
		SemanticName:   term.DisplayName,
		SourceType:     "SEMANTIC",
	}, nil
}

// resolvePhysicalMapping resolves a semantic term to physical table + column via catalog edges.
//
// This queries catalog edges and follows them to physical catalog nodes:
//
//	TERM → (CatalogEdge TERM_MAPS_TO_COLUMN) → ColumnNode → (parent) → TableNode
func (r *FieldResolver) resolvePhysicalMapping(
	ctx context.Context,
	termID string,
	datasourceID string,
) (string, string, error) {
	// Query catalog edges for this term and datasource
	edges, err := r.catalogRepo.GetEdges(ctx, termID, datasourceID)
	if err != nil {
		return "", "", err
	}

	if len(edges) == 0 {
		return "", "", fmt.Errorf(
			"no catalog edges found for semantic term %s in datasource %s",
			termID, datasourceID,
		)
	}

	// Iterate through edges to find TERM_MAPS_TO_COLUMN
	for _, edge := range edges {
		if edge.Type != string(CatalogEdgeTypeMapsToColumn) {
			continue
		}

		// Load the target catalog node (should be a column)
		columnNode, err := r.catalogRepo.GetNode(ctx, edge.ToID)
		if err != nil {
			return "", "", fmt.Errorf("failed to fetch column node %s: %w", edge.ToID, err)
		}

		if columnNode.Type != "column" {
			continue
		}

		// Get parent (table)
		if columnNode.ParentID == nil {
			return "", "", fmt.Errorf("column node %s has no parent table", columnNode.ID)
		}

		tableNode, err := r.catalogRepo.GetNode(ctx, *columnNode.ParentID)
		if err != nil {
			return "", "", fmt.Errorf("failed to fetch table node %s: %w", *columnNode.ParentID, err)
		}

		if tableNode.Type != "table" {
			return "", "", fmt.Errorf("parent node %s is not a table", *columnNode.ParentID)
		}

		return tableNode.Name, columnNode.Name, nil
	}

	return "", "", fmt.Errorf(
		"no TERM_MAPS_TO_COLUMN edge found for semantic term %s",
		termID,
	)
}

// ResolveAllFields resolves all fields for a BO in a single pass.
// This is more efficient than calling ResolveFieldToPhysical per field.
func (r *FieldResolver) ResolveAllFields(
	ctx context.Context,
	boID string,
	datasourceID string,
) (map[string]*ResolvedField, error) {
	// Fetch all fields for the BO
	fields, err := r.boRepo.GetFieldsForBO(ctx, boID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*ResolvedField)

	for _, field := range fields {
		resolved, err := r.ResolveFieldToPhysical(ctx, field.ID, datasourceID)
		if err != nil {
			// Log but don't fail the entire operation; mark as unresolved
			result[field.ID] = &ResolvedField{
				FieldID:      field.ID,
				FieldName:    field.Name,
				SourceType:   "ERROR",
				SemanticName: fmt.Sprintf("(unresolved: %v)", err),
			}
			continue
		}
		result[field.ID] = resolved
	}

	return result, nil
}
