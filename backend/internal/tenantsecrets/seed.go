// Package tenantsecrets writes one tenant instance's connection secrets to the
// secret store. The layout is fixed: tenants/<tenant_code>/<env>/.
//
// Values are never returned, logged or included in error text.
package tenantsecrets

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/hondyman/uisce/backend/internal/provisioning"
	"go.temporal.io/sdk/temporal"
)

const (
	errTypeSecretsInput  = "TenantSecretsInvalidInput"
	errTypeSecretsConfig = "TenantSecretsNotConfigured"
)

// Environments a tenant instance can run in. The wizard picks one per run.
var environments = map[string]bool{"dev": true, "uat": true, "prod": true}

// dbNamePattern matches the identifier rule used for tenant databases.
var dbNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// SecretStore is the subset of secrets.Provider this step needs.
type SecretStore interface {
	PutMap(ctx context.Context, key string, values map[string]string) error
}

// Input is one instance's connection details.
type Input struct {
	TenantCode   string
	Environment  string
	DatabaseName string
	// DatabaseURL is a secret. It is written to the store and never echoed.
	DatabaseURL string
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

// SeedInstanceSecrets validates the input and writes the instance's secrets.
// It validates everything before the store is touched.
func (a *Activities) SeedInstanceSecrets(ctx context.Context, in Input) (Result, error) {
	if a.Secrets == nil {
		return Result{}, nonRetryable(errTypeSecretsConfig, errors.New("secret store is not configured"))
	}
	if !provisioning.ValidTenantCode(in.TenantCode) {
		return Result{}, nonRetryable(errTypeSecretsInput, fmt.Errorf("tenant code %q is not valid", in.TenantCode))
	}
	if !environments[in.Environment] {
		return Result{}, nonRetryable(errTypeSecretsInput, fmt.Errorf("environment %q is not one of dev, uat, prod", in.Environment))
	}
	if !dbNamePattern.MatchString(in.DatabaseName) {
		return Result{}, nonRetryable(errTypeSecretsInput, errors.New("database name is not a valid identifier"))
	}
	if err := validateDatabaseURL(in.DatabaseURL); err != nil {
		return Result{}, nonRetryable(errTypeSecretsInput, err)
	}

	path := Path(in.TenantCode, in.Environment)
	values := map[string]string{
		"database_url":  in.DatabaseURL,
		"database_name": in.DatabaseName,
	}
	if err := a.Secrets.PutMap(ctx, path, values); err != nil {
		// The store error can echo request detail, so the cause is not included.
		return Result{}, errors.New("write instance secrets: the secret store rejected the write")
	}
	return Result{Path: path, Keys: []string{"database_url", "database_name"}}, nil
}

// Path returns the secret path for one tenant instance.
func Path(tenantCode, environment string) string {
	return "tenants/" + tenantCode + "/" + environment
}

func validateDatabaseURL(raw string) error {
	if raw == "" {
		return errors.New("database URL is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		// Do not include the parse error: it can quote the URL, password included.
		return errors.New("database URL is not a valid URL")
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return errors.New("database URL must use the postgres scheme")
	}
	if u.Host == "" {
		return errors.New("database URL has no host")
	}
	return nil
}

func nonRetryable(kind string, err error) error {
	return temporal.NewNonRetryableApplicationError(err.Error(), kind, err)
}
