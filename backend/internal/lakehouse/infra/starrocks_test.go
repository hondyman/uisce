package infra

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var testTenant = uuid.MustParse("11111111-2222-3333-4444-555555555555")

const testCat = "ivy_t_11111111222233334444555555555555"

// recorder captures every statement the adapter sends, and matches any.
type recorder struct{ stmts []string }

func (r *recorder) match(_, actual string) error { r.stmts = append(r.stmts, actual); return nil }

func newSR(t *testing.T, cfg StarRocksConfig) (*StarRocks, sqlmock.Sqlmock, *recorder) {
	t.Helper()
	rec := &recorder{}
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(rec.match)))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	if cfg.CatalogURI == "" {
		cfg = StarRocksConfig{CatalogURI: "http://lakekeeper:8181/catalog", S3Endpoint: "http://minio:9000"}
	}
	sr, err := NewStarRocks(db, cfg)
	require.NoError(t, err)
	return sr, mock, rec
}

func spec() AuditDestinationSpec {
	return AuditDestinationSpec{TenantID: testTenant, WarehouseName: "ivy-t-11111111222233334444555555555555", AccessKeyID: "IVYACCESSKEY0123456", SecretKey: "s3cr3tSECRETkey0123456789"}
}

func TestCatalogName(t *testing.T) {
	n, err := CatalogName(testTenant)
	require.NoError(t, err)
	require.Equal(t, testCat, n)
	require.True(t, catalogNameRE.MatchString(n))
	_, err = CatalogName(uuid.Nil)
	require.Error(t, err)
}

func TestEnsureAuditDestination_ExactDDLInOrder(t *testing.T) {
	sr, mock, rec := newSR(t, StarRocksConfig{})
	for i := 0; i < 3; i++ {
		mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 0))
	}
	require.NoError(t, sr.EnsureAuditDestination(context.Background(), spec()))
	require.NoError(t, mock.ExpectationsWereMet())

	require.Equal(t, []string{
		`CREATE EXTERNAL CATALOG IF NOT EXISTS ` + testCat + ` PROPERTIES ("type" = "iceberg", "iceberg.catalog.type" = "rest", ` +
			`"iceberg.catalog.uri" = "http://lakekeeper:8181/catalog", "iceberg.catalog.warehouse" = "ivy-t-11111111222233334444555555555555", ` +
			`"aws.s3.endpoint" = "http://minio:9000", "aws.s3.access_key" = "IVYACCESSKEY0123456", "aws.s3.secret_key" = "s3cr3tSECRETkey0123456789", ` +
			`"aws.s3.enable_path_style_access" = "true")`,
		`CREATE DATABASE IF NOT EXISTS ` + testCat + `.audit`,
		"CREATE TABLE IF NOT EXISTS " + testCat + ".audit.lakehouse_events (`id` BIGINT, `tenant_id` STRING, `at` DATETIME, `actor_id` STRING, " +
			"`actor_role` STRING, `action` STRING, `before_json` STRING, `after_json` STRING, `prev_hash` STRING, `hash` STRING) " +
			`PROPERTIES ("file_format" = "parquet")`,
	}, rec.stmts, "catalog first, then database, then table; every statement IF NOT EXISTS so a re-run is safe")
}

func TestEnsureAuditDestination_OptionalRegion(t *testing.T) {
	sr, mock, rec := newSR(t, StarRocksConfig{CatalogURI: "http://l:8181/catalog", S3Endpoint: "http://m:9000", S3Region: "us-east-1"})
	for i := 0; i < 3; i++ {
		mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 0))
	}
	require.NoError(t, sr.EnsureAuditDestination(context.Background(), spec()))
	require.Contains(t, rec.stmts[0], `"aws.s3.region" = "us-east-1"`)
}

// A value that could end the string literal or the statement must be refused before ANY
// statement is sent. Validation, not escaping.
func TestEnsureAuditDestination_RefusesUnsafeValuesBeforeSendingAnything(t *testing.T) {
	hostile := []string{
		`x"; DROP CATALOG other; --`, `a'b`, `a\b`, "a b", "a\nb", "a;b", `a"b`, "", "a`b", "a#b", "a/*b", "$(x)", "a,b",
	}
	for _, field := range []string{"warehouse", "access", "secret"} {
		for _, bad := range hostile {
			sr, mock, rec := newSR(t, StarRocksConfig{})
			// Accept any statement, so that if the adapter DID send one it would be recorded and
			// the assertion below could fail. Without an expectation the mock rejects the call
			// without recording it, and "nothing was sent" would pass vacuously.
			for i := 0; i < 3; i++ {
				mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 0))
			}
			s := spec()
			switch field {
			case "warehouse":
				s.WarehouseName = bad
			case "access":
				s.AccessKeyID = bad
			case "secret":
				s.SecretKey = bad
			}
			err := sr.EnsureAuditDestination(context.Background(), s)
			require.Error(t, err, "%s=%q", field, bad)
			require.Empty(t, rec.stmts, "%s=%q must not reach the server", field, bad)
		}
	}
}

func TestEnsureAuditDestination_ErrorsNeverEchoTheCredential(t *testing.T) {
	sr, mock, _ := newSR(t, StarRocksConfig{})
	s := spec()
	// A MySQL error can quote the statement; here it quotes the secret and the access key.
	mock.ExpectExec("").WillReturnError(errors.New(`syntax error near '"aws.s3.access_key" = "` + s.AccessKeyID + `", "aws.s3.secret_key" = "` + s.SecretKey + `"'`))
	err := sr.EnsureAuditDestination(context.Background(), s)
	require.Error(t, err)
	require.NotContains(t, err.Error(), s.SecretKey)
	require.NotContains(t, err.Error(), s.AccessKeyID)
	require.Contains(t, err.Error(), "***")
	require.Contains(t, err.Error(), "step 1 of 3", "and it still says where it failed")
}

func TestEnsureAuditDestination_StopsAtTheFirstFailure(t *testing.T) {
	sr, mock, rec := newSR(t, StarRocksConfig{})
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("").WillReturnError(errors.New("catalog not found"))
	require.ErrorContains(t, sr.EnsureAuditDestination(context.Background(), spec()), "step 2 of 3")
	require.Len(t, rec.stmts, 2, "the table is not created in a catalog whose database could not be")
}

func TestNewStarRocks_ValidatesItsConfiguration(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	_, err := NewStarRocks(nil, StarRocksConfig{CatalogURI: "http://a", S3Endpoint: "http://b"})
	require.Error(t, err)
	for _, cfg := range []StarRocksConfig{
		{}, {CatalogURI: "http://a"}, {S3Endpoint: "http://b"},
		{CatalogURI: `http://a"; DROP`, S3Endpoint: "http://b"},
		{CatalogURI: "http://a", S3Endpoint: "http://b c"},
		{CatalogURI: "http://a", S3Endpoint: "http://b", S3Region: `r"`},
	} {
		_, err := NewStarRocks(db, cfg)
		require.Error(t, err, "%+v", cfg)
	}
}

func TestMaxAuditID(t *testing.T) {
	ctx := context.Background()
	sr, mock, rec := newSR(t, StarRocksConfig{})
	mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"m"}).AddRow(int64(42)))
	got, err := sr.MaxAuditID(ctx, testTenant)
	require.NoError(t, err)
	require.EqualValues(t, 42, got)
	require.Equal(t, "SELECT MAX(`id`) FROM "+testCat+".audit.lakehouse_events", rec.stmts[0], "reads the TENANT's own table")

	mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"m"}).AddRow(nil))
	got, err = sr.MaxAuditID(ctx, testTenant)
	require.NoError(t, err)
	require.Zero(t, got, "an empty copy resumes from the start")

	mock.ExpectQuery("").WillReturnError(errors.New("starrocks down"))
	_, err = sr.MaxAuditID(ctx, testTenant)
	require.ErrorContains(t, err, "starrocks down")

	_, err = sr.MaxAuditID(ctx, uuid.Nil)
	require.Error(t, err)
}

func TestAuditHash(t *testing.T) {
	ctx := context.Background()
	sr, mock, rec := newSR(t, StarRocksConfig{})
	mock.ExpectQuery("").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"h"}).AddRow("abc"))
	h, ok, err := sr.AuditHash(ctx, testTenant, 7)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "abc", h)
	require.Equal(t, "SELECT `hash` FROM "+testCat+".audit.lakehouse_events WHERE `id` = ?", rec.stmts[0], "the id is a bound parameter")

	mock.ExpectQuery("").WithArgs(int64(8)).WillReturnError(sql.ErrNoRows)
	_, ok, err = sr.AuditHash(ctx, testTenant, 8)
	require.NoError(t, err)
	require.False(t, ok, "absent is not an error")

	mock.ExpectQuery("").WithArgs(int64(9)).WillReturnError(errors.New("boom"))
	_, _, err = sr.AuditHash(ctx, testTenant, 9)
	require.Error(t, err)
}

func row(id int64, prev, hash string) AuditRow {
	return AuditRow{ID: id, At: time.Date(2026, 10, 3, 4, 5, 6, 789000000, time.FixedZone("EST", -5*3600)), ActorID: "alice", ActorRole: "global_admin",
		Action: "configured", BeforeJSON: "", AfterJSON: `{"audit_retention_days":365}`, PrevHash: prev, Hash: hash}
}

func TestAppendAudit_OneStatementBoundParametersUTC(t *testing.T) {
	sr, mock, rec := newSR(t, StarRocksConfig{})
	rows := []AuditRow{row(5, "p0", "h5"), row(6, "h5", "h6")}
	args := []driver.Value{}
	for _, r := range rows {
		args = append(args, r.ID, testTenant.String(), "2026-10-03 09:05:06.789000", "alice", "global_admin", "configured", "", `{"audit_retention_days":365}`, r.PrevHash, r.Hash)
	}
	mock.ExpectExec("").WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 2))
	require.NoError(t, sr.AppendAudit(context.Background(), testTenant, rows))
	require.NoError(t, mock.ExpectationsWereMet(), "args in column order, and the time converted to UTC")

	require.Len(t, rec.stmts, 1, "ONE statement, so one Iceberg commit")
	require.Equal(t, "INSERT INTO "+testCat+".audit.lakehouse_events (`id`, `tenant_id`, `at`, `actor_id`, `actor_role`, `action`, `before_json`, `after_json`, `prev_hash`, `hash`) VALUES "+
		"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", rec.stmts[0])
}

// Audit content is attacker-influenced text (an actor name, an error message). It must travel
// only as a bound argument, never inside the SQL text.
func TestAppendAudit_HostileContentIsNeverInTheSQL(t *testing.T) {
	sr, mock, rec := newSR(t, StarRocksConfig{})
	evil := `'); DROP TABLE ` + testCat + `.audit.lakehouse_events; --`
	r := row(1, "p", "h")
	r.ActorID, r.AfterJSON = evil, `{"error":"`+evil+`"}`
	mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, sr.AppendAudit(context.Background(), testTenant, []AuditRow{r}))
	require.NotContains(t, rec.stmts[0], "DROP", "the statement text is fixed; only placeholders vary")
	require.Equal(t, 10, strings.Count(rec.stmts[0], "?"))
}

func TestAppendAudit_RefusesBadBatchesBeforeSendingAnything(t *testing.T) {
	sr, mock, rec := newSR(t, StarRocksConfig{})
	// Permissive expectations, so a statement that WAS sent would be recorded (see the note in
	// the unsafe-values test): without them the mock rejects silently and this passes vacuously.
	for i := 0; i < 8; i++ {
		mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(0, 0))
	}
	ctx := context.Background()
	require.Error(t, sr.AppendAudit(ctx, testTenant, nil))
	require.Error(t, sr.AppendAudit(ctx, testTenant, []AuditRow{row(2, "a", "b"), row(2, "b", "c")}), "equal ids")
	require.Error(t, sr.AppendAudit(ctx, testTenant, []AuditRow{row(3, "a", "b"), row(2, "b", "c")}), "descending ids")
	big := make([]AuditRow, MaxAppendRows+1)
	for i := range big {
		big[i] = row(int64(i+1), "p", "h")
	}
	require.Error(t, sr.AppendAudit(ctx, testTenant, big))
	require.Error(t, sr.AppendAudit(ctx, uuid.Nil, []AuditRow{row(1, "p", "h")}))
	require.Empty(t, rec.stmts)
}

func TestAppendAudit_FailureIsReturned(t *testing.T) {
	sr, mock, _ := newSR(t, StarRocksConfig{})
	mock.ExpectExec("").WillReturnError(errors.New("commit conflict"))
	require.ErrorContains(t, sr.AppendAudit(context.Background(), testTenant, []AuditRow{row(1, "p", "h")}), "commit conflict")
}

func TestOpenStarRocksDSN(t *testing.T) {
	_, err := OpenStarRocksDSN("not a dsn \x00 secretpassword")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "secretpassword", "the DSN holds a password and is never echoed")

	db, err := OpenStarRocksDSN("root:pw@tcp(127.0.0.1:9030)/")
	require.NoError(t, err)
	require.NoError(t, db.Close())
}

func TestAuditDestinationFromEnv_UnconfiguredAndLazy(t *testing.T) {
	for _, v := range []string{"LAKEHOUSE_STARROCKS_DSN", "LAKEHOUSE_STARROCKS_CATALOG_URI", "LAKEHOUSE_STARROCKS_S3_ENDPOINT"} {
		t.Setenv(v, "")
	}
	ctx := context.Background()
	d := AuditDestinationFromEnv()
	require.ErrorIs(t, d.EnsureAuditDestination(ctx, spec()), ErrNotConfigured)
	_, err := d.MaxAuditID(ctx, testTenant)
	require.ErrorIs(t, err, ErrNotConfigured)
	_, _, err = d.AuditHash(ctx, testTenant, 1)
	require.ErrorIs(t, err, ErrNotConfigured)
	require.ErrorIs(t, d.AppendAudit(ctx, testTenant, []AuditRow{row(1, "p", "h")}), ErrNotConfigured)
	require.ErrorContains(t, d.EnsureAuditDestination(ctx, spec()), "LAKEHOUSE_STARROCKS_DSN", "and says what is missing")

	var builds int
	orig := destinationBuilder
	t.Cleanup(func() { destinationBuilder = orig })
	sr, _, _ := newSR(t, StarRocksConfig{})
	destinationBuilder = func() (AuditDestination, error) { builds++; return sr, nil }
	lazy := AuditDestinationFromEnv()
	require.Zero(t, builds, "creating it at worker start must open nothing")
	_, err = lazy.MaxAuditID(ctx, testTenant) // no expectation set: fails at the mock, after building
	require.Error(t, err)
	_, _ = lazy.MaxAuditID(ctx, testTenant)
	require.Equal(t, 1, builds, "a successful build is cached")
}

func TestAuditRange_ReadsInOrderWithBoundParametersAndEitherTimestampForm(t *testing.T) {
	ctx := context.Background()
	sr, mock, rec := newSR(t, StarRocksConfig{})
	cols := []string{"id", "at", "actor_id", "actor_role", "action", "before_json", "after_json", "prev_hash", "hash"}
	when := time.Date(2026, 10, 3, 4, 5, 6, 789000000, time.UTC)
	mock.ExpectQuery("").WithArgs(int64(4), 2).WillReturnRows(sqlmock.NewRows(cols).
		AddRow(int64(5), when, "alice", "global_admin", "configured", nil, `{"a":1}`, "p", "h5").
		AddRow(int64(6), []byte("2026-10-03 04:05:06.789"), "bob", "r", "x", "", "", "h5", "h6"))
	got, err := sr.AuditRange(ctx, testTenant, 4, 2)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, when, got[0].At)
	require.Equal(t, when, got[1].At, "text and time forms read the same instant")
	require.Equal(t, "", got[0].BeforeJSON, "a NULL column is empty text")
	require.Equal(t, "SELECT `id`, `at`, `actor_id`, `actor_role`, `action`, `before_json`, `after_json`, `prev_hash`, `hash` FROM "+
		testCat+".audit.lakehouse_events WHERE `id` > ? ORDER BY `id` ASC LIMIT ?", rec.stmts[0], "bound parameters and chain order")
}

func TestAuditRange_RefusesWhatItCannotReadExactly(t *testing.T) {
	ctx := context.Background()
	sr, mock, _ := newSR(t, StarRocksConfig{})
	_, err := sr.AuditRange(ctx, testTenant, 0, 0)
	require.Error(t, err)
	_, err = sr.AuditRange(ctx, testTenant, 0, MaxAppendRows+1)
	require.Error(t, err)
	_, err = sr.AuditRange(ctx, uuid.Nil, 0, 10)
	require.Error(t, err)

	cols := []string{"id", "at", "actor_id", "actor_role", "action", "before_json", "after_json", "prev_hash", "hash"}
	mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(cols).AddRow(int64(1), "garbage", "a", "r", "x", "", "", "p", "h"))
	_, err = sr.AuditRange(ctx, testTenant, 0, 10)
	require.ErrorContains(t, err, "unreadable timestamp", "a guessed timestamp could call a changed row unchanged")

	mock.ExpectQuery("").WillReturnError(errors.New("boom"))
	_, err = sr.AuditRange(ctx, testTenant, 0, 10)
	require.Error(t, err)
}
