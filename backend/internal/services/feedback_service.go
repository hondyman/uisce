package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FeedbackService handles user feedback for NLQ responses.
type FeedbackService struct {
	db     *sqlx.DB
	hasura HasuraClient
}

// NewFeedbackService creates a new FeedbackService.
func NewFeedbackService(db *sqlx.DB) *FeedbackService {
	return &FeedbackService{
		db: db,
	}
}

// NewFeedbackServiceWithHasura creates a FeedbackService with Hasura support
func NewFeedbackServiceWithHasura(db *sqlx.DB, hasura HasuraClient) *FeedbackService {
	return &FeedbackService{
		db:     db,
		hasura: hasura,
	}
}

// FeedbackRequest represents the payload for submitting feedback.
type FeedbackRequest struct {
	QueryID  uuid.UUID `json:"query_id"`
	TenantID string    `json:"tenant_id"`
	UserID   string    `json:"user_id"`
	Rating   int       `json:"rating"`
	Comment  string    `json:"comment"`
}

// SubmitFeedback stores user feedback in the database.
func (s *FeedbackService) SubmitFeedback(ctx context.Context, req FeedbackRequest) error {
	if req.Rating < 1 || req.Rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	err := s.submitFeedbackRecord(ctx, req.QueryID, req.TenantID, req.UserID, req.Rating, req.Comment, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert feedback: %w", err)
	}

	return nil
}

// ============================================================================
// HASURA-FIRST HELPERS
// ============================================================================

// submitFeedbackRecord inserts feedback into the database
// Hasura-first with SQL fallback
func (s *FeedbackService) submitFeedbackRecord(ctx context.Context, queryID uuid.UUID, tenantID, userID string, rating int, comment string, createdAt time.Time) error {
	if s.hasura != nil {
		mutation := `
			mutation SubmitFeedback(
				$queryID: uuid!
				$tenantID: String!
				$userID: String!
				$rating: Int!
				$comment: String!
				$createdAt: timestamptz!
			) {
				insert_nlq_feedback_one(object: {
					query_id: $queryID
					tenant_id: $tenantID
					user_id: $userID
					rating: $rating
					comment: $comment
					created_at: $createdAt
				}) {
					query_id
				}
			}
		`

		variables := map[string]interface{}{
			"queryID":   queryID,
			"tenantID":  tenantID,
			"userID":    userID,
			"rating":    rating,
			"comment":   comment,
			"createdAt": createdAt,
		}

		_, err := s.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
	query := `
		INSERT INTO nlq_feedback (query_id, tenant_id, user_id, rating, comment, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.db.ExecContext(ctx, query, queryID, tenantID, userID, rating, comment, createdAt)
	return err
}
