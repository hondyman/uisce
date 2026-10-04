package metadata

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func ptr(b bool) *bool { return &b }

func TestProfileData_TheRequestBeatsTheDatasourceWhichBeatsTheDefault(t *testing.T) {
	cfg := func(s string) DatasourceConfig { return DatasourceConfig{ConnectionDetails: s} }
	req := func(b *bool) context.Context {
		return WithScanOptions(context.Background(), ScanOptions{ProfileData: b})
	}
	for name, tc := range map[string]struct {
		ctx  context.Context
		ds   DatasourceConfig
		want bool
	}{
		"no opinion anywhere: profile, as before":                       {context.Background(), cfg(`{"host":"h"}`), true},
		"the datasource says no":                                        {context.Background(), cfg(`{"profile_data":false}`), false},
		"the datasource says yes":                                       {context.Background(), cfg(`{"profile_data":true}`), true},
		"the request says no":                                           {req(ptr(false)), cfg(`{"host":"h"}`), false},
		"the request says no over a yes":                                {req(ptr(false)), cfg(`{"profile_data":true}`), false},
		"the request says yes over a no":                                {req(ptr(true)), cfg(`{"profile_data":false}`), true},
		"a request with no preference defers":                           {req(nil), cfg(`{"profile_data":false}`), false},
		"connection details that are not JSON":                          {context.Background(), cfg(`not json`), true},
		"a value that is not a boolean is ignored, never read as false": {context.Background(), cfg(`{"profile_data":"no"}`), true},
		"empty connection details":                                      {context.Background(), cfg(``), true},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, profileData(tc.ctx, tc.ds))
		})
	}
}

// skipScanner records whether it was asked to leave out the data profile.
type skipScanner struct {
	fakeScanner
	skipped bool
}

func (s *skipScanner) SkipDataProfile() { s.skipped = true }

func scanOnce(t *testing.T, ctx context.Context, details string, sc MetadataScanner) {
	t.Helper()
	orig := []interface{}{newMetadataScanner, buildERDChart, buildEnhancedERDChart, buildTechnicalLineageChart, buildSemanticLineageChart}
	defer func() {
		newMetadataScanner = orig[0].(func(*sql.DB, uuid.UUID, uuid.UUID, string, map[string]db.GoldCopyNodeInfo, bool, []string) (MetadataScanner, error))
		buildERDChart = orig[1].(func(context.Context, *sql.DB, string, bool) error)
		buildEnhancedERDChart = orig[2].(func(context.Context, *sql.DB, string, bool) error)
		buildTechnicalLineageChart = orig[3].(func(context.Context, *sql.DB, string, bool) error)
		buildSemanticLineageChart = orig[4].(func(context.Context, *sql.DB, string, bool) error)
	}()
	dsID := uuid.New()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()
	mock.MatchExpectationsInOrder(false)
	mock.ExpectExec("UPDATE public.tenant_product_datasource").WithArgs("running", sqlmock.AnyArg(), "", dsID).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE public.tenant_product_datasource").WithArgs("complete", sqlmock.AnyArg(), sqlmock.AnyArg(), dsID).WillReturnResult(sqlmock.NewResult(1, 1))
	svc := &CatalogScanService{alphaDB: sqlx.NewDb(mockDB, "postgres")}
	svc.connectTargetDBFunc = func(context.Context, string) (*sql.DB, error) {
		return sql.Open("pgx", "postgres://u:p@localhost:5432/d?sslmode=disable")
	}
	svc.storeFunc = func(context.Context, uuid.UUID, []*models.CatalogNode, []models.CatalogEdge, chan<- models.ScanProgress) (int64, int64, int64, error) {
		return 0, 0, 0, nil
	}
	newMetadataScanner = func(*sql.DB, uuid.UUID, uuid.UUID, string, map[string]db.GoldCopyNodeInfo, bool, []string) (MetadataScanner, error) {
		return sc, nil
	}
	noop := func(context.Context, *sql.DB, string, bool) error { return nil }
	buildERDChart, buildEnhancedERDChart, buildTechnicalLineageChart, buildSemanticLineageChart = noop, noop, noop, noop
	_, err = svc.scanSingleDatasource(ctx, DatasourceConfig{ID: dsID, TenantID: uuid.New(), Name: "ds", SourceSystem: "pg", ConnectionDetails: details, IsGoldCopy: true}, nil, nil)
	require.NoError(t, err)
}

func TestScan_SkipsTheDataProfileOnlyWhenAskedTo(t *testing.T) {
	base := `{"host":"h","port":5432,"database":"d"`
	for name, tc := range map[string]struct {
		ctx     context.Context
		details string
		skipped bool
	}{
		"default":                    {context.Background(), base + `}`, false},
		"the datasource asks":        {context.Background(), base + `,"profile_data":false}`, true},
		"the request asks":           {WithScanOptions(context.Background(), ScanOptions{ProfileData: ptr(false)}), base + `}`, true},
		"the request overrides a no": {WithScanOptions(context.Background(), ScanOptions{ProfileData: ptr(true)}), base + `,"profile_data":false}`, false},
	} {
		t.Run(name, func(t *testing.T) {
			sc := &skipScanner{}
			scanOnce(t, tc.ctx, tc.details, sc)
			require.Equal(t, tc.skipped, sc.skipped)
		})
	}
}

// A scanner that has no such option (a fake, or another implementation) is scanned as usual, never failed.
func TestScan_AScannerThatCannotSkipIsNotFailedForIt(t *testing.T) {
	scanOnce(t, WithScanOptions(context.Background(), ScanOptions{ProfileData: ptr(false)}), `{"host":"h"}`, &fakeScanner{})
}
