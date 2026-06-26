package guardrails

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/audit"
)

// GuardrailHandler handles HTTP requests for guardrail operations
type GuardrailHandler struct {
	engine *GuardrailEngine
}

// NewGuardrailHandler creates a new guardrail HTTP handler
func NewGuardrailHandler(auditService *audit.Service) *GuardrailHandler {
	return &GuardrailHandler{
		engine: NewGuardrailEngine(auditService),
	}
}

// FilterRequest represents a request to filter AI output
type FilterRequest struct {
	Content  string `json:"content" binding:"required"`
	TenantID string `json:"tenant_id" binding:"required"`
	UserID   string `json:"user_id" binding:"required"`
	Context  string `json:"context,omitempty"` // e.g., "CHAT", "REPORT_GENERATION", "EMAIL"
}

// FilterAIOutput handles POST /api/guardrails/filter
func (h *GuardrailHandler) FilterAIOutput(w http.ResponseWriter, r *http.Request) {
	var req FilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	result, err := h.engine.FilterAIOutput(r.Context(), req.Content, req.TenantID, req.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Guardrail check failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetPolicyStatus handles GET /api/guardrails/policies
func (h *GuardrailHandler) GetPolicyStatus(w http.ResponseWriter, r *http.Request) {
	policies := h.engine.policies.policies

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"policies": policies,
		"total":    len(policies),
	})
}

// GetGuardrailStats handles GET /api/guardrails/stats
func (h *GuardrailHandler) GetGuardrailStats(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	// TODO: Query audit log for statistics
	stats := map[string]interface{}{
		"tenant_id":          tenantID,
		"total_checks_today": 127,
		"violations_today":   8,
		"blocked_outputs":    2,
		"pii_redactions":     6,
		"approval_rate":      0.98,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// RegisterRoutes registers all guardrail routes
func (h *GuardrailHandler) RegisterRoutes(r chi.Router) {
	r.Route("/guardrails", func(r chi.Router) {
		r.Post("/filter", h.FilterAIOutput)
		r.Get("/policies", h.GetPolicyStatus)
		r.Get("/stats", h.GetGuardrailStats)
	})
}
