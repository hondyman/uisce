package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
	"go.uber.org/zap"
)

// BusinessProcessHandler handles business process API requests
type BusinessProcessHandler struct {
	service *services.BusinessProcessService
	logger  *zap.Logger
}

// NewBusinessProcessHandler creates a new handler
func NewBusinessProcessHandler(service *services.BusinessProcessService) *BusinessProcessHandler {
	logger, _ := zap.NewProduction()
	return &BusinessProcessHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers business process routes
func (h *BusinessProcessHandler) RegisterRoutes(r chi.Router) {
	r.Route("/processes", func(r chi.Router) {
		// Process definitions
		r.Get("/", h.ListProcesses)
		r.Get("/{key}", h.GetProcess)
		r.Get("/{key}/steps", h.GetProcessSteps)

		// Process instances
		r.Post("/start", h.StartProcess)
		r.Route("/instances", func(r chi.Router) {
			r.Get("/{instanceId}", h.GetInstance)
			r.Get("/{instanceId}/history", h.GetInstanceHistory)
			r.Post("/{instanceId}/advance", h.AdvanceProcess)
			r.Post("/{instanceId}/complete", h.CompleteProcess)
		})

		// Entity processes
		r.Get("/entity/{entityType}/{entityId}", h.ListInstancesForEntity)
	})
}

// StartProcessRequest represents the request body
type StartProcessRequest struct {
	ProcessKey string                 `json:"process_key"`
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// AdvanceProcessRequest represents the advance request body
type AdvanceProcessRequest struct {
	Action   string                 `json:"action"` // approved, rejected, completed, skipped
	Comments string                 `json:"comments,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// ListProcesses handles GET /processes
func (h *BusinessProcessHandler) ListProcesses(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	processes, err := h.service.ListProcesses(r.Context(), category)
	if err != nil {
		h.logger.Error("Failed to list processes", zap.Error(err))
		http.Error(w, "Failed to list processes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processes)
}

// GetProcess handles GET /processes/{key}
func (h *BusinessProcessHandler) GetProcess(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	process, err := h.service.GetProcessByKey(r.Context(), key)
	if err != nil {
		h.logger.Error("Failed to get process", zap.Error(err))
		http.Error(w, "Failed to get process", http.StatusInternalServerError)
		return
	}

	if process == nil {
		http.Error(w, "Process not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(process)
}

// GetProcessSteps handles GET /processes/{key}/steps
func (h *BusinessProcessHandler) GetProcessSteps(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	process, err := h.service.GetProcessByKey(r.Context(), key)
	if err != nil || process == nil {
		http.Error(w, "Process not found", http.StatusNotFound)
		return
	}

	steps, err := h.service.GetProcessSteps(r.Context(), process.ID)
	if err != nil {
		h.logger.Error("Failed to get steps", zap.Error(err))
		http.Error(w, "Failed to get steps", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(steps)
}

// StartProcess handles POST /processes/start
func (h *BusinessProcessHandler) StartProcess(w http.ResponseWriter, r *http.Request) {
	var req StartProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ProcessKey == "" || req.EntityType == "" || req.EntityID == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "system"
	}

	instance, err := h.service.StartProcess(r.Context(), req.ProcessKey, req.EntityType, req.EntityID, actor, req.Data)
	if err != nil {
		h.logger.Error("Failed to start process", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(instance)
}

// GetInstance handles GET /processes/instances/{instanceId}
func (h *BusinessProcessHandler) GetInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := chi.URLParam(r, "instanceId")

	instance, err := h.service.GetInstance(r.Context(), instanceID)
	if err != nil {
		h.logger.Error("Failed to get instance", zap.Error(err))
		http.Error(w, "Instance not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instance)
}

// GetInstanceHistory handles GET /processes/instances/{instanceId}/history
func (h *BusinessProcessHandler) GetInstanceHistory(w http.ResponseWriter, r *http.Request) {
	instanceID := chi.URLParam(r, "instanceId")

	history, err := h.service.GetInstanceHistory(r.Context(), instanceID)
	if err != nil {
		h.logger.Error("Failed to get history", zap.Error(err))
		http.Error(w, "Failed to get history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// AdvanceProcess handles POST /processes/instances/{instanceId}/advance
func (h *BusinessProcessHandler) AdvanceProcess(w http.ResponseWriter, r *http.Request) {
	instanceID := chi.URLParam(r, "instanceId")

	var req AdvanceProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "system"
	}

	if err := h.service.AdvanceProcess(r.Context(), instanceID, req.Action, actor, req.Comments, req.Data); err != nil {
		h.logger.Error("Failed to advance process", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return updated instance
	instance, _ := h.service.GetInstance(r.Context(), instanceID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instance)
}

// CompleteProcess handles POST /processes/instances/{instanceId}/complete
func (h *BusinessProcessHandler) CompleteProcess(w http.ResponseWriter, r *http.Request) {
	instanceID := chi.URLParam(r, "instanceId")

	if err := h.service.CompleteProcess(r.Context(), instanceID); err != nil {
		h.logger.Error("Failed to complete process", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "completed"})
}

// ListInstancesForEntity handles GET /processes/entity/{entityType}/{entityId}
func (h *BusinessProcessHandler) ListInstancesForEntity(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entityType")
	entityID := chi.URLParam(r, "entityId")

	instances, err := h.service.ListInstancesForEntity(r.Context(), entityType, entityID)
	if err != nil {
		h.logger.Error("Failed to list instances", zap.Error(err))
		http.Error(w, "Failed to list instances", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instances)
}
