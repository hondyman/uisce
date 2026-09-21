package handlers

import (
	"database/sql/driver"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

const (
	vrGold   = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	vrTenant = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	vrRuleID = "00000000-0000-4000-8000-000000000001"
)

var vrCols = []string{"id", "node_name", "description", "properties", "config", "is_active", "tenant_id", "created_at", "updated_at"}

func vrRow(ruleTenant string) []driver.Value {
	return []driver.Value{vrRuleID, "r", "", []byte(`{"bo_name":"party","tenant_id":"` + ruleTenant + `","severity":"BLOCK","timing":"pre_write"}`),
		[]byte(`{"rule_ast":{"type":"condition","field":"amount","fieldPath":"amount","operator":">","value":10,"valueType":"number"}}`),
		true, ruleTenant, "2026-01-01", "2026-01-01"}
}

// newVRRouter returns the real routes over a mocked DB. gold is looked up first by the service, then
// the rule query runs with the tenants the caller may see; rows are returned only when the rule's tenant
// is one of them, which is what the SQL filter does in Postgres.
func newVRRouter(t *testing.T, ruleTenant string, expectQuery bool) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	x := sqlx.NewDb(db, "postgres")
	if expectQuery {
		mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(vrGold))
		rows := sqlmock.NewRows(vrCols)
		if ruleTenant == vrTenant || ruleTenant == vrGold {
			rows.AddRow(vrRow(ruleTenant)...)
		}
		mock.ExpectQuery(`FROM catalog_node n`).WillReturnRows(rows)
		// a successful load also asks for the gold-copy tenant again to classify the origin
		mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(vrGold))
	}
	r := chi.NewRouter()
	NewValidationRuleHandler(analytics.NewValidationRuleService(x), x).RegisterRoutes(r)
	return r, mock
}

func do(h http.Handler, method, path, body, tenant string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if tenant != "" {
		req = req.WithContext(security.WithAuthInfo(req.Context(), security.AuthInfo{UserID: "u", TenantIDs: []string{tenant}}))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestRuleByID_RequiresATenant(t *testing.T) {
	h, mock := newVRRouter(t, vrTenant, false)
	for _, c := range []struct{ method, path, body string }{
		{"GET", "/validation-rule-nodes/" + vrRuleID, ""},
		{"POST", "/validation-rule-nodes/" + vrRuleID + "/evaluate", `{"amount": 50}`},
	} {
		if w := do(h, c.method, c.path, c.body, ""); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a tenant = %d; want 401", c.method, c.path, w.Code)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("no query may run before the tenant is known: %v", err)
	}
}

func TestRuleByID_AnotherTenantsRuleIsNotFound(t *testing.T) {
	other := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	for _, c := range []struct{ method, path, body string }{
		{"GET", "/validation-rule-nodes/" + vrRuleID, ""},
		{"POST", "/validation-rule-nodes/" + vrRuleID + "/evaluate", `{"amount": 50}`},
	} {
		h, _ := newVRRouter(t, other, true) // the rule belongs to a tenant the caller cannot see: no row comes back
		w := do(h, c.method, c.path, c.body, vrTenant)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s %s for another tenant's rule = %d %q; want 404", c.method, c.path, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), other) || strings.Contains(w.Body.String(), "amount") {
			t.Errorf("%s %s leaked rule details: %s", c.method, c.path, w.Body.String())
		}
	}
}

func TestRuleByID_OwnAndCoreRulesAreReadable(t *testing.T) {
	for _, ruleTenant := range []string{vrTenant, vrGold} {
		h, _ := newVRRouter(t, ruleTenant, true)
		if w := do(h, "GET", "/validation-rule-nodes/"+vrRuleID, "", vrTenant); w.Code != http.StatusOK {
			t.Errorf("GET of a rule from %s = %d %q; want 200", ruleTenant, w.Code, w.Body.String())
		}
	}
	h, _ := newVRRouter(t, vrGold, true)
	w := do(h, "GET", "/validation-rule-nodes/"+vrRuleID, "", vrTenant)
	if !strings.Contains(w.Body.String(), `"origin":"core"`) {
		t.Errorf("a gold-copy rule must be reported as core to a tenant: %s", w.Body.String())
	}
}

func TestEvaluate_RunsOnlyRulesTheTenantMaySee(t *testing.T) {
	h, _ := newVRRouter(t, vrGold, true)
	w := do(h, "POST", "/validation-rule-nodes/"+vrRuleID+"/evaluate", `{"amount": 50}`, vrTenant)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"result":true`) {
		t.Errorf("evaluating an inherited core rule = %d %q; want 200 result:true", w.Code, w.Body.String())
	}
}
