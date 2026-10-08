// Package tenantnetwork applies a tenant's IP allowlist during provisioning.
//
// The allowlist is fail-open by design: a tenant with no entries is allowed
// from anywhere, so an empty input skips the step and makes no store call.
package tenantnetwork

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
)

const (
	errTypeAllowlistInput  = "TenantAllowlistInvalidInput"
	errTypeAllowlistConfig = "TenantAllowlistNotConfigured"

	maxEntries  = 256
	maxEntryLen = 64
)

// AllowlistStore replaces one tenant's allowlist. Implementations must write
// an assignment for every entry, scoped to the tenant. Request-time enforcement
// (middleware/session_auth.go) reads only assigned entries, so an entry written
// without an assignment is silently ignored: the tenant gets no protection from
// it. Always write the assignment.
//
// The replacement must be atomic: a failure leaves the previous list in place.
type AllowlistStore interface {
	ReplaceTenantAllowlist(ctx context.Context, tenantID string, cidrs []string) error
}

// Input is the wizard's allowlist for one tenant.
type Input struct {
	TenantID string
	Entries  []string // IPs or CIDRs. Empty means fail-open: skip the step.
}

// Result reports what was applied. Applied is false when the step was skipped.
type Result struct {
	Applied    bool
	Normalized []string // canonical CIDRs, sorted and de-duplicated
}

// Activities holds the dependencies of the allowlist step.
type Activities struct {
	Store AllowlistStore
}

// ApplyIpAllowlist validates and normalizes the entries, then replaces the
// tenant's allowlist in one store call. Re-running with the same input gives
// the same result, so it is safe to retry.
func (a *Activities) ApplyIpAllowlist(ctx context.Context, in Input) (Result, error) {
	if len(in.Entries) == 0 {
		return Result{}, nil
	}
	if a.Store == nil {
		return Result{}, nonRetryable(errTypeAllowlistConfig, errors.New("allowlist store is not configured"))
	}
	if _, err := uuid.Parse(in.TenantID); err != nil {
		return Result{}, nonRetryable(errTypeAllowlistInput, errors.New("tenant ID is not a valid UUID"))
	}
	normalized, err := Normalize(in.Entries)
	if err != nil {
		return Result{}, nonRetryable(errTypeAllowlistInput, err)
	}
	if err := a.Store.ReplaceTenantAllowlist(ctx, in.TenantID, normalized); err != nil {
		// The store error can echo entries or tenant detail, so the cause is not included.
		return Result{}, errors.New("apply IP allowlist: the store rejected the write")
	}
	return Result{Applied: true, Normalized: normalized}, nil
}

// Normalize turns IPs and CIDRs into canonical CIDRs, sorted and de-duplicated.
// A bare IP becomes a /32 or /128. Wildcards and anything else that is not an
// address or network are rejected.
func Normalize(entries []string) ([]string, error) {
	if len(entries) > maxEntries {
		return nil, fmt.Errorf("too many allowlist entries: at most %d", maxEntries)
	}
	seen := make(map[string]bool, len(entries))
	out := make([]string, 0, len(entries))
	for _, raw := range entries {
		canonical, err := normalizeOne(raw)
		if err != nil {
			return nil, err
		}
		if !seen[canonical] {
			seen[canonical] = true
			out = append(out, canonical)
		}
	}
	sort.Strings(out)
	return out, nil
}

func normalizeOne(raw string) (string, error) {
	// Report only the rule, never the entry: entries are client input.
	if raw == "" || len(raw) > maxEntryLen {
		return "", errors.New("allowlist entry must be 1 to 64 characters")
	}
	if strings.Contains(raw, "/") {
		_, network, err := net.ParseCIDR(raw)
		if err != nil {
			return "", errors.New("allowlist entry is not a valid CIDR")
		}
		return network.String(), nil
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return "", errors.New("allowlist entry is not a valid IP or CIDR")
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String() + "/32", nil
	}
	return ip.String() + "/128", nil
}

func nonRetryable(kind string, err error) error {
	return temporal.NewNonRetryableApplicationError(err.Error(), kind, err)
}
