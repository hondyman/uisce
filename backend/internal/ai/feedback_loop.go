package ai

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type FeedbackLoopService struct {
	db *sql.DB
}

func NewFeedbackLoopService(db *sql.DB) *FeedbackLoopService {
	return &FeedbackLoopService{db: db}
}

// ProcessFeedback adjusts recommendation affinities and vector rankings based on user validation
func (s *FeedbackLoopService) ProcessFeedback(
	ctx context.Context,
	tenantID, interactionID uuid.UUID,
	rating int, // +1 or -1
	errorCategory string,
	termIDs []uuid.UUID,
) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if s.db == nil {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertFeedback := `
		INSERT INTO catalog_ai.interaction_feedback (
			interaction_id, tenant_id, rating, error_category, created_at
		) VALUES ($1, $2, $3, $4, NOW());`

	if _, err := tx.ExecContext(ctx, insertFeedback, interactionID, tenantID, rating, errorCategory); err != nil {
		return fmt.Errorf("failed logging feedback: %w", err)
	}

	weightDelta := 0.2500
	if rating < 0 {
		weightDelta = -0.5000
	}

	for i := 0; i < len(termIDs); i++ {
		for j := i + 1; j < len(termIDs); j++ {
			reinforceQuery := `
				INSERT INTO catalog_ai.term_recommendation_weights (
					tenant_id, source_term_node_id, target_term_node_id, affinity_score, co_occurrence_count, last_reinforced_at
				) VALUES ($1, $2, $3, GREATEST(0.1, 1.0 + $4), 1, NOW())
				ON CONFLICT (tenant_id, source_term_node_id, target_term_node_id) DO UPDATE SET
					affinity_score = GREATEST(0.01, catalog_ai.term_recommendation_weights.affinity_score + $4),
					co_occurrence_count = catalog_ai.term_recommendation_weights.co_occurrence_count + 1,
					last_reinforced_at = NOW();`

			_, _ = tx.ExecContext(ctx, reinforceQuery, tenantID, termIDs[i], termIDs[j], weightDelta)
		}
	}

	return tx.Commit()
}
