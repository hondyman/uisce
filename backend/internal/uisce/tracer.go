package uisce

import (
	"context"
	"encoding/json"
	"time"
)

// DebugStep represents one "Stop" in the pipeline
type DebugStep struct {
	FilterName    string                 `json:"filterName"`
	Stage         string                 `json:"stage"`         // PRE_TRADE, POST_TRADE
	InputSnapshot map[string]interface{} `json:"inputSnapshot"` // Data BEFORE rule
	Status        string                 `json:"status"`        // PASS / FAIL
	ErrorDetails  string                 `json:"errorDetails,omitempty"`
	DurationMs    int64                  `json:"durationMs"`
}

// TraceResult is the full report returned to the UI
type TraceResult struct {
	TradeID string      `json:"tradeId"`
	Steps   []DebugStep `json:"steps"`
	Success bool        `json:"success"`
}

// Tracer handles the recording of pipeline steps
type Tracer struct {
	Steps []DebugStep
}

func NewTracer() *Tracer {
	return &Tracer{
		Steps: []DebugStep{},
	}
}

func (t *Tracer) RecordStep(name string, state map[string]interface{}, err error, duration time.Duration) {
	// Deep copy state to snapshot
	snapshot := deepCopy(state)

	status := "PASS"
	details := ""
	if err != nil {
		status = "FAIL"
		details = err.Error()
	}

	t.Steps = append(t.Steps, DebugStep{
		FilterName:    name,
		InputSnapshot: snapshot,
		Status:        status,
		ErrorDetails:  details,
		DurationMs:    duration.Milliseconds(),
	})
}

// Helper to deep copy map for snapshotting
func deepCopy(src map[string]interface{}) map[string]interface{} {
	dest := make(map[string]interface{})
	bytes, _ := json.Marshal(src)
	_ = json.Unmarshal(bytes, &dest)
	return dest
}

// Filter defines a single processing step
type Filter interface {
	Name() string
	Purify(ctx context.Context, data map[string]interface{}) error
}
