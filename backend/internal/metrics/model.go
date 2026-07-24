package metrics

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MetricDefinition represents a metric definition stored in the database
type MetricDefinition struct {
	ID                  uuid.UUID       `json:"id" db:"id"`
	Name                string          `json:"name" db:"name"`
	DisplayName         string          `json:"display_name" db:"display_name"`
	Description         string          `json:"description" db:"description"`
	Domain              string          `json:"domain" db:"domain"`
	Granularity         string          `json:"granularity" db:"granularity"`
	AggregationFunction string          `json:"aggregation_function" db:"aggregation_function"`
	BaseQuery           string          `json:"base_query" db:"base_query"`
	Dimensions          json.RawMessage `json:"dimensions" db:"dimensions"`
	SLAConfig           json.RawMessage `json:"sla_config" db:"sla_config"`
	Owner               string          `json:"owner" db:"owner"`
	CreatedAt           time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at" db:"updated_at"`
}
