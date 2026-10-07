package limits

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/shopspring/decimal"
)

func TestLimitUtilization_HeadroomCalculations(t *testing.T) {
	svc := NewService(nil)
	tenantID := uuid.New()

	filter := LimitFilter{
		TenantID: tenantID,
	}

	records, err := svc.GetLimitUtilization(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) == 0 {
		t.Fatalf("expected limit records, got 0")
	}

	var foundUCITS bool
	for _, rec := range records {
		if rec.RuleCode == "UCITS_5_10_40_RULE" {
			foundUCITS = true
			if rec.UtilizationPct.LessThan(decimal.RequireFromString("80.0")) {
				t.Errorf("expected UCITS utilization >= 80%%, got %s", rec.UtilizationPct)
			}
			if rec.HeadroomAmount.IsZero() {
				t.Errorf("expected non-zero headroom dollar amount")
			}
			if len(rec.History) != 8 { // 7 days + day 0
				t.Errorf("expected 8 sparkline history points, got %d", len(rec.History))
			}
		}
	}

	if !foundUCITS {
		t.Errorf("expected to find UCITS 5/10/40 limit record")
	}
}

func TestLimitUtilization_Handler_Authenticated(t *testing.T) {
	svc := NewService(nil)
	handler := NewHandler(svc)

	tenantID := uuid.New()
	claims := &jwtmiddleware.JWTClaims{
		TenantID: tenantID.String(),
		Email:    "risk_officer@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user-456",
		},
	}

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/compliance/limits/utilization?rule_pack=UCITS", nil)
	ctx := context.WithValue(req.Context(), jwtmiddleware.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data       []LimitUtilizationRecord `json:"data"`
		TotalCount int                      `json:"total_count"`
		TenantID   string                   `json:"tenant_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TenantID != tenantID.String() {
		t.Errorf("expected tenant %s, got %s", tenantID, resp.TenantID)
	}
	for _, d := range resp.Data {
		if d.RulePack != "UCITS" {
			t.Errorf("expected only UCITS pack due to filter, got %s", d.RulePack)
		}
	}
}

func TestLimitUtilization_Handler_Unauthenticated_401(t *testing.T) {
	svc := NewService(nil)
	handler := NewHandler(svc)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/compliance/limits/utilization", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing JWT claims, got %d", rec.Code)
	}
}

func TestLimitUtilization_Handler_CrossTenantIDOR_403(t *testing.T) {
	svc := NewService(nil)
	handler := NewHandler(svc)

	attackerTenantID := uuid.New()
	targetVictimTenantID := uuid.New()

	claims := &jwtmiddleware.JWTClaims{
		TenantID:  attackerTenantID.String(),
		TenantIDs: []string{attackerTenantID.String()},
		Email:     "attacker@example.com",
		Roles:     []string{"compliance_viewer"},
	}

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/compliance/limits/utilization?tenant_id="+targetVictimTenantID.String(), nil)
	ctx := context.WithValue(req.Context(), jwtmiddleware.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for cross-tenant IDOR attack, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLimitHistory_Handler(t *testing.T) {
	svc := NewService(nil)
	handler := NewHandler(svc)

	tenantID := uuid.New()
	claims := &jwtmiddleware.JWTClaims{
		TenantID: tenantID.String(),
		Email:    "risk_officer@example.com",
	}

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/compliance/limits/history?rule_id=SEC-1940-5-10-40&days=14", nil)
	ctx := context.WithValue(req.Context(), jwtmiddleware.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data     []LimitDataPoint `json:"data"`
		RuleID   string           `json:"rule_id"`
		TenantID string           `json:"tenant_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.RuleID != "SEC-1940-5-10-40" {
		t.Errorf("expected rule SEC-1940-5-10-40, got %s", resp.RuleID)
	}
	if len(resp.Data) != 15 { // 14 days + day 0
		t.Errorf("expected 15 historical points, got %d", len(resp.Data))
	}
}
