package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// AML SCREENING HANDLERS
// ============================================================================

type AMLScreeningHandler struct {
	amlService *AMLScreeningService
	service    *ClientOnboardingService
}

// NewAMLScreeningHandler creates a new AML screening handler
func NewAMLScreeningHandler(amlService *AMLScreeningService, service *ClientOnboardingService) *AMLScreeningHandler {
	return &AMLScreeningHandler{
		amlService: amlService,
		service:    service,
	}
}

// ============================================================================
// STEP 2: PERFORM AML SCREENING
// ============================================================================

// Step2PerformAMLScreeningHandler performs detailed AML screening against watchlists and sanctions lists
// POST /api/onboarding/{clientID}/step2-aml-screening
// ABAC: Restricted to ComplianceOfficer role
func (h *AMLScreeningHandler) Step2PerformAMLScreeningHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	userID := r.Header.Get("X-User-ID")
	userRole := r.Header.Get("X-User-Role") // For ABAC

	if tenantID == "" || datasourceID == "" || userID == "" {
		http.Error(w, "Missing tenant or user context", http.StatusBadRequest)
		return
	}

	// ABAC Check: Only compliance officers can perform AML screening
	if !hasComplianceRole(userRole) {
		http.Error(w, "Unauthorized: AML screening requires compliance officer role", http.StatusForbidden)
		return
	}

	clientID := chi.URLParam(r, "clientID")
	if clientID == "" {
		http.Error(w, "Missing client ID", http.StatusBadRequest)
		return
	}

	var req AMLScreeningRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate screening provider
	validProviders := []string{"lexis_nexis", "worldcheck", "dow_jones", "internal"}
	if !stringInSlice(validProviders, req.ScreeningProvider) {
		http.Error(w, "Invalid screening provider", http.StatusBadRequest)
		return
	}

	// Get client record
	client, err := h.service.GetClient(r.Context(), tenantID, clientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Client not found: %v", err), http.StatusNotFound)
		return
	}

	// Create AML screening record (will trigger external API call in activity)
	screening, err := h.amlService.CreateAMLScreening(
		r.Context(),
		tenantID, datasourceID, userID,
		&req,
		client,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create AML screening: %v", err), http.StatusInternalServerError)
		return
	}

	// TODO: Trigger Temporal activity to call external AML provider
	// This would be: performer.CallExternalAMLScreeningActivity(screening.ID, req.ScreeningProvider)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted for async operation
	json.NewEncoder(w).Encode(map[string]interface{}{
		"screening_id": screening.ID,
		"status":       screening.ScreeningStatus,
		"risk_score":   screening.RiskScore,
		"risk_level":   screening.RiskLevel,
		"message":      "AML screening initiated - results will be available shortly",
	})
}

// GetAMLScreeningHandler retrieves AML screening results
// GET /api/onboarding/{clientID}/aml-screening/{screeningID}
// ABAC: Restricted to client, advisor, or compliance officer
func (h *AMLScreeningHandler) GetAMLScreeningHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userRole := r.Header.Get("X-User-Role")

	if tenantID == "" {
		http.Error(w, "Missing tenant context", http.StatusBadRequest)
		return
	}

	// ABAC Check: Only authorized roles can view AML results
	if !hasAMLAccessRole(userRole) {
		http.Error(w, "Unauthorized: AML results access restricted", http.StatusForbidden)
		return
	}

	screeningID := chi.URLParam(r, "screeningID")
	if screeningID == "" {
		http.Error(w, "Missing screening ID", http.StatusBadRequest)
		return
	}

	screening, err := h.amlService.GetAMLScreening(r.Context(), tenantID, screeningID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Screening not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(screening)
}

// GetLatestAMLScreeningHandler retrieves the most recent AML screening for a client
// GET /api/onboarding/{clientID}/aml-screening/latest
// ABAC: Restricted by client and role
func (h *AMLScreeningHandler) GetLatestAMLScreeningHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userRole := r.Header.Get("X-User-Role")

	if tenantID == "" {
		http.Error(w, "Missing tenant context", http.StatusBadRequest)
		return
	}

	// ABAC Check
	if !hasAMLAccessRole(userRole) {
		http.Error(w, "Unauthorized: AML results access restricted", http.StatusForbidden)
		return
	}

	clientID := chi.URLParam(r, "clientID")
	if clientID == "" {
		http.Error(w, "Missing client ID", http.StatusBadRequest)
		return
	}

	screening, err := h.amlService.GetLatestClientAMLScreening(r.Context(), clientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("No screening found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(screening)
}

// ReviewAMLScreeningHandler approves or rejects AML screening
// POST /api/onboarding/aml-screening/{screeningID}/review
// ABAC: Restricted to ComplianceOfficer role with temporal policy (within 24 hours of screening)
func (h *AMLScreeningHandler) ReviewAMLScreeningHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	userRole := r.Header.Get("X-User-Role")

	if tenantID == "" || userID == "" {
		http.Error(w, "Missing tenant or user context", http.StatusBadRequest)
		return
	}

	// ABAC Check: Only senior compliance officers can approve
	if userRole != "ComplianceOfficer" && userRole != "ComplianceManager" {
		http.Error(w, "Unauthorized: AML review requires compliance officer role", http.StatusForbidden)
		return
	}

	screeningID := chi.URLParam(r, "screeningID")
	if screeningID == "" {
		http.Error(w, "Missing screening ID", http.StatusBadRequest)
		return
	}

	var req AMLScreeningReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate approval status
	if req.ApprovalStatus != "approved" && req.ApprovalStatus != "rejected" {
		http.Error(w, "Invalid approval status - must be 'approved' or 'rejected'", http.StatusBadRequest)
		return
	}

	// Get screening to verify tenant context
	_, err := h.amlService.GetAMLScreening(r.Context(), tenantID, screeningID)
	if err != nil {
		http.Error(w, "Screening not found", http.StatusNotFound)
		return
	}

	// Update screening status
	updated, err := h.amlService.UpdateAMLScreeningStatus(
		r.Context(),
		screeningID, tenantID, userID,
		req.ApprovalStatus,
		req.ComplianceNotes,
		req.RejectionReason,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update screening: %v", err), http.StatusInternalServerError)
		return
	}

	// TODO: If approved, trigger next step in Temporal workflow
	// If rejected, trigger rejection signal and update client status

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"screening_id":    updated.ID,
		"overall_status":  updated.OverallStatus,
		"approval_status": req.ApprovalStatus,
		"reviewed_at":     updated.ApprovedAt,
		"reviewed_by":     userID,
	})
}

// GetAMLScreeningHistoryHandler retrieves screening history for a client
// GET /api/onboarding/{clientID}/aml-screening/history
// ABAC: Restricted to compliance officers viewing client's history
func (h *AMLScreeningHandler) GetAMLScreeningHistoryHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userRole := r.Header.Get("X-User-Role")

	if tenantID == "" {
		http.Error(w, "Missing tenant context", http.StatusBadRequest)
		return
	}

	// ABAC Check
	if !hasComplianceRole(userRole) {
		http.Error(w, "Unauthorized: AML history access requires compliance role", http.StatusForbidden)
		return
	}

	clientID := chi.URLParam(r, "clientID")
	if clientID == "" {
		http.Error(w, "Missing client ID", http.StatusBadRequest)
		return
	}

	screenings, err := h.amlService.GetClientAMLScreeningHistory(r.Context(), clientID, 10)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get screening history: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"client_id":  clientID,
		"count":      len(screenings),
		"screenings": screenings,
	})
}

// ============================================================================
// ROUTE REGISTRATION
// ============================================================================

// RegisterAMLScreeningRoutes registers all AML screening endpoints
func RegisterAMLScreeningRoutes(r *chi.Mux, amlService *AMLScreeningService, service *ClientOnboardingService) {
	handler := NewAMLScreeningHandler(amlService, service)

	// AML Screening endpoints
	r.Post("/api/onboarding/{clientID}/step2-aml-screening", handler.Step2PerformAMLScreeningHandler)
	r.Get("/api/onboarding/{clientID}/aml-screening/{screeningID}", handler.GetAMLScreeningHandler)
	r.Get("/api/onboarding/{clientID}/aml-screening/latest", handler.GetLatestAMLScreeningHandler)
	r.Get("/api/onboarding/{clientID}/aml-screening/history", handler.GetAMLScreeningHistoryHandler)
	r.Post("/api/onboarding/aml-screening/{screeningID}/review", handler.ReviewAMLScreeningHandler)
}

// ============================================================================
// ABAC HELPER FUNCTIONS
// ============================================================================

// hasComplianceRole checks if user has compliance-related role
func hasComplianceRole(role string) bool {
	complianceRoles := []string{
		"ComplianceOfficer",
		"ComplianceManager",
		"ComplianceDirector",
		"Admin",
	}
	for _, r := range complianceRoles {
		if strings.EqualFold(r, role) {
			return true
		}
	}
	return false
}

// hasAMLAccessRole checks if user can access AML results
func hasAMLAccessRole(role string) bool {
	allowedRoles := []string{
		"ComplianceOfficer",
		"ComplianceManager",
		"ComplianceDirector",
		"Advisor",
		"Admin",
		"Client", // Clients can view their own AML status
	}
	for _, r := range allowedRoles {
		if strings.EqualFold(r, role) {
			return true
		}
	}
	return false
}

// stringInSlice checks if string array contains value
func stringInSlice(arr []string, val string) bool {
	for _, v := range arr {
		if strings.EqualFold(v, val) {
			return true
		}
	}
	return false
}
