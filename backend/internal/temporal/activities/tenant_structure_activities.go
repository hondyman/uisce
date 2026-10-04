package activities

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/migrations"
	"github.com/hondyman/uisce/backend/internal/provisioning"
	"github.com/hondyman/uisce/backend/internal/tenantschema"
)

// A tenant's structure is compiled from what alpha holds after the gold copy's scan, not cloned from the gold copy's
// database (ADR-048). Two steps: PlanTenantStructure runs first, before anything is created, so a scan that is not
// deployable (incomplete, stale in a way the scan itself shows, unsupported, not the gold copy's) refuses the run with
// nothing to undo; ApplyTenantStructure then applies exactly what was planned, and refuses if the template has changed.

const (
	errTypeTenantStructureRefused = "TenantStructureRefused" // the template cannot be deployed as it stands; rescan or fix it
	errTypeTenantStructureChanged = "TenantStructureChanged" // the template compiles to something else than was planned

	// StructureApp is the migration-log target a compiled structure is recorded under. It is not a directory.
	StructureApp = "structure"
	// structureFile is the one generated migration. Its sha256 is the plan hash, so a database built from a different
	// compilation is drift, never silently mixed.
	structureFile = "0001_structure.up.sql"
)

// ErrTenantStructureNotConfigured: the worker was started without a template loader.
var ErrTenantStructureNotConfigured = errors.New("tenant structure compilation is not configured on this worker")

// compile loads the template and compiles it. Refusals are non-retryable; an infrastructure error is retried.
func (a *TenantProvisioningActivities) compileStructure(ctx context.Context, templateDatasourceID string) (*tenantschema.Plan, []string, error) {
	if a.Templates == nil {
		return nil, nil, nonRetryable(errTypeTenantDBConfig, ErrTenantStructureNotConfigured)
	}
	tpl, err := a.Templates.Load(ctx, templateDatasourceID)
	switch {
	case errors.Is(err, tenantschema.ErrNotGoldCopy), errors.Is(err, tenantschema.ErrTemplateNotFound),
		errors.Is(err, tenantschema.ErrNoSchemas), errors.Is(err, tenantschema.ErrBadSchemaName), errors.Is(err, tenantschema.ErrNoNodes):
		return nil, nil, nonRetryable(errTypeTenantStructureRefused, err)
	case err != nil:
		return nil, nil, fmt.Errorf("load the template: %w", err)
	}
	plan, err := tenantschema.Compile(tpl.Nodes, tenantschema.Options{Schemas: tpl.Schemas})
	switch {
	case errors.Is(err, tenantschema.ErrScanNotComplete), errors.Is(err, tenantschema.ErrUnsupported), errors.Is(err, tenantschema.ErrBadScan):
		return nil, nil, nonRetryable(errTypeTenantStructureRefused, err)
	case err != nil:
		return nil, nil, nonRetryable(errTypeTenantStructureRefused, err)
	}
	return plan, tpl.Schemas, nil
}

// PlanTenantStructure compiles the template and reports its hash. It creates nothing.
func (a *TenantProvisioningActivities) PlanTenantStructure(ctx context.Context, in provisioning.TenantDatabaseInput) (provisioning.StructurePlan, error) {
	if in.TemplateDatasourceID == "" {
		return provisioning.StructurePlan{}, nonRetryable(errTypeTenantDBInput, errors.New("a template datasource is required"))
	}
	// Planning runs before anything is created, so this is where a malformed role group refuses the run with nothing to undo.
	if err := a.checkRoleGroup(); err != nil {
		return provisioning.StructurePlan{}, err
	}
	plan, schemas, err := a.compileStructure(ctx, in.TemplateDatasourceID)
	if err != nil {
		return provisioning.StructurePlan{}, err
	}
	return provisioning.StructurePlan{Hash: plan.Hash(), Tables: plan.Tables, Statements: len(plan.Statements), Schemas: schemas}, nil
}

// ApplyTenantStructure applies the planned structure to the tenant's database in one transaction, through the same
// migration runner (advisory lock, log, drift) as every other tenant migration. The Report is the saga's "applied" probe.
func (a *TenantProvisioningActivities) ApplyTenantStructure(ctx context.Context, in provisioning.TenantDatabaseInput) (migrations.Report, error) {
	role, err := a.validateTenantDatabase(in)
	if err != nil {
		return migrations.Report{}, err
	}
	if in.TemplateDatasourceID == "" || in.StructureHash == "" {
		return migrations.Report{}, nonRetryable(errTypeTenantDBInput, errors.New("a template datasource and the planned structure hash are required"))
	}
	plan, schemas, err := a.compileStructure(ctx, in.TemplateDatasourceID)
	if err != nil {
		return migrations.Report{}, err
	}
	if plan.Hash() != in.StructureHash {
		return migrations.Report{}, nonRetryable(errTypeTenantStructureChanged,
			fmt.Errorf("the template now compiles to %s, not the planned %s: it was rescanned during this run; start the run again", short(plan.Hash()), short(in.StructureHash)))
	}

	conn, err := a.TenantDB.Open(in.DatabaseName)
	if err != nil {
		return migrations.Report{}, fmt.Errorf("connect to %s: %w", in.DatabaseName, err)
	}
	defer conn.Close()

	base := migrations.TenantRunner{Root: migrations.DefaultTenantRoot()}
	if a.Migrations != nil {
		base = *a.Migrations
	}
	runner := base
	prev := base.Generated
	runner.Generated = func(t migrations.Target) ([]migrations.GeneratedFile, bool, error) {
		if t.App == StructureApp {
			return []migrations.GeneratedFile{{Filename: structureFile, SQL: plan.SQL()}}, true, nil
		}
		if prev != nil {
			return prev(t)
		}
		return nil, false, nil
	}
	t := migrations.Target{TenantID: in.TenantID, App: StructureApp}
	rep, err := runner.Apply(ctx, conn, t)
	switch {
	case errors.Is(err, migrations.ErrDrift):
		return rep, nonRetryable(errTypeTenantDBDrift, err)
	case err != nil:
		return rep, err
	case !rep.Done:
		return rep, nonRetryable(errTypeTenantDBDrift, fmt.Errorf("the structure for %s is not complete: %d pending", t, len(rep.Pending)))
	}
	// The role was created before the structure existed and was granted on public only. Grant it each schema of the
	// template, and the tables and sequences later changes add to them. It never gets DDL: the structure is built by the
	// administrator. Grants are repeatable, so a retried run converges.
	if err := grantStructureAccess(ctx, conn, in.DatabaseName, role, schemas); err != nil {
		return rep, fmt.Errorf("grant the tenant role access to the structure: %w", err)
	}
	return rep, nil
}

// grantStructureAccess gives the tenant's role what it needs on every schema of its structure, no more: usage, DML on
// tables, use of sequences, and the same for what is created later. Names are quoted by the server (format %I), never by
// string concatenation here.
func grantStructureAccess(ctx context.Context, conn *sql.DB, database, role string, schemas []string) error {
	for _, schema := range schemas {
		for _, tmpl := range []string{
			`GRANT USAGE ON SCHEMA %3$I TO %2$I`,
			`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %3$I TO %2$I`,
			`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA %3$I TO %2$I`,
			`ALTER DEFAULT PRIVILEGES IN SCHEMA %3$I GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %2$I`,
			`ALTER DEFAULT PRIVILEGES IN SCHEMA %3$I GRANT USAGE, SELECT ON SEQUENCES TO %2$I`,
		} {
			var stmt string
			if err := conn.QueryRowContext(ctx, `SELECT format($1::text, $2::text, $3::text, $4::text)`, tmpl, database, role, schema).Scan(&stmt); err != nil {
				return fmt.Errorf("build grant: %w", err)
			}
			if _, err := conn.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("grant on %s (%s): %w", schema, strings.SplitN(tmpl, " %", 2)[0], err)
			}
		}
	}
	return nil
}

// short is the first 12 characters of a hash for a message, safe for any input: the planned hash comes from workflow
// input and must never be able to panic the activity.
func short(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}
