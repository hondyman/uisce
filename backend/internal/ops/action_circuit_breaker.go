package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CircuitBreakerAction toggles circuit breaker state
type CircuitBreakerAction struct {
	store Store
}

// CircuitBreakerParams are parameters for circuit breaker toggle
type CircuitBreakerParams struct {
	CircuitID    string `json:"circuit_id"`
	TargetState  string `json:"target_state"`            // open, closed, half-open
	DurationSecs int    `json:"duration_secs,omitempty"` // How long to hold state
}

// CircuitBreakerResult tracks circuit breaker state change
type CircuitBreakerResult struct {
	CircuitID       string    `json:"circuit_id"`
	PreviousState   string    `json:"previous_state"`
	NewState        string    `json:"new_state"`
	Duration        int       `json:"duration_secs"`
	ReuquestBlocked int       `json:"requests_blocked"`
	Timestamp       time.Time `json:"timestamp"`
}

// NewCircuitBreakerAction creates a new circuit breaker action
func NewCircuitBreakerAction(store Store) *CircuitBreakerAction {
	return &CircuitBreakerAction{store: store}
}

// ID returns the action type identifier
func (a *CircuitBreakerAction) ID() string {
	return "circuit_breaker_toggle"
}

// Name returns the human-readable action name
func (a *CircuitBreakerAction) Name() string {
	return "Toggle Circuit Breaker"
}

// Validate checks if circuit breaker parameters are valid
func (a *CircuitBreakerAction) Validate(ctx context.Context, params json.RawMessage) error {
	var p CircuitBreakerParams
	if err := json.Unmarshal(params, &p); err != nil {
		return fmt.Errorf("invalid circuit breaker parameters: %w", err)
	}

	if p.CircuitID == "" {
		return fmt.Errorf("circuit_id is required")
	}

	validStates := map[string]bool{
		"open":      true,
		"closed":    true,
		"half-open": true,
	}

	if !validStates[p.TargetState] {
		return fmt.Errorf("target_state must be one of: open, closed, half-open")
	}

	return nil
}

// Execute toggles the circuit breaker
// In production, this would call circuit breaker framework (Hystrix, Polly, etc)
func (a *CircuitBreakerAction) Execute(ctx context.Context, params json.RawMessage) (map[string]interface{}, error) {
	var p CircuitBreakerParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Simulate circuit breaker state change
	// In production: Call circuit breaker framework API
	previousState := "closed" // Simulated previous state

	result := CircuitBreakerResult{
		CircuitID:       p.CircuitID,
		PreviousState:   previousState,
		NewState:        p.TargetState,
		Duration:        p.DurationSecs,
		ReuquestBlocked: 150, // Simulated blocked requests
		Timestamp:       time.Now(),
	}

	// Simulate blocking some requests
	time.Sleep(100 * time.Millisecond)

	return map[string]interface{}{
		"circuit_id":       result.CircuitID,
		"previous_state":   result.PreviousState,
		"new_state":        result.NewState,
		"duration_secs":    result.Duration,
		"requests_blocked": result.ReuquestBlocked,
		"timestamp":        result.Timestamp.Format(time.RFC3339),
	}, nil
}

// Rollback restores circuit breaker to previous state
func (a *CircuitBreakerAction) Rollback(ctx context.Context, store Store, historyID uuid.UUID) error {
	// In production: Restore previous state
	return nil
}
