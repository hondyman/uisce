package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func setupLookupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	schema := `
	CREATE TABLE IF NOT EXISTS tenants (
		id TEXT PRIMARY KEY,
		gold_copy BOOLEAN DEFAULT FALSE
	);

	CREATE TABLE IF NOT EXISTS lookups (
        id TEXT PRIMARY KEY,
        tenant_id TEXT NOT NULL,
        name TEXT NOT NULL,
        description TEXT,
		source_table TEXT NULL,
        created_at DATETIME,
        updated_at DATETIME
    );

    CREATE TABLE IF NOT EXISTS lookup_values (
        id TEXT PRIMARY KEY,
        lookup_id TEXT NOT NULL,
        tenant_id TEXT NOT NULL,
        value TEXT NOT NULL,
        label TEXT NOT NULL,
        parent_id TEXT NULL,
        metadata TEXT NULL,
        created_at DATETIME
    );
    `
	_, err = db.Exec(schema)
	require.NoError(t, err)

	// Insert a test tenant
	_, err = db.Exec(`INSERT INTO tenants (id, gold_copy) VALUES ('t1', 0)`)
	require.NoError(t, err)

	return db
}

func TestListLookupsPaginationAndSearch(t *testing.T) {
	db := setupLookupTestDB(t)
	defer db.Close()

	// Seed a few lookups
	_, err := db.Exec(`INSERT INTO lookups (id, tenant_id, name, description, created_at, updated_at) VALUES ('l1', 't1','iso_countries','Countries', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP), ('l2', 't1','iso_currencies','Currencies', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP), ('l3', 't1','domains','Domains', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)

	r := chi.NewRouter()
	RegisterLookupsRoutes(r, db)

	// Test pagination (limit 2): should return two items and a next_cursor that's > 0
	req := httptest.NewRequest(http.MethodGet, "/lookups?tenant_id=t1&limit=2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 but got %d - body: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Contains(t, body, "items")
	require.Contains(t, body, "next_cursor")

	// Test search (q=iso) should include iso_countries and iso_currencies
	req2 := httptest.NewRequest(http.MethodGet, "/lookups?tenant_id=t1&q=iso", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	var body2 map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &body2))
	items2 := body2["items"].([]interface{})
	require.GreaterOrEqual(t, len(items2), 2)
}

func TestGetLookupValues(t *testing.T) {
	db := setupLookupTestDB(t)
	defer db.Close()

	// Insert a lookup and some values
	_, err := db.Exec(`INSERT INTO lookups (id, tenant_id, name, created_at, updated_at) VALUES ('lu1', 't1', 'test_lu', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO lookup_values (id, lookup_id, tenant_id, value, label, created_at) VALUES ('v1','lu1','t1','us','United States', CURRENT_TIMESTAMP),('v2','lu1','t1','ca','Canada', CURRENT_TIMESTAMP)`)
	require.NoError(t, err)

	r := chi.NewRouter()
	RegisterLookupsRoutes(r, db)

	req := httptest.NewRequest(http.MethodGet, "/lookups/lu1/values?tenant_id=t1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	items, ok := resp["items"].([]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(items))
}

func TestLookupCRUD(t *testing.T) {
	db := setupLookupTestDB(t)
	defer db.Close()

	r := chi.NewRouter()
	RegisterLookupsRoutes(r, db)

	// Create a lookup
	body := strings.NewReader(`{"name":"test_lookup","description":"desc"}`)
	req := httptest.NewRequest(http.MethodPost, "/lookups?tenant_id=t1", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var created Lookup
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.Equal(t, "test_lookup", created.Name)

	// Update
	upBody := strings.NewReader(`{"name":"test_lookup2","description":"desc2"}`)
	req2 := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/lookups/%s?tenant_id=t1", created.ID), upBody)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusNoContent, w2.Code)

	// List and ensure updated name present
	req3 := httptest.NewRequest(http.MethodGet, "/lookups?tenant_id=t1", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &resp))
	items := resp["items"].([]interface{})
	found := false
	for _, it := range items {
		m := it.(map[string]interface{})
		if m["name"] == "test_lookup2" {
			found = true
		}
	}
	require.True(t, found)

	// Create a lookup value
	valBody := strings.NewReader(`{"value":"v1","label":"Label1"}`)
	req4 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/lookups/%s/values?tenant_id=t1", created.ID), valBody)
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	require.Equal(t, http.StatusOK, w4.Code)
	var v LookupValue
	require.NoError(t, json.Unmarshal(w4.Body.Bytes(), &v))
	require.Equal(t, "v1", v.Value)

	// Delete value
	req5 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/lookups/%s/values/%s?tenant_id=t1", created.ID, v.ID), nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	require.Equal(t, http.StatusNoContent, w5.Code)

	// Delete lookup
	req6 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/lookups/%s?tenant_id=t1", created.ID), nil)
	w6 := httptest.NewRecorder()
	r.ServeHTTP(w6, req6)
	require.Equal(t, http.StatusNoContent, w6.Code)
}

func TestGetLookupValuesWithParent(t *testing.T) {
	db := setupLookupTestDB(t)
	defer db.Close()

	// Insert hierarchical data: parent -> child
	_, err := db.Exec(`INSERT INTO lookups (id, tenant_id, name, created_at, updated_at) VALUES ('lu2', 't1', 'domains', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	// Insert parent
	_, err = db.Exec(`INSERT INTO lookup_values (id, lookup_id, tenant_id, value, label, created_at) VALUES ('p1','lu2','t1','finance','Finance', CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	// Insert child referencing parent id
	_, err = db.Exec(`INSERT INTO lookup_values (id, lookup_id, tenant_id, value, label, parent_id, created_at) VALUES ('c1','lu2','t1','capital_markets','Capital Markets','p1', CURRENT_TIMESTAMP)`)
	require.NoError(t, err)

	r := chi.NewRouter()
	RegisterLookupsRoutes(r, db)

	req := httptest.NewRequest(http.MethodGet, "/lookups/lu2/values?tenant_id=t1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	items, ok := resp["items"].([]interface{})
	require.True(t, ok)
	// Default request should return top-level items (parent rows) only
	require.GreaterOrEqual(t, len(items), 1)

	// Now fetch child items by parent_id and assert we receive the child
	reqChild := httptest.NewRequest(http.MethodGet, "/lookups/lu2/values?tenant_id=t1&parent_id=p1", nil)
	wChild := httptest.NewRecorder()
	r.ServeHTTP(wChild, reqChild)
	require.Equal(t, http.StatusOK, wChild.Code)
	var respChild map[string]interface{}
	require.NoError(t, json.Unmarshal(wChild.Body.Bytes(), &respChild))
	childItems := respChild["items"].([]interface{})
	require.GreaterOrEqual(t, len(childItems), 1)
	// Ensure child has parent_id pointing to p1
	var foundChild bool
	for _, it := range childItems {
		v := it.(map[string]interface{})
		if v["value"] == "capital_markets" {
			foundChild = true
			require.Equal(t, "p1", v["parent_id"])
		}
	}
	require.True(t, foundChild)
}

func TestGetLookupValuesFromSourceTable(t *testing.T) {
	db := setupLookupTestDB(t)
	defer db.Close()

	// Create a source table emulating a catalog table (id, tenant_id, name, parent_id)
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS dept_source (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL, parent_id TEXT NULL)`)
	require.NoError(t, err)

	// Insert sample rows: parent p1 and child c1
	_, err = db.Exec(`INSERT INTO dept_source (id, tenant_id, name) VALUES ('p1', 't1', 'Parent'), ('p2', 't1', 'Another Parent')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO dept_source (id, tenant_id, name, parent_id) VALUES ('c1', 't1', 'Child', 'p1')`)
	require.NoError(t, err)

	// Create a lookup that references the source table
	_, err = db.Exec(`INSERT INTO lookups (id, tenant_id, name, source_table, created_at, updated_at) VALUES ('l_src', 't1', 'dept_lookup', 'dept_source', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)

	r := chi.NewRouter()
	RegisterLookupsRoutes(r, db)

	// Default request - should return top-level items (parents)
	req := httptest.NewRequest(http.MethodGet, "/lookups/l_src/values?tenant_id=t1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	items := resp["items"].([]interface{})
	require.GreaterOrEqual(t, len(items), 2)

	// Fetch child items by parent_id
	req2 := httptest.NewRequest(http.MethodGet, "/lookups/l_src/values?tenant_id=t1&parent_id=p1", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	var resp2 map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	childItems := resp2["items"].([]interface{})
	require.GreaterOrEqual(t, len(childItems), 1)

	// Fetch child items by parent_value (name) should also resolve and return the child
	req3 := httptest.NewRequest(http.MethodGet, "/lookups/l_src/values?tenant_id=t1&parent_value=Parent", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code)
	var resp3 map[string]interface{}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &resp3))
	childItems2 := resp3["items"].([]interface{})
	require.GreaterOrEqual(t, len(childItems2), 1)
}
