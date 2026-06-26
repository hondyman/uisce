package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/models"
)

type mockSemanticSvc struct {
	created []*models.FabricDefn
	err     error
}

func (m *mockSemanticSvc) GenerateDefaultSemanticModel(id uuid.UUID) ([]*models.FabricDefn, error) {
	return m.created, m.err
}

func TestHandleGenerateDefaults_ValidationErrors(t *testing.T) {
	// Missing payload
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/fabric/models/generate-defaults", bytes.NewReader([]byte(`{}`)))
	handleGenerateDefaults(rr, req, &mockSemanticSvc{})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing datasource_id, got %d", rr.Code)
	}
	var er ErrorResponse
	_ = json.NewDecoder(rr.Body).Decode(&er)
	if er.Error == "" {
		t.Fatalf("expected error message in JSON response")
	}

	// Malformed UUID
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/fabric/models/generate-defaults", bytes.NewReader([]byte(`{"datasource_id":"not-a-uuid"}`)))
	handleGenerateDefaults(rr2, req2, &mockSemanticSvc{})
	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid uuid, got %d", rr2.Code)
	}
}

func TestHandleGenerateDefaults_Success(t *testing.T) {
	// Provide a valid UUID and mock service that returns empty slice
	id := uuid.New()
	payload := map[string]string{"datasource_id": id.String()}
	b, _ := json.Marshal(payload)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/fabric/models/generate-defaults", bytes.NewReader(b))
	svc := &mockSemanticSvc{created: []*models.FabricDefn{}}
	handleGenerateDefaults(rr, req, svc)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rr.Code)
	}
}
