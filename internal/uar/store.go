package uar

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryStore is an in-memory Universal Audit Record store for development.
type InMemoryStore struct {
	mu      sync.RWMutex
	records []map[string]any
}

// NewInMemoryUAR creates a new in-memory UAR store.
func NewInMemoryUAR() *InMemoryStore {
	return &InMemoryStore{
		records: make([]map[string]any, 0),
	}
}

// Write persists a record to the in-memory store.
func (s *InMemoryStore) Write(ctx context.Context, tenantID, portfolioID, eventType string, payload map[string]any) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	recordID := uuid.New().String()
	record := map[string]any{
		"id":           recordID,
		"tenant_id":    tenantID,
		"portfolio_id": portfolioID,
		"event_type":   eventType,
		"payload":      payload,
		"created_at":   time.Now(),
	}

	s.records = append(s.records, record)
	return recordID, nil
}

// Read retrieves all records from the in-memory store (for debugging).
func (s *InMemoryStore) Read(ctx context.Context, tenantID string) ([]map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []map[string]any
	for _, rec := range s.records {
		if tid, ok := rec["tenant_id"].(string); ok && tid == tenantID {
			result = append(result, rec)
		}
	}
	return result, nil
}

// PostgresStore is a Postgres-backed Universal Audit Record store.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new Postgres UAR store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Write persists a record to Postgres.
func (s *PostgresStore) Write(ctx context.Context, tenantID, portfolioID, eventType string, payload map[string]any) (string, error) {
	recordID := uuid.New().String()
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	query := `
		INSERT INTO universal_audit_records (id, tenant_id, portfolio_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = s.db.ExecContext(ctx, query, recordID, tenantID, portfolioID, eventType, payloadJSON, time.Now())
	if err != nil {
		return "", fmt.Errorf("insert UAR: %w", err)
	}

	return recordID, nil
}

// Read retrieves records from Postgres for a given tenant.
func (s *PostgresStore) Read(ctx context.Context, tenantID string) ([]map[string]any, error) {
	query := `
		SELECT id, tenant_id, portfolio_id, event_type, payload, created_at
		FROM universal_audit_records
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query UAR: %w", err)
	}
	defer rows.Close()

	var records []map[string]any
	for rows.Next() {
		var id, tid, pid, eventType string
		var payloadJSON []byte
		var createdAt time.Time

		if err := rows.Scan(&id, &tid, &pid, &eventType, &payloadJSON, &createdAt); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		var payload map[string]any
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			return nil, fmt.Errorf("unmarshal payload: %w", err)
		}

		records = append(records, map[string]any{
			"id":           id,
			"tenant_id":    tid,
			"portfolio_id": pid,
			"event_type":   eventType,
			"payload":      payload,
			"created_at":   createdAt,
		})
	}

	return records, rows.Err()
}

// Verify recomputes the chain for a tenant (placeholder for future implementation).
func (s *InMemoryStore) Verify(ctx context.Context, tenantID string) error {
	// For now, just confirm we have records for this tenant
	records, err := s.Read(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return fmt.Errorf("no records found for tenant %s", tenantID)
	}
	return nil
}

// Verify recomputes the chain for a tenant (placeholder for future implementation).
func (s *PostgresStore) Verify(ctx context.Context, tenantID string) error {
	// For now, just confirm we have records for this tenant
	records, err := s.Read(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return fmt.Errorf("no records found for tenant %s", tenantID)
	}
	return nil
}

