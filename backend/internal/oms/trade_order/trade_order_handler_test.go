package trade_order

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

type testTradeOrderService struct {
	records map[uuid.UUID]TradeOrderRecord
}

func (s *testTradeOrderService) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]TradeOrderRecord, error) {
	var result []TradeOrderRecord
	for _, rec := range s.records {
		if rec.TenantID == tenantID && (subtypeCode == "" || rec.SubtypeCode == subtypeCode) {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (s *testTradeOrderService) Get(ctx context.Context, tenantID, id uuid.UUID) (*TradeOrderRecord, error) {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return nil, nil
	}
	return &rec, nil
}

func (s *testTradeOrderService) Create(ctx context.Context, tenantID uuid.UUID, rec *TradeOrderRecord) error {
	rec.TenantID = tenantID
	s.records[rec.ID] = *rec
	return nil
}

func (s *testTradeOrderService) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return ErrNotFound
	}
	delete(s.records, id)
	return nil
}

func setupTestTradeOrderRouter(svc *testTradeOrderService) http.Handler {
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

func TestTradeOrderHandler_List_Unauthorized(t *testing.T) {
	svc := &testTradeOrderService{records: make(map[uuid.UUID]TradeOrderRecord)}
	r := setupTestTradeOrderRouter(svc)

	req := httptest.NewRequest("GET", "/api/oms/trade-orders", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestTradeOrderHandler_List_WithAuth(t *testing.T) {
	tenantID := uuid.New()
	svc := &testTradeOrderService{records: make(map[uuid.UUID]TradeOrderRecord)}
	r := setupTestTradeOrderRouter(svc)

	rec := TradeOrderRecord{
		ID:              uuid.New(),
		TenantID:        tenantID,
		AccountID:       uuid.New(),
		SecurityID:      uuid.New(),
		OrderSide:       "BUY",
		OrderedQuantity: decimal.NewFromFloat(100),
		OrderStatus:     "pending",
		SubtypeCode:     "dma_execution",
	}
	svc.records[rec.ID] = rec

	req := httptest.NewRequest("GET", "/api/oms/trade-orders", nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result []TradeOrderRecord
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
}

func TestTradeOrderHandler_Get_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testTradeOrderService{records: make(map[uuid.UUID]TradeOrderRecord)}
	r := setupTestTradeOrderRouter(svc)

	req := httptest.NewRequest("GET", "/api/oms/trade-orders/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestTradeOrderHandler_Create_InvalidJSON(t *testing.T) {
	tenantID := uuid.New()
	svc := &testTradeOrderService{records: make(map[uuid.UUID]TradeOrderRecord)}
	r := setupTestTradeOrderRouter(svc)

	req := httptest.NewRequest("POST", "/api/oms/trade-orders", bytes.NewBufferString("invalid json"))
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestTradeOrderHandler_Delete_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testTradeOrderService{records: make(map[uuid.UUID]TradeOrderRecord)}
	r := setupTestTradeOrderRouter(svc)

	req := httptest.NewRequest("DELETE", "/api/oms/trade-orders/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
