package bundles

import "time"

// Entitlement represents a single claim/permission in the system.
type Entitlement struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// UsageEvent is a normalized usage log entry linking a user and an entitlement.
type UsageEvent struct {
	Timestamp     time.Time `json:"ts"`
	UserID        string    `json:"user_id"`
	TenantID      string    `json:"tenant_id"`
	EntitlementID string    `json:"entitlement_id"`
	Count         int       `json:"count"`
}

// CandidateBundle is the output of the miner: a proposed bundle.
type CandidateBundle struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Claims       []string          `json:"claims" db:"claims"`
	Scope        string            `json:"scope"`
	Score        float64           `json:"score"`
	Risk         float64           `json:"risk"`
	Explanations map[string]string `json:"explanations"`
	Status       string            `json:"status"` // candidate|approved|rejected
	CreatedAt    time.Time         `json:"created_at"`
}
