package customer

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

type testCustomerService struct {
	records map[uuid.UUID]CustomerRecord
}

func (s *testCustomerService) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]CustomerRecord, error) {
	var result []CustomerRecord
	for _, rec := range s.records {
		if rec.TenantID == tenantID && (subtypeCode == "" || rec.SubtypeCode == subtypeCode) {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (s *testCustomerService) Get(ctx context.Context, tenantID, id uuid.UUID) (*CustomerRecord, error) {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return nil, nil
	}
	return &rec, nil
}

func (s *testCustomerService) Create(ctx context.Context, tenantID uuid.UUID, rec *CustomerRecord) error {
	rec.TenantID = tenantID
	s.records[rec.ID] = *rec
	return nil
}

func (s *testCustomerService) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return ErrNotFound
	}
	delete(s.records, id)
	return nil
}

func setupTestCustomerRouter(svc *testCustomerService) http.Handler {
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

func TestCustomerHandler_List_Unauthorized(t *testing.T) {
	svc := &testCustomerService{records: make(map[uuid.UUID]CustomerRecord)}
	r := setupTestCustomerRouter(svc)

	req := httptest.NewRequest("GET", "/api/master/customers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestCustomerHandler_List_WithAuth(t *testing.T) {
	tenantID := uuid.New()
	svc := &testCustomerService{records: make(map[uuid.UUID]CustomerRecord)}
	r := setupTestCustomerRouter(svc)

	rec := CustomerRecord{
		ID:           uuid.New(),
		TenantID:     tenantID,
		CustomerName: "Test Customer",
		SubtypeCode:  "institutional_client",
		KYCStatus:    "approved",
	}
	svc.records[rec.ID] = rec

	req := httptest.NewRequest("GET", "/api/master/customers", nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result []CustomerRecord
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
}

func TestCustomerHandler_Get_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testCustomerService{records: make(map[uuid.UUID]CustomerRecord)}
	r := setupTestCustomerRouter(svc)

	req := httptest.NewRequest("GET", "/api/master/customers/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestCustomerHandler_Create_InvalidJSON(t *testing.T) {
	tenantID := uuid.New()
	svc := &testCustomerService{records: make(map[uuid.UUID]CustomerRecord)}
	r := setupTestCustomerRouter(svc)

	req := httptest.NewRequest("POST", "/api/master/customers", bytes.NewBufferString("invalid json"))
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCustomerHandler_Delete_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testCustomerService{records: make(map[uuid.UUID]CustomerRecord)}
	r := setupTestCustomerRouter(svc)

	req := httptest.NewRequest("DELETE", "/api/master/customers/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
