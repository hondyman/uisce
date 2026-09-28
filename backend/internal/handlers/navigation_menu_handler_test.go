package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

var navCols = []string{"id", "tenant_id", "parent_id", "node_key", "label", "icon", "target_page_key", "display_order", "required_entitlement"}

// A tenant's menu is the gold copy's (inherited, read-only) plus its own,
// which may hang under a gold section; the list query also drops entries
// for core pages the tenant switched off.
func TestNavigationMenu_ListInheritsGoldCopy(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tenant := uuid.New()
	mdm, console, mine := uuid.New(), uuid.New(), uuid.New()
	slug := "mastering-console"
	own := "desk-notes"

	mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))
	mock.ExpectQuery(`FROM navigation_menu_nodes\s+WHERE \(tenant_id = \$1 OR tenant_id = \$2\)\s+AND \(target_page_key IS NULL OR target_page_key NOT IN .*NOT a.active`).
		WithArgs(tenant, goldTenant).
		WillReturnRows(sqlmock.NewRows(navCols).
			AddRow(mdm, goldTenant, nil, "mdm", "Master Data", nil, nil, 0, "BASE_USER").
			AddRow(console, goldTenant, mdm, "mastering-console", "Mastering console", nil, slug, 0, "BASE_USER").
			AddRow(mine, tenant, mdm, "desk-notes", "Desk notes", nil, own, 1, "BASE_USER"))

	h := NewNavigationMenuHandler(sqlx.NewDb(db, "sqlmock"))
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, tenantRequest(http.MethodGet, "/navigation-menu/", tenant.String(), security.AuthInfo{UserID: "u"}, ""))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var roots []struct {
		Label     string `json:"label"`
		Inherited bool   `json:"inherited"`
		Children  []struct {
			Label     string `json:"label"`
			Inherited bool   `json:"inherited"`
		} `json:"children"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &roots); err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 || roots[0].Label != "Master Data" || !roots[0].Inherited || len(roots[0].Children) != 2 {
		t.Fatalf("tree: %s", w.Body.String())
	}
	if !roots[0].Children[0].Inherited || roots[0].Children[1].Inherited {
		t.Fatalf("gold entry inherited, tenant entry its own: %s", w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMenuPlacements(t *testing.T) {
	mdm, console := uuid.New(), uuid.New()
	slug := "mastering-console"
	got := menuPlacements([]NavigationMenuNode{
		{ID: mdm, Label: "Master Data", Inherited: true},
		{ID: console, ParentID: &mdm, Label: "Mastering console", TargetPageKey: &slug, Inherited: true},
	})
	p := got[slug]
	if len(p) != 1 || strings.Join(p[0].Path, " › ") != "Master Data › Mastering console" || !p[0].Inherited || p[0].NodeID != console {
		t.Fatalf("got %+v", got)
	}
}

// A tenant may place entries under its own nodes or the gold copy's, never
// under another tenant's.
func TestNavigationMenu_CreateRejectsForeignParent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tenant, other, parent := uuid.New(), uuid.New(), uuid.New()
	mock.ExpectQuery(`SELECT tenant_id FROM navigation_menu_nodes WHERE id = \$1`).WithArgs(parent).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(other))
	mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))

	h := NewNavigationMenuHandler(sqlx.NewDb(db, "sqlmock"))
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	body := `{"parentId":"` + parent.String() + `","nodeKey":"x","label":"X","targetPageKey":"x"}`
	r.ServeHTTP(w, tenantRequest(http.MethodPost, "/navigation-menu/", tenant.String(), security.AuthInfo{UserID: "u"}, body))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
