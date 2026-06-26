package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExportJob represents a request for an asynchronous data export.
type ExportJob struct {
	ID        uuid.UUID       `db:"id"`
	CreatedAt time.Time       `db:"created_at"`
	UserID    string          `db:"user_id"`
	Request   json.RawMessage `db:"request"`
	Status    string          `db:"status"` // queued, running, succeeded, failed
	ResultURL *string         `db:"result_url"`
	Error     *string         `db:"error"`
}
