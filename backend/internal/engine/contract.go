package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EvalType ensures we track what invoked the engine for audit purposes
type EvalType string

const (
	EvalTypeCalculation EvalType = "CALCULATION"
	EvalTypeValidation  EvalType = "VALIDATION"
	EvalTypeWorkflow    EvalType = "WORKFLOW"
)

// EvalContext strictly enforces Rule 7 (Tenant Security Mandate)
type EvalContext struct {
	TenantID      uuid.UUID         `json:"tenant_id"`
	UserID        uuid.UUID         `json:"user_id"`
	TransactionID uuid.UUID         `json:"transaction_id"`
	AsOfDate      time.Time         `json:"as_of_date"`
	Metadata      map[string]string `json:"metadata"`
}

// ExecutionPayload contains the DSL/AST resolved from the Graph
type ExecutionPayload struct {
	Type         EvalType
	ASTPayload   []byte          // The compiled graph nodes/CUE schema
	InputVectors json.RawMessage // The raw data to process
	RequiresOLAP bool            // True = Route to StarRocks, False = Route to WASM
	CacheHash    string          // L3 Cache Identifier (SHA-256 of AST + Inputs)
}

// EvalResult is the standardized output across all engines
type EvalResult struct {
	OutputData   json.RawMessage `json:"output_data"`
	IsCompliant  bool            `json:"is_compliant"` // Used by Validation & Workflow
	LineageTrace []string        `json:"lineage_trace"`
	ExecutionMs  float64         `json:"execution_ms"`
	ExecutedVia  string          `json:"executed_via"` // "WASM", "STARROCKS", "CACHE_L1", etc.
}

// UnifiedDispatcher is the core port injected into workflows and business objects
type UnifiedDispatcher interface {
	Evaluate(ctx context.Context, evalCtx EvalContext, payload ExecutionPayload) (*EvalResult, error)
}
