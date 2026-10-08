// Package tenantdbcreds creates the database role for a tenant instance and
// stores its credentials at tenants/<code>/<env>/db_credentials.
//
// This is the only owner of tenant database role credentials. The saga must not
// create roles, and the existing datasource credential writer (dscreds) is a
// different layout and failure domain, so it is not reused here.
//
// Boundary rule: the password is generated inside this activity and never
// returned. The admin credential is read from the store, not passed in.
package tenantdbcreds

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"regexp"

	"github.com/hondyman/uisce/backend/internal/provisioning"
	"github.com/hondyman/uisce/backend/internal/tenantsecrets"
	"go.temporal.io/sdk/temporal"
)

const (
	errTypeDBCredsInput  = "TenantDBCredsInvalidInput"
	errTypeDBCredsConfig = "TenantDBCredsNotConfigured"
	errTypeDBCredsState  = "TenantDBCredsRoleWithoutCredentials"

	// AdminUsernameKey and AdminPasswordKey are the keys under the platform admin path.
	AdminUsernameKey = "username"
	AdminPasswordKey = "password"

	passwordBytes = 32
)

var (
	identPattern  = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
	regionPattern = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-[0-9]+$`)
	// safePassword is the only alphabet a generated password uses. It needs no
	// quoting, so it can be embedded in CREATE ROLE safely.
	safePassword    = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	hostnamePattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)
)

// SecretStore is the subset of secrets.Provider this step needs.
type SecretStore interface {
	GetMap(ctx context.Context, key string) (map[string]string, error)
	PutMap(ctx context.Context, key string, values map[string]string) error
}

// RoleAdmin performs the role operations. Its implementation holds an admin
// connection, which must never be exposed outside this package.
type RoleAdmin interface {
	RoleExists(ctx context.Context, role string) (bool, error)
	CreateRole(ctx context.Context, role, password string) error
	GrantConnect(ctx context.Context, role, database string) error
	DropRole(ctx context.Context, role string) error
	Close() error
}

// RoleAdminConnector opens a RoleAdmin for one region's Postgres endpoint.
type RoleAdminConnector interface {
	Connect(ctx context.Context, host string, port int, adminUser, adminPassword string) (RoleAdmin, error)
}

// Input references only. The admin credential is read from the store.
type Input struct {
	TenantCode   string
	Environment  string
	Region       string // region code; selects the admin credential path
	DatabaseName string
	DatabaseHost string
	DatabasePort int
}

// Result holds no secrets. Skipped means credentials already existed, so
// nothing was created or rotated.
type Result struct {
	Role    string
	Path    string
	Created bool
	Skipped bool
}

// Activities holds the dependencies of the credential step.
type Activities struct {
	Secrets   SecretStore
	Connector RoleAdminConnector
	// Random is the source of password bytes. Tests may replace it.
	Random func([]byte) (int, error)
}

// AdminPath is where a region's Postgres admin credential lives.
func AdminPath(region string) (string, error) {
	if !regionPattern.MatchString(region) || len(region) > 32 {
		return "", fmt.Errorf("region %q is not a valid region code", region)
	}
	return "platform/postgres/" + region + "/admin", nil
}

// RoleName derives the tenant role name from validated parts, so it is an
// identifier by construction.
func RoleName(tenantCode, environment string) (string, error) {
	if !provisioning.ValidTenantCode(tenantCode) {
		return "", fmt.Errorf("tenant code %q is not valid", tenantCode)
	}
	if environment != "dev" && environment != "uat" && environment != "prod" {
		return "", fmt.Errorf("environment %q is not one of dev, uat, prod", environment)
	}
	name := "tenant_" + tenantCode + "_" + environment
	if !identPattern.MatchString(name) {
		return "", errors.New("derived role name is not a valid identifier")
	}
	return name, nil
}

// ProvisionDbCredentials creates the role and stores its credentials. It is
// idempotent in the safe direction: if the credentials already exist it does
// nothing, and it never rotates a password mid-provisioning.
func (a *Activities) ProvisionDbCredentials(ctx context.Context, in Input) (Result, error) {
	if a.Secrets == nil || a.Connector == nil {
		return Result{}, nonRetryable(errTypeDBCredsConfig, errors.New("database credential step is not configured"))
	}
	role, err := RoleName(in.TenantCode, in.Environment)
	if err != nil {
		return Result{}, nonRetryable(errTypeDBCredsInput, err)
	}
	credPath, err := tenantsecrets.CredentialPath(in.TenantCode, in.Environment)
	if err != nil {
		return Result{}, nonRetryable(errTypeDBCredsInput, err)
	}
	adminPath, err := AdminPath(in.Region)
	if err != nil {
		return Result{}, nonRetryable(errTypeDBCredsInput, err)
	}
	if !identPattern.MatchString(in.DatabaseName) {
		return Result{}, nonRetryable(errTypeDBCredsInput, errors.New("database name is not a valid identifier"))
	}
	if !hostnamePattern.MatchString(in.DatabaseHost) && net.ParseIP(in.DatabaseHost) == nil {
		return Result{}, nonRetryable(errTypeDBCredsInput, errors.New("database host is not a valid hostname or IP"))
	}
	if in.DatabasePort < 1 || in.DatabasePort > 65535 {
		return Result{}, nonRetryable(errTypeDBCredsInput, fmt.Errorf("database port %d is out of range", in.DatabasePort))
	}

	// Never rotate: if credentials are already stored, this run is a no-op.
	existing, err := a.Secrets.GetMap(ctx, credPath)
	if err != nil {
		return Result{}, errors.New("read instance credentials: the secret store could not be read")
	}
	if existing[tenantsecrets.CredentialKeyUsername] != "" && existing[tenantsecrets.CredentialKeyPassword] != "" {
		return Result{Role: role, Path: credPath, Skipped: true}, nil
	}

	admin, err := a.adminCredentials(ctx, adminPath)
	if err != nil {
		return Result{}, err
	}
	conn, err := a.Connector.Connect(ctx, in.DatabaseHost, in.DatabasePort, admin[AdminUsernameKey], admin[AdminPasswordKey])
	if err != nil {
		return Result{}, errors.New("connect as the platform admin: the connection was refused")
	}
	defer conn.Close()

	exists, err := conn.RoleExists(ctx, role)
	if err != nil {
		return Result{}, errors.New("check role existence: the database rejected the query")
	}
	if exists {
		// A role with no stored credentials means an earlier run created it and
		// then failed to store the password. The password is unrecoverable, so
		// this needs a human, not a retry.
		return Result{}, nonRetryable(errTypeDBCredsState, errors.New("role exists without stored credentials; reconcile manually"))
	}

	password, err := a.generatePassword()
	if err != nil {
		return Result{}, errors.New("generate password: the random source failed")
	}
	if err := conn.CreateRole(ctx, role, password); err != nil {
		return Result{}, errors.New("create role: the database rejected the statement")
	}
	if err := conn.GrantConnect(ctx, role, in.DatabaseName); err != nil {
		// Undo the role so a retry starts clean. Failure here is ignored: the
		// retry will then report the state as a conflict and a human reconciles.
		_ = conn.DropRole(ctx, role)
		return Result{}, errors.New("grant connect: the database rejected the statement")
	}
	if err := a.Secrets.PutMap(ctx, credPath, map[string]string{
		tenantsecrets.CredentialKeyUsername: role,
		tenantsecrets.CredentialKeyPassword: password,
	}); err != nil {
		// No credentials were stored, so undo the role. Otherwise a retry would
		// hit the unrecoverable state above.
		_ = conn.DropRole(ctx, role)
		return Result{}, errors.New("store instance credentials: the secret store rejected the write")
	}
	return Result{Role: role, Path: credPath, Created: true}, nil
}

func (a *Activities) adminCredentials(ctx context.Context, adminPath string) (map[string]string, error) {
	admin, err := a.Secrets.GetMap(ctx, adminPath)
	if err != nil {
		return nil, errors.New("read platform admin credential: the secret store could not be read")
	}
	if admin[AdminUsernameKey] == "" || admin[AdminPasswordKey] == "" {
		return nil, nonRetryable(errTypeDBCredsConfig, errors.New("platform admin credential is not configured for this region"))
	}
	return admin, nil
}

func (a *Activities) generatePassword() (string, error) {
	random := a.Random
	if random == nil {
		random = rand.Read
	}
	buf := make([]byte, passwordBytes)
	if n, err := random(buf); err != nil || n != len(buf) {
		return "", errors.New("random source failed")
	}
	pw := base64.RawURLEncoding.EncodeToString(buf)
	if !safePassword.MatchString(pw) {
		return "", errors.New("generated password has unexpected characters")
	}
	return pw, nil
}

func nonRetryable(kind string, err error) error {
	return temporal.NewNonRetryableApplicationError(err.Error(), kind, err)
}
