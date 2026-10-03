package infra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
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
type AuditDestinationSpec struct {
	TenantID      uuid.UUID
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
// interpolation.
func OpenStarRocksDSN(dsn string) (*sql.DB, error) {
	c, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, errors.New("the StarRocks DSN is not valid") // never echo it: it holds a password
	}
	c.InterpolateParams = true
	return sql.Open("mysql", c.FormatDSN())
}

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

// ---- configuration from the environment ----

// AuditDestinationFromEnv returns the StarRocks-backed destination, or one that reports
// ErrNotConfigured. It is built on first use: opening the connection must never slow or break
// worker startup.
//
//	LAKEHOUSE_STARROCKS_DSN          user:password@tcp(host:9030)/
//	LAKEHOUSE_STARROCKS_CATALOG_URI  Lakekeeper's Iceberg REST URL as StarRocks reaches it
//	LAKEHOUSE_STARROCKS_S3_ENDPOINT  the object store as StarRocks' backends reach it
//	LAKEHOUSE_STARROCKS_S3_REGION    optional
func AuditDestinationFromEnv() AuditDestination {
	return &lazyDestination{build: func() (AuditDestination, error) { return destinationBuilder() }}
}

// destinationBuilder is a variable so a test can count constructions.
var destinationBuilder = buildDestinationFromEnv

func buildDestinationFromEnv() (AuditDestination, error) {
	dsn := os.Getenv("LAKEHOUSE_STARROCKS_DSN")
	uri := os.Getenv("LAKEHOUSE_STARROCKS_CATALOG_URI")
	endpoint := os.Getenv("LAKEHOUSE_STARROCKS_S3_ENDPOINT")
	if dsn == "" || uri == "" || endpoint == "" {
		return nil, notConfigured("StarRocks", "LAKEHOUSE_STARROCKS_DSN", "LAKEHOUSE_STARROCKS_CATALOG_URI", "LAKEHOUSE_STARROCKS_S3_ENDPOINT")
	}
	db, err := OpenStarRocksDSN(dsn)
	if err != nil {
		return nil, err
	}
	return NewStarRocks(db, StarRocksConfig{CatalogURI: uri, S3Endpoint: endpoint, S3Region: os.Getenv("LAKEHOUSE_STARROCKS_S3_REGION")})
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
