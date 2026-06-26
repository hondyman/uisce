package analytics

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// AutoGenerateSemanticTerms scans all columns in a datasource and automatically creates semantic terms
// for those that meet the confidence threshold.
func (s *SemanticMappingService) AutoGenerateSemanticTerms(ctx context.Context, tenantID, datasourceID string, threshold float64) (*AutoEnrichmentResult, error) {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Starting auto-generation of semantic terms for datasource %s with threshold %.2f", datasourceID, threshold)

	// 1. List all database columns
	columns, err := s.ListDatabaseColumns(ctx, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list database columns: %w", err)
	}

	result := &AutoEnrichmentResult{
		TotalColumns: len(columns),
	}

	var totalConfidence float64

	// 2. Iterate through columns
	for _, col := range columns {
		// Check if already mapped (optional optimization: skip if edge exists)
		// For now, we'll check if we can improve or if it's missing.
		// But SuggestEnrichment doesn't check for existing edges, it just suggests.

		// Check if edge exists to avoid re-processing mapped columns if desired?
		// The requirement says "suggest and/or creates".
		// Let's check if it's already mapped to avoid duplicates or overwrites unless needed.
		// But for "auto-create", we usually skip already mapped ones.
		// We can check edge existence.
		// We don't have a quick "IsMapped" on the column struct.
		// We can use checkEdgeExists if we had the semantic term ID, but we don't know it yet.
		// Let's assume we proceed and if it exists, we might update or skip.
		// Actually, ApplyEnrichment creates edges.

		// Let's generate proposal
		// We need a NodeProperties object. The column struct has the fields.
		profile := &NodeProperties{
			DataType:         col.DataType,
			Cardinality:      col.Cardinality,
			FrequentValues:   col.FrequentValues,
			InferredPatterns: col.InferredPatterns,
		}

		proposal, err := s.SuggestEnrichment(ctx, &col, profile)
		if err != nil {
			logger.Warnf("Failed to suggest enrichment for column %s: %v", col.QualifiedPath, err)
			result.FailedColumns++
			continue
		}

		totalConfidence += proposal.Confidence

		// 3. Check threshold
		if proposal.Confidence >= threshold {
			logger.Infof("Auto-creating term for %s (Confidence: %.2f)", col.QualifiedPath, proposal.Confidence)

			req := &ApplyEnrichmentRequest{
				Proposal:     proposal,
				ColumnID:     col.NodeID,
				TenantID:     tenantID,
				DatasourceID: datasourceID,
				Column:       &col,       // Pass column data for intelligent property inference
				ColumnName:   col.Column, // Pass column name for SQL property generation
			}

			_, err := s.ApplyEnrichment(ctx, req)
			if err != nil {
				logger.Errorf("Failed to apply enrichment for %s: %v", col.QualifiedPath, err)
				result.FailedColumns++
			} else {
				result.EnrichedColumns++
			}
		} else {
			result.SkippedColumns++
		}
	}

	if result.TotalColumns > 0 {
		result.AverageConfidence = totalConfidence / float64(result.TotalColumns)
	}

	logger.Infof("Auto-generation completed. Stats: %+v", result)
	return result, nil
}
