package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// LifeEvent represents a significant financial event
type LifeEvent struct {
	ID         uuid.UUID       `db:"id" json:"id"`
	ClientID   uuid.UUID       `db:"client_id" json:"client_id"`
	EventType  string          `db:"event_type" json:"event_type"`
	EventDate  time.Time       `db:"event_date" json:"event_date"`
	Status     string          `db:"status" json:"status"`
	Attributes json.RawMessage `db:"attributes" json:"attributes"`
	CreatedAt  time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at" json:"updated_at"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: sqlx.NewDb(db, "postgres")}
}

// CreateEvent creates a new life event and triggers a recalculation signal
func (s *Service) CreateEvent(ctx context.Context, event *LifeEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	
	query := `
		INSERT INTO life_events (id, client_id, event_type, event_date, status, attributes)
		VALUES (:id, :client_id, :event_type, :event_date, :status, :attributes)
	`
	
	_, err := s.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	// Trigger Recalculation Signal (Mock for now)
	s.emitRecalculationSignal(event.ClientID)

	return nil
}

// GetEventsByClient retrieves all active events for a client
func (s *Service) GetEventsByClient(ctx context.Context, clientID uuid.UUID) ([]LifeEvent, error) {
	var events []LifeEvent
	query := `SELECT * FROM life_events WHERE client_id = $1 ORDER BY event_date ASC`
	err := s.db.SelectContext(ctx, &events, query, clientID)
	return events, err
}

func (s *Service) emitRecalculationSignal(clientID uuid.UUID) {
	// In a real system, this would publish to RabbitMQ or Temporal
	fmt.Printf("🚀 SIGNAL: Triggering Plan Recalculation for Client %s\n", clientID)
}
