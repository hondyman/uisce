package financial

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FinancialTool represents a tool definition stored in the database
type FinancialTool struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	Name             string          `json:"name" db:"name"`
	Description      string          `json:"description" db:"description"`
	ParametersSchema json.RawMessage `json:"parameters_schema" db:"parameters_schema"`
	HandlerType      string          `json:"handler_type" db:"handler_type"` // 'internal', 'script', 'api'
	HandlerConfig    json.RawMessage `json:"handler_config" db:"handler_config"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}
