package models

import (
	"net"
	"time"

	"github.com/google/uuid"
)

// APIKeyUsage represents a single API key usage record for audit trail
type APIKeyUsage struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	APIKeyID  uuid.UUID  `db:"api_key_id" json:"api_key_id"`
	UserID    *uuid.UUID `db:"user_id" json:"user_id"`
	TenantID  *uuid.UUID `db:"tenant_id" json:"tenant_id"`
	Path      string     `db:"path" json:"path"`
	Method    string     `db:"method" json:"method"`
	Region    *string    `db:"region" json:"region"`
	IPAddress *net.IP    `db:"ip_address" json:"ip_address"`
	UserAgent *string    `db:"user_agent" json:"user_agent"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

// APIKeyUsageCreateRequest represents a request to log API key usage
type APIKeyUsageCreateRequest struct {
	APIKeyID  uuid.UUID
	UserID    *uuid.UUID
	TenantID  *uuid.UUID
	Path      string
	Method    string
	Region    *string
	IPAddress *net.IP
	UserAgent *string
}

// DailyUsageStats represents daily usage statistics
type DailyUsageStats struct {
	Day   string `json:"day" db:"day"`
	Count int    `json:"count" db:"count"`
}

// EndpointUsageStats represents per-endpoint usage statistics
type EndpointUsageStats struct {
	Path  string `json:"path" db:"path"`
	Count int    `json:"count" db:"count"`
}
