package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"go.uber.org/zap"
)

// ... existing code ...

// ReadableError represents a structured, human-readable validation error
type ReadableError struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
	Why      string `json:"why,omitempty"`
	Severity string `json:"severity,omitempty"`
}

// SimulationResult captures the output of a CUE simulation
type SimulationResult struct {
	Success    bool                   `json:"success"`
	ResultData map[string]interface{} `json:"result_data"`
	Errors     []string               `json:"errors"`
	Messages   []ReadableError        `json:"messages"`
}

// ExtractReadableErrors traverses the CUE value to find structured error objects
func ExtractReadableErrors(val cue.Value) []ReadableError {
	var messages []ReadableError

	// Walk(before func(Value) bool, after func(Value))
	val.Walk(func(v cue.Value) bool {
		// Check if this node has an "error" field
		errVal := v.LookupPath(cue.ParsePath("error"))
		if errVal.Exists() && errVal.Kind() == cue.StringKind {
			// Found a structured error!
			msg, _ := errVal.String()

			fix := ""
			if f := v.LookupPath(cue.ParsePath("fix")); f.Exists() {
				fix, _ = f.String()
			}

			why := ""
			if w := v.LookupPath(cue.ParsePath("why")); w.Exists() {
				why, _ = w.String()
			}

			severity := "error"
			if s := v.LookupPath(cue.ParsePath("severity")); s.Exists() {
				severity, _ = s.String()
			}

			// Path: remove "record." prefix if present
			path := v.Path().String()

			messages = append(messages, ReadableError{
				Field:    path,
				Message:  msg,
				Fix:      fix,
				Why:      why,
				Severity: severity,
			})

			// Don't validate/walk inside the error message string itself
			return false
		}
		return true
	}, nil)

	return messages
}

// SimulateRule evaluates a rule against data and returns the full unified result for inspection
func (e *CueEngine) SimulateRule(ctx context.Context, script string, data map[string]interface{}) (*SimulationResult, error) {
	c := cuecontext.New()
	val := c.CompileString(script)
	if val.Err() != nil {
		return &SimulationResult{
			Success: false,
			Errors:  []string{fmt.Sprintf("Script compilation failed: %v", val.Err())},
		}, nil
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	// Unify record: <data> with the script
	dataVal := c.CompileString(fmt.Sprintf("record: %s", string(dataBytes)))
	if dataVal.Err() != nil {
		return &SimulationResult{
			Success: false,
			Errors:  []string{fmt.Sprintf("Data compilation failed: %v", dataVal.Err())},
		}, nil
	}

	res := val.Unify(dataVal)

	// Validate
	success := true
	var errorMsgs []string
	if err := res.Validate(); err != nil {
		success = false
		// Extract multiple errors
		for _, e := range errors.Errors(err) {
			errorMsgs = append(errorMsgs, e.Error())
		}
	}

	// Decode full result to map
	var out map[string]interface{}
	if err := res.Decode(&out); err != nil {
		if success {
			return nil, fmt.Errorf("failed to decode result: %w", err)
		}
	}

	messages := ExtractReadableErrors(res)

	return &SimulationResult{
		Success:    success,
		ResultData: out,
		Errors:     errorMsgs,
		Messages:   messages,
	}, nil
}

// CueValidationResult represents the result of a usage of Cue for validation
type CueValidationResult struct {
	IsValid  bool   `json:"is_valid"`
	Message  string `json:"message,omitempty"`
	Severity string `json:"severity"` // error, warning, info
}

// CueEngine provides CUE-based expression evaluation
type CueEngine struct {
	logger *zap.Logger
	mu     sync.Mutex
}

// NewCueEngine creates a new CUE engine
func NewCueEngine() *CueEngine {
	logger, _ := zap.NewProduction()
	return &CueEngine{
		logger: logger,
	}
}

// EvaluateValidation runs a CUE validation script against data
func (e *CueEngine) EvaluateValidation(ctx context.Context, script string, data map[string]interface{}) (*CueValidationResult, error) {
	start := time.Now()
	defer func() {
		// e.logger.Debug("Cue validation executed", zap.Duration("duration", time.Since(start)))
		_ = start
	}()

	c := cuecontext.New()

	// Compile the user script (schema)
	val := c.CompileString(script)
	if val.Err() != nil {
		return &CueValidationResult{
			IsValid:  false,
			Message:  fmt.Sprintf("Script compilation failed: %v", val.Err()),
			Severity: "error",
		}, nil
	}

	// Unify with data
	// We expect the script to define validation rules that apply to the data.
	// Typically, we can wrap the data in a struct matching the schema, or
	// unify specifically.
	// Convention: The script defines constraints on a 'record' field, or at root.
	// Let's assume the script defines constraints. We unify 'record: <data>' with it.

	// Convert data to JSON to ensure clean types for CUE
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	// Create a CUE value from the data, wrapped in "record" key if the script expects it,
	// or we can allow the script to be flexible.
	// Starlark engine used 'record = context.record'.
	// Let's enforce that the input data corresponds to the schema.
	// We will fill the value of the script with the data.

	// Option 1: script is 'record: { ... constraints ... }', we unify with 'record: <data>'
	// Option 2: script is '#Schema: { ... }', and we unify.
	// Let's go with a simple unification model: Script + Data must be valid.

	// Ideally, the user script looks like:
	// record: {
	//    field: > 10
	// }

	// We will wrap the input data into a scope definition:
	// dataScope: {
	//    record: <actual data>
	// }

	dataVal := c.CompileString(fmt.Sprintf("record: %s", string(dataBytes)))
	if dataVal.Err() != nil {
		return nil, fmt.Errorf("failed to compile data value: %w", dataVal.Err())
	}

	// Unify script and data
	res := val.Unify(dataVal)
	if err := res.Validate(); err != nil {
		return &CueValidationResult{
			IsValid:  false,
			Message:  fmt.Sprintf("Validation failed: %v", err),
			Severity: "error",
		}, nil
	}

	// Check if the script proactively sets `valid: false` or similar?
	// CUE philosophy: if it unifies, it is valid.
	// However, users might want to output a message.
	// Let's check for an output field `result: { valid: bool, message: string }` if present.
	// If not present, unification success == IsValid: true.

	resultVal := res.LookupPath(cue.ParsePath("result"))
	if resultVal.Exists() {
		var customRes struct {
			Valid   bool   `json:"valid"`
			Message string `json:"message"`
		}
		if err := resultVal.Decode(&customRes); err == nil {
			severity := "error"
			if customRes.Valid {
				severity = "info"
			}
			return &CueValidationResult{
				IsValid:  customRes.Valid,
				Message:  customRes.Message,
				Severity: severity,
			}, nil
		}
	}

	// Default: unification success implies validity
	return &CueValidationResult{
		IsValid:  true,
		Severity: "info",
	}, nil
}
