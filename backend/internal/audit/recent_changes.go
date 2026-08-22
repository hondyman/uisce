package audit

import (
	"time"
)

// RecentChange represents a summary of a recent entity change
type RecentChange struct {
	EntityType   string    `json:"entity_type"`
	EntityID     string    `json:"entity_id"`
	EntityName   string    `json:"entity_name,omitempty"`
	ChangeType   string    `json:"change_type"`
	ChangedBy    string    `json:"changed_by"`
	SystemFrom   time.Time `json:"system_from"`
	VersionCount int       `json:"version_count"`
}
