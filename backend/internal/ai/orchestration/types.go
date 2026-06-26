package orchestration

import (
	"context"
	"encoding/json"
	"time"
)

// RequestStatus constants
const (
	StatusPending = "PENDING"
	StatusRunning = "RUNNING"
	StatusSuccess = "SUCCESS"
	StatusFailed  = "FAILED"
)

// RequestTypes
const (
	TypeChangeSet    = "CHANGESET"
	TypeIncident     = "INCIDENT"
	TypeRisk         = "RISK"
	TypeDrift        = "DRIFT"
	TypeSLO          = "SLO"
	TypeBusinessTerm = "BUSINESS_TERM"
)

// AIRequest represents a task in the queue
type AIRequest struct {
	ID        string          `json:"id" db:"id"`
	Type      string          `json:"type" db:"type"`
	Payload   json.RawMessage `json:"payload" db:"payload"`
	Status    string          `json:"status" db:"status"`
	Output    json.RawMessage `json:"output" db:"output"`
	Error     string          `json:"error" db:"error"`
	Attempts  int             `json:"attempts" db:"attempts"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// AIStrategy defines how to handle specific AI task types
type AIStrategy interface {
	// BuildPrompt constructs the system and user prompts from the payload
	BuildPrompt(payload json.RawMessage) (string, string)

	// Validate checks if the raw LLM output is valid for this strategy (e.g. schema check)
	// Returns true if valid, false otherwise
	Validate(raw string) bool

	// Parse converts valid raw output into a structured result (to be stored in AIRequest.Output)
	Parse(raw string) (any, error)

	// Apply executes any side effects (e.g. creating a ChangeSet, updating a Job)
	// This runs after successful Parse
	Apply(ctx context.Context, output any) error
}
