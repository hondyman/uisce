package metadata

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// A BO's is_core is decided by the caller's tenant: core in the gold copy
// (the only place ListBusinessObjectsComposed looks there), custom anywhere
// else. Product was created non-core from the gold copy and vanished.
func TestCreateBusinessObject_IsCoreFollowsGoldCopyTenant(t *testing.T) {
	const tenantID = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	const tableNode = "11111111-1111-4111-8111-111111111111"
	const idNode = "22222222-2222-4222-8222-222222222222"

	for _, tc := range []struct {
		name string
		gold bool
	}{
		{"gold copy tenant creates core", true},
		{"regular tenant creates custom", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			svc := NewBusinessObjectService(sqlx.NewDb(db, "postgres"), nil, nil, nil)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM public.catalog_node WHERE id = $1::uuid`)).
				WithArgs(tableNode).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(tableNode))
			mock.ExpectQuery(`SELECT id FROM public.catalog_node\s+WHERE parent_id = \$1::uuid AND node_name = 'id'`).
				WithArgs(tableNode).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(idNode))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE($1::uuid = public.uisce_gold_copy_tenant_id(), false)`)).
				WithArgs(tenantID).WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(tc.gold))
			mock.ExpectExec(`INSERT INTO public.business_objects`).
				WithArgs(sqlmock.AnyArg(), tenantID, "product", "Product", "", tableNode, idNode,
					tableNode, "products", sqlmock.AnyArg(), true, tc.gold).
				WillReturnResult(sqlmock.NewResult(0, 1))

			bo, err := svc.CreateBusinessObject(context.Background(),
				&security.Context{TenantID: tenantID, DatasourceID: "ds1"},
				models.CreateBusinessObjectRequest{Name: "Product", DriverTableID: tableNode, DriverTableName: "products"},
				"user1")
			require.NoError(t, err)
			require.Equal(t, tc.gold, bo.IsCore)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
