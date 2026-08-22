package position

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
	"github.com/shopspring/decimal"
)

type testPositionService struct {
	records map[uuid.UUID]PositionRecord
}

func (s *testPositionService) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]PositionRecord, error) {
	var result []PositionRecord
	for _, rec := range s.records {
		if rec.TenantID == tenantID && (subtypeCode == "" || rec.SubtypeCode == subtypeCode) {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (s *testPositionService) Get(ctx context.Context, tenantID, id uuid.UUID) (*PositionRecord, error) {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return nil, nil
	}
	return &rec, nil
}

func (s *testPositionService) Create(ctx context.Context, tenantID uuid.UUID, rec *PositionRecord) error {
	rec.TenantID = tenantID
	s.records[rec.ID] = *rec
	return nil
}

func (s *testPositionService) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return ErrNotFound
	}
	delete(s.records, id)
	return nil
}

func setupTestPositionRouter(svc *testPositionService) http.Handler {
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

func TestPositionHandler_List_Unauthorized(t *testing.T) {
	svc := &testPositionService{records: make(map[uuid.UUID]PositionRecord)}
	r := setupTestPositionRouter(svc)

	req := httptest.NewRequest("GET", "/api/oms/positions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestPositionHandler_List_WithAuth(t *testing.T) {
	tenantID := uuid.New()
	svc := &testPositionService{records: make(map[uuid.UUID]PositionRecord)}
	r := setupTestPositionRouter(svc)

	rec := PositionRecord{
		ID:          uuid.New(),
		TenantID:    tenantID,
		AccountID:   uuid.New(),
		SecurityID:  uuid.New(),
		Quantity:    decimal.NewFromFloat(100),
		MarketValue: decimal.NewFromFloat(1000),
		Currency:    "USD",
		SubtypeCode: "settled_long",
	}
	svc.records[rec.ID] = rec

	req := httptest.NewRequest("GET", "/api/oms/positions", nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result []PositionRecord
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
}

func TestPositionHandler_Get_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testPositionService{records: make(map[uuid.UUID]PositionRecord)}
	r := setupTestPositionRouter(svc)

	req := httptest.NewRequest("GET", "/api/oms/positions/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestPositionHandler_Create_InvalidJSON(t *testing.T) {
	tenantID := uuid.New()
	svc := &testPositionService{records: make(map[uuid.UUID]PositionRecord)}
	r := setupTestPositionRouter(svc)

	req := httptest.NewRequest("POST", "/api/oms/positions", bytes.NewBufferString("invalid json"))
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestPositionHandler_Delete_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testPositionService{records: make(map[uuid.UUID]PositionRecord)}
	r := setupTestPositionRouter(svc)

	req := httptest.NewRequest("DELETE", "/api/oms/positions/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
