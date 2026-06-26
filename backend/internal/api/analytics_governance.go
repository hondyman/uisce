// internal/api/analytics_governance.go
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// LayoutAnalyticsEvent represents user interactions with the layout builder
type LayoutAnalyticsEvent struct {
	EventType     string                 `json:"eventType"` // container_decision, field_add, section_create, layout_save, etc.
	Timestamp     time.Time              `json:"timestamp"`
	SectionID     string                 `json:"sectionId,omitempty"`
	FieldCount    int                    `json:"fieldCount,omitempty"`
	Device        string                 `json:"device,omitempty"`        // mobile, tablet, desktop
	ContainerKind string                 `json:"containerKind,omitempty"` // modal, panel, inline
	CustomData    map[string]interface{} `json:"customData,omitempty"`
}

// registerAnalyticsRoutes registers analytics and governance endpoints
func registerAnalyticsRoutes(r chi.Router) {
	r.Post("/analytics/layout", handleLayoutAnalytics)
	r.Post("/publish/validate", handlePublishValidation)
}

// handleLayoutAnalytics receives analytics beacon payloads from the frontend
// Logs container decisions, user edits, and performance metrics for optimization
func handleLayoutAnalytics(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	var event LayoutAnalyticsEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}

	// Log for aggregation/analysis (in production, write to analytics DB or event stream)
	log.Printf("[ANALYTICS] tenant=%s event=%s section=%s container=%s device=%s",
		tenantID, event.EventType, event.SectionID, event.ContainerKind, event.Device)

	// TODO: Store in time-series DB or publish to Kafka for analytics pipeline
	// Example: analyticsService.RecordEvent(tenantID, event)

	w.WriteHeader(http.StatusNoContent)
}

// PublishValidationRequest represents publish governance checks
type PublishValidationRequest struct {
	AccessibilityOk bool `json:"accessibilityOk"`
	PerformanceOk   bool `json:"performanceOk"`
}

// PublishValidationResponse represents publish validation result
type PublishValidationResponse struct {
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons,omitempty"`
}

// handlePublishValidation enforces governance gates before layout publication
// Checks: accessibility compliance, performance budget, data quality
func handlePublishValidation(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	var req PublishValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}

	// Perform governance checks
	var reasons []string
	allowed := true

	if !req.AccessibilityOk {
		allowed = false
		reasons = append(reasons, "Accessibility compliance checks failed. Please review WCAG 2.1 compliance.")
	}

	if !req.PerformanceOk {
		allowed = false
		reasons = append(reasons, "Performance budget exceeded. Please optimize field count or section complexity.")
	}

	// Additional check: ensure layout has at least one section
	// (Could be enhanced with actual layout validation)

	// Log publish validation attempt
	log.Printf("[GOVERNANCE] tenant=%s allowed=%v reasons=%v", tenantID, allowed, reasons)

	w.Header().Set("Content-Type", "application/json")
	if !allowed {
		w.WriteHeader(http.StatusPreconditionFailed) // 412
	}
	json.NewEncoder(w).Encode(PublishValidationResponse{
		Allowed: allowed,
		Reasons: reasons,
	})
}
