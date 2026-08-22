package security

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

type testSecurityService struct {
	records map[uuid.UUID]SecurityRecord
}

func (s *testSecurityService) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]SecurityRecord, error) {
	var result []SecurityRecord
	for _, rec := range s.records {
		if rec.TenantID == tenantID && (subtypeCode == "" || rec.SubtypeCode == subtypeCode) {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (s *testSecurityService) Get(ctx context.Context, tenantID, id uuid.UUID) (*SecurityRecord, error) {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return nil, nil
	}
	return &rec, nil
}

func (s *testSecurityService) Create(ctx context.Context, tenantID uuid.UUID, rec *SecurityRecord) error {
	rec.TenantID = tenantID
	s.records[rec.ID] = *rec
	return nil
}

func (s *testSecurityService) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return ErrNotFound
	}
	delete(s.records, id)
	return nil
}

func setupTestSecurityRouter(svc *testSecurityService) http.Handler {
	r := chi.NewRouter()
	h := NewHandlerWithService(svc)
	h.RegisterRoutes(r)
	return r
}

func withAuthContext(r *http.Request, tenantID uuid.UUID) *http.Request {
	claims := &jwtmiddleware.JWTClaims{
		TenantID: tenantID.String(),
		UserID:   "test-user",
		Email:    "test@example.com",
		IsActive: true,
	}
	ctx := context.WithValue(r.Context(), jwtmiddleware.ClaimsContextKey, claims)
	return r.WithContext(ctx)
}

func TestSecurityHandler_List_Unauthorized(t *testing.T) {
	svc := &testSecurityService{records: make(map[uuid.UUID]SecurityRecord)}
	r := setupTestSecurityRouter(svc)

	req := httptest.NewRequest("GET", "/api/oms/securities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestSecurityHandler_List_WithAuth(t *testing.T) {
	tenantID := uuid.New()
	svc := &testSecurityService{records: make(map[uuid.UUID]SecurityRecord)}
	r := setupTestSecurityRouter(svc)

	rec := SecurityRecord{
		ID:              uuid.New(),
		TenantID:        tenantID,
		SecurityName:    "Test Security",
		IdentifierType:  "CUSIP",
		IdentifierValue: "123456789",
		SubtypeCode:     "equity",
	}
	svc.records[rec.ID] = rec

	req := httptest.NewRequest("GET", "/api/oms/securities", nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result []SecurityRecord
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
}

func TestSecurityHandler_Get_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testSecurityService{records: make(map[uuid.UUID]SecurityRecord)}
	r := setupTestSecurityRouter(svc)

	req := httptest.NewRequest("GET", "/api/oms/securities/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestSecurityHandler_Create_InvalidJSON(t *testing.T) {
	tenantID := uuid.New()
	svc := &testSecurityService{records: make(map[uuid.UUID]SecurityRecord)}
	r := setupTestSecurityRouter(svc)

	req := httptest.NewRequest("POST", "/api/oms/securities", bytes.NewBufferString("invalid json"))
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSecurityHandler_Delete_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testSecurityService{records: make(map[uuid.UUID]SecurityRecord)}
	r := setupTestSecurityRouter(svc)

	req := httptest.NewRequest("DELETE", "/api/oms/securities/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
