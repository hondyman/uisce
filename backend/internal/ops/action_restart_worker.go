package ops

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RestartWorkerAction restarts a worker service to recover from stuck/unhealthy state
type RestartWorkerAction struct{}

// RestartWorkerParams defines the parameters for a restart worker action
type RestartWorkerParams struct {
	WorkerID string `json:"worker_id"`
	Force    bool   `json:"force,omitempty"`
}

// RestartWorkerResult is the result of a restart worker action
type RestartWorkerResult struct {
	WorkerID      string    `json:"worker_id"`
	PreviousState string    `json:"previous_state"`
	NewState      string    `json:"new_state"`
	RestartedAt   time.Time `json:"restarted_at"`
	JobsRequeued  int       `json:"jobs_requeued"`
}

// NewRestartWorkerAction creates a new restart worker action
func NewRestartWorkerAction() *RestartWorkerAction {
	return &RestartWorkerAction{}
}

// ID returns the action identifier
func (a *RestartWorkerAction) ID() string {
	return "restart_worker"
}

// Name returns the human-readable action name
func (a *RestartWorkerAction) Name() string {
	return "Restart Worker"
}

// Validate checks if preconditions are met
func (a *RestartWorkerAction) Validate(ctx context.Context, params json.RawMessage) error {
	// In production, would check:
	// - Worker exists
	// - Worker is unhealthy or stuck
	// - User has permission
	// For now, basic validation
	return nil
}

// Execute performs the restart worker action
func (a *RestartWorkerAction) Execute(ctx context.Context, params json.RawMessage) (map[string]interface{}, error) {
	// In production, this would:
	// 1. Connect to worker orchestration system (Kubernetes, Nomad, etc.)
	// 2. Send graceful shutdown signal
	// 3. Wait for graceful shutdown or force kill after timeout
	// 4. Monitor startup
	// 5. Verify health
	//
	// For now, simulate with 100ms delay
	time.Sleep(100 * time.Millisecond)

	result := RestartWorkerResult{
		WorkerID:      "worker-default-1",
		PreviousState: "unhealthy",
		NewState:      "starting",
		RestartedAt:   time.Now().UTC(),
		JobsRequeued:  42,
	}

	return map[string]interface{}{
		"worker_id":      result.WorkerID,
		"previous_state": result.PreviousState,
		"new_state":      result.NewState,
		"restarted_at":   result.RestartedAt,
		"jobs_requeued":  result.JobsRequeued,
	}, nil
}

// Rollback attempts to undo the restart (no-op for restart, since it's idempotent)
func (a *RestartWorkerAction) Rollback(ctx context.Context, store Store, historyID uuid.UUID) error {
	// Restart is idempotent; no rollback needed
	return nil
}
