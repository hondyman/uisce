package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/rules"
)

type mockRefFetcher struct{}

func (m *mockRefFetcher) GetPortfolioReferenceState(ctx context.Context, tenantID uuid.UUID, portfolioID string, isin string) (map[string]any, error) {
	return map[string]any{
		"portfolio.total_aum":            10000000.0,
		"position.current_market_value":  2100000.0,
	}, nil
}

func (m *mockRefFetcher) GetExternalMapping(ctx context.Context, tenantID uuid.UUID, systemID string) (map[string]string, error) {
	return map[string]string{
		"account_num":   "account.id",
		"security_isin": "security.isin",
		"order_qty":     "order.quantity",
		"order_px":      "order.price",
	}, nil
}

func (m *mockRefFetcher) GetRuleChain(ctx context.Context, tenantID uuid.UUID, chainID string) (*rules.RuleChain, error) {
	return nil, nil
}

func TestHandleEvaluateExternal_MissingTenantID(t *testing.T) {
	handler := api.NewExternalComplianceHandler(nil, &mockRefFetcher{}, nil, nil, "")

	reqBody, _ := json.Marshal(api.ExternalEvaluateRequest{
		SystemIdentifier: "BLOOMBERG_EMS",
		PortfolioID:      "PT-88120",
		ProposedTrade:   map[string]any{},
	})

	req := httptest.NewRequest("POST", "/api/v1/compliance/external/evaluate-external", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()

	handler.HandleEvaluateExternal(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleEvaluateExternal_InvalidTenantID(t *testing.T) {
	handler := api.NewExternalComplianceHandler(nil, &mockRefFetcher{}, nil, nil, "")

	reqBody, _ := json.Marshal(api.ExternalEvaluateRequest{
		SystemIdentifier: "BLOOMBERG_EMS",
		PortfolioID:      "PT-88120",
		ProposedTrade:   map[string]any{},
	})

	req := httptest.NewRequest("POST", "/api/v1/compliance/external/evaluate-external", bytes.NewReader(reqBody))
	req.Header.Set("X-Tenant-ID", "not-a-uuid")
	rr := httptest.NewRecorder()

	handler.HandleEvaluateExternal(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleEvaluateExternal_InvalidJSON(t *testing.T) {
	handler := api.NewExternalComplianceHandler(nil, &mockRefFetcher{}, nil, nil, "")

	req := httptest.NewRequest("POST", "/api/v1/compliance/external/evaluate-external", bytes.NewReader([]byte("not json")))
	req.Header.Set("X-Tenant-ID", uuid.New().String())
	rr := httptest.NewRecorder()

	handler.HandleEvaluateExternal(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleEvaluateExternalBatch_MissingTenantID(t *testing.T) {
	handler := api.NewExternalComplianceHandler(nil, &mockRefFetcher{}, nil, nil, "")

	reqBody, _ := json.Marshal(api.ExternalBatchEvaluateRequest{
		SystemIdentifier: "BLOOMBERG_EMS",
		BatchID:         "BATCH-991",
		Trades:          []api.ExternalTradeItem{},
	})

	req := httptest.NewRequest("POST", "/api/v1/compliance/external/evaluate-external-batch", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()

	handler.HandleEvaluateExternalBatch(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleEvaluateExternalBatch_EmptyTrades(t *testing.T) {
	tenantID := uuid.New()
	handler := api.NewExternalComplianceHandler(nil, &mockRefFetcher{}, nil, nil, "")

	reqBody, _ := json.Marshal(api.ExternalBatchEvaluateRequest{
		SystemIdentifier: "BLOOMBERG_EMS",
		BatchID:         "BATCH-991",
		Trades:          []api.ExternalTradeItem{},
	})

	req := httptest.NewRequest("POST", "/api/v1/compliance/external/evaluate-external-batch", bytes.NewReader(reqBody))
	req.Header.Set("X-Tenant-ID", tenantID.String())
	rr := httptest.NewRecorder()

	handler.HandleEvaluateExternalBatch(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}
