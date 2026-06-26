package genui

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Service provides GenUI API handlers
type Service struct {
	classifier       *IntentClassifier
	layoutBuilder    *LayoutBuilder
	approvalsService *ApprovalsService
}

func NewService() *Service {
	return &Service{
		classifier:    NewIntentClassifier(),
		layoutBuilder: NewLayoutBuilder(),
		// approvalsService will be set when Temporal client is available
	}
}

// SetApprovalsService sets the approvals service (called after Temporal client is initialized)
func (s *Service) SetApprovalsService(approvalsService *ApprovalsService) {
	s.approvalsService = approvalsService
}

// RegisterRoutes registers GenUI API routes
func (s *Service) RegisterRoutes(r chi.Router) {
	r.Post("/genui/intent", s.HandleIntent)
	r.Get("/genui/templates", s.HandleListTemplates)
	r.Post("/genui/approvals/signal", s.HandleApprovalSignal)
	r.Get("/genui/approvals", s.HandleGetPendingApprovals)
}

// IntentRequest represents a request to generate a layout from natural language
type IntentRequest struct {
	Query      string         `json:"query"`
	TenantID   string         `json:"tenant_id"`
	UserID     string         `json:"user_id"`
	Context    map[string]any `json:"context,omitempty"`
}

// IntentResponse contains the generated layout
type IntentResponse struct {
	Intent *Intent    `json:"intent"`
	Layout *LayoutDef `json:"layout"`
}

// HandleIntent processes a natural language query and returns a layout
func (s *Service) HandleIntent(w http.ResponseWriter, r *http.Request) {
	var req IntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Classify intent
	intent, err := s.classifier.Classify(r.Context(), req.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate layout
	layout, err := s.layoutBuilder.Build(r.Context(), intent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	resp := IntentResponse{
		Intent: intent,
		Layout: layout,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// TemplateListResponse contains available layout templates
type TemplateListResponse struct {
	Templates []TemplateInfo `json:"templates"`
}

// TemplateInfo describes a layout template
type TemplateInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// HandleListTemplates returns available layout templates
func (s *Service) HandleListTemplates(w http.ResponseWriter, r *http.Request) {
	templates := []TemplateInfo{
		{
			ID:          "portfolio_dashboard",
			Name:        "Portfolio Dashboard",
			Description: "Overview of portfolio performance and holdings",
			Tags:        []string{"portfolio", "dashboard"},
		},
		{
			ID:          "performance_chart",
			Name:        "Performance Chart",
			Description: "Time-series performance visualization",
			Tags:        []string{"performance", "chart"},
		},
		{
			ID:          "holdings_grid",
			Name:        "Holdings Grid",
			Description: "Detailed holdings table with filtering",
			Tags:        []string{"holdings", "grid"},
		},
	}

	resp := TemplateListResponse{Templates: templates}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ApprovalSignalRequest represents a request to send an approval signal
type ApprovalSignalRequest struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id,omitempty"`
	Approved   bool   `json:"approved"`
	Rationale  string `json:"rationale"`
}

// HandleApprovalSignal sends an approval/rejection signal to a Temporal workflow
func (s *Service) HandleApprovalSignal(w http.ResponseWriter, r *http.Request) {
	if s.approvalsService == nil {
		http.Error(w, "Approvals service not initialized", http.StatusServiceUnavailable)
		return
	}

	var req ApprovalSignalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.WorkflowID == "" {
		http.Error(w, "workflow_id is required", http.StatusBadRequest)
		return
	}

	err := s.approvalsService.SendApprovalSignal(r.Context(), req.WorkflowID, req.RunID, req.Approved, req.Rationale)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Approval signal sent successfully",
	})
}

// HandleGetPendingApprovals returns pending approvals for the tenant
func (s *Service) HandleGetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	if s.approvalsService == nil {
		http.Error(w, "Approvals service not initialized", http.StatusServiceUnavailable)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "demo_tenant" // default for demo
	}

	approvals, err := s.approvalsService.GetPendingApprovals(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"approvals": approvals,
		"count":     len(approvals),
	})
}
