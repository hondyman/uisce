package settlement

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/shopspring/decimal"
)

type testSettlementService struct {
	records map[uuid.UUID]SettlementRecord
}

func (s *testSettlementService) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]SettlementRecord, error) {
	var result []SettlementRecord
	for _, rec := range s.records {
		if rec.TenantID == tenantID && (subtypeCode == "" || rec.SubtypeCode == subtypeCode) {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (s *testSettlementService) Get(ctx context.Context, tenantID, id uuid.UUID) (*SettlementRecord, error) {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return nil, nil
	}
	return &rec, nil
}

func (s *testSettlementService) Create(ctx context.Context, tenantID uuid.UUID, rec *SettlementRecord) error {
	rec.TenantID = tenantID
	s.records[rec.ID] = *rec
	return nil
}

func (s *testSettlementService) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return ErrNotFound
	}
	delete(s.records, id)
	return nil
}

func setupTestSettlementRouter(svc *testSettlementService) http.Handler {
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

func TestSettlementHandler_List_Unauthorized(t *testing.T) {
	svc := &testSettlementService{records: make(map[uuid.UUID]SettlementRecord)}
	r := setupTestSettlementRouter(svc)

	req := httptest.NewRequest("GET", "/api/cash-flow/settlements", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestSettlementHandler_List_WithAuth(t *testing.T) {
	tenantID := uuid.New()
	svc := &testSettlementService{records: make(map[uuid.UUID]SettlementRecord)}
	r := setupTestSettlementRouter(svc)

	rec := SettlementRecord{
		ID:              uuid.New(),
		TenantID:        tenantID,
		AccountID:       uuid.New(),
		Amount:          decimal.NewFromFloat(100),
		Currency:        "USD",
		SettlementDate:  time.Now(),
		SettlementStatus: "pending",
		SubtypeCode:     "dividend",
	}
	svc.records[rec.ID] = rec

	req := httptest.NewRequest("GET", "/api/cash-flow/settlements", nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result []SettlementRecord
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
}

func TestSettlementHandler_Get_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testSettlementService{records: make(map[uuid.UUID]SettlementRecord)}
	r := setupTestSettlementRouter(svc)

	req := httptest.NewRequest("GET", "/api/cash-flow/settlements/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestSettlementHandler_Create_InvalidJSON(t *testing.T) {
	tenantID := uuid.New()
	svc := &testSettlementService{records: make(map[uuid.UUID]SettlementRecord)}
	r := setupTestSettlementRouter(svc)

	req := httptest.NewRequest("POST", "/api/cash-flow/settlements", bytes.NewBufferString("invalid json"))
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSettlementHandler_Delete_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testSettlementService{records: make(map[uuid.UUID]SettlementRecord)}
	r := setupTestSettlementRouter(svc)

	req := httptest.NewRequest("DELETE", "/api/cash-flow/settlements/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
