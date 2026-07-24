package tiering

import (
	"time"

	"github.com/google/uuid"
)

// StorageEventType defines the type of storage event
type StorageEventType string

const (
	EventMovedToCold    StorageEventType = "MOVED_TO_COLD"
	EventMovedToHot     StorageEventType = "MOVED_TO_HOT"
	EventMovedToArchive StorageEventType = "MOVED_TO_ARCHIVE"
	EventClassChanged   StorageEventType = "CLASS_CHANGED"
)

// StorageEvent represents a significant change in data storage tier
type StorageEvent struct {
	ID        uuid.UUID        `json:"id"`
	TenantID  string           `json:"tenant_id"`
	Type      StorageEventType `json:"type"`
	TableName string           `json:"table_name"`
	OldTier   StorageTier      `json:"old_tier"`
	NewTier   StorageTier      `json:"new_tier"`
	Timestamp time.Time        `json:"timestamp"`
	Metadata  map[string]any   `json:"metadata,omitempty"`
}

// StorageEventListener defines the interface for handling storage events
type StorageEventListener interface {
	OnStorageEvent(event StorageEvent) error
}
