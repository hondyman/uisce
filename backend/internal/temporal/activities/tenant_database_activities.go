package activities

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/migrations"
	"github.com/hondyman/uisce/backend/internal/provisioning"
	"github.com/hondyman/uisce/backend/internal/secrets"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/tenantdb"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"time"
)

// This file holds the saga steps that give a tenant database its own role and credential,
// point the tenant's app datasource at it, migrate it and prove it connects (ADR-030). They run
// only when the provisioning request names an App; without one the saga is unchanged.

// ErrTenantDatabaseNotConfigured: the cluster admin connection or the secrets provider is not
// configured. There is no default host, user or password.
var ErrTenantDatabaseNotConfigured = errors.New("tenant database provisioning is not configured")

// ErrTenantDatabaseCredentialLost: the credential was issued but the secrets store cannot
// produce it. Minting a new one would silently rotate a credential the registry believes is
// stable, so this stops and asks for an operator.
var ErrTenantDatabaseCredentialLost = errors.New("tenant database credential was issued but is not in the secrets store")

const (
	errTypeTenantDBConfig = "TenantDatabaseNotConfigured"
	errTypeTenantDBLost   = "TenantDatabaseCredentialLost"
	errTypeTenantDBInput  = "TenantDatabaseInvalidInput"
	errTypeTenantDBDrift  = "TenantDatabaseMigrationFailed"
)

var pgIdent = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// TenantDatabaseAdmin is how the saga reaches the shared Postgres cluster as an administrator.
// Every field is required; nothing is defaulted.
type TenantDatabaseAdmin struct {
	Host     string
	Port     int
	User     string
	Password string
}

// TenantDatabaseAdminFromEnv reads DB_HOST, DB_PORT, DB_USER and DB_PASS. Unlike the older
// provisioning steps it has no default for any of them.
func TenantDatabaseAdminFromEnv() (TenantDatabaseAdmin, error) {
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	a := TenantDatabaseAdmin{Host: os.Getenv("DB_HOST"), Port: port, User: os.Getenv("DB_USER"), Password: os.Getenv("DB_PASS")}
	if a.Host == "" || a.Port <= 0 || a.User == "" || a.Password == "" {
		return TenantDatabaseAdmin{}, fmt.Errorf("%w: DB_HOST, DB_PORT, DB_USER and DB_PASS are all required", ErrTenantDatabaseNotConfigured)
	}
	return a, nil
}

func (a TenantDatabaseAdmin) configured() bool {
	return a.Host != "" && a.Port > 0 && a.User != "" && a.Password != ""
}

// Open connects to one database of the cluster as the administrator.
func (a TenantDatabaseAdmin) Open(database string) (*sql.DB, error) {
	u := url.URL{
		Scheme: "postgres", User: url.UserPassword(a.User, a.Password),
		Host: net.JoinHostPort(a.Host, strconv.Itoa(a.Port)), Path: "/" + database,
	}
	if sm := os.Getenv("DB_SSLMODE"); sm != "" {
		q := url.Values{}
		q.Set("sslmode", sm)
		u.RawQuery = q.Encode()
	}
	return sql.Open("pgx", u.String())
}

// TenantDatabaseRole is the role's name for a database.
func TenantDatabaseRole(database string) (string, error) {
	if !pgIdent.MatchString(database) {
		return "", fmt.Errorf("database name %q is not a safe identifier", database)
	}
	role := database + "_app"
	if len(role) > 63 {
		return "", fmt.Errorf("database name %q is too long to derive a role", database)
	}
	return role, nil
}

func (a *TenantProvisioningActivities) validateTenantDatabase(in provisioning.TenantDatabaseInput) (string, error) {
	t := migrations.Target{TenantID: in.TenantID, App: in.App}
	if err := t.Validate(); err != nil {
		return "", nonRetryable(errTypeTenantDBInput, err)
	}
	role, err := TenantDatabaseRole(in.DatabaseName)
	if err != nil {
		return "", nonRetryable(errTypeTenantDBInput, err)
	}
	if !a.TenantDB.configured() {
		return "", nonRetryable(errTypeTenantDBConfig, ErrTenantDatabaseNotConfigured)
	}
	return role, nil
}

// BindTenantDatabase repoints the tenant's app datasource at its own database and records the
// binding in 'provisioning'. The datasource is the cloned row for in.App; any OTHER cloned row of
// the instance that still names the gold copy's database loses its database so it fails closed
// instead of resolving to the gold copy.
func (a *TenantProvisioningActivities) BindTenantDatabase(ctx context.Context, in provisioning.TenantDatabaseInput) (provisioning.TenantDatabaseBinding, error) {
	role, err := a.validateTenantDatabase(in)
	if err != nil {
		return provisioning.TenantDatabaseBinding{}, err
	}
	var out provisioning.TenantDatabaseBinding
	err = db.WithTenantTransaction(ctx, a.ControlDB.DB, in.TenantID, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT tpd.id
			FROM public.tenant_product_datasource tpd
			JOIN public.tenant_product tp ON tp.id = tpd.tenant_product_id
			JOIN public.alpha_datasource ad ON ad.id = tpd.alpha_datasource_id
			WHERE tp.datasource_id = $1 AND ad.datasource_code = $2`, in.InstanceID, in.App)
		if err != nil {
			return fmt.Errorf("find the %s datasource: %w", in.App, err)
		}
		var ids []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(ids) != 1 {
			return nonRetryable(errTypeTenantDBInput,
				fmt.Errorf("instance %s has %d datasource(s) for app %q; exactly one is required", in.InstanceID, len(ids), in.App))
		}
		out.DatasourceID = ids[0]
		out.Role = role
		if out.SecretPath, err = dscreds.CanonicalPath(dscreds.KindDatasource, in.TenantID, out.DatasourceID); err != nil {
			return nonRetryable(errTypeTenantDBInput, err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE public.tenant_product_datasource
			SET config = COALESCE(config, '{}'::jsonb)
			    || jsonb_build_object('host', $2::text, 'port', $3::int, 'database', $4::text, 'secret_path', $5::text)
			WHERE id = $1`, out.DatasourceID, a.TenantDB.Host, a.TenantDB.Port, in.DatabaseName, out.SecretPath); err != nil {
			return fmt.Errorf("repoint datasource: %w", err)
		}
		if in.GoldCopyDatabase != "" {
			if _, err := tx.ExecContext(ctx, `
				UPDATE public.tenant_product_datasource tpd
				SET config = tpd.config - 'database'
				FROM public.tenant_product tp
				WHERE tp.id = tpd.tenant_product_id AND tp.datasource_id = $1
				  AND tpd.id <> $2 AND tpd.config ->> 'database' = $3`, in.InstanceID, out.DatasourceID, in.GoldCopyDatabase); err != nil {
				return fmt.Errorf("clear cloned datasources that name the gold copy: %w", err)
			}
		}
		// The role is recorded once; a re-run never replaces it, and never downgrades a
		// binding that is already further along.
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO public.tenant_datasource_binding (datasource_id, tenant_id, pg_role, lifecycle_state)
			VALUES ($1, $2, $3, 'provisioning')
			ON CONFLICT (datasource_id) DO UPDATE
			SET pg_role = COALESCE(public.tenant_datasource_binding.pg_role, EXCLUDED.pg_role)`,
			out.DatasourceID, in.TenantID, role); err != nil {
			return fmt.Errorf("record binding: %w", err)
		}
		return nil
	})
	if err != nil {
		return provisioning.TenantDatabaseBinding{}, err
	}
	return out, nil
}

// ProvisionTenantDatabaseAccess creates the database's role, grants it the tenant's schema and
// stores its credential. The credential goes into the secrets store BEFORE the role is created
// with it, and pg_credential_issued_at is set only after both, so a crash at any point leaves a
// state this step can resume: the stored password is reused, never replaced. Once the marker is
// set a missing secret is ErrTenantDatabaseCredentialLost, never a new credential.
func (a *TenantProvisioningActivities) ProvisionTenantDatabaseAccess(ctx context.Context, in provisioning.TenantDatabaseInput) error {
	role, err := a.validateTenantDatabase(in)
	if err != nil {
		return err
	}
	if a.Secrets == nil {
		return nonRetryable(errTypeTenantDBConfig, fmt.Errorf("%w: no secrets provider", ErrTenantDatabaseNotConfigured))
	}
	path, err := dscreds.CanonicalPath(dscreds.KindDatasource, in.TenantID, in.DatasourceID)
	if err != nil {
		return nonRetryable(errTypeTenantDBInput, err)
	}

	var boundRole sql.NullString
	var issued sql.NullTime
	if err := db.WithTenantTransaction(ctx, a.ControlDB.DB, in.TenantID, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `
			SELECT pg_role, pg_credential_issued_at FROM public.tenant_datasource_binding WHERE datasource_id = $1`,
			in.DatasourceID).Scan(&boundRole, &issued)
	}); err != nil {
		return fmt.Errorf("read binding: %w", err)
	}
	if !boundRole.Valid || boundRole.String != role {
		return nonRetryable(errTypeTenantDBInput, fmt.Errorf("binding for datasource %s does not name role %q", in.DatasourceID, role))
	}

	stored, _ := a.Secrets.GetMap(ctx, path) // the store reports every failure as "not found"
	password := ""
	if stored[dscreds.KeyUsername] == role {
		password = stored[dscreds.KeyPassword]
	}
	if password == "" {
		if issued.Valid {
			return nonRetryable(errTypeTenantDBLost, fmt.Errorf("%w: %s", ErrTenantDatabaseCredentialLost, path))
		}
		if password, err = newPassword(); err != nil {
			return err
		}
		if err := a.Secrets.PutMap(ctx, path, map[string]string{dscreds.KeyUsername: role, dscreds.KeyPassword: password}); err != nil {
			return fmt.Errorf("store credential: %w", err)
		}
	}

	if err := a.ensureRole(ctx, in.DatabaseName, role, password); err != nil {
		return err
	}
	return db.WithTenantTransaction(ctx, a.ControlDB.DB, in.TenantID, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE public.tenant_datasource_binding
			SET pg_credential_issued_at = COALESCE(pg_credential_issued_at, now())
			WHERE datasource_id = $1`, in.DatasourceID)
		return err
	})
}

func newPassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ensureRole creates the role (or resets its password to the stored one) and grants it DML on
// the tenant's schema. It never gets DDL: migrations run as the administrator.
func (a *TenantProvisioningActivities) ensureRole(ctx context.Context, database, role, password string) error {
	conn, err := a.TenantDB.Open(database)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", database, err)
	}
	defer conn.Close()

	// format(%I, %L) quotes the identifier and the literal on the server, so neither the
	// role name nor the password is ever spliced into SQL by this code.
	var ddl string
	if err := conn.QueryRowContext(ctx, `
		SELECT CASE WHEN EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)
		            THEN format('ALTER ROLE %I WITH LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS', $1::text, $2::text)
		            ELSE format('CREATE ROLE %I WITH LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS', $1::text, $2::text)
		       END`, role, password).Scan(&ddl); err != nil {
		return fmt.Errorf("build role statement: %w", err)
	}
	if _, err := conn.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("create role: %w", redact(err, password))
	}

	for _, tmpl := range []string{
		`REVOKE CONNECT ON DATABASE %1$I FROM PUBLIC`,
		`GRANT CONNECT ON DATABASE %1$I TO %2$I`,
		`GRANT USAGE ON SCHEMA public TO %2$I`,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %2$I`,
		`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %2$I`,
		// Tables and sequences later migrations create get the same grants.
		`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %2$I`,
		`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO %2$I`,
	} {
		var stmt string
		if err := conn.QueryRowContext(ctx, `SELECT format($1::text, $2::text, $3::text)`, tmpl, database, role).Scan(&stmt); err != nil {
			return fmt.Errorf("build grant: %w", err)
		}
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("grant (%s): %w", strings.SplitN(tmpl, " %", 2)[0], err)
		}
	}
	return nil
}

func redact(err error, secret string) error {
	if err == nil || secret == "" {
		return err
	}
	return errors.New(strings.ReplaceAll(err.Error(), secret, "[redacted]"))
}

// ApplyTenantMigrations records the gold-copy baseline (if the request names one and the
// database has applied nothing) and applies the app's pending tenant migrations. The Report is
// returned so the saga can use it as its "migrations applied" probe; the step fails unless it is
// Done.
func (a *TenantProvisioningActivities) ApplyTenantMigrations(ctx context.Context, in provisioning.TenantDatabaseInput) (migrations.Report, error) {
	if _, err := a.validateTenantDatabase(in); err != nil {
		return migrations.Report{}, err
	}
	conn, err := a.TenantDB.Open(in.DatabaseName)
	if err != nil {
		return migrations.Report{}, fmt.Errorf("connect to %s: %w", in.DatabaseName, err)
	}
	defer conn.Close()

	runner := a.Migrations
	if runner == nil {
		runner = &migrations.TenantRunner{Root: migrations.DefaultTenantRoot()}
	}
	t := migrations.Target{TenantID: in.TenantID, App: in.App}
	if in.BaselineThrough != "" {
		if _, err := runner.Baseline(ctx, conn, t, in.BaselineThrough); err != nil && !errors.Is(err, migrations.ErrBaselineNotEmpty) {
			return migrations.Report{}, nonRetryable(errTypeTenantDBDrift, err)
		}
	}
	rep, err := runner.Apply(ctx, conn, t)
	switch {
	case errors.Is(err, migrations.ErrDrift), errors.Is(err, migrations.ErrNoApp):
		return rep, nonRetryable(errTypeTenantDBDrift, err)
	case err != nil:
		return rep, err
	case !rep.Done:
		return rep, nonRetryable(errTypeTenantDBDrift, fmt.Errorf("migrations for %s are not complete: %d pending", t, len(rep.Pending)))
	}
	return rep, nil
}

// ProbeTenantDatabase connects the way production will: through tenantdb, with the registered
// role and credential, as the owning tenant, while the binding is still 'provisioning'.
func (a *TenantProvisioningActivities) ProbeTenantDatabase(ctx context.Context, in provisioning.TenantDatabaseInput) error {
	creds := a.Creds
	if creds == nil {
		creds = dscreds.Default()
	}
	router, err := tenantdb.New(tenantdb.Config{
		Registry: &tenantdb.AlphaRegistry{
			DB: a.ControlDB.DB, Resolver: security.NewDBDatasourceResolver(a.ControlDB), Creds: creds,
		},
		CallerTenant:    db.GetTenantIDFromCtx,
		MaxPools:        1,
		MaxConnsPerPool: 1,
		IdleTTL:         time.Minute,
		DialTimeout:     10 * time.Second,
	})
	if err != nil {
		return err
	}
	defer router.Close()
	if err := router.Probe(db.WithTenantContextToCtx(ctx, in.TenantID), in.DatasourceID); err != nil {
		return err
	}
	return a.verifyRoleIsIsolated(ctx, in)
}

// ErrTenantRoleNotIsolated: the tenant's role can connect to a database other than its own.
var ErrTenantRoleNotIsolated = errors.New("tenant database role can connect to other databases")

const errTypeTenantDBIsolation = "TenantDatabaseNotIsolated"

// verifyRoleIsIsolated proves the new role can reach exactly one database, its own. It tries every
// other connectable database on the cluster as the role and fails provisioning if any accepts, or
// if it cannot tell (an unexpected error is not "denied").
//
// The saga revokes PUBLIC's CONNECT on the tenant's own database, but a cluster's other databases
// keep the default grant to PUBLIC: postgres, template1 and, unless an operator revoked it, the
// control plane's alpha. A role that can connect to postgres can list every database and role on
// the cluster, which is every tenant's code. This is deliberately a check and not a REVOKE: the
// saga does not silently change privileges that other roles on the cluster depend on.
func (a *TenantProvisioningActivities) verifyRoleIsIsolated(ctx context.Context, in provisioning.TenantDatabaseInput) error {
	role, err := TenantDatabaseRole(in.DatabaseName)
	if err != nil {
		return nonRetryable(errTypeTenantDBInput, err)
	}
	path, err := dscreds.CanonicalPath(dscreds.KindDatasource, in.TenantID, in.DatasourceID)
	if err != nil {
		return nonRetryable(errTypeTenantDBInput, err)
	}
	if a.Secrets == nil {
		return nonRetryable(errTypeTenantDBConfig, fmt.Errorf("%w: no secrets provider", ErrTenantDatabaseNotConfigured))
	}
	creds, err := a.Secrets.GetMap(ctx, path)
	if err != nil || creds[dscreds.KeyUsername] != role || creds[dscreds.KeyPassword] == "" {
		return fmt.Errorf("cannot verify isolation: the role's credential is not readable at %s", path)
	}

	adm, err := a.TenantDB.Open("postgres")
	if err != nil {
		return fmt.Errorf("connect to list databases: %w", err)
	}
	defer adm.Close()
	rows, err := adm.QueryContext(ctx, `SELECT datname FROM pg_database WHERE datallowconn AND datname <> $1 ORDER BY datname`, in.DatabaseName)
	if err != nil {
		return fmt.Errorf("list databases: %w", err)
	}
	var others []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return err
		}
		others = append(others, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	var reachable []string
	for _, name := range others {
		cfg, err := pgx.ParseConfig("")
		if err != nil {
			return err
		}
		cfg.Host, cfg.Port, cfg.Database = a.TenantDB.Host, uint16(a.TenantDB.Port), name
		cfg.User, cfg.Password = role, creds[dscreds.KeyPassword]
		cfg.ConnectTimeout = 10 * time.Second
		c, cerr := pgx.ConnectConfig(ctx, cfg)
		if cerr == nil {
			_ = c.Close(ctx)
			reachable = append(reachable, name)
			continue
		}
		var pgErr *pgconn.PgError
		if errors.As(cerr, &pgErr) && pgErr.Code == "42501" { // insufficient_privilege: CONNECT is denied
			continue
		}
		return fmt.Errorf("cannot verify isolation: connecting to %q as the tenant role failed for a reason other than a denied privilege: %w", name, redact(cerr, creds[dscreds.KeyPassword]))
	}
	if len(reachable) > 0 {
		var fixes []string
		for _, n := range reachable {
			fixes = append(fixes, fmt.Sprintf("REVOKE CONNECT ON DATABASE %s FROM PUBLIC;", pgQuoteIdent(n)))
		}
		return nonRetryable(errTypeTenantDBIsolation, fmt.Errorf(
			"%w: role %s can connect to %d other database(s): %s. Revoke the default PUBLIC grant (and grant CONNECT only to the roles that need it): %s",
			ErrTenantRoleNotIsolated, role, len(reachable), strings.Join(reachable, ", "), strings.Join(fixes, " ")))
	}
	return nil
}

func pgQuoteIdent(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

// ActivateTenantDatabase makes the binding live. Idempotent; it never moves a binding that is
// not 'provisioning' (a suspended or offboarding one stays so).
func (a *TenantProvisioningActivities) ActivateTenantDatabase(ctx context.Context, in provisioning.TenantDatabaseInput) error {
	return db.WithTenantTransaction(ctx, a.ControlDB.DB, in.TenantID, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE public.tenant_datasource_binding
			SET lifecycle_state = 'active', version = version + 1, updated_at = now()
			WHERE datasource_id = $1 AND lifecycle_state = 'provisioning'`, in.DatasourceID)
		return err
	})
}

// RollbackTenantDatabase undoes BindTenantDatabase and ProvisionTenantDatabaseAccess for a run
// that owns the tenant: drops the role (and its privileges in the tenant database, which must
// still exist), deletes the stored credential, and removes the binding. It only ever removes a
// binding still in 'provisioning'.
func (a *TenantProvisioningActivities) RollbackTenantDatabase(ctx context.Context, in provisioning.TenantDatabaseInput) error {
	if in.DatasourceID == "" {
		return nil // Bind never completed; nothing was created
	}
	var role sql.NullString
	var lifecycle sql.NullString
	err := db.WithTenantTransaction(ctx, a.ControlDB.DB, in.TenantID, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `SELECT pg_role, lifecycle_state FROM public.tenant_datasource_binding WHERE datasource_id = $1`,
			in.DatasourceID).Scan(&role, &lifecycle)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	})
	if err != nil {
		return fmt.Errorf("read binding: %w", err)
	}
	if !lifecycle.Valid || lifecycle.String != "provisioning" {
		return nil // never remove a live (or already-handled) binding
	}
	if role.Valid && role.String != "" && a.TenantDB.configured() && pgIdent.MatchString(in.DatabaseName) {
		if conn, err := a.TenantDB.Open(in.DatabaseName); err == nil {
			// DROP OWNED releases the grants in this database; without it DROP ROLE refuses.
			var drop string
			if qerr := conn.QueryRowContext(ctx, `SELECT format('DROP OWNED BY %I', $1::text)`, role.String).Scan(&drop); qerr == nil {
				_, _ = conn.ExecContext(ctx, drop)
			}
			conn.Close()
		}
		if adm, err := a.TenantDB.Open("postgres"); err == nil {
			defer adm.Close()
			var drop string
			if qerr := adm.QueryRowContext(ctx, `SELECT format('DROP ROLE IF EXISTS %I', $1::text)`, role.String).Scan(&drop); qerr == nil {
				if _, derr := adm.ExecContext(ctx, drop); derr != nil {
					return fmt.Errorf("drop role: %w", derr)
				}
			}
		}
	}
	if a.Secrets != nil {
		if path, err := dscreds.CanonicalPath(dscreds.KindDatasource, in.TenantID, in.DatasourceID); err == nil {
			if derr := a.Secrets.Delete(ctx, path); derr != nil && !errors.Is(derr, secrets.ErrSecretNotFound) {
				return fmt.Errorf("delete credential: %w", derr)
			}
		}
	}
	return db.WithTenantTransaction(ctx, a.ControlDB.DB, in.TenantID, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM public.tenant_datasource_binding WHERE datasource_id = $1 AND lifecycle_state = 'provisioning'`, in.DatasourceID)
		return err
	})
}

// ActivityRegistrar is the part of a Temporal worker the registration below needs.
type ActivityRegistrar interface {
	RegisterActivity(a interface{})
}

// RegisterTenantDatabaseActivities registers, on a worker, the saga steps that every worker
// running TenantInstanceProvisioningWorkflowFn must have: InspectProvisioningState (the
// compensation gate, which the workflow calls on every run) and the tenant-database steps. Both
// workers call this, so they cannot drift apart. None of these is BP-Designer client-safe: they
// create database roles and credentials, so they are never passed to RegisterSafeActivity.
func (a *TenantProvisioningActivities) RegisterTenantDatabaseActivities(w ActivityRegistrar) {
	w.RegisterActivity(a.InspectProvisioningState)
	w.RegisterActivity(a.BindTenantDatabase)
	w.RegisterActivity(a.ProvisionTenantDatabaseAccess)
	w.RegisterActivity(a.ApplyTenantMigrations)
	w.RegisterActivity(a.ProbeTenantDatabase)
	w.RegisterActivity(a.ActivateTenantDatabase)
	w.RegisterActivity(a.RollbackTenantDatabase)
}

// ConfigureTenantDatabaseFromEnv wires the cluster admin connection, the secrets provider and
// the credentials resolver from the environment. Anything missing is logged and left unset, so
// the tenant-database steps fail closed (non-retryably) instead of using a default.
func (a *TenantProvisioningActivities) ConfigureTenantDatabaseFromEnv() {
	if admin, err := TenantDatabaseAdminFromEnv(); err != nil {
		a.Logger.Warnf("tenant database provisioning disabled: %v", err)
	} else {
		a.TenantDB = admin
	}
	if p, err := dscreds.ProviderFromEnv(); err != nil {
		a.Logger.Warnf("tenant database credentials cannot be stored: %v", err)
	} else {
		a.Secrets = p
	}
	a.Creds = dscreds.Default()
}
