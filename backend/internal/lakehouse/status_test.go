package lakehouse

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	"github.com/stretchr/testify/require"
)

// stubLister satisfies TenantLister for tests. list returns the configured items; get
// looks them up by id.
type stubLister struct {
	items []registry.Config
}

func (s *stubLister) List(_ context.Context, _ string, _ int, _ int) ([]registry.Config, int, error) {
	return s.items, len(s.items), nil
}

func (s *stubLister) Get(_ context.Context, id uuid.UUID) (*registry.Config, error) {
	for i := range s.items {
		if s.items[i].TenantID == id.String() {
			return &s.items[i], nil
		}
	}
	return nil, registry.ErrTenantNotFound
}

// fixedClock returns a stable time for the generated_at field.
func fixedClock() time.Time { return time.Date(2026, 10, 7, 14, 22, 1, 0, time.UTC) }

func TestStatusService_PlatformStatus_Shape(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// SHOW FRONTENDS: 1 FE alive with version.
	feCols := []string{"Name", "Host", "Alive", "Version"}
	feVals := []driverValue{"fe", "10.0.0.5", "true", "3.3.22-753696f"}
	mock.ExpectQuery(regexp.QuoteMeta("SHOW FRONTENDS")).
		WillReturnRows(sqlmock.NewRows(feCols).AddRow(feVals...))

	// SHOW BACKENDS: 1 BE alive.
	beCols := []string{"Host", "Alive", "TotalCapacityB", "TotalUsedProportion"}
	beVals := []driverValue{"10.0.0.6", "true", int64(1759218604441), 0.0001}
	mock.ExpectQuery(regexp.QuoteMeta("SHOW BACKENDS")).
		WillReturnRows(sqlmock.NewRows(beCols).AddRow(beVals...))

	// SHOW RESOURCE GROUPS ALL (3.3 form: probe picks the working one).
	mock.ExpectQuery(regexp.QuoteMeta("SHOW WORKLOAD GROUPS ALL")).
		WillReturnError(errShowSyntax) // 4.1 not available — fall back to 3.3
	rgCols := []string{"name", "cpu_weight", "mem_limit", "concurrency_limit", "big_query_mem_limit", "classifiers"}
	rgVals := []driverValue{
		"tenant_northwinds_wg",
		"4",
		"35.0%",
		"10",
		"2147483648",
		"user='tenant_northwinds_svc', weight=1.0",
	}
	// The probe consumes one call; resourceGroups then issues a second call with the
	// same statement. sqlmock matches in order, so both need their own ExpectQuery.
	mock.ExpectQuery(regexp.QuoteMeta("SHOW RESOURCE GROUPS ALL")).
		WillReturnRows(sqlmock.NewRows(rgCols).AddRow(rgVals...))
	mock.ExpectQuery(regexp.QuoteMeta("SHOW RESOURCE GROUPS ALL")).
		WillReturnRows(sqlmock.NewRows(rgCols).AddRow(rgVals...))

	// SHOW DATA FROM DATABASE for the tenant's lowercased slug. The DB doesn't exist
	// yet, so this returns no rows and the panel renders exists=false.
	mock.ExpectQuery(regexp.QuoteMeta("SHOW DATA FROM DATABASE `northwinds`")).
		WillReturnError(errShowSyntax)

	svc := NewStatusService(db, &stubLister{items: []registry.Config{
		{TenantID: uuid.New().String(), TenantName: "northwinds", Configured: true, Provisioned: true},
	}}, false)
	svc.Now = fixedClock

	out, err := svc.PlatformStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, fixedClock(), out.GeneratedAt)
	require.Len(t, out.Cluster.Frontends, 1)
	require.Equal(t, "fe", out.Cluster.Frontends[0].Name)
	require.True(t, out.Cluster.Frontends[0].Alive)
	require.Equal(t, "3.3.22-753696f", out.Cluster.Frontends[0].Version)
	require.Len(t, out.Cluster.Backends, 1)
	require.True(t, out.Cluster.Backends[0].Alive)
	require.Len(t, out.ResourceGroups, 1)
	require.Equal(t, "tenant_northwinds_wg", out.ResourceGroups[0].Name)
	require.NotNil(t, out.ResourceGroups[0].CPUWeight)
	require.Equal(t, 4, *out.ResourceGroups[0].CPUWeight)
	require.Equal(t, "35.0%", out.ResourceGroups[0].MemLimit)
	require.Len(t, out.ResourceGroups[0].Classifiers, 1)
	require.Equal(t, "tenant_northwinds_svc", out.ResourceGroups[0].Classifiers[0].User)
	require.Len(t, out.Tenants, 1)
	require.Equal(t, "northwinds", out.Tenants[0].Name)
	require.Equal(t, "NORTHWINDS", out.Tenants[0].Slug)
	// DB doesn't exist; expect a warning (registry says provisioned, DB doesn't exist).
	require.NotEmpty(t, out.Tenants[0].Warnings)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStatusService_TenantStatus_Scoped(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// The tenant handler does NOT issue SHOW commands (cluster + RGs are stripped).
	// SHOW DATA FROM DATABASE still runs because tenantBlock calls it.
	mock.ExpectQuery(regexp.QuoteMeta("SHOW DATA FROM DATABASE `northwinds`")).
		WillReturnError(errShowSyntax)

	id := uuid.New()
	svc := NewStatusService(db, &stubLister{items: []registry.Config{
		{TenantID: id.String(), TenantName: "northwinds", Configured: true},
	}}, false)
	svc.Now = fixedClock

	out, err := svc.TenantStatus(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, fixedClock(), out.GeneratedAt)
	require.Equal(t, "northwinds", out.Tenant.Name)
	require.Equal(t, "NORTHWINDS", out.Tenant.Slug)
	// Cluster + RGs are absent from the tenant-scoped payload — encoded by the absence
	// of fields, not as nulls.
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStatusService_TenantStatus_NotFound(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewStatusService(db, &stubLister{}, false)
	_, err = svc.TenantStatus(context.Background(), uuid.New())
	require.Error(t, err)
	var notFound *TenantNotFoundError
	require.ErrorAs(t, err, &notFound)
}

func TestStatusService_PartialFailureDoesNotPanic(t *testing.T) {
	// The panel must serve whatever it can when one tenant's SHOW DATA errors. Here we
	// hand the service a tenant whose DB lookup will fail, and a tenant whose DB lookup
	// returns empty rows — both must yield blocks without 500-ing.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Frontend query.
	mock.ExpectQuery(regexp.QuoteMeta("SHOW FRONTENDS")).
		WillReturnRows(sqlmock.NewRows([]string{"Name", "Alive"}).AddRow("fe", "true"))
	// Backend query.
	mock.ExpectQuery(regexp.QuoteMeta("SHOW BACKENDS")).
		WillReturnRows(sqlmock.NewRows([]string{"Alive"}))
	// RGs: probe fails both, panel serves empty with a note.
	mock.ExpectQuery(regexp.QuoteMeta("SHOW WORKLOAD GROUPS ALL")).
		WillReturnError(errShowSyntax)
	mock.ExpectQuery(regexp.QuoteMeta("SHOW RESOURCE GROUPS ALL")).
		WillReturnError(errShowSyntax)
	// Two backend queries: probe + actual. Probe fails both; actual fails the chosen
	// one too. The panel surfaces this as a note, not a 500.
	mock.ExpectQuery(regexp.QuoteMeta("SHOW RESOURCE GROUPS ALL")).
		WillReturnError(errShowSyntax)
	// DB lookup: error.
	mock.ExpectQuery(regexp.QuoteMeta("SHOW DATA FROM DATABASE `a`")).
		WillReturnError(errShowSyntax)

	svc := NewStatusService(db, &stubLister{items: []registry.Config{
		{TenantID: uuid.New().String(), TenantName: "a", Configured: true, Provisioned: true},
	}}, false)
	svc.Now = fixedClock

	out, err := svc.PlatformStatus(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, out.Notes, "notes captured the resource-groups failure")
	require.Len(t, out.Tenants, 1)
	require.NotEmpty(t, out.Tenants[0].Warnings, "tenant block captured the DB SHOW error")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestParseClassifiers_HandlesVariants(t *testing.T) {
	cases := []struct {
		in   string
		want []ResourceClassifier
	}{
		{"", nil},
		{"user='alice', weight=1.0", []ResourceClassifier{{User: "alice", Weight: 1.0}}},
		{"user='alice', weight=1.0; user='bob', weight=2.0", []ResourceClassifier{
			{User: "alice", Weight: 1.0}, {User: "bob", Weight: 2.0},
		}},
		{"role='analyst', weight=1.0", []ResourceClassifier{{Role: "analyst", Weight: 1.0}}},
		// Garbage: drop it rather than crash.
		{"!!! not parseable !!!", nil},
	}
	for _, c := range cases {
		got := parseClassifiers(c.in)
		require.Equal(t, c.want, got, "input %q", c.in)
	}
}

// errShowSyntax is the kind of error StarRocks returns when a SHOW statement's grammar
// doesn't exist in the running version. sqlmock returns it for the 4.1 fallback probe.
var errShowSyntax = &sqlMockError{msg: "syntax error"}

type sqlMockError struct{ msg string }

func (e *sqlMockError) Error() string { return e.msg }

// driverValue is a small helper to make sqlmock row construction readable in this file.
type driverValue = driver.Value