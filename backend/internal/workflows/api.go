package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.temporal.io/sdk/client"
)

// Handler exposes workflow execution via HTTP (for Hasura Actions)
type Handler struct {
	client client.Client
}

func NewHandler(c client.Client) *Handler {
	return &Handler{client: c}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/workflows/execute", h.ExecuteWorkflow)
}

type ExecuteRequest struct {
	WorkflowType string          `json:"workflow_type"`
	Input        json.RawMessage `json:"input"`
}

type ExecuteResponse struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	Status     string `json:"status"`
}

func (h *Handler) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	options := client.StartWorkflowOptions{
		ID:        "wf_" + req.WorkflowType + "_" + GenerateID(),
		TaskQueue: "wealth-stream-queue",
	}

	var input interface{}
	// Map input based on workflow type
	switch req.WorkflowType {
	case "DriftMonitor":
		var i DriftMonitorInput
		json.Unmarshal(req.Input, &i)
		input = i
	default:
		http.Error(w, "Unknown workflow type", http.StatusBadRequest)
		return
	}

	we, err := h.client.ExecuteWorkflow(context.Background(), options, GetWorkflowFunc(req.WorkflowType), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ExecuteResponse{
		WorkflowID: we.GetID(),
		RunID:      we.GetRunID(),
		Status:     "STARTED",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func GetWorkflowFunc(name string) interface{} {
	switch name {
	case "DriftMonitor":
		return DriftMonitorWorkflow
	default:
		return nil
	}
}

func GenerateID() string {
	// Simple ID generator
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
