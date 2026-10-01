package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

var fragCols = []string{"id", "tenant_id", "slug", "version", "name", "description", "content", "content_hash", "is_core", "created_at", "created_by"}

func fragRow(slug string, version int, content, hash string) *sqlmock.Rows {
	return sqlmock.NewRows(fragCols).AddRow(uuid.New(), goldTenant, slug, version, slug, "", []byte(content), hash, false, "2026-12-01", "u")
}

func newFragHandler(t *testing.T) (*PageStudioHandler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &PageStudioHandler{db: sqlx.NewDb(db, "sqlmock")}, mock
}

func publish(h *PageStudioHandler, admin bool, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.publishFragment(w, tenantRequest(http.MethodPost, "/page-studio/fragments", goldTenant.String(), security.AuthInfo{UserID: "u", IsGlobalAdmin: admin}, body))
	return w
}

func TestCanonicalFragment_EqualContentHashesEqual(t *testing.T) {
	a, _, err := canonicalFragment([]byte(`{"components":{"a":{"id":"a","type":"TextBlock"}},"nodes":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	// Different key order and explicit empty lists: the same fragment.
	b, _, _ := canonicalFragment([]byte(`{"queries":[],"variables":[],"nodes":{},"components":{"a":{"type":"TextBlock","id":"a"}}}`))
	if fragmentHash(a) != fragmentHash(b) {
		t.Fatalf("equal content hashed differently:\n%s\n%s", a, b)
	}
	c, _, _ := canonicalFragment([]byte(`{"components":{"a":{"id":"a","type":"KeyValue"}},"nodes":{}}`))
	if fragmentHash(a) == fragmentHash(c) {
		t.Fatal("different content hashed the same")
	}
}

func TestPublishFragment_NewSlugIsVersionOne(t *testing.T) {
	h, mock := newFragHandler(t)
	mock.ExpectQuery(`FROM page_fragments WHERE tenant_id`).WillReturnRows(sqlmock.NewRows(fragCols))
	mock.ExpectQuery(`INSERT INTO page_fragments`).
		WithArgs(goldTenant, "orders-list", "Orders", "", sqlmock.AnyArg(), sqlmock.AnyArg(), false, "u").
		WillReturnRows(fragRow("orders-list", 1, `{}`, "h"))
	if w := publish(h, false, `{"slug":"orders-list","name":"Orders","content":{"components":{},"nodes":{}}}`); w.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPublishFragment_SameContentIsNotANewVersion(t *testing.T) {
	h, mock := newFragHandler(t)
	canon, _, _ := canonicalFragment([]byte(`{"components":{},"nodes":{}}`))
	mock.ExpectQuery(`FROM page_fragments WHERE tenant_id`).WillReturnRows(fragRow("orders-list", 3, `{}`, fragmentHash(canon)))
	if w := publish(h, false, `{"slug":"orders-list","name":"Orders","content":{"components":{},"nodes":{}}}`); w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil { // no INSERT expected
		t.Fatal(err)
	}
}

func TestPublishFragment_OnlyGoldAdminMakesItCore(t *testing.T) {
	h, mock := newFragHandler(t)
	mock.ExpectQuery(`FROM page_fragments WHERE tenant_id`).WillReturnRows(sqlmock.NewRows(fragCols))
	mock.ExpectQuery(`INSERT INTO page_fragments`).
		WithArgs(goldTenant, "orders-list", "Orders", "", sqlmock.AnyArg(), sqlmock.AnyArg(), false, "u"). // asked for core, not an admin
		WillReturnRows(fragRow("orders-list", 1, `{}`, "h"))
	publish(h, false, `{"slug":"orders-list","name":"Orders","isCore":true,"content":{}}`)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPublishFragment_Refusals(t *testing.T) {
	h, mock := newFragHandler(t)
	cases := []struct {
		name, body string
		want       int
	}{
		{"bad slug", `{"slug":"Bad Slug","name":"x","content":{}}`, http.StatusBadRequest},
		{"no name", `{"slug":"ok-slug","content":{}}`, http.StatusBadRequest},
		{"not json", `{"slug":"ok-slug","name":"x","content":"nope"}`, http.StatusBadRequest},
		{"uses itself", `{"slug":"ok-slug","name":"x","content":{"uses":[{"fragment":"ok-slug","version":1}]}}`, http.StatusUnprocessableEntity},
	}
	for _, c := range cases {
		if w := publish(h, false, c.body); w.Code != c.want {
			t.Errorf("%s: status %d, want %d (%s)", c.name, w.Code, c.want, w.Body.String())
		}
	}
	_ = mock
}

func TestPublishFragment_UsesMustExistAndNotNestDeeper(t *testing.T) {
	h, mock := newFragHandler(t)
	mock.ExpectQuery(`FROM page_fragments WHERE slug`).WillReturnRows(sqlmock.NewRows(fragCols)) // missing
	if w := publish(h, false, `{"slug":"top","name":"x","content":{"uses":[{"fragment":"gone","version":1}]}}`); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("missing: status %d: %s", w.Code, w.Body.String())
	}
	mock.ExpectQuery(`FROM page_fragments WHERE slug`).WillReturnRows(fragRow("mid", 1, `{"uses":[{"fragment":"leaf","version":1}]}`, "h"))
	if w := publish(h, false, `{"slug":"top","name":"x","content":{"uses":[{"fragment":"mid","version":1}]}}`); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("too deep: status %d: %s", w.Code, w.Body.String())
	}
}
