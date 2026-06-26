package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestValidateModelHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	svc := analytics.NewSemanticModelService(sqlxDB)
	h := NewSemanticModelHandler(svc)

	// Fabric defn returned by GetModelDefinition
	resolvedJSON := []byte(`{"cubes":[{"name":"base_cube","dimensions":{},"measures":{},"joins":{}}]}`)
	rows := sqlmock.NewRows([]string{"id", "tenant_id", "tenant_datasource_id", "model_key", "version", "status", "title", "description", "source_config", "resolved_config", "created_by", "is_current"}).AddRow(uuid.New(), uuid.New(), uuid.New(), "/base", 1, "draft", "base", "", "{}", string(resolvedJSON), uuid.Nil, true)
	mock.ExpectQuery(`SELECT \* FROM public.fabric_defn WHERE tenant_datasource_id = \$1 AND model_key = \$2 AND is_current = true`).WithArgs(sqlmock.AnyArg(), "/base").WillReturnRows(rows)

	// GatherColumnsMapForDatasource select catalog_node
	mock.ExpectQuery(`SELECT node_name FROM public.catalog_node WHERE tenant_datasource_id = \$1 AND node_type_id = \$2`).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"node_name"}).AddRow("id").AddRow("present_col"))

	// Build request body similar to SaveExtensionRequest
	payload := map[string]any{
		"base_model_key": "/base",
		"model_object": map[string]any{
			"name":       "ext",
			"dimensions": map[string]any{"d1": map[string]any{"sql": "missing_col"}},
		},
	}
	b, _ := json.Marshal(payload)

	// Create request
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/?datasource_id="+uuid.New().String(), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	h.ValidateModel(w, req)

	require.Equal(t, 200, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	_, ok := resp["issues"]
	require.True(t, ok)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestValidateModelHandler_MissingBaseKey(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	svc := analytics.NewSemanticModelService(sqlxDB)
	h := NewSemanticModelHandler(svc)

	// Build request body without base_model_key and without extends in model_object
	payload := map[string]any{
		"model_object": map[string]any{"name": "ext"},
	}
	b, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/?datasource_id="+uuid.New().String(), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	h.ValidateModel(w, req)

	require.Equal(t, 400, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Contains(t, resp["error"], "base_model_key is required")
}

func TestValidateModelHandler_MalformedModelObject(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	svc := analytics.NewSemanticModelService(sqlxDB)
	h := NewSemanticModelHandler(svc)

	// model_object is a string which should fail unmarshalling into cube.Cube
	payload := map[string]any{
		"base_model_key": "/base",
		"model_object":   "not-an-object",
	}
	b, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/?datasource_id="+uuid.New().String(), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	h.ValidateModel(w, req)

	require.Equal(t, 400, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// Binding may fail earlier (e.g. request body model_object is not an object) and
	// return a generic "Invalid request body" message, or unmarshalling may fail with
	// "Invalid model_object structure". Accept either as valid behavior for this edge case.
	errMsg := fmt.Sprintf("%v", resp["error"])
	require.True(t, strings.Contains(errMsg, "Invalid model_object structure") || strings.Contains(errMsg, "Invalid request body"), "unexpected error: %s", errMsg)
}
