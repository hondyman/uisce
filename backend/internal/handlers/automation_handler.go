package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// AutomationHandler handles API requests for the automation engine.
type AutomationHandler struct {
	service *services.AutomationService
}

// NewAutomationHandler creates a new AutomationHandler.
func NewAutomationHandler(service *services.AutomationService) *AutomationHandler {
	return &AutomationHandler{service: service}
}

// RegisterRoutes mounts automation routes
func (h *AutomationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/automation", func(r chi.Router) {
		r.Post("/run", h.HandleRunAutomationCycle)
		r.Get("/logs", h.HandleListAutomationLogs)
		r.Post("/pause", h.HandlePauseAutomation)
		r.Post("/resume", h.HandleResumeAutomation)
		r.Get("/policies", h.HandleListAutomationPolicies)
	})
}

// HandleRunAutomationCycle manually triggers an automation cycle.
func (h *AutomationHandler) HandleRunAutomationCycle(w http.ResponseWriter, r *http.Request) {
	actorID := "current_admin" // In a real app, get this from the auth context
	logs, err := h.service.RunAutomationCycle(r.Context(), actorID)
	if err != nil {
		http.Error(w, "Failed to run automation cycle", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "completed", "logs": logs})
}

// HandleListAutomationLogs retrieves recent automation logs.
func (h *AutomationHandler) HandleListAutomationLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.service.ListAutomationLogs(r.Context())
	if err != nil {
		http.Error(w, "Failed to list automation logs", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// HandlePauseAutomation pauses the automation engine.
func (h *AutomationHandler) HandlePauseAutomation(w http.ResponseWriter, r *http.Request) {
	err := h.service.PauseAutomation(r.Context())
	if err != nil {
		http.Error(w, "Failed to pause automation", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "paused"})
}

// HandleResumeAutomation resumes the automation engine.
func (h *AutomationHandler) HandleResumeAutomation(w http.ResponseWriter, r *http.Request) {
	err := h.service.ResumeAutomation(r.Context())
	if err != nil {
		http.Error(w, "Failed to resume automation", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "resumed"})
}

// HandleListAutomationPolicies lists all automation policies.
func (h *AutomationHandler) HandleListAutomationPolicies(w http.ResponseWriter, r *http.Request) {
	policies, err := h.service.ListAutomationPolicies(r.Context())
	if err != nil {
		http.Error(w, "Failed to list automation policies", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policies)
}
