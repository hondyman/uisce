package handlers

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

// jsonArg matches a jsonb argument by JSON value, whatever its Go type.
type jsonArg struct{ want string }

func (a jsonArg) Match(v driver.Value) bool {
	var b []byte
	switch x := v.(type) {
	case []byte:
		b = x
	case string:
		b = []byte(x)
	default:
		return false
	}
	var got, want interface{}
	if json.Unmarshal(b, &got) != nil || json.Unmarshal([]byte(a.want), &want) != nil {
		return false
	}
	gb, _ := json.Marshal(got)
	wb, _ := json.Marshal(want)
	return bytes.Equal(gb, wb)
}

func pageRow(app interface{}) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"id", "tenant_id", "name", "slug", "description", "layout", "tabs", "components", "data_sources",
		"presentation_events", "filter_bar", "app_model", "version", "is_core", "status", "created_at", "updated_at",
	}).AddRow(uuid.New(), uuid.New(), "Console", "console", "", []byte(`{}`), []byte(`[]`), []byte(`{}`), []byte(`[]`),
		[]byte(`[]`), []byte(`{}`), app, 1, false, "draft", now, now)
}

func createRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/page-studio/pages", bytes.NewBufferString(body))
	return req.WithContext(security.WithAuthInfo(req.Context(), security.AuthInfo{TenantIDs: []string{uuid.NewString()}}))
}

// The page application model (variables, queries, tab state) is written on
// create and comes back on the response - it must never be silently
// dropped the way tabs and filterBar once were.
func TestPageStudio_Create_RoundTripsAppModel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := `{"variables":[{"name":"entity"}],"tabVariable":"tab","chrome":"none"}`
	mock.ExpectQuery(`INSERT INTO page_definitions .*app_model`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Console", "console", "", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 1, false, "draft", jsonArg{app}).
		WillReturnRows(pageRow([]byte(app)))

	h := &PageStudioHandler{db: sqlx.NewDb(db, "sqlmock")}
	w := httptest.NewRecorder()
	h.create(w, createRequest(t, `{"name":"Console","slug":"console","app":`+app+`}`))
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !(jsonArg{app}).Match(driver.Value([]byte(got["app"]))) {
		t.Fatalf("app not returned: %s", got["app"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// A plain Business Object page has no app model: stored as SQL NULL and
// omitted from the response, not written as the jsonb value null.
func TestPageStudio_Create_NoAppModelIsNull(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`INSERT INTO page_definitions`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Console", "console", "", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 1, false, "draft", nil).
		WillReturnRows(pageRow(nil))

	h := &PageStudioHandler{db: sqlx.NewDb(db, "sqlmock")}
	w := httptest.NewRecorder()
	h.create(w, createRequest(t, `{"name":"Console","slug":"console","app":null}`))
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte(`"app"`)) {
		t.Fatalf("expected no app key: %s", w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
