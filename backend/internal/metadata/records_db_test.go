package metadata

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"

	dbpkg "github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/sourceconn"
)

// fakeSources stands in for alpha: it answers the connector's datasource lookup.
type fakeSources struct {
	src      sourceconn.Source
	err      error
	tenant   string
	policy   sourceconn.Policy
	lookedUp int
}

func (f *fakeSources) Source(_ context.Context, tenantID, _ string, p sourceconn.Policy) (sourceconn.Source, error) {
	f.lookedUp++
	f.tenant, f.policy = tenantID, p
	return f.src, f.err
}

// connectorFor builds a source connector over a fake lookup and the test's credentials store.
func connectorFor(t *testing.T, reg sourceconn.Registry) *sourceconn.Connector {
	t.Helper()
	c, err := sourceconn.New(sourceconn.Config{
		Registry: reg, Credentials: sourceconn.DSCreds{R: datasourceCreds()}, CallerTenant: dbpkg.GetTenantIDFromCtx,
		MaxPools: 2, MaxConnsPerPool: 1, IdleTTL: time.Minute, DialTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestRecordsDBStrict(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	reg := &fakeSources{err: sourceconn.ErrNotFound}
	s := &BusinessObjectService{db: sqlx.NewDb(db, "sqlmock")}
	s.SetSourceConnector(connectorFor(t, reg))
	ctx := context.Background()

	// No bound backend: the records live in the metadata DB.
	mock.ExpectQuery("FROM public.business_object_binding").WillReturnError(sql.ErrNoRows)
	got, err := s.recordsDBStrict(ctx, "t-1", "bo-1")
	if err != nil || got != s.db {
		t.Fatalf("unbound BO: got %v, %v", got, err)
	}
	if reg.lookedUp != 0 {
		t.Fatal("an unbound BO must not touch the source connector")
	}

	// Bound to a datasource with no connection config: an error, never s.db.
	mock.ExpectQuery("FROM public.business_object_binding").
		WillReturnRows(sqlmock.NewRows([]string{"backend_id"}).AddRow("ds-9"))
	got, err = s.recordsDBStrict(ctx, "t-1", "bo-2")
	if err == nil || got != nil || !strings.Contains(err.Error(), "no connection configuration") {
		t.Fatalf("unresolvable datasource: got %v, %v", got, err)
	}
	if reg.tenant != "t-1" || reg.policy != sourceconn.OwnerOrGoldCopy {
		t.Fatalf("the backend must be looked up for the calling tenant with the gold-copy policy: %q %v", reg.tenant, reg.policy)
	}

	// Reads still degrade when the backend cannot be reached (unchanged behaviour); writes do not.
	mock.ExpectQuery("FROM public.business_object_binding").
		WillReturnRows(sqlmock.NewRows([]string{"backend_id"}).AddRow("ds-9"))
	d, err := s.resolveRecordsDB(ctx, "t-1", "bo-2")
	if err != nil || d != s.db {
		t.Error("reads keep their fallback for an unreachable or unconfigured backend")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

// An authorization failure is not an outage: it must never degrade to the metadata database.
func TestResolveRecordsDB_AnAuthorizationFailureNeverFallsBackToAlpha(t *testing.T) {
	for name, e := range map[string]error{"not allowed": sourceconn.ErrNotAllowed} {
		db, mock, _ := sqlmock.New()
		reg := &fakeSources{err: e}
		s := &BusinessObjectService{db: sqlx.NewDb(db, "sqlmock")}
		s.SetSourceConnector(connectorFor(t, reg))
		mock.ExpectQuery("FROM public.business_object_binding").
			WillReturnRows(sqlmock.NewRows([]string{"backend_id"}).AddRow("ds-of-another-tenant"))

		got, err := s.resolveRecordsDB(context.Background(), "t-1", "bo-1")
		if got != nil || !errors.Is(err, e) {
			t.Fatalf("%s: must be refused, not served the alpha DB: got %v, %v", name, got, err)
		}
		db.Close()
	}

	// No tenant at all is refused the same way.
	db, mock, _ := sqlmock.New()
	defer db.Close()
	s := &BusinessObjectService{db: sqlx.NewDb(db, "sqlmock")}
	s.SetSourceConnector(connectorFor(t, &fakeSources{}))
	mock.ExpectQuery("FROM public.business_object_binding").
		WillReturnRows(sqlmock.NewRows([]string{"backend_id"}).AddRow("ds-1"))
	if got, err := s.resolveRecordsDB(context.Background(), "", "bo-1"); got != nil || !errors.Is(err, sourceconn.ErrNoTenant) {
		t.Fatalf("no tenant: got %v, %v", got, err)
	}
}

func TestRecordsDBStrict_WithoutAConnectorABoundBackendFailsClosed(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	s := &BusinessObjectService{db: sqlx.NewDb(db, "sqlmock")}
	mock.ExpectQuery("FROM public.business_object_binding").
		WillReturnRows(sqlmock.NewRows([]string{"backend_id"}).AddRow("ds-1"))
	got, err := s.recordsDBStrict(context.Background(), "t-1", "bo-1")
	if got != nil || err == nil || !strings.Contains(err.Error(), "source access is not configured") {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestEnforceWriteBatch_RefusesUnresolvableDatasource(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	s := &BusinessObjectService{db: sqlx.NewDb(db, "sqlmock")}
	s.SetSourceConnector(connectorFor(t, &fakeSources{err: sourceconn.ErrNotFound}))
	mock.ExpectQuery("FROM public.business_objects bo").
		WillReturnRows(sqlmock.NewRows([]string{"id", "bo_key", "driver_table_name"}).AddRow("bo-2", "security", "/orm/security"))
	mock.ExpectQuery("FROM public.business_object_binding").
		WillReturnRows(sqlmock.NewRows([]string{"backend_id"}).AddRow("ds-9"))
	wrote := false
	_, err := s.EnforceWriteBatch(context.Background(), "t1", "security", 1, false, func(*sqlx.Tx, int) (map[string]interface{}, error) {
		wrote = true
		return nil, nil
	})
	if err == nil || wrote {
		t.Fatalf("err=%v wrote=%v: the write must be refused before any transaction", err, wrote)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err) // in particular: no BEGIN on the metadata DB
	}
}
