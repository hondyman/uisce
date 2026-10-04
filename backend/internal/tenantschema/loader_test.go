package tenantschema

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/scanner"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/models"
	"github.com/stretchr/testify/require"
)

var dsID = uuid.NewString()

type fakeStore struct {
	owner, schemas string
	gold           bool
	nodes          []*models.CatalogNode
	ownerErr       error
	goldErr        error
	readErr        error
	reads          int
}

func (f *fakeStore) Owner(context.Context, string) (string, error) { return f.owner, f.ownerErr }
func (f *fakeStore) IsGoldCopy(context.Context, string) (bool, error) {
	return f.gold, f.goldErr
}
func (f *fakeStore) Read(context.Context, string, string) (string, []*models.CatalogNode, error) {
	f.reads++
	return f.schemas, f.nodes, f.readErr
}

func okStore() *fakeStore {
	return &fakeStore{owner: "gold-tenant", gold: true, schemas: "orm, mdm,cash_flow", nodes: []*models.CatalogNode{{}}}
}

func TestLoad_ReturnsTheDeclaredSchemasInTheirDeclaredOrder(t *testing.T) {
	tpl, err := Loader{Store: okStore()}.Load(context.Background(), dsID)
	require.NoError(t, err)
	require.Equal(t, []string{"orm", "mdm", "cash_flow"}, tpl.Schemas)
	require.Equal(t, "gold-tenant", tpl.GoldTenantID)
	require.Equal(t, dsID, tpl.DatasourceID)
}

// A tenant's own datasource is never a template, and nothing of theirs is read to find that out.
func TestLoad_RefusesADatasourceThatIsNotTheGoldCopysAndReadsNothing(t *testing.T) {
	s := okStore()
	s.gold = false
	_, err := Loader{Store: s}.Load(context.Background(), dsID)
	require.ErrorIs(t, err, ErrNotGoldCopy)
	require.Zero(t, s.reads, "the ownership check comes before any metadata is read")
}

func TestLoad_RefusesWhatItCannotTrust(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func(*fakeStore)
		ds     string
		want   error
	}{
		"not a uuid":          {func(*fakeStore) {}, "not-a-uuid", ErrTemplateNotFound},
		"unknown datasource":  {func(s *fakeStore) { s.ownerErr = ErrTemplateNotFound }, dsID, ErrTemplateNotFound},
		"no schemas declared": {func(s *fakeStore) { s.schemas = "" }, dsID, ErrNoSchemas},
		"only separators":     {func(s *fakeStore) { s.schemas = " , ," }, dsID, ErrNoSchemas},
		"an injected name":    {func(s *fakeStore) { s.schemas = `orm; drop schema x` }, dsID, ErrBadSchemaName},
		"a quoted name":       {func(s *fakeStore) { s.schemas = `"orm"` }, dsID, ErrBadSchemaName},
		"upper case":          {func(s *fakeStore) { s.schemas = "ORM" }, dsID, ErrBadSchemaName},
		"nothing scanned":     {func(s *fakeStore) { s.nodes = nil }, dsID, ErrNoNodes},
	} {
		t.Run(name, func(t *testing.T) {
			s := okStore()
			tc.mutate(s)
			_, err := Loader{Store: s}.Load(context.Background(), tc.ds)
			require.ErrorIs(t, err, tc.want)
		})
	}
	_, err := Loader{}.Load(context.Background(), dsID)
	require.Error(t, err)
	s := okStore()
	s.goldErr = errors.New("db down")
	_, err = Loader{Store: s}.Load(context.Background(), dsID)
	require.ErrorContains(t, err, "db down", "a failed check is an error, never a pass")
	require.Zero(t, s.reads)
}

func TestLoad_ADeclaredSchemaListedTwiceIsListedOnce(t *testing.T) {
	s := okStore()
	s.schemas = "orm,mdm,orm"
	tpl, err := Loader{Store: s}.Load(context.Background(), dsID)
	require.NoError(t, err)
	require.Equal(t, []string{"orm", "mdm"}, tpl.Schemas)
}

// ---- the alpha implementation ----------------------------------------------------------------

type fakeResolver struct {
	tenant string
	err    error
}

func (f fakeResolver) Resolve(context.Context, string) (*security.ResolvedDatasource, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &security.ResolvedDatasource{TenantID: f.tenant}, nil
}

func TestAlphaStore_OwnerComesFromTheResolverAndAnUnavailableOneIsNotFound(t *testing.T) {
	s := &AlphaStore{Resolver: fakeResolver{tenant: "t1"}}
	got, err := s.Owner(context.Background(), dsID)
	require.NoError(t, err)
	require.Equal(t, "t1", got)
	s = &AlphaStore{Resolver: fakeResolver{err: security.ErrDatasourceNotAvailable}}
	_, err = s.Owner(context.Background(), dsID)
	require.ErrorIs(t, err, ErrTemplateNotFound)
	_, err = (&AlphaStore{}).Owner(context.Background(), dsID)
	require.Error(t, err)
}

func TestAlphaStore_ReadsAsTheOwnerWithRowLevelSecurityAndOnlyWhatTheCompilerUses(t *testing.T) {
	sdb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer sdb.Close()
	ds := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec(`set_config\('uisce.current_tenant'`).WithArgs("gold-tenant").WillReturnResult(driver.ResultNoRows)
	mock.ExpectExec(`set_config\('app.tenant_id'`).WithArgs("gold-tenant").WillReturnResult(driver.ResultNoRows)
	mock.ExpectQuery(`config->>'schema' FROM public.tenant_product_datasource WHERE id = \$1`).WithArgs(ds.String()).
		WillReturnRows(sqlmock.NewRows([]string{"s"}).AddRow("orm,mdm"))
	mock.ExpectQuery(`FROM public\.catalog_node\s+WHERE tenant_datasource_id = \$1 AND is_active AND node_type_id IN \(\$2, \$3, \$4\)`).
		WithArgs(ds.String(), scanner.NODE_TYPE_COLUMN, scanner.NODE_TYPE_TABLE, scanner.NODE_TYPE_SCHEMA).
		WillReturnRows(sqlmock.NewRows([]string{"id", "t", "n", "p", "props"}).
			AddRow(uuid.New(), scanner.NODE_TYPE_TABLE, "account", "/orm/account", []byte(`{"schema":"orm"}`)))
	mock.ExpectCommit()

	schemas, nodes, err := (&AlphaStore{DB: sdb}).Read(context.Background(), "gold-tenant", ds.String())
	require.NoError(t, err)
	require.Equal(t, "orm,mdm", schemas)
	require.Len(t, nodes, 1)
	require.Equal(t, "/orm/account", nodes[0].QualifiedPath)
	require.JSONEq(t, `{"schema":"orm"}`, string(nodes[0].Properties))
	require.Equal(t, ds, nodes[0].TenantDatasourceId)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAlphaStore_AMissingDatasourceRowIsNotFound(t *testing.T) {
	sdb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer sdb.Close()
	mock.ExpectBegin()
	mock.ExpectExec(`set_config`).WillReturnResult(driver.ResultNoRows)
	mock.ExpectExec(`set_config`).WillReturnResult(driver.ResultNoRows)
	mock.ExpectQuery(`config->>'schema'`).WillReturnRows(sqlmock.NewRows([]string{"s"}))
	mock.ExpectRollback()
	_, _, err = (&AlphaStore{DB: sdb}).Read(context.Background(), "gold-tenant", uuid.NewString())
	require.ErrorIs(t, err, ErrTemplateNotFound)
}
