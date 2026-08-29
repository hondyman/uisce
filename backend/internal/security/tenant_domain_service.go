package security

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// TenantDomainService resolves a verified user email domain to the tenant a
// client user should be scoped to. Backs "domain -> tenant" auto-provisioning
// on first login: the tenant is never asserted by the client (a header, a
// claim the caller controls) — it is looked up server-side against
// security.tenant_domains, which only an admin can write to.
type TenantDomainService struct {
	db *sql.DB
}

// NewTenantDomainService creates a new TenantDomainService.
func NewTenantDomainService(db *sql.DB) *TenantDomainService {
	return &TenantDomainService{db: db}
}

// ErrDomainNotRegistered is returned when the email's domain has no
// registered tenant. Callers MUST reject the login with a clear message
// (e.g. "Your organization isn't set up yet. Contact support.") — there is no
// default tenant to fall back to.
var ErrDomainNotRegistered = fmt.Errorf("email domain is not registered to a tenant")

// ResolveTenantByEmailDomain looks up the tenant registered for a verified
// email's domain (case-insensitive). Returns ErrDomainNotRegistered — not a
// zero-value tenant — when the domain isn't registered, so callers can't
// mistake "not found" for "no tenant needed."
func (s *TenantDomainService) ResolveTenantByEmailDomain(ctx context.Context, email string) (uuid.UUID, error) {
	domain, err := extractEmailDomain(email)
	if err != nil {
		return uuid.Nil, err
	}

	var tenantID uuid.UUID
	err = s.db.QueryRowContext(ctx, `
		SELECT tenant_id FROM security.tenant_domains
		WHERE domain = $1 AND is_active = true
	`, domain).Scan(&tenantID)
	if err == sql.ErrNoRows {
		return uuid.Nil, ErrDomainNotRegistered
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("tenant domain lookup failed: %w", err)
	}
	return tenantID, nil
}

// extractEmailDomain returns the lowercase domain portion of an email
// address, or an error if the address has no single '@'.
func extractEmailDomain(email string) (string, error) {
	email = strings.TrimSpace(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid email address")
	}
	return strings.ToLower(parts[1]), nil
}
