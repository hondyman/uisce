package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/trino"

	"github.com/google/uuid"
)

// trinoTimestampFormat is SQL-style timestamp format with microseconds and UTC
// Trino's TIMESTAMP(6) WITH TIME ZONE requires this format: "2006-01-02 15:04:05.000000 UTC"
const trinoTimestampFormat = "2006-01-02 15:04:05.000000 UTC"

// EntityChange represents a change to an entity that should be tracked
type EntityChange struct {
	EntityType   string                 // "tenant", "instance", "connection", "product"
	EntityID     string                 // UUID of the entity
	ChangeType   string                 // "INSERT", "UPDATE", "DELETE", "RESTORE"
	ValidFrom    time.Time              // When this change is valid from (business time)
	ValidTo      *time.Time             // When this change is valid until (nil for current)
	EntityData   map[string]interface{} // Full entity snapshot
	ChangedBy    string                 // User ID or "system"
	ChangeReason string                 // Optional reason for the change
}

// EntitySnapshot represents a historical snapshot of an entity
type EntitySnapshot struct {
	VersionID    string                 `json:"version_id"`
	ValidFrom    time.Time              `json:"valid_from"`
	ValidTo      *time.Time             `json:"valid_to,omitempty"`
	SystemFrom   time.Time              `json:"system_from"`
	SystemTo     *time.Time             `json:"system_to,omitempty"`
	ChangeType   string                 `json:"change_type"`
	ChangedBy    string                 `json:"changed_by"`
	ChangeReason string                 `json:"change_reason,omitempty"`
	EntityData   map[string]interface{} `json:"entity_data"`
	IsCurrent    bool                   `json:"is_current"`
	IsDeleted    bool                   `json:"is_deleted"`
}

// HistoryFilters defines filters for querying entity history
type HistoryFilters struct {
	From           *time.Time // Filter by system_from >= this time
	To             *time.Time // Filter by system_from <= this time
	ValidFrom      *time.Time // Filter by valid_from >= this time
	ValidTo        *time.Time // Filter by valid_to <= this time
	IncludeDeleted bool       // Include deleted versions
	Limit          int        // Max number of results
	Offset         int        // Pagination offset
}

// BitemporalTracker manages bitemporal audit tracking in Iceberg
type BitemporalTracker struct {
	trinoClient *trino.Client
}

// NewBitemporalTracker creates a new bitemporal tracker
func NewBitemporalTracker(trinoClient *trino.Client) *BitemporalTracker {
	return &BitemporalTracker{
		trinoClient: trinoClient,
	}
}

// TrackEntityChange records a change with bitemporal semantics
func (bt *BitemporalTracker) TrackEntityChange(ctx context.Context, change EntityChange) error {
	// Validate input
	if err := bt.validateEntityChange(change); err != nil {
		return fmt.Errorf("invalid entity change: %w", err)
	}

	// Generate version ID
	versionID := uuid.New().String()
	systemFrom := time.Now()

	// Close previous version (set system_to)
	if err := bt.closePreviousVersion(ctx, change.EntityType, change.EntityID, systemFrom); err != nil {
		return fmt.Errorf("failed to close previous version: %w", err)
	}

	// Serialize entity data
	entityDataJSON, err := json.Marshal(change.EntityData)
	if err != nil {
		return fmt.Errorf("failed to marshal entity data: %w", err)
	}

	// Determine if this is a deletion
	isDeleted := change.ChangeType == "DELETE"

	// Insert new version
	tableName := bt.getTableName(change.EntityType)
	idColumn := bt.getEntityIDColumn(change.EntityType)

	validFromStr := change.ValidFrom.Format(trinoTimestampFormat)
	systemFromStr := systemFrom.Format(trinoTimestampFormat)

	validToExpr := "NULL"
	if change.ValidTo != nil {
		validToExpr = fmt.Sprintf("CAST('%s' AS TIMESTAMP(6) WITH TIME ZONE)", change.ValidTo.Format(trinoTimestampFormat))
	}

	// Escape single quotes in entity data JSON
	entityDataStr := strings.ReplaceAll(string(entityDataJSON), "'", "''")
	changeReasonStr := strings.ReplaceAll(change.ChangeReason, "'", "''")

	query := fmt.Sprintf(`
		INSERT INTO %s (
			%s, version_id, valid_from, valid_to, system_from, system_to,
			change_type, changed_by, change_reason, entity_data, is_current, is_deleted, created_at
		) VALUES (
			'%s',
			'%s',
			CAST('%s' AS TIMESTAMP(6) WITH TIME ZONE),
			%s,
			CAST('%s' AS TIMESTAMP(6) WITH TIME ZONE),
			NULL,
			'%s',
			'%s',
			'%s',
			'%s',
			true,
			%t,
			CAST('%s' AS TIMESTAMP(6) WITH TIME ZONE)
		)
	`, tableName, idColumn,
		change.EntityID,
		versionID,
		validFromStr,
		validToExpr,
		systemFromStr,
		change.ChangeType,
		change.ChangedBy,
		changeReasonStr,
		entityDataStr,
		isDeleted,
		systemFromStr,
	)

	_, err = bt.trinoClient.Execute(ctx, query)

	if err != nil {
		return fmt.Errorf("failed to insert new version: %w", err)
	}

	return nil
}

// GetEntityAtTime retrieves entity state at a specific point in time
func (bt *BitemporalTracker) GetEntityAtTime(ctx context.Context, entityType, entityID string, asOf time.Time) (*EntitySnapshot, error) {
	tableName := bt.getTableName(entityType)
	idColumn := bt.getEntityIDColumn(entityType)

	query := fmt.Sprintf(`
		SELECT 
			version_id, valid_from, valid_to, system_from, system_to,
			change_type, changed_by, change_reason, entity_data, is_current, is_deleted
		FROM %s
		WHERE %s = ?
		  AND system_from <= ?
		  AND (system_to IS NULL OR system_to > ?)
		  AND valid_from <= ?
		  AND (valid_to IS NULL OR valid_to > ?)
		ORDER BY system_from DESC
		LIMIT 1
	`, tableName, idColumn)

	asOfStr := asOf.Format(time.RFC3339Nano)
	rows, err := bt.trinoClient.Query(ctx, query, entityID, asOfStr, asOfStr, asOfStr, asOfStr)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity at time: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("entity not found at time %s", asOf)
	}

	snapshot, err := bt.scanEntitySnapshot(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan entity snapshot: %w", err)
	}

	return snapshot, nil
}

// GetEntityHistory retrieves all versions of an entity
func (bt *BitemporalTracker) GetEntityHistory(ctx context.Context, entityType, entityID string, filters HistoryFilters) ([]EntitySnapshot, error) {
	tableName := bt.getTableName(entityType)
	idColumn := bt.getEntityIDColumn(entityType)

	// Trino timestamp format
	const trinoTSFormat = "2006-01-02 15:04:05.000000 UTC"

	// Build query with filters - use inline SQL instead of placeholders
	query := fmt.Sprintf(`
		SELECT 
			version_id, valid_from, valid_to, system_from, system_to,
			change_type, changed_by, change_reason, entity_data, is_current, is_deleted
		FROM %s
		WHERE %s = '%s'
	`, tableName, idColumn, entityID)

	if filters.From != nil {
		query += fmt.Sprintf(" AND system_from >= CAST('%s' AS TIMESTAMP(6) WITH TIME ZONE)", filters.From.UTC().Format(trinoTSFormat))
	}

	if filters.To != nil {
		query += fmt.Sprintf(" AND system_from <= CAST('%s' AS TIMESTAMP(6) WITH TIME ZONE)", filters.To.UTC().Format(trinoTSFormat))
	}

	if filters.ValidFrom != nil {
		query += fmt.Sprintf(" AND valid_from >= CAST('%s' AS TIMESTAMP(6) WITH TIME ZONE)", filters.ValidFrom.UTC().Format(trinoTSFormat))
	}

	if filters.ValidTo != nil {
		query += fmt.Sprintf(" AND (valid_to IS NULL OR valid_to <= CAST('%s' AS TIMESTAMP(6) WITH TIME ZONE))", filters.ValidTo.UTC().Format(trinoTSFormat))
	}

	if !filters.IncludeDeleted {
		query += " AND is_deleted = false"
	}

	query += " ORDER BY system_from DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.Limit)
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	rows, err := bt.trinoClient.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity history: %w", err)
	}
	defer rows.Close()

	var snapshots []EntitySnapshot
	for rows.Next() {
		snapshot, err := bt.scanEntitySnapshot(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity snapshot: %w", err)
		}
		snapshots = append(snapshots, *snapshot)
	}

	return snapshots, nil
}

// RestoreEntityToTime restores an entity to a previous state
func (bt *BitemporalTracker) RestoreEntityToTime(ctx context.Context, entityType, entityID string, restoreToTime time.Time, reason string) error {
	// Get the entity state at the restore time
	snapshot, err := bt.GetEntityAtTime(ctx, entityType, entityID, restoreToTime)
	if err != nil {
		return fmt.Errorf("failed to get entity at restore time: %w", err)
	}

	// Track the restore as a new change
	// Extract changedBy from context if available, otherwise use "system"
	changedBy := "system"
	if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
		changedBy = userID
	}

	return bt.TrackEntityChange(ctx, EntityChange{
		EntityType:   entityType,
		EntityID:     entityID,
		ChangeType:   "RESTORE",
		ValidFrom:    time.Now(),
		EntityData:   snapshot.EntityData,
		ChangedBy:    changedBy,
		ChangeReason: fmt.Sprintf("Restored to state from %s. Reason: %s", restoreToTime.Format(time.RFC3339), reason),
	})
}

// Helper methods

func (bt *BitemporalTracker) validateEntityChange(change EntityChange) error {
	if change.EntityType == "" {
		return fmt.Errorf("entity_type is required")
	}
	if change.EntityID == "" {
		return fmt.Errorf("entity_id is required")
	}
	if change.ChangeType == "" {
		return fmt.Errorf("change_type is required")
	}
	if change.ValidFrom.IsZero() {
		return fmt.Errorf("valid_from is required")
	}
	if change.EntityData == nil {
		return fmt.Errorf("entity_data is required")
	}
	return nil
}

func (bt *BitemporalTracker) closePreviousVersion(ctx context.Context, entityType, entityID string, systemTo time.Time) error {
	tableName := bt.getTableName(entityType)
	idColumn := bt.getEntityIDColumn(entityType)

	// Iceberg UPDATE requires proper timestamp literals
	// Use CAST to convert the string to timestamp with time zone
	systemToStr := systemTo.Format(trinoTimestampFormat)
	query := fmt.Sprintf(`
		UPDATE %s
		SET system_to = CAST('%s' AS TIMESTAMP(6) WITH TIME ZONE), is_current = false
		WHERE %s = '%s' AND is_current = true
	`, tableName, systemToStr, idColumn, entityID)

	_, err := bt.trinoClient.Execute(ctx, query)
	return err
}

func (bt *BitemporalTracker) getTableName(entityType string) string {
	return fmt.Sprintf("iceberg.audit.%s_history", entityType)
}

func (bt *BitemporalTracker) getEntityIDColumn(entityType string) string {
	return fmt.Sprintf("%s_id", entityType)
}

func (bt *BitemporalTracker) scanEntitySnapshot(rows interface{}) (*EntitySnapshot, error) {
	// Type assert to sql.Rows (or whatever your Trino client returns)
	sqlRows, ok := rows.(interface {
		Scan(dest ...interface{}) error
	})
	if !ok {
		return nil, fmt.Errorf("invalid rows type")
	}

	var snapshot EntitySnapshot
	var entityDataJSON string
	var validToTime, systemToTime sql.NullTime

	err := sqlRows.Scan(
		&snapshot.VersionID,
		&snapshot.ValidFrom,
		&validToTime,
		&snapshot.SystemFrom,
		&systemToTime,
		&snapshot.ChangeType,
		&snapshot.ChangedBy,
		&snapshot.ChangeReason,
		&entityDataJSON,
		&snapshot.IsCurrent,
		&snapshot.IsDeleted,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable timestamps
	if validToTime.Valid {
		snapshot.ValidTo = &validToTime.Time
	}

	if systemToTime.Valid {
		snapshot.SystemTo = &systemToTime.Time
	}

	// Parse entity data JSON
	if err := json.Unmarshal([]byte(entityDataJSON), &snapshot.EntityData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal entity_data: %w", err)
	}

	return &snapshot, nil
}
