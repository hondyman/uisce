// Package tenantidentity provisions a tenant's identity realm in Keycloak: one
// realm per tenant, named by tenant_code, with an optional LDAP federation.
//
// Boundary rule for every activity in this repo: inputs are references (tenant
// code, environment, region ID, secret paths), never secret values. The
// activity reads secrets from the store itself. Outputs are paths, statuses and
// non-secret identifiers, never values. Temporal stores both inputs and outputs
// in permanent event history.
package tenantidentity

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/provisioning"
	"go.temporal.io/sdk/temporal"
)

// ErrRealmExists means the realm is already there. Whether this run may treat
// it as its own is decided by RealmOwner, not by this error.
var ErrRealmExists = errors.New("keycloak realm already exists")

const (
	errTypeIdentityInput    = "TenantIdentityInvalidInput"
	errTypeIdentityConfig   = "TenantIdentityNotConfigured"
	errTypeIdentityConflict = "TenantIdentityRealmExists"

	// LDAPBindPasswordKey is the key under the identity path where the wizard
	// handler stores the LDAP bind password before the workflow starts.
	LDAPBindPasswordKey = "ldap_bind_password"
	// ClientSecretKey is where the activity stores the platform client secret.
	ClientSecretKey = "client_secret"

	maxBindPasswordLen = 256
)

// KeycloakAdmin is the subset of Keycloak's admin API the identity step needs.
// Implementations must use a realm-management service account, not the
// master-realm admin. The adapter is not built yet; see the plan.
//
// Every method must be idempotent. Temporal retries an activity whose earlier
// attempt may have partly succeeded, so a repeat call must converge, not fail.
type KeycloakAdmin interface {
	// CreateRealm creates the realm and records tenantID on it as an attribute.
	// It returns ErrRealmExists if the realm is already there.
	CreateRealm(ctx context.Context, realm, tenantID string) error
	// RealmOwner returns the tenant ID recorded on the realm, or "" if the realm
	// has no owner record.
	RealmOwner(ctx context.Context, realm string) (tenantID string, err error)
	// DeleteRealm removes the realm. Used only for realms this run owns.
	DeleteRealm(ctx context.Context, realm string) error
	// CreatePlatformClient creates the platform's OIDC client in the realm, or
	// returns the existing client's secret if it is already there.
	CreatePlatformClient(ctx context.Context, realm string) (clientSecret string, err error)
	// AddLDAPFederation attaches an LDAP user-federation provider to the realm.
	// It is a no-op if a federation with the same name is already attached.
	AddLDAPFederation(ctx context.Context, realm string, ldap LDAPConfig, bindPassword string) error
}

// SecretStore is the subset of secrets.Provider the identity step needs.
// secrets.Provider satisfies it.
type SecretStore interface {
	PutMap(ctx context.Context, key string, values map[string]string) error
	GetMap(ctx context.Context, key string) (map[string]string, error)
}

// IssuerStore records the realm's issuer URL on the tenant. SetIssuer must be
// an upsert: a retry writes the same value again.
type IssuerStore interface {
	SetIssuer(ctx context.Context, tenantCode, issuer string) error
}

// LDAPConfig is the LDAP federation's non-secret settings. The bind password is
// read from the store by the activity and never appears here.
type LDAPConfig struct {
	Host   string
	Port   int
	BaseDN string
	BindDN string
}

// Input is what the wizard supplies for the identity step.
type Input struct {
	TenantID   string // the tenant's UUID; recorded on the realm as its owner
	TenantCode string
	LDAP       *LDAPConfig // nil means Keycloak-local users only
}

// Result describes what the step did. It holds no secret values.
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

// IdentityPath is the store path for one tenant's identity secrets. It is
// derived from the validated tenant code, never supplied by a caller.
func IdentityPath(tenantCode string) (string, error) {
	if !provisioning.ValidTenantCode(tenantCode) {
		return "", fmt.Errorf("tenant code %q is not valid", tenantCode)
	}
	return "tenants/" + tenantCode + "/identity", nil
}

// ConfigureTenantIdentity creates the tenant's realm, its platform client,
// the optional LDAP federation, and the client secret, then records the issuer.
//
// Validation and the LDAP bind password read happen before anything is created,
// so a bad input or a missing secret changes nothing in Keycloak.
func (a *Activities) ConfigureTenantIdentity(ctx context.Context, in Input) (Result, error) {
	if a.Keycloak == nil || a.Secrets == nil || a.Issuers == nil {
		return Result{}, nonRetryable(errTypeIdentityConfig, errors.New("tenant identity is not configured"))
	}
	if _, err := uuid.Parse(in.TenantID); err != nil {
		return Result{}, nonRetryable(errTypeIdentityInput, errors.New("tenant ID is not a valid UUID"))
	}
	identityPath, err := IdentityPath(in.TenantCode)
	if err != nil {
		return Result{}, nonRetryable(errTypeIdentityInput, err)
	}
	issuer, err := issuerURL(a.PublicBaseURL, in.TenantCode)
	if err != nil {
		return Result{}, nonRetryable(errTypeIdentityConfig, err)
	}

	var bindPassword string
	if in.LDAP != nil {
		if err := validateLDAP(*in.LDAP); err != nil {
			return Result{}, nonRetryable(errTypeIdentityInput, err)
		}
		bindPassword, err = a.readBindPassword(ctx, identityPath)
		if err != nil {
			return Result{}, err
		}
	}

	res := Result{Realm: in.TenantCode, Issuer: issuer, SecretPath: identityPath}

	if err := a.Keycloak.CreateRealm(ctx, in.TenantCode, in.TenantID); err != nil {
		if !errors.Is(err, ErrRealmExists) {
			return res, fmt.Errorf("create realm: %w", err)
		}
		// The realm exists. It is ours only if this tenant recorded itself as the
		// owner, which means an earlier attempt of this same run created it. An
		// unowned or foreign realm is a conflict, and this run must not touch it.
		owner, ownerErr := a.Keycloak.RealmOwner(ctx, in.TenantCode)
		if ownerErr != nil {
			return res, errors.New("check realm ownership: the identity provider could not be read")
		}
		if owner != in.TenantID {
			return res, nonRetryable(errTypeIdentityConflict, fmt.Errorf("realm for tenant %s already exists and is not owned by this tenant", in.TenantCode))
		}
	}
	res.RealmCreated = true

	clientSecret, err := a.Keycloak.CreatePlatformClient(ctx, in.TenantCode)
	if err != nil {
		return res, fmt.Errorf("create platform client: %w", err)
	}
	res.ClientCreated = true

	if in.LDAP != nil {
		if err := a.Keycloak.AddLDAPFederation(ctx, in.TenantCode, *in.LDAP, bindPassword); err != nil {
			return res, fmt.Errorf("add LDAP federation: %w", err)
		}
	}

	// The client secret is written to the store here and returned to no caller.
	if err := a.Secrets.PutMap(ctx, identityPath, map[string]string{ClientSecretKey: clientSecret}); err != nil {
		return res, errors.New("store identity secrets: the secret store rejected the write")
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

func (a *Activities) readBindPassword(ctx context.Context, identityPath string) (string, error) {
	values, err := a.Secrets.GetMap(ctx, identityPath)
	if err != nil {
		return "", errors.New("read LDAP bind password: the secret store could not be read")
	}
	pw := values[LDAPBindPasswordKey]
	if pw == "" || len(pw) > maxBindPasswordLen {
		return "", nonRetryable(errTypeIdentityInput, fmt.Errorf("LDAP bind password must be present and at most %d characters", maxBindPasswordLen))
	}
	return pw, nil
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
	return nil
}

func nonRetryable(kind string, err error) error {
	return temporal.NewNonRetryableApplicationError(err.Error(), kind, err)
}
