// Package infra holds the adapters the tenant lakehouse provisioning workflow uses to
// reach infrastructure: KES for the per-tenant key, and the MinIO admin API plus the
// secrets store for the per-tenant, bucket-scoped storage credential (ADR-032).
//
// Each adapter sits behind a narrow interface so every branch can be tested with a
// fake. They have not been exercised against a live KES or MinIO; the tests prove the
// logic, not the wire format.
package infra

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrNotConfigured means the infrastructure this step needs has no configuration. It is
// returned, never papered over with a default: a missing key must not fall back to a
// shared one, and a missing admin credential must not fall back to "minioadmin".
var ErrNotConfigured = errors.New("lakehouse infrastructure is not configured")

// Keys manages a tenant's KMS key.
type Keys interface {
	EnsureKey(ctx context.Context, name string) error
}

// ErrCredentialLost means a tenant's storage credential was issued before, but the secrets
// store cannot produce it. That is an outage or a loss, and issuing a new credential would
// silently strand the tenant's warehouse on the old, now-stale keys, so it is refused. A
// person must restore the secret or rotate it deliberately.
var ErrCredentialLost = errors.New("the storage credential was issued but cannot be read from the secrets store; refusing to issue a new one")

// Credentials issues and reads a tenant's bucket-scoped storage credential.
type Credentials interface {
	// EnsureBucketCredential makes sure the credential exists. mayMint says whether a
	// credential has never been issued for this tenant (the registry records that); when
	// false and none can be read, it must return ErrCredentialLost, never mint a new one.
	EnsureBucketCredential(ctx context.Context, tenantID uuid.UUID, bucket string, mayMint bool) error
	Read(ctx context.Context, tenantID uuid.UUID) (accessKeyID, secretAccessKey string, err error)
}

func notConfigured(what string, vars ...string) error {
	return fmt.Errorf("%w: %s needs %v", ErrNotConfigured, what, vars)
}
