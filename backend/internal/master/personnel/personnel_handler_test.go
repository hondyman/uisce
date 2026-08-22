package personnel

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

type testPersonnelService struct {
	records map[uuid.UUID]PersonnelRecord
}

func (s *testPersonnelService) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]PersonnelRecord, error) {
	var result []PersonnelRecord
	for _, rec := range s.records {
		if rec.TenantID == tenantID && (subtypeCode == "" || rec.SubtypeCode == subtypeCode) {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (s *testPersonnelService) Get(ctx context.Context, tenantID, id uuid.UUID) (*PersonnelRecord, error) {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return nil, nil
	}
	return &rec, nil
}

func (s *testPersonnelService) Create(ctx context.Context, tenantID uuid.UUID, rec *PersonnelRecord) error {
	rec.TenantID = tenantID
	s.records[rec.ID] = *rec
	return nil
}

func (s *testPersonnelService) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return ErrNotFound
	}
	delete(s.records, id)
	return nil
}

func setupTestPersonnelRouter(svc *testPersonnelService) http.Handler {
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

func TestPersonnelHandler_List_Unauthorized(t *testing.T) {
	svc := &testPersonnelService{records: make(map[uuid.UUID]PersonnelRecord)}
	r := setupTestPersonnelRouter(svc)

	req := httptest.NewRequest("GET", "/api/master/personnel", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestPersonnelHandler_List_WithAuth(t *testing.T) {
	tenantID := uuid.New()
	svc := &testPersonnelService{records: make(map[uuid.UUID]PersonnelRecord)}
	r := setupTestPersonnelRouter(svc)

	rec := PersonnelRecord{
		ID:        uuid.New(),
		TenantID:  tenantID,
		FullName:  "John Doe",
		Email:     "john@example.com",
		SubtypeCode: "portfolio_manager",
	}
	svc.records[rec.ID] = rec

	req := httptest.NewRequest("GET", "/api/master/personnel", nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result []PersonnelRecord
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
}

func TestPersonnelHandler_Get_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testPersonnelService{records: make(map[uuid.UUID]PersonnelRecord)}
	r := setupTestPersonnelRouter(svc)

	req := httptest.NewRequest("GET", "/api/master/personnel/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestPersonnelHandler_Create_InvalidJSON(t *testing.T) {
	tenantID := uuid.New()
	svc := &testPersonnelService{records: make(map[uuid.UUID]PersonnelRecord)}
	r := setupTestPersonnelRouter(svc)

	req := httptest.NewRequest("POST", "/api/master/personnel", bytes.NewBufferString("invalid json"))
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestPersonnelHandler_Delete_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testPersonnelService{records: make(map[uuid.UUID]PersonnelRecord)}
	r := setupTestPersonnelRouter(svc)

	req := httptest.NewRequest("DELETE", "/api/master/personnel/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
