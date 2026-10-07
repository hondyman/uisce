package calendar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

func TestComplianceCalendar_DeadlineArithmetic(t *testing.T) {
	svc := NewService(nil)

	tenantID := uuid.New()
	refDay := time.Date(2026, time.November, 2, 9, 0, 0, 0, time.UTC) // Monday, Nov 2, 2026

	// Query for Nov 2026 events
	filter := CalendarFilter{
		TenantID:     tenantID,
		FromDate:     time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC),
		ToDate:       time.Date(2026, time.November, 30, 23, 59, 59, 0, time.UTC),
		ReferenceDay: refDay,
	}

	events, err := svc.GetCalendarEvents(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) == 0 {
		t.Fatalf("expected calendar events for Nov 2026, got 0")
	}

	// Verify SEC 13F Q3 event (due Nov 14 / nearest trading day)
	var found13F bool
	for _, e := range events {
		if e.Regulation == "SEC Form 13F" {
			found13F = true
			if e.Jurisdiction != "US" {
				t.Errorf("expected 13F jurisdiction US, got %s", e.Jurisdiction)
			}
			if e.DaysRemaining <= 0 {
				t.Errorf("expected positive days remaining from Nov 2 to Nov 13/14, got %d", e.DaysRemaining)
			}
			if e.Status != "DUE_SOON" && e.Status != "UPCOMING" {
				t.Errorf("unexpected status: %s", e.Status)
			}
		}
	}
	if !found13F {
		t.Errorf("expected SEC Form 13F event in Nov 2026")
	}

	// Verify UK Takeover Panel Rule 8.3 event
	var foundRule83 bool
	for _, e := range events {
		if e.Regulation == "Takeover Panel Rule 8.3" {
			foundRule83 = true
			if e.Jurisdiction != "UK" {
				t.Errorf("expected UK jurisdiction, got %s", e.Jurisdiction)
			}
			if e.DeadlineType != "DISCLOSURE_CUTOFF" {
				t.Errorf("expected DISCLOSURE_CUTOFF deadline type, got %s", e.DeadlineType)
			}
		}
	}
	if !foundRule83 {
		t.Errorf("expected UK Takeover Panel Rule 8.3 event in Nov 2026")
	}
}

func TestComplianceCalendar_OverdueStatus(t *testing.T) {
	svc := NewService(nil)
	tenantID := uuid.New()

	// Reference date after due date
	refDay := time.Date(2026, time.November, 20, 9, 0, 0, 0, time.UTC)
	filter := CalendarFilter{
		TenantID:     tenantID,
		FromDate:     time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC),
		ToDate:       time.Date(2026, time.November, 30, 23, 59, 59, 0, time.UTC),
		ReferenceDay: refDay,
	}

	events, err := svc.GetCalendarEvents(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, e := range events {
		if e.Regulation == "SEC Form 13F" {
			if e.DaysRemaining >= 0 {
				t.Errorf("expected negative days remaining for past 13F deadline, got %d", e.DaysRemaining)
			}
			if e.Status != "OVERDUE" {
				t.Errorf("expected OVERDUE status, got %s", e.Status)
			}
		}
	}
}

func TestComplianceCalendar_Handler_Authenticated(t *testing.T) {
	svc := NewService(nil)
	handler := NewHandler(svc)

	tenantID := uuid.New()
	claims := &jwtmiddleware.JWTClaims{
		TenantID: tenantID.String(),
		Email:    "compliance@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user-123",
		},
	}

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/compliance/calendar?jurisdiction=US", nil)
	ctx := context.WithValue(req.Context(), jwtmiddleware.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data       []ComplianceCalendarEvent `json:"data"`
		TotalCount int                       `json:"total_count"`
		TenantID   string                    `json:"tenant_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TenantID != tenantID.String() {
		t.Errorf("expected tenant %s, got %s", tenantID, resp.TenantID)
	}
	for _, e := range resp.Data {
		if e.Jurisdiction != "US" {
			t.Errorf("expected only US jurisdiction due to filter, got %s", e.Jurisdiction)
		}
	}
}

func TestComplianceCalendar_Handler_Unauthenticated_401(t *testing.T) {
	svc := NewService(nil)
	handler := NewHandler(svc)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	// No claims in context
	req := httptest.NewRequest("GET", "/compliance/calendar", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing JWT claims, got %d", rec.Code)
	}
}

func TestComplianceCalendar_Handler_CrossTenantIDOR_403(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/compliance/calendar?tenant_id="+targetVictimTenantID.String(), nil)
	ctx := context.WithValue(req.Context(), jwtmiddleware.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for cross-tenant IDOR attack, got %d: %s", rec.Code, rec.Body.String())
	}
}
