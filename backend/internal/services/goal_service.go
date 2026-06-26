package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// GoalService provides methods for managing goals.
type GoalService struct {
	db *sqlx.DB
}

// NewGoalService creates a new GoalService.
func NewGoalService(db *sqlx.DB) *GoalService {
	return &GoalService{db: db}
}

// ListGoals retrieves goals for a given user.
// NOTE: This is a mocked implementation.
func (s *GoalService) ListGoals(ctx context.Context, userID string) ([]models.Goal, error) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	mockGoals := []models.Goal{
		{
			ID:          uuid.New(),
			Name:        "Keep Churn Below 5%",
			Description: sql.NullString{String: "Monitor daily churn rate from the main retention query.", Valid: true},
			Status:      "met",
			LastChecked: &yesterday,
		},
		{
			ID:          uuid.New(),
			Name:        "Weekly Sales > $1M",
			Description: sql.NullString{String: "Ensure weekly sales revenue exceeds the $1M target.", Valid: true},
			Status:      "missed",
			LastChecked: &now,
		},
	}

	return mockGoals, nil
}
