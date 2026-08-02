package middleware

import "time"

type APIKey struct {
	ID          string     `json:"id"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	TenantID    string     `json:"tenant_id"`
	Permissions []string   `json:"permissions"`
	RateLimit   int        `json:"rate_limit"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active"`
}
