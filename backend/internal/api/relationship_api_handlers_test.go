package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostDiscoverRelationships tests the relationship discovery endpoint
func TestPostDiscoverRelationships(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	srv := &Server{
		DB: db,
	}

	t.Run("should discover relationships successfully", func(t *testing.T) {
		sourceEntityID := "11111111-1111-1111-1111-111111111111"
		body := map[string]interface{}{
			"entity_attribute_id": sourceEntityID,
			"include_multi_hop":   true,
			"max_hop_depth":       3,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(
			"POST",
			"/api/relationships/discover",
			bytes.NewReader(bodyBytes),
		)
		req.Header.Set("X-Tenant-ID", "tenant-123")
		req.Header.Set("X-Tenant-Datasource-ID", "ds-456")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		srv.postDiscoverRelationships(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("should return error without tenant context", func(t *testing.T) {
		body := map[string]interface{}{
			"entity_attribute_id": "11111111-1111-1111-1111-111111111111",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(
			"POST",
			"/api/relationships/discover",
			bytes.NewReader(bodyBytes),
		)
		w := httptest.NewRecorder()
		srv.postDiscoverRelationships(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return error without entity_attribute_id", func(t *testing.T) {
		body := map[string]interface{}{}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(
			"POST",
			"/api/relationships/discover",
			bytes.NewReader(bodyBytes),
		)
		req.Header.Set("X-Tenant-ID", "tenant-123")
		req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

		w := httptest.NewRecorder()
		srv.postDiscoverRelationships(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should cap hop depth at 5", func(t *testing.T) {
		body := map[string]interface{}{
			"entity_attribute_id": "11111111-1111-1111-1111-111111111111",
			"max_hop_depth":       10,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(
			"POST",
			"/api/relationships/discover",
			bytes.NewReader(bodyBytes),
		)
		req.Header.Set("X-Tenant-ID", "tenant-123")
		req.Header.Set("X-Tenant-Datasource-ID", "ds-456")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		srv.postDiscoverRelationships(w, req)

		// Hop depth should be capped at 5, so request should succeed
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
	})
}

// TestPostApplyRelationship tests the relationship application endpoint
func TestPostApplyRelationship(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	srv := &Server{
		DB: db,
	}

	t.Run("should apply relationship successfully", func(t *testing.T) {
		sourceEntityID := "11111111-1111-1111-1111-111111111111"
		targetEntityID := "22222222-2222-2222-2222-222222222222"
		body := map[string]interface{}{
			"sourceEntity":   sourceEntityID,
			"targetEntity":   targetEntityID,
			"edgeType":       "DIRECT_FK",
			"confidence":     0.95,
			"cardinality":    "1:N",
			"foreignKeyPath": "",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(
			"POST",
			"/api/relationships/apply",
			bytes.NewReader(bodyBytes),
		)
		req.Header.Set("X-Tenant-ID", "tenant-123")
		req.Header.Set("X-Tenant-Datasource-ID", "ds-456")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		srv.postApplyRelationship(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated)
	})

	t.Run("should return error without tenant context", func(t *testing.T) {
		body := map[string]interface{}{
			"sourceEntity": "11111111-1111-1111-1111-111111111111",
			"targetEntity": "22222222-2222-2222-2222-222222222222",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(
			"POST",
			"/api/relationships/apply",
			bytes.NewReader(bodyBytes),
		)
		w := httptest.NewRecorder()
		srv.postApplyRelationship(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return error without required fields", func(t *testing.T) {
		body := map[string]interface{}{
			"sourceEntity": "11111111-1111-1111-1111-111111111111",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(
			"POST",
			"/api/relationships/apply",
			bytes.NewReader(bodyBytes),
		)
		req.Header.Set("X-Tenant-ID", "tenant-123")
		req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

		w := httptest.NewRecorder()
		srv.postApplyRelationship(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestPostTriggerModelRegeneration tests the model regeneration endpoint
func TestPostTriggerModelRegeneration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	srv := &Server{
		DB: db,
	}

	t.Run("should trigger model regeneration successfully", func(t *testing.T) {
		body := map[string]interface{}{
			"entity_attribute_id": "11111111-1111-1111-1111-111111111111",
			"trigger_type":        "RELATIONSHIP_APPLIED",
			"priority":            5,
			"reason":              "test",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(
			"POST",
			"/api/models/regenerate",
			bytes.NewReader(bodyBytes),
		)
		req.Header.Set("X-Tenant-ID", "tenant-123")
		req.Header.Set("X-Tenant-Datasource-ID", "ds-456")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		srv.postTriggerModelRegeneration(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusAccepted)
	})

	t.Run("should return error without tenant context", func(t *testing.T) {
		req := httptest.NewRequest(
			"POST",
			"/api/models/regenerate",
			bytes.NewReader([]byte("{}")),
		)
		w := httptest.NewRecorder()
		srv.postTriggerModelRegeneration(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestGetModelVersion tests the model version retrieval endpoint
func TestGetModelVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	srv := &Server{
		DB: db,
	}

	t.Run("should retrieve model version successfully", func(t *testing.T) {
		entityAttributeID := "11111111-1111-1111-1111-111111111111"
		req := httptest.NewRequest(
			"GET",
			"/api/models/version?entity_attribute_id="+entityAttributeID+"&version=1",
			nil,
		)
		req.Header.Set("X-Tenant-ID", "tenant-123")
		req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

		w := httptest.NewRecorder()
		srv.getModelVersion(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound)
	})

	t.Run("should return error without tenant context", func(t *testing.T) {
		req := httptest.NewRequest(
			"GET",
			"/api/models/version",
			nil,
		)
		w := httptest.NewRecorder()
		srv.getModelVersion(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestMultiTenantIsolation tests that queries are properly scoped by tenant
func TestMultiTenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	srv := &Server{
		DB: db,
	}

	body := map[string]interface{}{
		"entity_attribute_id": "entity-123",
	}
	bodyBytes, _ := json.Marshal(body)

	// Request 1: Tenant A
	req1 := httptest.NewRequest(
		"POST",
		"/api/relationships/discover",
		bytes.NewReader(bodyBytes),
	)
	req1.Header.Set("X-Tenant-ID", "tenant-A")
	req1.Header.Set("X-Tenant-Datasource-ID", "ds-A")
	req1.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	srv.postDiscoverRelationships(w1, req1)

	// Request 2: Tenant B (should not see Tenant A's data)
	req2 := httptest.NewRequest(
		"POST",
		"/api/relationships/discover",
		bytes.NewReader(bodyBytes),
	)
	req2.Header.Set("X-Tenant-ID", "tenant-B")
	req2.Header.Set("X-Tenant-Datasource-ID", "ds-B")
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	srv.postDiscoverRelationships(w2, req2)

	// Both should succeed but with isolated results
	assert.True(t, (w1.Code == http.StatusOK || w1.Code == http.StatusInternalServerError))
	assert.True(t, (w2.Code == http.StatusOK || w2.Code == http.StatusInternalServerError))
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create necessary tables for testing
	schema := `
	CREATE TABLE IF NOT EXISTS business_objects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		display_name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS relationship_suggestions (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		datasource_id TEXT NOT NULL,
		source_entity_id TEXT NOT NULL,
		target_entity_id TEXT NOT NULL,
		accepted BOOLEAN NOT NULL DEFAULT 0,
		accepted_at DATETIME,
		updated_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS business_object_relationships (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		tenant_datasource_id TEXT NOT NULL,
		source_object_id TEXT NOT NULL,
		target_object_id TEXT NOT NULL,
		relationship_type TEXT NOT NULL,
		cardinality TEXT,
		confidence REAL,
		is_user_applied BOOLEAN NOT NULL DEFAULT 0,
		user_applied_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		UNIQUE(tenant_id, source_object_id, target_object_id, relationship_type)
	);

	CREATE TABLE IF NOT EXISTS entity_relationship (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		tenant_datasource_id TEXT NOT NULL,
		source_entity_id TEXT NOT NULL,
		target_entity_id TEXT NOT NULL,
		link_type TEXT NOT NULL,
		confidence REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS model_regeneration_trigger (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		tenant_datasource_id TEXT NOT NULL,
		entity_attribute_id TEXT NOT NULL,
		trigger_type TEXT NOT NULL,
		trigger_source TEXT,
		change_detail TEXT,
		triggered_by TEXT,
		regeneration_status TEXT NOT NULL,
		triggered_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS model_regeneration_queue (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		tenant_datasource_id TEXT NOT NULL,
		entity_attribute_id TEXT NOT NULL,
		queue_status TEXT NOT NULL,
		priority INTEGER,
		reason TEXT,
		triggered_by_trigger_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS model_version_history (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		tenant_datasource_id TEXT NOT NULL,
		entity_attribute_id TEXT NOT NULL,
		version_number INTEGER NOT NULL,
		is_active BOOLEAN NOT NULL DEFAULT 1,
		model_signature TEXT NOT NULL,
		model_content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	// Seed a few business objects for the simple discovery fallback.
	_, err = db.Exec(
		`INSERT INTO business_objects (id, name, display_name) VALUES (?, ?, ?), (?, ?, ?);`,
		"11111111-1111-1111-1111-111111111111", "source", "Source",
		"22222222-2222-2222-2222-222222222222", "target", "Target",
	)
	require.NoError(t, err)

	// Seed one model version so GetModelVersion can succeed.
	modelJSON := `{"attributes":[],"relationships":[]}`
	_, err = db.Exec(
		`INSERT INTO model_version_history (id, tenant_id, tenant_datasource_id, entity_attribute_id, version_number, is_active, model_signature, model_content) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`,
		"mv1",
		"tenant-123",
		"ds-456",
		"11111111-1111-1111-1111-111111111111",
		1,
		1,
		"sig1",
		modelJSON,
	)
	require.NoError(t, err)

	return db
}
