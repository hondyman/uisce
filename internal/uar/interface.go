package uar

import "context"

// UARStore abstracts the audit‑record persistence layer.
type UARStore interface {
    // Write persists a record and returns the new head hash (or an opaque ID).
    Write(ctx context.Context, tenantID, clientID, eventType string, payload map[string]any) (string, error)
    // Verify can be used to recompute the chain for a tenant.
    Verify(ctx context.Context, tenantID string) error
}
