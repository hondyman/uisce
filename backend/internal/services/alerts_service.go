package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/logging"
	public_models "github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// AlertsService handles business logic for alerts.
type AlertsService struct {
	db            *sqlx.DB
	modelProvider *analytics.ModelProvider
}

// NewAlertsService creates a new AlertsService.
func NewAlertsService(db *sqlx.DB, modelProvider *analytics.ModelProvider) *AlertsService {
	return &AlertsService{db: db, modelProvider: modelProvider}
}

// List retrieves alerts for a user. Mock implementation.
func (s *AlertsService) List(ctx context.Context, userID string) ([]public_models.Alert, error) {
	// Mock data for now
	return []public_models.Alert{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Severity:  "critical",
			Message:   "Sales revenue forecast is 25% below target for next month.",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			IsRead:    false,
		},
		{
			ID:        uuid.New(),
			UserID:    userID,
			Severity:  "warning",
			Message:   "The 'daily_rollup' pre-aggregation for the 'orders' model is stale.",
			CreatedAt: time.Now().Add(-1 * 24 * time.Hour),
			IsRead:    false,
		},
	}, nil
}

// MarkRead marks an alert as read. Mock implementation.
func (s *AlertsService) MarkRead(ctx context.Context, alertID uuid.UUID, userID string) error {
	// In a real app, this would update the database.
	logging.GetLogger().Sugar().Infof("User %s marked alert %s as read", userID, alertID)
	return nil
}
