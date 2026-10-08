// Package tenantsecrets writes one tenant instance's connection secrets to the
// secret store. The layout is fixed: tenants/<tenant_code>/<env>/.
//
// Boundary rule: inputs are references (tenant code, environment, database
// name, host, port). The database credentials are read from the store, and the
// connection URL is built here. Values never appear in inputs, outputs or errors.
package tenantsecrets

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"

	"github.com/hondyman/uisce/backend/internal/provisioning"
	"go.temporal.io/sdk/temporal"
)

const (
	errTypeSecretsInput  = "TenantSecretsInvalidInput"
	errTypeSecretsConfig = "TenantSecretsNotConfigured"

	// CredentialKeyUsername and CredentialKeyPassword are the keys under the
	// credential path. The writer of that path is not built yet; see the plan.
	CredentialKeyUsername = "username"
	CredentialKeyPassword = "password"
	// URLKey and DatabaseNameKey are the keys this step writes.
	URLKey          = "database_url"
	DatabaseNameKey = "database_name"

	maxPasswordLen = 256
	credentialLeaf = "db_credentials"
)

// Environments a tenant instance can run in. The wizard picks one per run.
var environments = map[string]bool{"dev": true, "uat": true, "prod": true}

var (
	// dbNamePattern is also the rule for role names: both are identifiers.
	dbNamePattern   = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
	hostnamePattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)
)

// SecretStore is the subset of secrets.Provider this step needs. secrets.Provider
// satisfies it.
type SecretStore interface {
	GetMap(ctx context.Context, key string) (map[string]string, error)
	PutMap(ctx context.Context, key string, values map[string]string) error
}

// Input is one instance's references. It holds no credentials.
type Input struct {
	TenantCode   string
	Environment  string
	DatabaseName string
	DatabaseHost string
	DatabasePort int
}

// Result reports where the secrets were written. It holds no values.
type Result struct {
	Path string
	Keys []string
}

// Activities holds the dependencies of the seeding step.
type Activities struct {
	Secrets SecretStore
}

// SeedInstanceSecrets reads the instance's database credentials from the store,
// builds the connection URL, and writes it with the database name to the
// instance path. Everything is validated before the store is written.
func (a *Activities) SeedInstanceSecrets(ctx context.Context, in Input) (Result, error) {
	if a.Secrets == nil {
		return Result{}, nonRetryable(errTypeSecretsConfig, errors.New("secret store is not configured"))
	}
	path, err := Path(in.TenantCode, in.Environment)
	if err != nil {
		return Result{}, nonRetryable(errTypeSecretsInput, err)
	}
	credPath, err := CredentialPath(in.TenantCode, in.Environment)
	if err != nil {
		return Result{}, nonRetryable(errTypeSecretsInput, err)
	}
	if !dbNamePattern.MatchString(in.DatabaseName) {
		return Result{}, nonRetryable(errTypeSecretsInput, errors.New("database name is not a valid identifier"))
	}
	if !hostnamePattern.MatchString(in.DatabaseHost) && net.ParseIP(in.DatabaseHost) == nil {
		return Result{}, nonRetryable(errTypeSecretsInput, errors.New("database host is not a valid hostname or IP"))
	}
	if in.DatabasePort < 1 || in.DatabasePort > 65535 {
		return Result{}, nonRetryable(errTypeSecretsInput, fmt.Errorf("database port %d is out of range", in.DatabasePort))
	}

	creds, err := a.Secrets.GetMap(ctx, credPath)
	if err != nil {
		// The store error can echo the path or values, so the cause is dropped.
		return Result{}, errors.New("read instance credentials: the secret store could not be read")
	}
	user, pass := creds[CredentialKeyUsername], creds[CredentialKeyPassword]
	if !dbNamePattern.MatchString(user) {
		return Result{}, nonRetryable(errTypeSecretsInput, errors.New("instance credentials are missing a valid username"))
	}
	if pass == "" || len(pass) > maxPasswordLen {
		return Result{}, nonRetryable(errTypeSecretsInput, fmt.Errorf("instance credentials are missing a password of 1 to %d characters", maxPasswordLen))
	}

	dsn := (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, pass),
		Host:     net.JoinHostPort(in.DatabaseHost, strconv.Itoa(in.DatabasePort)),
		Path:     "/" + in.DatabaseName,
		RawQuery: "sslmode=require",
	}).String()

	if err := a.Secrets.PutMap(ctx, path, map[string]string{URLKey: dsn, DatabaseNameKey: in.DatabaseName}); err != nil {
		// The store error can echo the request, so the cause is not included.
		return Result{}, errors.New("write instance secrets: the secret store rejected the write")
	}
	return Result{Path: path, Keys: []string{URLKey, DatabaseNameKey}}, nil
}

// Path returns the secret path for one tenant instance. It validates both
// components itself, so no caller can build a path from unchecked input.
func Path(tenantCode, environment string) (string, error) {
	if !provisioning.ValidTenantCode(tenantCode) {
		return "", fmt.Errorf("tenant code %q is not valid", tenantCode)
	}
	if !environments[environment] {
		return "", fmt.Errorf("environment %q is not one of dev, uat, prod", environment)
	}
	return "tenants/" + tenantCode + "/" + environment, nil
}

// CredentialPath returns where an instance's database credentials are stored.
func CredentialPath(tenantCode, environment string) (string, error) {
	base, err := Path(tenantCode, environment)
	if err != nil {
		return "", err
	}
	return base + "/" + credentialLeaf, nil
}

func nonRetryable(kind string, err error) error {
	return temporal.NewNonRetryableApplicationError(err.Error(), kind, err)
}
