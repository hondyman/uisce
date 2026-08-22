package vendor

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

type testVendorService struct {
	records map[uuid.UUID]VendorRecord
}

func (s *testVendorService) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]VendorRecord, error) {
	var result []VendorRecord
	for _, rec := range s.records {
		if rec.TenantID == tenantID && (subtypeCode == "" || rec.SubtypeCode == subtypeCode) {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (s *testVendorService) Get(ctx context.Context, tenantID, id uuid.UUID) (*VendorRecord, error) {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return nil, nil
	}
	return &rec, nil
}

func (s *testVendorService) Create(ctx context.Context, tenantID uuid.UUID, rec *VendorRecord) error {
	rec.TenantID = tenantID
	s.records[rec.ID] = *rec
	return nil
}

func (s *testVendorService) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return ErrNotFound
	}
	delete(s.records, id)
	return nil
}

func setupTestVendorRouter(svc *testVendorService) http.Handler {
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

func TestVendorHandler_List_Unauthorized(t *testing.T) {
	svc := &testVendorService{records: make(map[uuid.UUID]VendorRecord)}
	r := setupTestVendorRouter(svc)

	req := httptest.NewRequest("GET", "/api/master/vendors", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestVendorHandler_List_WithAuth(t *testing.T) {
	tenantID := uuid.New()
	svc := &testVendorService{records: make(map[uuid.UUID]VendorRecord)}
	r := setupTestVendorRouter(svc)

	rec := VendorRecord{
		ID:           uuid.New(),
		TenantID:     tenantID,
		VendorName:   "Test Vendor",
		SubtypeCode:  "market_data",
	}
	svc.records[rec.ID] = rec

	req := httptest.NewRequest("GET", "/api/master/vendors", nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result []VendorRecord
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
}

func TestVendorHandler_Get_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testVendorService{records: make(map[uuid.UUID]VendorRecord)}
	r := setupTestVendorRouter(svc)

	req := httptest.NewRequest("GET", "/api/master/vendors/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestVendorHandler_Create_InvalidJSON(t *testing.T) {
	tenantID := uuid.New()
	svc := &testVendorService{records: make(map[uuid.UUID]VendorRecord)}
	r := setupTestVendorRouter(svc)

	req := httptest.NewRequest("POST", "/api/master/vendors", bytes.NewBufferString("invalid json"))
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestVendorHandler_Delete_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testVendorService{records: make(map[uuid.UUID]VendorRecord)}
	r := setupTestVendorRouter(svc)

	req := httptest.NewRequest("DELETE", "/api/master/vendors/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
