package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// PHASE 4a: CQRS PATTERN IMPLEMENTATION
// ============================================================================
//
// CQRS (Command Query Responsibility Segregation) separates read and write
// operations into independent models, enabling:
//
// 1. Optimized writes: Focus on correctness, business rules, ACID transactions
// 2. Optimized reads: Pre-aggregated, denormalized data, fast queries
// 3. Independent scaling: Read model can scale independently from write model
// 4. Event sourcing: All changes are immutable events, enabling audit trail & replay
// 5. Eventual consistency: Read model updates asynchronously from events
//
// ============================================================================
// READ MODEL PROJECTIONS - Optimized for queries
// ============================================================================

// BusinessObjectProjection represents the read model for fast BO queries
// Denormalized from events for instant access
type BusinessObjectProjection struct {
	ID             string    `db:"id" json:"id"`
	Key            string    `db:"key" json:"key"`
	TenantID       string    `db:"tenant_id" json:"tenant_id"`
	Name           string    `db:"name" json:"name"`
	DisplayName    string    `db:"display_name" json:"display_name"`
	Description    string    `db:"description" json:"description"`
	Category       string    `db:"category" json:"category"`
	IsActive       bool      `db:"is_active" json:"is_active"`
	FieldCount     int       `db:"field_count" json:"field_count"`
	InstanceCount  int       `db:"instance_count" json:"instance_count"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
	LastModifiedBy string    `db:"last_modified_by" json:"last_modified_by"`
	LastEventID    string    `db:"last_event_id" json:"last_event_id"`
	LastEventType  string    `db:"last_event_type" json:"last_event_type"`
	CorrelationID  string    `db:"correlation_id" json:"correlation_id"`
}

// CQRSReadModelRepository provides optimized read-only queries
type CQRSReadModelRepository struct {
	db *sqlx.DB
}

// NewCQRSReadModelRepository creates a new CQRS read model repository
func NewCQRSReadModelRepository(db *sqlx.DB) *CQRSReadModelRepository {
	return &CQRSReadModelRepository{db: db}
}

// GetBusinessObjectProjection retrieves a single BO from the read model
// No joins needed - data is pre-aggregated in projection
func (r *CQRSReadModelRepository) GetBusinessObjectProjection(
	ctx context.Context,
	tenantID string,
	boKey string,
) (*BusinessObjectProjection, error) {
	var bo BusinessObjectProjection

	query := `
		SELECT id, key, tenant_id, name, display_name, description, category,
		       is_active, field_count, instance_count, created_at, updated_at,
		       last_modified_by, last_event_id, last_event_type, correlation_id
		FROM bo_projections
		WHERE tenant_id = $1 AND key = $2
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &bo, query, tenantID, boKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get BO projection: %w", err)
	}

	return &bo, nil
}

// ListBusinessObjectProjections retrieves a paginated list of BOs from read model
// Optimized with indexed columns: tenant_id, is_active, updated_at (for sorting)
func (r *CQRSReadModelRepository) ListBusinessObjectProjections(
	ctx context.Context,
	tenantID string,
	offset, limit int,
) ([]*BusinessObjectProjection, int64, error) {
	var bos []*BusinessObjectProjection

	// Get results
	query := `
		SELECT id, key, tenant_id, name, display_name, description, category,
		       is_active, field_count, instance_count, created_at, updated_at,
		       last_modified_by, last_event_id, last_event_type, correlation_id
		FROM bo_projections
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`

	err := r.db.SelectContext(ctx, &bos, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list BO projections: %w", err)
	}

	// Get total count (separate query - read model allows this optimization)
	var total int64
	countQuery := `
		SELECT COUNT(*) FROM bo_projections
		WHERE tenant_id = $1 AND is_active = true
	`
	err = r.db.GetContext(ctx, &total, countQuery, tenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count BO projections: %w", err)
	}

	return bos, total, nil
}

// ============================================================================
// PROJECTION UPDATER - Subscribes to events and updates read model
// ============================================================================

// CQRSProjectionUpdater updates read model projections from domain events
// Implements eventual consistency - reads are eventually consistent with writes
type CQRSProjectionUpdater struct {
	db *sqlx.DB
}

// NewCQRSProjectionUpdater creates a new projection updater
func NewCQRSProjectionUpdater(db *sqlx.DB) *CQRSProjectionUpdater {
	return &CQRSProjectionUpdater{db: db}
}

// HandleBOCreatedEvent updates projection when BO is created
// Called when BOCreated event is published from command handler
func (pu *CQRSProjectionUpdater) HandleBOCreatedEvent(
	ctx context.Context,
	event *Event,
	boData map[string]interface{},
) error {
	// Upsert into read model (denormalized projection)
	query := `
		INSERT INTO bo_projections (
			id, key, tenant_id, name, display_name, description, category,
			is_active, field_count, instance_count, created_at, updated_at,
			last_modified_by, last_event_id, last_event_type, correlation_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		ON CONFLICT (tenant_id, key) DO UPDATE SET
			name = EXCLUDED.name,
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at,
			last_modified_by = EXCLUDED.last_modified_by,
			last_event_id = EXCLUDED.last_event_id,
			last_event_type = EXCLUDED.last_event_type,
			correlation_id = EXCLUDED.correlation_id
	`

	boID := getStringFromMap(boData, "id")
	boKey := getStringFromMap(boData, "key")
	name := getStringFromMap(boData, "name")
	displayName := getStringFromMap(boData, "displayName")
	description := getStringFromMap(boData, "description")
	category := getStringFromMap(boData, "category")

	_, err := pu.db.ExecContext(ctx, query,
		boID,                // id
		boKey,               // key
		event.TenantID,      // tenant_id
		name,                // name
		displayName,         // display_name
		description,         // description
		category,            // category
		true,                // is_active
		0,                   // field_count (will be aggregated separately)
		0,                   // instance_count (will be aggregated separately)
		time.Now(),          // created_at
		time.Now(),          // updated_at
		event.UserID,        // last_modified_by
		event.ID,            // last_event_id
		string(event.Type),  // last_event_type
		event.CorrelationID, // correlation_id
	)

	if err != nil {
		return fmt.Errorf("failed to update BO projection: %w", err)
	}

	return nil
}

// HandleBOUpdatedEvent updates projection when BO is updated
func (pu *CQRSProjectionUpdater) HandleBOUpdatedEvent(
	ctx context.Context,
	event *Event,
	boData map[string]interface{},
) error {
	// Same upsert logic as create - idempotent update
	return pu.HandleBOCreatedEvent(ctx, event, boData)
}

// HandleBODeletedEvent updates projection when BO is deleted
func (pu *CQRSProjectionUpdater) HandleBODeletedEvent(
	ctx context.Context,
	event *Event,
) error {
	boKey := getStringFromMap(event.Data.(map[string]interface{}), "key")

	query := `
		UPDATE bo_projections
		SET is_active = false, updated_at = $1, last_event_id = $2, last_event_type = $3
		WHERE tenant_id = $4 AND key = $5
	`

	_, err := pu.db.ExecContext(ctx, query,
		time.Now(),
		event.ID,
		string(event.Type),
		event.TenantID,
		boKey,
	)

	if err != nil {
		return fmt.Errorf("failed to mark BO projection as deleted: %w", err)
	}

	return nil
}

// ============================================================================
// CQRS QUERY SERVICE - Read-only queries using projections
// ============================================================================

// CQRSQueryService provides all read-only operations
// Uses read model projections for optimized queries
type CQRSQueryService struct {
	repo *CQRSReadModelRepository
}

// NewCQRSQueryService creates a new CQRS query service
func NewCQRSQueryService(db *sqlx.DB) *CQRSQueryService {
	return &CQRSQueryService{
		repo: NewCQRSReadModelRepository(db),
	}
}

// GetBusinessObject returns a single BO with minimal latency
// Uses pre-aggregated projection - O(1) query
func (s *CQRSQueryService) GetBusinessObject(
	ctx context.Context,
	tenantID string,
	boKey string,
) (*BusinessObjectProjection, error) {
	return s.repo.GetBusinessObjectProjection(ctx, tenantID, boKey)
}

// ListBusinessObjects returns a paginated list of BOs
// Uses pre-aggregated projection - O(n) scan with index
func (s *CQRSQueryService) ListBusinessObjects(
	ctx context.Context,
	tenantID string,
	offset, limit int,
) ([]*BusinessObjectProjection, int64, error) {
	return s.repo.ListBusinessObjectProjections(ctx, tenantID, offset, limit)
}

// GetBusinessObjectStats returns aggregated statistics
// Example of advanced read model query
func (s *CQRSQueryService) GetBusinessObjectStats(
	ctx context.Context,
	tenantID string,
) (map[string]interface{}, error) {
	var stats struct {
		TotalBOs       int64     `db:"total_bos"`
		ActiveBOs      int64     `db:"active_bos"`
		TotalInstances int64     `db:"total_instances"`
		LastUpdated    time.Time `db:"last_updated"`
	}

	query := `
		SELECT
			COUNT(*) as total_bos,
			COUNT(CASE WHEN is_active THEN 1 END) as active_bos,
			COALESCE(SUM(instance_count), 0) as total_instances,
			MAX(updated_at) as last_updated
		FROM bo_projections
		WHERE tenant_id = $1
	`

	err := s.repo.db.GetContext(ctx, &stats, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get BO stats: %w", err)
	}

	return map[string]interface{}{
		"total_bos":       stats.TotalBOs,
		"active_bos":      stats.ActiveBOs,
		"total_instances": stats.TotalInstances,
		"last_updated":    stats.LastUpdated,
	}, nil
}

// ============================================================================
// IDEMPOTENCY STORE - Prevent duplicate command processing
// ============================================================================

// IdempotencyRecord tracks processed commands for deduplication
type IdempotencyRecord struct {
	CorrelationID string    `db:"correlation_id"`
	CommandType   string    `db:"command_type"`
	ResultID      string    `db:"result_id"`
	ProcessedAt   time.Time `db:"processed_at"`
	ExpiresAt     time.Time `db:"expires_at"` // TTL for cleanup
}

// CQRSIdempotencyRepository manages idempotency records
type CQRSIdempotencyRepository struct {
	db *sqlx.DB
}

// NewCQRSIdempotencyRepository creates a new idempotency repository
func NewCQRSIdempotencyRepository(db *sqlx.DB) *CQRSIdempotencyRepository {
	return &CQRSIdempotencyRepository{db: db}
}

// RecordCommandExecution records a command for idempotency checking
func (ir *CQRSIdempotencyRepository) RecordCommandExecution(
	ctx context.Context,
	correlationID string,
	commandType string,
	resultID string,
) error {
	query := `
		INSERT INTO idempotency_records (correlation_id, command_type, result_id, processed_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (correlation_id) DO NOTHING
	`

	now := time.Now()
	expiresAt := now.Add(24 * time.Hour) // Keep for 24 hours

	_, err := ir.db.ExecContext(ctx, query,
		correlationID,
		commandType,
		resultID,
		now,
		expiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to record command execution: %w", err)
	}

	return nil
}

// IsCommandProcessed checks if a command was already processed
// Returns (processed bool, resultID string, error)
func (ir *CQRSIdempotencyRepository) IsCommandProcessed(
	ctx context.Context,
	correlationID string,
) (bool, string, error) {
	var resultID string

	query := `
		SELECT result_id FROM idempotency_records
		WHERE correlation_id = $1 AND expires_at > NOW()
	`

	err := ir.db.GetContext(ctx, &resultID, query, correlationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", fmt.Errorf("failed to check command idempotency: %w", err)
	}

	return true, resultID, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// getStringFromMap safely extracts string from map
func getStringFromMap(data map[string]interface{}, key string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GenerateProjectionID generates a unique ID for projections
func GenerateProjectionID() string {
	return uuid.New().String()
}
