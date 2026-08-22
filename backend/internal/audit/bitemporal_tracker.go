package audit

import (
	"context"
	"fmt"
	"time"
)

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

// BitemporalTracker manages bitemporal audit tracking
// Deprecated: Trino audit chain has been removed. All methods return empty results.
type BitemporalTracker struct{}

// NewBitemporalTracker creates a new bitemporal tracker
// Deprecated: Returns a stub tracker with no-op methods
func NewBitemporalTracker() *BitemporalTracker {
	return &BitemporalTracker{}
}

// TrackEntityChange records a change with bitemporal semantics
// Deprecated: Always returns nil (no-op)
func (bt *BitemporalTracker) TrackEntityChange(ctx context.Context, change EntityChange) error {
	return nil
}

// GetEntityAtTime retrieves entity state at a specific point in time
// Deprecated: Always returns nil, nil (not implemented)
func (bt *BitemporalTracker) GetEntityAtTime(ctx context.Context, entityType, entityID string, asOf time.Time) (*EntitySnapshot, error) {
	return nil, nil
}

// GetEntityHistory retrieves all versions of an entity
// Deprecated: Always returns empty slice
func (bt *BitemporalTracker) GetEntityHistory(ctx context.Context, entityType, entityID string, filters HistoryFilters) ([]EntitySnapshot, error) {
	return []EntitySnapshot{}, nil
}

// RestoreEntityToTime restores an entity to a previous state
// Deprecated: Always returns an error indicating feature is disabled
func (bt *BitemporalTracker) RestoreEntityToTime(ctx context.Context, entityType, entityID string, restoreToTime time.Time, reason string) error {
	return fmt.Errorf("bitemporal restore is disabled: Trino audit chain removed")
}

// GetRecentChanges returns recent changes for an entity type
// Deprecated: Always returns empty slice
func (bt *BitemporalTracker) GetRecentChanges(ctx context.Context, from, to *time.Time, entityType string, limit int) ([]EntitySnapshot, error) {
	return []EntitySnapshot{}, nil
}
