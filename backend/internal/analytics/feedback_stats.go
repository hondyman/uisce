package analytics

import (
	"context"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// FeedbackRecord represents historical approval/rejection data
type FeedbackRecord struct {
	ColumnID     string  `db:"column_id"`
	ColumnName   string  `db:"column_name"`
	SemanticTerm string  `db:"suggested_semantic_term"`
	BusinessTerm string  `db:"suggested_business_term"`
	Status       string  `db:"status"` // 'approved' or 'rejected'
	Confidence   float64 `db:"confidence"`
}

// MappingFeedbackStats holds aggregated feedback statistics for semantic mappings
type MappingFeedbackStats struct {
	ApprovedPatterns map[string]int    // "column_name:semantic_term" -> count
	RejectedPatterns map[string]int    // "column_name:semantic_term" -> count
	ColumnMappings   map[string]string // column_id -> semantic_term (for already mapped)
}

// loadFeedbackStats queries historical approvals/rejections from pending_semantic_mappings
func (s *SemanticMappingService) loadFeedbackStats(ctx context.Context, tenantID, datasourceID string) (*MappingFeedbackStats, error) {
	logger := logging.GetLogger().Sugar()

	stats := &MappingFeedbackStats{
		ApprovedPatterns: make(map[string]int),
		RejectedPatterns: make(map[string]int),
		ColumnMappings:   make(map[string]string),
	}

	// Query historical feedback from term_ai_feedback (NEW source including manual overrides)
	feedbackQuery := `
		SELECT features->>'column_name' as column_name, features->>'semantic_term' as suggested_semantic_term, action as status, (features->>'confidence')::float as confidence
		FROM term_ai_feedback
		WHERE tenant_id = $1 AND (datasource_id = $2 OR datasource_id IS NULL)
	`
	var records []FeedbackRecord
	_ = s.db.SelectContext(ctx, &records, feedbackQuery, tenantID, datasourceID) // Ignore error, might be empty

	// Also query pending_semantic_mappings for historical data
	pendingQuery := `
		SELECT column_name, suggested_semantic_term, status, confidence
		FROM pending_semantic_mappings
		WHERE tenant_id = $1 AND datasource_id = $2 AND status IN ('approved', 'rejected')
	`
	var pendingRecords []FeedbackRecord
	_ = s.db.SelectContext(ctx, &pendingRecords, pendingQuery, tenantID, datasourceID)

	// Merge records
	allRecords := append(records, pendingRecords...)

	// Build feedback maps
	for _, record := range allRecords {
		key := fmt.Sprintf("%s:%s", strings.ToLower(record.ColumnName), strings.ToLower(record.SemanticTerm))

		// Normalize status
		status := strings.ToLower(record.Status)

		if status == "approved" {
			stats.ApprovedPatterns[key]++
			logger.Debugf("Loaded approved pattern: %s (count: %d)", key, stats.ApprovedPatterns[key])
		} else if status == "rejected" {
			stats.RejectedPatterns[key]++
			logger.Debugf("Loaded rejected pattern: %s (count: %d)", key, stats.RejectedPatterns[key])
		}
	}

	logger.Infof("Loaded feedback stats: %d approved, %d rejected patterns",
		len(stats.ApprovedPatterns), len(stats.RejectedPatterns))

	return stats, nil
}
