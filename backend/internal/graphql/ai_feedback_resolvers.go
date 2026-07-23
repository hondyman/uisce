package graphql

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// LogTermFeedbackInput represents the input for logging feedback
type LogTermFeedbackInput struct {
	TenantID     string                 `json:"tenantId"`
	DatasourceID string                 `json:"datasourceId"`
	TermID       string                 `json:"termId"`
	NodeID       *string                `json:"nodeId"`
	SuggestionID string                 `json:"suggestionId"`
	Action       string                 `json:"action"`
	Reason       *string                `json:"reason"`
	OldTermID    *string                `json:"oldTermId"`
	Features     map[string]interface{} `json:"features"`
}

// LogTermAISuggestionFeedback records user feedback for AI suggestions
func (r *Resolver) LogTermAISuggestionFeedback(ctx context.Context, input LogTermFeedbackInput) (string, error) {
	// 1. Generate new Feedback ID
	feedbackID := uuid.New().String()

	// 2. Marshal features to JSON
	featuresJson, err := json.Marshal(input.Features)
	if err != nil {
		return "", fmt.Errorf("failed to marshal features: %w", err)
	}

	// 3. Insert into database
	query := `
		INSERT INTO term_ai_feedback (
			feedback_id, tenant_id, datasource_id, term_id, node_id, suggestion_id,
			action, reason, old_term_id, created_at, features
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), $10
		) RETURNING feedback_id
	`

	err = r.DB.QueryRowContext(
		ctx,
		query,
		feedbackID,
		input.TenantID,
		input.DatasourceID,
		input.TermID,
		input.NodeID,
		input.SuggestionID,
		input.Action,
		input.Reason,
		input.OldTermID,
		featuresJson,
	).Scan(&feedbackID)

	if err != nil {
		return "", fmt.Errorf("failed to log feedback: %w", err)
	}

	return feedbackID, nil
}
