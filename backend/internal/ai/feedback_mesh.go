package ai

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type FeedbackPayload struct {
	InteractionID string     `json:"interactionId"`
	UserID        string     `json:"userId"`
	IsPositive    bool       `json:"isPositive"`
	ErrorCategory string     `json:"errorCategory,omitempty"` // WRONG_TABLE, BAD_FORMULA, BAD_JOIN, HALLUCINATION
	UserNotes     string     `json:"userNotes,omitempty"`
	ResolvedBOID  *uuid.UUID `json:"resolvedBoId,omitempty"`
}

type FeedbackMeshService struct {
	db *sqlx.DB
}

func NewFeedbackMeshService(db *sqlx.DB) *FeedbackMeshService {
	return &FeedbackMeshService{db: db}
}

// RecordTwoStageFeedback persists binary feedback and updates embedding weighting penalties
func (s *FeedbackMeshService) RecordTwoStageFeedback(
	ctx context.Context,
	tenantID uuid.UUID,
	payload FeedbackPayload,
) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if s.db == nil {
		return nil
	}

	weightDelta := 0.05
	if !payload.IsPositive {
		weightDelta = -0.15 // Decay ranking penalty on negative feedback
	}

	query := `
		INSERT INTO ai_telemetry.explicit_feedback_ledger (
			tenant_id, interaction_id, user_id, is_positive,
			error_category, user_notes, resolved_bo_id, applied_weight_delta, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW());
	`
	_, err := s.db.ExecContext(ctx, query,
		tenantID, payload.InteractionID, payload.UserID, payload.IsPositive,
		payload.ErrorCategory, payload.UserNotes, payload.ResolvedBOID, weightDelta,
	)
	return err
}
