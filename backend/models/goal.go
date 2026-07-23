package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Goal represents a user-defined target to track over time.
type Goal struct {
	ID           uuid.UUID      `db:"id" json:"id"`
	OwnerUserID  string         `db:"owner_user_id" json:"owner_user_id"`
	Name         string         `db:"name" json:"name"`
	Description  sql.NullString `db:"description" json:"description,omitempty"`
	DatasourceID string         `db:"datasource_id" json:"datasource_id"`
	QueryID      uuid.UUID      `db:"query_id" json:"query_id"`
	Metric       string         `db:"metric" json:"metric"`
	Condition    string         `db:"condition" json:"condition"`
	Frequency    string         `db:"frequency" json:"frequency"`
	LastChecked  *time.Time     `db:"last_checked" json:"last_checked,omitempty"`
	Status       string         `db:"status" json:"status"` // e.g., 'met', 'missed', 'trending_up'
}
