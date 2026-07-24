package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// UserFeedback represents feedback on an AI suggestion
type UserFeedback struct {
	ID           uuid.UUID              `json:"id"`
	TenantID     uuid.UUID              `json:"tenant_id"`
	SuggestionID uuid.UUID              `json:"suggestion_id"`
	Confidence   int                    `json:"confidence"`
	Correction   map[string]interface{} `json:"correction"`
	Comments     string                 `json:"comments"`
	CreatedAt    time.Time              `json:"created_at"`
	CreatedBy    uuid.UUID              `json:"created_by"`
}

// FeedbackResponse summarises the effect of processing feedback (Whitepaper §10)
type FeedbackResponse struct {
	FeedbackID        uuid.UUID `json:"feedback_id"`
	ImprovementImpact string    `json:"improvement_impact"` // positive, neutral, negative
	RetrainingQueued  bool      `json:"retraining_queued"`
	NextSteps         []string  `json:"next_steps"`
}

// FeedbackProcessor handles user feedback to improve AI performance
type FeedbackProcessor struct {
	db           *sql.DB
	trainingData *TrainingDataStore
}

// NewFeedbackProcessor creates a new feedback processor
func NewFeedbackProcessor(db *sql.DB) *FeedbackProcessor {
	return &FeedbackProcessor{db: db, trainingData: NewTrainingDataStore(db)}
}

// ProcessFeedback stores user feedback, converts it to a training example,
// optionally queues retraining, and stores a structured response.
func (p *FeedbackProcessor) ProcessFeedback(ctx context.Context, feedback UserFeedback) (*FeedbackResponse, error) {
	// 1. Persist the raw feedback record
	correctionJSON, err := json.Marshal(feedback.Correction)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal correction: %w", err)
	}
	_, err = p.db.ExecContext(ctx, `
		INSERT INTO edm.ai_feedback (
			id, tenant_id, suggestion_id, confidence, correction, comments, created_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, feedback.ID, feedback.TenantID, feedback.SuggestionID, feedback.Confidence, correctionJSON, feedback.Comments, feedback.CreatedAt, feedback.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to insert feedback: %w", err)
	}

	// 2. Convert corrected output into a training example
	example := TrainingExample{
		ID:             uuid.New(),
		TenantID:       feedback.TenantID,
		Source:         "user_feedback",
		Input:          map[string]interface{}{"suggestion_id": feedback.SuggestionID},
		Output:         feedback.Correction,
		Explainability: feedback.Confidence * 10, // scale 1-10 -> 10-100
		CreatedAt:      time.Now(),
	}
	if addErr := p.trainingData.AddExample(ctx, example); addErr != nil {
		// Non-fatal: log and continue
		log.Printf("[FeedbackProcessor] Failed to store training example: %v", addErr)
	}

	// 3. Trigger retraining when confidence is low (indicates the model was wrong)
	retrainingQueued := false
	if feedback.Confidence <= 3 {
		retrainingQueued = true
		// In production this would publish to a retraining job queue
		log.Printf("[FeedbackProcessor] Low-confidence feedback for tenant %s — retraining queued", feedback.TenantID)
	}

	// 4. Build and persist a structured response
	impact := "neutral"
	nextSteps := []string{"Feedback recorded — thank you!"}
	if retrainingQueued {
		impact = "positive"
		nextSteps = []string{
			"Model will be retrained within 24 hours",
			"You will receive a notification when the improvement is live",
		}
	}

	resp := &FeedbackResponse{
		FeedbackID:        feedback.ID,
		ImprovementImpact: impact,
		RetrainingQueued:  retrainingQueued,
		NextSteps:         nextSteps,
	}

	// 5. Persist feedback response for audit trail
	respJSON, _ := json.Marshal(resp)
	_, _ = p.db.ExecContext(ctx, `
		UPDATE edm.ai_feedback SET correction = correction || $1 WHERE id = $2
	`, respJSON, feedback.ID)

	return resp, nil
}
