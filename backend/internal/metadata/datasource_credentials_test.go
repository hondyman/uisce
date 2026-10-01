package metadata

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/secrets"
	"github.com/jmoiron/sqlx"
)

var errNoDial = errors.New("test: not dialling")

func withCredsStore(t *testing.T, values map[string]map[string]string) {
	t.Helper()
	p := secrets.NewMemoryProvider()
	for path, v := range values {
		if err := p.PutMap(context.Background(), path, v); err != nil {
			t.Fatal(err)
		}
	}
	r := dscreds.NewResolver(p)
	prev := datasourceCreds
	datasourceCreds = func() *dscreds.Resolver { return r }
	t.Cleanup(func() { datasourceCreds = prev })
}

// captureConnect records the connection details handed to the dialer.
func captureConnect(svc *CatalogScanService) *string {
	var got string
	svc.connectTargetDBFunc = func(ctx context.Context, details string) (*sql.DB, error) {
		got = details
		return nil, errNoDial
	}
	return &got
}

func TestTestConnectionByID_HydratesFromDatasourceSecret(t *testing.T) {
	tenant, ds := uuid.New(), uuid.New()
	path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, tenant.String(), ds.String())
	withCredsStore(t, map[string]map[string]string{path: {dscreds.KeyPassword: "from-store"}})

	svc := &CatalogScanService{}
	svc.getDatasourcesFunc = func(*uuid.UUID) ([]DatasourceConfig, error) {
		return []DatasourceConfig{{ID: ds, TenantID: tenant, Name: "orm", CredentialKind: "datasources",
			ConnectionDetails: `{"host":"h","port":5432,"database":"d","secret_path":"` + path + `","auth":{"basic":{"username":"u","password":"stale"}}}`}}, nil
	}
	got := captureConnect(svc)

	if err := svc.TestConnectionByID(context.Background(), ds); !errors.Is(err, errNoDial) {
		t.Fatalf("want dial attempt, got %v", err)
	}
	if !strings.Contains(*got, `"from-store"`) || strings.Contains(*got, "stale") {
		t.Fatalf("dialer did not get the stored credential only: %s", *got)
	}
}

// A datasource linked to a public.connections row takes its credentials from
// that connection's own secret, keyed by the connection's tenant and id.
func TestTestConnectionByID_HydratesFromConnectionSecret(t *testing.T) {
	tenant, ds, conn := uuid.New(), uuid.New(), uuid.New()
	path, _ := dscreds.CanonicalPath(dscreds.KindConnection, tenant.String(), conn.String())
	withCredsStore(t, map[string]map[string]string{path: {dscreds.KeyPassword: "conn-secret", dscreds.KeyUsername: "conn-user"}})

	svc := &CatalogScanService{}
	svc.getDatasourcesFunc = func(*uuid.UUID) ([]DatasourceConfig, error) {
		return []DatasourceConfig{{ID: ds, TenantID: tenant, Name: "crims",
			CredentialKind: "connections", CredentialOwnerID: conn.String(), CredentialTenantID: tenant.String(),
			ConnectionDetails: `{"host":"h","port":5432,"database":"d","username":null,"password":null,"secret_path":"` + path + `"}`}}, nil
	}
	got := captureConnect(svc)

	if err := svc.TestConnectionByID(context.Background(), ds); !errors.Is(err, errNoDial) {
		t.Fatalf("want dial attempt, got %v", err)
	}
	if !strings.Contains(*got, `"conn-secret"`) || !strings.Contains(*got, `"conn-user"`) {
		t.Fatalf("connection secret not applied: %s", *got)
	}
}

func TestTestConnectionByID_RefusesAnotherTenantsSecret(t *testing.T) {
	owner, other, ds := uuid.New(), uuid.New(), uuid.New()
	path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, owner.String(), ds.String())
	withCredsStore(t, map[string]map[string]string{path: {dscreds.KeyPassword: "owner-only"}})

	svc := &CatalogScanService{}
	svc.getDatasourcesFunc = func(*uuid.UUID) ([]DatasourceConfig, error) {
		// Row belongs to `other` but its config points at owner's secret.
		return []DatasourceConfig{{ID: ds, TenantID: other, Name: "x", ConnectionDetails: `{"secret_path":"` + path + `"}`}}, nil
	}
	got := captureConnect(svc)

	err := svc.TestConnectionByID(context.Background(), ds)
	if !errors.Is(err, dscreds.ErrRefMismatch) {
		t.Fatalf("want ErrRefMismatch, got %v", err)
	}
	if *got != "" {
		t.Fatal("dialer called with another tenant's credentials")
	}
}

func TestRecordsDBStrict_ResolvesCredentialsByRowTenant(t *testing.T) {
	owner, other, ds := uuid.New(), uuid.New(), uuid.New()
	path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, owner.String(), ds.String())
	withCredsStore(t, map[string]map[string]string{path: {dscreds.KeyPassword: "owner-only"}})

	sqlDB, mock, _ := sqlmock.New()
	defer sqlDB.Close()
	s := &BusinessObjectService{db: sqlx.NewDb(sqlDB, "sqlmock")}

	mock.ExpectQuery("FROM public.business_object_binding").
		WillReturnRows(sqlmock.NewRows([]string{"backend_id"}).AddRow(ds.String()))
	mock.ExpectQuery("FROM public.tenant_product_datasource").
		WillReturnRows(sqlmock.NewRows([]string{"config", "tenant_id"}).
			AddRow(`{"host":"h","port":5432,"database":"d","secret_path":"`+path+`"}`, other.String()))

	got, err := s.recordsDBStrict(context.Background(), "bo-1")
	if got != nil || !errors.Is(err, dscreds.ErrRefMismatch) {
		t.Fatalf("want ErrRefMismatch and no DB, got %v, %v", got, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
