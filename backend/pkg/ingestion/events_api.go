package ingestion

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hondyman/uisce/backend/pkg/workflows"
	"go.temporal.io/sdk/client"
)

// EventsHandler processes high-throughput external events (e.g. Market Data, Risk Alerts)
// and triggers associated Business Processes.
type EventsHandler struct {
	temporalClient client.Client
}

func NewEventsHandler(c client.Client) *EventsHandler {
	return &EventsHandler{temporalClient: c}
}

type MarketEvent struct {
	EventID   string      `json:"eventId"`
	Type      string      `json:"type"` // e.g. "PRICE_DROP", "MARGIN_CALL"
	Symbol    string      `json:"symbol"`
	Timestamp int64       `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// IngestMarketEvent is the webhook endpoint for algorithmic triggers
func (h *EventsHandler) IngestMarketEvent(w http.ResponseWriter, r *http.Request) {
	var evt MarketEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		http.Error(w, "Invalid Payload", http.StatusBadRequest)
		return
	}

	// 1. Identify which BP to trigger based on Event Type
	// In a real system, this comes from a "Subscription" or "Trigger Rule" table.
	// Hardcoding mapping for demo.
	var bpID string
	if evt.Type == "PRICE_DROP" {
		bpID = "bp_margin_call_protocol" // The "Smart Branching" workflow we built
	} else if evt.Type == "KYC_UPDATE" {
		bpID = "bp_kyc_refresh"
	} else {
		// No trigger defined, just log and ack
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"IGNORED_NO_TRIGGER"}`))
		return
	}

	// 2. Trigger the Interpreter Workflow (by stored ID)
	// The Event Payload becomes the "Input Data" for the Business Process
	workflowOptions := client.StartWorkflowOptions{
		ID: fmt.Sprintf("event-%s-%s", evt.Type, evt.EventID),
		// workflows.DeployedBPTaskQueue ("bp_queue"), not BPTaskQueue: the
		// worker actually deployed in production is cmd/worker (see
		// backend/Dockerfile's "./worker" and docker-compose.hybrid.yml /
		// docker-compose.starrocks.yml), which polls "bp_queue" and
		// registers RunStoredWorkflow there. cmd/bp-worker (BPTaskQueue,
		// "bp-framework-queue") is not deployed anywhere.
		TaskQueue: workflows.DeployedBPTaskQueue,
	}

	input := workflows.InterpreterInput{
		WorkflowID: bpID,
		InitialData: map[string]interface{}{
			"event":   evt,
			"source":  "MARKET_FEED",
			"trigger": "AUTOMATED",
		},
	}

	// RunStoredWorkflow, not InterpreterWorkflow: input here is
	// InterpreterInput{WorkflowID, InitialData}, which matches
	// RunStoredWorkflow's signature, not InterpreterWorkflow's (which
	// takes a WorkflowDefinition directly) — this was a genuine
	// type-mismatch bug that would have failed at workflow-start time on
	// every real call.
	we, err := h.temporalClient.ExecuteWorkflow(r.Context(), workflowOptions, workflows.RunStoredWorkflow, input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to trigger workflow: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":      "TRIGGERED",
		"workflowId":  we.GetID(),
		"runId":       we.GetRunID(),
		"triggeredBp": bpID,
	})
}
