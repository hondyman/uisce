package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/corecustom"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

var goldTenant = uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")

// A two-tab core page shaped like the mastering console.
func corePage(version int, tabs, components string) *PageStudioPage {
	app := json.RawMessage(`{"variables":[{"name":"entity","default":"product"}]}`)
	return &PageStudioPage{
		ID: uuid.MustParse("11111111-1111-4111-8111-111111111111"), TenantID: goldTenant,
		Name: "Mastering", Slug: "mastering", Version: version, IsCore: true, Status: "published",
		Layout: json.RawMessage(`[]`), Tabs: json.RawMessage(tabs), Components: json.RawMessage(components),
		DataSources: json.RawMessage(`[]`), PresentationEvents: json.RawMessage(`[]`), FilterBar: emptyPageLayoutJSON, App: &app,
	}
}

const v1Tabs = `[
  {"id":"golden","label":"Golden records","layout":{"root":"g","nodes":{"g":{"id":"g","type":"Column","children":["grid"]}}}},
  {"id":"runs","label":"Runs","layout":{"root":"r","nodes":{"r":{"id":"r","type":"Column","children":["runs_grid"]}}}}
]`
const v1Components = `{"grid":{"id":"grid","type":"DataGrid","props":{"title":"Golden records"}},"runs_grid":{"id":"runs_grid","type":"DataGrid","props":{"title":"Runs"}}}`

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A tenant adds a note widget to the golden tab of core v1; the gold copy
// then ships v2 (renames the runs tab, adds a KPI to the golden tab).
// Compare shows one customization - the widget - and two core updates;
// upgrading keeps the widget on v2, removing it gives exactly v2.
func TestPageStudioCore_CompareAndUpgrade(t *testing.T) {
	v1 := corePage(1, v1Tabs, v1Components)
	extTabs := strings.Replace(v1Tabs, `"children":["grid"]`, `"children":["grid","note"]`, 1)
	extComponents := strings.Replace(v1Components, `{"grid"`, `{"note":{"id":"note","type":"Text","props":{"title":"Desk notes"}},"grid"`, 1)
	ext := corePage(1, extTabs, extComponents)

	v2Tabs := strings.Replace(strings.Replace(v1Tabs, `"label":"Runs"`, `"label":"Mastering runs"`, 1), `"children":["grid"]`, `"children":["kpi","grid"]`, 1)
	v2Components := strings.Replace(v1Components, `{"grid"`, `{"kpi":{"id":"kpi","type":"KPI","props":{"title":"Open exceptions"}},"grid"`, 1)
	v2 := corePage(2, v2Tabs, v2Components)

	a := &pageAdoption{Mode: "extended", Active: true, BaseVersion: sql.NullInt64{Int64: 1, Valid: true},
		BaseSnapshot: mustJSON(t, contentOf(v1)), Extension: mustJSON(t, contentOf(ext))}

	base, extDoc, cur, err := extensionDocs(v2, a)
	if err != nil {
		t.Fatal(err)
	}
	rep := corecustom.Compare(base, extDoc, cur, pageGrouper)
	if len(rep.Customizations) != 1 {
		t.Fatalf("customizations: %s", mustJSON(t, rep.Customizations))
	}
	c := rep.Customizations[0]
	if c.ID != "component:note" || c.Summary != "added" || c.Conflict || len(c.Changes) != 2 {
		t.Fatalf("customization: %s", mustJSON(t, c))
	}
	if !strings.Contains(c.Label, "Desk notes") {
		t.Fatalf("label %q", c.Label)
	}
	if len(rep.CoreUpdates) != 2 {
		t.Fatalf("core updates: %s", mustJSON(t, rep.CoreUpdates))
	}

	kept, vanilla, err := upgradeExtension(v2, a, nil)
	if err != nil || vanilla {
		t.Fatalf("keep: vanilla=%v err=%v", vanilla, err)
	}
	var tabs []struct {
		ID     string `json:"id"`
		Label  string `json:"label"`
		Layout struct {
			Nodes map[string]struct {
				Children []string `json:"children"`
			} `json:"nodes"`
		} `json:"layout"`
	}
	if err := json.Unmarshal(kept.Tabs, &tabs); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(tabs[0].Layout.Nodes["g"].Children, ","); got != "kpi,grid,note" {
		t.Fatalf("golden tab children %q", got)
	}
	if tabs[1].Label != "Mastering runs" {
		t.Fatalf("core rename lost: %q", tabs[1].Label)
	}
	if !strings.Contains(string(kept.Components), `"note"`) || !strings.Contains(string(kept.Components), `"kpi"`) {
		t.Fatalf("components %s", kept.Components)
	}

	_, vanilla, err = upgradeExtension(v2, a, map[string]bool{"component:note": true})
	if err != nil || !vanilla {
		t.Fatalf("removing the only customization must leave vanilla v2: vanilla=%v err=%v", vanilla, err)
	}
}

func TestPresentCore(t *testing.T) {
	v2 := corePage(2, v1Tabs, v1Components)
	ext := corePage(1, v1Tabs, v1Components)
	ext.Name = "Mastering (desk)"
	a := &pageAdoption{Mode: "extended", Active: false, BaseVersion: sql.NullInt64{Int64: 1, Valid: true},
		BaseSnapshot: mustJSON(t, contentOf(corePage(1, v1Tabs, v1Components))), Extension: mustJSON(t, contentOf(ext))}
	presentCore(v2, a, true)
	c := v2.Customization
	if v2.Name != "Mastering (desk)" || c.Mode != "extended" || c.Active || !c.UpgradeAvailable || c.BaseVersion != 1 || c.CoreVersion != 2 {
		t.Fatalf("got name %q customization %+v", v2.Name, c)
	}
	if v2.Editable || !v2.CanCustomize {
		t.Fatalf("a tenant never edits core in place: editable=%v canCustomize=%v", v2.Editable, v2.CanCustomize)
	}

	plain := corePage(3, v1Tabs, v1Components)
	presentCore(plain, nil, false)
	if plain.Customization.Mode != "vanilla" || !plain.Customization.Active || plain.Customization.UpgradeAvailable {
		t.Fatalf("vanilla: %+v", plain.Customization)
	}
}

func pageRows(p *PageStudioPage) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"id", "tenant_id", "name", "slug", "description", "layout", "tabs", "components", "data_sources",
		"presentation_events", "filter_bar", "app_model", "version", "is_core", "status", "created_at", "updated_at",
	}).AddRow(p.ID, p.TenantID, p.Name, p.Slug, "", []byte(p.Layout), []byte(p.Tabs), []byte(p.Components), []byte(p.DataSources),
		[]byte(p.PresentationEvents), []byte(p.FilterBar), nil, p.Version, p.IsCore, p.Status, now, now)
}

func tenantRequest(method, target, tenant string, auth security.AuthInfo, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	auth.TenantIDs = []string{tenant}
	return req.WithContext(security.WithAuthInfo(req.Context(), auth))
}

// A core page a tenant switched off is not served at runtime.
func TestPageStudioCore_GetBySlug_InactiveIs404(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tenant := uuid.New()
	core := corePage(2, v1Tabs, v1Components)

	mock.ExpectQuery(`FROM page_definitions WHERE slug = \$1 AND tenant_id = \$2`).
		WithArgs("mastering", tenant).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))
	mock.ExpectQuery(`FROM page_definitions WHERE slug = \$1 AND is_core = true AND tenant_id = \$2`).
		WithArgs("mastering", goldTenant).WillReturnRows(pageRows(core))
	mock.ExpectQuery(`FROM core_object_adoption WHERE tenant_id = \$1 AND object_type = \$2 AND core_object_id = \$3`).
		WithArgs(tenant, "page", core.ID).
		WillReturnRows(sqlmock.NewRows([]string{"core_object_id", "active", "mode", "base_version", "base_snapshot", "extension", "clone_object_id"}).
			AddRow(core.ID, false, "vanilla", nil, nil, nil, nil))

	h := &PageStudioHandler{db: sqlx.NewDb(db, "sqlmock")}
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, tenantRequest(http.MethodGet, "/page-studio/pages/slug/mastering", tenant.String(), security.AuthInfo{UserID: "u"}, ""))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Everything the gold-copy admin authors is core unless they say
// otherwise - every MDM page is saved as core.
func TestPageStudio_Create_GoldCopyDefaultsToCore(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))
	mock.ExpectQuery(`INSERT INTO page_definitions`).
		WithArgs(sqlmock.AnyArg(), goldTenant, "Mastering", "mastering", "", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 1, true, "draft", nil).
		WillReturnRows(pageRows(corePage(1, `[]`, `{}`)))

	h := &PageStudioHandler{db: sqlx.NewDb(db, "sqlmock")}
	w := httptest.NewRecorder()
	h.create(w, tenantRequest(http.MethodPost, "/page-studio/pages", goldTenant.String(),
		security.AuthInfo{UserID: "u", IsGlobalAdmin: true}, `{"name":"Mastering","slug":"mastering"}`))
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// The gold copy edits its core pages directly; the lifecycle endpoints are
// for other tenants only, and need an admin.
func TestPageStudioCore_CustomizeNeedsTenantAdminOutsideGold(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	h := &PageStudioHandler{db: sqlx.NewDb(db, "sqlmock")}
	tenant := uuid.New()

	plain := tenantRequest(http.MethodGet, "/", tenant.String(), security.AuthInfo{UserID: "u", Roles: []string{"analyst"}}, "")
	if h.canCustomize(plain, tenant) {
		t.Fatal("a non-admin must not customize core pages")
	}
	mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))
	inGold := tenantRequest(http.MethodGet, "/", goldTenant.String(), security.AuthInfo{UserID: "u", Roles: []string{"tenant_admin"}}, "")
	if h.canCustomize(inGold, goldTenant) {
		t.Fatal("the gold copy edits core directly, it does not customize it")
	}
	mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))
	admin := tenantRequest(http.MethodGet, "/", tenant.String(), security.AuthInfo{UserID: "u", Roles: []string{"tenant_admin"}}, "")
	if !h.canCustomize(admin, tenant) {
		t.Fatal("a tenant admin customizes core pages")
	}
}

// Publishing changes only the status: a core page stays at its version
// (no false upgrade for tenants), and only the gold-copy admin may do it.
func TestPageStudio_SetStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	h := &PageStudioHandler{db: sqlx.NewDb(db, "sqlmock")}
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	core := corePage(3, v1Tabs, v1Components)
	path := "/page-studio/pages/" + core.ID.String() + "/status"

	w := httptest.NewRecorder()
	r.ServeHTTP(w, tenantRequest(http.MethodPut, path, goldTenant.String(), security.AuthInfo{UserID: "u", IsGlobalAdmin: true}, `{"status":"live"}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad status value: %d", w.Code)
	}

	// Not an admin: refused before anything is written.
	mock.ExpectQuery(`SELECT is_core FROM page_definitions`).WithArgs(core.ID, goldTenant).
		WillReturnRows(sqlmock.NewRows([]string{"is_core"}).AddRow(true))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, tenantRequest(http.MethodPut, path, goldTenant.String(), security.AuthInfo{UserID: "u"}, `{"status":"published"}`))
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-admin: %d %s", w.Code, w.Body.String())
	}

	// Gold-copy admin: status only, version untouched.
	mock.ExpectQuery(`SELECT is_core FROM page_definitions`).WithArgs(core.ID, goldTenant).
		WillReturnRows(sqlmock.NewRows([]string{"is_core"}).AddRow(true))
	mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))
	published := *core
	published.Status = "published"
	mock.ExpectQuery(`UPDATE page_definitions SET status = \$1, updated_at = NOW\(\)\s+WHERE id = \$2 AND tenant_id = \$3`).
		WithArgs("published", core.ID, goldTenant).WillReturnRows(pageRows(&published))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, tenantRequest(http.MethodPut, path, goldTenant.String(), security.AuthInfo{UserID: "u", IsGlobalAdmin: true}, `{"status":"published"}`))
	if w.Code != http.StatusOK {
		t.Fatalf("admin: %d %s", w.Code, w.Body.String())
	}
	var got struct {
		Status  string `json:"status"`
		Version int    `json:"version"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Status != "published" || got.Version != 3 {
		t.Fatalf("got %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
