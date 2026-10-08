// Package tenantplatform starts a tenant instance in v1 scope: it runs three
// health checks, then marks the instance running and records the endpoints the
// provisioning saga used, in one write. Data-platform bring-up (Lakekeeper,
// StarRocks, Debezium) is not part of this step.
//
// Success ordering: all checks pass, then one guarded write marks the instance
// running with its endpoints. A failed check or a failed write leaves the
// instance in "provisioning".
package tenantplatform

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
	"github.com/hondyman/uisce/backend/internal/tenantsecrets"
	"go.temporal.io/sdk/temporal"
)

const (
	errTypePlatformInput  = "TenantPlatformInvalidInput"
	errTypePlatformConfig = "TenantPlatformNotConfigured"
	errTypePlatformCheck  = "TenantPlatformCheckFailed"
)

// Check names. Failure reports use these names, never the check's inputs or outputs.
const (
	CheckDatabase = "database"
	CheckIdentity = "identity"
	CheckSecrets  = "secrets"
)

var (
	dbNamePattern   = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
	hostnamePattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)
	environments    = map[string]bool{"dev": true, "uat": true, "prod": true}
)

// Endpoints are the addresses the provisioning saga used for this instance.
// StartPlatform checks these exact addresses. It never re-resolves the region,
// so it cannot ping a host other than the one that was provisioned.
type Endpoints struct {
	PostgresHost string
	PostgresPort int
	DatabaseName string
	Issuer       string
}

// Input identifies the instance being started.
type Input struct {
	InstanceID  string
	TenantCode  string
	Environment string
	Endpoints   Endpoints
}

// DatabaseProbe connects to the instance database and runs SELECT 1.
//
// It is given the instance path, not connection details. The implementation
// reads database_url at that path, which SeedInstanceSecrets wrote from the
// same endpoint the saga provisioned. The credential is therefore the seeded
// tenant credential, and the URL never leaves the probe.
type DatabaseProbe interface {
	Ping(ctx context.Context, instancePath string) error
}

// IdentityProbe checks the realm's discovery document. It must confirm the
// document's issuer equals the recorded issuer, which catches a wrong realm.
type IdentityProbe interface {
	CheckIssuer(ctx context.Context, issuer string) error
}

// SecretsProbe asserts that a secret path exists. It must never read values.
type SecretsProbe interface {
	Exists(ctx context.Context, path string) (bool, error)
}

// InstanceStore marks an instance running and records its endpoints in one
// write. The write is a guarded transition: it applies only while the instance
// is still provisioning. A retry after success is not an error; see the store.
type InstanceStore interface {
	MarkRunning(ctx context.Context, instanceID string, ep Endpoints) error
}

// Activities holds the dependencies of StartPlatform.
type Activities struct {
	DB       DatabaseProbe
	Identity IdentityProbe
	Secrets  SecretsProbe
	Store    InstanceStore
}

// Result lists the checks that passed, in order.
type Result struct {
	ChecksPassed []string
}

// StartPlatform validates the input, runs the three checks, then marks the
// instance running with its endpoints. Every check runs even after one fails,
// so the report names all of them.
func (a *Activities) StartPlatform(ctx context.Context, in Input) (Result, error) {
	if a.DB == nil || a.Identity == nil || a.Secrets == nil || a.Store == nil {
		return Result{}, nonRetryable(errTypePlatformConfig, errors.New("platform start is not configured"))
	}
	if err := validateInput(in); err != nil {
		return Result{}, nonRetryable(errTypePlatformInput, err)
	}

	instancePath, err := tenantsecrets.Path(in.TenantCode, in.Environment)
	if err != nil {
		return Result{}, nonRetryable(errTypePlatformInput, err)
	}

	var passed, failed []string

	if err := a.DB.Ping(ctx, instancePath); err != nil {
		failed = append(failed, CheckDatabase)
	} else {
		passed = append(passed, CheckDatabase)
	}

	if err := a.Identity.CheckIssuer(ctx, in.Endpoints.Issuer); err != nil {
		failed = append(failed, CheckIdentity)
	} else {
		passed = append(passed, CheckIdentity)
	}

	secretsOK := true
	for _, p := range []string{
		"tenants/" + in.TenantCode + "/identity",
		instancePath,
	} {
		ok, err := a.Secrets.Exists(ctx, p)
		if err != nil || !ok {
			secretsOK = false
		}
	}
	if secretsOK {
		passed = append(passed, CheckSecrets)
	} else {
		failed = append(failed, CheckSecrets)
	}

	if len(failed) > 0 {
		// Name the checks. Do not include probe errors, which can echo inputs.
		return Result{ChecksPassed: passed}, nonRetryable(errTypePlatformCheck,
			fmt.Errorf("instance %s failed health checks: %s", in.InstanceID, strings.Join(failed, ", ")))
	}

	if err := a.Store.MarkRunning(ctx, in.InstanceID, in.Endpoints); err != nil {
		return Result{ChecksPassed: passed}, errors.New("mark instance running: the store rejected the write")
	}
	return Result{ChecksPassed: passed}, nil
}

func validateInput(in Input) error {
	if _, err := uuid.Parse(in.InstanceID); err != nil {
		return errors.New("instance ID is not a valid UUID")
	}
	if !provisioning.ValidTenantCode(in.TenantCode) {
		return fmt.Errorf("tenant code %q is not valid", in.TenantCode)
	}
	if !environments[in.Environment] {
		return fmt.Errorf("environment %q is not one of dev, uat, prod", in.Environment)
	}
	ep := in.Endpoints
	if !hostnamePattern.MatchString(ep.PostgresHost) && net.ParseIP(ep.PostgresHost) == nil {
		return errors.New("postgres host is not a valid hostname or IP")
	}
	if ep.PostgresPort < 1 || ep.PostgresPort > 65535 {
		return fmt.Errorf("postgres port %d is out of range", ep.PostgresPort)
	}
	if !dbNamePattern.MatchString(ep.DatabaseName) {
		return errors.New("database name is not a valid identifier")
	}
	u, err := url.Parse(ep.Issuer)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return errors.New("issuer is not a valid http or https URL")
	}
	return nil
}

func nonRetryable(kind string, err error) error {
	return temporal.NewNonRetryableApplicationError(err.Error(), kind, err)
}
