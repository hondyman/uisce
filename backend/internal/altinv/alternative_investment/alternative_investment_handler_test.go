package alternative_investment

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
)

type testAlternativeInvestmentService struct {
	records map[uuid.UUID]AlternativeInvestmentRecord
}

func (s *testAlternativeInvestmentService) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]AlternativeInvestmentRecord, error) {
	var result []AlternativeInvestmentRecord
	for _, rec := range s.records {
		if rec.TenantID == tenantID && (subtypeCode == "" || rec.SubtypeCode == subtypeCode) {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (s *testAlternativeInvestmentService) Get(ctx context.Context, tenantID, id uuid.UUID) (*AlternativeInvestmentRecord, error) {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return nil, nil
	}
	return &rec, nil
}

func (s *testAlternativeInvestmentService) Create(ctx context.Context, tenantID uuid.UUID, rec *AlternativeInvestmentRecord) error {
	rec.TenantID = tenantID
	s.records[rec.ID] = *rec
	return nil
}

func (s *testAlternativeInvestmentService) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return ErrNotFound
	}
	delete(s.records, id)
	return nil
}

func setupTestAlternativeInvestmentRouter(svc *testAlternativeInvestmentService) http.Handler {
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

func TestAlternativeInvestmentHandler_List_Unauthorized(t *testing.T) {
	svc := &testAlternativeInvestmentService{records: make(map[uuid.UUID]AlternativeInvestmentRecord)}
	r := setupTestAlternativeInvestmentRouter(svc)

	req := httptest.NewRequest("GET", "/api/altinv/alternative-investments", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestAlternativeInvestmentHandler_List_WithAuth(t *testing.T) {
	tenantID := uuid.New()
	svc := &testAlternativeInvestmentService{records: make(map[uuid.UUID]AlternativeInvestmentRecord)}
	r := setupTestAlternativeInvestmentRouter(svc)

	rec := AlternativeInvestmentRecord{
		ID:                   uuid.New(),
		TenantID:             tenantID,
		InvestmentID:         uuid.New(),
		ClientID:             uuid.New(),
		InvestmentType:       "Private Equity",
		FundName:             "Test Fund",
		SubtypeCode:          "PRIVATE_EQUITY",
		TotalCommitmentAmount: 1000000,
		UnfundedCommitment:    500000,
		TotalCapitalCalled:    500000,
		TotalDistributions:    100000,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		ValidFrom:            time.Now(),
	}
	svc.records[rec.ID] = rec

	req := httptest.NewRequest("GET", "/api/altinv/alternative-investments", nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result []AlternativeInvestmentRecord
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
}

func TestAlternativeInvestmentHandler_Get_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testAlternativeInvestmentService{records: make(map[uuid.UUID]AlternativeInvestmentRecord)}
	r := setupTestAlternativeInvestmentRouter(svc)

	req := httptest.NewRequest("GET", "/api/altinv/alternative-investments/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestAlternativeInvestmentHandler_Create_InvalidJSON(t *testing.T) {
	tenantID := uuid.New()
	svc := &testAlternativeInvestmentService{records: make(map[uuid.UUID]AlternativeInvestmentRecord)}
	r := setupTestAlternativeInvestmentRouter(svc)

	req := httptest.NewRequest("POST", "/api/altinv/alternative-investments", bytes.NewBufferString("invalid json"))
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAlternativeInvestmentHandler_Delete_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testAlternativeInvestmentService{records: make(map[uuid.UUID]AlternativeInvestmentRecord)}
	r := setupTestAlternativeInvestmentRouter(svc)

	req := httptest.NewRequest("DELETE", "/api/altinv/alternative-investments/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
