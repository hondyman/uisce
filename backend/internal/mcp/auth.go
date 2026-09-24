package mcp

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
)

const authRequiredMsg = "auth required: JWT missing tenant_id claim (account may lack a tenant assignment)"

func tenantFromAuth(ctx context.Context) (uuid.UUID, error) {
	auth, ok := security.AuthInfoFromContext(ctx)
	if !ok || len(auth.TenantIDs) == 0 {
		return uuid.Nil, fmt.Errorf("%s", authRequiredMsg)
	}
	id, err := uuid.Parse(auth.TenantIDs[0])
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid tenant_id format")
	}
	return id, nil
}
