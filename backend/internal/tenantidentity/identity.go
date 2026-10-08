// Package tenantidentity provisions a tenant's identity realm in Keycloak: one
// realm per tenant, named by tenant_code, with an optional LDAP federation.
//
// Secrets never appear in error messages or return values. They go to the
// SecretStore and nowhere else.
package tenantidentity

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/hondyman/uisce/backend/internal/provisioning"
	"go.temporal.io/sdk/temporal"
)

// ErrRealmExists means the tenant's realm was already there before this run.
// The run must not delete it on rollback.
var ErrRealmExists = errors.New("keycloak realm already exists")

const (
	errTypeIdentityInput    = "TenantIdentityInvalidInput"
	errTypeIdentityConfig   = "TenantIdentityNotConfigured"
	errTypeIdentityConflict = "TenantIdentityRealmExists"
)

// KeycloakAdmin is the subset of Keycloak's admin API the identity step needs.
// Implementations must use the dedicated realm-management service account, not
// the master-realm admin.
type KeycloakAdmin interface {
	// CreateRealm creates the realm, or returns ErrRealmExists if it is there.
	CreateRealm(ctx context.Context, realm string) error
	// DeleteRealm removes the realm. Used only for realms this run created.
	DeleteRealm(ctx context.Context, realm string) error
	// CreatePlatformClient creates the platform's OIDC client in the realm and
	// returns its client secret.
	CreatePlatformClient(ctx context.Context, realm string) (clientSecret string, err error)
	// AddLDAPFederation attaches an LDAP user-federation provider to the realm.
	AddLDAPFederation(ctx context.Context, realm string, ldap LDAPConfig) error
}

// SecretStore is the subset of secrets.Provider the identity step needs.
// secrets.Provider satisfies it.
type SecretStore interface {
	PutMap(ctx context.Context, key string, values map[string]string) error
}

// IssuerStore records the realm's issuer URL on the tenant.
type IssuerStore interface {
	SetIssuer(ctx context.Context, tenantCode, issuer string) error
}

// LDAPConfig is the optional LDAP federation. The bind password is a secret.
type LDAPConfig struct {
	Host         string
	Port         int
	BaseDN       string
	BindDN       string
	BindPassword string
}

// Input is what the wizard supplies for the identity step.
type Input struct {
	TenantCode string
	LDAP       *LDAPConfig // nil means Keycloak-local users only
}

// Result describes what the step did. RealmCreated tells compensation whether
// the realm is this run's to delete. It is set even when the step returns an
// error, so a partial run can still be cleaned up.
type Result struct {
	Realm          string
	Issuer         string
	SecretPath     string
	RealmCreated   bool
	ClientCreated  bool
	SecretsWritten bool
}

// Activities holds the dependencies of the identity step.
type Activities struct {
	Keycloak      KeycloakAdmin
	Secrets       SecretStore
	Issuers       IssuerStore
	PublicBaseURL string // e.g. https://keycloak.example.internal
}

var (
	hostnamePattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)
	// dnPattern allows the characters a DN uses, and rejects control characters
	// and anything that could change the LDAP filter or the config file.
	dnPattern = regexp.MustCompile(`^[A-Za-z0-9=,. _-]{1,512}$`)
)

// ConfigureTenantIdentity creates the tenant's realm, its platform client,
// the optional LDAP federation, and the secrets, then records the issuer.
//
// Validation happens before any Keycloak call. The returned Result reports
// what was created so the caller can compensate after a failure.
func (a *Activities) ConfigureTenantIdentity(ctx context.Context, in Input) (Result, error) {
	if a.Keycloak == nil || a.Secrets == nil || a.Issuers == nil {
		return Result{}, nonRetryable(errTypeIdentityConfig, errors.New("tenant identity is not configured"))
	}
	if !provisioning.ValidTenantCode(in.TenantCode) {
		return Result{}, nonRetryable(errTypeIdentityInput, fmt.Errorf("tenant code %q is not valid", in.TenantCode))
	}
	issuer, err := issuerURL(a.PublicBaseURL, in.TenantCode)
	if err != nil {
		return Result{}, nonRetryable(errTypeIdentityConfig, err)
	}
	if in.LDAP != nil {
		if err := validateLDAP(*in.LDAP); err != nil {
			return Result{}, nonRetryable(errTypeIdentityInput, err)
		}
	}

	res := Result{Realm: in.TenantCode, Issuer: issuer, SecretPath: "tenants/" + in.TenantCode + "/identity"}

	if err := a.Keycloak.CreateRealm(ctx, in.TenantCode); err != nil {
		if errors.Is(err, ErrRealmExists) {
			return res, nonRetryable(errTypeIdentityConflict, fmt.Errorf("realm for tenant %s already exists", in.TenantCode))
		}
		return res, fmt.Errorf("create realm: %w", err)
	}
	res.RealmCreated = true

	clientSecret, err := a.Keycloak.CreatePlatformClient(ctx, in.TenantCode)
	if err != nil {
		return res, fmt.Errorf("create platform client: %w", err)
	}
	res.ClientCreated = true

	values := map[string]string{"client_secret": clientSecret}
	if in.LDAP != nil {
		if err := a.Keycloak.AddLDAPFederation(ctx, in.TenantCode, *in.LDAP); err != nil {
			return res, fmt.Errorf("add LDAP federation: %w", err)
		}
		values["ldap_bind_password"] = in.LDAP.BindPassword
	}

	if err := a.Secrets.PutMap(ctx, res.SecretPath, values); err != nil {
		return res, fmt.Errorf("store identity secrets: %w", err)
	}
	res.SecretsWritten = true

	if err := a.Issuers.SetIssuer(ctx, in.TenantCode, issuer); err != nil {
		return res, fmt.Errorf("record issuer: %w", err)
	}
	return res, nil
}

// RollbackConfigureTenantIdentity deletes the realm. The caller passes it only
// when Result.RealmCreated is true for this run.
func (a *Activities) RollbackConfigureTenantIdentity(ctx context.Context, tenantCode string) error {
	if a.Keycloak == nil {
		return nonRetryable(errTypeIdentityConfig, errors.New("tenant identity is not configured"))
	}
	if !provisioning.ValidTenantCode(tenantCode) {
		return nonRetryable(errTypeIdentityInput, fmt.Errorf("tenant code %q is not valid", tenantCode))
	}
	return a.Keycloak.DeleteRealm(ctx, tenantCode)
}

func issuerURL(base, tenantCode string) (string, error) {
	u, err := url.Parse(base)
	if err != nil || base == "" {
		return "", errors.New("public base URL is not a valid URL")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", errors.New("public base URL must use http or https")
	}
	if u.Host == "" {
		return "", errors.New("public base URL has no host")
	}
	return strings.TrimRight(base, "/") + "/realms/" + tenantCode, nil
}

func validateLDAP(c LDAPConfig) error {
	if !hostnamePattern.MatchString(c.Host) && net.ParseIP(c.Host) == nil {
		return errors.New("LDAP host is not a valid hostname or IP")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("LDAP port %d is out of range", c.Port)
	}
	if !dnPattern.MatchString(c.BaseDN) {
		return errors.New("LDAP base DN is not valid")
	}
	if !dnPattern.MatchString(c.BindDN) {
		return errors.New("LDAP bind DN is not valid")
	}
	if c.BindPassword == "" || len(c.BindPassword) > 256 {
		return errors.New("LDAP bind password must be 1 to 256 characters")
	}
	return nil
}

func nonRetryable(kind string, err error) error {
	return temporal.NewNonRetryableApplicationError(err.Error(), kind, err)
}
