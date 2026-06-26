package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.temporal.io/sdk/client"

	"github.com/hondyman/semlayer/backend/internal/feed/approvals"
	"github.com/hondyman/semlayer/backend/internal/feed/workflows"
)

type ActionHandler struct {
	temporalClient client.Client
	approvalSvc    *approvals.Service
}

func NewActionHandler(temporalClient client.Client, approvalSvc *approvals.Service) *ActionHandler {
	return &ActionHandler{
		temporalClient: temporalClient,
		approvalSvc:    approvalSvc,
	}
}

// ExecuteActionRequest represents the HTTP request to execute an action
type ExecuteActionRequest struct {
	CardID        string                 `json:"card_id"`
	ClientID      string                 `json:"client_id"`
	TenantID      string                 `json:"tenant_id"`
	ActionDetails map[string]interface{} `json:"action_details"`
}

// ExecuteActionResponse is the HTTP response
type ExecuteActionResponse struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	Message    string `json:"message"`
}

func (h *ActionHandler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	var req ExecuteActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Map card ID to action type
	actionType := getActionType(req.CardID)
	if actionType == "" {
		http.Error(w, "Unknown card ID", http.StatusBadRequest)
		return
	}

	// Start Temporal workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("trade-%s-%s", req.ClientID, req.CardID),
		TaskQueue: "wealth-actions",
	}

	input := workflows.TradeWorkflowInput{
		TenantID:      req.TenantID,
		ClientID:      req.ClientID,
		ActionType:    actionType,
		ActionDetails: req.ActionDetails,
	}

	we, err := h.temporalClient.ExecuteWorkflow(context.Background(), workflowOptions, workflows.TradeWorkflow, input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start workflow: %v", err), http.StatusInternalServerError)
		return
	}

	response := ExecuteActionResponse{
		WorkflowID: we.GetID(),
		RunID:      we.GetRunID(),
		Message:    "Action workflow started successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func getActionType(cardID string) string {
	mapping := map[string]string{
		"tax_loss_harvest": "tax_loss_harvest",
		"portfolio_drift":  "rebalance",
	}
	return mapping[cardID]
}
