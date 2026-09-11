package reports

import (
	"time"

	"github.com/google/uuid"
)

// ReportFolder represents a user-scoped organizational folder.
type ReportFolder struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	UserID    string     `json:"user_id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	Name      string     `json:"name"`
	ItemCount int        `json:"item_count"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ReportFolderItem represents a report assigned to a folder.
type ReportFolderItem struct {
	FolderID   uuid.UUID `json:"folder_id"`
	TemplateID uuid.UUID `json:"template_id"`
	AddedAt    time.Time `json:"added_at"`
}
