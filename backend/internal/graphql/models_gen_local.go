// Code copied from previous generated models and adapted to package graphql.
package graphql

import (
	"time"

	"github.com/google/uuid"
)

type IPWhitelistEntry struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    *uuid.UUID `json:"tenantId,omitempty"`
	IPAddress   string     `json:"ipAddress"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type IPWhitelistEntryInput struct {
	TenantID    *uuid.UUID `json:"tenantId,omitempty"`
	IPAddress   string     `json:"ipAddress"`
	Description *string    `json:"description,omitempty"`
}

// Minimal types to satisfy generated references where needed.
type Mutation struct{}
type Query struct{}
