package semantic

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SemanticObject represents a versioned semantic object
type SemanticObject struct {
	ID        string          `json:"id" db:"id"`
	Version   int             `json:"version" db:"version"`
	Env       string          `json:"env" db:"env"`
	TenantID  *string         `json:"tenant_id" db:"tenant_id"`
	Type      string          `json:"type" db:"type"`
	Payload   json.RawMessage `json:"payload" db:"payload"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	CreatedBy string          `json:"created_by" db:"created_by"`
}

// SemanticHead tracks the latest version of an object
type SemanticHead struct {
	ID             string  `json:"id" db:"id"`
	Env            string  `json:"env" db:"env"`
	TenantID       *string `json:"tenant_id" db:"tenant_id"`
	Type           string  `json:"type" db:"type"`
	CurrentVersion int     `json:"current_version" db:"current_version"`
}

// ChangeSet represents a collection of changes to be reviewed
type ChangeSet struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Env       string    `json:"env" db:"env"`
	TenantID  *string   `json:"tenant_id" db:"tenant_id"`
	Author    string    `json:"author" db:"author"`
	Status    string    `json:"status" db:"status"` // draft | in_review | approved | rejected | promoted
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ChangeSetItem represents a single object change within a ChangeSet
type ChangeSetItem struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	ChangeSetID uuid.UUID       `json:"change_set_id" db:"change_set_id"`
	ObjectID    string          `json:"object_id" db:"object_id"`
	ObjectType  string          `json:"object_type" db:"object_type"`
	OldVersion  int             `json:"old_version" db:"old_version"`
	NewVersion  int             `json:"new_version" db:"new_version"`
	Payload     json.RawMessage `json:"payload" db:"payload"`
}

// SemanticTest represents a test definition
type SemanticTest struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	Env        string          `json:"env" db:"env"`
	TenantID   *string         `json:"tenant_id" db:"tenant_id"`
	ScopeType  string          `json:"scope_type" db:"scope_type"`
	ScopeID    string          `json:"scope_id" db:"scope_id"`
	Name       string          `json:"name" db:"name"`
	Type       string          `json:"type" db:"type"` // contract | entitlement | regression | calc
	Definition json.RawMessage `json:"definition" db:"definition"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	CreatedBy  string          `json:"created_by" db:"created_by"`
	Enabled    bool            `json:"enabled" db:"enabled"`
}

// TestResult represents the outcome of a semantic test run
type TestResult struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	TestID     uuid.UUID       `json:"test_id" db:"test_id"`
	Env        string          `json:"env" db:"env"`
	TenantID   *string         `json:"tenant_id" db:"tenant_id"`
	Status     string          `json:"status" db:"status"` // pending | running | passed | failed
	StartedAt  time.Time       `json:"started_at" db:"started_at"`
	FinishedAt time.Time       `json:"finished_at" db:"finished_at"`
	Result     json.RawMessage `json:"result" db:"result"`
}

// SemanticDiffDTO represents the unmarshalled diff structure
type SemanticDiffDTO map[string]struct {
	Changes []SemanticDiffChange `json:"changes"`
}

type SemanticDiffChange struct {
	Path string      `json:"path"`
	Old  interface{} `json:"old"`
	New  interface{} `json:"new"`
	Type string      `json:"type"` // added | removed | modified
}
