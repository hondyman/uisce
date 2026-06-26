package services

import (
	"context"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/repository"
)

// InternalEventService manages internal events and publishes changes
type InternalEventService struct {
	repo      *repository.GoogleSyncRepo
	publisher *EventPublisher
}

// NewInternalEventService creates a new service
func NewInternalEventService(repo *repository.GoogleSyncRepo, publisher *EventPublisher) *InternalEventService {
	return &InternalEventService{
		repo:      repo,
		publisher: publisher,
	}
}

// CreateEvent creates a new event and publishes it
func (s *InternalEventService) CreateEvent(ctx context.Context, event *models.InternalEvent) error {
	if err := s.repo.CreateInternalEvent(ctx, event); err != nil {
		return err
	}
	// Publishing is best-effort or transactional outbox pattern.
	// Here simple publish after commit (Hasura mutate is atomic per request).
	s.publisher.PublishInternalEventCreated(ctx, event, event.UserID.String())
	return nil
}

// UpdateEvent updates an event and publishes it
func (s *InternalEventService) UpdateEvent(ctx context.Context, event *models.InternalEvent) error {
	if err := s.repo.UpdateInternalEvent(ctx, event); err != nil {
		return err
	}
	s.publisher.PublishInternalEventUpdated(ctx, event, event.UserID.String())
	return nil
}

// DeleteEvent deletes an event and publishes it
func (s *InternalEventService) DeleteEvent(ctx context.Context, eventID, tenantID, userID string) error {
	if err := s.repo.DeleteInternalEvent(ctx, eventID); err != nil {
		return err
	}
	s.publisher.PublishInternalEventDeleted(ctx, eventID, tenantID, userID)
	return nil
}
