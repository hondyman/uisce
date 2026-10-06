package infra

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

// ADR-036: a tenant's lakehouse audit chain is copied into an Iceberg table in the tenant's own
// warehouse, written by StarRocks through a per-tenant external catalog over Lakekeeper.
//
// This adapter builds DDL that embeds the tenant's storage credential and runs INSERTs, so it
// follows three rules:
//   - Values placed inside a statement (catalog URI, warehouse, keys) are VALIDATED against a
//     conservative character set and refused otherwise. They are never escaped-and-hoped.
//   - Row data goes through bound parameters (the driver interpolates and escapes client-side,
//     which StarRocks accepts where server-side prepared INSERTs are not guaranteed).
//   - Any error text is redacted of the credential, because a MySQL error can echo the statement.
//
// It has not been run against a live StarRocks or Lakekeeper. The DDL follows the StarRocks 3.3
// documentation and the property names this repository's own external catalogs already use.

// AuditRow is one audit entry as stored in the Iceberg copy. Before/After are JSON text.
type AuditRow struct {
	ID         int64
	At         time.Time
	ActorID    string
	ActorRole  string
	Action     string
	BeforeJSON string
	AfterJSON  string
	PrevHash   string
	Hash       string
}

// AuditDestinationSpec identifies a tenant's warehouse and the credential StarRocks reads it with.
//
// TenantName is the tenant's canonical name (matches public.tenants.name). The audit destination
// router uses it to look up the per-tenant DSN and resource group; without it, the router falls
// back to the legacy single-DSN path (or errors under LAKEHOUSE_STARROCKS_REQUIRE_TENANT_DSN).
type AuditDestinationSpec struct {
	TenantID      uuid.UUID
	TenantName    string
	WarehouseName string
	AccessKeyID   string
	SecretKey     string
}

// AuditDestination is where a tenant's audit copy lives.
type AuditDestination interface {
	// EnsureAuditDestination creates the tenant's catalog, the audit database and the events
	// table if they are missing. Idempotent.
	EnsureAuditDestination(ctx context.Context, spec AuditDestinationSpec) error
	// MaxAuditID is the highest audit id already in the copy, or 0 if it is empty. It is the
	// resume point: the destination, not a counter, says where the copy has got to.
	MaxAuditID(ctx context.Context, tenantID uuid.UUID) (int64, error)
	// AuditHash returns the hash stored for an audit id, and whether the id is present.
	AuditHash(ctx context.Context, tenantID uuid.UUID, id int64) (string, bool, error)
	// AppendAudit adds the rows in ONE statement, so it is one Iceberg commit. Rows must be in
	// strictly increasing id order.
	AppendAudit(ctx context.Context, tenantID uuid.UUID, rows []AuditRow) error
	// AuditRange returns up to limit copied rows with an id greater than afterID, in id order. It
	// exists so the copy can be verified against alpha (ADR-035); nothing is ever shipped from it.
	AuditRange(ctx context.Context, tenantID uuid.UUID, afterID int64, limit int) ([]AuditRow, error)
}

// MaxAppendRows bounds one append. One statement is one Iceberg commit, and under Object Lock
// every commit's files are kept for the whole retention period, so batches are large but bounded.
const MaxAppendRows = 1000

// StarRocksConfig is the deployment's view of how StarRocks reaches Lakekeeper and the object store.
type StarRocksConfig struct {
	// CatalogURI is the Iceberg REST endpoint StarRocks uses (Lakekeeper's catalog URL).
	CatalogURI string
	// S3Endpoint is the object store endpoint as StarRocks' backends see it, which can differ
	// from the one the worker uses.
	S3Endpoint string
	// S3Region is optional.
	S3Region string
}

// StarRocks is the audit destination backed by a StarRocks external Iceberg catalog.
type StarRocks struct {
	db  *sql.DB
	cfg StarRocksConfig
}

var (
	catalogNameRE = regexp.MustCompile(`^ivy_t_[0-9a-f]{32}$`)
	// safeValueRE is every character a value placed in a PROPERTIES literal may contain. No
	// quote, backslash, whitespace, semicolon or comment character is in it, so no value can end
	// the literal or the statement.
	safeValueRE = regexp.MustCompile(`^[A-Za-z0-9._~:/@+=-]+$`)
)

// CatalogName is the tenant's StarRocks external catalog: "ivy_t_" plus the tenant id without
// hyphens. Derived from the id, never supplied, so a tenant cannot be pointed at another's catalog.
func CatalogName(tenantID uuid.UUID) (string, error) {
	if tenantID == uuid.Nil {
		return "", errors.New("a tenant id is required")
	}
	return "ivy_t_" + strings.ReplaceAll(tenantID.String(), "-", ""), nil
}

// NewStarRocks wraps an open connection. Row data is sent as bound parameters, which the driver
// interpolates client-side (interpolateParams), so the connection must have been opened with it.
func NewStarRocks(db *sql.DB, cfg StarRocksConfig) (*StarRocks, error) {
	if db == nil {
		return nil, errors.New("a StarRocks connection is required")
	}
	for name, v := range map[string]string{"catalog URI": cfg.CatalogURI, "S3 endpoint": cfg.S3Endpoint} {
		if !safeValueRE.MatchString(v) {
			return nil, fmt.Errorf("StarRocks %s is empty or contains characters that are not allowed in a catalog property", name)
		}
	}
	if cfg.S3Region != "" && !safeValueRE.MatchString(cfg.S3Region) {
		return nil, errors.New("StarRocks S3 region contains characters that are not allowed in a catalog property")
	}
	return &StarRocks{db: db, cfg: cfg}, nil
}

// OpenStarRocksDSN opens a StarRocks connection from a MySQL DSN and forces client-side parameter
// interpolation. Optional init statements run on every new physical connection (database/sql may
// open multiple per *sql.DB; the init connector pins each one). A failing init statement closes
// the connection and returns the error — a bad init is a deployment-critical outage for that
// tenant, so the cost of returning here is the right one.
func OpenStarRocksDSN(dsn string, initSQL ...string) (*sql.DB, error) {
	c, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, errors.New("the StarRocks DSN is not valid") // never echo it: it holds a password
	}
	c.InterpolateParams = true
	base, err := mysql.NewConnector(c)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to StarRocks: %w", err)
	}
	if len(initSQL) == 0 {
		return sql.OpenDB(base), nil
	}
	return sql.OpenDB(&initConnector{base: base, init: append([]string(nil), initSQL...)}), nil
}

// initConnector wraps a driver.Connector so every new physical connection runs the given init
// statements. The init list is copied so a caller mutating their slice cannot affect subsequent
// connections.
type initConnector struct {
	base driver.Connector
	init []string
}

func (c *initConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	for _, q := range c.init {
		// go-sql-driver/mysql implements ExecerContext; assert with the two-value form so a
		// future driver swap that drops it fails with a clear error rather than a panic.
		exec, ok := conn.(driver.ExecerContext)
		if !ok {
			_ = conn.Close()
			return nil, fmt.Errorf("the StarRocks driver does not implement ExecerContext, init SQL %q cannot run", q)
		}
		if _, err := exec.ExecContext(ctx, q, nil); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("run init SQL %q: %w", q, err)
		}
	}
	return conn, nil
}

func (c *initConnector) Driver() driver.Driver { return c.base.Driver() }

func (s *StarRocks) redactor(spec AuditDestinationSpec) func(error) error {
	return func(err error) error {
		if err == nil {
			return nil
		}
		msg := err.Error()
		for _, secret := range []string{spec.SecretKey, spec.AccessKeyID} {
			if secret != "" {
				msg = strings.ReplaceAll(msg, secret, "***")
			}
		}
		return errors.New(msg)
	}
}

func (s *StarRocks) EnsureAuditDestination(ctx context.Context, spec AuditDestinationSpec) error {
	cat, err := CatalogName(spec.TenantID)
	if err != nil {
		return err
	}
	for name, v := range map[string]string{"warehouse": spec.WarehouseName, "access key": spec.AccessKeyID, "secret key": spec.SecretKey} {
		if !safeValueRE.MatchString(v) {
			return fmt.Errorf("the %s is empty or contains characters that are not allowed in a catalog property", name)
		}
	}
	redact := s.redactor(spec)

	props := []string{
		`"type" = "iceberg"`,
		`"iceberg.catalog.type" = "rest"`,
		`"iceberg.catalog.uri" = "` + s.cfg.CatalogURI + `"`,
		`"iceberg.catalog.warehouse" = "` + spec.WarehouseName + `"`,
		`"aws.s3.endpoint" = "` + s.cfg.S3Endpoint + `"`,
		`"aws.s3.access_key" = "` + spec.AccessKeyID + `"`,
		`"aws.s3.secret_key" = "` + spec.SecretKey + `"`,
		`"aws.s3.enable_path_style_access" = "true"`,
	}
	if s.cfg.S3Region != "" {
		props = append(props, `"aws.s3.region" = "`+s.cfg.S3Region+`"`)
	}

	stmts := []string{
		"CREATE EXTERNAL CATALOG IF NOT EXISTS " + cat + " PROPERTIES (" + strings.Join(props, ", ") + ")",
		"CREATE DATABASE IF NOT EXISTS " + cat + ".audit",
		"CREATE TABLE IF NOT EXISTS " + cat + ".audit.lakehouse_events (" +
			"`id` BIGINT, `tenant_id` STRING, `at` DATETIME, `actor_id` STRING, `actor_role` STRING, " +
			"`action` STRING, `before_json` STRING, `after_json` STRING, `prev_hash` STRING, `hash` STRING" +
			`) PROPERTIES ("file_format" = "parquet")`,
	}
	for i, q := range stmts {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return redact(fmt.Errorf("prepare the audit destination (step %d of %d): %w", i+1, len(stmts), err))
		}
	}
	return nil
}

func (s *StarRocks) table(tenantID uuid.UUID) (string, error) {
	cat, err := CatalogName(tenantID)
	if err != nil {
		return "", err
	}
	if !catalogNameRE.MatchString(cat) {
		return "", errors.New("invalid catalog name")
	}
	return cat + ".audit.lakehouse_events", nil
}

func (s *StarRocks) MaxAuditID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	t, err := s.table(tenantID)
	if err != nil {
		return 0, err
	}
	var max sql.NullInt64
	if err := s.db.QueryRowContext(ctx, "SELECT MAX(`id`) FROM "+t).Scan(&max); err != nil {
		return 0, fmt.Errorf("read the audit copy's highest id: %w", err)
	}
	if !max.Valid {
		return 0, nil
	}
	return max.Int64, nil
}

func (s *StarRocks) AuditHash(ctx context.Context, tenantID uuid.UUID, id int64) (string, bool, error) {
	t, err := s.table(tenantID)
	if err != nil {
		return "", false, err
	}
	var h string
	err = s.db.QueryRowContext(ctx, "SELECT `hash` FROM "+t+" WHERE `id` = ?", id).Scan(&h)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read audit entry %d from the copy: %w", id, err)
	}
	return h, true, nil
}

func (s *StarRocks) AppendAudit(ctx context.Context, tenantID uuid.UUID, rows []AuditRow) error {
	if len(rows) == 0 {
		return errors.New("there are no audit rows to append")
	}
	if len(rows) > MaxAppendRows {
		return fmt.Errorf("at most %d rows per append, got %d", MaxAppendRows, len(rows))
	}
	for i := 1; i < len(rows); i++ {
		if rows[i].ID <= rows[i-1].ID {
			return errors.New("audit rows must be in strictly increasing id order")
		}
	}
	t, err := s.table(tenantID)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("INSERT INTO " + t + " (`id`, `tenant_id`, `at`, `actor_id`, `actor_role`, `action`, `before_json`, `after_json`, `prev_hash`, `hash`) VALUES ")
	args := make([]any, 0, len(rows)*10)
	for i, r := range rows {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args, r.ID, tenantID.String(), r.At.UTC().Format("2006-01-02 15:04:05.000000"),
			r.ActorID, r.ActorRole, r.Action, r.BeforeJSON, r.AfterJSON, r.PrevHash, r.Hash)
	}
	if _, err := s.db.ExecContext(ctx, b.String(), args...); err != nil {
		return fmt.Errorf("append %d audit rows: %w", len(rows), err)
	}
	return nil
}

func (s *StarRocks) AuditRange(ctx context.Context, tenantID uuid.UUID, afterID int64, limit int) ([]AuditRow, error) {
	if limit < 1 || limit > MaxAppendRows {
		return nil, fmt.Errorf("limit must be between 1 and %d", MaxAppendRows)
	}
	t, err := s.table(tenantID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT `id`, `at`, `actor_id`, `actor_role`, `action`, `before_json`, `after_json`, `prev_hash`, `hash` FROM "+
		t+" WHERE `id` > ? ORDER BY `id` ASC LIMIT ?", afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("read audit rows after %d from the copy: %w", afterID, err)
	}
	defer rows.Close()
	var out []AuditRow
	for rows.Next() {
		var r AuditRow
		var at any
		var before, after sql.NullString
		if err := rows.Scan(&r.ID, &at, &r.ActorID, &r.ActorRole, &r.Action, &before, &after, &r.PrevHash, &r.Hash); err != nil {
			return nil, fmt.Errorf("read an audit row from the copy: %w", err)
		}
		if r.At, err = parseCopyTime(at); err != nil {
			return nil, fmt.Errorf("audit entry %d in the copy: %w", r.ID, err)
		}
		r.BeforeJSON, r.AfterJSON = before.String, after.String
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read audit rows from the copy: %w", err)
	}
	return out, nil
}

// parseCopyTime reads the `at` column whether the driver returns a time or its text. Anything else is
// an error: a verifier that guessed a timestamp could call a changed row unchanged.
func parseCopyTime(v any) (time.Time, error) {
	switch x := v.(type) {
	case time.Time:
		return x.UTC(), nil
	case []byte:
		return parseCopyTimeText(string(x))
	case string:
		return parseCopyTimeText(x)
	}
	return time.Time{}, fmt.Errorf("unexpected timestamp type %T", v)
}

func parseCopyTimeText(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02 15:04:05.999999", "2006-01-02 15:04:05", time.RFC3339Nano} {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unreadable timestamp %q", s)
}

// ---- configuration from the environment ----

// AuditDestinationFromEnv returns the StarRocks-backed destination, or one that reports
// ErrNotConfigured. It is built on first use: opening the connection must never slow or break
// worker startup.
//
//	LAKEHOUSE_STARROCKS_DSN                          user:password@tcp(host:9030)/  (legacy fallback)
//	LAKEHOUSE_STARROCKS_CATALOG_URI                  Lakekeeper's Iceberg REST URL as StarRocks reaches it
//	LAKEHOUSE_STARROCKS_S3_ENDPOINT                  the object store as StarRocks' backends reach it
//	LAKEHOUSE_STARROCKS_S3_REGION                    optional
//
// Per-tenant overrides (any of these present switches the destination into per-tenant mode;
// env vars whose suffix after TENANT_ does not match a real tenant name silently win nothing,
// and the destination falls back to LAKEHOUSE_STARROCKS_DSN with a WARN log):
//
//	LAKEHOUSE_STARROCKS_DSN_TENANT_<NAME>            user:password@tcp(host:9030)/<db>  (DB in path: cross-tenant access fails at connect time)
//	LAKEHOUSE_STARROCKS_RESOURCE_GROUP_TENANT_<NAME> resource group name to pin via SET resource_group = '<rg>'
//
//	LAKEHOUSE_STARROCKS_REQUIRE_TENANT_DSN           "true" → unmapped tenants ERROR (no silent fallback).
//	                                                 Default false keeps single-tenant deploys unbroken;
//	                                                 flip on once every real tenant has its own DSN.
func AuditDestinationFromEnv() AuditDestination {
	return &lazyDestination{build: func() (AuditDestination, error) { return destinationBuilder() }}
}

// StatusAdminDBFromEnv opens a read-only StarRocks connection for the status panel. It uses
// the LEGACY DSN (LAKEHOUSE_STARROCKS_DSN), not the per-tenant router, because the panel's
// SHOW FRONTENDS / SHOW BACKENDS / SHOW RESOURCE GROUPS ALL are cluster-wide queries that
// don't go through tenant routing. Returns nil, nil if the env is not configured; the
// panel then serves the tenants-only view with a "cluster not configured" note.
//
// Any StarRocks user can run these SHOW commands; the root credential is NOT required. On a
// multi-tenant cluster, a dedicated read-only user (e.g. status_reader) is preferred, but
// this function does not require it.
func StatusAdminDBFromEnv() (*sql.DB, error) {
	dsn := os.Getenv("LAKEHOUSE_STARROCKS_DSN")
	if dsn == "" {
		return nil, nil
	}
	return OpenStarRocksDSN(dsn)
}

// destinationBuilder is a variable so a test can count constructions.
var destinationBuilder = buildDestinationFromEnv

func buildDestinationFromEnv() (AuditDestination, error) {
	uri := os.Getenv("LAKEHOUSE_STARROCKS_CATALOG_URI")
	endpoint := os.Getenv("LAKEHOUSE_STARROCKS_S3_ENDPOINT")
	if uri == "" || endpoint == "" {
		return nil, notConfigured("StarRocks", "LAKEHOUSE_STARROCKS_DSN", "LAKEHOUSE_STARROCKS_CATALOG_URI", "LAKEHOUSE_STARROCKS_S3_ENDPOINT")
	}
	cfg := StarRocksConfig{CatalogURI: uri, S3Endpoint: endpoint, S3Region: os.Getenv("LAKEHOUSE_STARROCKS_S3_REGION")}

	// Per-tenant overrides win when any are present; otherwise we fall back to the legacy DSN
	// (a single tenant cluster keeps working unchanged).
	name2dsn, name2initRG := ScanTenantEnv(os.Environ())
	require := strings.EqualFold(os.Getenv("LAKEHOUSE_STARROCKS_REQUIRE_TENANT_DSN"), "true")
	if len(name2dsn) == 0 {
		dsn := os.Getenv("LAKEHOUSE_STARROCKS_DSN")
		if dsn == "" {
			return nil, notConfigured("StarRocks", "LAKEHOUSE_STARROCKS_DSN")
		}
		db, err := OpenStarRocksDSN(dsn)
		if err != nil {
			return nil, err
		}
		return NewStarRocks(db, cfg)
	}
	return newTenantRouter(cfg, name2dsn, name2initRG, os.Getenv("LAKEHOUSE_STARROCKS_DSN"), require, defaultLogger{})
}

// TenantEnvKey is the canonical env-var suffix for a tenant name. Tenant names that don't
// slug to a valid env-var key (spaces, hyphens) are mapped here so the router can look up
// LAKEHOUSE_STARROCKS_DSN_TENANT_<KEY> reliably. ASCII letters and digits are kept verbatim;
// every run of any other character collapses to a single underscore; leading and trailing
// underscores are stripped.
//
//	"Demo Tenant CRD Bakeoff" → DEMO_TENANT_CRD_BAKEOFF
//	"northwinds-prod"         → NORTHWINDS_PROD
//	"  tenant-A  "            → TENANT_A
//
// Exported so the status panel can ask "what wiring has this process discovered for tenant
// X?" using the same slug the router uses on every connect.
func TenantEnvKey(name string) string {
	if name == "" {
		return ""
	}
	var b strings.Builder
	lastUnderscore := true // suppresses a leading underscore
	for _, r := range strings.ToUpper(name) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		case !lastUnderscore:
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.TrimRight(b.String(), "_")
}

const (
	dsnTenantPrefix        = "LAKEHOUSE_STARROCKS_DSN_TENANT_"
	initRGEnvPrefix        = "LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP_TENANT_"
	tenantMaxOpenConns     = 10               // matches the resource group's concurrency_limit
	tenantConnMaxLifetime  = 5 * time.Minute  // rotates stale resource-group bindings after classifier edits
	tenantConnMaxIdleTime  = 60 * time.Second // releases idle conns so 12.5 GB node doesn't sit on dead pool capacity
)

// ScanTenantEnv returns per-tenant DSN and init-resource-group maps keyed by tenant slug (via
// `TenantEnvKey` at lookup time). It does not validate DSNs or resource-group names; the router
// does that on first use so a typo in one tenant doesn't prevent the cluster from booting.
//
// Init-resource-group env vars mirror the DSN convention: one per tenant, slug-suffixed.
// A global opt-in was tried first but it OVERRIDES classifier routing (the SET pins the
// connection's group, the classifier no longer gets to choose) — which means a single
// shared value actively defeats per-tenant isolation. The per-tenant shape is the only one
// that composes safely with the classifier system.
//
// Exported so the status service can read the same wiring the router uses, without
// duplicating the prefix-matching logic.
func ScanTenantEnv(env []string) (name2dsn, name2initRG map[string]string) {
	name2dsn = make(map[string]string)
	name2initRG = make(map[string]string)
	for _, kv := range env {
		eq := strings.IndexByte(kv, '=')
		if eq <= 0 {
			continue
		}
		k, v := kv[:eq], kv[eq+1:]
		switch {
		case strings.HasPrefix(k, dsnTenantPrefix):
			name2dsn[strings.ToUpper(k[len(dsnTenantPrefix):])] = v
		case strings.HasPrefix(k, initRGEnvPrefix):
			name2initRG[strings.ToUpper(k[len(initRGEnvPrefix):])] = v
		}
	}
	return name2dsn, name2initRG
}

type lazyDestination struct {
	mu    sync.Mutex
	build func() (AuditDestination, error)
	inner AuditDestination
}

func (l *lazyDestination) get() (AuditDestination, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.inner != nil {
		return l.inner, nil
	}
	d, err := l.build()
	if err != nil {
		return nil, err
	}
	l.inner = d
	return d, nil
}

func (l *lazyDestination) EnsureAuditDestination(ctx context.Context, spec AuditDestinationSpec) error {
	d, err := l.get()
	if err != nil {
		return err
	}
	return d.EnsureAuditDestination(ctx, spec)
}

func (l *lazyDestination) MaxAuditID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	d, err := l.get()
	if err != nil {
		return 0, err
	}
	return d.MaxAuditID(ctx, tenantID)
}

func (l *lazyDestination) AuditHash(ctx context.Context, tenantID uuid.UUID, id int64) (string, bool, error) {
	d, err := l.get()
	if err != nil {
		return "", false, err
	}
	return d.AuditHash(ctx, tenantID, id)
}

func (l *lazyDestination) AppendAudit(ctx context.Context, tenantID uuid.UUID, rows []AuditRow) error {
	d, err := l.get()
	if err != nil {
		return err
	}
	return d.AppendAudit(ctx, tenantID, rows)
}

func (l *lazyDestination) AuditRange(ctx context.Context, tenantID uuid.UUID, afterID int64, limit int) ([]AuditRow, error) {
	d, err := l.get()
	if err != nil {
		return nil, err
	}
	return d.AuditRange(ctx, tenantID, afterID, limit)
}

// ---- per-tenant router ----

// tenantRouter fans audit-copy traffic across per-tenant StarRocks destinations. Per-tenant
// DSNs and resource groups are configured via env vars; an unmapped tenant falls back to the
// legacy single-DSN with a WARN log unless LAKEHOUSE_STARROCKS_REQUIRE_TENANT_DSN is set,
// in which case the call errors. TenantName (in AuditDestinationSpec) is the lookup key;
// calls that don't pass a name (MaxAuditID, etc.) require EnsureAuditDestination to have
// been called for that tenant first so the id→name mapping is known.
type tenantRouter struct {
	cfg        StarRocksConfig
	name2dsn   map[string]string
	name2initRG map[string]string // slug → resource-group name; empty/absent → no init SQL for that tenant
	require    bool

	// legacy is built lazily from LAKEHOUSE_STARROCKS_DSN and used as the fallback. nil if
	// that env var is empty; in that case unmapped tenants always error.
	legacyOnce sync.Once
	legacyErr  error
	legacy     AuditDestination
	legacyDSN  string

	mu      sync.Mutex
	id2name map[uuid.UUID]string
	cache   map[string]AuditDestination // tenant name → built (production uses *StarRocks; tests stub)
	warned  map[string]struct{}        // tenant names we've already WARN-logged for the unmapped path

	openDB func(dsn string, initSQL []string) (*sql.DB, error)
	newSR  func(*sql.DB, StarRocksConfig) (AuditDestination, error)
	logf   func(format string, args ...any)
}

type defaultLogger struct{}

func (defaultLogger) Printf(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...) }

func newTenantRouter(cfg StarRocksConfig, name2dsn, name2initRG map[string]string, legacyDSN string, require bool, logf interface {
	Printf(format string, args ...any)
}) (*tenantRouter, error) {
	if name2dsn == nil || logf == nil {
		return nil, errors.New("name2dsn and a logger are required")
	}
	if name2initRG == nil {
		name2initRG = map[string]string{}
	}
	r := &tenantRouter{
		cfg:        cfg,
		name2dsn:   name2dsn,
		name2initRG: name2initRG,
		require:    require,
		legacyDSN:  legacyDSN,
		id2name:    make(map[uuid.UUID]string),
		cache:      make(map[string]AuditDestination),
		warned:     make(map[string]struct{}),
		openDB:     func(dsn string, initSQL []string) (*sql.DB, error) { return OpenStarRocksDSN(dsn, initSQL...) },
		newSR:      func(db *sql.DB, cfg StarRocksConfig) (AuditDestination, error) { return NewStarRocks(db, cfg) },
	}
	if l, ok := logf.(interface {
		Printf(format string, args ...any)
	}); ok {
		r.logf = l.Printf
	} else {
		r.logf = defaultLogger{}.Printf
	}
	for name := range name2dsn {
		if rg := name2initRG[name]; rg != "" {
			r.logf("StarRocks tenant %q → DSN configured; init SQL: SET resource_group = %q", name, rg)
		} else {
			r.logf("StarRocks tenant %q → DSN configured; no init SQL (classifier routes)", name)
		}
	}
	if require {
		r.logf("StarRocks LAKEHOUSE_STARROCKS_REQUIRE_TENANT_DSN=true: unmapped tenants will error")
	} else if legacyDSN != "" {
		r.logf("StarRocks legacy DSN is set; unmapped tenants will fall back with a WARN")
	} else {
		r.logf("StarRocks has no legacy DSN; unmapped tenants will error")
	}
	return r, nil
}

// forTenantByName returns the destination for the given tenant name (the lookup key). If the
// name has a per-tenant override the destination is built lazily and cached; otherwise it
// falls back to the legacy DSN (with a WARN) or errors. Cache keys are slug-normalized so
// "northwinds", "Northwinds", and "NORTHWINDS" share one entry.
func (r *tenantRouter) forTenantByName(name string) (AuditDestination, error) {
	if name == "" {
		// No name on the call: caller did not enforce tenant. Use legacy, but only if it
		// exists — silently failing here is worse than erroring.
		return r.getLegacy(true)
	}
	key := TenantEnvKey(name)
	r.mu.Lock()
	if d, ok := r.cache[key]; ok {
		r.mu.Unlock()
		return d, nil
	}
	r.mu.Unlock()

	dsn, hasOverride := r.name2dsn[key]
	if !hasOverride {
		if r.require {
			envName := dsnTenantPrefix + key
			return nil, fmt.Errorf("no per-tenant StarRocks DSN configured for tenant %q (looked for %s); LAKEHOUSE_STARROCKS_REQUIRE_TENANT_DSN=true refuses the legacy fallback", name, envName)
		}
		r.mu.Lock()
		_, already := r.warned[name]
		r.warned[name] = struct{}{}
		r.mu.Unlock()
		if !already {
			r.logf("WARN: tenant %q has no per-tenant StarRocks DSN; falling back to the legacy LAKEHOUSE_STARROCKS_DSN — set %s%s to silence this", name, dsnTenantPrefix, key)
		}
		return r.getLegacy(true)
	}
	initSQL := []string{}
	if rg := r.name2initRG[key]; rg != "" {
		// 3.3 form: `SET resource_group = '<rg>'`. The classifier pins this. The SET
		// is belt-and-braces for 3.3; do not set it on 4.1 (the variable name may have
		// changed) — leaving it absent keeps `initConnector.Connect` succeeding.
		initSQL = append(initSQL, "SET resource_group = '"+rg+"'")
	}
	db, err := r.openDB(dsn, initSQL)
	if err != nil {
		return nil, fmt.Errorf("open StarRocks DSN for tenant %q: %w", name, err)
	}
	sr, err := r.newSR(db, r.cfg)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("wrap StarRocks for tenant %q: %w", name, err)
	}
	db.SetMaxOpenConns(tenantMaxOpenConns)
	db.SetConnMaxLifetime(tenantConnMaxLifetime)
	db.SetConnMaxIdleTime(tenantConnMaxIdleTime)
	r.mu.Lock()
	r.cache[key] = sr
	r.mu.Unlock()
	return sr, nil
}

// forTenantByID resolves a tenant ID to its name (recorded by EnsureAuditDestination) and
// delegates. If the name has never been recorded, the call errors — the caller forgot to
// call EnsureAuditDestination first. The legacy fallback handles the case where the tenant
// is configured before the router sees the ID, but never for a name we have not seen.
func (r *tenantRouter) forTenantByID(id uuid.UUID) (AuditDestination, error) {
	r.mu.Lock()
	name, ok := r.id2name[id]
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no tenant name recorded for id %s; EnsureAuditDestination must be called first", id)
	}
	return r.forTenantByName(name)
}

// getLegacy builds and caches the legacy single-DSN destination. Called from the unmapped
// path. logWhenUnknown controls whether to emit the explanatory WARN the first time it's
// used for a tenant that has no override (the forTenantByName branch logs before this is
// reached, so this method only logs in the no-name-on-the-call branch).
func (r *tenantRouter) getLegacy(logWhenUnknown bool) (AuditDestination, error) {
	r.legacyOnce.Do(func() {
		if r.legacyDSN == "" {
			r.legacyErr = notConfigured("StarRocks", "LAKEHOUSE_STARROCKS_DSN")
			return
		}
		db, err := r.openDB(r.legacyDSN, nil)
		if err != nil {
			r.legacyErr = fmt.Errorf("open legacy StarRocks DSN: %w", err)
			return
		}
		sr, err := r.newSR(db, r.cfg)
		if err != nil {
			_ = db.Close()
			r.legacyErr = fmt.Errorf("wrap legacy StarRocks: %w", err)
			return
		}
		r.legacy = sr
		if logWhenUnknown {
			r.logf("WARN: using legacy LAKEHOUSE_STARROCKS_DSN for a call that did not supply a tenant name")
		}
	})
	if r.legacyErr != nil {
		return nil, r.legacyErr
	}
	return r.legacy, nil
}

func (r *tenantRouter) EnsureAuditDestination(ctx context.Context, spec AuditDestinationSpec) error {
	d, err := r.forTenantByName(spec.TenantName)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.id2name[spec.TenantID] = spec.TenantName
	r.mu.Unlock()
	return d.EnsureAuditDestination(ctx, spec)
}

func (r *tenantRouter) MaxAuditID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	d, err := r.forTenantByID(tenantID)
	if err != nil {
		return 0, err
	}
	return d.MaxAuditID(ctx, tenantID)
}

func (r *tenantRouter) AuditHash(ctx context.Context, tenantID uuid.UUID, id int64) (string, bool, error) {
	d, err := r.forTenantByID(tenantID)
	if err != nil {
		return "", false, err
	}
	return d.AuditHash(ctx, tenantID, id)
}

func (r *tenantRouter) AppendAudit(ctx context.Context, tenantID uuid.UUID, rows []AuditRow) error {
	d, err := r.forTenantByID(tenantID)
	if err != nil {
		return err
	}
	return d.AppendAudit(ctx, tenantID, rows)
}

func (r *tenantRouter) AuditRange(ctx context.Context, tenantID uuid.UUID, afterID int64, limit int) ([]AuditRow, error) {
	d, err := r.forTenantByID(tenantID)
	if err != nil {
		return nil, err
	}
	return d.AuditRange(ctx, tenantID, afterID, limit)
}

// asInt reads a positive integer from an env var, returning 0 if unset or unparseable.
func asInt(s string) int {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}
