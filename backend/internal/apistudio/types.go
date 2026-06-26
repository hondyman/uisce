package apistudio

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// APIEndpoint defines a curated REST or GraphQL surface over a BO
type APIEndpoint struct {
	ID                uuid.UUID       `db:"id" json:"id"`
	Env               string          `db:"env" json:"env"`
	TenantID          string          `db:"tenant_id" json:"tenant_id"`
	Name              string          `db:"name" json:"name"`
	Path              string          `db:"path" json:"path"`
	Method            string          `db:"method" json:"method"`
	Type              string          `db:"type" json:"type"` // rest | graphql
	BOName            string          `db:"bo_name" json:"bo_name"`
	Fields            json.RawMessage `db:"fields" json:"fields"` // []string
	Filters           json.RawMessage `db:"filters" json:"filters"`
	Pagination        json.RawMessage `db:"pagination" json:"pagination"`
	AuthPolicy        *string         `db:"auth_policy" json:"auth_policy,omitempty"`
	Version           int             `db:"version" json:"version"`
	Status            string          `db:"status" json:"status"` // active | deprecated | retired
	SemanticVersion   string          `db:"semantic_version" json:"semantic_version"`
	PreviousVersionID *uuid.UUID      `db:"previous_version_id" json:"previous_version_id,omitempty"`
	OwnerTeam         string          `db:"owner_team" json:"owner_team"`
	DeprecatedAt      *time.Time      `db:"deprecated_at" json:"deprecated_at,omitempty"`
	RetiredAt         *time.Time      `db:"retired_at" json:"retired_at,omitempty"`
	RequestSchemaID   *string         `db:"request_schema_id" json:"request_schema_id,omitempty"`
	ResponseSchemaID  *string         `db:"response_schema_id" json:"response_schema_id,omitempty"`
	CreatedAt         time.Time       `db:"created_at" json:"created_at"`
	CreatedBy         string          `db:"created_by" json:"created_by"`
}

// APICatalog groups endpoints for a specific consumer
type APICatalog struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Env         string    `db:"env" json:"env"`
	TenantID    string    `db:"tenant_id" json:"tenant_id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	CreatedBy   string    `db:"created_by" json:"created_by"`
}

// APICatalogEntry maps an endpoint to a catalog
type APICatalogEntry struct {
	ID                 uuid.UUID       `db:"id" json:"id"`
	CatalogID          uuid.UUID       `db:"catalog_id" json:"catalog_id"`
	EndpointID         uuid.UUID       `db:"endpoint_id" json:"endpoint_id"`
	PathOverride       *string         `db:"path_override" json:"path_override,omitempty"`
	AuthPolicyOverride *string         `db:"auth_policy_override" json:"auth_policy_override,omitempty"`
	RateLimit          json.RawMessage `db:"rate_limit" json:"rate_limit"`
	Enabled            bool            `db:"enabled" json:"enabled"`
}

// APITest defines a test for an API endpoint
type APITest struct {
	ID         uuid.UUID       `db:"id" json:"id"`
	Env        string          `db:"env" json:"env"`
	TenantID   string          `db:"tenant_id" json:"tenant_id"`
	EndpointID uuid.UUID       `db:"endpoint_id" json:"endpoint_id"`
	Name       string          `db:"name" json:"name"`
	Type       string          `db:"type" json:"type"` // contract | latency | pii | regression
	Definition json.RawMessage `db:"definition" json:"definition"`
	CreatedAt  time.Time       `db:"created_at" json:"created_at"`
	CreatedBy  string          `db:"created_by" json:"created_by"`
	Enabled    bool            `db:"enabled" json:"enabled"`
}

// APITestRun tracks test execution results
type APITestRun struct {
	ID         uuid.UUID       `db:"id" json:"id"`
	APITestID  uuid.UUID       `db:"api_test_id" json:"api_test_id"`
	Env        string          `db:"env" json:"env"`
	TenantID   string          `db:"tenant_id" json:"tenant_id"`
	Status     string          `db:"status" json:"status"` // pending | running | passed | failed
	StartedAt  *time.Time      `db:"started_at" json:"started_at"`
	FinishedAt *time.Time      `db:"finished_at" json:"finished_at"`
	Result     json.RawMessage `db:"result" json:"result"`
	Logs       []string        `db:"logs" json:"logs"`
}
