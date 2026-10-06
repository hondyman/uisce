package handlers

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/security"
)

const identityPage = `{"id":"11111111-1111-4111-8111-111111111111","name":"Orders","slug":"orders","version":7,"isCore":true,"status":"published",
 "createdAt":"2026-01-01","editable":true,"menuPlacements":[{"x":1}],
 "layout":{"root":"r"},"components":{"a":{"id":"a","type":"TextBlock"}},"app":{"fragments":[{"fragment":"orders-list","version":1}]}}`

func TestTemplateBundle_CarriesContentNeverIdentity(t *testing.T) {
	canon, _, err := templateBundle(json.RawMessage(identityPage), nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(canon)
	for _, leak := range []string{"11111111", `"slug"`, `"version":7`, "isCore", "published", "createdAt", "editable", "menuPlacements", `"name":"Orders"`} {
		if strings.Contains(s, leak) {
			t.Errorf("template bundle leaks %s: %s", leak, s)
		}
	}
	for _, keep := range []string{`"components"`, `"layout"`, `"fragments"`} {
		if !strings.Contains(s, keep) {
			t.Errorf("template bundle lost %s", keep)
		}
	}
}

func TestTemplateBundle_HashIgnoresKeyOrderAndTracksContent(t *testing.T) {
	_, a, _ := templateBundle(json.RawMessage(`{"layout":{"root":"r"},"components":{"a":1}}`), nil)
	_, b, _ := templateBundle(json.RawMessage(`{"components":{"a":1},"layout":{"root":"r"},"id":"x","version":3}`), nil)
	_, c, _ := templateBundle(json.RawMessage(`{"layout":{"root":"r"},"components":{"a":2}}`), nil)
	if a != b {
		t.Error("the same content hashed differently (identity or key order leaked into the hash)")
	}
	if a == c {
		t.Error("different content hashed the same")
	}
}

func TestNewPageFromTemplate_FreshDraftWithNoInheritedIdentity(t *testing.T) {
	canon, _, _ := templateBundle(json.RawMessage(identityPage), nil)
	var b PageBundle
	_ = json.Unmarshal(canon, &b)
	out, err := newPageFromTemplate(b.Page, "My orders", "my-orders")
	if err != nil {
		t.Fatal(err)
	}
	var page map[string]any
	_ = json.Unmarshal(out, &page)
	if page["name"] != "My orders" || page["slug"] != "my-orders" || page["status"] != "draft" {
		t.Fatalf("got %v", page)
	}
	for _, k := range []string{"id", "version", "isCore", "createdAt"} {
		if _, has := page[k]; has {
			t.Errorf("new page inherited %s", k)
		}
	}
	if page["components"] == nil || page["layout"] == nil {
		t.Error("content was dropped")
	}
}

func tmplCols() []string {
	return []string{"id", "tenant_id", "slug", "version", "name", "description", "category", "bundle", "bundle_hash", "is_core", "created_at", "created_by"}
}

func tmplRow(slug string, version int, bundle []byte, hash string) *sqlmock.Rows {
	return sqlmock.NewRows(tmplCols()).AddRow("11111111-1111-4111-8111-111111111111", goldTenant, slug, version, slug, "", "general", bundle, hash, false, "2026-12-02", "u")
}

func publishTemplateReq(h *PageStudioHandler) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.publishTemplate(w, tenantRequest(http.MethodPost, "/page-studio/templates", goldTenant.String(), security.AuthInfo{UserID: "u"},
		`{"slug":"orders-starter","name":"Orders","category":"sales","pageId":"11111111-1111-4111-8111-111111111111"}`))
	return w
}

func TestPublishTemplate_NewVersionThenSameContentIsNotAnother(t *testing.T) {
	page := corePage(1, `[]`, `{"a":{"id":"a","type":"TextBlock"}}`)
	doc, _ := json.Marshal(page)
	_ = doc

	h, mock := newFragHandler(t)
	hash := &captured{}
	mock.ExpectQuery(`FROM page_definitions WHERE id`).WillReturnRows(pageRows(page))
	mock.ExpectQuery(`FROM page_templates WHERE tenant_id`).WillReturnRows(sqlmock.NewRows(tmplCols()))
	mock.ExpectQuery(`INSERT INTO page_templates`).
		WithArgs(goldTenant, "orders-starter", "Orders", "", "sales", sqlmock.AnyArg(), hash, false, "u").
		WillReturnRows(tmplRow("orders-starter", 1, []byte(`{}`), "x"))
	if w := publishTemplateReq(h); w.Code != http.StatusCreated {
		t.Fatalf("first: %d %s", w.Code, w.Body.String())
	}

	mock.ExpectQuery(`FROM page_definitions WHERE id`).WillReturnRows(pageRows(page))
	mock.ExpectQuery(`FROM page_templates WHERE tenant_id`).WillReturnRows(tmplRow("orders-starter", 1, []byte(`{}`), hash.value))
	if w := publishTemplateReq(h); w.Code != http.StatusOK { // same content: no INSERT expected
		t.Fatalf("again: %d %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInstantiateTemplate_ConflictDryRunApply(t *testing.T) {
	frag := stored(t, "orders-list", 1, `{}`)
	doc := json.RawMessage(`{"layout":{"root":"r"},"components":{},"app":{"fragments":[{"fragment":"orders-list","version":1}]}}`)
	bundle, hash, _ := templateBundle(doc, []BundleFragment{{Slug: frag.Slug, Version: 1, Name: frag.Name, ContentHash: frag.ContentHash, Content: frag.Content}})

	h, mock := newFragHandler(t)
	post := func(q string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.instantiateTemplate(w, withURLParams(tenantRequest(http.MethodPost, "/x"+q, goldTenant.String(), security.AuthInfo{UserID: "u"}, `{"name":"Mine","slug":"mine"}`), "slug", "orders-starter", "version", "1"))
		return w
	}
	template := func() {
		mock.ExpectQuery(`FROM page_templates WHERE slug`).WillReturnRows(tmplRow("orders-starter", 1, bundle, hash))
	}
	hasFragment := func(h string) {
		mock.ExpectQuery(`SELECT content_hash FROM page_fragments`).WillReturnRows(sqlmock.NewRows([]string{"content_hash"}).AddRow(h))
	}
	noFragment := func() {
		mock.ExpectQuery(`SELECT content_hash FROM page_fragments`).WillReturnRows(sqlmock.NewRows([]string{"content_hash"}))
	}

	template()
	hasFragment("someone else's version")
	if w := post(""); w.Code != http.StatusConflict {
		t.Fatalf("conflict: %d %s", w.Code, w.Body.String())
	}
	template()
	noFragment()
	if w := post("?dryRun=true"); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"applied":false`) || !strings.Contains(w.Body.String(), `"slug":"mine"`) {
		t.Fatalf("dry run: %d %s", w.Code, w.Body.String())
	}
	template()
	hasFragment(frag.ContentHash) // already have it: reuse, nothing to write
	if w := post(""); w.Code != http.StatusCreated || !strings.Contains(w.Body.String(), `"applied":true`) {
		t.Fatalf("reuse: %d %s", w.Code, w.Body.String())
	}
	template()
	noFragment()
	mock.ExpectBegin()
	// goldCopyID + ApplyTenantGUCs (R3 wave1): gold resolve, then current/app/gold GUCs
	mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))
	mock.ExpectExec("SELECT set_config").WithArgs(goldTenant.String()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SELECT set_config").WithArgs(goldTenant.String()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SELECT set_config").WithArgs(goldTenant.String()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO page_fragments`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if w := post(""); w.Code != http.StatusCreated {
		t.Fatalf("apply: %d %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInstantiateTemplate_RefusesATamperedBundle(t *testing.T) {
	frag := stored(t, "orders-list", 1, `{}`)
	bundle, hash, _ := templateBundle(json.RawMessage(`{"app":{"fragments":[{"fragment":"orders-list","version":1}]}}`),
		[]BundleFragment{{Slug: "orders-list", Version: 1, ContentHash: "not the real hash", Content: frag.Content}})
	h, mock := newFragHandler(t)
	mock.ExpectQuery(`FROM page_templates WHERE slug`).WillReturnRows(tmplRow("orders-starter", 1, bundle, hash))
	w := httptest.NewRecorder()
	h.instantiateTemplate(w, withURLParams(tenantRequest(http.MethodPost, "/x", goldTenant.String(), security.AuthInfo{UserID: "u"}, `{"name":"Mine","slug":"mine"}`), "slug", "orders-starter", "version", "1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestPublishTemplate_Refusals(t *testing.T) {
	h, _ := newFragHandler(t)
	for name, body := range map[string]string{
		"bad slug": `{"slug":"Bad Slug","name":"x","pageId":"11111111-1111-4111-8111-111111111111"}`,
		"no name":  `{"slug":"ok-slug","pageId":"11111111-1111-4111-8111-111111111111"}`,
		"no page":  `{"slug":"ok-slug","name":"x"}`,
	} {
		w := httptest.NewRecorder()
		h.publishTemplate(w, tenantRequest(http.MethodPost, "/x", goldTenant.String(), security.AuthInfo{UserID: "u"}, body))
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: %d", name, w.Code)
		}
	}
}

// captured records an argument so a later step can use what the handler computed.
type captured struct{ value string }

func (c *captured) Match(v driver.Value) bool {
	str, ok := v.(string)
	c.value = str
	return ok
}

// withURLParams gives a handler the chi path params a router would.
func withURLParams(r *http.Request, kv ...string) *http.Request {
	rc := chi.NewRouteContext()
	for i := 0; i+1 < len(kv); i += 2 {
		rc.URLParams.Add(kv[i], kv[i+1])
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rc))
}
