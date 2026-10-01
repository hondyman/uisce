package api_test

import (
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	httpapi "github.com/hondyman/uisce/backend/internal/api"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// A binding's is_core follows the caller's tenant and ignores the body's
// isCore: the gold copy always gets a core binding (even when the wizard
// sends isCore:false), and a regular tenant can never mint one (even when
// it sends isCore:true).
func TestCreateBusinessObjectBinding_IsCoreFollowsGoldCopyTenant(t *testing.T) {
	for _, tc := range []struct {
		name       string
		gold       bool
		bodyIsCore string
	}{
		{"gold copy tenant, client says false", true, "false"},
		{"regular tenant, client says true", false, "true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			h := httpapi.NewBusinessObjectHandler(&fakeService{}, &mockResolver{}, sqlx.NewDb(db, "postgres"))
			r := chi.NewRouter()
			h.RegisterRoutes(r)

			mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM public.business_objects`).
				WithArgs("bo1", "ten").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM public.physical_backend`).
				WithArgs("be1").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			mock.ExpectQuery(`SELECT node_name FROM public.catalog_node`).
				WithArgs("node1").WillReturnRows(sqlmock.NewRows([]string{"node_name"}).AddRow("products"))
			mock.ExpectQuery(`SELECT count\(\*\) FROM public.business_object_binding`).
				WithArgs("bo1", "ten").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE($1::uuid = public.uisce_gold_copy_tenant_id(), false)`)).
				WithArgs("ten").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(tc.gold))
			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE public.business_object_binding SET is_default = false`).
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery(`INSERT INTO public.business_object_binding`).
				WithArgs("ten", "bo1", "be1", "node1", "Product Binding", sqlmock.AnyArg(), "NONE", true, tc.gold, sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"bo_binding_id"}).AddRow("bind1"))
			mock.ExpectCommit()

			body := `{"backendId":"be1","drivingNodeId":"node1","bindingName":"Product Binding","isCore":` + tc.bodyIsCore + `}`
			req := httptest.NewRequest("POST", "/business-objects/bo1/bindings", strings.NewReader(body))
			req = withValidHeaders(req, "ten", "ds1")
			req = withAuth(req, "ten")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, 201, w.Result().StatusCode, w.Body.String())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
