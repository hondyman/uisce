package activities

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hondyman/uisce/backend/internal/migrations/ormmove"
	"github.com/hondyman/uisce/backend/internal/provisioning"
)

// This file holds the saga steps of the product path ("create tenant X in region R, register
// product P with label L"). They create the tenant database through the one administrator
// connection the worker holds (a.TenantDB), after proving that connection is the cluster the
// region names, instead of through DATABASE_URL and defaults as the older CreateTenantDatabase does.

const errTypeTenantDBRegion = "TenantDatabaseRegionMismatch"

// sameHost compares two hosts. Names compare case-insensitively; two IP addresses compare by value
// ("::1" and "0:0:0:0:0:0:0:1" are one host). Nothing is resolved: an unresolvable difference is a
// refusal, not a guess.
func sameHost(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if ia, ib := net.ParseIP(a), net.ParseIP(b); ia != nil && ib != nil {
		return ia.Equal(ib)
	}
	return strings.EqualFold(a, b)
}

// isDuplicateDatabase matches the SQLSTATE (duplicate_database), not the message, so a retry after a
// partial failure continues instead of failing forever.
func isDuplicateDatabase(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P04"
}

// checkRegionCluster refuses unless the worker's administrator connection is the cluster the region
// names.
func (a *TenantProvisioningActivities) checkRegionCluster(in provisioning.RegionDatabaseInput) error {
	if !pgIdent.MatchString(in.DatabaseName) {
		return nonRetryable(errTypeTenantDBInput, fmt.Errorf("database name %q is not a safe identifier", in.DatabaseName))
	}
	if in.Region == "" || in.Host == "" || in.Port <= 0 {
		return nonRetryable(errTypeTenantDBInput, fmt.Errorf("the region, host and port of the cluster are required"))
	}
	if !a.TenantDB.configured() {
		return nonRetryable(errTypeTenantDBConfig, ErrTenantDatabaseNotConfigured)
	}
	if !sameHost(in.Host, a.TenantDB.Host) || in.Port != a.TenantDB.Port {
		return nonRetryable(errTypeTenantDBRegion, fmt.Errorf(
			"region %s runs on %s:%d, but this worker administers %s:%d; no database was created",
			in.Region, in.Host, in.Port, a.TenantDB.Host, a.TenantDB.Port))
	}
	return nil
}

// AssertRegionCluster runs before anything is created. It proves the worker administers the region's
// cluster, and that no database of this name exists there: a name already in use is another tenant's
// or an earlier run's, and "already exists" must not be taken as ours.
func (a *TenantProvisioningActivities) AssertRegionCluster(ctx context.Context, in provisioning.RegionDatabaseInput) error {
	if err := a.checkRegionCluster(in); err != nil {
		return err
	}
	adm, err := a.TenantDB.Open("postgres")
	if err != nil {
		return fmt.Errorf("connect to the cluster: %w", err)
	}
	defer adm.Close()
	var exists bool
	if err := adm.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, in.DatabaseName).Scan(&exists); err != nil {
		return fmt.Errorf("check for database %s: %w", in.DatabaseName, err)
	}
	if exists {
		return nonRetryable(errTypeTenantDBInput, fmt.Errorf("database %s already exists on %s:%d; choose another label", in.DatabaseName, in.Host, in.Port))
	}
	return nil
}

// CreateTenantDatabaseInRegion creates the database on the region's cluster. It is safe to retry:
// the three statements are the ones CreateTenantDatabase runs, in the same order, and each is
// idempotent, so a retry after a failure anywhere in them finishes the job.
func (a *TenantProvisioningActivities) CreateTenantDatabaseInRegion(ctx context.Context, in provisioning.RegionDatabaseInput) error {
	if err := a.checkRegionCluster(in); err != nil {
		return err
	}
	adm, err := a.TenantDB.Open("postgres")
	if err != nil {
		return fmt.Errorf("connect to the cluster: %w", err)
	}
	defer adm.Close()

	stmts := createTenantDatabaseStatements(in.DatabaseName)
	if _, err := adm.ExecContext(ctx, stmts.create); err != nil && !isDuplicateDatabase(err) {
		return fmt.Errorf("failed to create database: %w", err)
	}
	if _, err := adm.ExecContext(ctx, stmts.closePublic); err != nil {
		return fmt.Errorf("failed to revoke PUBLIC connect on %s: %w", in.DatabaseName, err)
	}
	if _, err := adm.ExecContext(ctx, stmts.allowConnections); err != nil {
		return fmt.Errorf("failed to enable connections to %s: %w", in.DatabaseName, err)
	}
	a.Logger.Infof("Created database %s on %s:%d (region %s)", in.DatabaseName, in.Host, in.Port, in.Region)
	return nil
}

// RollbackCreateTenantDatabaseInRegion drops a database CreateTenantDatabaseInRegion made. It is
// compensation: the workflow calls it only for a database this run created.
func (a *TenantProvisioningActivities) RollbackCreateTenantDatabaseInRegion(ctx context.Context, in provisioning.RegionDatabaseInput) error {
	if err := a.checkRegionCluster(in); err != nil {
		return err
	}
	adm, err := a.TenantDB.Open("postgres")
	if err != nil {
		return fmt.Errorf("connect to the cluster: %w", err)
	}
	defer adm.Close()
	// FORCE ends any connection still open (a probe that has not closed yet), which would otherwise
	// leave the database behind and the name taken.
	if _, err := adm.ExecContext(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, in.DatabaseName)); err != nil {
		a.Logger.Errorf("Failed to roll back database %s: %v", in.DatabaseName, err)
		return err
	}
	return nil
}

// SeedTenantDatabase gives a new tenant database the reference rows its app needs, after its
// structure is applied. For the orm app these are the rows of the shared reference tenant, read from
// the control database; the tenant's own tables start empty. It is idempotent (every insert is ON
// CONFLICT DO NOTHING) and verified: it fails unless every table's counts agree.
//
// An app with no seed is a no-op, not an error: its structure is all it needs.
func (a *TenantProvisioningActivities) SeedTenantDatabase(ctx context.Context, in provisioning.TenantDatabaseInput) (ormmove.Report, error) {
	if !pgIdent.MatchString(in.DatabaseName) {
		return ormmove.Report{}, nonRetryable(errTypeTenantDBInput, fmt.Errorf("database name %q is not a safe identifier", in.DatabaseName))
	}
	if in.App != "orm" {
		a.Logger.Infof("App %q has no seed; nothing to do for %s", in.App, in.DatabaseName)
		return ormmove.Report{TenantID: in.TenantID, Done: true}, nil
	}
	if !a.TenantDB.configured() {
		return ormmove.Report{}, nonRetryable(errTypeTenantDBConfig, ErrTenantDatabaseNotConfigured)
	}
	if a.ControlDB == nil {
		return ormmove.Report{}, nonRetryable(errTypeTenantDBConfig, fmt.Errorf("%w: no control database to read the reference rows from", ErrTenantDatabaseNotConfigured))
	}
	target, err := a.TenantDB.Open(in.DatabaseName)
	if err != nil {
		return ormmove.Report{}, fmt.Errorf("connect to %s: %w", in.DatabaseName, err)
	}
	defer target.Close()
	return a.moveSeed(ctx, a.ControlDB.DB, target, in)
}

// moveSeed copies and verifies. It is split out so a test can hand it both ends.
func (a *TenantProvisioningActivities) moveSeed(ctx context.Context, source, target *sql.DB, in provisioning.TenantDatabaseInput) (ormmove.Report, error) {
	rep, err := (&ormmove.Mover{Source: source}).Move(ctx, target, in.TenantID)
	if err != nil {
		return rep, fmt.Errorf("seed %s: %w", in.DatabaseName, err)
	}
	if !rep.Done {
		return rep, nonRetryable(errTypeTenantDBDrift, fmt.Errorf("seed %s: %s", in.DatabaseName, rep.Error))
	}
	a.Logger.Infof("Seeded %s: %d tables checked", in.DatabaseName, len(rep.Counts))
	return rep, nil
}
