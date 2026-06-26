package trade

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type TradeHandlers struct {
	MetadataService *MetadataService
	TemporalClient  client.Client
}

func NewTradeHandlers(ms *MetadataService, tc client.Client) *TradeHandlers {
	return &TradeHandlers{
		MetadataService: ms,
		TemporalClient:  tc,
	}
}

// GetWorkflowDefinition retrieves a workflow definition
// GET /api/trade/metadata/workflows
func (h *TradeHandlers) GetWorkflowDefinition(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	name := r.URL.Query().Get("name")

	if tenantID == "" || name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant_id and name required"})
		return
	}

	wd, err := h.MetadataService.GetWorkflowDefinition(tenantID, name)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wd)
}

// CreateWorkflowDefinition creates a new workflow definition
// POST /api/trade/metadata/workflows
func (h *TradeHandlers) CreateWorkflowDefinition(w http.ResponseWriter, r *http.Request) {
	var wd WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&wd); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if err := h.MetadataService.CreateWorkflowDefinition(&wd); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wd)
}

// StartTrade starts a new trade workflow
// POST /api/trade/start
func (h *TradeHandlers) StartTrade(w http.ResponseWriter, r *http.Request) {
	var input TradeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        "trade-" + uuid.New().String(),
		TaskQueue: "trade-queue",
	}

	we, err := h.TemporalClient.ExecuteWorkflow(context.Background(), workflowOptions, TradeWorkflow, input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to start workflow: " + err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"workflow_id": we.GetID(), "run_id": we.GetRunID()})
}

// SignalApprovePreCompliance signals approval for pre-compliance
// POST /api/trade/signal/approve_pre_compliance
func (h *TradeHandlers) SignalApprovePreCompliance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkflowID string `json:"workflow_id"`
		RunID      string `json:"run_id"`
		Approved   bool   `json:"approved"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	err := h.TemporalClient.SignalWorkflow(context.Background(), req.WorkflowID, req.RunID, SignalApprovePreCompliance, req.Approved)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to signal workflow: " + err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "signaled"})
}

// RegisterRoutes registers the trade routes
func (h *TradeHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/trade", func(r chi.Router) {
		r.Get("/metadata/workflows", h.GetWorkflowDefinition)
		r.Post("/metadata/workflows", h.CreateWorkflowDefinition)
		r.Post("/start", h.StartTrade)
		r.Post("/signal/approve_pre_compliance", h.SignalApprovePreCompliance)
	})
}
