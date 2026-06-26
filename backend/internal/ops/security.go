package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RateLimiter tracks action execution per user with configurable limits
type RateLimiter struct {
	maxActionsPerMinute int
	userActions         map[string][]time.Time
	mu                  sync.RWMutex
}

// NewRateLimiter creates a new rate limiter with the specified max actions per minute
func NewRateLimiter(maxActionsPerMinute int) *RateLimiter {
	limiter := &RateLimiter{
		maxActionsPerMinute: maxActionsPerMinute,
		userActions:         make(map[string][]time.Time),
	}

	// Start cleanup goroutine to remove old entries
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.cleanup()
		}
	}()

	return limiter
}

// IsAllowed checks if a user is allowed to execute an action
func (rl *RateLimiter) IsAllowed(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)

	// Get user's recent actions
	actions := rl.userActions[userID]

	// Filter to last minute
	var recentActions []time.Time
	for _, t := range actions {
		if t.After(oneMinuteAgo) {
			recentActions = append(recentActions, t)
		}
	}

	// Check if under limit
	if len(recentActions) >= rl.maxActionsPerMinute {
		return false
	}

	// Record action
	rl.userActions[userID] = append(recentActions, now)
	return true
}

// GetRemaining returns how many actions the user has remaining in the current minute
func (rl *RateLimiter) GetRemaining(userID string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)

	actions := rl.userActions[userID]
	var recentCount int
	for _, t := range actions {
		if t.After(oneMinuteAgo) {
			recentCount++
		}
	}

	remaining := rl.maxActionsPerMinute - recentCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// cleanup removes old entries (called periodically)
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	fiveMinutesAgo := now.Add(-5 * time.Minute)

	for userID, actions := range rl.userActions {
		var filtered []time.Time
		for _, t := range actions {
			if t.After(fiveMinutesAgo) {
				filtered = append(filtered, t)
			}
		}

		if len(filtered) == 0 {
			delete(rl.userActions, userID)
		} else {
			rl.userActions[userID] = filtered
		}
	}
}

// ParameterValidator validates action parameters based on action type
type ParameterValidator struct{}

// NewParameterValidator creates a new parameter validator
func NewParameterValidator() *ParameterValidator {
	return &ParameterValidator{}
}

// Validate checks if parameters are valid for the given action type
func (pv *ParameterValidator) Validate(actionType string, params json.RawMessage) error {
	var data map[string]interface{}
	if err := json.Unmarshal(params, &data); err != nil {
		return fmt.Errorf("invalid JSON parameters: %w", err)
	}

	switch actionType {
	case "restart_worker":
		return pv.validateRestartWorker(data)
	case "throttle_tenant":
		return pv.validateThrottleTenant(data)
	case "trigger_runbook":
		return pv.validateTriggerRunbook(data)
	case "circuit_breaker_toggle":
		return pv.validateCircuitBreaker(data)
	case "failover_toggle":
		return pv.validateFailover(data)
	default:
		return fmt.Errorf("unknown action type: %s", actionType)
	}
}

func (pv *ParameterValidator) validateRestartWorker(data map[string]interface{}) error {
	if workerID, ok := data["worker_id"]; !ok || workerID == "" {
		return fmt.Errorf("worker_id is required")
	}

	return nil
}

func (pv *ParameterValidator) validateThrottleTenant(data map[string]interface{}) error {
	if tenantID, ok := data["tenant_id"]; !ok || tenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}

	if rateLimit, ok := data["rate_limit_per_sec"].(float64); !ok || rateLimit <= 0 {
		return fmt.Errorf("rate_limit_per_sec must be a positive number")
	}

	if duration, ok := data["duration_secs"].(float64); !ok || duration <= 0 {
		return fmt.Errorf("duration_secs must be a positive number")
	}

	return nil
}

func (pv *ParameterValidator) validateTriggerRunbook(data map[string]interface{}) error {
	if runbookID, ok := data["runbook_id"]; !ok || runbookID == "" {
		return fmt.Errorf("runbook_id is required")
	}

	return nil
}

func (pv *ParameterValidator) validateCircuitBreaker(data map[string]interface{}) error {
	if circuitID, ok := data["circuit_id"]; !ok || circuitID == "" {
		return fmt.Errorf("circuit_id is required")
	}

	if targetState, ok := data["target_state"].(string); !ok {
		return fmt.Errorf("target_state is required")
	} else if targetState != "open" && targetState != "closed" && targetState != "half-open" {
		return fmt.Errorf("target_state must be 'open', 'closed', or 'half-open'")
	}

	return nil
}

func (pv *ParameterValidator) validateFailover(data map[string]interface{}) error {
	sourceRegion, ok := data["source_region"].(string)
	if !ok || sourceRegion == "" {
		return fmt.Errorf("source_region is required")
	}

	targetRegion, ok := data["target_region"].(string)
	if !ok || targetRegion == "" {
		return fmt.Errorf("target_region is required")
	}

	if sourceRegion == targetRegion {
		return fmt.Errorf("source_region and target_region must be different")
	}

	return nil
}

// ResponseSanitizer removes sensitive data from action results
type ResponseSanitizer struct {
	sensitiveFields map[string]bool
}

// NewResponseSanitizer creates a new response sanitizer
func NewResponseSanitizer() *ResponseSanitizer {
	return &ResponseSanitizer{
		sensitiveFields: map[string]bool{
			"password":      true,
			"secret":        true,
			"token":         true,
			"api_key":       true,
			"private_key":   true,
			"access_token":  true,
			"refresh_token": true,
			"credentials":   true,
		},
	}
}

// Sanitize removes sensitive data from the result
func (rs *ResponseSanitizer) Sanitize(result map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	for key, value := range result {
		if rs.sensitiveFields[key] {
			sanitized[key] = "***REDACTED***"
		} else if nested, ok := value.(map[string]interface{}); ok {
			sanitized[key] = rs.Sanitize(nested)
		} else {
			sanitized[key] = value
		}
	}

	return sanitized
}

// AuditLog represents a detailed action audit log entry
type AuditLog struct {
	ID         uuid.UUID              `json:"id"`
	UserID     string                 `json:"user_id"`
	UserRole   string                 `json:"user_role"`
	ActionType string                 `json:"action_type"`
	IncidentID uuid.UUID              `json:"incident_id"`
	Status     string                 `json:"status"`
	Parameters json.RawMessage        `json:"parameters"`
	Result     map[string]interface{} `json:"result,omitempty"`
	ErrorMsg   *string                `json:"error_msg,omitempty"`
	ExecutedAt time.Time              `json:"executed_at"`
	DurationMs int64                  `json:"duration_ms"`
	SourceIP   string                 `json:"source_ip,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// AuditLogger logs action execution with full context
type AuditLogger struct {
	store Store
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(store Store) *AuditLogger {
	return &AuditLogger{store: store}
}

// LogAction logs an action execution for audit purposes
func (al *AuditLogger) LogAction(userID, userRole, actionType, sourceIP string, incidentID uuid.UUID, status string, params json.RawMessage, result map[string]interface{}, errorMsg *string, durationMs int64) (*AuditLog, error) {
	auditLog := &AuditLog{
		ID:         uuid.New(),
		UserID:     userID,
		UserRole:   userRole,
		ActionType: actionType,
		IncidentID: incidentID,
		Status:     status,
		Parameters: params,
		Result:     result,
		ErrorMsg:   errorMsg,
		ExecutedAt: time.Now(),
		DurationMs: durationMs,
		SourceIP:   sourceIP,
		CreatedAt:  time.Now(),
	}

	// Persist audit log to database
	if err := al.store.InsertAuditLog(context.Background(), auditLog); err != nil {
		// Log error but don't fail the action - audit logging should not block action execution
		fmt.Printf("Failed to persist audit log: %v\n", err)
	}

	return auditLog, nil
}
